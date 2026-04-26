# Carrier Onboarding and FMCSA Verification Domain Model

## Purpose

This document preserves the implementation-ready domain model for carrier onboarding and FMCSA verification in the shipper-to-carrier marketplace MVP.

It assumes:

- carriers gain **progressive access** rather than a binary approved or rejected outcome
- FMCSA data is an external evidence source, not the sole owner of carrier truth
- proprietary scorecard and fraud threshold logic remain internal to the platform
- identity linking across owners, addresses, VINs, authority, and insurance is a first-class anti-fraud concern

## Conceptual model

```text
CarrierAccount
  ├─ CarrierProfile
  ├─ CarrierOwnerIdentity [1..n]
  ├─ CarrierAddress [1..n]
  ├─ CarrierEquipment [0..n]
  ├─ CarrierInsurancePolicy [0..n]
  ├─ CarrierAuthorityLink [0..n]
  ├─ VerificationCase [1..n]
  │    ├─ VerificationRequirement [1..n]
  │    ├─ VerificationDocument [0..n]
  │    ├─ VerificationDecision [0..n]
  │    └─ VerificationEvent [0..n]
  ├─ ExternalRecordSnapshot [0..n]
  │    ├─ FMCSARegistrationRecord
  │    ├─ FMCSASafetyRecord
  │    └─ BusinessRegistryRecord
  ├─ IdentityLink [0..n]
  ├─ FraudSignal [0..n]
  ├─ CarrierScoreInput [0..n]
  ├─ CarrierScorecard [0..n]
  └─ AccessGrant [0..n]
```

## Entity groups

### 1. Canonical business identities

These are platform-owned records referenced by the rest of the system.

| Entity | Responsibility | Important fields |
|---|---|---|
| **CarrierAccount** | Canonical carrier business identity in the marketplace | `id`, `legal_name`, `doing_business_as`, `status`, `onboarding_stage`, `created_at` |
| **CarrierProfile** | Operational carrier profile | `carrier_account_id`, `contact_phone`, `contact_email`, `fleet_size_declared`, `operating_regions`, `preferred_load_types` |
| **CarrierOwnerIdentity** | Human identities tied to the carrier | `id`, `carrier_account_id`, `full_name`, `dob_hash`, `government_id_hash`, `phone`, `email`, `ownership_role`, `is_primary_contact` |
| **CarrierAddress** | Normalized address history for verification and matching | `id`, `carrier_account_id`, `address_type`, `line1`, `city`, `state`, `postal_code`, `country`, `valid_from`, `valid_to` |
| **CarrierEquipment** | Declared truck and trailer assets | `id`, `carrier_account_id`, `vin`, `plate_number`, `equipment_type`, `status` |
| **CarrierInsurancePolicy** | Insurance coverage required for eligibility | `id`, `carrier_account_id`, `provider_name`, `policy_number_hash`, `coverage_type`, `effective_at`, `expires_at`, `verification_status` |
| **CarrierAuthorityLink** | Links a carrier to FMCSA identifiers | `id`, `carrier_account_id`, `dot_number`, `mc_number`, `usdot_status`, `authority_type` |

### 2. External verification records

These are source-specific facts. Raw external data should not overwrite canonical carrier identity.

| Entity | Responsibility | Important fields |
|---|---|---|
| **ExternalRecordSnapshot** | Raw or normalized wrapper for any external lookup | `id`, `carrier_account_id`, `source`, `source_key`, `fetched_at`, `status`, `payload_json`, `checksum` |
| **FMCSARegistrationRecord** | Carrier registration and authority evidence | `snapshot_id`, `dot_number`, `legal_name`, `address`, `entity_type`, `authority_status`, `out_of_service`, `operating_status` |
| **FMCSASafetyRecord** | Safety and inspection evidence | `snapshot_id`, `safety_rating`, `crash_count`, `inspection_count`, `oos_rate`, `incident_window_start`, `incident_window_end` |
| **BusinessRegistryRecord** | State company registration evidence | `snapshot_id`, `state`, `entity_name`, `registration_status`, `registered_agent`, `formed_at` |
| **InsuranceVerificationRecord** | Third-party coverage verification result | `snapshot_id`, `policy_status`, `verified_named_insured`, `coverage_limits`, `expires_at` |

### 3. Platform-generated operational events

These make onboarding auditable and replayable.

| Entity | Responsibility | Important fields |
|---|---|---|
| **VerificationCase** | One onboarding review flow for a carrier | `id`, `carrier_account_id`, `case_type`, `status`, `opened_at`, `closed_at`, `assigned_admin_id` |
| **VerificationRequirement** | Required steps inside a case | `id`, `verification_case_id`, `requirement_type`, `status`, `due_at`, `satisfied_at` |
| **VerificationDocument** | Submitted onboarding evidence | `id`, `verification_case_id`, `document_type`, `storage_key`, `status`, `uploaded_at` |
| **VerificationDecision** | Manual or automated case decision | `id`, `verification_case_id`, `decision_type`, `actor_type`, `actor_id`, `reason_code`, `created_at` |
| **VerificationEvent** | Immutable event history for onboarding | `id`, `carrier_account_id`, `event_type`, `occurred_at`, `event_payload_json` |

