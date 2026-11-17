---
type: VerificationRecord
title: HYB-290 verification of the P05 release evidence API
description: Format, vet, unit and edge tests with coverage, race runs, cross-platform builds, mutation testing, a smoke run of the built binary, the design and change review outcomes and the planning cross-check for the P05 release evidence API over modules/releasetrace and modules/perfgate. Run on mac_mini on 2026-11-17 and 2026-11-18.
status: final
tags: [HYB-290, pre-launch, release-evidence, verification]
sources:
  - resource: plan.md
  - resource: ../HB-288/verification.md
  - resource: ../../reviews/HYB-291-planning-cross-check/report.md
---

# Scope

This record verifies [HYB-290](https://linear.app/hybrid-billing/issue/HYB-290) against its [plan](plan.md). Checks ran on mac_mini (macOS, darwin/arm64) with the Go toolchain from `go.mod` and `GOCACHE` in the session scratchpad, using only synthetic sample data. A delegated agent drafted the plan and wrote the code in the ticket worktree. Root re-ran the checks marked "root" on the final code.

# Results

| Check | Result | Run by |
| --- | --- | --- |
| `gofmt -l` on `apps/api/server` and `modules/releasetrace` | nothing listed | root |
| `go vet ./...` | clean | root |
| `go test ./...` | all packages ok | root |
| Coverage | `apps/api/server` 100.0%, `modules/releasetrace` 100.0%, `modules/perfgate` 100.0% (unchanged) | root |
| Cross builds of `./apps/api/cmd/hb-api` | linux/amd64, windows/amd64 and darwin/arm64 build | root |
| Smoke run of the built binary (fresh 0700 data dir, `--seed sample`, two administrators) | the read returns 200 at r21 with candidate `0.0.2`, release `not_connected`, both paths `designed` with the planned blockers, gate `ready: false` (`blocking`, `not_tested` ×2, `path_evidence`, `path_inactive`), readiness `ready: false` (`e2e` provisional, `daily_inflow` and `monthly_close` failing, `interrupt_resume` and `catch_up` not_run), 7 results and 28 proposals; the read contains no `"ready": true`, `customer_proven`, `"passed"` or runner identity; `retire_requirement` 200 `pending`; self-approval 422 `prerequisite_failed`; approval by the other administrator 200 `applied`; the requirement reads retired and the gate stays `ready: false` | root |
| `go test -race -count=40 -timeout 45m -run Evidence ./apps/api/server` | did not finish within 45 minutes at a load average of about 55 from unrelated workloads (`FAIL … 2700.508s`); only the last lines were kept, so no conclusion is drawn from it | root |
| `go test -race -count=40 -timeout 150m -run Evidence ./apps/api/server`, full log kept | ok (3826.880 s), 0 `DATA RACE` lines | root, code before the change review round 2 fix |
| `go test -race -count=5 -timeout 60m -run Evidence ./apps/api/server` and `go test ./...` | race ok (383.839 s), 0 `DATA RACE` lines; all packages ok | root, final code |
| Mutation testing of `evidence.go`, `evidenceload.go`, `evidencecmds.go`, `evidenceseed.go` and `modules/releasetrace/exports.go` | 394 mutants, 393 killed; the survivor (`evidence.go:509`, the replacement branch of the requirement read forced off) is now killed by a read assertion added to `TestEvidenceRetireRequirement`, shown by applying the mutant (the test fails) and restoring the file byte-identical | root |
| Targeted mutants of the round 2 L2 fix (`resultNotCounted`, `testOwners`, `notCounted`) | 6 applied: 5 killed; `built <= implemented` survives and is equivalent, because applied sequence numbers are unique, so a build and an implementation never share one. The error return in `resultNotCounted` is covered by the read fault test (`trace-requirement` 24 calls, each faulted to 503). | root |

The mutation run ran at a load average above 20 in parts. A mutant that times out counts as killed, so a few rows may be killed by time rather than by an assertion. An earlier run, before the change review fixes, left three survivors at the then lines `evidence.go:356, 381, 382` (a `Check` error branch and the customer-proof count); the final run kills that code.

No Go test file contains the substring "flo" + "at" (spelled apart here on purpose). The largest changed file is `evidence_api_test.go` at 931 lines; none exceeds 1,500.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only): round 1 CHANGES REQUIRED (High: the dropped `revoke_harness` command still appeared in the body; Medium: the common requirement offered a retirement that could never succeed; seven Lows on the no-candidate shape, a field named `passed` on uncounted results and the AST checks); round 2 PASS, with one Low and two Nits folded in as "Design review round 2 clarifications".
- **Change review** (independent reviewer, read-only): round 1 PASS for the code with no High or Medium. Lows: L1 this record was missing; L2 one uncounted result got two different reasons; L3 path columns without a candidate read as a real zero; L4 a failed `Check` became a blocker or a silently shorter replacement list instead of a read error. Under the owner's "faithful to planning" instruction, L2 to L4 were fixed in code rather than recorded. Round 2 CHANGES REQUIRED: L3, L4 and the new mutant-killing assertion PASS; L2 Medium (the store answered `no_implementation` whenever its own implementation entry was gone, even when the module had one, and a test asserted it); this record Medium (cross-check reasons described this ticket instead of each requirement, and AUD-DB-01 was overclaimed as covered). Fixes: `no_implementation` now comes only from the module (`Commit` empty in the requirement view, for test rows and for artifacts result rows); the store answers only `build_predates_implementation` or `not_counted_by_module`; the test asserts `not_counted_by_module` on both rows when the entries are gone; the table below was rewritten from the requirement text and AUD-DB-01 is deferred, here and in the plan. Round 3 PASS (Low: the test file line count in this record was stale, now corrected; Nits: `common_missing` is asserted only in the module tests, now stated in the table; after lost store entries the designed requirement's test row reads `no_implementation` and its artifacts row `not_counted_by_module`, both true).

