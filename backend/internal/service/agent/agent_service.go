package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/antigravity/prometheus/internal/llm"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
)

const maxToolIterations = 10

func buildSystemPrompt(autonomyLevel int) string {
	autonomyDesc := ""
	switch autonomyLevel {
	case model.AutonomyReadOnly:
		autonomyDesc = "Dein Autonomie-Level ist 'Nur Lesen'. Du darfst nur lesende Tools verwenden. Schreibende Aktionen sind nicht erlaubt."
	case model.AutonomyConfirm:
		autonomyDesc = "Dein Autonomie-Level ist 'Mit Bestaetigung'. Schreibende Aktionen erfordern eine Genehmigung des Benutzers."
	case model.AutonomyFullAuto:
		autonomyDesc = "Dein Autonomie-Level ist 'Voll-Automatisch'. Du darfst alle Aktionen sofort ausfuehren."
	}

	return fmt.Sprintf(`Du bist Prometheus, ein autonomer KI-Assistent fuer Proxmox-Infrastruktur-Management.

Deine Faehigkeiten:
- Nodes und VMs ueberwachen und verwalten
- VMs starten, stoppen und migrieren
- Backups erstellen und wiederherstellen
- SSH-Befehle auf Nodes ausfuehren
- Metriken und Anomalien analysieren
- Konfigurationsdrift erkennen
- Updates pruefen und Empfehlungen geben
- Wissen speichern und abrufen

%s

Du bist proaktiv: Wenn der Benutzer eine Aufgabe beschreibt, fuehre sie eigenstaendig aus.
Nutze deine Tools um Informationen zu sammeln und Aktionen durchzufuehren.
Erklaere was du tust und warum.
Antworte immer auf Deutsch. Sei praezise und hilfreich.`, autonomyDesc)
}

type Service struct {
	llmRegistry     *llm.Registry
	toolRegistry    *ToolRegistry
	convRepo        repository.ChatConversationRepository
	msgRepo         repository.ChatMessageRepository
	toolCallRepo    repository.ToolCallRepository
	approvalRepo    repository.ApprovalRepository
	userRepo        repository.UserRepository
	agentConfigRepo repository.AgentConfigRepository
}

func NewService(
	llmRegistry *llm.Registry,
	toolRegistry *ToolRegistry,
	convRepo repository.ChatConversationRepository,
	msgRepo repository.ChatMessageRepository,
	toolCallRepo repository.ToolCallRepository,
	approvalRepo repository.ApprovalRepository,
	userRepo repository.UserRepository,
	agentConfigRepo repository.AgentConfigRepository,
) *Service {
	return &Service{
		llmRegistry:     llmRegistry,
		toolRegistry:    toolRegistry,
		convRepo:        convRepo,
		msgRepo:         msgRepo,
		toolCallRepo:    toolCallRepo,
		approvalRepo:    approvalRepo,
		userRepo:        userRepo,
		agentConfigRepo: agentConfigRepo,
	}
}

func (s *Service) GetTool(name string) (Tool, bool) {
	return s.toolRegistry.Get(name)
}

// getConfiguredModel reads the model from agent_config table. Returns empty string if not configured.
func (s *Service) getConfiguredModel(ctx context.Context) string {
	if s.agentConfigRepo == nil {
		return ""
	}
	model, err := s.agentConfigRepo.Get(ctx, "llm_model")
	if err != nil {
		return ""
	}
	return model
}

