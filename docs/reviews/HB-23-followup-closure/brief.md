---
type: Work Brief
title: HB-23 follow-up and remaining finding closure plan
description: Defines scope, ownership, allowed files, validation and deferrals for closing HB-22 follow-ups and HB-15 FA and SR findings against main 35e2d4c.
status: draft
---

# HB-23 plan

Tracking: Linear ticket [HB-23](https://linear.app/hybrid-billing/issue/HB-23). Owner: Seung Woo Park. Machine: `mac_mini`. Branch `docs/HB-23-followup-closure` from main `35e2d4c`. Date: 2026-10-22. Planning gate applies: documentation only.

## Basis

- HB-15 [re-audit](../HB-15-full-planning/re-audit.md) FA-01 to FA-14 and [source review](../HB-15-full-planning/source-review.md) SR-01 to SR-03 were written against HB-21 schema revision `129c525`; merged HB-21 content is that schema. They are open.
- FR-01 to FR-05 were remediated by HB-16 with recorded residual boundaries ([HB-16 verification](../HB-16-remediation/verification.md)); FR-02 and FR-04 keep residuals noted in the re-audit. FR-06 is partial and stays open under FA-14 and FA-12. The HB-22 statement that FR-01 to FR-06 remain open is partly wrong and is corrected narrowly.
- HB-22 follow-ups are listed in [HB-22 verification](../HB-22-source-omission/verification.md).

## Approach

Restore missing relationships as typed, source-cited planning contracts. Amend existing Git documents only where a current statement contradicts the source or omits a guard (FA-01, FA-02, FA-03). Do not invent customer values, engines or table layouts; unknowns stay decisions.

Merged screen specifications needed here (B09, B11, I04, S02, S06, U02) are edited in place. The other 39 specifications are authorized local Korean companions outside Git. Their bindings are recorded in one English screen-binding register; adopting the text into each specification belongs to [HB-24](https://linear.app/hybrid-billing/issue/HB-24) under FA-14. A register entry does not remove contradicting local text, so every finding that depends on a local specification (for example FA-05 in local B05, FA-04 in B04 and B08, FA-13 in P02 and M05, SR-03 in P06) is dispositioned as partially remediated with the residual named, never as remediated.

## Work packages

| Package | Findings | Writer | Allowed files |
| --- | --- | --- | --- |
| W1 schema | FA-01, FA-02, FA-03, FA-06 to FA-12, SR-01, SR-02 | implementer 1 | `docs/architecture/data-ownership/logical-data-schema.md`, `monthly-settlement-schema.md`, `evidence-detail-schema.md`, new `docs/architecture/data-ownership/source-relationship-bindings.md`, `docs/contracts/billing/inputs-pricing-and-erp.md`, `docs/contracts/billing/source-pricing-and-cost-obligations.md` |
| W2 commands and screens | FA-04, FA-05, FA-13, SR-03, HB-22 follow-ups 2 and 3 | implementer 2 | `docs/contracts/commands/planning-actions.md`, `docs/ui/screens/{B09,B11,I04,S02,S06,U02}/spec.md`, new `docs/ui/screen-binding-register.md` |
| W3 integration | HB-22 follow-up 1, FR correction, FA-04 acceptance mapping, dispositions | root | local `docs/planning/v1.16/architecture.md` (outside Git; exception required by HB-22 follow-up 1, before and after SHA-256 recorded), local `docs/planning/v1.16/acceptance-trace.json` (outside Git; MS-05 and MS-09 mapping, hashes recorded), `docs/reviews/HB-22-source-omission/verification.md`, `docs/reviews/HB-23-followup-closure/*`, `docs/planning/v1.16/planning-reconciliation.md` |

One writer per file. W1 and W2 run in separate worktrees; root integrates. Reviews start only after both writers finish, so at most four agents are active including root.

## Deferrals

- FA-14 (portable canonical English baseline and companion publication) and the FR-06 remainder are deferred to [HB-24](https://linear.app/hybrid-billing/issue/HB-24). This ticket does not claim them.
- Runtime, database, provider, ERP and UI behavior: NOT_RUN.

## Validation

- Each finding gets a disposition row: remediated with location, partially remediated with residual, or deferred with ticket.
- Every new statement cites a source locator or a merged baseline line; derived proposals are labelled.
- Root checks: OKF frontmatter, local link targets, JSON validity, case-ID uniqueness, credential pattern scan, whitespace.
- Independent read-only reviews after writing: DBA (FA-01 to FA-04, FA-11, FA-12, SR-02: financial keys, FX, credit, adjustment), security (FA-02 issuer isolation, FA-08 permission-safe return, FA-09 enrollment and authentication policy, FA-10 transfer and disclosure, W2 command permissions), architect (W1 relationships, W2 command envelope), QA (dispositions, screen register, acceptance testability). Reviews run in two waves of at most three agents (wave 1: DBA, security, architect; wave 2: QA and any re-review). Re-review of fixes before PR.
- Plan review: a read-only reviewer returned FAIL with eight fixable items. Re-review of the revision returned PASS-WITH-FIXES: items 1 to 7 resolved; item 8 (concurrency) partly open until the reviews were split into waves, which this revision records.
