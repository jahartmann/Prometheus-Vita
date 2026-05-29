package handler

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseNodeIDParamsAcceptsSingularAndPlural(t *testing.T) {
	a := uuid.New()
	b := uuid.New()

	// Singular form (what the export dialog actually sends).
	got, err := parseNodeIDParams("", a.String())
	if err != nil {
		t.Fatalf("singular: unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != a {
		t.Fatalf("singular: got %v, want [%s]", got, a)
	}

	// Plural, comma-joined.
	got, err = parseNodeIDParams(a.String()+","+b.String(), "")
	if err != nil {
		t.Fatalf("plural: unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("plural: got %d ids, want 2", len(got))
	}

	// Both supplied → union without duplicates.
	got, err = parseNodeIDParams(a.String(), a.String())
	if err != nil {
		t.Fatalf("both: unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("both: got %d ids, want 1 (deduped)", len(got))
	}
}

func TestParseNodeIDParamsRejectsInvalid(t *testing.T) {
	if _, err := parseNodeIDParams("not-a-uuid", ""); err == nil {
		t.Fatalf("expected error for invalid uuid, got nil")
	}
}

// Regression for the invalid Redis stream end-ID "<ms>-+" that made XRANGE
// return nothing, so exports were always empty.
func TestRedisStreamBoundsAreValidIDs(t *testing.T) {
	from := time.UnixMilli(1000)
	to := time.UnixMilli(2000)
	minID, maxID := redisStreamBounds(from, to)

	if minID != "1000-0" {
		t.Fatalf("minID = %q, want %q", minID, "1000-0")
	}
	// The end bound must be a valid Redis ID. "<ms>-+" is NOT valid; either a
	// bare "<ms>" (auto-fills max seq) or "<ms>-<seq>" is.
	if strings.Contains(maxID, "+") {
		t.Fatalf("maxID = %q contains '+', which Redis rejects as an invalid stream ID", maxID)
	}
	if maxID != "2000" {
		t.Fatalf("maxID = %q, want %q", maxID, "2000")
	}
}
