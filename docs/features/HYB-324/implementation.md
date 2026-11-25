---
type: Record
title: Implementation record of removing chosen people from a filter list
description: What each step of HYB-324 built, how it was checked, what the reviews found and what is left.
status: draft
---

# Step 1, the module and the service (2026-11-25)

Built against [the design](design.md).

- `modules/bulkrevoke`: `Remove` marks people of the caller's own filter draft as removed by choice, keeping them in the draft; it ignores identifiers that are not in the draft and the caller's own, counts those already marked, and renews the draft's lifetime only when it marked someone. `DraftChosen` returns the marked people. `Prepare` refuses a prepare in which a marked person is among the targets. The view carries `Chosen`; the exclusion reason `chosen` is new.
- `apps/api/server/bulkstage.go`: the command `bulk_revocation.remove_targets` on a draft, with the one field `users`, under the definition shared by the bulk commands (bulk revocation permission, the owner only, a reason, the sign-in confirmation of every changing command). Result fields `draft`, `removed`, `already`, `ignored`, `chosen`.
- `apps/api/server/bulk.go`: at prepare, a marked person who was classified as a target becomes an exclusion with the reason `chosen`; a marked person excluded for another reason keeps that reason.
- `apps/api/server/bulkread.go`: the row of a filter list gains `chosen`; the exclusions counted by reason gain the key `chosen` for filter lists.
- `apps/api/server/trusthold.go`: the command is allowed during a trust hold, like the other bulk commands.
- The public description of commands is generated from the definitions, so the command appears in it without a separate edit.

Refusals as built: a draft of chosen people and a prepared list addressed by its own identifier answer `invalid_stage`; a draft of another person, an identifier that is not a draft and the staging target are refused before the handler by the resolution of the target; a malformed identifier or an empty `users` answers `invalid_request`.

Checks on the development machine (macOS arm64):

