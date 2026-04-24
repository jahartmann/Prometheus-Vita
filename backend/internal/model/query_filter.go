package model

import (
	"time"

	"github.com/google/uuid"
)

type QueryFilter struct {
	Limit    int
	Offset   int
	Source   string
	Status   string
	Severity string
	Category string
	Type     string
	Method   string
	Query    string
	NodeID   *uuid.UUID
	UserID   *uuid.UUID
	From     *time.Time
	To       *time.Time
}

func (f QueryFilter) NormalizedLimit(defaultLimit, maxLimit int) int {
	if f.Limit <= 0 {
		return defaultLimit
	}
	if maxLimit > 0 && f.Limit > maxLimit {
		return maxLimit
	}
	return f.Limit
}
