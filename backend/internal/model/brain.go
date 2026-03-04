package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type BrainEntry struct {
	ID             uuid.UUID       `json:"id"`
	Category       string          `json:"category"`
	Subject        string          `json:"subject"`
	Content        string          `json:"content"`
	Metadata       json.RawMessage `json:"metadata"`
	RelevanceScore float64         `json:"relevance_score"`
	AccessCount    int             `json:"access_count"`
	LastAccessedAt *time.Time      `json:"last_accessed_at,omitempty"`
	CreatedBy      *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreateBrainEntryRequest struct {
	Category string `json:"category" validate:"required"`
	Subject  string `json:"subject" validate:"required"`
	Content  string `json:"content" validate:"required"`
}
