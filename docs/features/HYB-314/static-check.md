---
type: ReviewReport
title: UI-10 static check of the phase 2 to 5 screens
description: Findings of a read-only check that the sign-in, administration, organization and installation screens keep unknown, zero, stale and not-connected states apart and have no fake success or dead action.
status: draft
---

# Scope and method

HYB-314, machine:mac_mini. Requirement UI-10 (`docs/planning/v1.16/ui-design.md:553`, translated): keep stale, null, zero, unsupported and unconfirmed apart; no fake success and no action that does nothing. Its parts about channel acceptance against user confirmation and inquiry against resolution belong to messages and cases, which are not connected, and are outside this check.

On 2026-11-24 three role-based AI reviewers, separate from root and from each other, read the console source at main d9bcf99 for the screens of phases 2 to 5 (`docs/features/HB-257/plan.md`, phase table): the sign-in and entry screens L01 to L03, M01 to M05, O01 to O03 and P01 to P07. Nothing was executed or rendered. Each reviewer applied five checks: (1) a value the server did not provide is not shown as zero, as "none" or as a default; (2) a failed or loading read is distinguishable from an empty result and an old value is not left as if current; (3) unsupported or not connected is said as such; (4) no success shown without the server's confirmation; (5) no button without effect and no control disabled without a stated reason.

Root then read six of the confirmed findings in the code and found each as reported; they are marked "root". The other findings rest on the reviewers' reading. A finding is "confirmed" when the code path is unambiguous by reading and "suspected" when a render or a server response is needed to settle it. Paths are under `apps/console/src/`.

This is the static part of the ticket. No fix is made by this report, and the rendered part of the check has not been run.

# Result by screen

| Screen | Result | Confirmed findings | Suspected |
| --- | --- | --- | --- |
| L01 to L03 entry screens | no finding | none; none is server-bound and each says it is not connected | none |
| M01 users | findings | A1, A2, A3, A5, A6 | A4, A7 |
| M02 authentication | findings | B1, B2, B3 | A4 |
| M03 access | findings | C2 | C1, A4 |
| M04 service accounts | findings | A5 | A4 |
| M05 my security | findings | D1, D2 | D3, D4 |
| O01 organizations | findings | E1, E2, E3, E4 | E5 |
| O02 changes | findings | F1, F2 | F3, F4 |
| O03 contracts | findings | G1, G2, G3, G5, G6 | G4 |
| O03 issuance setup views | one finding | G5 | none in the files read |
| P01 installation | findings | H1, H3 | H2, H4, H5 |
| P02 integrations | findings | none | H2, H5 |
| P03 recovery | findings | H1 | H5 |
| P04 retention | findings | H1 | none |
| P05 release evidence | unverified | none | H4; four view files unread |
| P06 demo, P07 licences | no finding | none; both render static screens that say not connected | P07 notice link targets not verified |

# Findings

## A common question behind the "fake success" items

Many screens show their success text when a command request resolves, without reading the returned status or business state (`service/admin.ts`, `outcomeOf` and `send`; a resolved request is any 2xx). The command layer has two success statuses, completed and accepted (`modules/commands/types.go:224`, `:225`). Whether each command named below can return a 2xx whose state is not the one the success text states was not traced, so these are suspected, not confirmed. P03, P04, P05 and the O03 issuance setup stores already guard this step and treat an unexpected result as unknown, which is the pattern to follow.

## M01 users

- A1, check 5, confirmed (root). `modules/identity/pages/users/BulkPrepare.tsx:269` to `:274`: discarding a draft catches every failure and returns, so the discard buttons do nothing visible when the command fails. The same holds for the fixed list at `:278` to `:283` for most failures.
- A2, check 5, confirmed. "Show more" gives no feedback on failure (`BulkPrepare.tsx:37`, `:401`; `RevocationJobs.tsx:252`).
- A3, check 2, confirmed. `users/Live.tsx:769`, `:788`, `:825`: below 1280 px with a detail open, a failed reload shows no error while the detail keeps the values from before the action beside its done notice.
- A4, check 4, suspected, shared by M01, M02, M03 and M04: success text on any resolved command (`users/Live.tsx:152`, `:219`; `service-accounts/Live.tsx:88`; `authentication/Live.tsx:142`; `access/Requests.tsx:77`). Trust activation on M02 does read the state.
- A5, check 2, confirmed, M01 and M04. `users/Live.tsx:266` and `service-accounts/Live.tsx:461`, `:523`: any failure other than "unavailable" (rate limit, denied, session ended) is reported as "this link, or secret, can no longer be shown; issue a new one".
- A6, check 1, confirmed in code. Defaults stand in for absent or unrecognised values: membership role, state and period (`service/admin.ts:105`, `:110`), any unrecognised account state reads "Active" (`users/live-model.ts:19`), an absent warning period becomes 14 days (`admin.ts:976`, `users/Live.tsx:672`).
- A7, suspected. A caught staging error followed by "the fixed list was discarded" (`users/Live.tsx:705`, `:740`); a swallowed bulk-list failure (`:599`); absent match counts shown as 0 (`match-model.ts:32`, `BulkPrepare.tsx:94`); a failed read of excluded people rendering nothing under its heading (`BulkPrepare.tsx:41`).

