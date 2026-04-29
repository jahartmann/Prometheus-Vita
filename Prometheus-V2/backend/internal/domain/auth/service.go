package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/antigravity/prometheus-v2/internal/db/repo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Token lifetimes. Access tokens are short-lived to bound the blast radius
// of a leaked bearer token; refresh tokens are long-lived but revocable
// via the sessions table.
const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
)

// Service-level errors. Handlers translate these to API responses; tests
// match against them with errors.Is.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserDisabled       = errors.New("user disabled")
	ErrUserNotFound       = errors.New("user not found")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
)

// Querier is the subset of sqlc-generated queries the auth service needs.
// Defining it here (instead of using repo.Querier) keeps the service
// mockable and decoupled from the rest of the schema.
type Querier interface {
	GetUserByEmail(ctx context.Context, email string) (repo.User, error)
	GetUserByID(ctx context.Context, id pgtype.UUID) (repo.User, error)
	CountUsers(ctx context.Context) (int64, error)
	CreateUser(ctx context.Context, arg repo.CreateUserParams) (repo.User, error)
	CreateSession(ctx context.Context, arg repo.CreateSessionParams) (repo.Session, error)
	GetSessionByRefreshHash(ctx context.Context, refreshTokenHash string) (repo.Session, error)
	TouchSession(ctx context.Context, id pgtype.UUID) error
	RevokeSession(ctx context.Context, id pgtype.UUID) error
	RevokeUserSessions(ctx context.Context, userID pgtype.UUID) error
}

// Service implements login, refresh and logout. It is concurrency-safe:
// all state lives in the database; the only in-memory fields are the
// signer and clock, both immutable after construction.
type Service struct {
	queries Querier
	signer  *JWTSigner
	now     func() time.Time
}

// NewService constructs a Service. The clock is fixed to time.Now; tests
// that need determinism should override the now field directly via the
// internal-test build (see service_test.go).
func NewService(q Querier, signer *JWTSigner) *Service {
	return &Service{
		queries: q,
		signer:  signer,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// LoginRequest is the input to Login. UserAgent and IPAddress are
// best-effort metadata sourced from the HTTP layer; both may be empty.
type LoginRequest struct {
	Email     string
	Password  string
	UserAgent string
	IPAddress string
}

// Login verifies credentials and issues an access+refresh token pair.
// Wrong-email and wrong-password failures both return ErrInvalidCredentials
// (no user enumeration). Disabled users return ErrUserDisabled.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthTokens, error) {
	u, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	if !u.Enabled {
		return nil, ErrUserDisabled
	}
	if err := VerifyPassword(u.PasswordHash, req.Password); err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.issueTokens(ctx, u, req.UserAgent, req.IPAddress)
}

// Refresh rotates a refresh token: the supplied raw token is hashed,
// looked up, revoked, and a fresh access+refresh pair is minted bound
// to the same user. If the session was already revoked or has expired,
// the call fails without rotating.
func (s *Service) Refresh(ctx context.Context, rawRefresh, userAgent, ipAddress string) (*AuthTokens, error) {
	hash := HashRefreshToken(rawRefresh)
	sess, err := s.queries.GetSessionByRefreshHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	if sess.RevokedAt.Valid {
		// Re-use of an already-revoked refresh token is suspicious; we do
		// not silently re-issue. Caller must log in again.
		return nil, ErrSessionNotFound
	}
	if !sess.ExpiresAt.Valid || s.now().After(sess.ExpiresAt.Time) {
		return nil, ErrSessionExpired
	}
	// Rotate: revoke the old session before minting the new one.
	if err := s.queries.RevokeSession(ctx, sess.ID); err != nil {
		return nil, fmt.Errorf("revoke old session: %w", err)
	}
	u, err := s.queries.GetUserByID(ctx, sess.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Session pointed at a now-deleted user. Treat as orphaned
			// session: client must log in again. Not a credential failure.
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	if !u.Enabled {
		return nil, ErrUserDisabled
	}
	return s.issueTokens(ctx, u, userAgent, ipAddress)
}

// Logout revokes the session bound to the supplied refresh token. The
// operation is idempotent: an unknown or already-revoked token is a
// no-op success so a user clicking "logout" twice never sees an error.
func (s *Service) Logout(ctx context.Context, rawRefresh string) error {
	if rawRefresh == "" {
		return nil
	}
	hash := HashRefreshToken(rawRefresh)
	sess, err := s.queries.GetSessionByRefreshHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get session: %w", err)
	}
	if sess.RevokedAt.Valid {
		return nil
	}
	if err := s.queries.RevokeSession(ctx, sess.ID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// GetUser implements Reader. Other domains depend on this signature, not
// on the concrete Service.
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	u, err := s.queries.GetUserByID(ctx, pgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	out := userFromRepo(u)
	return &out, nil
}

// issueTokens mints an access token and a refresh-token-bound session in
// a single logical step. All timestamps share the same `now` so access
// and refresh expiries do not drift apart.
func (s *Service) issueTokens(ctx context.Context, u repo.User, userAgent, ipAddress string) (*AuthTokens, error) {
	now := s.now()
	accessExp := now.Add(AccessTokenTTL)
	refreshExp := now.Add(RefreshTokenTTL)

	userID := fromPgUUID(u.ID)
	access, err := s.signer.SignAccessToken(userID, u.Role, PermissionsForRole(u.Role), AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}
	refresh, err := NewRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("new refresh token: %w", err)
	}
	if _, err := s.queries.CreateSession(ctx, repo.CreateSessionParams{
		ID:               pgUUID(uuid.New()),
		UserID:           u.ID,
		RefreshTokenHash: HashRefreshToken(refresh),
		UserAgent:        stringPtr(userAgent),
		IpAddress:        addrPtr(ipAddress),
		ExpiresAt:        pgTimestamp(refreshExp),
	}); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &AuthTokens{
		AccessToken:   access,
		RefreshToken:  refresh,
		AccessExpiry:  accessExp,
		RefreshExpiry: refreshExp,
		User:          userFromRepo(u),
	}, nil
}
