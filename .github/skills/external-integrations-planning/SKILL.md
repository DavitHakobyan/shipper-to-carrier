---
name: external-integrations-planning
description: Plan external data and payment integrations for the freight marketplace, including FMCSA, business registry, VIN, insurance, and ACH providers. Use when asked to design adapters, sync flows, or ownership boundaries for outside systems.
---

Use this skill when a task involves connecting the platform to third-party systems.

## Integration sources called out in the repository

- FMCSA public safety and licensing records
- state business registration databases
- VIN or vehicle databases
- insurance verification APIs
- Stripe or Dwolla for ACH payments

## Planning rules

For each integration, define:

1. why the platform needs it
2. the internal entity or workflow it supports
3. whether the data should be cached, normalized, or referenced on demand
4. how freshness, auditability, and failure handling affect the product flow

## Repository-specific expectations

- FMCSA and related compliance data support both carrier onboarding and scorecard computation.
- Business registry, identity, and VIN data support fraud-linking and entity resolution.
- Payment integrations should optimize for ACH-based settlement and explicitly preserve platform fee separation.
- External records should enrich internal trust decisions, but the proprietary scorecard logic remains platform-owned.

## Output expectations

When producing an integration plan, include:

- adapter boundaries
- ingestion or refresh flow
- internal source-of-truth decisions
- operational failure modes that affect user experience
- the minimum integration slice needed for phase 1 MVP versus later enhancements

Prefer designs that keep external dependencies behind clear interfaces so the core marketplace logic remains testable and portable.