## M02 authentication

- B1, check 1, confirmed in code. An absent impact becomes 0 members and 0 sessions and is printed as numbers (`service/admin.ts:589`; `authentication/TrustChanges.tsx:282`; `TrustConfirmBody.tsx:92`), while another line of the same file treats 0 as "not recorded". An absent sign-out flag defaults to true and prints the warning.
- B2, check 1, confirmed in code. Unknown change state reads "draft" (`admin.ts:579`); approvals needed fall back to 3 (`:633`); the step-up period falls back to 900 seconds with no "default" label (`authentication/live-model.ts:32`).
- B3, check 2, confirmed. The same narrow-layout gap as A3 (`authentication/Live.tsx:175`, `:190`).
- Low: the methods table is hard-coded; password "Active" is not read from the server.

## M03 access

- C1, check 5, suspected. With an empty catalogue the request button is enabled once a person is chosen and the submit returns silently (`access/Live.tsx:273`, `:282`, `:327`).
- C2, check 2, confirmed. After a failed reload the effects view renders from the retained page with no load error (`access/Live.tsx:438`).

## M05 my security

- D1, check 5, confirmed (root). `my-security/SessionOperations.tsx:83` to `:85`: for an unconfirmed operation the "Results" button only changes its expanded state; nothing is rendered.
- D2, check 1, confirmed. "12 hours after sign-in" is a constant shown among limits read from the server (`my-security/LiveSessions.tsx:144`).
- D3, suspected. Every identity row is labelled "Password" whatever its method (`my-security/Live.tsx:266`).
- D4, suspected. An unconfirmed operation, or any read error, shows as "unavailable" (`SessionOperations.tsx:143`).

## O01 organizations

- E1, check 2, confirmed (root). `organizations/Live.tsx:297` to `:304`: when the member-facts read fails, the detail's parent row shows "Loading…" indefinitely, with no error and no retry.
- E2, checks 2 and 4, confirmed. `organizations/drafts.tsx:425`, `:429`: if the read-back after a successful save or discard fails, the panel says the draft no longer exists while the live region announces "draft saved".
- E3, check 2, confirmed. The existing-group search in the draft panel shows neither loading nor error, and on failure keeps the rows of the previous term under the new term (`drafts.tsx:597`, `:789`).
- E4, check 1, confirmed in code. An absent member limit becomes 1000 and is shown as a real limit; an absent open limit becomes 0 and prints "n of 0"; absent member counts print as 0 (`drafts.tsx:59`, `:178`; `service/admin.ts:445`, `:1083`, `:1110`). Whether the server ever omits these fields is not verified.
- E5, check 4, suspected. A save with an empty returned identifier still announces "saved" (`drafts.tsx:452`, `:480`).

## O02 changes

- F1, check 2, confirmed. After a failed reload the old plan list and active revision stay on screen with no message; error text exists only when no page is held (`changes/Live.tsx:454`, `:747`, `:778`).
- F2, check 4, confirmed for the live region. "Activated", "sealed" or "discarded" is announced from an address parameter without checking the plan's state in the loaded page (`changes/Live.tsx:440`, `:486`).
- F3, check 4, suspected. Every resolved proposal not "rejected" is treated as sealed; activate and discard navigate to the result without reading the state (`:696`, `:707`).
- F4, check 1, not verified. Absent member count and open limit become 0 (`service/admin.ts:422`; `changes/Live.tsx:646`).

## O03 contracts

- G1, checks 1 and 2, confirmed (root). `contracts/Live.tsx:136`, `:147`: the counter reads "0 of 0 contracts" while loading and after a failed first read, beside facts that correctly show a dash.
- G2, check 2, confirmed. A failed detail refresh with a detail already held shows no error, including the reload after a write, so a success notice can sit above the detail from before the write (`contracts/Live.tsx:96`, `:192`, `:202`).
- G3, check 2, confirmed, transient. "This contract is not available" shows while the detail read is still pending for an identifier not in the loaded list (`:187`).
- G4, check 4, suspected. The done notice on any resolved command (`contracts/LiveActions.tsx:170`).
- G5, check 5, confirmed (root). `contracts/setup/Writes.tsx:62`: "Check" is disabled when the selected configuration is not on the loaded history page, with no reason shown.
- G6, check 2, confirmed, low. After a context read failure the sample-options section is a heading with nothing under it (`setup/read-store.ts:26`; `setup/Documents.tsx:82`).

In the issuance setup files that were read, the read stores drop their value on reload and on error, and the write stores never mark an entry recorded after a caught error.

## P01 to P05

