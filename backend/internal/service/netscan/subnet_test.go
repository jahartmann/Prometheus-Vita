package netscan

import "testing"

func TestParseSubnetFromIPAddr(t *testing.T) {
	out := `2: eth0    inet 192.168.1.10/24 brd 192.168.1.255 scope global eth0\       valid_lft forever preferred_lft forever`
	got, err := parseSubnetFromIPAddr(out)
	if err != nil || got != "192.168.1.10/24" {
		t.Fatalf("got %q err %v, want 192.168.1.10/24", got, err)
	}

	// A too-large subnet (/8) is rejected so we never launch a massive sweep.
	if _, err := parseSubnetFromIPAddr("2: eth0 inet 10.0.0.1/8 scope global eth0"); err == nil {
		t.Fatalf("expected /8 to be rejected as too large")
	}

	// No usable IPv4 subnet → error.
	if _, err := parseSubnetFromIPAddr("garbage without an inet field"); err == nil {
		t.Fatalf("expected error when no subnet present")
	}
}
