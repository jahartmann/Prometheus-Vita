package model

import (
	"time"

	"github.com/google/uuid"
)

type VMGroup struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	TagFilter   string    `json:"tag_filter"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	MemberCount int       `json:"member_count,omitempty"`
}

type VMGroupMember struct {
	GroupID uuid.UUID `json:"group_id"`
	NodeID  uuid.UUID `json:"node_id"`
	VMID    int       `json:"vmid"`
}

type CreateVMGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TagFilter   string `json:"tag_filter"`
}

type UpdateVMGroupRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	TagFilter   *string `json:"tag_filter,omitempty"`
}

type AddVMGroupMemberRequest struct {
	NodeID string `json:"node_id"`
	VMID   int    `json:"vmid"`
}

type RemoveVMGroupMemberRequest struct {
	NodeID string `json:"node_id"`
	VMID   int    `json:"vmid"`
}
