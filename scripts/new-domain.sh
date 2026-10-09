#!/usr/bin/env bash
# 새 도메인의 Go 뼈대를 만든다: server/internal/domain/<name>/ 과 server/internal/service/<name>/
# 사용: make domain NAME=article   (소문자와 숫자, 여러 단어는 consultantavailability 처럼 붙여 쓴다)
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
// Package $NAME 은 $NAME 의 순수한 데이터와 규칙을 담는다.
// 표준 라이브러리만 import한다. DB, HTTP, 외부 서비스를 알지 못한다.
package $NAME
GO

cat > "$SERVICE_DIR/ports.go" <<GO
package $NAME

// 이 서비스가 필요로 하는 바깥 세계(DB, 외부 서비스)를 인터페이스(포트)로 정의한다.
// 구현은 internal/database 나 internal/infra 에 둔다.
GO

cat > "$SERVICE_DIR/service.go" <<GO
// Package $NAME 은 $NAME 의 흐름을 조립한다: 포트를 불러 데이터를 읽고, domain 규칙을 적용하고, 저장한다.
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
