# Direct Shipper-to-Carrier Freight Marketplace
## Product Specification (from brainstorming session - April 25, 2026)

---

## Problem Statement

- Post-COVID freight rates dropped significantly, squeezing carrier and driver margins
- Freight brokers take 10–35% per load (average ~13–15%), acting as expensive middlemen
- Carriers lack visibility into what shippers are actually paying
- Shippers lack reliable, transparent data on carrier quality and history

---

## Core Concept

Build a direct freight marketplace that connects **shippers directly to carriers**, cutting out brokers. The platform provides:
- Transparent pricing (shippers post real rates, carriers see full amounts)
- A carrier scoring/credit system based on verifiable historical data
- Anti-fraud mechanisms to prevent bad actors from resetting their reputation

---

## Key Features

### 1. Carrier Credit Score System
A score built from:
- On-time delivery percentage
- Total delivery history (successes, failures)
- Safety incident history (pulled from FMCSA public records)
- Customer ratings from completed shipments on the platform
- Insurance and licensing status

**New Carrier Onboarding:**
- Start with small, low-risk, local loads
- Build track record progressively
- Unlock access to larger, better-paying loads as score improves
- Similar to a freelancer trust model — small tasks first, more responsibility as reliability is proven

### 2. Anti-Fraud / Identity Verification
To prevent bad carriers from closing company A and reopening as company B:
- Cross-reference owner/CEO identity
- Match business addresses
- Track VIN numbers of trucks and trailers
- Integrate with state business registration records and VIN databases

### 3. Shipper-to-Carrier Direct Matching
- Shippers post loads with real rates
- Carriers with qualifying credit scores can accept loads
- Threshold-based access: carriers must meet a minimum score to access higher-value loads
- No broker in the middle

### 4. Data & Scoring Engine
- Pull historical public records from FMCSA (up to 10–20 years)
- Aggregate: total deliveries, on-time rate, failures, safety incidents
- Build proprietary carrier scorecard — **do not sell this data**, it is the platform's moat
- As the platform grows, layer in platform-generated delivery history

---

## Monetization

- Charge a small platform fee per transaction (target: 5–8%)
- Still significantly cheaper than broker fees (10–35%)
- Payment processing via **ACH bank transfers** to minimize fees
  - Stripe ACH: ~1% or flat fee vs 2.9% + $0.30 for card
  - Consider offering two options: ACH (cheap, 1–2 days) and faster payment option (slightly higher fee) for drivers who need quick cash for fuel/expenses

---

## Tech Stack Recommendations

- **Database:** PostgreSQL (relational — data is not highly active, standard CRUD fits well)
- **Backend:** Go API layer
- **External Data Sources:**
  - FMCSA public safety records API
  - State business registration databases
  - VIN/vehicle databases
  - Stripe or Dwolla for ACH payments
- **Frontend:** Dashboard for shippers and carriers

---

## Data Sources to Integrate

| Source | Purpose |
|---|---|
| FMCSA | Safety ratings, incident history, licensing |
| State business registries | Fraud detection, identity cross-referencing |
| VIN databases | Truck/trailer identity tracking |
| Platform shipment data | On-time rate, completion rate, ratings |
| Insurance verification APIs | Coverage validation |

---

## Competitive Advantage

- Existing load boards (DAT, Truckstop.com) don't remove brokers — they enable them
- Trucker Path focuses on driver navigation/operational tools, not freight matching
- This platform's moat is the **proprietary carrier scoring data** — don't sell it, use it to power trust on the platform
- Network effects: more carriers → better data → more shippers → better loads → more carriers

---

## Phase 1 MVP Scope

1. Carrier signup + profile with identity verification
2. FMCSA data pull + basic scorecard
3. Shipper signup + load posting
4. Basic matching: shippers see carrier scores, carriers see load rates
5. ACH payment integration (Stripe or Dwolla)
6. Simple admin dashboard to monitor fraud signals

---

## Open Questions / Future Considerations

- Platform name (TBD)
- Mobile app for drivers (Flutter — consistent with existing stack)
- Load tracking / real-time visibility layer
- Integration with truck stop data, fuel prices (potential future expansion)
- Dispute resolution mechanism