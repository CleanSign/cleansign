-- +goose Up
-- Per-email send limit: how many codes were sent in the window that started at window_started_at.
ALTER TABLE email_codes
    ADD COLUMN send_count        integer     NOT NULL DEFAULT 0,
    ADD COLUMN window_started_at timestamptz NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE email_codes
    DROP COLUMN window_started_at,
    DROP COLUMN send_count;
