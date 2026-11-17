---
type: VerificationRecord
title: HYB-315 verification of the P05 support combinations and release evidence screen
description: Checks, browser QA, regression runs, review outcomes and the planning cross-check for the P05 service-mode screen bound to the HYB-290 release evidence API, run on mac_mini on 2026-11-18.
status: final
tags: [HYB-315, P05, release-evidence, screen, verification]
sources:
  - resource: plan.md
  - resource: visual-spec.md
  - resource: references.md
---

# Scope

This record verifies [HYB-315](https://linear.app/hybrid-billing/issue/HYB-315) against its [plan](plan.md) and [visual spec](visual-spec.md).
- The plan section "Root amendments after review (2026-11-18)" holds every decision taken during review. It overrides the spec on layout as well as copy.
- A delegated agent wrote the code in the ticket worktree. Root re-ran the checks marked "root" on the final tree.
- All browser runs used a fresh `hb-api` built from the worktree, synthetic data and new mode-0700 data directories that were deleted afterwards.

**Runtime.** Every result below comes from one machine: Node 26.7.0, React 19.3.1, Vite 8.3.5 and Playwright Chromium headless shell, on macOS 26.2. That is a tested combination, not evidence of other operating systems or browsers.

# Results

| Check | Result | Run by |
| --- | --- | --- |
| `tsc --noEmit` | clean | root, final tree |
| `npm test` | 608 pass, 0 fail | root, final tree |
| `npm run build` | ok | root, final tree |
| File size | largest changed file `live.css`, 1,093 lines; none over 1,500 | root |
| Explanatory comments | none in the changed files | root (grep), change review |
| Shared files | `service/admin.ts` (+9/−1), `app/screen-groups/operations.tsx` (+8/−1), `package.json` (+1/−1); unchanged since the regression run below | root |

# Browser QA (implementer, final tree)

| Group | Content | Result |
| --- | --- | --- |
| R | 45 renders: EN/KO/AR × 1440/1280/390 × five tabs. Includes overflow sweeps, decisions hit-area and line-box checks, hyphen-break checks | 481 pass, 0 fail |
| F | Evidence detail: split and step, pointer versus keyboard focus with real `page.mouse.click`, hover tint, computed divider and error-summary border colours | 132 pass, 0 fail |
| A | First administrator flows | 17 pass, 0 fail |
| B | Second administrator flows | 9 pass, 0 fail |
| C1 | Conflict | 2 pass, 0 fail |
| C2 | Lost response | 3 pass, 0 fail |
| C3 | Lost request, stubbed refusals, step-up, client checks, focus, drafts, URL state, read failures, revoked, member denied | 30 pass, 0 fail |
| E | `--seed none` and first-administrator activation | 142 pass, 0 fail |
| P | Preview without the service | 12 pass, 0 fail |

Groups A, B, C1, C2, C3, E and P ran on the round-2 tree. Later rounds changed only P05-scoped presentation, copy and tests, and R and F were re-run on every round.

**Layout measurements** (final tree):
- No horizontal scroll and no `td` or `dd` overflow in any of the 45 renders.
- `.p05-live` widths:

  | Viewport | `.p05-live` | With a detail open (list / panel) | Mode |
  | --- | --- | --- | --- |
  | 1440 | 1120 | 662.4 / 425.6 | split |
  | 1280 | 960 | 563.2 / 364.8 | split |
  | 390 | 358 | — | step |

  The split threshold is exact: 939 gives step mode and 940 gives split (548 / 360).
- At 1440, no requirement label or combination ID breaks at a hyphen.
- "sample-trace-harness" fits on one line in Results at 1440.
- Decision summary links have a hit area of 44–45px, and line spacing varies by at most 2px.

**Fail-closed rendering.** Unit tests feed unexpected server values into the screen: rule-1 values, connection fields and state-like fields. For each, they assert the field locator and that no raw positive value appears. A positive-word count, which skips negated forms anchored at word boundaries, compares each mutated render with an unmutated one in EN, KO and AR.

Screenshots and runners are in the session scratchpad and are not committed.

# Regression

Root confirmed that no shared file changed after these runs.

| Screen | Groups and checks | Failures |
| --- | --- | --- |
| P04 | R 559, A 28, B 20, C 46, D 11 | 0 |
| P03 | R 288, A 23, B 12, C 24 | 0 |
| P02 | A 162, B 32, C 10, D 16 | 0 |
| P01 | 46 | 0 |
| HYB-293 | A 21, B 32, C 27, D 54 | 0 |

# Reviews

All reviewers were read-only and separate from the writer. A review PASS is not owner acceptance.

**Change review (independent-reviewer).**
1. **Round 1: FAIL.**
   - Major: the unexpected-value sentence printed the positive server value.
   - Thirteen minors.
2. **Round 2: PASS.** Remaining minors: test strips, DOM ids, one KO term and aria-label isolation.
3. **Round 3: PASS.** Remaining minors:
   - the noPositive EN blind spot;
   - static not-connected copy;
   - summary link overlap.
4. **Round 4: PASS.** One last minor:
   - KO and AR negation anchors in the test helper.
   - Root also moved test outcome to the state-like fields. Both were applied.

**Rendered review (ui-designer).**
1. **Round 1: FAIL.** Blocker: harness ID overflow. Three majors:
   - the tests summary wrap;
   - the Customer proof header height;
   - panel state grouping.
2. **Round 2: PASS.** Minors N1–N5.
3. **Round 3: PASS.**
   - N4 and N6 open. The final polish applied N4, N6, N5 and the digest nit, and measured each.
   - Widths deviate from the reviewer's numbers where the reviewer's numbers overflowed. Root accepted the deviations.

**Terminology review (tech-writer role).**
1. **Round 1: FAIL.** 29 findings.
2. **Round 2: FAIL.** Unapplied KO and AR items, and amendment 8 was not yet built.
3. **Round 3: FAIL.** Two AR word-order strings, since fixed exactly as proposed.

Every root decision is recorded in the plan amendments, items 1–10.

# Planning cross-check

The requirement-by-requirement cross-check is the table at the end of the [plan](plan.md). It was written from the requirement text of:
- HB-04, HB-19, HB-24, HB-25, HB-26, HB-38, HB-39, HB-40 and HB-41;
- AUD-DB-01 and AUD-DB-09.

Root checked the built screen against it.

**Built as stated:**
- Separate design, implementation, test and customer proof states per combination and requirement (HB-04, HB-41).
- Common-requirement gate problems and no-replacement sentences (HB-24, AUD-DB-01).
- No path reads active and no requirement reads customer-proven, and unexpected positive values fail closed (HB-25, rule 1).
- A template appears only for an active, problem-free combination, which cannot occur (HB-26).
- Origin labels and sample notes (HB-38).
- Load profiles, budgets, runs and verdicts never read passed, and "Not run" and "Not measured" are distinct from 0 (HB-39).

**Still deferred, as the table states:**
- Real ERP and customer proof (HB-25).
- XLSX result registration (HB-26, AUD-DB-01).
- Scenario manifests and demo reset (HB-38).
- Budget approval and measurement at scale (HB-39).
- Distribution-notice evidence (HB-40, P07).
- The real planning trace (HB-41).

**Not connected and not built:** HB-19 and AUD-DB-09 (the commitment event matrix).

Nothing in this ticket changes those statuses.

# Residuals

- **Copy pending review.** KO and AR copy are installation defaults pending owner and native review (plan amendment item 11). This includes:
  - the AR issuance phrase and customer proof wording;
  - the AR verb-form labels;
  - the reauthentication-cancel sentence alignment with the shared sentence.
- **Wrapped test cell.** In the full matrix at 1440, the whole Test cell, including its failing line, takes 4 lines in EN and 5 in AR. The tested sentence alone takes at most 3.
- **AR wrapping.** The AR per-row tests summary wraps by count. The reading order of each count line is correct.
- **Not directly checked.**
  - The `P05:state:unconnected` review marker. It comes from the unchanged `onState` path.
  - Step mode between 720px and 939px of `.p05-live`, which has no table by decision.
- **Not connected and not built.** Release, activation, support decisions, customer proof intake, budget approval and artifact files remain not connected. This is visible on the screen.
- **Not owner acceptance.** None of the checks or reviews above is owner acceptance of the screen.
