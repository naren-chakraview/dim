-- Initialize PostgreSQL database for dim cluster mode

-- Dedup store table: tracks processed message IDs across all instances
CREATE TABLE IF NOT EXISTS dedup_store (
    id SERIAL PRIMARY KEY,
    dedup_key VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_dedup_key ON dedup_store(dedup_key);
CREATE INDEX IF NOT EXISTS idx_expires_at ON dedup_store(expires_at);

-- Lineage table: cluster-wide message lineage (replaces per-instance SQLite)
CREATE TABLE IF NOT EXISTS lineage_records (
    id VARCHAR(36) PRIMARY KEY,
    route_name VARCHAR(255) NOT NULL,
    subject_id VARCHAR(255),
    message_body TEXT,
    message_headers TEXT,
    metadata TEXT,
    route_version VARCHAR(50),
    contract_version VARCHAR(50),
    principal VARCHAR(255),
    retention_policy VARCHAR(100),
    instance_id VARCHAR(50) NOT NULL,  -- Which instance processed this
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP,
    expires_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_lineage_subject ON lineage_records(subject_id);
CREATE INDEX IF NOT EXISTS idx_lineage_route ON lineage_records(route_name);
CREATE INDEX IF NOT EXISTS idx_lineage_instance ON lineage_records(instance_id);
CREATE INDEX IF NOT EXISTS idx_lineage_expires ON lineage_records(expires_at);

-- Purge events table: audit trail of data purges
CREATE TABLE IF NOT EXISTS purge_events (
    id VARCHAR(36) PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,  -- "manual_purge", "auto_reap"
    subject_id VARCHAR(255),
    purged_count INTEGER,
    reason TEXT,
    purged_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_purge_subject ON purge_events(subject_id);
CREATE INDEX IF NOT EXISTS idx_purge_type ON purge_events(event_type);

-- Generation tracking table: monitors config/generation rollout across cluster
CREATE TABLE IF NOT EXISTS generation_tracking (
    instance_id VARCHAR(50) PRIMARY KEY,
    current_generation BIGINT NOT NULL,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO dimadmin;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO dimadmin;
