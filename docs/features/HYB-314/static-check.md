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

## M01 findings A1 and A2, M05 finding D1 (2026-11-24)

- A1: when discarding a draft or list fails, a message beside the discard button says that the discard was not confirmed and that the draft or list may still exist. It is cleared when the person resumes, prepares again or discards successfully.
- A2: when "Show more" fails for the people of a prepared list, the excluded people or the people of a job, a message beside the button says so and that the people already shown are unchanged. It is cleared when another job is opened or closed.
- D1: opening the results of an operation whose result is not confirmed shows a sentence saying that no results can be shown until it is confirmed.

Checks: `tsc` clean; console tests 1345, pass 1342, fail 0, skipped 3. No rendered check and no new test: these states need a failed command or read in the bulk revocation and session flows, which no existing render check stages. Copy and state review, ui-designer role (AI, by reading): FAIL with five findings (messages placed far from their buttons, a message that outlived its cause, and three wordings), corrected; the re-review passed all five and found one more message that outlived its cause, corrected afterwards and not reviewed again.

Left open: the new messages use the secondary text style, whose prominence is for the owner's screen review; three of them sit inside a container that is itself an alert; whether "try again" can succeed after a discard whose outcome was unknown was not traced; C1 on M03 is not fixed here.

## P01, P03 and P04 finding H1 (2026-11-24)

When a reload fails while a page is held, the three screens now say, after the failure alert, that everything on the screen including the time is from the last successful load and that the screen is read-only until a reload succeeds. P03 and P04 stop offering commands in that state through their existing command gate. P01 had no gate: its three command openers and the review and send buttons of an open form are disabled and point at the note.

Checks: `tsc` clean; console tests 1345, pass 1342, fail 0, skipped 3, with the P01 proposals render test extended for the stale state of the openers and of an open form's review button. The send button's gate is not asserted, because the confirm step cannot be reached in a static render. No rendered check: no render check exists for these three screens. Copy and state review, ui-designer role (AI, by reading): FAIL with two findings (an already open P01 form could still send; the sentence said "below" although the facts it refers to are above it), corrected; corrective re-review PASS.

Left open: the header still labels the old time "Now (UTC)" on these screens and on P02 and P05; P03 and P04 unmount an open form when the gate closes, so a message inside it is lost; a disabled opener takes no focus, so its reason is rarely announced; the same finding on M01, M02, M03, O02 and O03 (A3, B3, C2, F1, G2) is not fixed here.

## O02 finding F1 and O03 finding G2 (2026-11-24)

- F1: when a reload of the change plans fails while a page is held, an alert after the header says that the changes could not be loaded, that everything shown is from the last successful load, and to reload before acting on it.
- G2: on O03 the list failure alert adds the same sentence when a page is held, and a failed refresh of a held detail now shows an alert with that sentence and a retry above the detail.
- Found in the render and fixed with it: the contract screen worded a load failure with the sign-in failure texts ("We could not sign you in…", "The sign-in service is not responding…"). It now says "Contracts could not be loaded." and uses the general service sentence.

These two screens keep their commands enabled in that state, so the sentence advises a reload instead of declaring the screen read-only.

Checks: `tsc` clean; console tests 1345, pass 1342, fail 0, skipped 3; `tools/o03-trust-states-qa.mjs` extended for the kept list and detail and for the wording of the load failure, passing in English, Korean and Arabic at 1440 and 390; the parts render check still passes. O02 has no rendered check. Copy and state review, ui-designer role (AI, from four renders and source): PASS as a candidate, no blocking finding; its suggestion of a title on the detail alert was applied afterwards.

Left open: on O03 the alert can be thousands of pixels above the "Check permission and review" button of a held detail, so disabling that button in this state, as on P01, P03 and P04, is recommended as a follow-up; the header facts and the present-tense labels of a held detail still read as current; the alert button says "Try again" where the sentence and the toolbar say "Reload"; a failed detail with nothing held shows only the plural title; whether the write paths of these screens refuse a stale revision was not traced.

## M01, M02 and M03 findings A3, B3 and C2 (2026-11-24)

