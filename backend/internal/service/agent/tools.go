package agent

import (
	"context"
	"encoding/json"

	"github.com/antigravity/prometheus/internal/llm"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage
	Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
	ReadOnly() bool
}

type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *ToolRegistry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

func (r *ToolRegistry) SecurityCatalog() []ToolCatalogEntry {
	entries := make([]ToolCatalogEntry, 0, len(r.tools))
	for _, t := range r.tools {
		entries = append(entries, ToolCatalogEntry{
			Name:          t.Name(),
			Description:   t.Description(),
			ReadOnly:      t.ReadOnly(),
			Security:      securityForTool(t),
			SupportsDryRun: toolSupportsDryRun(t),
		})
	}
	return entries
}

type ToolCatalogEntry struct {
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	ReadOnly      bool         `json:"read_only"`
	Security      ToolSecurity `json:"security"`
	SupportsDryRun bool        `json:"supports_dry_run"`
}

func (r *ToolRegistry) ToDefinitions() []llm.ToolDefinition {
	defs := make([]llm.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, llm.ToolDefinition{
			Type: "function",
			Function: llm.ToolDefinitionFunc{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return defs
}
