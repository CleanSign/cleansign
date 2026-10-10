package auth

//go:generate go tool mockgen -typed -source=ports.go -destination=ports_mock_test.go -package=auth -self_package=github.com/CleanSign/cleansign/server/internal/service/auth

import (
	"context"
	"time"

	"github.com/CleanSign/cleansign/server/internal/domain/auth"
	"github.com/CleanSign/cleansign/server/internal/domain/user"
)

// Store ports return errs.ErrNotFound when the target does not exist.

type CodeRepository interface {
	Get(ctx context.Context, email string) (auth.EmailCode, error)
	// Upsert overwrites the single row per email (the attempt count resets to 0).
	Upsert(ctx context.Context, c auth.EmailCode) error
	IncrementAttempts(ctx context.Context, email string) error
	Delete(ctx context.Context, email string) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

type UserStore interface {
	GetByEmail(ctx context.Context, email string) (user.User, error)
	Create(ctx context.Context, u user.User) (user.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, s auth.Session) error
	Get(ctx context.Context, id string) (auth.Session, error)
	Delete(ctx context.Context, id string) error
	DeleteByUser(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// Mailer sends the verification-code mail.
type Mailer interface {
	SendCode(ctx context.Context, to, code string, ttl time.Duration) error
}

type TokenKind string

const (
	KindAccess  TokenKind = "access"
	KindRefresh TokenKind = "refresh"
)

// Claims are the values read from a token.
type Claims struct {
	UserID    string
	SessionID string
	Kind      TokenKind
}

// TokenIssuer creates and verifies tokens.
type TokenIssuer interface {
	Access(userID, sessionID string, now time.Time) (token string, expiresAt time.Time, err error)
	Refresh(userID, sessionID string, expiresAt time.Time) (string, error)
	// Parse verifies the signature and expiry. If ignoreExpiry is true, only expiry is ignored (for logout).
	Parse(token string, ignoreExpiry bool) (Claims, error)
}
