package handler

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/llm"
	"github.com/antigravity/prometheus/internal/repository"
	cryptoSvc "github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/labstack/echo/v4"
)

type AgentConfigHandler struct {
	agentConfigRepo repository.AgentConfigRepository
	llmRegistry     *llm.Registry
	ollamaProvider  *llm.OllamaProvider
	encryptor       *cryptoSvc.Encryptor
}

var allowedAgentConfigKeys = map[string]struct{}{
	"llm_provider":  {},
	"llm_model":     {},
	"ollama_url":    {},
	"openai_key":    {},
	"anthropic_key": {},
	"agent_approval_low_risk":      {},
	"agent_approval_medium_risk":   {},
	"agent_approval_high_risk":     {},
	"agent_approval_critical_risk": {},
	"agent_full_auto_allow_low_risk": {},
}

func NewAgentConfigHandler(
	agentConfigRepo repository.AgentConfigRepository,
	llmRegistry *llm.Registry,
	ollamaProvider *llm.OllamaProvider,
	encryptor *cryptoSvc.Encryptor,
) *AgentConfigHandler {
	return &AgentConfigHandler{
		agentConfigRepo: agentConfigRepo,
		llmRegistry:     llmRegistry,
		ollamaProvider:  ollamaProvider,
		encryptor:       encryptor,
	}
}

// GetConfig handles GET /agent/config.
// It returns all agent configuration key-value pairs.
func (h *AgentConfigHandler) GetConfig(c echo.Context) error {
	config, err := h.agentConfigRepo.List(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to load agent config")
	}

	return apiPkg.Success(c, sanitizeAgentConfigResponse(config))
}

// UpdateConfig handles PUT /agent/config.
// It saves configuration key-value pairs and triggers LLM registry reload.
func (h *AgentConfigHandler) UpdateConfig(c echo.Context) error {
	var req map[string]string
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	if err := validateAgentConfigUpdate(req); err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	ctx := c.Request().Context()
	currentConfig, err := h.agentConfigRepo.List(ctx)
	if err != nil {
		return apiPkg.InternalError(c, "failed to load agent config")
	}
	targetConfig := mergeAgentConfig(currentConfig, req)
	if err := validateAgentConfigTarget(targetConfig); err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}

	for key, value := range req {
		if isSecretAgentConfigKey(key) && strings.TrimSpace(value) == "" {
			continue
		}
		if isSecretAgentConfigKey(key) && h.encryptor != nil {
			encrypted, err := h.encryptor.Encrypt(value)
			if err != nil {
				return apiPkg.InternalError(c, "failed to encrypt agent secret")
			}
			value = encrypted
		}
		if err := h.agentConfigRepo.Set(ctx, key, value); err != nil {
			return apiPkg.InternalError(c, "failed to save agent config")
		}
	}

	// Reload LLM registry with updated config
	config, err := h.agentConfigRepo.List(ctx)
	if err == nil {
		runtimeConfig := h.runtimeAgentConfig(config)
		// Update ollamaProvider URL if changed
		if h.ollamaProvider != nil {
			newURL := runtimeConfig["ollama_url"]
			if newURL == "" {
				newURL = "http://localhost:11434"
			}
			h.ollamaProvider.SetBaseURL(newURL)
		}
		h.llmRegistry.Reload(
			runtimeConfig["ollama_url"],
			runtimeConfig["openai_key"],
			runtimeConfig["anthropic_key"],
		)
		if model, ok := runtimeConfig["llm_model"]; ok && model != "" {
			h.llmRegistry.SetDefault(model)
		}
	}

	return apiPkg.Success(c, sanitizeAgentConfigResponse(targetConfig))
}

func (h *AgentConfigHandler) RotateSecret(c echo.Context) error {
	keyName, err := secretConfigKey(c.Param("provider"))
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}
	var req struct {
		Key string `json:"key"`
	}
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}
	value := strings.TrimSpace(req.Key)
	if value == "" {
		return apiPkg.BadRequest(c, "API-Key darf nicht leer sein")
	}
	if len(value) > 4096 {
		return apiPkg.BadRequest(c, "API-Key ist zu lang")
	}
	if h.encryptor != nil {
		encrypted, encErr := h.encryptor.Encrypt(value)
		if encErr != nil {
			return apiPkg.InternalError(c, "failed to encrypt agent secret")
		}
		value = encrypted
	}
	ctx := c.Request().Context()
	if err := h.agentConfigRepo.Set(ctx, keyName, value); err != nil {
		return apiPkg.InternalError(c, "failed to rotate agent secret")
	}
	config, err := h.reloadRuntimeConfig(ctx)
	if err != nil {
		return apiPkg.InternalError(c, "failed to reload agent config")
	}
	return apiPkg.Success(c, sanitizeAgentConfigResponse(config))
}