- A3: on M01 below the wide layout, with a detail or action panel open, a failed reload now shows the load failure in the panel with the sentence that what is shown is from the last successful load, and a retry. Before, the error was rendered only in the hidden list.
- B3: on M02 the same failure is now rendered in the stepped detail as well, and the load failure adds that sentence whenever a page is held.
- C2: on M03 the effects view shows the load failure above the effects when a reload fails with a page held, with the same sentence.

Checks: `tsc` clean; console tests 1345, pass 1342, fail 0, skipped 3. No new test, no rendered check and no further review: the change places an existing alert in three more spots and reuses the sentence reviewed for O02 and O03. These states need a failed reload at a narrow width with a panel open, which no existing render check stages, so how they look is not verified.

Left open: commands on these three screens stay enabled in that state.

# Trace of the suspected "fake success" findings (2026-11-24)

The findings A4, E5, F3, G4 and H2 were suspected because screens show their success text when a command request resolves, without reading the status. Root traced the command layer and the commands named, by reading main at f78d319. Nothing was executed.

Facts:

| Fact | Evidence |
| --- | --- |
| A mutating command's result is either accepted, with a job identifier and no business state, or completed, with a valid result revision and a valid business state. Any other outcome is turned into an internal error, which the console receives as a failed request | `modules/commands/execute.go:184` to `:193`, `:421` to `:437` |
| Four command paths return accepted: approving an access request that revokes, revoking a delegation, executing a bulk revocation, and a trust activation whose outcome is unknown | `apps/api/server/accessrequests.go:347`, `revocations.go:145`, `bulk.go:129`, `trust.go:315` |
| The console words each of those as received and not as done: approving a revoking request and revoking a delegation say that the revocation was received and that its state is shown in the Revocations section; executing a bulk revocation hands over to the job view; trust activation reads the returned state | `access/request-copy.ts` (`doneApprovedRevoke`, `doneRevokedDeleg`), `access/Requests.tsx:79`, `users/BulkPrepare.tsx:153`, `authentication/TrustAfter.tsx` |
| Approving an access request that adds returns completed with the state "granted" | `accessrequests.go:345` |
| On O02, a completed activation always has the state "activated" and a completed discard "discarded" | `relplans.go:405` to `:451` |
| On P01, a completed command always returns the proposal number and the proposal's state | `installation.go:544` |

Result:

- A4 (M01, M02, M03, M04): refuted. For the commands on these screens a resolved request is a completed command, except the four accepted paths, which the console already words as received.
- F3: refuted for activate and discard. Not settled for propose, whose returned state is the plan's state and is treated as sealed unless it is "rejected"; the set of plan states at that point was not traced.
- H2 (P01): refuted for the missing proposal number and for withdraw. For approve the text shows the returned state, which is accurate. The P02 items were not traced.
- E5 (O01 draft save) and G4 (O03 contract actions): not settled; the fields and states returned by those commands were not traced.

So the pattern is narrower than the static check suggested: it does not hold on M01 to M04, and what remains is three command-specific questions (O02 propose, O01 draft save, O03 contract actions) and the P02 items.

## O01 finding E1 (2026-11-24)

When the read of a member's facts fails, the parent row of the detail no longer stays on "Loading…": it says "Could not be loaded." with a "Try again" link that repeats the read.

Checks: `tsc` clean; console tests 1345, pass 1342, fail 0, skipped 3. No new test, no rendered check and no review: one short sentence and the existing "Try again" label; no render check exists for O01 with a failing member read, so how it looks is not verified.

## P01 finding H3 (2026-11-24)

The release mark on P01 is now read as three cases, as P02 already does: an explicit null from the server is "None", a mark object is shown, and an absent or malformed value is "Unknown". Before, an absent or malformed value was shown as "None". The server sends an explicit null when a release has no mark (`apps/api/server/installation.go:139`).

Checks: `tsc` clean; console tests 1345, pass 1342, fail 0, skipped 3, with the installation parser test extended by a release without the field and one with a malformed value. No rendered check and no review: no new copy.

## M02 finding B1 (2026-11-24)

