CREATE TABLE IF NOT EXISTS external_record_snapshots (
    id TEXT PRIMARY KEY,
    carrier_account_id TEXT NOT NULL REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    source TEXT NOT NULL CHECK (source IN ('fmcsa')),
    source_key TEXT NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('matched', 'mismatch', 'unavailable')),
    payload_json JSONB NOT NULL DEFAULT '{}'::JSONB,
    checksum TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS fmcsa_registration_records (
    snapshot_id TEXT PRIMARY KEY REFERENCES external_record_snapshots(id) ON DELETE CASCADE,
    dot_number TEXT NOT NULL DEFAULT '',
    legal_name TEXT NOT NULL,
    address TEXT NOT NULL DEFAULT '',
    entity_type TEXT NOT NULL,
    authority_status TEXT NOT NULL,
    out_of_service BOOLEAN NOT NULL DEFAULT FALSE,
    operating_status TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS fmcsa_safety_records (
    snapshot_id TEXT PRIMARY KEY REFERENCES external_record_snapshots(id) ON DELETE CASCADE,
    safety_rating TEXT NOT NULL,
    crash_count INTEGER NOT NULL DEFAULT 0,
    inspection_count INTEGER NOT NULL DEFAULT 0,
    oos_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    incident_window_start TIMESTAMPTZ NOT NULL,
    incident_window_end TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS carrier_scorecards (
    id TEXT PRIMARY KEY,
    carrier_account_id TEXT NOT NULL REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    score_version TEXT NOT NULL,
    score_value INTEGER NOT NULL,
    score_band TEXT NOT NULL CHECK (score_band IN ('low', 'medium', 'high')),
    eligibility_tier TEXT NOT NULL CHECK (eligibility_tier IN ('review_pending', 'tier_0', 'restricted')),
    verification_completeness DOUBLE PRECISION NOT NULL,
    reason_summary TEXT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS carrier_score_inputs (
    id TEXT PRIMARY KEY,
    carrier_account_id TEXT NOT NULL REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    source_scorecard_id TEXT NOT NULL REFERENCES carrier_scorecards(id) ON DELETE CASCADE,
    input_type TEXT NOT NULL,
    source TEXT NOT NULL,
    value_numeric DOUBLE PRECISION NOT NULL DEFAULT 0,
    value_text TEXT NOT NULL DEFAULT '',
    effective_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS access_grants (
    id TEXT PRIMARY KEY,
    carrier_account_id TEXT NOT NULL REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    grant_type TEXT NOT NULL,
    grant_value TEXT NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    source_scorecard_id TEXT NOT NULL REFERENCES carrier_scorecards(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS fraud_signals (
    id TEXT PRIMARY KEY,
    carrier_account_id TEXT NOT NULL REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    source_scorecard_id TEXT NOT NULL REFERENCES carrier_scorecards(id) ON DELETE CASCADE,
    signal_type TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('low', 'medium', 'high')),
    status TEXT NOT NULL CHECK (status IN ('open')),
    detected_at TIMESTAMPTZ NOT NULL,
    evidence_json JSONB NOT NULL DEFAULT '{}'::JSONB,
    reviewed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS identity_links (
    id TEXT PRIMARY KEY,
    carrier_account_id TEXT NOT NULL REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    linked_entity_type TEXT NOT NULL,
    linked_entity_id TEXT NOT NULL,
    link_type TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL,
    evidence_json JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL
);
