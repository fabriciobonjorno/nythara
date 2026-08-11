-- Identidade pública: username global, case-insensitive e sem espaços.
-- NOT VALID preserva perfis Alpha antigos; PostgreSQL ainda aplica o CHECK
-- a todo INSERT/UPDATE novo.
ALTER TABLE player_profiles
    ADD CONSTRAINT player_profiles_username_format
    CHECK (display_name ~ '^[A-Za-z0-9_-]{2,32}$') NOT VALID;

CREATE UNIQUE INDEX player_profiles_username_ci_unique
    ON player_profiles (lower(display_name));