The cause was on the server. The trust change row always carried an impact object and left it at a default of zero members, zero sessions and "signs out everyone" when the change was not open or its facts could not be read (`apps/api/server/trustrows.go`). The read now sends `"impact": null` in that case, a change to a public read. The console keeps null apart from a computed zero: the impact section of an open change says "Not computed. This is not zero." and still shows the sign-out sentence, the confirmation rows say the same instead of 0, and the section is not shown for a change whose activation outcome is unknown, for which the server never computes an impact.

Checks: `go vet` clean; `go test ./... -count=1` 83 packages ok, 0 failed, with a new server test (an open change has an impact object, a cancelled one has null); console `tsc` clean and console tests passing with the parser test covering an absent value, an explicit null, an array and a computed zero. Independent change review (role-based AI reviewer separate from the writer, by reading): PASS with no blocking finding; it confirmed that no other reader of the field exists, that the approve and activate confirmations keep their sign-out sentence from static copy, and that approve and activate are refused when the facts read fails. Its corrections were applied afterwards and not reviewed again: an array is no longer parsed as a computed impact, the sentence no longer says "at this read", the sign-out sentence is kept in the not-computed state, and two documents that describe the field were updated.

Left open: no server test for an open change whose facts read fails; no render test of the not-computed state; `docs/features/HYB-297/visual-spec.md` has no not-computed state; B2 (other defaults on M02) is not fixed here. Observation for a separate ticket, not verified: a failed facts read on a validated change is reported as stale evidence.

# Trace of the "parser default" findings (2026-11-24)

After B1 turned out to be a server default and not a parser fault, root read the server side of the other findings of that pattern at main 4ec70b1. Nothing was executed and this section has no independent review.

| Finding | What the console defaults | What the server sends | Result |
| --- | --- | --- | --- |
| E4, F4 (O01, O02) | absent member counts and open limits become 0; an absent draft member limit becomes 1000 | `member_count`, `members_on` and `open_limit` are plain integers that are always serialised and computed from real data (`groupdrafts.go:225`, `:269`; `orgtargets.go:61`; `relplans.go:632`); the draft member limit is an installation setting that is always listed, with bounds 1 to 10000 (`orgsettings.go:29`) | the default applies only to a malformed response. Not a present defect |
| A6 (M01) | a membership state that is not "suspended" reads "active"; a role that is not "administrator" reads "member"; an account state that is neither "suspended" nor "departed" reads "Active" | the identity module has exactly these states and roles today | not a present defect, but the direction of the default is unsafe: a state added later, or a malformed value, would be shown as Active or as an ordinary member. Recommended hardening: an explicit unknown value in the parser and an "Unknown" label |
| B2 (M02) | an unrecognised change state reads "draft"; approvals needed fall back to 3; the step-up period falls back to 900 seconds | not traced field by field | not settled. The fallback for approvals needed and for the step-up period shows a number the server did not send, without saying so |
| H5 (P01, P02, P03) | empty strings shown as "Direct", "no verifier", "no series", "None" | not traced | not settled |

So of this pattern only B1 was a present defect, and it is fixed. A6 is a hardening of the direction of a default and is the one worth doing next, because it concerns how a person's access state is shown. E4 and F4 need no change.

## M01 finding A6, states only (2026-11-24)

An account state or membership state that the console does not recognise is no longer read as "Active". The membership parser has an explicit unknown value; the account label, the membership label and the role cells of the list say "Unknown state"; no account action is offered for an unrecognised account state and no membership action for an unrecognised membership state, each with a sentence saying why. The identity module sends only the known states today, so this changes nothing for a well-formed response. Roles are unchanged: an unrecognised role is still read as "member" by the parser.

Checks: `tsc` clean; console tests 1346, pass 1343, fail 0, skipped 3, with a new test for the labels and offers and a parser assertion. No rendered check. Copy and state review, ui-designer role (AI, by reading): FAIL with one finding (the note for an unknown membership state was grouped under sign-in), corrected with its other suggestions; corrective re-review PASS, with one more spot found (the wide role cell lacked the label), added afterwards and not reviewed again. The reviewer judged that offering no action for an unknown account state is the right default.

## O01 findings E2 and E3 (2026-11-24)

- E2: after a successful save or discard, a failed re-read of the draft is no longer shown as "This draft no longer exists". The panel has its own state for it and says that the request was recorded but the draft could not be read again, with the way back to the drafts. A draft that is really missing keeps the earlier text.
- E3: the existing-group search in the draft panel clears the earlier rows when a new search starts and shows that it is loading or that it failed, so results of an earlier term no longer stay under a new term.

