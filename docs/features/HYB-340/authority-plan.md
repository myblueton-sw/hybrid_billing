---
type: ImplementationPlan
title: Governed charge approval amount limits
description: Independently reviewed next HYB-340 boundary for explicit module amount authority without gate permission side effects.
status: draft
---

# Design and ownership

Continue active HYB-340 on machine:mac_mini and its existing branch/PR250 after07621f0. Root remains sole writer and integration owner. Independent refusal_review accepted the concrete design before implementation. platform_refusals owns read-only TW review; actual Claude CLI reviews changed rendered M01 context. Owned files are new financialauthority adapter/commands/tests, pure module preparation API/tests, narrow assembly/governor/guard/hold/impact integration, M01 category copy/QA and these records.

Existing M03 grant creation requires a console ceiling, while auth rejects a console Approve grant. Do not bypass this human-only boundary. Implement module amount authority first and a separate reviewed direct-human permission proposal afterward. Both will be required for financial decisions. No AddGrant belongs in the amount-authority confirmation.

# Pinned command boundary

financial_authority.propose, confirm, inspect and revoke operate only on explicitly service-bound StageCharge authority proposals in sample mode. The proposal ID determines the authority ID financial-limit-{proposal}; validate its length before any effect. Explicit recipient, currency, canonical exact nonnegative limit, from and to dates are required. Dates use the module's inclusive UTC calendar-day semantics and 2000–2099 validation. Zero remains a valid explicit limit; no positive minimum or unbounded validity default is invented.

Registry.PrepareAuthority is pure and reuses existing profile parsing and authority validation; it neither advances the clock nor creates records. The service rejects noncanonical representations before proposal. A current governor must be active, hold installation authorization as user and console and belong to the module administrator set. Confirm rechecks the stored proposer and actual confirmer, and the active human recipient's current membership. Both governors must differ from the recipient. Fresh-key historical confirmation remains a receipt and cannot revive a revoked authority.

A bounded immutable service binding preserves digest, proposer and proposal/confirmation/revocation reasons. Output bounds, derived identifiers, collisions and binding capacity are checked before effects under the server domain lock. Confirm/revoke reconcile exact existing authority identity. Revocation is a current-governor operation on one named module authority, preserves history and is permitted under hold. Propose/confirm refuse under hold; inspect remains available. Output calls its current date/revocation status module_amount_authority, not effective financial approval permission.

# Verification and remaining integration

Verify malformed/noncanonical values, exact start/end dates, oversized IDs, recipient/proposer/confirmer changes, self confirmation/recipient participation, immutable retry reasons, competing confirmations, replay after revocation, missing/corrupt bindings, pre-effect output bounds, sidecar limits, fallible pending impact, step-up and SeedNone. Pending M01 financial_authority_proposals is separate from policies and future direct approval permission; unavailable is never zero. EN/KO/AR terminology and actual renders require independent reviews.

The following direct-human permission proposal must bind exact Approve action, dedicated financial resource kind, scope, validity and recipient; use two current governors and deterministic grant reconciliation, with unknown effects remaining unresolved. Module amount authority and permission remain separately revocable. The actual financial decision transport needs its own narrow review because ordinary commands carry ConsoleClient while Approval commands reject it. No generic client stripping, production defaults, real financial execution, deployment or merge is authorized here.
