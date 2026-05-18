package telegram

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/agent"
	"github.com/google/uuid"
)

var telegramBotHTTPClient = &http.Client{Timeout: 15 * time.Second}

type BotService struct {
	botToken        string
	agentSvc        *agent.Service
	linkRepo        repository.TelegramLinkRepository
	convRepo        repository.TelegramConversationRepository
	agentConfigRepo repository.AgentConfigRepository
	approvalRepo    repository.ApprovalRepository
	lastOffset      int64
	offsetMu        sync.Mutex
	// pendingConfirmations tracks chatID -> pending action for autonomy level 1
	confirmMu            sync.Mutex
	pendingConfirmations map[int64]*pendingAction
}

type pendingAction struct {
	chatID  int64
	userID  uuid.UUID
	request model.ChatRequest
}

func NewBotService(
	botToken string,
	agentSvc *agent.Service,
	linkRepo repository.TelegramLinkRepository,
	convRepo repository.TelegramConversationRepository,
	agentConfigRepo repository.AgentConfigRepository,
	approvalRepo repository.ApprovalRepository,
) *BotService {
	return &BotService{
		botToken:             botToken,
		agentSvc:             agentSvc,
		linkRepo:             linkRepo,
		convRepo:             convRepo,
		agentConfigRepo:      agentConfigRepo,
		approvalRepo:         approvalRepo,
		pendingConfirmations: make(map[int64]*pendingAction),
	}
}

// Telegram API types

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	From      *User  `json:"from"`
}

// CallbackQuery is fired when the user taps an inline-keyboard button.
// `Data` is the opaque payload we baked into the button; we use it to route
// the action — e.g. "approve:<uuid>" or "deny:<uuid>" for the approval flow.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}

type getUpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

// InlineKeyboardButton is the JSON shape Telegram expects under reply_markup.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

type getMeResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		Username string `json:"username"`
	} `json:"result"`
}

// GetBotUsername returns the bot's username.
func (s *BotService) GetBotUsername(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", s.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := telegramBotHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result getMeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.OK {
		return "", fmt.Errorf("telegram getMe failed")
	}
	return result.Result.Username, nil
}

// getConfiguredModel reads the model from agent_config table.
func (s *BotService) getConfiguredModel(ctx context.Context) string {
	if s.agentConfigRepo == nil {
		return ""
	}
	model, err := s.agentConfigRepo.Get(ctx, "llm_model")
	if err != nil {
		return ""
	}
	return model
}

// PollUpdates fetches new messages from Telegram.
func (s *BotService) PollUpdates(ctx context.Context) error {
	s.offsetMu.Lock()
	offset := s.lastOffset + 1
	s.offsetMu.Unlock()

	// allowed_updates explicitly enables callback_query — without it Telegram
	// silently drops inline-button taps. The literal is URL-escaped JSON.
	url := fmt.Sprintf(
		"https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=1&limit=20&allowed_updates=%%5B%%22message%%22%%2C%%22callback_query%%22%%5D",
		s.botToken, offset,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create getUpdates request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("getUpdates: %w", err)
	}
	defer resp.Body.Close()

	var result getUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode getUpdates: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram getUpdates returned not ok")
	}

	for _, update := range result.Result {
		s.offsetMu.Lock()
		s.lastOffset = update.UpdateID
		s.offsetMu.Unlock()
		if update.Message != nil && update.Message.Text != "" {
			s.processUpdate(ctx, update.Message)
		}
		if update.CallbackQuery != nil {
			s.processCallback(ctx, update.CallbackQuery)
		}
	}

	return nil
}

// processCallback routes inline-button taps. We use a colon-separated payload
// like "approve:<uuid>" or "deny:<uuid>". Anything we don't recognize is
// silently acked — better than throwing visible errors at the user.
func (s *BotService) processCallback(ctx context.Context, cq *CallbackQuery) {
	if cq == nil || cq.From == nil || cq.Message == nil {
		return
	}
	chatID := cq.Message.Chat.ID
	messageID := cq.Message.MessageID

	link, err := s.linkRepo.GetByTelegramChatID(ctx, chatID)
	if err != nil {
		s.answerCallback(ctx, cq.ID, "Konto nicht verknüpft.", true)
		return
	}

	parts := strings.SplitN(cq.Data, ":", 2)
	action := parts[0]
	var argument string
	if len(parts) > 1 {
		argument = parts[1]
	}

	switch action {
	case "approve", "deny":
		s.handleApprovalCallback(ctx, cq.ID, chatID, messageID, link.UserID, argument, action == "approve")
	case "noop":
		s.answerCallback(ctx, cq.ID, "", false)
	default:
		s.answerCallback(ctx, cq.ID, "Aktion unbekannt.", false)
	}
}

