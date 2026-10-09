package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CleanSign/cleansign/server/internal/database/sqlc"
	"github.com/CleanSign/cleansign/server/internal/domain/auth"
)

// CodeRepo는 인증코드 포트를 구현한다.
type CodeRepo struct{ q *sqlc.Queries }

func NewCodeRepo(pool *pgxpool.Pool) *CodeRepo { return &CodeRepo{q: queries(pool)} }

func (r *CodeRepo) Get(ctx context.Context, email string) (auth.EmailCode, error) {
	row, err := r.q.GetEmailCode(ctx, email)
	if err != nil {
		return auth.EmailCode{}, notFound(err)
	}
	return auth.EmailCode{Email: row.Email, CodeHash: row.CodeHash, Attempts: int(row.Attempts), SentAt: row.SentAt.UTC(), ExpiresAt: row.ExpiresAt.UTC()}, nil
}

func (r *CodeRepo) Upsert(ctx context.Context, c auth.EmailCode) error {
	return r.q.UpsertEmailCode(ctx, sqlc.UpsertEmailCodeParams{Email: c.Email, CodeHash: c.CodeHash, SentAt: c.SentAt, ExpiresAt: c.ExpiresAt})
}

func (r *CodeRepo) IncrementAttempts(ctx context.Context, email string) error {
	_, err := r.q.IncrementEmailCodeAttempts(ctx, email)
	return notFound(err)
}

func (r *CodeRepo) Delete(ctx context.Context, email string) error {
	return r.q.DeleteEmailCode(ctx, email)
}

func (r *CodeRepo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return r.q.DeleteExpiredEmailCodes(ctx, now)
}

// SessionRepo는 세션 포트를 구현한다.
type SessionRepo struct{ q *sqlc.Queries }

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo { return &SessionRepo{q: queries(pool)} }

func (r *SessionRepo) Create(ctx context.Context, s auth.Session) error {
	_, err := r.q.CreateSession(ctx, sqlc.CreateSessionParams{ID: s.ID, UserID: s.UserID, RememberDevice: s.RememberDevice, ExpiresAt: s.ExpiresAt})
	return err
}

func (r *SessionRepo) Get(ctx context.Context, id string) (auth.Session, error) {
	row, err := r.q.GetSession(ctx, id)
	if err != nil {
		return auth.Session{}, notFound(err)
	}
	return auth.Session{ID: row.ID, UserID: row.UserID, RememberDevice: row.RememberDevice, ExpiresAt: row.ExpiresAt.UTC(), CreatedAt: row.CreatedAt.UTC()}, nil
}

// Delete는 세션이 없어도 오류가 아니다.
func (r *SessionRepo) Delete(ctx context.Context, id string) error {
	return r.q.DeleteSession(ctx, id)
}

func (r *SessionRepo) DeleteByUser(ctx context.Context, userID string) error {
	return r.q.DeleteSessionsByUser(ctx, userID)
}

func (r *SessionRepo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return r.q.DeleteExpiredSessions(ctx, now)
}