func (h *AgentConfigHandler) DeleteSecret(c echo.Context) error {
	keyName, err := secretConfigKey(c.Param("provider"))
	if err != nil {
		return apiPkg.BadRequest(c, err.Error())
	}
	ctx := c.Request().Context()
	config, err := h.agentConfigRepo.List(ctx)
	if err != nil {
		return apiPkg.InternalError(c, "failed to load agent config")
	}
	if err := h.agentConfigRepo.Delete(ctx, keyName); err != nil {
		return apiPkg.InternalError(c, "failed to delete agent secret")
	}
	if (keyName == "openai_key" && config["llm_provider"] == "openai") || (keyName == "anthropic_key" && config["llm_provider"] == "anthropic") {
		_ = h.agentConfigRepo.Set(ctx, "llm_provider", "ollama")
		_ = h.agentConfigRepo.Set(ctx, "llm_model", "llama3")
	}
	config, err = h.reloadRuntimeConfig(ctx)
	if err != nil {
		return apiPkg.InternalError(c, "failed to reload agent config")
	}
	return apiPkg.Success(c, sanitizeAgentConfigResponse(config))
}

func validateAgentConfigUpdate(req map[string]string) error {
	for key, value := range req {
		if _, ok := allowedAgentConfigKeys[key]; !ok {
			return fmt.Errorf("Unbekannte Agent-Konfiguration: %s", key)
		}
		if len(value) > 4096 {
			return fmt.Errorf("Wert fuer %s ist zu lang", key)
		}
		switch key {
		case "llm_provider":
			if value != "ollama" && value != "openai" && value != "anthropic" {
				return fmt.Errorf("Ungueltiger LLM-Provider")
			}
		case "llm_model":
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("LLM-Modell darf nicht leer sein")
			}
		case "ollama_url":
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("Ollama-URL darf nicht leer sein")
			}
			if err := validateOllamaDiscoveryURL(value); err != nil {
				return err
			}
		case "agent_approval_low_risk", "agent_approval_medium_risk", "agent_approval_high_risk", "agent_approval_critical_risk", "agent_full_auto_allow_low_risk":
			if value != "true" && value != "false" {
				return fmt.Errorf("%s muss true oder false sein", key)
			}
		}
	}

	return nil
}

func validateAgentConfigTarget(config map[string]string) error {
	provider := config["llm_provider"]
	model := strings.TrimSpace(config["llm_model"])
	if provider == "openai" && strings.TrimSpace(config["openai_key"]) == "" {
		return fmt.Errorf("OpenAI API-Key ist fuer den Provider erforderlich")
	}
	if provider == "anthropic" && strings.TrimSpace(config["anthropic_key"]) == "" {
		return fmt.Errorf("Anthropic API-Key ist fuer den Provider erforderlich")
	}
	if provider == "openai" && !strings.HasPrefix(strings.ToLower(model), "gpt") {
		return fmt.Errorf("OpenAI-Provider erfordert ein GPT-Modell")
	}
	if provider == "anthropic" && !strings.HasPrefix(strings.ToLower(model), "claude") {
		return fmt.Errorf("Anthropic-Provider erfordert ein Claude-Modell")
	}
	if provider == "ollama" && (strings.HasPrefix(strings.ToLower(model), "gpt") || strings.HasPrefix(strings.ToLower(model), "claude")) {
		return fmt.Errorf("Ollama-Provider erfordert ein lokales Modell")
	}
	return nil
}

func mergeAgentConfig(current, updates map[string]string) map[string]string {
	merged := make(map[string]string, len(current)+len(updates))
	for key, value := range current {
		merged[key] = value
	}
	for key, value := range updates {
		if isSecretAgentConfigKey(key) && strings.TrimSpace(value) == "" {
			continue
		}
		merged[key] = value
	}
	return merged
}

func sanitizeAgentConfigResponse(config map[string]string) map[string]string {
	safe := make(map[string]string, len(config)+2)
	for key, value := range config {
		if isSecretAgentConfigKey(key) {
			safe[key] = ""
			safe[key+"_configured"] = fmt.Sprintf("%t", strings.TrimSpace(value) != "")
			continue
		}
		safe[key] = value
	}
	if _, ok := safe["openai_key_configured"]; !ok {
		safe["openai_key_configured"] = "false"
	}
	if _, ok := safe["anthropic_key_configured"]; !ok {
		safe["anthropic_key_configured"] = "false"
	}
	return safe
}

