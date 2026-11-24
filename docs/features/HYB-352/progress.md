---
type: VerificationRecord
title: Lists capped at 50 rows shown as complete, what was corrected without a contract change
description: Which cases of HYB-352 needed no change to a public read and were corrected, how they were checked, and what remains for a decision.
status: draft
---

# Outcome and ownership

HYB-352, machine:mac_mini, 2026-11-24. Two of the P05 cases are corrected in the console alone, because the lists already carry a total and a "more" flag and the screen already has a line for it. The other cases are unchanged and still need the decision named in the ticket.

# Corrected

| Place | Before | Now |
| --- | --- | --- |
| P05 path detail, test cases of each requirement (`release-evidence/LiveMatrix.tsx`) | the list of at most 50 tests had no limit line, so a requirement with more tests looked complete | the existing line follows each list: "Showing 50 of n." when the total is known, "There may be more." when it is not, nothing when the list is complete |
| P05 requirements, build comparison (`release-evidence/LiveRequirements.tsx`) | "No requirement changed state between these builds", or the difference table, was computed from at most 50 requirement rows and read as covering all | when the requirement list is not known to be complete, a line above the result says: "This comparison covers only the requirements listed on this screen, which may not be all of them." |

Checks: `tsc` clean; console tests 1364, pass 1361, fail 0, skipped 3. No new test and no rendered check: the sample data has fewer than 50 rows in every one of these lists, so neither line appears with it. Independent change review (independent-reviewer role, AI, by reading; nothing executed): pass, no blocking finding; its optional Korean wording was adopted afterwards. The reviewer that first found these cases could not do this review because of a usage limit, so another model did it.

# P05 decisions, corrected in a second step

When the list of proposals may be incomplete (the service says there are more, or does not say):

- the two headings mark their numbers as counts of what is shown: "Waiting for approval (n shown)" and "Closed (n shown)";
- a line under the first heading says "Older proposals waiting for approval may not be listed.";
- an empty list says "No proposals are shown." and not "No proposals are recorded.".

With a complete list the headings and the empty text are unchanged.

Checks: `tsc` clean; console tests 1364, pass 1361, fail 0, skipped 3. No new test and no rendered check, for the same reason as above. Copy and state review (ui-designer role, AI, by reading): fail with one finding (the empty text still claimed the whole list), corrected, and its recommendation for the line about older proposals adopted; re-review pass.

The reviewer's remark stands as part of the decision: marking the count removes the false total, but a person who must act on waiting proposals is only fully served when the service returns every waiting proposal whatever its age. That is a change to a public read.

# Still open on this ticket

Unchanged, from the ticket:

- P05: the pending-retirement link to a proposal outside the newest 50 (suspected).

Found by the change review in the same two files and not changed here:

- the build comparison's "Fewer than two builds are recorded", its build options and its note about the newest 10 builds are also derived from the capped requirement rows;
- the path detail's text for a requirement that is not among the loaded rows already says that its test cases "are not in this read", which is accurate; nothing to change there.

The decision named in the ticket still stands for all of these: either the server sends what the console needs to say more than "there may be more", which changes public reads and needs independent design and change review, or each such place states the limit.

# P05 matrix, read again: no change needed (2026-11-24)

The change review listed two matrix sentences as derived from the capped list of paths. Read at main ae40ae1 (`release-evidence/LiveMatrix.tsx`, `MatrixView`), neither states something about the paths that were not loaded:

- The sentence about missing execution evidence is shown when any loaded path is blocked. Its second half, that no support combination can be activated and that activation is not connected in this build, is true of the build whatever the paths are. When no loaded path is blocked the sentence is absent, which claims nothing.
- The intake note says that customer proof intake is not connected, so the count cannot rise in this build. It is shown when every loaded path reports intake as not connected, and it is a statement about the build. An earlier review of this screen found that an active path, a proven customer state and a ready gate cannot occur in this build.

Both would need another look when activation or customer proof intake is connected, because from then on they would describe the paths and not the build. Nothing was executed for this section and it has no independent review.

# P01, the preview basis (2026-11-24)

Confirmed by reading and corrected. The server chooses the previewed version from all proposals, of which up to 128 can be retained (64 waiting and 64 closed, `modules/installer/registry.go:238`, `:274`), and a read lists the 50 newest. The console guessed from those 50 why the version was chosen, so a waiting plan older than 50 newer entries made it say "because it is the newest release not installed" although the waiting plan was the reason.

