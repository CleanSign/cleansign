// Package database는 Postgres 연결, 마이그레이션, 리포지토리(포트 구현)를 담는다.
package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/CleanSign/cleansign/server/internal/database/sqlc"
	"github.com/CleanSign/cleansign/server/internal/shared/errs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open은 커넥션 풀을 만들고 연결을 확인한다.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// Migrate는 아직 적용하지 않은 마이그레이션을 모두 적용한다.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, sub)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// notFound는 "행이 없음"을 계층 공용 오류로 바꾼다.
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errs.ErrNotFound
	}
	return err
}

// queries는 풀 위에서 쿼리를 실행한다. 트랜잭션이 필요하면 WithTx를 쓴다.
func queries(pool *pgxpool.Pool) *sqlc.Queries { return sqlc.New(pool) }
