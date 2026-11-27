---
type: ImplementationRecord
title: HYB-304 frozen selection and bounded result checkpoint
description: Records implemented server/client foundations, actual verification and the remaining M05 consumer migration.
status: stable
sources:
  - resource: ./selection-integration.md
  - resource: ./selection-review.md
  - resource: ./session-ui-integration.md
  - resource: ./selection-server-evidence.json
---

# Final status

The remaining M05 consumer migration listed below was completed later in HYB-304 ([session UI integration](session-ui-integration.md)). HYB-304 closed on 2026-11-27 ([closing audit](closing-audit.md)).

# Frozen checkpoint

This record and selection-server-evidence.json describe commit 5fbd7791127d7b369a03a5f11baaf33a0d3ddc1e. Source hashes are verified against that checkpoint, not later working-tree changes. Subsequent bounded /me and UI integration is recorded separately.

# Implemented behavior

On top of foundation 79856aed, M05 now has server-owned resumable selection preparation, immutable bounded preview, reference/digest/count acceptance and same-reference replay. The actual canonical owner/epoch/workspace/caller scope controls selection reads and changes. Accepted operation recovery permits another current session of the same canonical owner; unconfirmed intent remains caller-bound with null outcomes. Result/not-included arrays have dedicated bounded pages. Summary counts are updated with construction/cleanup transitions and never traverse retained results at read time.

Legacy and new termination paths share global record/reference accounting. Admission reserves frozen/results/held/retry capacity before effects. Expired unaccepted selections release capacity. Active cleanup survives visible retention; settlement/sweep releases retry references exactly once. Global pressure may evict settled oldest operations; equal completion times use monotonic operation sequence. Eviction removes selection, operation and owner-list indexes together. Selection preparation is explicitly a read-only POST policy row; termination still requires actual step-up. The central session-ending source guard remains enforced.

Strict OpenAPI paths and typed console API methods are implemented. Four client contract tests check bounded query parsing, building unknown counts, unconfirmed outcomes and immutable submission payload. Four model tests cover query navigation/precision and action locking rules. These helpers are not yet rendered consumers.

# Verification provenance

All executions below used synthetic fixtures on macOS arm64, Go 1.25.7, Node 26.7.0 and npm 11.19.0. No running user service was restarted. Generated logs/profiles remain outside Git; the evidence manifest stores hashes and sizes, not credentials. Temporary paths are not durable artifact storage.

- Initial full regression failed due to new 409 error metadata assumptions, the central termination guard and missing explicit POST policy enumeration. The corrections preserve existing assertions and add the new route/error entries; the focused four guard/policy checks passed 3.558s. An actual stale-auth scenario proves preparation does not authorize termination: end is refused before effects, then succeeds after reauthentication.
- After those corrections, `go test -short -coverprofile=… ./apps/api/server ./apps/api/cmd/hb-api` passed: server 163.090s, 98.3% statements; CLI 6.770s, 89.8%. This run preceded the later equal-time eviction tie correction and supplemental tests. It is not mislabelled as a full-suite run after that change.
- The final initial operation/selection race run passed 7.485s. Targeted regression/race runs after the tie correction are recorded in the evidence manifest. Darwin vet and Linux/Windows arm64 static vet passed before that one-line tie change; these are not Linux/Windows runtime executions.
- Console suite progressed from 917 to 920 passing tests as models were added; the final registered suite including the control-character cases is recorded separately in the manifest. TypeScript checks passed. No UI build/render acceptance is inferred from these API/model tests.
- Six private-cache guard mutations initially killed five and left the query-binding mutant alive: separate row-position checks masked its cross-view assertion. A page-size cross-use case then killed that mutant in a separate one-mutant rerun. Every mutated file was byte/SHA-restored, and both disposable caches were removed. Do not report six mutants rerun after strengthening.
- Additional HTTP tests cover complete immutable preview pagination/expiry, malformed/unauthorized reads and settled-record pressure. The first pressure fixture inadvertently hit the unrelated per-user session limit; correcting its configured capacity exposed the real equal-time eviction defect. The corrected pressure/paging/negative tests passed 0.818s. Independent source re-review passed the tie correction.

New behavior regressions reproduced actual missing routes/body support, review findings, summary traversal and overlapping schema shapes before correction. The new UI model tests initially lacked their module; that import failure is explicitly not an executable semantic regression. No critical-branch 100% coverage or complete rendered behavior is claimed; uncovered paths remain part of increment verification.

# Remaining completion boundary

`/me` still downloads full session populations and expanded recent operations. The current rendered M05 screen still consumes those fields; migrate it to the dedicated bounded APIs, wire navigation and retained uncertain-result recovery, then reconcile strict account types/fixtures and API declarations. Complete date validation, late-response and scope-loss tests, bounded detail/operation UI, EN/KO/AR at 1440/1280/1024/390, keyboard/focus, local-font and neutral-border checks, final review and QA. The root-authored UI proposal records external pattern evidence and the actual failed Claude CLI usage-limit call; it is not a Claude design or owner acceptance.

This checkpoint is committed/pushed only after actual staged inspection. No PR/merge or complete M05/UI-04/parent ticket result is established by this document. Existing user authorization permits the eventual reviewed merge; no new approval prompt is required. M04/M03/M02 follow the approved ticket sequence after M05. Parent HYB-304 remains In Progress with machine mac_mini claimed.
