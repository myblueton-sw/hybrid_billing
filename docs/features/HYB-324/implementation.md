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
