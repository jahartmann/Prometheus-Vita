package node

import (
	"strings"
	"testing"

	"github.com/antigravity/prometheus/internal/model"
)

// A node added by API token only (no onboarding) has no SSH key, so SSH-backed
// features must fail with a clear, actionable message rather than a cryptic
// "no authentication method provided" from deep in the SSH client.
func TestBuildSSHConfigRequiresCredentials(t *testing.T) {
	s := &Service{}
	_, err := s.buildSSHConfig(&model.Node{Hostname: "pve1", SSHPort: 22, SSHUser: "root"})
	if err == nil {
		t.Fatalf("expected an error for a node without SSH credentials")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "ssh") {
		t.Fatalf("error should mention SSH credentials, got: %v", err)
	}
}

func TestClassifySSHError(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ssh host key mismatch for pve1: expected ...", "Host-Key"},
		{"ssh handshake 1.2.3.4:22: ssh: unable to authenticate, no supported methods remain", "Authentifizierung"},
		{"dial tcp 1.2.3.4:22: i/o timeout", "erreichbar"},
	}
	for _, c := range cases {
		got := classifySSHError(stringError(c.in))
		if !strings.Contains(got, c.want) {
			t.Fatalf("classifySSHError(%q) = %q, want substring %q", c.in, got, c.want)
		}
	}
}

type stringError string

func (e stringError) Error() string { return string(e) }
