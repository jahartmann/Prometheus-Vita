package auth

import (
	"net/netip"
	"time"

	"github.com/antigravity/prometheus-v2/internal/db/repo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// pgUUID converts a uuid.UUID to pgtype.UUID for sqlc parameters.
func pgUUID(u uuid.UUID) pgtype.UUID {
	var out pgtype.UUID
	out.Bytes = u
	out.Valid = true
	return out
}

// fromPgUUID extracts a uuid.UUID from pgtype.UUID. If invalid, returns uuid.Nil.
func fromPgUUID(p pgtype.UUID) uuid.UUID {
	if !p.Valid {
		return uuid.Nil
	}
	return uuid.UUID(p.Bytes)
}

// pgTimestamp wraps a time.Time into a non-null pgtype.Timestamptz.
func pgTimestamp(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// stringPtr returns a pointer to s if non-empty, else nil. For nullable
// text columns where the empty string should map to SQL NULL.
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// addrPtr parses ip and returns a pointer to netip.Addr, or nil if empty/invalid.
// Invalid addresses are silently dropped to nil so a malformed X-Forwarded-For
// header never blocks login.
func addrPtr(ip string) *netip.Addr {
	if ip == "" {
		return nil
	}
	parsed, err := netip.ParseAddr(ip)
	if err != nil {
		return nil
	}
	return &parsed
}

// userFromRepo translates a repo.User into the domain User. Drops the
// password hash so it never escapes the auth package.
func userFromRepo(u repo.User) User {
	return User{
		ID:        fromPgUUID(u.ID),
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		Enabled:   u.Enabled,
		Version:   u.Version,
		CreatedAt: u.CreatedAt.Time,
		UpdatedAt: u.UpdatedAt.Time,
	}
}