### 4. Derived trust and anti-fraud outputs

These are internal platform outputs, not externally authoritative facts.

| Entity | Responsibility | Important fields |
|---|---|---|
| **IdentityLink** | Probable linkage between this carrier and another identity cluster | `id`, `carrier_account_id`, `linked_entity_type`, `linked_entity_id`, `link_type`, `confidence`, `evidence_json` |
| **FraudSignal** | Discrete risk indicators | `id`, `carrier_account_id`, `signal_type`, `severity`, `status`, `detected_at`, `evidence_json`, `reviewed_at` |
| **CarrierScoreInput** | Explainable inputs to scorecard calculation | `id`, `carrier_account_id`, `input_type`, `source`, `value_numeric`, `value_text`, `effective_at` |
| **CarrierScorecard** | Internal trust outcome used for eligibility | `id`, `carrier_account_id`, `score_version`, `score_band`, `eligibility_tier`, `verification_completeness`, `generated_at` |
| **AccessGrant** | Explicit capabilities unlocked for the carrier | `id`, `carrier_account_id`, `grant_type`, `grant_value`, `granted_at`, `revoked_at`, `source_scorecard_id` |

## Critical relationships

1. **CarrierAccount** is the root aggregate for onboarding.
2. **CarrierOwnerIdentity**, **CarrierAddress**, **CarrierEquipment**, **CarrierInsurancePolicy**, and **CarrierAuthorityLink** support verification and identity linking.
3. **VerificationCase** tracks process state, while **VerificationRequirement** tracks granular completion.
4. **ExternalRecordSnapshot** stores FMCSA and adjacent evidence without mutating canonical platform identity.
5. **FraudSignal** and **IdentityLink** are generated from canonical data plus external evidence.
6. **CarrierScorecard** consumes verified facts and operational history, then produces **AccessGrant** records that control progressive load access.

## State machines

### Carrier onboarding state

This is the business-facing lifecycle and should not be reduced to a single approval flag.

```text
Draft
  -> ContactVerified
  -> BusinessSubmitted
  -> AuthorityLinked
  -> FMCSAEvidenceFetched
  -> InsuranceSubmitted
  -> ReviewPending
  -> Tier0Eligible
  -> Tier1Eligible
  -> Tier2Eligible
  -> Restricted
  -> Suspended
```

#### Triggers

- `Draft -> ContactVerified`: phone or email verification completed
- `ContactVerified -> BusinessSubmitted`: legal entity, owner, and address data submitted
- `BusinessSubmitted -> AuthorityLinked`: DOT or MC linked, or pending authority recorded
- `AuthorityLinked -> FMCSAEvidenceFetched`: FMCSA lookup succeeds or partial evidence is captured
- `FMCSAEvidenceFetched -> InsuranceSubmitted`: insurance data submitted
- `InsuranceSubmitted -> ReviewPending`: minimum onboarding packet complete
- `ReviewPending -> Tier0Eligible`: automated checks pass and no blocking fraud signal exists
- `Tier0Eligible -> Tier1Eligible`: stronger verification completeness plus successful platform history or stronger external evidence
- `Tier1Eligible -> Tier2Eligible`: stronger scorecard and delivery history
- `Any active state -> Restricted`: severe fraud signal, expired insurance, or authority downgrade
- `Restricted -> prior tier`: issue resolved and reviewed
- `Any state -> Suspended`: confirmed fraud or regulatory disqualification

### FMCSA verification state

FMCSA evidence refreshes independently of onboarding.

```text
NotRequested
  -> PendingLookup
  -> Matched
  -> Mismatch
  -> Unavailable
  -> Stale
  -> Superseded
```

#### Triggers

- `NotRequested -> PendingLookup`: DOT or MC submitted, or refresh scheduled
- `PendingLookup -> Matched`: FMCSA record found and legal identity sufficiently matches
- `PendingLookup -> Mismatch`: FMCSA record found but name, address, or authority conflicts exist
- `PendingLookup -> Unavailable`: source unavailable or request failed
- `Matched/Mismatch -> Stale`: freshness SLA exceeded
- `Stale -> PendingLookup`: retry scheduled
- `Matched/Mismatch -> Superseded`: newer snapshot replaces prior snapshot

## Progressive trust model

Model eligibility as explicit tiers with access grants.

| Tier | Minimum conditions | Access grants |
|---|---|---|
| **Tier 0** | contact verified, business submitted, FMCSA matched or manually reviewed, insurance present, no blocking fraud signal | low-risk loads only, low payout caps, manual review on first acceptance |
| **Tier 1** | Tier 0 plus stronger verification completeness, clean FMCSA status, and first successful load or stronger external history | broader load pool, higher posted-rate ceilings, faster acceptance |
| **Tier 2** | Tier 1 plus consistent platform performance and low risk profile | high-value load access, higher payout limits, reduced manual review |
| **Restricted** | unresolved severe signal or compliance gap | limited or blocked visibility, no new load acceptance, payout holds possible |

