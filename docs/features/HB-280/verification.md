---
type: VerificationRecord
title: HB-280 verification of the O03 contract read and lifecycle API
description: Tests, property test, coverage, race, cross builds, mutation results and review outcomes for the contract list and detail reads and the suspend, resume, renew, end, withdraw, evidence, settle and reopen commands. Run on mac_mini on 2026-11-17; the test clock is fixed at 2026-11-16.
status: final
---

# Scope

This record verifies [HB-280](https://linear.app/hybrid-billing/issue/HB-280) against its [plan](plan.md). Every check was run on mac_mini (macOS) with Go 1.25.7 (darwin/arm64), using synthetic data only.

# Results

| Check | Result |
| --- | --- |
| `gofmt -l` and `go vet ./...` (apps/api) | clean |
| `go test ./server` | pass, 100.0% statement coverage of the package; `contracts.go` and `contractseed.go` are each 100% |
| `go test -race -count=40 -run Contract ./server` | pass |
| `go test -race ./...` (apps/api) | `server` passes. `cmd/hb-api` `TestRun` failed once when run beside the server package and passes on its own; the same test fails 5 of 40 race runs on main (`1ad3fef`) without this change, so it is a pre-existing timing flake outside this ticket |
| Cross builds `linux/amd64`, `windows/amd64`, `darwin/arm64`, `linux/arm64` | build |
| Mutation testing of `contracts.go` and `contractseed.go` (292 mutants on the final code) | 273 killed, 19 survived. Five survivors (cancel flag on settled and write-limit withdraw hints, the non-reserved settle and secured-evidence limit hints, and the `reopen` reason requirement) were killed by added tests, each re-run on its own. The other 14 are equivalent, listed below |

**Equivalent survivors**
- Seven `Required` flags on command fields (`from`, `event`, `kind`, `as_of`) and the reopen mapping: the handler validates the same values again and returns the same code.
- The settle-hint fence: replacing the reopen fence with the settle record, or dropping the batch fence. A secured record between a batch and a later end cannot cover the new end day, because secured coverage must be on or before today and a new end must be after the withdrawn one; a settle record always precedes a reopen.
- The evidence-kind check in the settle hint: the latest evidence cannot be an unresolved missing record when no gap is open.
- Two date-parse guards whose failure yields a zero day that the next check refuses.
- A fault-hook error value and the sequence-read error (only reachable through the fault).
- `<=` in place of `<` in a sort over unique IDs.

**Property test.** Twelve seeded runs of 30 random steps over the five sample contracts, with random clock advances up to 20 days. Each step reads the detail, picks a command and sends it with the hinted earliest day (or today). It asserts that the command succeeds exactly when the hint says allowed, that every list row's state equals the module's `StateOn`, and that no settled contract has an open gap.

**What the API tests cover**
- **Reads.** List shape, order, profile parties, state and lifecycle revision on a past day, invalid `on`; detail period defaults and bounds, terms with components, events with withdraw hints, segments, evidence with write IDs and owners, records newest first with actor and reason, `earlier_records` paging, every action hint, 404 for unknown, malformed and other-workspace IDs.
- **Lifecycle.** Suspend, resume on the same day refused, resume, withdraw order rules, withdraw of start and unknown events, cancel reason rules (none needed, needed, wrong, `recorded_in_error`, `contract_continues`), open gaps kept after withdrawing an end, withdraw refused once the event is in the past, past and malformed and beyond-horizon days, today allowed, ended and settled contracts refused.
- **Settlement.** Future end gives `pending_final`; secured evidence refused for future coverage and before the end day; gaps recorded before the end and resolved by write ID; settle refused without evidence, with short coverage, before the end day, with a future or malformed `as_of`, with open gaps, and after reopen until new evidence; reopen; missing evidence refused on a settled contract; resolving an already resolved gap refused.
- **Limits.** Write limit with the reserve for suspend, end, missing evidence and reopen; hints report `write_limit` and `horizon`.
- **Access and failure.** Member and anonymous refused for reads and commands; other-workspace contracts invisible and unresolvable; rationale length; stale revision gives `revision_conflict`; idempotent replay writes once; each fault hook; a billing profile read failure and an empty lifecycle fail closed with `dependency_unavailable`; seed failure paths.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only, DBA focus):
  - Round 1: CHANGES REQUIRED with one High (secured evidence for days not yet passed after a future end), five Mediums (gap write IDs not shown, inexact hints, withdrawing a start cancels the contract, error fields dropped for non-`invalid_request` codes, settled beyond ADM06 not recorded) and Lows. All were applied to the plan before implementation.
