package handler

import (
	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/llm"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/labstack/echo/v4"
)

type AgentConfigHandler struct {
	agentConfigRepo repository.AgentConfigRepository
	llmRegistry     *llm.Registry
	ollamaProvider  *llm.OllamaProvider
}

func NewAgentConfigHandler(
	agentConfigRepo repository.AgentConfigRepository,
	llmRegistry *llm.Registry,
	ollamaProvider *llm.OllamaProvider,
) *AgentConfigHandler {
	return &AgentConfigHandler{
		agentConfigRepo: agentConfigRepo,
		llmRegistry:     llmRegistry,
		ollamaProvider:  ollamaProvider,
	}
}

// GetConfig handles GET /agent/config.
// It returns all agent configuration key-value pairs.
func (h *AgentConfigHandler) GetConfig(c echo.Context) error {
	config, err := h.agentConfigRepo.List(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to load agent config")
	}

	return apiPkg.Success(c, config)
}

// UpdateConfig handles PUT /agent/config.
// It saves configuration key-value pairs and triggers LLM registry reload.
func (h *AgentConfigHandler) UpdateConfig(c echo.Context) error {
	var req map[string]string
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	ctx := c.Request().Context()
	for key, value := range req {
		if err := h.agentConfigRepo.Set(ctx, key, value); err != nil {
			return apiPkg.InternalError(c, "failed to save agent config")
		}
	}

	// Reload LLM registry with updated config
	config, err := h.agentConfigRepo.List(ctx)
	if err == nil {
		// Update ollamaProvider URL if changed
		if h.ollamaProvider != nil {
			newURL := config["ollama_url"]
			if newURL == "" {
				newURL = "http://localhost:11434"
			}
			h.ollamaProvider.SetBaseURL(newURL)
		}
		h.llmRegistry.Reload(
			config["ollama_url"],
			config["openai_key"],
			config["anthropic_key"],
		)
		if model, ok := config["llm_model"]; ok && model != "" {
			h.llmRegistry.SetDefault(model)
		}
	}

	return apiPkg.Success(c, req)
}

// GetModels handles GET /agent/models.
// It discovers available models from the active Ollama instance.
func (h *AgentConfigHandler) GetModels(c echo.Context) error {
	if h.ollamaProvider == nil {
		return apiPkg.InternalError(c, "Ollama ist nicht konfiguriert. Bitte stelle sicher, dass Ollama laeuft.")
	}

	models, err := h.ollamaProvider.DiscoverModels(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "Ollama nicht erreichbar unter "+h.ollamaProvider.BaseURL()+". Bitte pruefen, ob Ollama laeuft.")
	}

	return apiPkg.Success(c, models)
}