No review is owner acceptance.

# Deviations from the plan

- Requirements are listed in registration order, which matches their target numbers; the plan says "key order" (change review N2).
- The AST checks catch direct `s.trace.X` and `s.perf.X` selectors only; an alias would not be caught. No alias exists.

# Residuals

The plan's "Not connected" and "Residuals for the owner" sections stand unchanged. In short:
- Support-path activation, support decisions, customer proof intake, harness revocation and binding, evidence files, result recording outside the seed, budget approval, real load measurement, the real planning trace and a durable decision ledger are not connected.
- Every command uses the installation administration grant instead of P05's separate permissions.
- No search, filters, cursor paging or fixed snapshot; lists are capped with `total` and `more` (tracked in HYB-304 and HYB-305).
- There is no profile for concurrent collect, close and delete load (SCALE-11).
- Registers, store and journal reset on restart.
- The P05 screen binding follows in its own ticket; `docs/ui/screen-binding-register.md` catch-up is HYB-295.
- All release gates and SCALE-01 to SCALE-12 remain NOT_RUN at system level.

# Planning cross-check

Source: `docs/ui/screens/P05/spec.md` and `docs/planning/v1.16/requirements.json`, as amended in the plan (Planning cross-check amendments). Status meanings follow the [HYB-291 report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-04 | deferred | Provider onboarding and readiness: support by provider, contract, account role and data kind is shown with evidence. Built: support paths with their requirements, states and blockers; nothing can read active. Not built: activation, the activation review request, support decisions as records, and the operational read tests after a permission is revoked. | plan.md Not connected |
| HB-19 | deferred | Accepting RI/SP/CUD commitment results. P05's part is DB-09, the commitment event matrix. Built: nothing for commitments in this ticket. Not built: the event matrix per commitment profile (AUD-DB-09 below). | plan.md Planning cross-check amendments |
| HB-24 | deferred | ERP common submission ledger. P05's part is DB-01: the release gate must not drop common features. Built: the gate's common-requirement blockers over sample requirements (see AUD-DB-01; `common_missing` is module-tested only). Not built: the submission ledger itself and its tests in the real trace. | plan.md Not connected |
| HB-25 | deferred | Odoo and SAP proof; a mock ERP pass is not compatibility certification. Built: paths cannot read active and no requirement can read customer-proven, so a sample or mock result is never shown as proof. Not built: ERP paths, product and version proof, and customer proof intake. | plan.md Not connected |
| HB-26 | deferred | Standard XLSX manual handoff (DB-01). Built: the sample common XLSX requirement cannot be retired without a replacement, and templates are listed only for problem-free paths. Not built: the XLSX files, result registration and their tests in the real trace. | plan.md Not connected |
| HB-38 | deferred | Simulator and independent expected values. Built: the seed is synthetic and every seeded record reads `origin: sample_seed`. Not built: scenario manifests with seed, clock and independent expected values, and demo reset apart from operational data (P06). | plan.md Sample seed |
| HB-39 | deferred | Large-scale processing and performance budgets; not run is never shown as passed. Built: load profiles, hardware, runs and readiness from `modules/perfgate`, with failing, provisional and not-run verdicts and never passed; a test without a result reads `not_run_reason: no_result_recorded`. Not built: budget approval, real measurement at 1M+ daily and 30M+ monthly, the concurrent collect, close and delete profile (SCALE-11), and system-level gate runs. | plan.md Not connected |
| HB-40 | deferred | Reuse licences and commercial rights. Built: nothing in this ticket. Not built: release evidence for distribution notices and licence boundaries. | none |
| HB-41 | deferred | Release trace and actual go-live: design, implementation, test pass and customer proof are separate states. Built: separate columns, and no reachable state reads customer-proven or a released path. Not built: the real planning trace import, owners, API and screen links per requirement, and customer proof intake. | plan.md Not connected |
| AUD-DB-01 | deferred | Standard XLSX common release scope. Built, over sample requirements only: the gate's `common_missing` problem (passed through from the module and asserted only in the module tests; the sample candidate includes the common requirement), the `common_needs_replacement` refusal, the `no_replacement` blocker (`TestEvidenceReplacements`; the command refusal in `TestEvidenceRetireRequirement`), and templates listed only for problem-free paths. Not built: first reconciliation through standard XLSX and result registration without an API, ERP upload template verification state, and the real trace. | plan.md Planning cross-check amendments |
| AUD-DB-09 | not connected | No event test matrix per commitment profile exists in the seed or the modules. | plan.md Planning cross-check amendments |