Checks: `tsc` clean; console tests 1346, pass 1343, fail 0, skipped 3. No new test and no rendered check: no render check stages a failed read on the O01 draft panel. Copy review, ui-designer role (AI, by reading): PASS as a candidate.

Left open: whether the saved or discarded announcement is still spoken beside the new alert; a draft beyond the 40 pages that the lookup reads is still reported as missing.

## M03 finding C1 (2026-11-24)

On the effect form, when the page lists no kind of data the "Show effect" button is disabled, the start hint is hidden and a sentence beside the button says that no kind of data is listed and to reload the page. Before, the button was enabled once a person was chosen and submitting did nothing. The chosen kind is also derived from the current catalogue, so a kind that is no longer listed is not sent. The server always sends the catalogue (`access.go:190`), so this concerns a malformed response only.

Checks: `tsc` clean; console tests 1346, pass 1343, fail 0, skipped 3. No new test and no rendered check. Copy review, ui-designer role (AI, by reading): PASS as a candidate; its Korean wording was adopted afterwards.

## M05 finding D2, and D3 by reading (2026-11-24)

- D2: the longest session shown on M05 is read from the server. The self account read has a new required field `absolute_minutes`, the server's absolute session lifetime (720 today, a constant, not an installation setting); the console formats it with the same duration formatter as the other limits. For 720 the text is unchanged in the three languages.
- D3, not a present defect: the server sends only the `local-password` identity method (`self.go`), so the "Password" label is right today. A second method would need its own label.

Checks: `go build`, `go vet`, the whole `apps/api/server` test package pass; `tsc` clean; console tests pass with one new test for the formatted text. No rendered check. Independent change review (independent-reviewer role, AI, by reading): PASS, no blocking finding; its two suggestions (binding document, a test for the text) were adopted afterwards. It noted that a console built from this change fails the self read against an older server, which follows how the earlier required fields of this read were added, and that the unused `staleNote` copy still states 12 hours.

## M05 finding D4 (2026-11-24)

A referenced operation whose result is not confirmed is no longer reported as "The retained operation is unavailable". It has its own sentence: the result is not confirmed, so it cannot be shown yet; check again. The read's answer is the server's, so this is a stated state and not a failure.

Not changed: a failed read and a missing operation still share the "unavailable" text. The console's failure kinds do not tell a network failure from a missing record on this read (both arrive as "unavailable"), so they cannot be told apart without a change to the service client.

Checks: `tsc` clean; console tests 1347, pass 1344, fail 0, skipped 3. No new test and no rendered check. Copy review, ui-designer role (AI, by reading): PASS as a candidate; its recommendation to announce the unconfirmed state as a status and not an alert was adopted afterwards.

## M01 and M04 finding A5 (2026-11-24)

The one-time pickup of a sign-in link (M01) or a credential secret (M04) is reported as gone only when the service refuses it, which is the server's answer for a pickup that was consumed, expired or belongs to another session (`users.go`, `collect`). Any other failure (the service not answering, an internal error, a lost response) leaves the retry in place instead of telling the administrator to issue a new link or revoke the credential. If the pickup was in fact consumed, the retry is refused and the gone text follows. The M04 rule that allowed a single retry is removed for the same reason.

Checks: `tsc` clean; console tests pass with one new test for the rule. No rendered check. Independent change review (independent-reviewer role, AI, by reading): PASS, no blocking finding. Adopted afterwards: the M04 retry text says "Try again" in English and Korean, since the single-retry limit is gone (the Arabic already said so). It confirmed that a caller who lost installation administration is also told the pickup is gone, as before this change, and that a consumed secret whose body was malformed now takes one more click before the gone text.

## O02 finding F2 (2026-11-24)

The announcement on arrival ("Sealed", "Rejected", "Discarded", "Activated. Relationship revision n is now active.") is made only when the plan named in the address is in the loaded page and is in that state. The revision announced for an activation is the plan's own `activated_revision`, not the number in the address. An address typed or edited by hand no longer produces an announcement the page does not support. When the plan is not in the loaded page nothing is announced.

