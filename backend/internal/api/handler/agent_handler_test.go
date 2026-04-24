package handler

import "testing"

func TestValidateOllamaDiscoveryURLAllowsLocalOllama(t *testing.T) {
	tests := []string{
		"http://localhost:11434",
		"http://127.0.0.1:11434",
		"http://10.0.0.5:11434",
	}

	for _, testURL := range tests {
		t.Run(testURL, func(t *testing.T) {
			if err := validateOllamaDiscoveryURL(testURL); err != nil {
				t.Fatalf("validateOllamaDiscoveryURL(%q) returned error: %v", testURL, err)
			}
		})
	}
}

func TestValidateOllamaDiscoveryURLRejectsUnsafeSchemesAndPorts(t *testing.T) {
	tests := []string{
		"file:///etc/passwd",
		"ftp://localhost:11434",
		"http://localhost",
		"http://127.0.0.1",
		"http://10.0.0.5",
		"http://127.0.0.1:22",
	}

	for _, testURL := range tests {
		t.Run(testURL, func(t *testing.T) {
			if err := validateOllamaDiscoveryURL(testURL); err == nil {
				t.Fatalf("validateOllamaDiscoveryURL(%q) returned nil error, want rejection", testURL)
			}
		})
	}
}

func TestValidateAgentConfigUpdateRejectsUnknownKeys(t *testing.T) {
	req := map[string]string{
		"llm_provider": "ollama",
		"surprise":     "nope",
	}

	if err := validateAgentConfigUpdate(req); err == nil {
		t.Fatal("expected unknown config key to be rejected")
	}
}

func TestValidateAgentConfigUpdateRejectsInvalidProvider(t *testing.T) {
	req := map[string]string{
		"llm_provider": "shell",
	}

	if err := validateAgentConfigUpdate(req); err == nil {
		t.Fatal("expected invalid provider to be rejected")
	}
}

func TestValidateAgentConfigUpdateRejectsUnsafeOllamaURL(t *testing.T) {
	req := map[string]string{
		"ollama_url": "http://127.0.0.1:22",
	}

	if err := validateAgentConfigUpdate(req); err == nil {
		t.Fatal("expected unsafe ollama url to be rejected")
	}
}

func TestValidateAgentConfigUpdateAcceptsSafeOllamaConfig(t *testing.T) {
	req := map[string]string{
		"llm_provider": "ollama",
		"llm_model":    "llama3",
		"ollama_url":   "http://localhost:11434",
	}

	if err := validateAgentConfigUpdate(req); err != nil {
		t.Fatalf("validateAgentConfigUpdate returned error: %v", err)
	}
}

func TestSanitizeAgentConfigResponseMasksSecrets(t *testing.T) {
	config := map[string]string{
		"llm_provider":  "openai",
		"openai_key":    "sk-secret",
		"anthropic_key": "",
	}

	safe := sanitizeAgentConfigResponse(config)

	if safe["openai_key"] != "" {
		t.Fatal("expected openai_key to be masked")
	}
	if safe["openai_key_configured"] != "true" {
		t.Fatalf("openai_key_configured = %q, want true", safe["openai_key_configured"])
	}
	if safe["anthropic_key_configured"] != "false" {
		t.Fatalf("anthropic_key_configured = %q, want false", safe["anthropic_key_configured"])
	}
}

func TestMergeAgentConfigDoesNotClearBlankSecrets(t *testing.T) {
	current := map[string]string{"openai_key": "sk-secret"}
	updates := map[string]string{"openai_key": ""}

	merged := mergeAgentConfig(current, updates)

	if merged["openai_key"] != "sk-secret" {
		t.Fatalf("openai_key = %q, want existing secret", merged["openai_key"])
	}
}

func TestValidateAgentConfigTargetRejectsProviderModelMismatch(t *testing.T) {
	config := map[string]string{
		"llm_provider": "openai",
		"llm_model":    "llama3",
		"openai_key":   "sk-secret",
	}

	if err := validateAgentConfigTarget(config); err == nil {
		t.Fatal("expected provider/model mismatch to be rejected")
	}
}
