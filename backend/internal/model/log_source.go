package model

import (
	"time"

	"github.com/google/uuid"
)

type LogSource struct {
	ID           uuid.UUID `json:"id"`
	NodeID       uuid.UUID `json:"node_id"`
	Path         string    `json:"path"`
	Enabled      bool      `json:"enabled"`
	IsBuiltin    bool      `json:"is_builtin"`
	ParserType   string    `json:"parser_type"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

type UpdateLogSourcesRequest struct {
	Sources []LogSourceUpdate `json:"sources" validate:"required"`
}

type LogSourceUpdate struct {
	Path    string `json:"path" validate:"required"`
	Enabled bool   `json:"enabled"`
}