Checks: `tsc` clean; console tests pass with one new test for the rule. No rendered check and no copy change.

## P01 and P05 finding H4 (2026-11-24)

When a command is denied and the reload that follows fails, the command form on P01 (installation proposals) and P05 (release evidence) now shows the refusal ("This proposal is no longer available." and its variants) as an alert in the form. Before, the form ended with no message. This follows what the retention screen already does (`retention/LiveCommand.tsx`). With a successful reload the behaviour is unchanged.

Checks: `tsc` clean; console tests 1349, pass 1346, fail 0, skipped 3. No new test, no rendered check, no new copy and no review. Not settled: the text says the item is no longer available, which the failed reload could not confirm; the stale-page alert added for P01 in an earlier fix is shown beside it.

## M01 finding A7, excluded people (2026-11-24)

In the bulk revocation preview, a failed first read of the excluded people is shown under its heading with a sentence and a "Try again" button. Before, nothing was rendered under "Excluded: n".

Checks: `tsc` clean; console tests 1349, pass 1346, fail 0, skipped 3. No new test and no rendered check. Copy review, ui-designer role (AI, by reading): PASS as a candidate; its wording for the second sentence was adopted afterwards.

The other parts of A7 are not changed and remain suspected: a discard that fails with a stage error is reported as discarded (`users/Live.tsx`, `discardFixed`); a failed read of the pending fixed list on load is silent (`users/Live.tsx`, the `bulkClient(api).list` effect); an absent match count is read as 0 (`match-model.ts`, `fixedOfFields`).

## M02 finding B2, traced against the server (2026-11-24)

Read at main 4a17b1d; nothing executed; no independent review of this section.

| Console default | What the server sends | Result |
| --- | --- | --- |
| approvals needed fall back to 3 (`admin.ts`, `holders.needed`) | `needed` is always set from the server's own required number of people (`policy.go:183`) | applies only to a malformed response |
| the step-up period falls back to 900 seconds (`authentication/live-model.ts`, `effectiveStep`) | `step_up_seconds` is always set: the server's default, lowered by the active policy when there is one (`policy.go:186-195`) | applies only to a malformed response |
| an unrecognised change state reads "draft" (`admin.ts`, `trustStates`) | the eight states of the trust change module (`modules/trustchange/types.go:11-18`) are the eight the console lists | not a present defect. The direction is the same as A6: a state added later would be shown as a draft. The actions offered come from the server's `can` flags and not from the state, so the exposure is to the label and the emphasis only |

B2 is therefore not a present defect. A possible hardening, not done: read an unrecognised change state as its own unknown value, as was done for the account and membership states on M01.

## O01 finding E5 and O03 finding G4 (2026-11-24)

- E5, hardening: the server always returns the new draft's identifier from the create command (`groupdraftcmds.go:189`), so the "saved" announcement with an empty identifier does not occur with a well-formed response. With a malformed one the console would have sent the create command again for the next batch of members, making a second draft, and then announced "saved". It now stops after a completed create without an identifier, reloads the drafts and shows the text for a recorded request whose draft could not be read again.
- G4, not a defect: the O03 done notice states the result state that the command returned ("Final settlement: …", with "Unknown (x)" for a word it does not know); it does not claim a state of its own. The command layer returns a completed result only with a business state.

Checks: `tsc` clean; console tests pass. No new test, no rendered check, no new copy and no review.

## M01 finding A7, discard after a stage error (2026-11-24)

Confirmed by reading the server: `bulk_revocation.discard` is refused with a stage error when the list is no longer a draft or a prepared list, that is when it was already discarded, expired or run (`modules/bulkrevoke/registry.go`, `Discard`). The console treated that refusal as a discard and announced "The fixed list was discarded. Nothing was revoked.", which is false for a list that was run. It now shows, visibly on the page and once to the live region: "There was nothing to discard: this list was already discarded, expired or run. If it was run, the result is on Revocation jobs." The note is cleared on a view change, when a new fixed list exists and when a bulk run starts.

