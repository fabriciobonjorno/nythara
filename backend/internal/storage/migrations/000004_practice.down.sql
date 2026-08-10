-- Remove os dados de treino antes de restaurar as restrições antigas.
DELETE FROM matches WHERE mode = 'practice';
ALTER TABLE match_commands DROP CONSTRAINT match_commands_origin_check;
ALTER TABLE match_commands ADD CONSTRAINT match_commands_origin_check
    CHECK (origin IN ('system', 'client', 'timeout'));
ALTER TABLE matches DROP COLUMN mode;
-- A conta do bot permanece por segurança referencial de auditoria.
