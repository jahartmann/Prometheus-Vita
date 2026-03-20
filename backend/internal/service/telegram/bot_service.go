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
	lastOffset      int64
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
) *BotService {
	return &BotService{
		botToken:             botToken,
		agentSvc:             agentSvc,
		linkRepo:             linkRepo,
		convRepo:             convRepo,
		agentConfigRepo:      agentConfigRepo,
		pendingConfirmations: make(map[int64]*pendingAction),
	}
}

// Telegram API types

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	From      *User  `json:"from"`
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
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=1&limit=20",
		s.botToken, s.lastOffset+1)

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
		s.lastOffset = update.UpdateID
		if update.Message != nil && update.Message.Text != "" {
			s.processUpdate(ctx, update.Message)
		}
	}

	return nil
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
