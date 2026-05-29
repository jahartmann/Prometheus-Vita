package node

import (
	"encoding/json"
	"testing"

	"github.com/antigravity/prometheus/internal/model"
)

// These tests lock in the PVE-node resolution behavior that the topology, VM
// health/dependency/snapshot-policy and rightsizing services now depend on
// (they previously used pveNodes[0], which picked the wrong host on multi-node
// clusters).

func TestResolvePVENodePrefersMetadata(t *testing.T) {
	n := &model.Node{Name: "prometheus-label", Metadata: json.RawMessage(`{"pve_node":"pve02"}`)}
	if got := ResolvePVENode(n, []string{"pve01", "pve02", "pve03"}); got != "pve02" {
		t.Fatalf("got %q, want pve02", got)
	}
}

func TestResolvePVENodeMatchesByNameCaseInsensitive(t *testing.T) {
	n := &model.Node{Name: "PVE03"}
	if got := ResolvePVENode(n, []string{"pve01", "pve02", "pve03"}); got != "pve03" {
		t.Fatalf("got %q, want pve03", got)
	}
}

func TestResolvePVENodeFallsBackToFirstWhenNoMatch(t *testing.T) {
	n := &model.Node{Name: "unrelated", Metadata: json.RawMessage(`{"pve_node":"ghost"}`)}
	if got := ResolvePVENode(n, []string{"pve01", "pve02"}); got != "pve01" {
		t.Fatalf("got %q, want pve01 (fallback)", got)
	}
}
