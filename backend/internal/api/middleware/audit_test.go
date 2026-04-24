package middleware

import "testing"

func TestClassifyAuditEventSkipsReadRequests(t *testing.T) {
	meta := classifyAuditEvent("GET", "/api/v1/users", 200)
	if meta != nil {
		t.Fatalf("expected no metadata for GET, got %+v", meta)
	}
}

func TestClassifyAuditEventMarksCriticalPermissionChanges(t *testing.T) {
	meta := classifyAuditEvent("PUT", "/api/v1/permissions/roles/operator", 204)
	if meta == nil {
		t.Fatal("expected metadata")
	}
	if !meta.Critical {
		t.Fatalf("expected critical metadata, got %+v", meta)
	}
	if meta.Category != "permissions" || meta.Action != "update" || meta.Risk != "high" || !meta.Success {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
}

func TestClassifyAuditEventMarksVMOperations(t *testing.T) {
	meta := classifyAuditEvent("POST", "/api/v1/nodes/node-1/vms/100/start", 202)
	if meta == nil {
		t.Fatal("expected metadata")
	}
	if meta.Category != "vm" || meta.Action != "create_or_execute" || meta.Risk != "high" {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
}

func TestClassifyAuditEventCapturesFailures(t *testing.T) {
	meta := classifyAuditEvent("DELETE", "/api/v1/gateway/tokens/token-1", 403)
	if meta == nil {
		t.Fatal("expected metadata")
	}
	if meta.Success {
		t.Fatalf("expected unsuccessful metadata, got %+v", meta)
	}
}
