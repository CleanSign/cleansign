// Package pgtest는 테스트마다 비어 있는 임시 데이터베이스를 만들어 준다.
// 관리자 접속 주소는 TEST_DATABASE_URL(기본: 로컬 docker compose의 Postgres)이고, 닿지 않으면 테스트를 건너뛴다.
package pgtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CleanSign/cleansign/server/internal/database"
)

const defaultAdminURL = "postgres://postgres:mysecretpassword@localhost:5433/postgres?sslmode=disable"

// New는 마이그레이션이 적용된 임시 DB의 풀을 돌려주고, 테스트가 끝나면 지운다.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		adminURL = defaultAdminURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Skipf("테스트용 Postgres에 연결할 수 없어 건너뜁니다 (TEST_DATABASE_URL 또는 `make db`): %v", err)
	}
	defer func() { _ = admin.Close(ctx) }()

	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	name := "cleansign_test_" + hex.EncodeToString(suffix[:])
	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, name)); err != nil {
		t.Fatalf("임시 DB 생성: %v", err)
	}

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name

	pool, err := database.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		c, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		a, err := pgx.Connect(c, adminURL)
		if err != nil {
			return
		}
		defer func() { _ = a.Close(c) }()
		_, _ = a.Exec(c, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, name))
	})

	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("마이그레이션: %v", err)
	}
	return pool
}
