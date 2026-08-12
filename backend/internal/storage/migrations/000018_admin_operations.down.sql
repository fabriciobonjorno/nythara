DROP TABLE IF EXISTS admin_invites;
DROP TABLE IF EXISTS player_bans;

UPDATE users SET role='admin' WHERE role='owner';
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('player', 'admin'));