func isSecretAgentConfigKey(key string) bool {
	return key == "openai_key" || key == "anthropic_key"
}

func secretConfigKey(provider string) (string, error) {
	switch strings.ToLower(provider) {
	case "openai":
		return "openai_key", nil
	case "anthropic":
		return "anthropic_key", nil
	default:
		return "", fmt.Errorf("Unbekannter Secret-Provider")
	}
}

func (h *AgentConfigHandler) reloadRuntimeConfig(ctx context.Context) (map[string]string, error) {
	config, err := h.agentConfigRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	h.applyRuntimeConfig(config)
	return config, nil
}

func (h *AgentConfigHandler) applyRuntimeConfig(config map[string]string) {
	runtimeConfig := h.runtimeAgentConfig(config)
	if h.ollamaProvider != nil {
		newURL := runtimeConfig["ollama_url"]
		if newURL == "" {
			newURL = "http://localhost:11434"
		}
		h.ollamaProvider.SetBaseURL(newURL)
	}
	h.llmRegistry.Reload(
		runtimeConfig["ollama_url"],
		runtimeConfig["openai_key"],
		runtimeConfig["anthropic_key"],
	)
	if model, ok := runtimeConfig["llm_model"]; ok && model != "" {
		h.llmRegistry.SetDefault(model)
	}
}

func (h *AgentConfigHandler) runtimeAgentConfig(config map[string]string) map[string]string {
	runtimeConfig := make(map[string]string, len(config))
	for key, value := range config {
		if isSecretAgentConfigKey(key) && strings.TrimSpace(value) != "" && h.encryptor != nil {
			decrypted, err := h.encryptor.Decrypt(value)
			if err == nil {
				runtimeConfig[key] = decrypted
				continue
			}
			// Plaintext fallback keeps older deployments usable until the
			// next save rotates the secret into encrypted storage.
		}
		runtimeConfig[key] = value
	}
	return runtimeConfig
}

// GetModels handles GET /agent/models.
// It discovers available models from the active Ollama instance.
// Accepts optional ?url= query parameter to test a specific URL before saving.
func (h *AgentConfigHandler) GetModels(c echo.Context) error {
	if h.ollamaProvider == nil {
		return apiPkg.InternalError(c, "Ollama ist nicht konfiguriert. Bitte stelle sicher, dass Ollama laeuft.")
	}

	// If a URL is provided as query param, temporarily use it for discovery.
	testURL := c.QueryParam("url")
	if testURL != "" {
		if err := validateOllamaDiscoveryURL(testURL); err != nil {
			return apiPkg.BadRequest(c, err.Error())
		}
		tempProvider := llm.NewOllamaProvider(testURL)
		models, err := tempProvider.DiscoverModels(c.Request().Context())
		if err != nil {
			return apiPkg.InternalError(c, "Ollama nicht erreichbar unter "+testURL+". Bitte pruefen, ob Ollama laeuft.")
		}
		return apiPkg.Success(c, models)
	}

	models, err := h.ollamaProvider.DiscoverModels(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "Ollama nicht erreichbar unter "+h.ollamaProvider.BaseURL()+". Bitte pruefen, ob Ollama laeuft.")
	}

	return apiPkg.Success(c, models)
}

func validateOllamaDiscoveryURL(rawURL string) error {
	parsed, parseErr := url.Parse(rawURL)
	if parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("Ungueltige URL")
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("Ungueltige URL")
	}

	port := parsed.Port()
	if port != "" && port != "11434" {
		return fmt.Errorf("Ollama-Discovery ist nur auf Port 11434 erlaubt")
	}

	addrs, err := net.LookupHost(parsed.Hostname())
	if err != nil {
		return fmt.Errorf("Host nicht aufloesbar")
	}
	isLocalOrPrivateTarget := false
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil ||
			ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() ||
			ip.IsUnspecified() ||
			ip.IsMulticast() {
			return fmt.Errorf("Zugriff auf diese Adresse ist nicht erlaubt")
		}
		if ip.IsLoopback() || ip.IsPrivate() {
			isLocalOrPrivateTarget = true
		}
	}
	if isLocalOrPrivateTarget && port != "11434" {
		return fmt.Errorf("Lokale oder private Ollama-URLs muessen Port 11434 verwenden")
	}

	return nil
}