The server now states the basis: the installation read has the field `preview_basis` with `pending_plan`, `newest_release` or an empty value when nothing is previewed, and the console shows what it is told. Without a served value the console says nothing about the basis and no longer guesses.

This is an added field of a public read. The installation read is not in the OpenAPI document; the screen binding document (`docs/ui/screen-binding-fields/operations.md`) is updated.

Checks: `go build`, `go vet` and the whole `apps/api/server` test package pass, with a test of a waiting plan hidden behind more entries than a read shows; `tsc` clean; console tests pass, with tests of the parser and of the basis. No rendered check. Independent change review (independent-reviewer role, AI, by reading): pass with one condition, the binding document, corrected; its suggested parser test was added. Neither was reviewed again.

Not changed, noted by the review: when the preview is not available the basis sentence is still shown beside the reason for that, as before this change.

Confirmed by the same review and still open: the server does not refuse an identical waiting plan from the same person; it answers with the existing proposal. When that proposal is outside the 50 listed, the console offers "Propose", and after sending it says that the plan was proposed as that proposal, as if a new one had been created, while the reason the person typed is dropped and the stored one kept (`modules/installer/registry.go:224-227`; `installation/LiveProposals.tsx:391`). Telling the person so needs the server to say that the answer is an existing proposal, which is another added field and is left for the decision of this ticket.

# P02, the delivery list (2026-11-24)

Confirmed by reading and corrected. A subscription can hold up to 100 dead letters (`WebhookLimits`), a read lists its first 50 deliveries in the order open, queued, dead, cancelled (`modules/webhooks/deliveries.go:457-465`), and the screen said only "Showing up to 50 deliveries per subscription", so a dead letter beyond the first 50 was absent with no sign that the list was cut.

The server now says for which subscriptions the list was cut: the events section of the integrations read has the field `deliveries_more`, a list of subscription identifiers, empty when nothing was cut. The console shows a line under the existing note: for a cut list, that more deliveries exist than are listed and, because of the order, dead letters may be among those not shown; when the service does not say, that it did not say. The sentence after a denied approval of a dead letter that is not in the list no longer says that the letter "is gone" but that it "is no longer listed".

This is an added field of a public read. The integrations read's section is not itemised in the OpenAPI document; the screen binding document is updated.

Checks: `go build`, `go vet` and the whole `apps/api/server` test package pass; `tsc` clean; console tests 1366, pass 1363, fail 0, skipped 3, with a test of the model and the parser. No rendered check. Independent change review (independent-reviewer role, AI, by reading): pass with one condition, the binding document, corrected; its suggestions for the Arabic terms, for hiding the line when the list itself is unknown and for the sentence about the letter were adopted. None of these was reviewed again.

A gap the review accepted and that remains: the function that cuts the list is tested, and the read of the sample is tested to report nothing as cut, but no test fills a real subscription past 50, because that needs a signed delivery round the tests cannot reproduce. A change of the request for one row more than is shown would not be caught by a test.

Not changed: the list still holds only the first 50, so a dead letter beyond them cannot be acted on from this screen; changing the order or paging the list is part of the decision of this ticket.

# P01, a plan that was already proposed (2026-11-24)

Corrected. When a person proposes a plan that the same person already has waiting with the same content, the module answers with the existing proposal. The command result now says so with the field `existing: "true"`, absent for a new proposal, and the console decides from that field alone, no longer from the 50 listed proposals. Its text is: "This plan was already proposed as #n. No new proposal was created, and that proposal keeps its original reason."

Deciding from the served field also removes an older wrong case: when a first attempt had in fact succeeded and its retry replayed after the page reloaded, the console found the new number in the list and called a new proposal a duplicate.

Checks: `go build`, `go vet` and the whole `apps/api/server` test package pass; `tsc` clean; console tests pass. No rendered check. Independent change review (independent-reviewer role, AI, by reading): pass with two conditions, the wording (the first version said that nothing new was recorded, although the command log records the attempt) and the binding document; both corrected, and its suggestion to decide from the served field alone adopted. Not reviewed again.

Not changed: the "Propose" button is still offered when the person's identical plan waits outside the listed proposals; pressing it now gives the accurate text. The same pattern exists on other screens, which decide "already proposed" from their own listed rows (checked later, see the last section): P03 (`recovery/LiveResume.tsx:67`), P04 (`retention/LiveHolds.tsx:146`, `LivePolicies.tsx:154`) and the release evidence screen.

# P02, a module change that was already proposed (2026-11-24)

