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

// Defines the outside world this service needs (DB, external services) as interfaces (ports).
// Implementations live in internal/database or internal/infra.
GO

cat > "$SERVICE_DIR/service.go" <<GO
// Package $NAME assembles the flow of $NAME: it calls ports to read data, applies domain rules, and saves.
package $NAME

import _ "$MODULE/domain/$NAME"
GO

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