### Example access grants

- `load_value_cap = 2500`
- `allowed_load_risk_band = low`
- `requires_manual_dispatch_review = true`
- `max_open_loads = 1`

## Ownership boundaries

### Carrier Identity and Verification

Owns:

- `CarrierAccount`
- `CarrierProfile`
- `CarrierOwnerIdentity`
- `CarrierAddress`
- `CarrierAuthorityLink`
- `VerificationCase`
- `VerificationRequirement`
- `VerificationDocument`
- onboarding state transitions

### External Evidence

Owns:

- FMCSA adapters
- `ExternalRecordSnapshot`
- `FMCSARegistrationRecord`
- `FMCSASafetyRecord`
- refresh scheduling
- staleness tracking

### Trust and Fraud

Owns:

- `IdentityLink`
- `FraudSignal`
- `CarrierScoreInput`
- `CarrierScorecard`
- `AccessGrant`

### Admin Operations

Owns:

- manual review queues
- overrides
- `VerificationDecision`
- operational fraud resolution

**Rule:** FMCSA fetching and normalization belongs to External Evidence, but carrier eligibility decisions belong to Trust and Fraud plus onboarding policy.

## Integration points and responsibilities

### FMCSA

Pattern: the platform **pulls**, stores snapshots, and derives internal outcomes.

Responsibilities:

- trigger lookup when DOT or MC is submitted
- run nightly refresh for active carriers
- allow on-demand refresh during review or before tier promotion
- normalize source data into source-specific tables
- mark stale snapshots rather than silently overwriting trust outcomes

Fallback behavior:

- if FMCSA is unavailable, keep onboarding in `ReviewPending` or allow tightly limited `Tier0Eligible` only after manual review
- never auto-promote on missing FMCSA data
- keep last matched snapshot but mark it stale after freshness expiry

### Business registry

- validate entity existence
- compare legal name and address
- feed mismatch signals rather than direct eligibility decisions

### Insurance verification

- confirm active coverage
- expiry should automatically trigger restriction

### VIN and equipment data

- support identity linking and anti-fraud review
- can be optional for the first MVP tier, but should be modeled from the start

## Early PostgreSQL tables

### Core onboarding

- `carrier_accounts`
- `carrier_profiles`
- `carrier_owner_identities`
- `carrier_addresses`
- `carrier_authority_links`
- `carrier_insurance_policies`

### Verification workflow

- `verification_cases`
- `verification_requirements`
- `verification_documents`
- `verification_decisions`
- `verification_events`

### External evidence

- `external_record_snapshots`
- `fmcsa_registration_records`
- `fmcsa_safety_records`

### Trust and fraud

- `identity_links`
- `fraud_signals`
- `carrier_score_inputs`
- `carrier_scorecards`
- `access_grants`

## Go service outline

1. **carrieridentity**
   - carrier registration
   - owner and address capture
   - onboarding state machine enforcement
2. **externalevidence**
   - FMCSA client
   - snapshot persistence
   - refresh scheduler
3. **trust**
   - fraud signal generation
   - identity linking
   - score input assembly
   - access grant calculation
4. **adminops**
   - review actions
   - override APIs
   - audit views

## API shape

### Carrier-facing API

- `POST /carriers`
- `POST /carriers/{id}/owners`
- `POST /carriers/{id}/authority`
- `POST /carriers/{id}/insurance`
- `GET /carriers/{id}/onboarding-status`

Expose:

- onboarding stage
- missing requirements
- verification completeness
- eligibility tier outcome

Do not expose:

- raw fraud score
- internal linkage confidence
- proprietary threshold logic

### Admin API

- `GET /admin/verification-cases`
- `POST /admin/verification-cases/{id}/decisions`
- `GET /admin/carriers/{id}/signals`
- `POST /admin/carriers/{id}/override-tier`

## Assumptions and constraints

1. `docs/shipper-to-carrier-plan.md` is the effective source of truth because no separate `shipper-to-carrier.md` is present.
2. FMCSA data is authoritative for authority and safety evidence, but not guaranteed fresh or always available.
3. The MVP should favor progressive eligibility over blocking every carrier until perfect verification is available.
4. Proprietary score thresholds and fraud heuristics remain internal.
5. ACH and broader marketplace history are outside this document except where future access grants may affect payout limits.
6. Some carriers will have incomplete or inconsistent external records, so manual review must remain first-class.

## Major risks and fallback behavior

| Risk | Impact | Fallback |
|---|---|---|
| FMCSA outage or latency | onboarding stalls | hold at `ReviewPending` or allow tightly limited Tier 0 after manual review |
| Name or address mismatch between carrier and FMCSA | false blocks or missed fraud | create `FraudSignal`, require admin review, do not auto-reject |
| Insurance expiration after onboarding | hidden marketplace risk | transition to `Restricted` and revoke relevant access grants |
| Reused owner, address, or VIN across entities | fraud evasion | create `IdentityLink` and `FraudSignal`, require review before tier increase |
| One-off flags replacing state machines | inconsistent policy | enforce explicit state transitions and immutable verification events |
