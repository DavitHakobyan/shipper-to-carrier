# Copilot Instructions

## Current repository state

The repository currently contains a product specification in `shipper-to-carrier.md` and no application code yet. Treat that spec as the main source of truth for business requirements, MVP scope, and planned system boundaries until implementation docs are added.

## Build, test, and lint commands

No build, test, or lint commands are defined in the repository yet. There is no checked-in package manifest, module definition, or task runner at this stage, so future sessions should discover commands from committed project files instead of assuming a stack.

## High-level architecture

The product described in `shipper-to-carrier.md` is a direct freight marketplace connecting shippers to carriers without brokers. The intended platform shape is:

- A shipper-facing flow for account creation and load posting
- A carrier-facing flow for onboarding, identity verification, and load acceptance
- A scoring and trust engine that combines public records and platform history to gate access to higher-value loads
- An admin/fraud-monitoring surface for reviewing identity conflicts and risk signals
- ACH-based payment handling, with Stripe or Dwolla called out as likely providers

The spec recommends a Go API backed by PostgreSQL, with a dashboard frontend and external integrations for FMCSA data, state business registries, VIN/vehicle databases, insurance verification, and ACH payments. The carrier scorecard is described as the platform's moat, so score inputs, derived metrics, and fraud-linking logic should be treated as core domain architecture rather than incidental integrations.

Phase 1 MVP is centered on carrier signup and verification, FMCSA-backed scoring, shipper signup and load posting, basic matching, ACH payments, and a simple admin dashboard.

## Key conventions

- Use the marketplace language from the spec consistently: **shipper**, **carrier**, **load**, **scorecard**, **fraud signals**, and **ACH payment** are first-class domain concepts.
- Preserve the brokerless model in product and data design: carriers should see real posted rates, and platform fees should be modeled separately from broker-style markups.
- Model carrier trust as progressive access, not a single pass/fail check. New carriers start with smaller, lower-risk loads and unlock better loads as verified performance improves.
- Treat anti-fraud identity linking as a cross-cutting concern. Owner identity, business address, VINs, licensing, insurance, and historical entity relationships are meant to work together rather than as isolated checks.
- Keep proprietary scoring data internal. The spec explicitly frames aggregated carrier performance and trust data as a competitive advantage that should power marketplace behavior instead of being exposed as a standalone product.
