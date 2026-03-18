package model

import (
	"time"

	"github.com/google/uuid"
)

type NetworkPort struct {
	ID             uuid.UUID `json:"id"`
	DeviceID       uuid.UUID `json:"device_id"`
	Port           int       `json:"port"`
	Protocol       string    `json:"protocol"`
	State          string    `json:"state"`
	ServiceName    string    `json:"service_name,omitempty"`
	ServiceVersion string    `json:"service_version,omitempty"`
	LastSeen       time.Time `json:"last_seen"`
}
