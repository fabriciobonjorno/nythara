-- Progressão global de conta. O nível nunca é persistido: ele é derivado do
-- XP, que tem teto físico correspondente ao nível 50.
CREATE TABLE player_account_progress (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    xp integer NOT NULL DEFAULT 0 CHECK (xp BETWEEN 0 AND 28420),
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO player_account_progress(user_id,xp)
SELECT id,0 FROM users
ON CONFLICT DO NOTHING;

-- Uma composição antiga com Lendária pode continuar persistida para o dono
-- decidir a substituição, mas não pode prendê-lo na trava de 24h quando o gate
-- de nível entrar em vigor. A API impedirá esse deck de entrar em partida.
UPDATE decks d SET locked_until=NULL
WHERE EXISTS (
    SELECT 1 FROM deck_cards dc
    JOIN card_definitions cd
      ON cd.id=dc.card_id AND cd.ruleset_version=dc.ruleset_version
    WHERE dc.deck_id=d.id AND cd.rarity='Lendária'
);
