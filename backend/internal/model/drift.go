package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DriftStatus string

const (
	DriftStatusPending   DriftStatus = "pending"
	DriftStatusRunning   DriftStatus = "running"
	DriftStatusCompleted DriftStatus = "completed"
	DriftStatusFailed    DriftStatus = "failed"
)

type DriftCheck struct {
	ID                uuid.UUID       `json:"id"`
	NodeID            uuid.UUID       `json:"node_id"`
	Status            DriftStatus     `json:"status"`
	TotalFiles        int             `json:"total_files"`
	ChangedFiles      int             `json:"changed_files"`
	AddedFiles        int             `json:"added_files"`
	RemovedFiles      int             `json:"removed_files"`
	Details           json.RawMessage `json:"details,omitempty"`
	AIAnalysis        json.RawMessage `json:"ai_analysis,omitempty"`
	ErrorMessage      string          `json:"error_message,omitempty"`
	BaselineUpdatedAt *time.Time      `json:"baseline_updated_at,omitempty"`
	CheckedAt         time.Time       `json:"checked_at"`
	CreatedAt         time.Time       `json:"created_at"`
}

type DriftFileDetail struct {
	FilePath     string          `json:"file_path"`
	Status       string          `json:"status"` // added, removed, modified, unchanged
	Diff         string          `json:"diff,omitempty"`
	Acknowledged bool            `json:"acknowledged"`
	AIFileAnalysis *AIFileAnalysis `json:"ai_file_analysis,omitempty"`
}

// AIFileAnalysis contains the AI-generated analysis for a single file change.
type AIFileAnalysis struct {
	FilePath       string `json:"file_path"`
	Severity       int    `json:"severity"`        // 1-10
	SeverityReason string `json:"severity_reason"`
	Category       string `json:"category"`        // Security, Performance, Network, Configuration, Cosmetic
	RiskAssessment string `json:"risk_assessment"`
	Recommendation string `json:"recommendation"`  // fix, accept, monitor
	Summary        string `json:"summary"`         // German human-readable summary
}

// AIAnalysisResult is the top-level AI analysis stored in the drift check.
type AIAnalysisResult struct {
	AnalyzedAt    time.Time       `json:"analyzed_at"`
	Model         string          `json:"model"`
	FileAnalyses  []AIFileAnalysis `json:"file_analyses"`
	OverallSeverity int           `json:"overall_severity"`
	OverallSummary  string        `json:"overall_summary"`
}

// CompareNodesRequest is the request body for cross-node comparison.
type CompareNodesRequest struct {
	FilePaths []string  `json:"file_paths"`
	NodeIDs   []string  `json:"node_ids"`
}

// NodeFileContent holds the content of a file from a specific node.
type NodeFileContent struct {
	NodeID   string `json:"node_id"`
	NodeName string `json:"node_name"`
	Content  string `json:"content"`
	Error    string `json:"error,omitempty"`
}

// NodeComparisonEntry represents a single file comparison across nodes.
type NodeComparisonEntry struct {
	FilePath    string            `json:"file_path"`
	NodeFiles   []NodeFileContent `json:"node_files"`
	Differences []NodeDifference  `json:"differences"`
}

// NodeDifference represents a diff between two nodes for a single file.
type NodeDifference struct {
	NodeA     string `json:"node_a"`
	NodeAName string `json:"node_a_name"`
	NodeB     string `json:"node_b"`
	NodeBName string `json:"node_b_name"`
	Diff      string `json:"diff"`
	Identical bool   `json:"identical"`
}

// CompareNodesResponse is the response for cross-node comparison.
type CompareNodesResponse struct {
	Comparisons []NodeComparisonEntry `json:"comparisons"`
}
