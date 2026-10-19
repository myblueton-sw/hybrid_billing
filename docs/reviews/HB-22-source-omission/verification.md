---
type: Verification Record
title: HB-22 source omission remediation verification
description: Records ownership, changed artifacts, independent review results, checks performed and open follow-ups for the HB-22 remediation.
status: draft
---

# HB-22 verification

Tracking: Linear ticket [HB-22](https://linear.app/hybrid-billing/issue/HB-22). Accountable owner: Seung Woo Park. Root performed PM/planning, writing and integration. Machine: `mac_mini`. Branch `docs/HB-22-source-omission-remediation` starts at main `b5c65cf66992f703247b8e4713f86cef7835b767`. Date: 2026-10-14. Planning gate applies: documentation only; no product development.

## Changed artifacts

- New: [source pricing and cost](../../contracts/billing/source-pricing-and-cost-obligations.md), [source ERP and issuance](../../contracts/billing/source-erp-and-issuance-obligations.md), [source policy, onboarding and verification](../../contracts/operations/source-policy-onboarding-and-verification-obligations.md), [source omission trace](../../planning/v1.16/source-omission-trace.json), [findings](findings.md) and this record.
- Cross-reference paragraphs: `inputs-pricing-and-erp.md`, `holds-onboarding-and-favorites.md`, `domain-contracts.md`, `planning-reconciliation.md`.
- Unchanged: original DOCX, HB-15 reports, screen specifications, local planning companions.

## Independent review

| Reviewer | Scope | Result |
| --- | --- | --- |
| DBA (read-only) | Pricing contract, numeric oracles, consistency with PC-01, AR-06, Claim/numeric | PASS-WITH-FIXES; one BLOCKING (H22-P10 allowed under-allocation) and nine non-blocking |
| Architect (read-only) | ERP and operations contracts, command envelope, supersession, boundaries | PASS-WITH-FIXES; nine non-blocking |
| QA (read-only) | 33-finding closure, trace integrity, acceptance testability | PASS-WITH-FIXES; ten non-blocking |
| Re-review (read-only, separate agent) | All applied fixes | PASS, conditional on a missing link target (this file) and three citation mismatches; all four fixed afterwards |

All reviewers were separate read-only agents. None of them could approve product policy or merge. The final link and citation fixes were checked by root only and have not been re-reviewed.

## Checks performed by root

- Every trace locator resolves in the source extraction; every contract section and screen ID exists; every `H22-*` case referenced in the trace is defined in a contract, and every defined case is referenced: PASS.
- Recomputed synthetic figures: 108.50, 158,625 KRW, 600,000/300,000/100,000, overage 300 and the 7:6 split (161.54 and 138.46): PASS.
- Runtime, provider, ERP, database and user-task behavior: NOT_RUN. Every acceptance case is NOT_RUN.

## Open follow-ups

1. Amend the default-enablement sentence of the local `docs/planning/v1.16/architecture.md` external-execution paragraph to match EI-02.
2. Register the proposed `erp_route.*` commands in the [planning actions](../../contracts/commands/planning-actions.md) registry.
3. Bring the contract clauses into each screen specification listed in the trace's `screen_followup` field.
4. HB-15 findings FR-01–06, FA-01–14 and SR-01–03 remain open under their own tracking.

Planning completion still requires the owner's explicit declaration.
