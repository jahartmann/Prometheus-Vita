package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type NodeType string

const (
	NodeTypePVE NodeType = "pve"
	NodeTypePBS NodeType = "pbs"
)

func (n NodeType) IsValid() bool {
	switch n {
	case NodeTypePVE, NodeTypePBS:
		return true
	}
	return false
}

type Node struct {
	ID             uuid.UUID       `json:"id"`
	Name           string          `json:"name"`
	Type           NodeType        `json:"type"`
	Hostname       string          `json:"hostname"`
	Port           int             `json:"port"`
	APITokenID     string          `json:"-"` // encrypted in DB
	APITokenSecret string          `json:"-"` // encrypted in DB
	SSHPort        int             `json:"ssh_port"`
	SSHUser        string          `json:"ssh_user"`
	SSHPrivateKey  string          `json:"-"` // encrypted in DB, not exposed in JSON
	IsOnline       bool            `json:"is_online"`
	LastSeen       *time.Time      `json:"last_seen,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	EnvironmentID  *uuid.UUID      `json:"environment_id,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// DTOs

type CreateNodeRequest struct {
	Name           string          `json:"name" validate:"required"`
	Type           NodeType        `json:"type" validate:"required"`
	Hostname       string          `json:"hostname" validate:"required"`
	Port           int             `json:"port"`
	APITokenID     string          `json:"api_token_id" validate:"required"`
	APITokenSecret string          `json:"api_token_secret" validate:"required"`
	SSHPort        int             `json:"ssh_port,omitempty"`
	SSHUser        string          `json:"ssh_user,omitempty"`
	SSHPrivateKey  string          `json:"ssh_private_key,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

type UpdateNodeRequest struct {
	Name           *string          `json:"name,omitempty"`
	Hostname       *string          `json:"hostname,omitempty"`
	Port           *int             `json:"port,omitempty"`
	APITokenID     *string          `json:"api_token_id,omitempty"`
	APITokenSecret *string          `json:"api_token_secret,omitempty"`
	SSHPort        *int             `json:"ssh_port,omitempty"`
	SSHUser        *string          `json:"ssh_user,omitempty"`
	SSHPrivateKey  *string          `json:"ssh_private_key,omitempty"`
	Metadata       *json.RawMessage `json:"metadata,omitempty"`
}

type NodeResponse struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Type          NodeType        `json:"type"`
	Hostname      string          `json:"hostname"`
	Port          int             `json:"port"`
	SSHPort       int             `json:"ssh_port"`
	SSHUser       string          `json:"ssh_user"`
	IsOnline      bool            `json:"is_online"`
	LastSeen      *time.Time      `json:"last_seen,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	EnvironmentID *uuid.UUID      `json:"environment_id,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func (n *Node) ToResponse() NodeResponse {
	return NodeResponse{
		ID:            n.ID,
		Name:          n.Name,
		Type:          n.Type,
		Hostname:      n.Hostname,
		Port:          n.Port,
		SSHPort:       n.SSHPort,
		SSHUser:       n.SSHUser,
		IsOnline:      n.IsOnline,
		LastSeen:      n.LastSeen,
		Metadata:      n.Metadata,
		EnvironmentID: n.EnvironmentID,
		CreatedAt:     n.CreatedAt,
		UpdatedAt:     n.UpdatedAt,
	}
}

type OnboardNodeRequest struct {
	Name     string   `json:"name" validate:"required"`
	Type     NodeType `json:"type" validate:"required,oneof=pve pbs"`
	Hostname string   `json:"hostname" validate:"required"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password" validate:"required"`
	SSHPort  int      `json:"ssh_port"`
}

// Bulk VM operations

type BulkVMRequest struct {
	VMIDs  []int  `json:"vmids"`
	Action string `json:"action"` // "start", "stop", "shutdown"
}

type BulkVMResult struct {
	VMID    int    `json:"vmid"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	UPID    string `json:"upid,omitempty"`
}

type TestConnectionRequest struct {
	Hostname       string   `json:"hostname" validate:"required"`
	Port           int      `json:"port"`
	Type           NodeType `json:"type" validate:"required"`
	APITokenID     string   `json:"api_token_id" validate:"required"`
	APITokenSecret string   `json:"api_token_secret" validate:"required"`
}

type TestConnectionResponse struct {
	Success bool   `json:"success"`
	Version string `json:"version,omitempty"`
	Node    string `json:"node,omitempty"`
	Error   string `json:"error,omitempty"`
}
