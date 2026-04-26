CREATE TABLE IF NOT EXISTS carrier_accounts (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    legal_name TEXT NOT NULL,
    doing_business_as TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('active')),
    onboarding_stage TEXT NOT NULL CHECK (onboarding_stage IN ('draft', 'business_submitted', 'authority_linked', 'insurance_submitted', 'review_pending')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS carrier_profiles (
    carrier_account_id TEXT PRIMARY KEY REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    contact_phone TEXT NOT NULL,
    contact_email TEXT NOT NULL,
    fleet_size_declared INTEGER NOT NULL DEFAULT 0,
    operating_regions TEXT[] NOT NULL DEFAULT '{}',
    preferred_load_types TEXT[] NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS carrier_owner_identities (
    id TEXT PRIMARY KEY,
    carrier_account_id TEXT NOT NULL REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    full_name TEXT NOT NULL,
    phone TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL,
    ownership_role TEXT NOT NULL,
    is_primary_contact BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS carrier_addresses (
    id TEXT PRIMARY KEY,
    carrier_account_id TEXT NOT NULL REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    address_type TEXT NOT NULL,
    line1 TEXT NOT NULL,
    line2 TEXT NOT NULL DEFAULT '',
    city TEXT NOT NULL,
    state TEXT NOT NULL,
    postal_code TEXT NOT NULL,
    country TEXT NOT NULL,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS carrier_authority_links (
    id TEXT PRIMARY KEY,
    carrier_account_id TEXT NOT NULL UNIQUE REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    dot_number TEXT NOT NULL DEFAULT '',
    mc_number TEXT NOT NULL DEFAULT '',
    usdot_status TEXT NOT NULL DEFAULT '',
    authority_type TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS carrier_insurance_policies (
    id TEXT PRIMARY KEY,
    carrier_account_id TEXT NOT NULL REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    provider_name TEXT NOT NULL,
    policy_number_hash TEXT NOT NULL,
    coverage_type TEXT NOT NULL,
    effective_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    verification_status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS verification_cases (
    id TEXT PRIMARY KEY,
    carrier_account_id TEXT NOT NULL UNIQUE REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    case_type TEXT NOT NULL CHECK (case_type IN ('onboarding')),
    status TEXT NOT NULL CHECK (status IN ('open', 'review_pending')),
    opened_at TIMESTAMPTZ NOT NULL,
    closed_at TIMESTAMPTZ,
    assigned_admin_id TEXT REFERENCES accounts(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS verification_requirements (
    id TEXT PRIMARY KEY,
    verification_case_id TEXT NOT NULL REFERENCES verification_cases(id) ON DELETE CASCADE,
    requirement_type TEXT NOT NULL CHECK (requirement_type IN ('business_profile', 'owner_identity', 'operating_address', 'authority_link', 'insurance_policy')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'satisfied')),
    satisfied_at TIMESTAMPTZ,
    notes TEXT NOT NULL DEFAULT '',
    UNIQUE (verification_case_id, requirement_type)
);

CREATE TABLE IF NOT EXISTS verification_documents (
    id TEXT PRIMARY KEY,
    verification_case_id TEXT NOT NULL REFERENCES verification_cases(id) ON DELETE CASCADE,
    document_type TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    status TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS verification_decisions (
    id TEXT PRIMARY KEY,
    verification_case_id TEXT NOT NULL REFERENCES verification_cases(id) ON DELETE CASCADE,
    decision_type TEXT NOT NULL,
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    reason_code TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS verification_events (
    id TEXT PRIMARY KEY,
    carrier_account_id TEXT NOT NULL REFERENCES carrier_accounts(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    event_payload_json JSONB NOT NULL DEFAULT '{}'::JSONB,
    occurred_at TIMESTAMPTZ NOT NULL
);
