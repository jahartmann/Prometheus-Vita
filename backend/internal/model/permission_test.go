package model

import "testing"

func TestRoleHasPermission(t *testing.T) {
	tests := []struct {
		name       string
		role       UserRole
		permission Permission
		want       bool
	}{
		{"admin can manage settings", RoleAdmin, PermissionSettingsManage, true},
		{"operator can start and stop VMs", RoleOperator, PermissionVMPower, true},
		{"operator can delete backups", RoleOperator, PermissionBackupsDelete, true},
		{"operator can manage security operations", RoleOperator, PermissionSecurityManage, true},
		{"operator cannot manage settings", RoleOperator, PermissionSettingsManage, false},
		{"viewer can read nodes", RoleViewer, PermissionNodesRead, true},
		{"viewer cannot restore backups", RoleViewer, PermissionBackupsRestore, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RoleHasPermission(tt.role, tt.permission); got != tt.want {
				t.Fatalf("RoleHasPermission(%q, %q) = %v, want %v", tt.role, tt.permission, got, tt.want)
			}
		})
	}
}

func TestPermissionSetAllowsWildcardAndExactMatches(t *testing.T) {
	set := NewPermissionSet([]Permission{PermissionNodesRead, PermissionAll})

	if !set.Allows(PermissionNodesRead) {
		t.Fatal("expected exact permission to be allowed")
	}
	if !set.Allows(PermissionSettingsManage) {
		t.Fatal("expected wildcard permission to allow settings management")
	}
}

func TestPermissionSetRejectsUnknownPermission(t *testing.T) {
	set := NewPermissionSet([]Permission{PermissionNodesRead})

	if set.Allows(PermissionSettingsManage) {
		t.Fatal("expected missing permission to be rejected")
	}
}

func TestBuildPermissionCatalogContainsAllRoles(t *testing.T) {
	catalog := BuildPermissionCatalog()

	if len(catalog.Permissions) == 0 {
		t.Fatal("expected permission definitions")
	}
	if len(catalog.Roles) != 3 {
		t.Fatalf("roles length = %d, want 3", len(catalog.Roles))
	}
}

func TestNormalizeRolePermissionsRejectsUnknownPermission(t *testing.T) {
	_, err := NormalizeRolePermissions(RoleOperator, []Permission{PermissionNodesRead, Permission("nope.nope")})
	if err == nil {
		t.Fatal("expected unknown permission to be rejected")
	}
}

func TestNormalizeRolePermissionsRequiresAdminWildcard(t *testing.T) {
	_, err := NormalizeRolePermissions(RoleAdmin, []Permission{PermissionNodesRead})
	if err == nil {
		t.Fatal("expected admin without wildcard to be rejected")
	}
}

func TestNormalizeRolePermissionsDeduplicates(t *testing.T) {
	permissions, err := NormalizeRolePermissions(RoleOperator, []Permission{
		PermissionNodesRead,
		PermissionNodesRead,
		PermissionVMsRead,
	})
	if err != nil {
		t.Fatalf("NormalizeRolePermissions returned error: %v", err)
	}
	if len(permissions) != 2 {
		t.Fatalf("permissions length = %d, want 2", len(permissions))
	}
}
