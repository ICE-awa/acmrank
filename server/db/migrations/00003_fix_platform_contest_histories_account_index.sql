-- +goose Up
DROP INDEX IF EXISTS platform_contest_histories_user_idx;

CREATE INDEX platform_contest_histories_account_idx
  ON platform_contest_histories (platform_account_id, participated_at DESC);

-- +goose Down
DROP INDEX IF EXISTS platform_contest_histories_account_idx;

CREATE INDEX platform_contest_histories_user_idx
  ON platform_contest_histories (site_user_id, platform, participated_at DESC);
