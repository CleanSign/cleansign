package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CleanSign/cleansign/server/internal/database/sqlc"
	"github.com/CleanSign/cleansign/server/internal/domain/user"
)

// UserRepo implements the user port for service/user and service/auth.
type UserRepo struct{ q *sqlc.Queries }

func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{q: queries(pool)} }

func toUser(r sqlc.User) user.User {
	return user.User{ID: r.ID, Email: r.Email, Name: r.Name, EmailVerified: r.EmailVerified, CreatedAt: r.CreatedAt.UTC()}
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (user.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return user.User{}, notFound(err)
	}
	return toUser(row), nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (user.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return user.User{}, notFound(err)
	}
	return toUser(row), nil
}

func (r *UserRepo) Create(ctx context.Context, u user.User) (user.User, error) {
	row, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{ID: u.ID, Email: u.Email, Name: u.Name, EmailVerified: u.EmailVerified})
	if err != nil {
		return user.User{}, err
	}
	return toUser(row), nil
}

func (r *UserRepo) CreateConsent(ctx context.Context, c user.Consent) (user.Consent, error) {
	row, err := r.q.CreateUserConsent(ctx, sqlc.CreateUserConsentParams{
		ID: c.ID, UserID: c.UserID, Terms: c.Terms, Privacy: c.Privacy, Marketing: c.Marketing,
	})
	if err != nil {
		return user.Consent{}, err
	}
	return user.Consent{ID: row.ID, UserID: row.UserID, Terms: row.Terms, Privacy: row.Privacy, Marketing: row.Marketing, CreatedAt: row.CreatedAt.UTC()}, nil
}