func (s *BotService) handleApprovalCallback(
	ctx context.Context,
	callbackID string,
	chatID int64,
	messageID int64,
	userID uuid.UUID,
	approvalIDStr string,
	approve bool,
) {
	if s.approvalRepo == nil {
		s.answerCallback(ctx, callbackID, "Approval-System nicht verfügbar.", true)
		return
	}
	approvalID, err := uuid.Parse(approvalIDStr)
	if err != nil {
		s.answerCallback(ctx, callbackID, "Ungültige Approval-ID.", true)
		return
	}

	// Authorization: the user resolving the approval must be the user who
	// requested the agent action. Otherwise we silently refuse.
	approval, err := s.approvalRepo.GetByID(ctx, approvalID)
	if err != nil {
		s.answerCallback(ctx, callbackID, "Approval nicht gefunden.", true)
		return
	}
	if approval.UserID != userID {
		s.answerCallback(ctx, callbackID, "Diese Approval gehört einem anderen Konto.", true)
		return
	}
	if approval.Status != model.ApprovalPending {
		s.answerCallback(ctx, callbackID, "Approval wurde bereits entschieden.", false)
		s.editMessageMarkup(ctx, chatID, messageID, nil) // strip buttons
		return
	}

	status := model.ApprovalApproved
	verb := "freigegeben"
	if !approve {
		status = model.ApprovalRejected
		verb = "abgelehnt"
	}
	if err := s.approvalRepo.Resolve(ctx, approvalID, status, userID); err != nil {
		s.answerCallback(ctx, callbackID, "Konnte nicht gespeichert werden: "+err.Error(), true)
		return
	}

	s.answerCallback(ctx, callbackID, "Approval "+verb+".", false)
	// Replace buttons with a static "decided" footer so the chat stays clean.
	s.editMessageMarkup(ctx, chatID, messageID, [][]InlineKeyboardButton{
		{{Text: "✓ " + verb, CallbackData: "noop"}},
	})

	// On approve: actually run the tool. The HTTP approval handler does the
	// same — we mirror the flow so a Telegram-approved action behaves
	// identically to a UI-approved one. Run in a goroutine because the tool
	// can take seconds (SSH commands, backups) and we don't want to block
	// the callback-loop response.
	if approve && s.agentSvc != nil {
		go func(approval *model.AgentPendingApproval) {
			runCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			result, execErr := s.agentSvc.ExecuteApprovedTool(runCtx, userID, approval)
			if execErr != nil {
				s.sendMessage(runCtx, chatID, fmt.Sprintf("⚠️ Aktion `%s` fehlgeschlagen: %s",
					escapeTelegramMarkdown(approval.ToolName),
					escapeTelegramMarkdown(execErr.Error())))
				return
			}
			preview := string(result)
			if len(preview) > 400 {
				preview = preview[:400] + "…"
			}
			s.sendMessage(runCtx, chatID, fmt.Sprintf("✅ `%s` ausgeführt.\n\n```\n%s\n```",
				escapeTelegramMarkdown(approval.ToolName), preview))
		}(approval)
	}
}

func (s *BotService) processUpdate(ctx context.Context, msg *Message) {
	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID

	// Handle commands
	if strings.HasPrefix(text, "/") {
		s.handleCommand(ctx, chatID, text, msg.From)
		return
	}

	// Check for pending confirmation (autonomy level 1)
	s.confirmMu.Lock()
	pending, ok := s.pendingConfirmations[chatID]
	if ok {
		delete(s.pendingConfirmations, chatID)
	}
	s.confirmMu.Unlock()
	if ok {
		lower := strings.ToLower(text)
		if lower == "ja" || lower == "yes" || lower == "j" || lower == "y" {
			s.sendMessage(ctx, chatID, "Aktion wird ausgefuehrt...")
			s.executeChatRequest(ctx, chatID, pending.userID, pending.request)
		} else {
			s.sendMessage(ctx, chatID, "Aktion abgebrochen.")
		}
		return
	}

	// Forward to AI agent
	s.handleChatMessage(ctx, chatID, text)
}

