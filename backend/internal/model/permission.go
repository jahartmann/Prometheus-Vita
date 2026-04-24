package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Permission string

type PermissionDefinition struct {
	Key         Permission `json:"key"`
	Label       string     `json:"label"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	Risk        string     `json:"risk"`
}

type RolePermissionSummary struct {
	Role        UserRole     `json:"role"`
	Permissions []Permission `json:"permissions"`
	UpdatedAt   *time.Time   `json:"updated_at,omitempty"`
	UpdatedBy   *uuid.UUID   `json:"updated_by,omitempty"`
}

type PermissionCatalog struct {
	Permissions []PermissionDefinition  `json:"permissions"`
	Roles       []RolePermissionSummary `json:"roles"`
}

type RolePermissionOverride struct {
	Role        UserRole     `json:"role"`
	Permissions []Permission `json:"permissions"`
	UpdatedAt   time.Time    `json:"updated_at"`
	UpdatedBy   *uuid.UUID   `json:"updated_by,omitempty"`
}

type UpdateRolePermissionsRequest struct {
	Permissions []Permission `json:"permissions"`
}

const (
	PermissionAll Permission = "*"

	PermissionNodesRead   Permission = "nodes.read"
	PermissionNodesWrite  Permission = "nodes.write"
	PermissionNodesDelete Permission = "nodes.delete"

	PermissionVMsRead  Permission = "vms.read"
	PermissionVMPower Permission = "vms.power"
	PermissionVMsWrite Permission = "vms.write"

	PermissionBackupsRead    Permission = "backups.read"
	PermissionBackupsCreate  Permission = "backups.create"
	PermissionBackupsRestore Permission = "backups.restore"
	PermissionBackupsDelete  Permission = "backups.delete"

	PermissionLogsRead   Permission = "logs.read"
	PermissionLogsManage Permission = "logs.manage"

	PermissionSecurityRead   Permission = "security.read"
	PermissionSecurityManage Permission = "security.manage"
	PermissionAuditRead      Permission = "audit.read"

	PermissionAgentUse     Permission = "agent.use"
	PermissionAgentExecute Permission = "agent.execute"
	PermissionAgentManage  Permission = "agent.manage"

	PermissionUsersManage     Permission = "users.manage"
	PermissionAPITokensManage Permission = "api_tokens.manage"
	PermissionSettingsManage  Permission = "settings.manage"
)

var allPermissionDefinitions = []PermissionDefinition{
	{Key: PermissionNodesRead, Label: "Nodes lesen", Description: "Inventar, Status, Storage, Netzwerk und Metriken abrufen", Category: "Infrastruktur", Risk: "low"},
	{Key: PermissionNodesWrite, Label: "Nodes verwalten", Description: "Nodes anlegen, bearbeiten, Tags synchronisieren und Aliase setzen", Category: "Infrastruktur", Risk: "medium"},
	{Key: PermissionNodesDelete, Label: "Nodes löschen", Description: "Nodes aus Prometheus entfernen", Category: "Infrastruktur", Risk: "high"},
	{Key: PermissionVMsRead, Label: "VMs lesen", Description: "VMs, Container, Metriken und Gesundheitsdaten abrufen", Category: "Workloads", Risk: "low"},
	{Key: PermissionVMPower, Label: "VM-Power", Description: "VMs starten, stoppen, pausieren, fortsetzen und Bulk-Power-Aktionen ausführen", Category: "Workloads", Risk: "high"},
	{Key: PermissionVMsWrite, Label: "VMs verwalten", Description: "Snapshots, Migrationen, VM-Cockpit, Console und Abhängigkeiten bearbeiten", Category: "Workloads", Risk: "high"},
	{Key: PermissionBackupsRead, Label: "Backups lesen", Description: "Backups, Dateien, Diff und Recovery-Guides abrufen", Category: "Backup & DR", Risk: "low"},
	{Key: PermissionBackupsCreate, Label: "Backups erstellen", Description: "Konfigurations- und VM-Backups sowie Zeitpläne erstellen", Category: "Backup & DR", Risk: "medium"},
	{Key: PermissionBackupsRestore, Label: "Backups wiederherstellen", Description: "Restore-, DR- und Runbook-Aktionen ausführen", Category: "Backup & DR", Risk: "high"},
	{Key: PermissionBackupsDelete, Label: "Backups löschen", Description: "Backups und Backup-Zeitpläne entfernen", Category: "Backup & DR", Risk: "high"},
	{Key: PermissionLogsRead, Label: "Logs lesen", Description: "Logs, Log-Analysen und Anomalien abrufen", Category: "Logs", Risk: "low"},
	{Key: PermissionLogsManage, Label: "Logs verwalten", Description: "Log-Analysen, Bookmarks, Quellen und Reports steuern", Category: "Logs", Risk: "medium"},
	{Key: PermissionSecurityRead, Label: "Security lesen", Description: "Security-Events, Status und Empfehlungen abrufen", Category: "Security", Risk: "low"},
	{Key: PermissionSecurityManage, Label: "Security verwalten", Description: "Security-Modus, Incidents, Baselines und Anomalien bearbeiten", Category: "Security", Risk: "high"},
	{Key: PermissionAuditRead, Label: "Audit lesen", Description: "Audit- und Gateway-Protokolle einsehen", Category: "Security", Risk: "medium"},
	{Key: PermissionAgentUse, Label: "KI nutzen", Description: "KI-Chat und Wissensabfragen verwenden", Category: "KI-Agent", Risk: "low"},
	{Key: PermissionAgentExecute, Label: "KI-Aktionen", Description: "Agent-Tools und Approval-Entscheidungen ausführen", Category: "KI-Agent", Risk: "high"},
	{Key: PermissionAgentManage, Label: "KI verwalten", Description: "Agent-, LLM- und Wissensbasis-Einstellungen bearbeiten", Category: "KI-Agent", Risk: "high"},
	{Key: PermissionUsersManage, Label: "Benutzer verwalten", Description: "Benutzerkonten, Rollen und Aktivierung verwalten", Category: "Einstellungen", Risk: "high"},
	{Key: PermissionAPITokensManage, Label: "API-Tokens verwalten", Description: "API-Tokens erstellen, widerrufen und löschen", Category: "Einstellungen", Risk: "high"},
	{Key: PermissionSettingsManage, Label: "Einstellungen verwalten", Description: "System-, Integrations- und Sicherheitssettings ändern", Category: "Einstellungen", Risk: "high"},
}

var rolePermissions = map[UserRole][]Permission{
	RoleAdmin: {
		PermissionAll,
	},
	RoleOperator: {
		PermissionNodesRead,
		PermissionNodesWrite,
		PermissionVMsRead,
		PermissionVMPower,
		PermissionVMsWrite,
		PermissionBackupsRead,
		PermissionBackupsCreate,
		PermissionBackupsRestore,
		PermissionBackupsDelete,
		PermissionLogsRead,
		PermissionLogsManage,
		PermissionSecurityRead,
		PermissionSecurityManage,
		PermissionAuditRead,
		PermissionAgentUse,
		PermissionAgentExecute,
	},
	RoleViewer: {
		PermissionNodesRead,
		PermissionVMsRead,
		PermissionBackupsRead,
		PermissionLogsRead,
		PermissionSecurityRead,
		PermissionAgentUse,
	},
}

var validPermissions = buildValidPermissionMap()

func buildValidPermissionMap() map[Permission]struct{} {
	permissions := make(map[Permission]struct{}, len(allPermissionDefinitions)+1)
	permissions[PermissionAll] = struct{}{}
	for _, definition := range allPermissionDefinitions {
		permissions[definition.Key] = struct{}{}
	}
	return permissions
}

type PermissionSet struct {
	permissions map[Permission]struct{}
}

func NewPermissionSet(permissions []Permission) PermissionSet {
	set := PermissionSet{permissions: make(map[Permission]struct{}, len(permissions))}
	for _, permission := range permissions {
		if permission == "" {
			continue
		}
		set.permissions[permission] = struct{}{}
	}
	return set
}

func (s PermissionSet) Allows(permission Permission) bool {
	if _, ok := s.permissions[PermissionAll]; ok {
		return true
	}
	_, ok := s.permissions[permission]
	return ok
}

func RolePermissions(role UserRole) []Permission {
	permissions := rolePermissions[role]
	out := make([]Permission, len(permissions))
	copy(out, permissions)
	return out
}

func RoleHasPermission(role UserRole, permission Permission) bool {
	return NewPermissionSet(RolePermissions(role)).Allows(permission)
}

func IsKnownPermission(permission Permission) bool {
	_, ok := validPermissions[permission]
	return ok
}

func NormalizeRolePermissions(role UserRole, permissions []Permission) ([]Permission, error) {
	if !role.IsValid() {
		return nil, fmt.Errorf("invalid role %q", role)
	}

	seen := make(map[Permission]struct{}, len(permissions))
	out := make([]Permission, 0, len(permissions))
	for _, permission := range permissions {
		if permission == "" {
			continue
		}
		if !IsKnownPermission(permission) {
			return nil, fmt.Errorf("unknown permission %q", permission)
		}
		if permission == PermissionAll && role != RoleAdmin {
			return nil, fmt.Errorf("wildcard permission is reserved for admins")
		}
		if _, ok := seen[permission]; ok {
			continue
		}
		seen[permission] = struct{}{}
		out = append(out, permission)
	}

	if role == RoleAdmin && !NewPermissionSet(out).Allows(PermissionAll) {
		return nil, fmt.Errorf("admin role must keep wildcard permission")
	}

	return out, nil
}

func AllPermissionDefinitions() []PermissionDefinition {
	out := make([]PermissionDefinition, len(allPermissionDefinitions))
	copy(out, allPermissionDefinitions)
	return out
}

func BuildPermissionCatalog() PermissionCatalog {
	return BuildPermissionCatalogWithRoles([]RolePermissionSummary{
		{Role: RoleAdmin, Permissions: RolePermissions(RoleAdmin)},
		{Role: RoleOperator, Permissions: RolePermissions(RoleOperator)},
		{Role: RoleViewer, Permissions: RolePermissions(RoleViewer)},
	})
}

func BuildPermissionCatalogWithRoles(roles []RolePermissionSummary) PermissionCatalog {
	return PermissionCatalog{
		Permissions: AllPermissionDefinitions(),
		Roles:       roles,
	}
}