- `go vet` clean for the module and `apps/api`; the module test package passes.
- Module: `TestRemoveMarksPeopleOfAFilterList` (counts, the lifetime rule, the draft keeps everyone, the listing, other owner and workspace, malformed input changes nothing, a marked person among the targets is refused, the digest differs from the same list without the removal, a prepared list by either identifier), `TestRemoveIsForFilterListsOnly` (a draft of chosen people, an expired draft).
- Service: `TestRemoveTargetsLeavesChosenPeopleOutOfAFilterList` (through HTTP: the result counts, the row's `chosen`, a marked person who also stopped matching is shown as no longer matching, the counts by reason add up to the excluded total and to everyone matched, a prepared list is refused, and after execution the removed person is an excluded item with the reason and a target is not), `TestRemoveTargetsRefusals`, `TestRemovingAdministratorsLetsAFilterListBePrepared` (a list of administrators that cannot be prepared can be prepared after removals, and the removed people are excluded by choice; it does not show that one removal is enough). The trust hold table test lists 197 commands.
- `TestChosenIsCountedForFilterListsOnly`: a filter list without removals reports the key with zero, a list of chosen people does not report it.
- The whole `apps/api/server` package passes (433 seconds, thirty-minute limit), run before the last test was added; that test and the other new ones were run afterwards.
- Each rule removed or weakened in turn (13 changes): 12 made a test fail; the remaining one, the zero count for filter lists, was not covered and now is.

The console is not changed in this step: it does not send the command, and it would show the new reason as its code if one appeared, which cannot happen without the command.

Independent change review (AI, Claude Opus 5.5, by reading; it ran nothing), permissions and public contract, 2026-11-25: approve, no blocking finding. It confirmed that only the owner of an unfrozen, unexpired filter draft can mark anyone, that a person who is not the owner gets the same answer whether or not the list exists, that the reasons the service decides win over the choice, that the role and administrator checks rightly run over the remaining targets only, and that an ignored read error at prepare cannot turn a marked person into a target. Observations taken: a test was renamed because it did not show that one removal is enough; a test now pins that another person's draft and a draft that never existed answer alike; the design now says that the count is on the row of a filter list in every state. Left open: no test of the size limit of the field or of a trailing empty line for this command (both behave as for adding chosen people, by reading); after a restart a replayed job shows neither its filter nor this count, as before for the number matched; the glossary line that calls this ticket open is updated with the console step.

# Step 2, the console (2026-11-25)

- `users/Matching.tsx` `FixedBar`: "Exclude people from this revocation" between "Prepare" and "Discard list"; in that mode a hint, "Exclude N (they keep access)" (shown as "Select people to exclude", disabled and muted, with nobody selected) and "Stop excluding"; the line "Excluded by you, keeping access: N" whenever anyone is marked or the mode is on; a line with the last result, announced once through the screen's announcer.
- `users/Selection.tsx`: in that mode the checkboxes are named "Keep access for …" and the caller's own cell says that the caller's access is never revoked by the list.
- `users/Live.tsx`: the people selected for removal are their own state, never the selection used for revoking chosen people, and are cleared whenever the fixed list changes or goes away (discarded, prepared, closed, expired, fixed again). The selection column is on while a list is fixed only in removal mode. The command is sent in batches under the size limit of a field; the answers are added up; a list that turns out closed is handled as elsewhere on the bar.
- `users/BulkPrepare.tsx`: the preview and the confirmation dialog show, under the excluded count and as part of it, "including N you excluded, who keep their access" when anyone was excluded by choice; the reasons of a filter list include the new one; the refusal for a list that would leave no administrator says that keeping one administrator on the list also resolves it and offers "Back to the list" before "Discard list".
- `service/identity-bulk.ts`, `users/match-model.ts`: the row's `chosen` is read and restored with the fixed list after a reload.
- Earlier documents that said this control does not exist are annotated (`HYB-323/visual-spec.md`, `HYB-323/plan.md`, `HYB-294/plan.md`, the glossary line).

Differences from the design: the screen says "exclude", with "keep access", where the design and the command say "remove", because on a revocation screen "remove" reads as taking access away. The command's reason is a fixed sentence ("Excluded from the list on the revocation screen"), as for discarding a list from the bar; the design said the screen asks for a reason. The reason asked for at prepare and at execution is unchanged.

Checks on the development machine:

- `tsc` clean. Console tests: 1390, pass 1387, fail 0, skipped 3 (four added: the removal answers adding up and the restored count; the bar in every state of the mode in three languages; the preview's line and reason with the number agreement of the new sentences; the names of the checkboxes in the mode).
- Rendered check against a real local service with the sample data (`tools/bulk-removal-qa.mjs`, new; headless Chromium; English, Korean and Arabic; the whole flow at 1440 and the removal mode also at 1024 and 390): 53 passed, 0 failed, run again after the corrections below. It fixes a list of everyone, checks that the list has no checkboxes, enters the mode, checks the names of the checkboxes, that stopping hides them and that starting again begins with nobody selected, that an empty selection sends nothing, excludes one person, checks that the command was sent once, that the bar counts one and takes the focus, reloads the screen and finds the count again, prepares the list and finds one person counted as keeping access. At the end it picks two people in the mode, stops, discards the list and checks that nobody is selected for revocation. Not an owner acceptance.

Seen while writing the rendered check, and correct: removing two of the four sample people left nobody who could be revoked, and the prepare was refused with the existing sentence for that case.

Not covered by the rendered check: a selection too large for one request, a person selected who is not in the list, a failure between two requests, the refusal for a list that would leave no administrator (the sample caller remains one), the confirmation dialog.

The planning line `ui-design.md:421` (targets, exclusions and base revision fixed in a server manifest) was recorded as partly covered in `HYB-323/plan.md:59` and `:347`; with the exclusions chosen by the person built, it is covered, and those two lines are annotated.

Reviews of step 2 (AI, Claude Opus 5.5, by reading and, for the designer, from screenshots; neither ran anything), 2026-11-25:

- Independent change review: approve with conditions. From the code it found the central risk handled: nothing copies the people picked to keep access into the selection for revoking, the removal picks are not kept across reloads and are cleared on every change of the fixed list, and the counts shown come only from the service's answers. Two conditions, done: no test guarded that separation (the rendered check now does, see above); the status of the planning line was not recorded (above). Taken as well: the shared limit warning is reset with the list. Left open: people picked while a request is being sent are not sent and are cleared on success; the sentence about the limit of a selection still speaks of revocation in this mode.
- ui-designer review: FAIL with five blocking findings, all corrected: "remove" could be read as taking access away (the labels now say exclude and keep access); the checkboxes gave a screen reader the same name in both modes; the count of people kept stood beside the other counts as if they added up (it is now a line under the excluded count); number agreement in English and Arabic; the disabled button looked enabled. Taken as well: focus goes to the count line after excluding, one announcement instead of two, the wording and order of the dead end. Both re-reviews of the corrections: approve and PASS. Two small points from them taken afterwards (a Korean term, and where the focus goes when nobody ended up excluded). Left for a follow-up: the preview uses two Arabic roots for "excluded" side by side, one of them older than this change. Left for the owner: no row shows that it was excluded before the list is prepared; "Prepare revocation" is not visually the primary action; the Arabic of the new sentences is not certified by the reviewer.