func (s *BotService) handleCommand(ctx context.Context, chatID int64, text string, from *User) {
	parts := strings.Fields(text)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/start":
		// Handle deep link: /start CODE (from t.me/bot?start=CODE)
		if len(parts) >= 2 {
			code := parts[1]
			s.handleLink(ctx, chatID, code, from)
			return
		}
		s.sendMessage(ctx, chatID,
			"Willkommen beim Prometheus Bot!\n\n"+
				"Verwende /link <CODE> um dein Konto zu verknuepfen.\n"+
				"Verwende /help fuer weitere Befehle.")

	case "/link":
		if len(parts) < 2 {
			s.sendMessage(ctx, chatID, "Bitte gib deinen Verifikationscode an: /link <CODE>")
			return
		}
		code := parts[1]
		s.handleLink(ctx, chatID, code, from)

	case "/help":
		s.sendMessage(ctx, chatID,
			"Verfuegbare Befehle:\n\n"+
				"/start - Bot starten\n"+
				"/link <CODE> - Konto verknuepfen\n"+
				"/new - Neue Konversation starten\n"+
				"/help - Diese Hilfe anzeigen\n\n"+
				"Sende eine Nachricht, um mit dem KI-Assistenten zu sprechen.")

	case "/new":
		s.handleNewConversation(ctx, chatID)

	default:
		s.sendMessage(ctx, chatID, "Unbekannter Befehl. Verwende /help fuer Hilfe.")
	}
}

func (s *BotService) handleLink(ctx context.Context, chatID int64, code string, from *User) {
	link, err := s.linkRepo.GetByVerificationCode(ctx, code)
	if err != nil {
		s.sendMessage(ctx, chatID, "Ungueltiger oder abgelaufener Verifikationscode.")
		return
	}

	username := ""
	if from != nil {
		username = from.Username
	}

	if err := s.linkRepo.Verify(ctx, link.ID, chatID, username); err != nil {
		slog.Error("failed to verify telegram link", slog.Any("error", err))
		s.sendMessage(ctx, chatID, "Fehler bei der Verifikation. Bitte versuche es erneut.")
		return
	}

	s.sendMessage(ctx, chatID, "Konto erfolgreich verknuepft! Du kannst jetzt mit dem KI-Assistenten chatten.")
}

func (s *BotService) handleNewConversation(ctx context.Context, chatID int64) {
	link, err := s.linkRepo.GetByTelegramChatID(ctx, chatID)
	if err != nil {
		s.sendMessage(ctx, chatID, "Dein Konto ist nicht verknuepft. Verwende /link <CODE>.")
		return
	}

	// Create new telegram conversation mapping
	tc := &model.TelegramConversation{
		TelegramChatID: chatID,
	}
	if err := s.convRepo.Create(ctx, tc); err != nil {
		slog.Error("failed to create telegram conversation", slog.Any("error", err))
		s.sendMessage(ctx, chatID, "Fehler beim Erstellen der Konversation.")
		return
	}

	_ = link // used for authorization check
	s.sendMessage(ctx, chatID, "Neue Konversation gestartet. Sende deine erste Nachricht!")
}

func (s *BotService) handleChatMessage(ctx context.Context, chatID int64, text string) {
	link, err := s.linkRepo.GetByTelegramChatID(ctx, chatID)
	if err != nil {
		s.sendMessage(ctx, chatID, "Dein Konto ist nicht verknuepft. Verwende /link <CODE>.")
		return
	}

	// Get or create conversation mapping
	tc, err := s.convRepo.GetByChatID(ctx, chatID)
	if err != nil {
		// Create new mapping
		tc = &model.TelegramConversation{
			TelegramChatID: chatID,
		}
		if err := s.convRepo.Create(ctx, tc); err != nil {
			slog.Error("failed to create telegram conversation", slog.Any("error", err))
			s.sendMessage(ctx, chatID, "Interner Fehler.")
			return
		}
	}

	// Use configured model from agent_config
	configuredModel := s.getConfiguredModel(ctx)

	// Build chat request
	chatReq := model.ChatRequest{
		Message:        text,
		ConversationID: tc.ConversationID,
		Model:          configuredModel,
	}

	s.executeChatRequest(ctx, chatID, link.UserID, chatReq)
}

