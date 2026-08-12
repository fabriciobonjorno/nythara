-- Identidade pública: username global, case-insensitive e sem espaços.
-- NOT VALID preserva perfis Alpha antigos; PostgreSQL ainda aplica o CHECK
-- a todo INSERT/UPDATE novo.
ALTER TABLE player_profiles
    ADD CONSTRAINT player_profiles_username_format
    CHECK (display_name ~ '^[A-Za-z0-9_-]{2,32}$') NOT VALID;

-- Perfis Alpha podiam repetir o mesmo nome com diferenças apenas de caixa.
-- Preserva o perfil mais antigo e renomeia os demais de forma determinística
-- antes de criar a unicidade. O lock fecha a janela entre a correção e o
-- índice para que um cadastro concorrente não reintroduza a colisão.
LOCK TABLE player_profiles IN SHARE ROW EXCLUSIVE MODE;

DO $$
DECLARE
    duplicate RECORD;
    candidate TEXT;
    attempt INTEGER;
BEGIN
    FOR duplicate IN
        SELECT user_id
        FROM (
            SELECT user_id,
                   row_number() OVER (
                       PARTITION BY lower(display_name)
                       ORDER BY created_at NULLS LAST, user_id
                   ) AS position
            FROM player_profiles
        ) ranked
        WHERE position > 1
        ORDER BY user_id
    LOOP
        candidate := NULL;
        FOR attempt IN 0..9999 LOOP
            candidate := 'u_'
                || left(replace(duplicate.user_id::text, '-', ''), 25)
                || '_'
                || lpad(attempt::text, 4, '0');
            IF NOT EXISTS (
                SELECT 1
                FROM player_profiles
                WHERE user_id <> duplicate.user_id
                  AND lower(display_name) = lower(candidate)
            ) THEN
                EXIT;
            END IF;
            candidate := NULL;
        END LOOP;

        IF candidate IS NULL THEN
            RAISE EXCEPTION 'não foi possível gerar username único para o perfil %', duplicate.user_id;
        END IF;

        UPDATE player_profiles
        SET display_name = candidate,
            updated_at = GREATEST(updated_at, now())
        WHERE user_id = duplicate.user_id;
    END LOOP;
END $$;

CREATE UNIQUE INDEX player_profiles_username_ci_unique
    ON player_profiles (lower(display_name));
