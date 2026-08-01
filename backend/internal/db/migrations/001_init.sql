-- +migrate Up
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS uploads (
    id UUID PRIMARY KEY,
    filename TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'processing', 'ready', 'failed')),
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS fights (
    id UUID PRIMARY KEY,
    upload_id UUID NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
    start_ts TIMESTAMPTZ NOT NULL,
    end_ts TIMESTAMPTZ NOT NULL,
    duration_ms BIGINT NOT NULL,
    title TEXT NOT NULL,
    "kill" BOOLEAN NOT NULL DEFAULT FALSE,
    participant_count INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_fights_upload_id ON fights(upload_id);

CREATE TABLE IF NOT EXISTS actors (
    id UUID PRIMARY KEY,
    upload_id UUID NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
    guid TEXT NOT NULL,
    name TEXT NOT NULL,
    flags BIGINT NOT NULL DEFAULT 0,
    is_player BOOLEAN NOT NULL DEFAULT FALSE,
    owner_guid TEXT,
    UNIQUE (upload_id, guid)
);

CREATE INDEX IF NOT EXISTS idx_actors_upload_id ON actors(upload_id);

CREATE TABLE IF NOT EXISTS fight_actors (
    fight_id UUID NOT NULL REFERENCES fights(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    PRIMARY KEY (fight_id, actor_id)
);

CREATE TABLE IF NOT EXISTS actor_stats (
    fight_id UUID NOT NULL REFERENCES fights(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    damage_done BIGINT NOT NULL DEFAULT 0,
    healing_done BIGINT NOT NULL DEFAULT 0,
    overheal BIGINT NOT NULL DEFAULT 0,
    damage_taken BIGINT NOT NULL DEFAULT 0,
    active_time_ms BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (fight_id, actor_id)
);

CREATE INDEX IF NOT EXISTS idx_actor_stats_damage ON actor_stats(fight_id, damage_done DESC);
CREATE INDEX IF NOT EXISTS idx_actor_stats_healing ON actor_stats(fight_id, healing_done DESC);
CREATE INDEX IF NOT EXISTS idx_actor_stats_taken ON actor_stats(fight_id, damage_taken DESC);

CREATE TABLE IF NOT EXISTS spell_stats (
    fight_id UUID NOT NULL REFERENCES fights(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    spell_id INT NOT NULL,
    spell_name TEXT NOT NULL,
    school INT NOT NULL DEFAULT 0,
    metric TEXT NOT NULL CHECK (metric IN ('damage', 'healing', 'damage_taken')),
    total BIGINT NOT NULL DEFAULT 0,
    hits INT NOT NULL DEFAULT 0,
    crits INT NOT NULL DEFAULT 0,
    ticks INT NOT NULL DEFAULT 0,
    PRIMARY KEY (fight_id, actor_id, spell_id, metric)
);

CREATE INDEX IF NOT EXISTS idx_spell_stats_actor ON spell_stats(fight_id, actor_id, total DESC);
