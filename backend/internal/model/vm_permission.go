package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	PermVMView           = "vm.view"
	PermVMShell          = "vm.shell"
	PermVMFilesRead      = "vm.files.read"
	PermVMFilesWrite     = "vm.files.write"
	PermVMSystemView     = "vm.system.view"
	PermVMSystemService  = "vm.system.service"
	PermVMSystemKill     = "vm.system.kill"
	PermVMSystemPackages = "vm.system.packages"
	PermVMPower          = "vm.power"
	PermVMSnapshots      = "vm.snapshots"
	PermVMAIProactive    = "vm.ai.proactive"
)

var AllVMPermissions = []string{
	PermVMView, PermVMShell, PermVMFilesRead, PermVMFilesWrite,
	PermVMSystemView, PermVMSystemService, PermVMSystemKill,
	PermVMSystemPackages, PermVMPower, PermVMSnapshots, PermVMAIProactive,
}

type VMPermission struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	NodeID      uuid.UUID `json:"node_id"`
	Permissions []string  `json:"permissions"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
