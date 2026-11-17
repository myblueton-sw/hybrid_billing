---
type: VerificationRecord
title: HB-273 verification of the live O01 organization and relationship explorer
description: Unit tests, build, browser QA in EN, KO and AR at 1440 and 390, the M01–M05 regression reruns, and the review outcomes for the live O01 screen. Run on mac_mini on 2026-11-16.
status: final
---

# Scope

This record verifies [HB-273](https://linear.app/hybrid-billing/issue/HB-273) against its [plan](plan.md).

**Environment**
- mac_mini (macOS), Node 26.7.0, Playwright 1.65.1 with the Chromium headless shell.
- A locally built `hb-api` from this branch.
- `vite preview` with `HB_GATED=1`.
- Synthetic sample data only.

# Results

| Check | Result |
| --- | --- |
| `npx tsc --noEmit` and `npm run build` | pass |
| `npm test` (console) | 372 pass, 0 fail |
| O01 browser QA (the script is kept outside the repository) | 53/53 pass, no console errors |
| M01–M05 QA reruns on this build | 51/51, 38/38, 38/38, 49/49 and 62/62 pass. The final build reran O01 and M05; the rest ran on the build before the last O01-only fixes. |

**What the O01 QA covers**
- **Scope:** the scope facts (revision, active marker, workspace, profile) and the roots.
- **Tree:**
  - Expand all keeps focus.
  - The move boundary: Account 02 is under North operations on 2026-09-30 and under North data platform on 2026-10-01, checked through the DOM.
  - The panel shows profiles with the current-registry note.
  - A selected row has a neutral border.
  - Members before the start of their relationship show as unattributed.
- **History:**
  - View history moves focus to the member select and keeps the date.
  - The exclusive end and "Applies" are shown.
  - Both view groups of Account 03 show "Applies".
- **Accounts:** the date and pinned revision are kept across tabs, and all six accounts are listed.
- **Groups:** the shared account appears in both groups, and the pool note is shown.
- **Access:** a member is denied.
- **Locales:** EN, KO and AR at 1440 and 390, on the tree, accounts and groups views:
  - text direction;
  - no horizontal scroll;
  - at 390, focus moves to the detail and returns after Back.

# Reviews

These reviews were done by AI agents, not by the owner.

**Design review** (independent reviewer, read-only, data accuracy)
- Round 1: FAIL.
  - F1 (High): false absences during an incomplete load.
  - Mediums: the registry versus the revision, multiple targets, tree completeness and the accounts join.
- Round 2: FAIL. Root parties during an incomplete load, and hidden subtrees.
- Round 3: PASS.

**Design comparison** (`ui-designer`, read-only)
- Round 1: FAIL. H1: tree names were centred and collided with the kind column.
- Round 2: PASS, with two follow-ups, both applied.

**Change review** (independent reviewer, read-only)
- Round 1: FAIL.
  - M1: the date and pin were lost across views.
  - M2: focus was lost after View history.
  - M3: only one history entry was marked as applying.
  - M4: contradictory empty messages.
- Round 2: PASS, with Lows N1–N5. All five were applied afterwards:
  - the pin survives the double effect run;
  - the revision is cleared only when the date changes;
  - counts use the primary kind;
  - URL updates replace the history entry;
  - focus moves to history only after a View history link.

No review is owner acceptance.

# Residuals

- O01 is admin-only, inherited from HB-272.
- The tree is built on the client from flat pages.
- There is no member filter, no `parent=` query and no link between accounts and billing profiles.
- The contract pool is left for O03.
- A one-frame render with the old date when the date changes.
- A possible flicker of the error state in accounts.
- No automated test of the paging cap, restart or inconsistency logic. The sample fits in one page.
- No `aria-busy` while reloading or paging.
- The shell still shows "Work service not connected", the screen-review strip and the side-navigation offset, as on M01–M05.
- KO and AR terms (지급 주체, 미귀속, 내부 비용 청구, الجهة الأم and others) need glossary and technical-writing review.
- The sample data deliberately crosses customers.
- Screen acceptance (FIX) is the owner's alone.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check.

## Requirement trace

Source: `docs/ui/screens/O01/spec.md:116` (HB-272 is the API, HB-273 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-01 | deferred | Covered (O01): parties, billing profiles and relationship kinds are listed separately. Not built: the three operating profiles on one core, internal settlement without an external invoice, and blocking ineligible cross-entity or cross-currency sums. | HB-272/plan.md |
| HB-02 | deferred | Read-only explorer over one pinned revision; the group-draft and change-plan actions (`O01/spec.md:72-73`) are missing. | Planning cross-check residuals below |
| HB-03 | deferred | Addressed within pre-launch scope for an admin-only read; registry-wide and leakage acceptance not built. | HB-272/plan.md:208-215 |
| HB-14 | deferred | The contract pool is left for O03; legal-entity, relationship-kind and provider filters (`O01/spec.md:61`) are missing; the client-built tree conflicts with `O01/spec.md:102` and UI-04. | HB-273/plan.md:190-199, residuals below |

## Planning cross-check residuals (HYB-291)

- **Missing actions and filters.** The screen offers neither the view group draft nor the create change plan action (`O01/spec.md:72-73`), nor the legal-entity, relationship-kind and provider filters (`O01/spec.md:61`).
