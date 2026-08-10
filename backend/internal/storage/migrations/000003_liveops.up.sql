-- Fase 7 — Admin/LiveOps: snapshots compiláveis de ruleset, drafts de carta,
-- bans emergenciais de ranked e trilha de auditoria administrativa.

CREATE TABLE ruleset_payloads (
    version text PRIMARY KEY REFERENCES rulesets(version),
    cards jsonb NOT NULL,
    champions jsonb NOT NULL,
    effects jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE card_drafts (
    id uuid PRIMARY KEY,
    card_id text NOT NULL,
    base_version text NOT NULL REFERENCES rulesets(version),
    status text NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'validated', 'published', 'discarded')),
    note text NOT NULL DEFAULT '',
    card jsonb NOT NULL,
    effects jsonb NOT NULL,
    last_validation jsonb,
    published_version text REFERENCES rulesets(version),
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX card_drafts_status ON card_drafts (status, updated_at DESC);

-- Ban emergencial: a carta some do competitivo sem apagar histórico nem
-- coleções. Um ban ativo é a linha com lifted_at IS NULL.
CREATE TABLE ranked_card_bans (
    id uuid PRIMARY KEY,
    card_id text NOT NULL,
    reason text NOT NULL,
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    lifted_by uuid REFERENCES users(id),
    lifted_at timestamptz
);

CREATE UNIQUE INDEX one_active_ban_per_card ON ranked_card_bans (card_id)
    WHERE lifted_at IS NULL;

CREATE TABLE admin_audit (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor uuid NOT NULL REFERENCES users(id),
    action text NOT NULL,
    subject text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX admin_audit_recent ON admin_audit (created_at DESC);