Checks: `tsc` clean; console tests pass. No new test and no rendered check. Copy and state review, ui-designer role (AI, by reading): FAIL with one finding (the text reached only the live region, so a sighted user saw the same as after a real discard), corrected; corrective re-review PASS; its three further suggestions (announce once through the live region, clear the note when a bulk run starts, follow a language switch) were adopted afterwards and not reviewed again.

Still open from A7: a failed read of the pending fixed list on load is silent; an absent match count is read as 0.

## O03 follow-up: contract actions while a held detail is stale (2026-11-24)

While the detail of a contract is held from an earlier load because its reload failed, the contract action buttons, the withdraw button of each event row, and the review and send buttons of an open command form are disabled and described by the stale alert, as on P01, P03 and P04. The form also refuses to review or send in that state. A successful reload enables them again.

Not covered: the writes of the setup sections inside the detail (documents, payment terms, issuance setup) are not gated by this change.

Checks: `tsc` clean; console tests 1349, pass 1346, fail 0, skipped 3. No new test, no rendered check, no copy change and no review; the pattern is the one reviewed for P01.

## Rendered regression run on O03 after the fixes (2026-11-24)

The seven existing O03 rendered checks were run against a console built from main b982afb (headless Chromium, built console with intercepted synthetic reads; no server, no mutation of real data, no owner acceptance). All seven ended with exit code 0 and no page error.

| Script | Cases | Unexpected reads |
| --- | --- | --- |
| `tools/o03-trust-states-qa.mjs` | 30 | 0 |
| `tools/o03-setup-changes-qa.mjs` | 72 | 0 |
| `tools/o03-setup-changes-writes-qa.mjs` | 27 | 0 |
| `tools/o03-setup-channels-qa.mjs` | 63 | 0 |
| `tools/o03-setup-channels-writes-qa.mjs` | 36 | 0 |
| `tools/o03-payment-terms-qa.mjs` | 99 | 0 |
| `tools/o03-payment-terms-writes-qa.mjs` | 27 | 0 |

Two scripts had fallen behind the screen and are brought up to date in the same change: `o03-setup-changes-qa.mjs` did not know the channel read and `o03-setup-channels-qa.mjs` did not know the reference read, both added to the documents view by later slices. Each now answers that read as denied, as it does for the other reads it does not stage, instead of counting it as unexpected. Their assertions are unchanged.

What this run shows: the O03 fixes of this ticket did not break what these scripts assert. What it does not show: none of the scripts asserts the newly disabled contract actions on a stale detail, and the other screens changed under this ticket (M01, M02, M03, M04, M05, O01, O02, P01, P03, P04, P05) were not run.

## Rendered regression run on M05, M03 and the rest of O03 (2026-11-24)

Run against a console built from main 5b5b90a (headless Chromium, synthetic reads, no server, no owner acceptance). Every script ended with exit code 0 and no page error.

| Script | Screen | Result |
| --- | --- | --- |
| `apps/console/tools/my-security-scope-qa.mjs` | M05 | pass |
| `apps/console/tools/my-security-history-qa.mjs` | M05 | pass, 61 checks |
| `apps/console/tools/my-security-selection-qa.mjs` | M05 | pass, 28 checks |
| `tools/issuance-authority-area-qa.mjs` | M03 | 84 cases; 13 entries counted as unexpected, all of them the page's own access read, which this script does not stage |
| `tools/o03-setup-parts-qa.mjs` | O03 | 54 cases |
| `tools/o03-setup-rules-qa.mjs` | O03 | 99 cases |
| `tools/o03-setup-rules-writes-qa.mjs` | O03 | 18 cases |
| `tools/o03-setup-statements-qa.mjs` | O03 | 72 cases |

`o03-setup-parts-qa.mjs` and `o03-setup-rules-writes-qa.mjs` counted reads added by later slices as unexpected; each now answers them as denied, as in the earlier update. Their assertions are unchanged. Whether the 13 entries of the M03 script were already there before this ticket was not checked.

Not run: `apps/console/tools/service-accounts-qa.mjs` (M04) needs a running server with a signed-in administrator. M01, M02, O01, O02, P01, P03, P04 and P05 have no rendered run in this record. None of these scripts asserts the states added under this ticket (the M05 unconfirmed operation, the M05 lifetime from the server, the M03 effect form without a kind of data).