Corrected in the same way as P01. The result of `integrations.plugin_propose` carries `existing: "true"` when the answer is a proposal that already existed, and the console decides from that field alone, no longer from the 50 listed proposals. Its text now adds that no new proposal was created and that the proposal keeps its original reason.

The subscription proposal of the same screen was not changed and does not need it: the read lists every subscription without a cap (`integrations.go`, `channelsAndEvents`), so the comparison with the listed subscriptions is complete. The review confirmed this by reading.

Checks: `go build`, `go vet` and the whole `apps/api/server` test package pass; `tsc` clean; console tests pass. No rendered check. Independent change review (independent-reviewer role, AI, by reading): pass with conditions, the binding document, an unused import, the unused branch of the snapshot helper with its test, and a test of the new decision; all corrected. Not reviewed again.

The other screens named by the review are settled in the next section.

# P03, P04 and release evidence: the duplicate decision (2026-11-24)

Checked by reading; not a defect, nothing changed. On these screens the console also decides "already proposed" from its listed rows, but the server never answers their proposal commands with an existing proposal. Each command is refused with `invalid_stage` while a proposal for the same object waits, whoever proposed it, before the module is asked:

| Screen | Command | Refusal while one waits | Test |
|---|---|---|---|
| P03 | `recovery.propose_unlock` | `recoverycmds.go:141` (`pendingUnlock`) | `recovery_commands_test.go:73-74` |
| P04 | `retention.propose_policy` | `retentioncmds.go:228` (`pendingPolicy`) | not looked for |
| P04 | `retention.propose_release` | `retentioncmds.go:328` (`pendingRelease`) | `retention_commands_test.go:138-139` |
| Release evidence | `release_evidence.retire_profile` | `evidencecmds.go:183` (`pendingProfileRetire`) | `evidence_commands_test.go:83-84` |
| Release evidence | requirement retirement | `evidencecmds.go:169` (`pendingRetire`) | not looked for |

The servers' stores drop only entries that are no longer waiting (`keep` in `recoveryseed.go`, `retentionseed.go`, `evidenceseed.go`), so a waiting proposal is always found by these checks. The opposite error, a new proposal called a duplicate after a reload, does not occur either: the comparison uses the rows listed before the first attempt.

Consequence: the sentence "This was already proposed as #n." of these three screens cannot be shown today. It is left in place; it would become reachable, and incomplete beyond 50 rows, only if one of these refusals were removed.

Limits: read, not run; no independent review of this record.

# Rendered check of the added states, and a correction of the P02 sentence (2026-11-24)

`apps/console/tools/capped-lists-qa.mjs` is new. Run against the console of this change and the API binary of main 6fd8bce (dev server, loopback, sample seed, a data directory the service created, synthetic data only, stopped afterwards): exit code 0, 25 checks passed, none failed, no page error. One run of the final script; the script was not reviewed.

P02, in English, Korean and Arabic at 1440 and 390: the service states that no delivery list was cut and the screen then shows no line; with a subscription named as cut, the line is shown with its full text; with the field left out, the line says that the service did not say. The last two were produced by changing the read's answer in the browser.

P01, in English at 1440: a plan is proposed through the screen and is new; with that proposal left out of the read's answer in the browser, so that the screen offers the action again, the same plan is proposed a second time. The service itself answers with the existing proposal and `existing: "true"`, and the screen says "This plan was already proposed as #8. No new proposal was created, and that proposal keeps its original reason." Only the listing was altered; the answer to the command is the service's own.

The run showed a wrong statement in the sentence merged earlier for the cut list. It said that the deliveries "are listed in the order open, queued, dead letter, cancelled". The service takes the 50 of a subscription in that order (`modules/webhooks/deliveries.go:457-465`), but the read then sorts them by sequence (`apps/api/server/integrations.go:793`), and the screenshot showed a dead letter above a waiting delivery. The sentence now reads: "More deliveries exist than are listed here. Open and queued deliveries are included before dead letters, so some dead letters may not be shown." Copy review (ui-designer role, AI, by reading the copy and term files; the two facts about the order came from the writer's brief): pass, with this shorter wording proposed by the reviewer and adopted in the three languages.

Checks for the copy change: `tsc` clean; console tests 1366, pass 1363, fail 0, skipped 3; the script above run with the new text.

Not covered by the script: the P02 module proposal that already exists, the P05 states, the P01 preview basis, Korean and Arabic for the P01 text, and keyboard order. Not an owner acceptance.

