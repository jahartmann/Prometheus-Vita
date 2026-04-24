package model

import (
	"time"

	"github.com/google/uuid"
)

type OperationTask struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Detail    string    `json:"detail"`
	Status    string    `json:"status"`
	Severity  string    `json:"severity"`
	Progress  int       `json:"progress"`
	EntityID  string    `json:"entity_id,omitempty"`
	NodeID    *uuid.UUID `json:"node_id,omitempty"`
	Href      string    `json:"href"`
	DueAt     *time.Time `json:"due_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type TimelineEvent struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Severity  string    `json:"severity"`
	Title     string    `json:"title"`
	Detail    string    `json:"detail"`
	Actor     string    `json:"actor"`
	EntityID  string    `json:"entity_id,omitempty"`
	NodeID    *uuid.UUID `json:"node_id,omitempty"`
	Href      string    `json:"href"`
	CreatedAt time.Time `json:"created_at"`
}

type RCACandidate struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Severity       string    `json:"severity"`
	NodeID         *uuid.UUID `json:"node_id,omitempty"`
	Evidence       []string  `json:"evidence"`
	Recommendation string    `json:"recommendation"`
	Href           string    `json:"href"`
}

type RCAAnalyzeRequest struct {
	Prompt  string     `json:"prompt,omitempty"`
	NodeID  *uuid.UUID `json:"node_id,omitempty"`
	UseLLM  bool       `json:"use_llm,omitempty"`
	Model   string     `json:"model,omitempty"`
	Limit   int        `json:"limit,omitempty"`
}

type RCAAnalyzeResponse struct {
	Summary     string          `json:"summary"`
	ModelUsed   string          `json:"model_used,omitempty"`
	Candidates  []RCACandidate  `json:"candidates"`
	Timeline    []TimelineEvent `json:"timeline"`
	GeneratedAt time.Time       `json:"generated_at"`
}

type KnowledgeGraphNode struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Label    string            `json:"label"`
	Status   string            `json:"status,omitempty"`
	NodeID   *uuid.UUID        `json:"node_id,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type KnowledgeGraphEdge struct {
	ID     string `json:"id"`
	From   string `json:"from"`
	To     string `json:"to"`
	Type   string `json:"type"`
	Label  string `json:"label,omitempty"`
	Status string `json:"status,omitempty"`
}

type KnowledgeGraphStats struct {
	Nodes        int `json:"nodes"`
	VMs          int `json:"vms"`
	Devices      int `json:"devices"`
	Services     int `json:"services"`
	Dependencies int `json:"dependencies"`
}

type KnowledgeGraphResponse struct {
	Nodes       []KnowledgeGraphNode `json:"nodes"`
	Edges       []KnowledgeGraphEdge `json:"edges"`
	Stats       KnowledgeGraphStats  `json:"stats"`
	GeneratedAt time.Time            `json:"generated_at"`
}

type OperationsReportRequest struct {
	Prompt   string `json:"prompt,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Severity string `json:"severity,omitempty"`
	Query    string `json:"query,omitempty"`
	UseLLM   bool   `json:"use_llm,omitempty"`
	Model    string `json:"model,omitempty"`
}

type OperationsReportResponse struct {
	Text        string         `json:"text"`
	ModelUsed   string         `json:"model_used,omitempty"`
	Counts      map[string]int `json:"counts"`
	GeneratedAt time.Time      `json:"generated_at"`
}
