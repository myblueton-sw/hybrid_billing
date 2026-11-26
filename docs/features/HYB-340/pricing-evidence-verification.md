---
type: VerificationRecord
title: Current pricing evidence prerequisite verification
description: Shared current policy evidence checks preserving explicit pricing-use authorization.
status: draft
---

# Scope

Aftera7a6651, HYB-340 adds the trusted internal pricingpolicy.CheckEvidence API and extracts Resolve's existing evidence checks into one locked helper. Root owns implementation. Independent refusal_review accepted the design before code and passed the resulting source/test review without blockers. It executed no tests. The future financial-subject adapter still needs actual caller authorization and fresh input/source evidence; no HTTP exposure, grant, capture behavior change or financial request is added.

# Validation

The complete pricingpolicy race suite passed1.171s (/private/tmp/hyb340-pricing-evidence-module.log). New tests compare evidence with authorized Resolve, prove denied Use remains denied, validate nested copy isolation, pending/revoked/foreign/missing/digest/scope/interval refusals, preserve target-unavailable errors, and verify monotonic clock rollback refusal without changing recorded decisions. Scoped go vet passed for pricingpolicy and apps/api/server (/private/tmp/hyb340-pricing-evidence-vet.log).

The affected server pricing/candidate race run passed: pricingpolicy1.594s and server96.791s (/private/tmp/hyb340-pricing-evidence-race.log). It covered TestEvidence, TestGoverned, TestApprovalOverlap, TestCurrentAuthority, TestCandidate, TestPricing and TestIntegrationActorsAndEffects patterns, not the entire server race suite. Independent platform_refusals passed the document scope review and inspected this final log. This increment changes no screen or copy; the previous actual M01/M03 review remains historical evidence, not a new UI test claim. macOS execution does not certify other platforms. Full W-23 remains active; no merge, deployment, payment or production operation occurred.
