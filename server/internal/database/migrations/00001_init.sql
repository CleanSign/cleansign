-- +goose Up
CREATE TYPE contract_status AS ENUM ('pending', 'completed', 'failed');
CREATE TYPE clause_severity AS ENUM ('high', 'medium', 'low');

CREATE TABLE users (
    id             text PRIMARY KEY,
    email          text        NOT NULL UNIQUE,
    name           text        NOT NULL DEFAULT '',
    email_verified boolean     NOT NULL DEFAULT true,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

-- 로그인 세션. refresh 토큰(JWT)의 sid가 이 행의 id를 가리킨다. 행을 지우면 세션이 폐기된다.
CREATE TABLE sessions (
    id              text PRIMARY KEY,
    user_id         text        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    remember_device boolean     NOT NULL DEFAULT false,
    expires_at      timestamptz NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- 이메일 인증코드. 이메일당 한 행(다시 보내면 덮어쓴다).
CREATE TABLE email_codes (
    email      text PRIMARY KEY,
    code_hash  text        NOT NULL,
    attempts   integer     NOT NULL DEFAULT 0,
    sent_at    timestamptz NOT NULL,
    expires_at timestamptz NOT NULL
);

CREATE TABLE user_consent (
    id         text PRIMARY KEY,
    user_id    text        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    terms      boolean     NOT NULL,
    privacy    boolean     NOT NULL,
    marketing  boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX user_consent_user_id_idx ON user_consent (user_id);

CREATE TABLE contract (
    id             text PRIMARY KEY,
    user_id        text            NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    file_name      text            NOT NULL,
    -- 오브젝트 스토리지 연결 전까지는 원본을 저장하지 않으므로 null
    file_key       text,
    mime_type      text            NOT NULL,
    status         contract_status NOT NULL DEFAULT 'pending',
    loss_amount    integer         NOT NULL DEFAULT 0,
    risk_score     integer,
    grade          clause_severity,
    loss_min       integer,
    loss_max       integer,
    deposit_amount integer,
    model_version  text,
    warnings       jsonb,
    risk_flow      jsonb,
    -- 분석 중(pending)일 때만 의미가 있다. stage: reading | ocr | splitting | classifying | scoring
    progress       jsonb,
    error_message  text,
    created_at     timestamptz     NOT NULL DEFAULT now()
);
CREATE INDEX contract_user_id_created_at_idx ON contract (user_id, created_at DESC);

CREATE TABLE contract_danger_clause (
    id           text PRIMARY KEY,
    contract_id  text            NOT NULL REFERENCES contract (id) ON DELETE CASCADE,
    original     text            NOT NULL,
    clause_type  text,
    confidence   real,
    l_max        integer,
    loss_amount  integer         NOT NULL DEFAULT 0,
    severity     clause_severity NOT NULL,
    reason       text            NOT NULL,
    law_reference text,
    safe_text    text            NOT NULL,
    order_index  integer         NOT NULL DEFAULT 0,
    created_at   timestamptz     NOT NULL DEFAULT now()
);
CREATE INDEX contract_danger_clause_contract_id_idx ON contract_danger_clause (contract_id);

-- +goose Down
DROP TABLE contract_danger_clause;
DROP TABLE contract;
DROP TABLE user_consent;
DROP TABLE email_codes;
DROP TABLE sessions;
DROP TABLE users;
DROP TYPE clause_severity;
DROP TYPE contract_status;
