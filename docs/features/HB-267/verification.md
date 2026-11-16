---
type: VerificationRecord
title: HB-267 verification of the live M03 access screen
description: Unit tests, build, browser QA in EN, KO and AR at 1440 and 390, M01 and M02 regression reruns, and the design comparison and change review results for the live M03 access screen. Run on mac_mini on 2026-11-16.
status: final
---

# Scope

This record verifies [HB-267](https://linear.app/hybrid-billing/issue/HB-267) against its [plan](plan.md). Every check below was run on mac_mini (macOS, Node 26.7.0, Playwright Chromium headless shell) against a locally built `hb-api` from this branch and `vite preview` with `HB_GATED=1`, using the sample seed only. No real customer data was used.

# Results

| Check | Result |
| --- | --- |
| `npx tsc --noEmit` | pass |
| `npm test` (console) | 353 pass, 0 fail, including the new `access/live-model.test.ts` and the access client cases in `admin.test.ts` |
| `npm run build` | pass |
| M03 browser QA (script kept outside the repository) | 38/38 checks pass, no console errors |
| M01 QA rerun on this build (shared CSS change) | 51/51 pass |
| M02 QA rerun on this build (shared CSS change) | 38/38 pass |

The M03 browser QA covers:
- the grants list, the console ceiling and the empty delegations state;
- self excluded from the person list and "Your grant" shown in place of Revoke;
- the grant form's validation summary, the live end date and impact line, a grant to the member, and the member then passing the API gate for a FOCUS reader;
- the revoke step with heading focus, a neutral selected row border, Cancel restoring focus, the conflict message after a concurrent change, the revoke itself, the ended disclosure, and the member then refused by the gate (403);
- the effects preview, its neutral note border, results cleared on an input change, and a delayed in-flight preview dropped after the person changes;
- roles and support unavailable views with "Go to Grants";
- the member denied view;
- EN, KO and AR at 1440 and 390: direction, no horizontal scroll, the stepped grant form with heading focus and "Back to grants" focus return, and the effects view at 390.

# Reviews

- **Design comparison** (`ui-designer`, AI, read-only): round 1 returned changes required with four Medium findings: conflict copy naming the user list, mislabelled revoke facts, the ragged effects row, and a disclosure without a marker. All four and Lows L1–L2 were fixed; round 2 returned PASS on the re-taken renders it opened. Its remaining Low L3 (the sidebar sitting lower in the conflict full-page capture) is attributed to the sticky sidebar captured after focus scrolls to the failure box; the revoke and effects captures of the same build show it in place. This inference was not measured.
- **Change review** (independent reviewer, AI, read-only): round 1 returned FAIL for one Medium (an in-flight preview could show a result for a person no longer selected). It was fixed with a sequence advance on every input change and a result-bound kind. Lows L1–L4 were fixed. Round 2 returned PASS with three new Lows: N1 (preview errors used save copy) and N2 (an empty panel when nothing is grantable) were fixed afterwards and covered by the final 38/38 QA run; N3 (mobile labels in bold M01/M02 cells become regular weight) is accepted as intended.

Neither review is owner acceptance.

# Residuals

- L5 from the change review: after a "partial" failure the settled reload renews the attempt, so a second submit can create a second grant. This is the same pattern as M01 and M02; the committed grant is visible in the reloaded list.
- Ended delegations are listed by ID only; delegation creation is not available in pre-launch.
- KO and AR copy is an unreviewed AI draft and needs the technical-writing and native-language review.
- The effects meta and the Why column repeat reason text per row; grouping was not adopted.
- Shell issues carried from HB-264/265: "Work service not connected" in the header, workspace truncation at 390 px.
- Screen acceptance (FIX) for M03 is the owner's alone.