func (s *BotService) executeChatRequest(ctx context.Context, chatID int64, userID uuid.UUID, chatReq model.ChatRequest) {
	resp, err := s.agentSvc.Chat(ctx, userID, chatReq)
	if err != nil {
		slog.Error("agent chat failed for telegram",
			slog.Int64("chat_id", chatID),
			slog.Any("error", err))
		s.sendMessage(ctx, chatID, "Fehler bei der Verarbeitung deiner Nachricht.")
		return
	}

	// Update conversation mapping if new
	tc, _ := s.convRepo.GetByChatID(ctx, chatID)
	if tc != nil && tc.ConversationID == nil {
		convID := resp.ConversationID
		if err := s.convRepo.UpdateConversationID(ctx, tc.ID, convID); err != nil {
			slog.Error("failed to update telegram conversation mapping", slog.Any("error", err))
		}
	}

	// Format tool calls for Telegram
	var parts []string
	if len(resp.ToolCalls) > 0 {
		parts = append(parts, formatToolCallsForTelegram(resp.ToolCalls))
	}

	// Format and send response
	if resp.Message.Content != "" {
		parts = append(parts, FormatAgentResponse(resp.Message.Content))
	}

	if len(parts) == 0 {
		s.sendMessage(ctx, chatID, "Keine Antwort erhalten.")
		return
	}

	fullResponse := strings.Join(parts, "\n\n")
	s.sendMessage(ctx, chatID, fullResponse)
}

// formatToolCallsForTelegram formats tool call results nicely for Telegram.
func formatToolCallsForTelegram(toolCalls []model.AgentToolCall) string {
	if len(toolCalls) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, tc := range toolCalls {
		icon := "✅"
		if tc.Status == "error" {
			icon = "❌"
		} else if tc.Status == "running" {
			icon = "⏳"
		}

		sb.WriteString(fmt.Sprintf("%s *%s*", icon, escapeTelegramMarkdown(tc.ToolName)))
		if tc.DurationMs > 0 {
			sb.WriteString(fmt.Sprintf(" (%dms)", tc.DurationMs))
		}
		sb.WriteString("\n")

		// Show brief result for errors
		if tc.Status == "error" && tc.Result != nil {
			var errResult map[string]string
			if json.Unmarshal(tc.Result, &errResult) == nil {
				if errMsg, ok := errResult["error"]; ok {
					sb.WriteString(fmt.Sprintf("  _Fehler: %s_\n", escapeTelegramMarkdown(errMsg)))
				}
			}
		}
	}

	return sb.String()
}

// SendDirect is a public wrapper around sendMessage so that other services
// (proactive push, briefing, intelligence) can deliver messages to a specific
// chat without depending on the internal helper. Errors are logged but not
// returned — pushes are best-effort.
func (s *BotService) SendDirect(ctx context.Context, chatID int64, text string) {
	s.sendMessage(ctx, chatID, text)
}

// SendApprovalRequest broadcasts an approval prompt with inline ✓/✗ buttons
// to the user who owns the approval. Returns true if the message was sent.
//
// We target the specific user's chat (not all linked users) — approvals are
// auth-scoped, so flooding every admin would be wrong.
func (s *BotService) SendApprovalRequest(
	ctx context.Context,
	userID uuid.UUID,
	approvalID uuid.UUID,
	toolName string,
	summary string,
) bool {
	if s == nil || s.linkRepo == nil {
		return false
	}
	link, err := s.linkRepo.GetByUserID(ctx, userID)
	if err != nil || link == nil || link.TelegramChatID == nil || !link.IsVerified {
		return false
	}
	text := fmt.Sprintf("🛂 *Freigabe nötig*\nDer Agent möchte `%s` ausführen.", escapeTelegramMarkdown(toolName))
	if summary != "" {
		text += "\n\n" + escapeTelegramMarkdown(summary)
	}
	keyboard := [][]InlineKeyboardButton{
		{
			{Text: "✓ Erlauben", CallbackData: "approve:" + approvalID.String()},
			{Text: "✗ Ablehnen", CallbackData: "deny:" + approvalID.String()},
		},
	}
	return s.sendWithKeyboard(ctx, *link.TelegramChatID, text, keyboard)
}

