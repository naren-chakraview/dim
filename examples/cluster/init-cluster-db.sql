-- Initialize PostgreSQL schema for dim cluster coordination
-- Tables for deduplication and lineage tracking across instances

-- Deduplication table: tracks message IDs across cluster
CREATE TABLE IF NOT EXISTS dedup_store (
    id SERIAL PRIMARY KEY,
    message_id TEXT NOT NULL UNIQUE,
    message_hash TEXT NOT NULL,
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_by TEXT NOT NULL,  -- Instance that processed it
    route_name TEXT,
    ttl_expires_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dedup_message_id ON dedup_store(message_id);
CREATE INDEX IF NOT EXISTS idx_dedup_ttl ON dedup_store(ttl_expires_at);

-- Lineage table: unified audit trail across all instances
CREATE TABLE IF NOT EXISTS lineage_records (
    id SERIAL PRIMARY KEY,
    correlation_id TEXT NOT NULL,
    message_id TEXT,
    route_name TEXT NOT NULL,
    step_name TEXT,
    action TEXT,  -- ingested, processed, routed, sink_write, error
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    instance_id TEXT NOT NULL,  -- Which instance processed this
    status TEXT,  -- success, error, filtered
    error_message TEXT,
    metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_lineage_correlation ON lineage_records(correlation_id);
CREATE INDEX IF NOT EXISTS idx_lineage_route ON lineage_records(route_name);
CREATE INDEX IF NOT EXISTS idx_lineage_instance ON lineage_records(instance_id);
CREATE INDEX IF NOT EXISTS idx_lineage_timestamp ON lineage_records(timestamp);

-- Log entries for monitoring cluster health
CREATE TABLE IF NOT EXISTS cluster_events (
    id SERIAL PRIMARY KEY,
    event_type TEXT,  -- instance_joined, instance_left, dedup_hit, lineage_recorded
    instance_id TEXT NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    details JSONB
);

CREATE INDEX IF NOT EXISTS idx_cluster_events_instance ON cluster_events(instance_id);
CREATE INDEX IF NOT EXISTS idx_cluster_events_timestamp ON cluster_events(timestamp);

-- Generation table: for coordinating hot reloads across instances
CREATE TABLE IF NOT EXISTS generation_tracker (
    id SERIAL PRIMARY KEY,
    generation_number INTEGER NOT NULL,
    route_name TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT NOT NULL,
    UNIQUE(route_name)
);

CREATE INDEX IF NOT EXISTS idx_generation_route ON generation_tracker(route_name);
