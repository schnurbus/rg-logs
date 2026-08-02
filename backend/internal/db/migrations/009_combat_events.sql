-- +migrate Up
CREATE TABLE IF NOT EXISTS abilities (
    upload_id UUID NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
    spell_id INT NOT NULL,
    name TEXT NOT NULL,
    school INT NOT NULL DEFAULT 0,
    PRIMARY KEY (upload_id, spell_id)
);

CREATE INDEX IF NOT EXISTS idx_abilities_upload ON abilities(upload_id);

CREATE TABLE IF NOT EXISTS combat_events (
    id BIGSERIAL PRIMARY KEY,
    fight_id UUID NOT NULL REFERENCES fights(id) ON DELETE CASCADE,
    ts TIMESTAMPTZ NOT NULL,
    offset_ms INT NOT NULL,
    event_type SMALLINT NOT NULL,
    source_actor_id UUID REFERENCES actors(id) ON DELETE SET NULL,
    target_actor_id UUID REFERENCES actors(id) ON DELETE SET NULL,
    spell_id INT NOT NULL DEFAULT 0,
    amount INT NOT NULL DEFAULT 0,
    overkill INT NOT NULL DEFAULT 0,
    overheal INT NOT NULL DEFAULT 0,
    absorbed INT NOT NULL DEFAULT 0,
    resisted INT NOT NULL DEFAULT 0,
    blocked INT NOT NULL DEFAULT 0,
    flags INT NOT NULL DEFAULT 0,
    miss_type SMALLINT,
    extra INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_combat_events_fight_offset ON combat_events(fight_id, offset_ms);
CREATE INDEX IF NOT EXISTS idx_combat_events_fight_type ON combat_events(fight_id, event_type);
