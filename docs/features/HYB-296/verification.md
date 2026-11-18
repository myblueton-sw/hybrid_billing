---
type: VerificationRecord
title: HYB-296 verification of the O02 activation hold
description: Formatting, vet, Go tests with race and coverage, console type check, unit and render tests, build, browser QA in EN, KO and AR at 1440, 1280 and 390, P01 to P05 regressions and mutation results for holding O02 activation until amount, publication and ERP impact evidence exists. Run on mac_mini on 2026-11-18 with synthetic sample data only.
status: final
sources:
  - resource: plan.md
  - resource: ../HB-274/plan.md
  - resource: ../HB-275/plan.md
---

# Scope

This record verifies [HYB-296](https://linear.app/hybrid-billing/issue/HYB-296) against its [plan](plan.md). Every check ran on mac_mini (macOS, Node 26, Go toolchain from the repository, headless Chromium through Playwright). The API binary was built from this branch with the sample seed. No real account, customer data or credential was used. Results are as observed; no check is reported that was not run.

# Planning cross-check

| Source | Requirement | Where it is met | Check |
| --- | --- | --- | --- |
| Owner decision 2026-11-17 (HYB-296) | Step 1: hold activation with a truthful blocked state until impact evidence exists; step 2 (phase 6) adds impact calculation and the B09 approval request | `relationship.activate` refuses `prerequisite_failed` without evidence; the read reports `impact_not_computed`; the console shows the held block and no activate button | `TestActivationHeldWithoutImpactEvidence`, render tests, browser runner |
| O02 spec line 33 | Progress: target → impact → approval → schedule or apply | Four steps Target, Impact, Approval, Apply; Impact is derived from the row `impact` | model test "four steps …", render test "progress has four named steps …", browser `done,held,na,na` |
| O02 spec line 54 | Amount, publication and ERP impact is required, read-only evidence | Per-plan `impact {money, publication, erp}` on each plan row; only all three `computed` counts as evidence | `TestEachMissingImpactPartHoldsActivation`, `TestImpactIsReadPerPlan`, admin.ts parsing test |
| O02 spec lines 71-72 | Impact calculation, then approval request with all dependency checks passing | Activation is refused while any part is not computed; no approval-request action is offered in step 1 | Go hold tests; browser checks for no activate button |
| O02 spec line 80 | A view without data is hidden or states a not-applicable reason; no fake values | The held sentence is the reason; impact rows read "Not calculated" with "This does not mean no impact." | browser "not calculated three times", "impact view held lead" |
| O02 spec line 85 | B09 is a linked screen | No B09 link is shown in step 1 because no approval request exists yet | not rendered; residual for step 2 |
| HB-274 plan lines 81-87 | `relationship.activate` preconditions | One precondition added after the stage checks and before the proposer check; order is `dependency_unavailable`, `invalid_stage`, impact, proposer | `TestHoldComesAfterStageChecks` |
| HB-274 plan line 102 | `impact` reported as `not_computed` | The page-level map stays as a legacy constant; rows carry the per-plan value | `TestActivationHeldWithoutImpactEvidence` (legacy map asserted) |

# Commands and results

GOCACHE pointed at the session scratchpad cache. Results below are from the round after the review fixes (see "Review fixes").

| Command | Result |
| --- | --- |
| `gofmt -l apps modules` | no files listed |
| `go vet ./...`, `GOOS=linux go vet ./...`, `GOOS=windows go vet ./...` | all clean |
| `go test ./...` | all 71 packages ok in two consecutive runs. One earlier run, while the browser regressions loaded the machine, failed `TestTimingIndistinguishable` on a timing ratio; that test is unrelated to this change and passed in the coverage run and both later full runs. |
| `go test -race -count=1 ./apps/api/server/` | ok (368.7 s) |
| `go test -count=1 -coverprofile … ./apps/api/server/` | ok, coverage 100.0% of statements |
| `npx tsc --noEmit` (apps/console) | clean |
| `npm test` (apps/console, now including `live-render.test.tsx`) | 634 tests, 634 pass, 0 fail |
| `npm run build` (apps/console) | built; the existing chunk-size warning only |
| grep for the binary floating-point type word in changed test files | no match |

# Browser QA

Runner: scratchpad `qa296.mjs` (the HB-275 `qa275.mjs` header with a retargeted body `qa296body.mjs`), run through `qa315reg.sh` with this worktree and `hb-api296` on port 4461.

Result: 209 PASS, 0 FAIL, no page or console errors.

The API port 8820 was in use by another ticket's runs at the time, so this run used a scratch copy of the built console (identical `dist`) whose preview proxy targets port 8829, with `hb-api296 --listen 127.0.0.1:8829`. The worktree was not changed for this. The P01 to P05 runs below used port 8820 and their API logs show no bind failure.

Covered:

- Proposal at 1440: errors summary, client date refusal, sealed proposal opens the impact view with focus on the plan title; state word "Changes locked"; "Not calculated" on the three impact rows and the Impact step, "This does not mean no impact." on each row and the pre-launch note; the held lead above the rows; four steps with `done,held,na,na` and no current step; the announcement "relationship changes are locked"; the impact button reads "Go to discard" and "Continue to approval" is gone; neutral tag, selection and held-block borders.
- Approval view for the proposer and for the second administrator: under the "Activation" heading the held block comes first, then the conditional restate "If this plan is activated …"; no "Activation is not available" title, no activate button, discard kept; the held block has a 16px bottom margin.
- Server member refusal on the field, rejected plan with `stopped,na,na,na` and no decision button.
- The second administrator discards a held plan (reason required, focus on the confirm title); the result shows "Discarded", steps `done,stopped,na,na`, no O01 link, and the announcement. Another open plan stays held and not stale; the active revision stays 1; selecting it in the result view offers "Go to discard".
- 390: proposal focus, decision button keeps the plan, approval view held with no activate button and discard kept.
- Member denied.
- EN, KO and AR at 1440, 1280 and 390 for the targets, impact, approval and result views: direction, `scrollWidth` not wider than the viewport, no clipping, out-of-viewport or overlapping step, held-block or approval-group elements, the held lead and second sentence in each language, four-step states, relabelled impact button, discard kept and no activate button; result detail without horizontal scroll; 390 back focus.

Screenshots are in the scratchpad `hb296qa/o02` folder (not committed).

HB-275 browser checks that need activation are unreachable in the product in step 1: "Admin 2 activates, and the O01 link shows the new attribution" (HB-275 plan line 254), "a stale plan after another activation" (line 258) and the closed "activated as revision N" row with the O01 link (lines 114-116). They are covered by Go tests with injected evidence (`TestProposeAndActivateMovesAttribution`, `TestStaleExpiredAndDiscardedPlans`, `TestRejectedPlansAndViewGroups`, `TestPlanReadsAcrossPagesAndLimits`) and by render tests with `can_activate: true` and all-computed fixtures. The browser gap is a residual.

# Regressions

Same API binary and preview, ports 4462 to 4467.

| Runner | Groups | Result |
| --- | --- | --- |
| P01 `qa283full` | one | 0 fails of 46 |
| P02 `qa285full` | A, B, C, D | 162, 32, 10, 16 pass; 0 fail |
| P03 `qa287full` | R, A, B, C | 288, 23, 12, 24 pass; 0 fail |
| P04 `qa289full` | R, A, B, C, D | 559, 28, 20, 46, 11 pass; 0 fail |
| P05 `qa315body-full` | R, A, B, C1, C2, C3, F, P (sample seed); E (`--seed none`, workspace `workspace`) | R 481, A 17, B 9, C1 2, C2 3, C3 30, F 132, P 12, E 142 pass; 0 fail |

# Mutation

Run on an rsync copy in the scratchpad (not the worktree) against the plan-related tests (`-run 'Plan|Propos|Rejected|Stale|Hold|Impact|Activation'`).

- Round 1: 35 mutants of the changed lines in `relplans.go` (each `==` and `&&` in `computed()`, each field of `impactAll`, the nil-source branch and its value, each activation case and refusal code, each read case, blocked word, case order, the row field, the JSON tag and the three state spellings). 33 killed. Two survived: the impact/proposer swap (equivalent: both answer `prerequisite_failed`) and a mutant meant to move the impact check before the stage check that, as written, left the order unchanged (identical code, so not a real mutant). The read order "impact unavailable before stale" survived and is listed in round 2.
- Tests added after round 1: the read of a stale and of a passed plan while the accessor fails (`TestHoldComesAfterStageChecks`).
- Round 2: 5 mutants (impact check before the stage check, accessor error before the stage check, impact-unavailable before stale, impact-unavailable before passed, the impact/proposer swap). 4 killed; the swap survives as the known equivalent.
- Round 3 (review fix C-L1, test only; no Go source line changed): the proposer check moved above the accessor-error check is killed by `TestImpactAccessorFailureHoldsActivation`; the impact/proposer swap still survives as the equivalent.

# Review fixes

Change review PASS (5 low), render review PASS (4 low), technical-writing review FAIL on copy. All items were applied.

| Item | Change | Test |
| --- | --- | --- |
| C-L1 | The proposer's own activation with a failing accessor answers `dependency_unavailable` | `TestImpactAccessorFailureHoldsActivation` |
| C-L2 | Console `canActivate` also requires all three impact parts `computed`; such a row is held (`impact` or `impact_unknown`) | model "steps, blocks and closings …", render "a row that claims activation without computed impact is still held" |
| C-L3 | The pre-launch note shows only when impact is not all computed | render "the impact view leads with the held sentence …" |
| C-L4, T2 | The impact view button and the result view's open-plan button share one label: "Go to discard" when held, otherwise "Continue to approval" | render `decisionLabel` assertions, browser |
| C-L5 | Unused `proposed` copy removed; double blank line removed | tsc, model tests |
| R-L1, T5 | Under the "Activation" heading the held block comes first, then the conditional restate | render "a held plan shows the held block …", browser |
| R-L4 | Held block bottom margin 16px | browser |
| T1, T1c | "Not calculated" for row tags and the step note; "Could not be read" for unknown; revised pre-launch note; KO held-unknown lead | render "renamed words …" |
| T3 | Limit sentence asks only for a discard | render "renamed words …" |
| T4, T4c | "Amounts", "Impact on amounts, publication and ERP", per-row "This does not mean no impact.", revised EN held lead | render tests, browser |
| T6a, T6c | "Changes locked", lead and announcement wording, `onlySealed` wording | render "renamed words …", browser |
| T7, T8, T9 | AR held step state, KO held-unknown second sentence, AR held second sentence | render "renamed words …", browser |
| T10 | Glossary: four O02 rows (activation held, changes locked, not calculated, could not be read) | document |

The render test file registers its CSS/SVG import stub with `registerHooks` when Node provides it and falls back to `register` on older Node 22 releases.

# Deviations from the plan

- The impact source is an unexported func field on the server's plan store (`s.plans.impact`), not a field declared on `Server`. `Server` is declared in `server.go`, which is outside the allowed files. The store is created only by `New`, is per server, and is never set by `New`, `Config`, a flag or a command; only `_test.go` files assign it.
- Copy changed beyond the plan text, following the review: `onlySealed`, the limit sentence, the restate, the impact heading, row labels and tags, `notZero`, `impactNote`, and the button labels.
- The impact row reads "Calculated" for a `computed` part and then omits the row note and the pre-launch note. No pre-launch producer exists, so this is reachable only through test injection.
- The console holds a row whose impact is not all computed even when the server sends `can_activate: true` (defence in depth).

# Pending

- Native-language review of the KO and AR drafts.
- Browser checks for activation stay a residual until step 2.

## Root re-verification (2026-11-18, main 9699fa1)

- `gofmt` clean, `go vet ./apps/api/server` ok, `go test -count=1 ./apps/api/server` ok (35 s).
- `tsc` clean, `npm test` 634/634 (root registered `live-render.test.tsx` in `apps/console/package.json`).
- No changed file over 1,500 lines; superseded strings ("Activation and discard", "Changes fixed", "Not computed") absent from the O02 console source.
- Reviews: design rounds 1–2; change review PASS; render review PASS; technical-writing review FAIL then fixed as in plan amendments 1–4. KO and AR remain drafts for native review.
- Browser note: O02 QA was run from a scratch copy of the built console with the API on port 8829 because another ticket's API held 8820; the bundle was identical to the worktree build.

