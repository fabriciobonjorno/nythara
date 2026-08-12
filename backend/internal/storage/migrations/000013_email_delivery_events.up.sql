CREATE TABLE email_delivery_events (
    provider_event_id text PRIMARY KEY,
    provider_message_id text NOT NULL,
    event_type text NOT NULL,
    event_created_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now(),
    CHECK (length(provider_event_id) BETWEEN 1 AND 200),
    CHECK (length(provider_message_id) BETWEEN 1 AND 200),
    CHECK (length(event_type) BETWEEN 1 AND 100)
);

CREATE INDEX email_delivery_events_message_idx
    ON email_delivery_events(provider_message_id, event_created_at DESC);
CREATE INDEX email_delivery_events_type_idx
    ON email_delivery_events(event_type, event_created_at DESC);
