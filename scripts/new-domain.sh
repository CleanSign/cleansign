#!/usr/bin/env bash
# Create a Go skeleton for a new domain: server/internal/domain/<name>/ and server/internal/service/<name>/
# Usage: make domain NAME=article   (lowercase letters and digits; join multiple words together, like consultantavailability)
set -euo pipefail

NAME="${1:-}"
ROOT="server/internal"
MODULE="github.com/CleanSign/cleansign/server/internal"

if [ -z "$NAME" ]; then
  echo "사용법: make domain NAME=article"; exit 1
fi
if ! [[ "$NAME" =~ ^[a-z][a-z0-9]*$ ]]; then
  echo "NAME은 소문자/숫자만 가능해요 (Go 패키지 이름 규칙, 예: article, consultantavailability)"; exit 1
fi

DOMAIN_DIR="$ROOT/domain/$NAME"
SERVICE_DIR="$ROOT/service/$NAME"
if [ -e "$DOMAIN_DIR" ] || [ -e "$SERVICE_DIR" ]; then
  echo "이미 있어요: $DOMAIN_DIR 또는 $SERVICE_DIR (덮어쓰지 않아요)"; exit 1
fi
mkdir -p "$DOMAIN_DIR" "$SERVICE_DIR"

cat > "$DOMAIN_DIR/$NAME.go" <<GO
// Package $NAME holds the pure data and rules of $NAME.
// It imports only the standard library and knows nothing about the DB, HTTP, or external services.
package $NAME
GO

cat > "$SERVICE_DIR/ports.go" <<GO
package $NAME

import "context"

//go:generate go tool mockgen -typed -source=ports.go -destination=ports_mock_test.go -package=$NAME -self_package=$MODULE/service/$NAME

// Repository is a placeholder port. Describe what this service needs from the outside world (DB, external services)
// as interfaces here. Implement them in internal/database or internal/infra.
type Repository interface {
	Ping(ctx context.Context) error
}
GO

cat > "$SERVICE_DIR/service.go" <<GO
// Package $NAME composes the $NAME flow: call ports to read data, apply domain rules, and save.
package $NAME

import "context"

// The interface is generated from the struct below (make gen). The package name is temporary on purpose:
// if it equals the domain package name, the generator drops the "$NAME." prefix from types.
//go:generate go tool ifacemaker -f service.go -s service -i Service -p ${NAME}gen -o service_interface.go -y "Service is the public API of the $NAME flow. Handlers depend on this interface; tests replace it with a mock."
//go:generate perl -pi -e "s/^package ${NAME}gen/package $NAME/" service_interface.go
type service struct {
	repo Repository
}

func NewService(repo Repository) Service { return &service{repo: repo} }

// Example is a placeholder. Delete it and write the real methods.
func (s *service) Example(ctx context.Context) error { return s.repo.Ping(ctx) }
GO

(cd server && go generate "./internal/service/$NAME/")

gofmt -l "$DOMAIN_DIR" "$SERVICE_DIR" >/dev/null

cat <<MSG
✅ 만들었어요
   $DOMAIN_DIR/$NAME.go      순수 데이터와 규칙
   $SERVICE_DIR/{service,ports}.go   흐름과 포트

다음에 할 일 (필요한 것만):
   1. make migration NAME=create_$NAME    테이블 추가 (server/internal/database/migrations)
   2. server/internal/database/queries/$NAME.sql 에 쿼리를 쓰고 make gen
   3. server/internal/database/$NAME.go    포트 구현 (리포지토리)
   4. server/api/openapi.yaml 에 엔드포인트를 추가하고 make gen
   5. server/internal/server/$NAME.go      handler 구현, internal/app/app.go 에서 조립
   (import 규칙은 CI의 golangci-lint(depguard)가 검사해요: domain ← service ← server·database·infra)
MSG