func (s *Service) Chat(ctx context.Context, userID uuid.UUID, req model.ChatRequest) (*model.ChatResponse, error) {
	// Resolve model: request > agent_config > registry default
	resolvedModel := req.Model
	if resolvedModel == "" || resolvedModel == "default" {
		resolvedModel = s.getConfiguredModel(ctx)
	}
	if resolvedModel == "" {
		resolvedModel = s.llmRegistry.DefaultModel()
	}

	// Get user autonomy level for system prompt
	autonomyLevel := model.AutonomyConfirm // default
	user, err := s.userRepo.GetByID(ctx, userID)
	if err == nil {
		autonomyLevel = user.AutonomyLevel
	}

	// 1. Get or create conversation
	var conv *model.ChatConversation
	isNew := false

	if req.ConversationID != nil {
		conv, err = s.convRepo.GetByID(ctx, *req.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("get conversation: %w", err)
		}
	} else {
		conv = &model.ChatConversation{
			UserID: userID,
			Title:  "Neue Konversation",
			Model:  resolvedModel,
		}
		if err := s.convRepo.Create(ctx, conv); err != nil {
			return nil, fmt.Errorf("create conversation: %w", err)
		}
		isNew = true
	}

	// 2. Load existing messages
	existingMsgs, err := s.msgRepo.ListByConversation(ctx, conv.ID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	// 3. Build LLM messages with autonomy-aware system prompt
	llmMessages := []llm.Message{
		{Role: "system", Content: buildSystemPrompt(autonomyLevel)},
	}
	for _, m := range existingMsgs {
		msg := llm.Message{
			Role:       string(m.Role),
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		if m.ToolCalls != nil {
			var toolCalls []llm.ToolCall
			if err := json.Unmarshal(m.ToolCalls, &toolCalls); err == nil {
				msg.ToolCalls = toolCalls
			}
		}
		llmMessages = append(llmMessages, msg)
	}

	// 4. Add user message to DB
	userMsg := &model.ChatMessage{
		ConversationID: conv.ID,
		Role:           model.RoleUser,
		Content:        req.Message,
	}
	if err := s.msgRepo.Create(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}

	llmMessages = append(llmMessages, llm.Message{
		Role:    "user",
		Content: req.Message,
	})

	// 5. Get LLM provider - use resolved model, fallback to conversation model
	modelName := resolvedModel
	if modelName == "" {
		modelName = conv.Model
	}
	if modelName == "" || modelName == "default" {
		modelName = s.llmRegistry.DefaultModel()
	}
	provider, err := s.llmRegistry.GetForModel(modelName)
	if err != nil {
		return nil, fmt.Errorf("get LLM provider: %w", err)
	}

	// 6. Tool call loop
	toolDefs := s.toolRegistry.ToDefinitions()
	var allToolCalls []model.AgentToolCall

	for i := 0; i < maxToolIterations; i++ {
		llmReq := llm.CompletionRequest{
			Model:    modelName,
			Messages: llmMessages,
			Tools:    toolDefs,
		}

		llmResp, err := provider.Complete(ctx, llmReq)
		if err != nil {
			return nil, fmt.Errorf("LLM completion failed: %w", err)
		}

		if len(llmResp.ToolCalls) == 0 {
			// No tool calls, save assistant response
			assistantMsg := &model.ChatMessage{
				ConversationID: conv.ID,
				Role:           model.RoleAssistant,
				Content:        llmResp.Content,
			}
			if err := s.msgRepo.Create(ctx, assistantMsg); err != nil {
				return nil, fmt.Errorf("save assistant message: %w", err)
			}

			// Auto-generate title from first user message if new conversation
			if isNew {
				s.autoGenerateTitle(ctx, conv.ID, req.Message)
			}

			return &model.ChatResponse{
				ConversationID: conv.ID,
				Message:        *assistantMsg,
				ToolCalls:      allToolCalls,
			}, nil
		}

		// Save assistant message with tool calls
		toolCallsJSON, _ := json.Marshal(llmResp.ToolCalls)
		assistantToolMsg := &model.ChatMessage{
			ConversationID: conv.ID,
			Role:           model.RoleAssistant,
			Content:        llmResp.Content,
			ToolCalls:      toolCallsJSON,
		}
		if err := s.msgRepo.Create(ctx, assistantToolMsg); err != nil {
			return nil, fmt.Errorf("save assistant tool message: %w", err)
		}

		llmMessages = append(llmMessages, llm.Message{
			Role:      "assistant",
			Content:   llmResp.Content,
			ToolCalls: llmResp.ToolCalls,
		})

		// Execute each tool call
		for _, tc := range llmResp.ToolCalls {
			toolResult := s.executeTool(ctx, userID, conv.ID, assistantToolMsg.ID, tc, &allToolCalls)

			llmMessages = append(llmMessages, llm.Message{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: tc.ID,
			})

			// Save tool result message
			toolMsg := &model.ChatMessage{
				ConversationID: conv.ID,
				Role:           model.RoleTool,
				Content:        toolResult,
				ToolCallID:     tc.ID,
			}
			if err := s.msgRepo.Create(ctx, toolMsg); err != nil {
				slog.Warn("failed to save tool message", slog.Any("error", err))
			}
		}
	}

	// Max iterations reached, return last state
	finalMsg := &model.ChatMessage{
		ConversationID: conv.ID,
		Role:           model.RoleAssistant,
		Content:        "Maximale Anzahl an Tool-Aufrufen erreicht. Bitte versuche es erneut.",
	}
	if err := s.msgRepo.Create(ctx, finalMsg); err != nil {
		slog.Warn("failed to save final message", slog.Any("error", err))
	}

	if isNew {
		s.autoGenerateTitle(ctx, conv.ID, req.Message)
	}

	return &model.ChatResponse{
		ConversationID: conv.ID,
		Message:        *finalMsg,
		ToolCalls:      allToolCalls,
	}, nil
}

func (s *Service) executeTool(ctx context.Context, userID uuid.UUID, convID uuid.UUID, msgID uuid.UUID, tc llm.ToolCall, allToolCalls *[]model.AgentToolCall) string {
	tool, ok := s.toolRegistry.Get(tc.Function.Name)
	if !ok {
		return fmt.Sprintf(`{"error": "Tool '%s' nicht gefunden"}`, tc.Function.Name)
	}

	// Check autonomy level for write tools
	if !tool.ReadOnly() {
		user, err := s.userRepo.GetByID(ctx, userID)
		if err == nil {
			switch user.AutonomyLevel {
			case model.AutonomyReadOnly:
				return `{"error": "Dein Autonomie-Level erlaubt keine schreibenden Operationen. Bitte erhoehe das Level in den Einstellungen."}`
			case model.AutonomyConfirm:
				// Create pending approval
				approval := &model.AgentPendingApproval{
					UserID:         userID,
					ConversationID: convID,
					MessageID:      msgID,
					ToolName:       tc.Function.Name,
					Arguments:      json.RawMessage(tc.Function.Arguments),
					Status:         model.ApprovalPending,
				}
				if s.approvalRepo != nil {
					if err := s.approvalRepo.Create(ctx, approval); err != nil {
						slog.Warn("failed to create approval", slog.Any("error", err))
					}
				}
				return fmt.Sprintf(`{"pending_approval": true, "approval_id": "%s", "message": "Diese Aktion erfordert eine Genehmigung. Bitte genehmige sie im Approval-Bereich."}`, approval.ID.String())
			// AutonomyFullAuto: fall through to execute
			}
		}
	}

	// Create tool call record
	agentTC := &model.AgentToolCall{
		MessageID: msgID,
		ToolName:  tc.Function.Name,
		Arguments: json.RawMessage(tc.Function.Arguments),
		Status:    "running",
	}
	if err := s.toolCallRepo.Create(ctx, agentTC); err != nil {
		slog.Warn("failed to create tool call record", slog.Any("error", err))
	}

	start := time.Now()
	result, err := tool.Execute(ctx, json.RawMessage(tc.Function.Arguments))
	duration := time.Since(start)
	durationMs := int(duration.Milliseconds())

	var resultJSON json.RawMessage
	status := "success"
	var resultStr string

	if err != nil {
		status = "error"
		errorResult := map[string]string{"error": err.Error()}
		resultJSON, _ = json.Marshal(errorResult)
		resultStr = string(resultJSON)
	} else {
		resultJSON = result
		resultStr = string(result)
	}

	// Update tool call record
	if err := s.toolCallRepo.UpdateResult(ctx, agentTC.ID, resultJSON, status, durationMs); err != nil {
		slog.Warn("failed to update tool call result", slog.Any("error", err))
	}

	agentTC.Result = resultJSON
	agentTC.Status = status
	agentTC.DurationMs = durationMs
	*allToolCalls = append(*allToolCalls, *agentTC)

	return resultStr
}

func (s *Service) autoGenerateTitle(ctx context.Context, convID uuid.UUID, firstMessage string) {
	title := firstMessage
	if len(title) > 60 {
		title = title[:57] + "..."
	}
	if title == "" {
		title = "Neue Konversation"
	}
	if err := s.convRepo.UpdateTitle(ctx, convID, title); err != nil {
		slog.Warn("failed to auto-generate conversation title", slog.Any("error", err))
	}
}

func (s *Service) ListConversations(ctx context.Context, userID uuid.UUID) ([]model.ChatConversation, error) {
	return s.convRepo.ListByUser(ctx, userID)
}

func (s *Service) GetConversation(ctx context.Context, id uuid.UUID) (*model.ChatConversation, error) {
	return s.convRepo.GetByID(ctx, id)
}

func (s *Service) GetMessages(ctx context.Context, conversationID uuid.UUID) ([]model.ChatMessage, error) {
	return s.msgRepo.ListByConversation(ctx, conversationID)
}

func (s *Service) DeleteConversation(ctx context.Context, id uuid.UUID) error {
	return s.convRepo.Delete(ctx, id)
}
