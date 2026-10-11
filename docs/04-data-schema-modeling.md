# 4. Data Schema Modeling

[`03-high-level-design.md`](./03-high-level-design.md)의 PostgreSQL에 실제로 들어갈 테이블 구조입니다. 주석 없는 필드는 `NOT NULL`, `nullable`로 표기된 것만 NULL 허용입니다.

> **구현 현황 (2026-10)**: 구현된 테이블은 `users`, `sessions`, `email_codes`, `user_consent`, `contract`, `contract_danger_clause`입니다. **`notification_settings`, 구독, 소식, 상담·예약 테이블은 아직 없습니다**(각 절에 표시). 실제 스키마의 기준은 **`server/internal/database/migrations/`** 이고, 이 문서는 설계 의도를 설명합니다. 마이그레이션은 server가 시작할 때 적용되며(goose) 쿼리 코드는 sqlc로 생성합니다. 시각은 모두 `timestamptz`입니다.

인터랙티브 ERD (설계 당시 기준이라 인증 테이블 등이 현재와 다릅니다): https://claude.ai/artifact/BNsxk821QQ7qSEr93Lbum6

---

## 4-0. 인증 — `users` · `sessions` · `email_codes` (구현됨)

비밀번호가 없는 이메일 인증코드 로그인을 위한 테이블입니다. 이전 설계는 Better Auth가 만드는 `user`·`session`·`account`·`verification`을 전제했지만, 인증을 server가 직접 구현하면서 아래처럼 바뀌었습니다(`account`, `verification`은 없음).

```mermaid
erDiagram
    USERS ||--o{ SESSIONS : "로그인 세션"
    USERS {
        text id PK
        text email UK "소문자로 정규화"
        text name "default: '' (가입 시 이름을 받지 않음)"
        boolean email_verified "default: true (인증코드로 확인한 뒤 생성)"
        timestamptz created_at "default: now()"
        timestamptz updated_at "default: now()"
    }
    SESSIONS {
        text id PK "refresh 토큰의 sid가 가리킴"
        text user_id FK "idx, on delete cascade"
        boolean remember_device "default: false"
        timestamptz expires_at "idx"
        timestamptz created_at "default: now()"
    }
    EMAIL_CODES {
        text email PK "이메일당 한 행, 다시 보내면 덮어씀"
        text code_hash "HMAC 해시. 원문 코드는 저장하지 않음"
        integer attempts "default: 0, 틀린 횟수"
        timestamptz sent_at "재요청 쿨다운 60초 기준"
        timestamptz expires_at "발송 후 5분"
    }
```

- `sessions` 행을 지우면 세션이 폐기된다(로그아웃, 전체 폐기). access 토큰도 이 행을 확인하므로 폐기 즉시 못 쓴다.
- 만료된 `email_codes`와 `sessions`는 1분마다 도는 청소 작업이 지운다.
- 사용자 ID는 `text`(UUID 문자열)다.

## 4-1. User 확장 — `user_consent` (구현됨) · `notification_settings` · `subscription_plan` · `user_subscription` (미구현)

```mermaid
erDiagram
    USERS ||--o{ USER_CONSENT : "동의 이력"
    USERS ||--|| NOTIFICATION_SETTINGS : "알림 설정 (미구현)"
    USERS ||--o{ USER_SUBSCRIPTION : "구독 이력 (미구현)"
    SUBSCRIPTION_PLAN ||--o{ USER_SUBSCRIPTION : "가입"

    USERS {
        text id PK "4-0 참고"
    }
    USER_CONSENT {
        text id PK
        text user_id FK "idx"
        boolean terms
        boolean privacy
        boolean marketing "default: false"
        timestamptz created_at "default: now()"
    }
    NOTIFICATION_SETTINGS {
        text user_id "PK, FK"
        boolean analysis "default: true"
        boolean booking "default: true"
        boolean marketing "default: false"
        boolean updates "default: true"
        timestamptz updated_at "default: now()"
    }
    SUBSCRIPTION_PLAN {
        text id PK
        text name
        text description "nullable"
        integer monthly_price
        integer yearly_price
        jsonb features "default: []"
        integer sort_order "default: 0"
        boolean is_active "default: true"
    }
    USER_SUBSCRIPTION {
        text id PK
        text user_id FK "idx"
        text plan_id FK
        enum billing_cycle "monthly | yearly"
        enum status "active | canceled | expired, default: active"
        timestamptz started_at "default: now()"
        timestamptz current_period_end
        timestamptz canceled_at "nullable"
    }
```

- `user_consent`는 append-only 이력 (재동의마다 새 행)
- `notification_settings`는 유저당 현재 상태 1행, `user_id`가 PK 겸 FK (1:1)
- `subscription_plan`은 유저와 무관한 공개 카탈로그
- 인덱스: `user_subscription(user_id, status)` — "현재 구독" 조회용

## 4-2. Contract — `contract` · `contract_danger_clause` (구현됨)

