#!/usr/bin/env bash
# 새 도메인 스키마 뼈대를 만든다: src/db/schema/<name>.ts 생성 + index.ts 에 export 추가
# 사용: make domain NAME=article   (여러 단어는 consultant-availability 처럼 소문자와 하이픈)
set -euo pipefail

NAME="${1:-}"
DIR="src/db/schema"

if [ -z "$NAME" ]; then
  echo "사용법: make domain NAME=article"; exit 1
fi
if ! [[ "$NAME" =~ ^[a-z][a-z0-9]*(-[a-z0-9]+)*$ ]]; then
  echo "NAME은 소문자/숫자/하이픈만 가능해요 (예: article, consultant-availability)"; exit 1
fi

FILE="$DIR/$NAME.ts"
if [ -e "$FILE" ]; then
  echo "이미 있어요: $FILE (덮어쓰지 않아요)"; exit 1
fi

# consultant-availability → 변수명 consultantAvailability / 테이블명 consultant_availability
VAR=$(echo "$NAME" | awk -F- '{ s=$1; for (i=2;i<=NF;i++) s = s toupper(substr($i,1,1)) substr($i,2); print s }')
TABLE="${NAME//-/_}"

cat > "$FILE" <<TS
import { pgTable, text, timestamp } from "drizzle-orm/pg-core";

export const $VAR = pgTable("$TABLE", {
  id: text("id")
    .primaryKey()
    .\$defaultFn(() => crypto.randomUUID()),
  createdAt: timestamp("created_at")
    .\$defaultFn(() => new Date())
    .notNull(),
});
TS

EXPORT="export * from \"./$NAME\";"
if ! grep -qxF "$EXPORT" "$DIR/index.ts"; then
  echo "$EXPORT" >> "$DIR/index.ts"
fi

echo "✅ $FILE 생성, $DIR/index.ts 에 등록"
echo "📝 컬럼을 채운 뒤 'make gen' 으로 SQL 생성 + DB 적용하세요"
