---
type: VerificationRecord
title: ERP common submission ledger verification
description: Records what was executed and reviewed for the first increment of work item W-24. The increment is a submission ledger with capability-checked API or file connections that never write the ERP database, master mappings, existing-path anchors and ledger surveys, idempotent submissions whose unknown outcomes are resolved by lookup before any retry, revision-ordered observations with divergence and resolution, a status watch, and row-level document links with attributed totals kept apart from document totals. The record covers results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-226](https://linear.app/hybrid-billing/issue/HB-226) adds the Go package `modules/erpledger` from main `e094a6c`. Its non-test sources use the standard library and the primitive `decimal` only. One registry has an injected clock and one mutex, and keeps the following per workspace:

- **Connections.**
  - Transport is `api` or `file`; `database` is refused.
  - Capabilities are declared per connection. A missing capability is refused as `unsupported` and has no fallback.
  - File templates start unverified. A second human verifies them with evidence, and an unverified template blocks the file handoff.
  - Mappings are revisioned by customer, product and tax code.
- **Existing paths.**
  - Anchors record existing sales orders, contracts, prepaid documents and AP documents. They block re-billing across the whole workspace and across connections.
  - An anchor recorded over a live submission marks that submission diverged.
  - Ledger surveys hold a submission while the existing ledger is unknown.
- **Submissions.**
  - Single-use keys.
  - Sources are unique among live submissions of the same purpose, across connections and transports.
  - An order-based path follows its anchor.
  - Commands are checked against both state and capability.
  - `Withdraw` covers prepared submissions only.
- **Observations.**
  - Sources are webhook, poll and result file, applied in revision order. Stale reports change nothing.
  - Orphans never create submissions.
  - `absent` is accepted only from a poll or result file, and only for an `unknown` submission. Earlier absences are premature and move the submission to `unknown`.
  - A report that contradicts a terminal state marks the submission diverged. The divergence spreads to replacements, and diverged submissions stay live.
  - `Resolve` accepts the reported state, keeps the current one, or supersedes the submission. It never leaves two live holders of a source, and never clears a submission still covered by an anchor.
- **Status watch.**
  - Watched: sent, draft, unknown and diverged submissions, and posted ones for 30 days.
  - `Due` lists checks that are due. `Checked` counts retries and turns a timed-out send into `unknown`.
- **Documents and links.**
  - Document revisions replace rows.
  - Each link allocates part of a row to a submission line, bounded by the row and the line with matching signs.
  - Changed rows mark links stale, and stale links can be replaced.
  - `Reconcile` reports linked amounts per line, plus full and attributed totals per document.

# Commands and results

All runs were made in the worktree `/private/tmp/hb226` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/erpledger` | no output |
| `go vet ./modules/erpledger` | pass |
| `go test -race -count=10 ./modules/erpledger/` | pass (11.0 s) |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null` in the package | builds |
| `go test -race -count=1 ./...` | pass, 39 packages with tests |
| statement coverage of `modules/erpledger` | 100.0% |
| property test against a model (300 seeds × 80 steps) | pass; every state, divergence, stale, orphan, resolution and link outcome reached at least 20 times |
| mutation testing (`mutate226.py`, 871 mutants) | 794 killed, 62 did not build, 11 survived (all equivalent) |

After every step, the property test checks invariants that do not use production logic:

- no source has two live holders unless both are diverged;
- no anchored source is held by a live submission that is not diverged;
- the non-stale allocations of every row stay within the row, with the same sign;
- the allocations of every line stay within the line;
- each document's attributed total equals the sum of its non-stale links to live submissions.

The model's liveness and observation rules mirror the production rules. Results after `Resolve` are copied into the model, not predicted.

The first full run on the reviewed code left 31 survivors. Tests were added for per-component bounds under a tight decimal profile, status commands on diverged submissions, the last state after a premature absence, `keep` not touching the posting time, anchors that a path follows or a purpose does not block, a changed anchor under the same key, an older identical document revision, the row limit, partial tax and quantity links, and a scale-one display. A redundant per-link check was removed, because the cumulative bounds imply it. In the second full run 12 mutants survived. A test for another blocking anchor over an order-based path then killed one of them on a targeted rerun.

The 11 remaining survivors are equivalent:

- The input length guard (`64`/`65`, `>`/`>=`). No accepted decimal string is longer than 24 bytes.
- Extra month keys such as `"7"`. The length check already refuses them.
- `Round(1)` on the clock.
- The pair array size in the limit check.
- `||` instead of `&&` in reference registration. An empty or identical reference changes nothing.
- `||` in the document identity comparison. Documents are keyed by connection and reference.
- `<=` in a sort over unique keys.
- `accept || state differs` in `Resolve`. No submission that can be accepted has a last state equal to its current state.

The 62 mutants that did not build were rejected by the compiler or `go vet`.

# Acceptance

| Criterion | Evidence |
| --- | --- |
| AUD-DB-01: first reconciliation through file and result registration; unverified templates shown as such | `TestFileFirstReconciliationWithoutAPI`. The release gate itself is not built. |
| AUD-DB-03: existing orders, prepaid and AP documents; unknown ledger held | `TestExistingPathsBlockDoubleBilling`, `TestAnchorAfterPrepareMarksDiverged`, `TestSwitchFromAPIToFileNeverRebills` |
| AUD-DB-04: merged and split documents reconcile only attributed amounts | `TestMergedAndSplitDocuments` |
| AUD-DB-11: no database transport; unsupported commands refused without change | `TestDatabaseTransportAndUnsupportedCommands` |
| AUD-DB-12: missing and reordered webhooks recovered by polling; nothing recreated | `TestMissingAndReorderedWebhooksRecoveredByPolling` |
| HB-24-V: local save failure, manual edits, wrong absence, API to file switch, timeouts | `TestExternalSuccessWithLocalSaveFailure`, `TestWrongAbsenceAndManualEdit`, `TestTimeoutNeedsConfirmedAbsence`, `TestPrematureAbsenceMovesToUnknown`, `TestSwitchFromAPIToFileNeverRebills`. Queue, concurrency and call-rate controls are not built. |

# Reviews

- **Design review.** Three rounds by an independent read-only reviewer, folded in after each round:
  1. CONDITIONAL PASS: 6 high, 6 medium and 4 low findings.
  2. CONDITIONAL PASS: 4 medium and 3 low findings.
  3. CONDITIONAL PASS: 2 medium and 1 low finding.
- **Change review.** Two rounds by a separate independent read-only reviewer:
  1. FAIL: 5 medium and 3 low findings, plus 3 hypotheses. No double-billing path was found. Everything was fixed.
  2. PASS, with three low notes recorded in the plan.

# Provisional choices

These are mine, not the owner's, and the owner needs to confirm them:

- the kinds, states, transitions, capability names, intervals and limits;
- invoice and payable purposes only;
- whole-line anchors;
- `diverged` and human resolution, instead of following manual ERP edits automatically;
- a 30-day watch on posted submissions;
- `absent` only from `unknown`;
- an unverified template blocks the file handoff;
- a webhook rejection of a draft frees the sources;
- system connectors may register observations and documents.

# Not verified or not built

- Provider connectors (Odoo, SAP; W-25), the standard XLSX files (W-26), real ERP calls and provider evidence.
- The AUD-DB-01 release gate, and queues, concurrency and call-rate limits per entity and connection.
- Persistence, screens I02, I03, I04, O03, S06, B10, R01 and U03, and DBA and QA sign-off.
- Joining the recognition-purpose submissions of `modules/recognition` into this ledger (the integration layer).