```mermaid
erDiagram
    USERS ||--o{ CONTRACT : "업로드"
    CONTRACT ||--o{ CONTRACT_DANGER_CLAUSE : "포함"

    USERS {
        text id PK
    }
    CONTRACT {
        text id PK
        text user_id FK "idx(user_id, created_at desc), on delete cascade"
        text file_name
        text file_key "nullable, 오브젝트 스토리지 연결 전에는 원본을 저장하지 않아 null"
        text mime_type
        enum status "pending | completed | failed, default: pending"
        integer loss_amount "default: 0, 하위 clause 합 캐시"
        integer risk_score "nullable, 0~100"
        enum grade "nullable, high | medium | low"
        integer loss_min "nullable, 손실 범위 하한"
        integer loss_max "nullable, 손실 범위 상한"
        integer deposit_amount "nullable, 추출한 보증금"
        text model_version "nullable, 모델 / 정책 버전"
        jsonb warnings "nullable, 사용자에게 보여 줄 경고 목록"
        jsonb risk_flow "nullable, 아직 생성하지 않음"
        jsonb progress "nullable, pending일 때만 { stage, done, total }"
        text error_message "nullable, failed일 때 사용자에게 보여 줄 메시지"
        timestamptz created_at "default: now()"
    }
    CONTRACT_DANGER_CLAUSE {
        text id PK
        text contract_id FK "idx, on delete cascade"
        text original
        text clause_type "nullable, 선순위근저당 | 보증금반환제한 | 원상복구과도부담 | 묵시적갱신배제 | 기타불이익"
        real confidence "nullable"
        integer l_max "nullable, 손실 상한"
        integer loss_amount "default: 0"
        enum severity "high | medium | low"
        text reason
        text law_reference "nullable, 여러 법령은 쉼표와 공백으로 이어 붙인 문자열"
        text safe_text
        integer order_index "default: 0"
        timestamptz created_at "default: now()"
    }
```

- `contract.status`는 **폴링에 쓰는 실제 상태**다. 업로드하면 `pending`(+ `progress`)으로 만들고, 분석이 끝나면 `completed`(결과 저장) 또는 `failed`(`error_message`)가 된다. 분석은 비동기라 클라이언트는 `GET /contracts/{id}`를 폴링한다(02-2-1 "결정 변경 (2026-10-06)").
- 결과(`completed`)는 계약서 행 갱신과 조항 저장을 **한 트랜잭션**으로 한다. 분석하는 동안 사용자가 계약서를 지웠으면(행이 없으면) 결과를 버린다.
- 오래된 `pending`(15분 초과)은 `failed`로 바뀐다(조회 시점과 1분 주기 청소).
- `law_reference`는 지금 법령 여러 개를 `", "`로 이어 붙인 문자열 한 컬럼이다. 상세 조회 응답에는 이 문자열이 그대로, 조항 상세 응답에는 `", "`로 쪼갠 배열이 나간다.
- `monthly_rent`, `lease_type`, `analysis_ms`, `precedent_ids` 같은 06-6-7의 추가 제안 컬럼은 아직 없다.
- `contract.loss_amount`는 하위 `contract_danger_clause.loss_amount` 합을 분석 시점에 캐시
- 인덱스: `contract(user_id, created_at desc)` — 히스토리 페이지네이션용, `contract_danger_clause(contract_id)`

## 4-3. Article (미구현)

```mermaid
erDiagram
    ARTICLE {
        text id PK
        text title
        text summary
        text body
        text thumbnail_url "nullable"
        text category
        text source_url "nullable"
        boolean is_published "default: false"
        timestamptz published_at "nullable"
        timestamptz created_at "default: now()"
        timestamptz updated_at "default: now()"
    }
```

- `source_url`은 법적 신뢰도를 위한 출처 표기
- `is_published`/`published_at`으로 검수 전 노출 방지 + 노출 정렬 기준 분리
- 인덱스: `article(is_published, published_at desc)`

## 4-4. Consultant 예약 — `consultant` · `consultant_availability` · `booking` (미구현)

```mermaid
erDiagram
    USERS ||--o{ BOOKING : "예약"
    CONSULTANT ||--o{ BOOKING : "대상"
    CONSULTANT ||--o{ CONSULTANT_AVAILABILITY : "제공"
    CONSULTANT_AVAILABILITY |o--o| BOOKING : "예약 성사 시 연결"

    USERS {
        text id PK
    }
    CONSULTANT {
        text id PK
        text name
        enum type "lawyer | realtor"
        text specialty
        text bio "nullable"
        jsonb career "nullable"
        text profile_image_url "nullable"
        numeric rating "default: 0"
        integer review_count "default: 0"
        integer consultation_fee
        integer consultation_minutes "default: 30"
        boolean is_active "default: true"
    }
    CONSULTANT_AVAILABILITY {
        text id PK
        text consultant_id FK "uk"
        date date "uk"
        text start_time "uk"
        boolean is_booked "default: false"
        text booking_id FK "nullable"
    }
    BOOKING {
        text id PK
        text user_id FK "idx"
        text consultant_id FK "idx"
        date date
        text start_time
        text note "nullable"
        integer amount "예약 시점 fee 스냅샷"
        enum status "pending_payment | paid | canceled, default: pending_payment"
        text payment_method "nullable"
        timestamptz paid_at "nullable"
        timestamptz created_at "default: now()"
    }
```

- `booking.amount`는 `consultant.consultation_fee`의 예약 시점 스냅샷 (이후 가격 변경과 무관)
- `booking.status`는 `pending_payment → paid`로만 흐름, `canceled`는 결제 전 이탈
- 유니크: `consultant_availability(consultant_id, date, start_time)` — 슬롯 중복 생성 방지
- 인덱스: `booking(user_id)`, `booking(consultant_id)`

**모델링 제외:** FAQ — DB/API 없이 클라이언트에 하드코딩.

---

## 다음

- 전체 요약: [`05-overview.md`](./05-overview.md)
