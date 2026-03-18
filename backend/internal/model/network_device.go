package model

import (
	"time"

	"github.com/google/uuid"
)

type NetworkDevice struct {
	ID        uuid.UUID `json:"id"`
	NodeID    uuid.UUID `json:"node_id"`
	IP        string    `json:"ip"`
	MAC       string    `json:"mac,omitempty"`
	Hostname  string    `json:"hostname,omitempty"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	IsKnown   bool      `json:"is_known"`
}

type UpdateNetworkDeviceRequest struct {
	Hostname *string `json:"hostname,omitempty"`
	IsKnown  *bool   `json:"is_known,omitempty"`
}
