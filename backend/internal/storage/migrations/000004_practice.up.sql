-- Modo treino contra bot: partidas ganham modo, comandos ganham origem 'bot'
-- e nasce a conta reservada do bot (sem login possível).

ALTER TABLE matches ADD COLUMN mode text NOT NULL DEFAULT 'pvp'
    CHECK (mode IN ('pvp', 'practice'));

ALTER TABLE match_commands DROP CONSTRAINT match_commands_origin_check;
ALTER TABLE match_commands ADD CONSTRAINT match_commands_origin_check
    CHECK (origin IN ('system', 'client', 'timeout', 'bot'));

-- Usuário reservado do bot. password_hash '!' jamais valida (formato inválido
-- para o verificador argon2), então a conta não autentica.
INSERT INTO users (id, email, password_hash, role)
VALUES ('00000000-0000-4000-8000-0000000000b0', 'bot@veurubro.internal', '!', 'player')
ON CONFLICT (id) DO NOTHING;
INSERT INTO player_profiles (user_id, display_name)
VALUES ('00000000-0000-4000-8000-0000000000b0', 'Treinador do Véu')
ON CONFLICT (user_id) DO NOTHING;
