package auth

// Role names. Persisted as text in users.role; bundle-of-permissions are
// resolved at login time via PermissionsForRole.
const (
	RoleViewer   = "viewer"
	RoleOperator = "operator"
	RoleAdmin    = "admin"
)

// Permission keys. Domain modules use these strings as middleware arguments.
// Keep the list lean: only permissions V2 actively enforces today. Add new
// ones as their first consumer lands.
const (
	PermAuthMe = "auth:me" // every authenticated user has this implicitly

	PermHostRead  = "host:read"
	PermHostWrite = "host:write"

	PermVMRead         = "vm:read"
	PermVMReadOwned    = "vm:read:owned"
	PermVMLifecycle    = "vm:lifecycle"
	PermVMLifecycleAny = "vm:lifecycle:any"

	PermAuditRead = "audit:read"

	PermAdminUserManage = "admin:user:manage"
	PermAdminRBACManage = "admin:rbac:manage"
	PermAdminSystem     = "admin:system"
)

// PermissionsForRole returns the permission set that a role grants. The
// returned slice is a fresh copy so callers can mutate or extend it.
func PermissionsForRole(role string) []string {
	switch role {
	case RoleViewer:
		return []string{
			PermAuthMe,
			PermHostRead,
			PermVMRead,
			PermVMReadOwned,
		}
	case RoleOperator:
		return []string{
			PermAuthMe,
			PermHostRead,
			PermVMRead,
			PermVMReadOwned,
			PermVMLifecycle,
		}
	case RoleAdmin:
		return []string{
			PermAuthMe,
			PermHostRead, PermHostWrite,
			PermVMRead, PermVMReadOwned,
			PermVMLifecycle, PermVMLifecycleAny,
			PermAuditRead,
			PermAdminUserManage, PermAdminRBACManage, PermAdminSystem,
		}
	default:
		return nil
	}
}

// HasPermission returns true if the granted slice contains the required
// permission. The implicit PermAuthMe is treated as always granted to any
// caller that has any other permission (i.e. authenticated user).
func HasPermission(granted []string, required string) bool {
	if required == PermAuthMe && len(granted) > 0 {
		return true
	}
	for _, p := range granted {
		if p == required {
			return true
		}
	}
	return false
}
