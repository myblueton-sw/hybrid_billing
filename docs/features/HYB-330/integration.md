---
type: VerificationRecord
title: HYB-330 consolidated diagnostic verification
description: Records reconciliation of both published HYB-330 candidates with merged HYB-327 and verification of the resulting frontend.
status: stable
sources:
  - resource: ./implementation.md
  - resource: ./integration-evidence.json
  - resource: ./verification.md
  - resource: ./prior-candidate-verification.md
---

# Consolidation

Ticket HYB-330; owner root; machine mac_mini. PR243 integrates candidate f42d05c and the existing PR238 candidate c323abb without rewriting history. The first merge a459c55 preserves main d7b6a3d (HYB-327). The second merge retains PR238's meaningful invariants while using one shared runtime implementation and renderer. No unrelated open PR is incorporated.

The shared `Refusals<T>` store keeps the original refusal values, deep-copies nested references and clears only the exact previous action/target on a fresh attempt. P02 grouped resume/redelivery withdrawals use the canonical request target. Monotonic scope generations protect reads across A → B → A, and synchronous cleanup invalidates old continuations. Terminal feedback and ordinary notices retain their associated references. Response tokens have a common 128-character bound; local confirmation cancellation explicitly identifies the preceding challenge. Superseded duplicate renderer/CSS are removed; their original history and distinct verification record remain available.

HYB-327's strict parsers and M01/M02/P02/P05 field bindings are preserved. Error metadata never authorizes retry, changes permissions, confirms an outcome or changes financial semantics.

# Executed checks

| Check | Actual result |
| --- | --- |
| Complete console suite after consolidation | PASS, 911 tests, zero failures |
| TypeScript and production build | PASS; existing bundle-size warning remains |
| Final browser entry-point QA | PASS, 404 assertions, zero failures |
| Immutable foundation/source-style checks | PASS, eight checks |
| Consolidated-source targeted mutations | PASS, all twelve killed; source bytes restored and SHA-checked after each |
| Documentation format and local links/changed-source size | PASS, two format tests; manual links/size check |
| Grouped request diagnostic assertion on unchanged main653adfe | FAIL as expected: two missing-reference assertions; three setup checks pass |
| Fresh-attempt exact marker reset on candidate f42d05c | FAIL as expected; final source passes |
| Nested reference isolation with old flat copy | FAIL as expected; final shared store tests pass |

Browser QA uses the real local API with synthetic activation/sign-in, actual grouped request setup and intercepted command-error replies. It covers P01–P05 reads, terminal feedback, missing/malformed metadata, cancellation, independent open forms and retained rows, late old-view results, reload/remount, grouped request targets, and neutral layout/bidi/token containment in EN/KO/AR at 1440/1280/1024/390. The final runner waits for local fonts and asserts Plex readiness. No unexpected JavaScript exception was observed. Expected intercepted HTTP failures are part of the scenarios, not a claim of zero network failures.

The initial candidate's ten killed mutations and 310 browser checks are recorded in its separate verification record. They are not relabelled as consolidated-candidate executions. The final evidence manifest binds actual source and selected local logs/renders by SHA-256. Raw images/logs are not committed. Go source does not change; the synthetic API binary comes from merged653adfe. No new whole-Go, scale or race pass is claimed.

# Independent review

Reviewer `/root/hyb310_review`: independent consolidated-source PASS, no actionable integration blocker. Inspected working source and relevant differences against a459c55 and c323abb, including shared stores, deep copies, exact grouped targets, reset behavior, monotonic read guards, synchronous command guards, terminal/notice association, cancellation provenance and preserved HYB-327 parsers. Reviewed adapted tests and clean whitespace/unresolved paths; did not execute suites or builds. Root-run checks remain separately attributed.

Reviewer `/root/hyb310_copy_render`: bounded consolidated TW/render PASS, no blocker. Inspected grouped-channels EN1440/KO390, grouped-events AR390, P04-open AR390 and cancellation KO390. Checked Action request references wording, canonical request targets, retained-row attribution, form references/cancellation provenance, neutral available-width layout and narrow RTL presentation. Read the root-run 404-PASS log including Plex readiness and independently matched all52 source hashes; did not execute the browser suite. Candidate-specific verification provenance remains distinct. Native-language/practitioner and full assistive-technology acceptance are not inferred.

# Limits and integration boundary

The Claude CLI ui-designer attempt returned its weekly quota limit (reset Nov24, Asia/Seoul); no successful Claude review is claimed. Native-language/practitioner, full assistive-technology, Linux/Windows execution and owner FIX remain UNVERIFIED. Remote CI workflows/protection are unconfigured, not a CI PASS. No deployment, real account/customer data, payment or external dispatch occurred.

PR243 merge, remote main ancestry read-back, PR238 disposition and Linear Done/machine release follow this pre-merge record and are recorded in the ticket after actual success. The full implementation-plan goal remains active.
