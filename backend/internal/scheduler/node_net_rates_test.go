package scheduler

import (
	"testing"

	"github.com/antigravity/prometheus/internal/proxmox"
)

func TestLatestNetRates(t *testing.T) {
	// Empty input → zero, no panic.
	if in, out := latestNetRates(nil); in != 0 || out != 0 {
		t.Fatalf("empty: got %d,%d want 0,0", in, out)
	}

	// Uses the most recent point that actually has traffic, skipping a trailing
	// empty/incomplete RRD bucket (which Proxmox returns as null → 0).
	points := []proxmox.RRDDataPoint{
		{Time: 1, NetIn: 100, NetOut: 200},
		{Time: 2, NetIn: 500, NetOut: 600},
		{Time: 3, NetIn: 0, NetOut: 0},
	}
	if in, out := latestNetRates(points); in != 500 || out != 600 {
		t.Fatalf("got %d,%d want 500,600", in, out)
	}

	// Genuinely idle node (all zero) → 0,0.
	if in, out := latestNetRates([]proxmox.RRDDataPoint{{Time: 1}}); in != 0 || out != 0 {
		t.Fatalf("all-zero: got %d,%d want 0,0", in, out)
	}
}
