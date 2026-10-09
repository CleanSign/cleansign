package auth

import (
	"context"
	"time"

	"github.com/CleanSign/cleansign/server/internal/domain/auth"
	"github.com/CleanSign/cleansign/server/internal/domain/user"
)

// 저장소 포트는 대상이 없으면 errs.ErrNotFound를 돌려준다.

type CodeRepository interface {
	Get(ctx context.Context, email string) (auth.EmailCode, error)
	// Upsert는 이메일당 한 행을 덮어쓴다(시도 횟수는 0으로 돌아간다).
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

// Mailer는 인증코드 메일을 보낸다.
type Mailer interface {
	SendCode(ctx context.Context, to, code string, ttl time.Duration) error
}

type TokenKind string

const (
	KindAccess  TokenKind = "access"
	KindRefresh TokenKind = "refresh"
)

// Claims는 토큰에서 읽은 값이다.
type Claims struct {
	UserID    string
	SessionID string
	Kind      TokenKind
}

// TokenIssuer는 토큰을 만들고 검증한다.
type TokenIssuer interface {
	Access(userID, sessionID string, now time.Time) (token string, expiresAt time.Time, err error)
	Refresh(userID, sessionID string, expiresAt time.Time) (string, error)
	// Parse는 서명과 만료를 검증한다. ignoreExpiry가 true면 만료만 무시한다(로그아웃용).
	Parse(token string, ignoreExpiry bool) (Claims, error)
}