- H1, check 2, confirmed (root for P01). On a failed reload P01, P03 and P04 keep and render the previous page, still show the old server time under "Now (UTC)", say only that the data could not be loaded, and keep commands offered on the old data (`administration/pages/installation/Live.tsx:66` to `:75`, `:142`; `recovery/Live.tsx:77`, `:162`; `retention/Live.tsx:86`, `:167`). P02 and P05 handle this: they disable commands on error and say so.
- H2, check 4, code path confirmed and effect suspected, P01 and P02. Success text with the state ignored, including "proposed as proposal Unknown" when the proposal identifier is absent (`installation/LiveProposals.tsx:300`, `:388`; `integrations/LiveProposals.tsx:169`, `LiveModules.tsx:251`, `LiveChannels.tsx:211`, `:323`, `LiveEvents.tsx:138`, `:189`, `LivePending.tsx:47`).
- H3, check 1, confirmed, P01. An absent or malformed mark is rendered as "None" (`service/installation.ts:139`; `installation/LivePlan.tsx:169`); the P02 parser keeps the two apart.
- H4, checks 4 and 5, suspected, P01 and P05. A denied command whose follow-up reload returns nothing ends with no message (`installation/LiveProposals.tsx:115`; `release-evidence/LiveCommand.tsx:209`).
- H5, check 1, suspected. Empty strings shown as definite values: "Direct", "no verifier", "no series", "None" (`installation/Live.tsx:238`; `recovery/LiveIsolation.tsx:161`; `recovery/LiveReconciliation.tsx:61`; `integrations/LiveModules.tsx:279`).

# Patterns

1. Stale page after a failed reload (A3, B3, C2, F1, G2, H1): six screens keep the earlier page without saying so. P02 and P05 show the handling to copy.
2. Parser defaults for absent fields (A6, B1, B2, E4, F4, H3, H5): `service/admin.ts` and screen models turn absent or unrecognised values into zero, "active", "draft" or a number. The fix is at the parser, with an explicit unknown.
3. Success text on a resolved request (A4, E5, F3, G4, H2): settled only by tracing which commands can return a non-final state.
4. Silent failure of a secondary action (A1, A2, C1, D1, G5): five places.

# What remains for the ticket

- Fix the confirmed findings; patterns 1 and 2 are each best done as one change across screens, with tests.
- Settle the suspected findings by tracing the commands or by a rendered check with injected failures.
- Read the files listed below, and run the rendered part of the check.

# Not read

Reported by the reviewers as not opened, so no statement is made about them: on M01, `RolePeriod.tsx`, `Selection.tsx`, `filters.tsx`, `Stamp.tsx`, part of `RevocationJobs.tsx` and the bulk, period and filter models; on M02, the trust model and copy; on M03, `Settings.tsx`, `Area.tsx`, `who.tsx`, the models and most of the result-authority helpers; on M04, `Held.tsx`, the model and part of `Live.tsx`; on M05, `SessionSettings.tsx` and several models; on O03, `Channels.tsx`, `Changes.tsx` and the four step-write components with their flow files; on P02 to P05, the models and formatters and, on P05, `LiveMatrix.tsx`, `LiveRequirements.tsx`, `LiveDecisions.tsx` and `LiveRetireForm.tsx`. Copy files were read by search only. No style sheet and no test was read. Which of the static and live components mounts for each screen was not verified for M02 to M05.

# Fixes

## O03 findings G1, G3, G5 and G6 (2026-11-24)

- G1: the list counter is rendered only once the list has loaded, so no "0 of 0" shows while loading or after a failed first read.
- G3: "This contract is not available." is not shown while the detail read is pending; the loading status shows instead.
- G5: when the settings chosen under "Start from" are not on the loaded page of recorded issuance settings, a sentence beside the disabled "Check permission and review" button says so and names the paging buttons. It is shown only when the history page is known.
- G6: after a failed scope read the sample section keeps its heading and says that it is not shown because the scope could not be loaded.

Checks: `tsc` clean; console tests 1345, pass 1342, fail 0, skipped 3; a new render check with injected read failures, `tools/o03-trust-states-qa.mjs`, passes in English, Korean and Arabic at 1440 and 390 for G1, G3 and G6; the existing parts, rules-writes and payment-terms render checks still pass. G5 has no rendered check. Copy and state review, ui-designer role (AI, from renders and source): FAIL with three findings on G5 and G6 (terms that did not match the view, the reason shown while the history page was unknown, a form line pointing at a hidden sample), corrected with its strings; corrective re-review PASS.

Left open from that review and this work: a refused contract detail shows the general failure text, whose wording speaks of signing in; the failed list shows its alert above the section and nothing in the list column; "Loading contracts…" is used for one contract; while the settings history is loading or failed, the disabled button has no reason of its own; Arabic uses two words for "sample" in two copy files; G2 (a failed detail refresh after a write shows no error) is not fixed here.
