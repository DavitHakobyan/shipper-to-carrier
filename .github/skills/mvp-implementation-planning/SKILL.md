---
name: mvp-implementation-planning
description: Turn the freight marketplace product spec into a concrete implementation plan, milestone breakdown, and engineering backlog. Use when asked to plan delivery work for this repository.
---

Use this skill when the task is to translate `shipper-to-carrier.md` into an actionable engineering plan.

## Primary source of truth

Read `shipper-to-carrier.md` first. Treat it as the current product requirements document unless a newer repository document clearly supersedes part of it.

## Planning goals

Produce plans that keep the MVP centered on the repository's stated phase 1 scope:

1. Carrier signup and profile creation with identity verification
2. FMCSA data pull and a basic carrier scorecard
3. Shipper signup and load posting
4. Basic shipper-to-carrier matching with score-based gating
5. ACH payment integration
6. A simple admin dashboard for fraud monitoring

## How to structure the plan

Break work into a small number of deliverable slices instead of a single large backlog. Prefer phases or epics such as:

- identity and account model
- carrier scorecard pipeline
- load posting and matching
- payment flow
- admin/fraud review

For each slice, identify:

- the user-facing outcome
- the core services or modules required
- the main data entities
- external integrations and what data they own
- the most important dependencies and sequencing constraints

## Repository-specific constraints

- Preserve the brokerless marketplace model: carriers see actual posted load rates.
- Treat carrier scoring and fraud-linking as core product capabilities, not side utilities.
- Model trust as progressive access to higher-value loads rather than a binary approved/not-approved state.
- Keep proprietary score inputs and derived trust metrics internal to the platform.
- Favor plans that fit the recommended stack from the spec: Go API, PostgreSQL, dashboard frontend, external data adapters.

## Output expectations

Prefer plans that are implementation-ready:

- define the initial bounded contexts or services
- call out the first database tables or aggregates that unlock the MVP
- identify which parts need manual review or operational tooling
- surface major open decisions only when they materially change implementation

Do not drift into marketplace strategy, branding, or growth ideas unless the task explicitly asks for them.
