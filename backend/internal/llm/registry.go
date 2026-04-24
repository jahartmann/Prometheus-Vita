package llm

import (
	"fmt"
	"strings"
	"sync"
)

type Registry struct {
	mu           sync.RWMutex
	providers    map[string]Provider
	defaultModel string
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

func (r *Registry) Register(name string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = provider
}

func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("LLM provider '%s' not found", name)
	}
	return p, nil
}

func (r *Registry) GetForModel(model string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.providers {
		for _, m := range p.Models() {
			if strings.EqualFold(m, model) {
				return p, nil
			}
		}
	}

	// Try matching by prefix
	modelLower := strings.ToLower(model)
	if strings.HasPrefix(modelLower, "gpt") {
		if p, ok := r.providers["openai"]; ok {
			return p, nil
		}
	}
	if strings.HasPrefix(modelLower, "claude") {
		if p, ok := r.providers["anthropic"]; ok {
			return p, nil
		}
	}

	// Fall back to ollama for unknown models
	if p, ok := r.providers["ollama"]; ok {
		return p, nil
	}

	return nil, fmt.Errorf("no provider found for model '%s'", model)
}

func (r *Registry) SetDefault(model string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultModel = model
}

func (r *Registry) DefaultModel() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.defaultModel != "" {
		return r.defaultModel
	}
	return "llama3"
}

func (r *Registry) ListModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var models []string
	for _, p := range r.providers {
		models = append(models, p.Models()...)
	}
	return models
}

func (r *Registry) Reload(ollamaURL, openaiKey, anthropicKey string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ollamaURL != "" {
		if p, ok := r.providers["ollama"]; ok {
			if op, ok := p.(*OllamaProvider); ok {
				op.SetBaseURL(ollamaURL)
			}
		} else {
			r.providers["ollama"] = NewOllamaProvider(ollamaURL)
		}
	}
	if openaiKey != "" {
		r.providers["openai"] = NewOpenAIProvider(openaiKey, "")
	} else {
		delete(r.providers, "openai")
	}
	if anthropicKey != "" {
		r.providers["anthropic"] = NewAnthropicProvider(anthropicKey)
	} else {
		delete(r.providers, "anthropic")
	}
}
