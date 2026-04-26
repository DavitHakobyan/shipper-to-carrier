# MVP Implementation Roadmap

## Purpose

This document preserves the implementation roadmap for the shipper-to-carrier marketplace MVP.

It assumes:

- `docs/shipper-to-carrier-plan.md` is the effective requirements source because `shipper-to-carrier.md` is not present
- the platform is delivered as a modular monolith with a Go API, PostgreSQL, and a dashboard frontend
- carriers gain progressive access rather than a binary approved or rejected outcome
- transparent posted rates are preserved, with the platform fee modeled separately
- proprietary scorecard logic and fraud thresholds remain internal

## Assumptions and constraints

1. The MVP must preserve **transparent posted rates**: eligible carriers see the real load rate.
2. The **platform fee** is recorded separately from the shipper-posted rate.
3. Carrier trust is progressive and enforced through explicit eligibility tiers and access grants.
4. FMCSA, insurance, business registry, and ACH providers are external systems; the platform owns marketplace state.
5. Manual review remains part of the MVP for verification conflicts, FMCSA mismatches, and severe fraud signals.
6. The initial implementation should fit the recommended stack from the product plan: Go API, PostgreSQL, dashboard frontend, and adapter boundaries for external data.

## Initial bounded contexts

| Context | Owns |
|---|---|
| **identity** | shipper accounts, carrier accounts, auth, membership, onboarding state |
| **verification** | carrier owners, addresses, authority links, verification cases, submitted evidence |
| **external-evidence** | FMCSA snapshots, registry snapshots, insurance verification snapshots |
| **trust** | carrier score inputs, carrier scorecards, fraud signals, identity links, access grants |
| **marketplace** | loads, load requirements, eligibility checks, load acceptance |
| **payments** | ACH account setup, payment lifecycle, payout lifecycle, platform fee ledger |
| **admin-ops** | manual reviews, overrides, audit trail |

## Critical path

1. Platform foundation and identity
2. Carrier onboarding and verification workflow
3. FMCSA ingestion and basic carrier scorecard
4. Shipper load posting
5. Score-gated carrier matching and load acceptance
6. ACH payment flow
7. Admin fraud and review dashboard

Without the first three steps, the marketplace cannot safely expose loads to carriers.

## Milestone roadmap

### 1. Platform foundation

**Outcome**
Establish a runnable system skeleton with shared auth, migrations, API structure, and dashboard shell.

**Go modules**

- `identity`
- `platform/http`
- `platform/auth`
- `platform/store`
- `platform/events`

**PostgreSQL tables**

- `accounts`
- `memberships`
- auth/session tables
- `audit_events`
- `outbox_events`

**API contracts**

- carrier and shipper account creation
- session/auth endpoints
- health/config endpoints

**Dependencies**

- none

**Acceptance criteria**

- shipper and carrier actors can create accounts
- migrations and local environment boot cleanly
- dashboard can authenticate and route by role

### 2. Carrier onboarding and verification workflow

**Outcome**
A carrier can submit company, owner, address, authority, and insurance details and move through explicit onboarding stages.

**Go modules**

- `carrieridentity`
- `verification`

**PostgreSQL tables**

- `carrier_accounts`
- `carrier_profiles`
- `carrier_owner_identities`
- `carrier_addresses`
- `carrier_authority_links`
- `carrier_insurance_policies`
- `verification_cases`
- `verification_requirements`
- `verification_documents`
- `verification_events`
- `verification_decisions`

**API contracts**

- `POST /carriers`
- `POST /carriers/{id}/owners`
- `POST /carriers/{id}/authority`
- `POST /carriers/{id}/insurance`
- `GET /carriers/{id}/onboarding-status`

**Dependencies**

- platform foundation

**Acceptance criteria**

- onboarding state machine is enforced
- missing verification requirements are visible
- manual review path exists for exceptions
- the model avoids a binary approved-carrier shortcut

### 3. FMCSA evidence ingestion and basic carrier scorecard

**Outcome**
The platform can fetch FMCSA records, persist evidence snapshots, generate a basic carrier scorecard, and assign a starting eligibility tier.

**Go modules**

- `externalevidence`
- `trust`

**PostgreSQL tables**

- `external_record_snapshots`
- `fmcsa_registration_records`
- `fmcsa_safety_records`
- `carrier_score_inputs`
- `carrier_scorecards`
- `access_grants`
- `fraud_signals`
- `identity_links`

**External integrations**

- FMCSA
- optional first-pass business registry hook
- insurance verification lookup or callback