// sendWithKeyboard sends a Markdown message with an inline keyboard. Falls
// back to plain text if Markdown parsing fails, mirroring sendMessage's
// retry path.
func (s *BotService) sendWithKeyboard(
	ctx context.Context,
	chatID int64,
	text string,
	keyboard [][]InlineKeyboardButton,
) bool {
	payload := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	if len(keyboard) > 0 {
		payload["reply_markup"] = map[string]any{
			"inline_keyboard": keyboard,
		}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("marshal telegram keyboard payload", slog.Any("error", err))
		return false
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := telegramBotHTTPClient.Do(req)
	if err != nil {
		slog.Error("telegram sendWithKeyboard", slog.Any("error", err))
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// answerCallback acks the callback_query so Telegram stops showing the
// loading spinner on the button. `text` shows a small toast; pass empty
// string to ack silently.
func (s *BotService) answerCallback(ctx context.Context, callbackID string, text string, alert bool) {
	payload := map[string]any{"callback_query_id": callbackID}
	if text != "" {
		payload["text"] = text
		payload["show_alert"] = alert
	}
	data, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", s.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := telegramBotHTTPClient.Do(req)
	if err == nil && resp != nil {
		resp.Body.Close()
	}
}

// editMessageMarkup replaces the inline keyboard on an existing message.
// Pass nil to remove the keyboard entirely. Used after an approval is
// resolved so the buttons can't be tapped twice.
func (s *BotService) editMessageMarkup(ctx context.Context, chatID, messageID int64, keyboard [][]InlineKeyboardButton) {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	if keyboard != nil {
		payload["reply_markup"] = map[string]any{"inline_keyboard": keyboard}
	} else {
		payload["reply_markup"] = map[string]any{"inline_keyboard": [][]InlineKeyboardButton{}}
	}
	data, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageReplyMarkup", s.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := telegramBotHTTPClient.Do(req)
	if err == nil && resp != nil {
		resp.Body.Close()
	}
}

// BroadcastToLinkedUsers sends the given text to every Telegram chat that has
// completed the verification flow. Used by the proactive push pipeline to keep
// the admin "in the loop" without requiring them to open the UI.
//
// Returns the number of chats reached. Failures are logged per-chat but never
// abort the broadcast — one offline user must not break delivery to the rest.
func (s *BotService) BroadcastToLinkedUsers(ctx context.Context, text string) int {
	if s == nil || s.linkRepo == nil {
		return 0
	}
	links, err := s.linkRepo.ListVerified(ctx)
	if err != nil {
		slog.Warn("telegram broadcast: list verified failed", slog.Any("error", err))
		return 0
	}
	count := 0
	for _, link := range links {
		if link == nil || link.TelegramChatID == nil {
			continue
		}
		s.sendMessage(ctx, *link.TelegramChatID, text)
		count++
	}
	return count
}

func (s *BotService) sendMessage(ctx context.Context, chatID int64, text string) {
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("marshal telegram send payload", slog.Any("error", err))
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		slog.Error("create telegram send request", slog.Any("error", err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := telegramBotHTTPClient.Do(req)
	if err != nil {
		slog.Error("send telegram message", slog.Any("error", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Retry without Markdown if parsing failed
		payload["parse_mode"] = ""
		data, _ = json.Marshal(payload)
		req2, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
		req2.Header.Set("Content-Type", "application/json")
		resp2, err := telegramBotHTTPClient.Do(req2)
		if err != nil {
			slog.Error("send telegram message retry", slog.Any("error", err))
			return
		}
		defer resp2.Body.Close()
	}
}

// GenerateVerificationCode creates a random verification code with 48-bit entropy.
func GenerateVerificationCode() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b)[:12])
}