- **Change review:**
  - Round 1: CHANGES REQUIRED with one Medium (missing evidence accepted on settled contracts) and five Lows (hints ignored the write limit and the horizon, a conservative hint with a held future end, an unguarded empty lifecycle, an ignored billing profile failure) plus test gaps. All were fixed or documented.
  - Round 2: PASS. Its three optional items (aggregate withdraw wording, the limits copy in one test, the fail-closed note) were applied.

No review is owner acceptance.

# Residuals

These are listed in the plan:
- One administrator with a reason performs lifecycle, evidence, settle and reopen commands; ADM06 separates operator and reviewer, and HB-274 requires a second administrator for relationship changes. Owner acceptance needed.
- `settled` means final evidence is settled; the final amount decision, money, proration, publication and ERP impact are not computed.
- No contract creation or `change` events; UTC only; no access-removal plan or late-source window policy.
- Rationale is bounded free text and references are ID-shaped; neither can prove it holds no customer data.
- All state is in memory; nothing persists across restarts.
- Seed dates are absolute; sample-contract-05 starts on 2027-01-01.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check.

## Requirement trace

Source: `docs/ui/screens/O03/spec.md:121-155` (HB-280 is the API, HB-281 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-01 | deferred | Covered (O03): contracts are linked to billing profiles and parties. Not built: the three operating profiles on one core, internal settlement without an external invoice, and blocking ineligible cross-entity or cross-currency sums. | HB-280/plan.md |
| HB-05 | not connected | No source connectors. | HB-280/plan.md:125-133 |
| HB-08 | deferred | Final evidence can be settled; durable evidence storage waits for the database decision. | HB-280/plan.md:125-133 |
| HB-10 | not connected | HawkEye results are not connected. | HB-280/plan.md:125-133 |
| HB-12 | not connected | On-premises cost pools are not connected. | HB-280/plan.md:125-133 |
| HB-15 | deferred | Lifecycle and state on a day are built; contract creation and change events are out of scope. | HB-280/plan.md:125-133 |
| HB-16 | not connected | Money is not computed. | HB-280/plan.md:125-133 |
| HB-18 | not connected | Exchange rates and rounding are not computed. | HB-280/plan.md:125-133 |
| HB-21 | not connected | `final_amount_decision: not_computed`. | HB-280/plan.md:125-133 |
| HB-22 | not connected | Prepaid credits are not connected. | HB-280/plan.md:125-133 |
| HB-23 | deferred | Lifecycle and settlement are single-administrator; two-person review is an owner residual. | HB-280/plan.md:125-133 |
| HB-24 | not connected | No ERP impact. | HB-280/plan.md:125-133 |
| HB-25 | not connected | No ERP adapters. | HB-280/plan.md:125-133 |
| HB-28 | not connected | Documents are not connected. | HB-281/verification.md Residuals |
| HB-32 | not connected | The assistant is not connected. | HB-280/plan.md:125-133 |
| HB-37 | deferred | Settle and reopen are built; access-removal plan and late-source window policy are not. | HB-280/plan.md:125-133 |
| AUD-AR-01 | not connected | No sale computation, so direct and resale paths are not exercised. | HB-280/plan.md:125-133 |
| AUD-AR-10 | not connected | No fixed-fee or SLA credit computation. | HB-280/plan.md:125-133 |
| AUD-DB-03 | not connected | No ERP submission. | HB-280/plan.md:125-133 |
| AUD-QA-01 | unclassified – owner review | Payment account and recipient changes are neither built nor recorded. | this section |
| AUD-QA-02 | deferred | Covered: contract and settlement states are separate, ending is never shown as settled, pending_final with secured or missing evidence and reopening are built. Not connected: the emergency source-permission revocation path that revokes immediately and records the evidence gap (O03/spec.md:159-161). | HB-280/plan.md |
