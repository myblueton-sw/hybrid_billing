---
type: VerificationRecord
title: Independent synthetic ERP result evidence and resumable internal application
description: Private authenticated service helpers retain reported discrepancies and reconcile owned multi-step ledger effects.
status: draft
---

# Implemented boundary

[HYB-341](https://linear.app/myblueton/issue/HYB-341), machine:mac_mini, follows the independently reviewed [ingestion plan](result-ingestion-plan.md). Root owns implementation. This checkpoint adds private service helpers to the existing financial ERP adapter. It does not yet expose result commands, protected history/context or result screen controls, and does not complete W-23.

Original UTF-8 JSON and parsed packet have separate hashes. The exact-case decoder rejects duplicate/unknown/null fields, malformed Unicode, noninteger revisions and extra JSON; portable revision range is 1..2147483647. Retained service-owned preparation and document evidence bind connection, currency, customer and period. Canonical decimal values, units, product mappings and one-to-one line coverage determine the historical comparison. Reported values come from the supplied packet. Well-formed discrepancies can be proposed and independently confirmed, but cannot be applied.

Every helper uses the actual human session and dedicated result permission. Confirmation rechecks the original proposer subject. Application additionally rechecks current proposer/confirmer subject rights, current charge approval, confirmed document, connection/mapping/survey facts and the highest confirmed revision. Revoked confirmations continue to contribute to that watermark. A newer confirmation can strand an older unknown application; ownership is retained, never silently released. Revocation and protected historical inspection remain available when producers are stale or the service is held.

Before a write, application reserves immutable intent, actor, external identity and preparation identity after bounded preflight. It refuses adoption of pre-existing unowned documents, including identical content. Document and Link are separate actual registry operations. An interruption preserves unknown ownership. Recovery validates the exact document, author and all existing allocations, then resumes only missing intended links. Completion requires one coherent ResultOf snapshot, exact preparation/document/products/links, per-line matched amounts and complete attribution. It records a digest receipt while leaving the submission prepared, with no external command or financial approval consumption.

# Independent correction

refusal_review required explicit units, portable revision bounds, unique confirmed revisions and a monotonic watermark during design. Root adopted these before implementation. Source review then found fixed-scale reconciliation values being compared with canonical input strings; root added lossless retained-scale formatting. The exact financialerpresultpacket.go decimal import was added to the existing file-specific composition allowance. The reviewer passed both corrections and found no further application-integrity blocker in that read. It did not execute tests in that first source pass.

refusal_review subsequently passed the private core and the three corrected boundary tests, inspected the final 12.797s race log and executed no tests itself. Production code was unchanged after the full regression; only the named test assertions were strengthened.

platform_refusals independently passed terminology and evidence scope after reading the ingestion plan, verification record, core source, test assertions and focused/race/vet logs. It did not execute tests or inspect new renders. Root separately observed exit 0 for the full 83-package run.

# Actual checks

| Check | Result | Evidence |
| --- | --- | --- |
| Focused actual-ledger core and source rules | Passed, 3.604s | /private/tmp/hyb341-result-core-focused.log |
| Focused server race | Passed, 14.266s | /private/tmp/hyb341-result-core-race.log |
| Final focused race after stronger boundary assertions | Passed, 12.797s | /private/tmp/hyb341-result-core-final-race.log |
| Server vet | Passed | /private/tmp/hyb341-result-core-vet.log |
| Full Go short regression | Passed all 83 packages; server 152.056s | /private/tmp/hyb341-result-core-full.log |

The tests use actual authenticated users, independently governed result grants and the actual ERP registry. They cover USD15 net/USD1 tax/quantity5, stable receipt replay, interruption after Document and each of two Links, immutable returned packet/preparation data, mismatch categories including an equal-total row swap, conflicting revision confirmation, newer-revision revocation without watermark rollback, permission loss during recovery, changed document/mapping/hold, foreign author/link/revision, unavailable document lookup, historical proposal after document revocation, held revocation and eight concurrent applications.

The initial focused test failed because the new fixture incorrectly reported net rows 10 and 5 while the existing source reports 10.25 and 4.75. Root corrected those explicit test inputs after tracing candidateseed.go. Production mismatch rejection was unchanged. The earlier compile-only command reported no tests to run; it is not behavioral evidence. Final independent review found three tests refusing too early to establish their intended boundaries. Root strengthened them with a proposer holding confirmation rights (SelfApprovalDenied), a different actor holding application rights (owned-operation EvidenceStale), and revocation of only the application console ceiling while human rights remain (PermissionDenied). Production behavior was unchanged.

# Remaining work

Public command envelopes, bounded protected projections/history/context, impact enumeration and HTTP failure recovery are next. Rendered result controls, independent Claude UI review and EN/KO/AR journey verification remain outstanding. Replacement revisions require their own reviewed protocol. StageERPFinal, issuance and delivery remain incomplete. No external ERP, posting, merge, deployment, production persistence, native-language certification or owner FIX occurred. This checkpoint changed no frontend source and claims no new rendered verification.
