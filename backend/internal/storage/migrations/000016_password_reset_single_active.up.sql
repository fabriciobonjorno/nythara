CREATE UNIQUE INDEX IF NOT EXISTS password_reset_tokens_one_active_per_user
    ON password_reset_tokens(user_id)
    WHERE used_at IS NULL;