**Dependencies**

- carrier onboarding data must exist first

**Acceptance criteria**

- FMCSA lookup runs when DOT or MC is submitted
- stale versus current FMCSA evidence is tracked explicitly
- scorecard outputs an internal eligibility tier
- fraud signals can block promotion or route to manual review

### 4. Shipper signup and load posting

**Outcome**
A shipper can create an account and post a load with a real rate and qualification requirements.

**Go modules**

- `shipperidentity`
- `marketplace/loadposting`

**PostgreSQL tables**

- `shipper_accounts`
- `shipper_profiles`
- `loads`
- `load_stops`
- `load_requirements`
- `load_documents`

**API contracts**

- `POST /shippers`
- `POST /loads`
- `PATCH /loads/{id}`
- `GET /loads/{id}`

**Dependencies**

- platform foundation

**Acceptance criteria**

- a shipper can post a load with posted rate, lane, equipment type, and timing
- the platform fee is stored separately from the posted rate
- qualification constraints are attached to the load

### 5. Score-gated matching and carrier acceptance

**Outcome**
Eligible carriers can discover and accept loads based on verification completeness and carrier scorecard tier.

**Go modules**

- `marketplace/matching`
- `marketplace/eligibility`

**PostgreSQL tables**

- `load_visibility_rules`
- `load_candidate_views` or a materialized read model
- `load_acceptances`
- `load_status_history`

**Key logic**

- evaluate `access_grants` against `load_requirements`
- preserve transparent posted rates
- restrict high-value loads for new or risky carriers

**Dependencies**

- load posting
- scorecards and access grants

**Acceptance criteria**

- only eligible carriers can view and accept gated loads
- Tier 0 carriers see only low-risk loads
- acceptance transitions are explicit and auditable
- shippers see carrier score outcomes, not proprietary scoring internals

### 6. ACH payments and admin operations

**Outcome**
The platform can initiate ACH-backed payment flows for accepted and completed loads and provide an admin surface for fraud and verification review.

**Go modules**

- `payments`
- `adminops`

**PostgreSQL tables**

- `payment_accounts`
- `payments`
- `payouts`
- `platform_fees`
- `payment_events`
- `admin_reviews`
- `admin_actions`

**External integrations**

- Stripe ACH or Dwolla behind one adapter boundary

**Dependencies**

- load acceptance lifecycle
- verification and trust review data

**Acceptance criteria**

- ACH flows are asynchronous and stateful
- platform fee ledger remains separate from the load rate
- admins can review fraud signals, mismatches, and overrides
- payout holds can be applied when a carrier becomes restricted

## Suggested engineering backlog

1. Bootstrap platform skeleton
2. Implement account and auth model
3. Implement carrier onboarding state machine
4. Add verification case and requirement engine
5. Integrate FMCSA snapshot ingestion
6. Implement basic carrier scorecard and access grants
7. Implement shipper account and load posting
8. Implement load eligibility and carrier discovery
9. Implement load acceptance lifecycle
10. Add ACH provider abstraction and payment state machine
11. Build admin verification and fraud dashboard
12. Add audit and override tooling

## Early schema that unlocks the MVP fastest

- `carrier_accounts`
- `carrier_owner_identities`
- `carrier_addresses`
- `carrier_authority_links`
- `verification_cases`
- `verification_requirements`
- `external_record_snapshots`
- `fmcsa_registration_records`
- `carrier_scorecards`
- `access_grants`
- `shipper_accounts`
- `loads`
- `load_requirements`
- `load_acceptances`

## Major risks and fallback behavior

| Risk | Impact | Fallback |
|---|---|---|
| FMCSA latency or outage | carrier onboarding stalls | keep carrier in review or tightly limited Tier 0 after manual review |
| carrier identity mismatch across sources | false positives or missed fraud | create a fraud signal and require admin review rather than auto-reject |
| score logic spread through marketplace handlers | inconsistent gating | centralize eligibility in trust and access-grant rules |
| ACH delays or failures | payout confusion | model asynchronous payment states and admin-visible retries or holds |
| exposing too much trust detail | gaming the system | expose tier and reason categories, not raw thresholds or fraud confidence |

## Delivery sequencing recommendation

**Phase A**

- milestone 1
- milestone 2

**Phase B**

- milestone 3

**Phase C**

- milestone 4
- milestone 5

**Phase D**

- milestone 6

This sequence brings carrier onboarding and trust online before opening the brokerless marketplace broadly, which is the safest way to preserve score-gated matching and fraud controls from day one.
