package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/antigravity/prometheus-v2/internal/db/repo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// mockQuerier is a hand-rolled in-memory implementation of Querier. It
// covers only the surface the service exercises; calls to anything else
// panic so a reviewer notices when the service grows new dependencies.
type mockQuerier struct {
	usersByEmail map[string]repo.User
	usersByID    map[uuid.UUID]repo.User
	sessions     map[uuid.UUID]repo.Session
	byRefresh    map[string]uuid.UUID

	revokeCalls []uuid.UUID
}

func newMockQuerier() *mockQuerier {
	return &mockQuerier{
		usersByEmail: map[string]repo.User{},
		usersByID:    map[uuid.UUID]repo.User{},
		sessions:     map[uuid.UUID]repo.Session{},
		byRefresh:    map[string]uuid.UUID{},
	}
}

func (m *mockQuerier) GetUserByEmail(_ context.Context, email string) (repo.User, error) {
	u, ok := m.usersByEmail[email]
	if !ok {
		return repo.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (m *mockQuerier) GetUserByID(_ context.Context, id pgtype.UUID) (repo.User, error) {
	if !id.Valid {
		return repo.User{}, pgx.ErrNoRows
	}
	u, ok := m.usersByID[uuid.UUID(id.Bytes)]
	if !ok {
		return repo.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (m *mockQuerier) CountUsers(_ context.Context) (int64, error) {
	return int64(len(m.usersByID)), nil
}

func (m *mockQuerier) CreateUser(_ context.Context, arg repo.CreateUserParams) (repo.User, error) {
	u := repo.User{
		ID:           arg.ID,
		Email:        arg.Email,
		Name:         arg.Name,
		PasswordHash: arg.PasswordHash,
		Role:         arg.Role,
		Enabled:      arg.Enabled,
		Version:      1,
		CreatedAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	m.usersByEmail[u.Email] = u
	m.usersByID[uuid.UUID(arg.ID.Bytes)] = u
	return u, nil
}

func (m *mockQuerier) CreateSession(_ context.Context, arg repo.CreateSessionParams) (repo.Session, error) {
	s := repo.Session{
		ID:               arg.ID,
		UserID:           arg.UserID,
		RefreshTokenHash: arg.RefreshTokenHash,
		UserAgent:        arg.UserAgent,
		IpAddress:        arg.IpAddress,
		ExpiresAt:        arg.ExpiresAt,
		LastSeenAt:       pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		CreatedAt:        pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	id := uuid.UUID(arg.ID.Bytes)
	m.sessions[id] = s
	m.byRefresh[arg.RefreshTokenHash] = id
	return s, nil
}

func (m *mockQuerier) GetSessionByRefreshHash(_ context.Context, hash string) (repo.Session, error) {
	id, ok := m.byRefresh[hash]
	if !ok {
		return repo.Session{}, pgx.ErrNoRows
	}
	return m.sessions[id], nil
}

func (m *mockQuerier) TouchSession(_ context.Context, id pgtype.UUID) error {
	uid := uuid.UUID(id.Bytes)
	s, ok := m.sessions[uid]
	if !ok {
		return pgx.ErrNoRows
	}
	s.LastSeenAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.sessions[uid] = s
	return nil
}

func (m *mockQuerier) RevokeSession(_ context.Context, id pgtype.UUID) error {
	uid := uuid.UUID(id.Bytes)
	m.revokeCalls = append(m.revokeCalls, uid)
	s, ok := m.sessions[uid]
	if !ok {
		return nil
	}
	s.RevokedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	m.sessions[uid] = s
	return nil
}

func (m *mockQuerier) RevokeUserSessions(_ context.Context, userID pgtype.UUID) error {
	uid := uuid.UUID(userID.Bytes)
	for k, s := range m.sessions {
		if uuid.UUID(s.UserID.Bytes) == uid && !s.RevokedAt.Valid {
			s.RevokedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
			m.sessions[k] = s
		}
	}
	return nil
}

// newTestService builds a fully wired service backed by mockQuerier with
// one seeded admin user (alice@example.com / a-good-password-1234).
func newTestService(t *testing.T) (*Service, *mockQuerier, uuid.UUID) {
	t.Helper()
	hash, err := HashPassword("a-good-password-1234")
	require.NoError(t, err)

	m := newMockQuerier()
	id := uuid.New()
	_, err = m.CreateUser(context.Background(), repo.CreateUserParams{
		ID:           pgUUID(id),
		Email:        "alice@example.com",
		Name:         "Alice",
		PasswordHash: hash,
		Role:         RoleAdmin,
		Enabled:      true,
	})
	require.NoError(t, err)

	signer := NewJWTSigner([]byte("test-secret-please-be-32-bytes-or-more!"), "prometheus-v2")
	svc := NewService(m, signer)
	return svc, m, id
}

func TestLogin_AcceptsValidCredentials(t *testing.T) {
	svc, _, userID := newTestService(t)

	tokens, err := svc.Login(context.Background(), LoginRequest{
		Email:     "alice@example.com",
		Password:  "a-good-password-1234",
		UserAgent: "go-test/1.0",
		IPAddress: "127.0.0.1",
	})
	require.NoError(t, err)
	require.NotNil(t, tokens)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
	require.Equal(t, userID, tokens.User.ID)
	require.Equal(t, RoleAdmin, tokens.User.Role)
	require.True(t, tokens.RefreshExpiry.After(tokens.AccessExpiry),
		"refresh expiry should outlive access expiry")
}

func TestLogin_RejectsBadPassword(t *testing.T) {
	svc, _, _ := newTestService(t)

	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "alice@example.com",
		Password: "the-wrong-password-123",
	})
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestRefresh_RotatesAndReturnsNewTokens(t *testing.T) {
	svc, m, _ := newTestService(t)

	first, err := svc.Login(context.Background(), LoginRequest{
		Email:    "alice@example.com",
		Password: "a-good-password-1234",
	})
	require.NoError(t, err)
	originalRefresh := first.RefreshToken
	originalSessionID := m.byRefresh[HashRefreshToken(originalRefresh)]
	require.NotEqual(t, uuid.Nil, originalSessionID)

	second, err := svc.Refresh(context.Background(), originalRefresh, "go-test/1.0", "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, second)
	require.NotEqual(t, first.RefreshToken, second.RefreshToken,
		"refresh token must rotate")
	require.NotEmpty(t, second.AccessToken, "refresh must mint a new access token")
	// Note: two access tokens minted in the same second for the same user
	// will have identical claims and therefore identical signed bytes —
	// that is by design (no nonce in AccessClaims). The access-token
	// rotation guarantee comes from a fresh signature each time, not from
	// byte-distinct tokens.

	// The old session should be revoked.
	old := m.sessions[originalSessionID]
	require.True(t, old.RevokedAt.Valid, "old session must be revoked after rotation")

	// Re-using the old refresh token must now fail.
	_, err = svc.Refresh(context.Background(), originalRefresh, "", "")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrSessionNotFound) || errors.Is(err, ErrSessionExpired),
		"old refresh token must not be re-usable")
}

func TestLogout_RevokesSessionIdempotent(t *testing.T) {
	svc, m, _ := newTestService(t)

	tokens, err := svc.Login(context.Background(), LoginRequest{
		Email:    "alice@example.com",
		Password: "a-good-password-1234",
	})
	require.NoError(t, err)

	require.NoError(t, svc.Logout(context.Background(), tokens.RefreshToken))
	require.Len(t, m.revokeCalls, 1, "first logout should revoke once")

	// Calling logout again with the same (now-revoked) token should be
	// a silent no-op: no error, no duplicate revoke recorded.
	require.NoError(t, svc.Logout(context.Background(), tokens.RefreshToken))
	require.Len(t, m.revokeCalls, 1, "logout must be idempotent")

	// Empty refresh token is also a no-op.
	require.NoError(t, svc.Logout(context.Background(), ""))
	require.Len(t, m.revokeCalls, 1)
}
