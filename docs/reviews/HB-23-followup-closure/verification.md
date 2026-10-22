---
type: Verification Record
title: HB-23 follow-up and remaining finding closure verification
description: Records finding dispositions, changed artifacts, local companion amendments, independent reviews, root checks and residuals for HB-23.
status: draft
---

# HB-23 verification

Tracking: Linear ticket [HB-23](https://linear.app/hybrid-billing/issue/HB-23). Owner: Seung Woo Park. Machine: `mac_mini`. Branch `docs/HB-23-followup-closure` from main `35e2d4c`. Date: 2026-10-22. Plan: [brief](brief.md). Planning gate applies: documentation only; no product development; planning completion is not declared.

## Dispositions

"Remediated" means the Git documents now carry the required relationship, guard or binding. It never means runtime behavior was verified. "Partial" names the residual.

| Finding | Disposition | Location | Residual |
| --- | --- | --- | --- |
| FA-01 approval subject | Remediated | [logical schema](../../architecture/data-ownership/logical-data-schema.md) ApprovalSubject and purpose table; [monthly schema](../../architecture/data-ownership/monthly-settlement-schema.md) final settlement row | None in Git documents |
| FA-02 credit issuer identity | Remediated | logical schema CreditAccount; cross-issuer transfer is a conditional derived proposal | Transfer support is an open decision |
| FA-03 positive FX | Remediated | logical schema FXEvidence; monthly FX stage; [inputs, pricing and ERP](../../contracts/billing/inputs-pricing-and-erp.md) PC-01; [source pricing](../../contracts/billing/source-pricing-and-cost-obligations.md) SP-03 | None in Git documents |
| FA-04 adjustment lifecycle | Partial | [planning actions](../../contracts/commands/planning-actions.md) adjustment and settlement families; [B09](../../ui/screens/B09/spec.md), [U02](../../ui/screens/U02/spec.md), [B11](../../ui/screens/B11/spec.md), [S06](../../ui/screens/S06/spec.md); local acceptance trace MS-05, MS-09 | Local B04 and B08 text conflicts or lacks bindings ([register](../../ui/screen-binding-register.md)); MS-05 and MS-09 have no work-task mapping yet; HB-24 |
| FA-05 FX components on screen | Partial | Registry FX component validation; B09 fields | Local B05 `spec.md:50` still applies one sign rule; HB-24 |
| FA-06 onboarding operations | Remediated | [source relationship bindings](../../architecture/data-ownership/source-relationship-bindings.md) FA-06 | Commands and screens not added; HB-24 |
| FA-07 plugin lifecycle | Remediated | Bindings FA-07 | Same as FA-06 |
| FA-08 result return and events | Remediated | Bindings FA-08 | Same as FA-06 |
| FA-09 enrollment and auth policy | Remediated | Bindings FA-09 | Same as FA-06 |
| FA-10 customer transfer | Remediated | Bindings FA-10 | Same as FA-06 |
| FA-11 commitment, opening balance, Marketplace, final work, inquiry | Remediated | Bindings FA-11 | Same as FA-06; conditional scope stays conditional |
| FA-12 ERP, issuance, artifact, delivery | Remediated | Bindings FA-12 | Same as FA-06; FR-06 remainder under HB-24 |
| FA-13 operational notification rules | Partial | Registry notification rule family; U02 | Local P02, M05 and U03 lack bindings; HB-24 |
| FA-14 portable canonical baseline | Deferred | — | [HB-24](https://linear.app/hybrid-billing/issue/HB-24) |
| SR-01 policy-owning Party | Remediated | Bindings SR-01; logical schema PolicyRevision pointer | Same as FA-06 |
| SR-02 supplier invoice bridge | Remediated | Bindings SR-02; monthly and evidence schema pointers | Same as FA-06 |
| SR-03 demo entry profile | Partial | Registry demo family | Local P06 text unchanged; HB-24 |
| HB-22 follow-up 1 | Remediated (local) | Local `docs/planning/v1.16/architecture.md` external-execution paragraph | Local companion, outside Git |
| HB-22 follow-up 2 | Remediated | Registry ERP route family, with dispatch-lock guard | None |
| HB-22 follow-up 3 | Partial | Tracked S02, S06, I04, B11 | 18 local screens in the register; HB-24 |
| HB-22 FR statement | Corrected | [HB-22 verification](../HB-22-source-omission/verification.md) follow-up 4 | FR-06 open under HB-24 |

Acceptance cases added: H23-S01 to H23-S49 (schema and contracts) and H23-C01 to H23-C23 (commands). All NOT_RUN.

## Local companion amendments

These files are authorized local companions outside Git; their edits are not visible in the PR. Before copies are kept under `.local/planning/hb23-before/` (ignored).

| File | Before SHA-256 | After SHA-256 | Change |
| --- | --- | --- | --- |
| `docs/planning/v1.16/architecture.md` | `fd8ff332254d7cbdfa999e10fb6ec60e068b3448a6540556d8d93379dca5a030` | `83f38765bc91a816d8355056fdb9bb9ac5a2d7e13afb0b3968b8575a089474c3` | Automatic posting default aligned with EI-02 |
| `docs/planning/v1.16/acceptance-trace.json` | `9e5dc2524b330b897cee3a139dfd9b5838bc698d941dd470d319c8f532577788` | `5a471d36c65518cb56e92aaf773489f683a5b41619d1d465b1deedc9610eed2b` | Added MS-05 and MS-09 to `added_tests` (NOT_RUN); their `source` names Linear ticket HB-23 to distinguish it from requirement group HB-23 |

## Independent review

All reviewers were separate read-only agents; writers W1 and W2 were separate implementers; root integrated.

| Review | Result |
| --- | --- |
| Plan | FAIL with eight fixable items; re-review PASS-WITH-FIXES; remaining item applied |
| Security (W1) | PASS-WITH-FIXES: two BLOCKING (handoff link guard, auth-policy weakening separation), eight NON-BLOCKING; fixes applied |
| DBA (W1) | PASS-WITH-FIXES: seven NON-BLOCKING; fixes applied |
| Architect (W1, W2) | PASS-WITH-FIXES: two BLOCKING (dispatch lock before manual assignment, duplicate screen paths), six NON-BLOCKING; W2 fixes re-reviewed RESOLVED with two minor wording items fixed by root |
| Security re-review | PASS: all ten items resolved; three NON-BLOCKING clarifications applied by root (installation-only storage holds no customer data, transfer ceiling carried across revisions, no self-approval by any approver) |
| DBA re-review | PASS: all seven items resolved; H23-S05 arithmetic reproduced; credit kind and currency added to H23-S05 by root |
| QA | PASS-WITH-FIXES: five NON-BLOCKING; root applied the FA-04 work-task residual, HB-22 wording, H23-C23 outcome, case-scope sentence and P02 citation wording |

Root's final clarifications after the re-reviews were checked by root only and were not re-reviewed.

## Root checks

- Staged diff: 19 files, all within the brief's allowed files; `git diff --cached --check` clean.
- OKF frontmatter with `type` on every changed Markdown file; balanced code fences.
- Every local link added by HB-23 resolves to a tracked file. Pre-existing links to local companions (for example `planning-gate.md`, local screen specifications) remain unresolved in a fresh clone; HB-24 scope.
- Case IDs: 72 table definitions (H23-S01 to S49, H23-C01 to C23), no duplicates.
- Credential pattern scan of changed files: no matches.
- Local `acceptance-trace.json` parses as JSON.
- Runtime, database, provider, ERP and rendered UI behavior: NOT_RUN.
