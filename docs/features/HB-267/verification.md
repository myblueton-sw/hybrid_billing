---
type: VerificationRecord
title: HB-267 verification of the live M03 access screen
description: Unit tests, build, browser QA in EN, KO and AR at 1440 and 390, M01 and M02 regression reruns, and the design comparison and change review results for the live M03 access screen. Run on mac_mini on 2026-11-16.
status: stable
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

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check. Rows marked "(classified by HYB-301)" replace rows that HYB-291 left unclassified; the [HYB-301 trace classification](../../reviews/HYB-301-trace-classification/report.md) gives the evidence.

## Requirement trace

Source: `docs/ui/screens/M03/spec.md:133-159` (HB-266 is the API, HB-267 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-03 | deferred | Addressed within pre-launch scope for grants, delegation revoke and the effect preview, with self-grant refused; registry-wide and leakage acceptance not built. | HB-266/plan.md |
| HB-04 | not connected | (classified by HYB-301) M03's part of HB-04 is keeping the setup role apart from export write and collector read (`requirements.json` HB-04 implementation_proposal) and testing operational reads after setup is revoked. Built: grants with a validity window and revoke (HB-266/plan.md:25, 119); the only grantable pair is `view focus_data rows` (HB-266/plan.md:106). Not built: a setup permission. It needs provider onboarding (S02) and collectors; S02 is outside the Go-bound pre-launch scope and served ungated on 8793 (HYB-291 M9). | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| HB-23 | deferred | Grant changes apply directly, without the approval request `M03/spec.md:83` requires. 2026-11-18: the approval request was built by HYB-298 (PR #212). | Planning cross-check residuals below |
| HB-33 | deferred | Grant commands use the common command path with idempotency keys and expected revisions, but after a partial failure a second submit can create a second grant. | HB-267/verification.md:42 |
| HB-36 | deferred | Support sessions are not available. | HB-266/plan.md:153-161 |
| HB-42 | deferred | The revocation states received, deny confirmed and cleanup running (`M03/spec.md:84, 109-113`) do not exist. 2026-11-18: built by HYB-298 (PR #212); received, deny confirmed and cleanup running pass inside one command. | Planning cross-check residuals below |
| AUD-AR-09 | deferred | Delegation creation and support sessions are not available; ended delegations are listed by ID only. | HB-266/plan.md:153-161, HB-267/verification.md Residuals |
| AUD-DB-08 | deferred | (classified by HYB-301) Built: grants expire and can be revoked (HB-266/plan.md:25, 119). Not connected: the CSP-side setup permission with a revocation condition and operational-read-only access after it is revoked (external). Not built: task-scoped expiring handoff links (`M03/spec.md:143-149`), which wait on binding S02 provider onboarding (owner decision; S02 is outside the Go-bound pre-launch scope). | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| AUD-QA-04 | deferred | Idempotency keys and expected revisions are enforced; OAuth client credentials, PKCE and the client-user intersection are not built. | HB-266/plan.md |
| IAM-01–IAM-08 | deferred | Role and revocation paths exist for local users only; approval request, revocation states, bulk paths and federated checks are not built. 2026-11-18: approval request and revocation states built by HYB-298 (PR #212), bulk revocation by HYB-299 (PR #218); federated checks remain open. | Planning cross-check residuals below |

## Planning cross-check residuals (HYB-291)

- **Approval request and revocation states.** The screen applies grants directly and shows no revocation states (`M03/spec.md:83-84, 109-113`); see HB-266. (2026-11-18: built by HYB-298, PR #212.)
- **Security-focused review not stated.** HB-257 (Verification for every phase) requires a security-focused review for phases 1–3. This record names design and change reviews but no security-focused review. None is claimed here; a security re-review or a recorded reason is open.
