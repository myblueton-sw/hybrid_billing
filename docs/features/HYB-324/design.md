---
type: Design
title: Removing chosen people from a filter list of a bulk revocation
description: Facts about the M01 bulk revocation by filter, the command proposed for leaving chosen people out of a fixed filter list, how the choice reaches the manifest and its digest, the screen changes, limits and steps.
status: draft
---

# Outcome and ownership

HYB-324, machine:mac_mini, 2026-11-25. This document proposes one new command and the screen change that uses it, and asks for independent design review before any code changes. The command changes who a bulk revocation reaches, so the review covers permissions and the public contract. It grants no new permission: it can only make a list smaller.

Requirement: the bulk table contract of the planning document, `docs/planning/v1.16/ui-design.md:421`: a selection of everything a filter matches fixes its targets, its exclusions and its base revision in a manifest on the server. HYB-294 and HYB-323 built the targets, the base revision and the exclusions the service decides; exclusions chosen by the person were left as an open finding with this ticket as its follow-up (`docs/features/HYB-294/plan.md:328-332`, `:746-750`; `docs/features/HYB-323/plan.md:59`, `:311`).

# Facts read on main 3745e4c

1. A filter list is made by `bulk_revocation.add_matching`: the service walks everyone, keeps those the filter matches, and stores them in a draft with the filter, the number matched, the time and the revision (`apps/api/server/bulkstage.go:93-125`, `modules/bulkrevoke/registry.go:150-183`). `add_targets` refuses a filter draft (`registry.go:125-126`), and nothing removes a person from any draft.
2. `bulk_revocation.prepare` classifies every person of the draft into a target or an exclusion with a reason, in this order: the caller (`self`); unknown, not a person, departed, or without a membership when the whole person is to be suspended (`not_found`, with the login blanked); for a filter list, no longer matching (`no_longer_matches`); no membership (`no_membership`); already without effect (`already_revoked`) (`apps/api/server/bulkimpact.go:140-167`). The module requires that targets and exclusions together are exactly the draft's people (`modules/bulkrevoke/manifest.go:86-100`).
3. The digest covers, for a filter list, the filter, the number matched, the time and the revision, then every target, then every exclusion with its user, login and reason (`manifest.go:31-60`). Execution requires the digest the person saw (`manifest.go:118-132`), holds any target that changed since (`apps/api/server/bulk.go:68-73`, `:100-120`) and records every exclusion as an excluded item with its reason (`manifest.go:158-160`).
4. Prepare is refused when the targets would leave the workspace without an administrator who can restore access (`bulk.go:50-52`, `apps/api/server/periods.go:50-70`); the rule looks at the targets only. For a list of chosen people the screen then lists the administrators with "Remove from selection"; for a filter list it can only say to narrow the filter or discard the list (`users/BulkPrepare.tsx:343-373`; `docs/features/HYB-294/plan.md:311-314`).
5. Every bulk command needs the bulk revocation permission and installation administration, is limited to the list's owner after staging, needs a reason, and needs a recent sign-in confirmation like every changing command that is not exempted (`bulkstage.go:47-62`, `:143-156`; `apps/api/server/stepup.go:42`). During a trust hold a command is refused unless the hold table lists it; the five bulk commands are listed as allowed (`apps/api/server/trusthold.go:149-153`, `:274-277`).
6. The reads: the list of bulk revocations with each row's counts and, for a filter list, the filter, the number matched, the time and the revision (`apps/api/server/bulkread.go:34-72`); one bulk revocation with its exclusions counted by reason (`:260-282`); its targets and its excluded people in pages (`:355-398`). No read returns the people of a draft.
7. On the screen, while a filter list is fixed, the table's selection column is off and a bar offers "Prepare" and "Discard list" (`users/Live.tsx:724`, `:821-830`; `users/Matching.tsx:105-134`). After prepare the preview shows the counts, the exclusions by reason and the excluded people (`users/BulkPrepare.tsx:35-123`). An exclusion reason the console does not know is shown as its code (`users/bulk-copy.ts:147-149`).

# Proposed

## Command

`bulk_revocation.remove_targets`, under the same permission, ownership, reason and sign-in confirmation rules as the other bulk commands (fact 5). It is added to the trust hold table as allowed, like them: it narrows a staged reduction and changes no access.

- Target: the draft. Field `users`: identifiers, one per line, as in `add_targets`, within the existing size limit of a text field (4096 bytes); the console sends larger selections in several requests, as it does for `add_targets`.
- Accepted only for a draft made by `add_matching` that is still a draft and belongs to the caller. A draft of chosen people answers `invalid_stage`. The draft identifier of a list that was prepared no longer resolves and answers `permission_denied`, like any identifier the caller cannot use; a prepared list addressed by its own identifier answers `invalid_stage`. A frozen list is changed only by discarding it.
- An identifier that is not a person of the list, and the caller's own identifier, are ignored and counted. Nothing is refused for them: while a list is fixed the table can show people who are not in it (people who started matching later, or other filters), so they are a normal part of a selection. The answer does not say which identifiers were ignored, and an identifier that is in nobody's list is indistinguishable from one that does not exist.
- The result carries `draft`, `removed` (newly marked by this request), `already` (were marked before), `ignored` (not in the list, or the caller) and `chosen` (all marked so far in the list).
- The mark is kept in the draft's own set of people, as a value on the entry that already exists, so a draft holds no more entries than before. The person is not deleted from the list, and the number matched does not change.
- A mark cannot be undone on the same list. Discarding the list and fixing it again takes a new snapshot and loses every mark. A restore command is not proposed; it is an open point for the owner.
- The command renews the draft's lifetime only when it marked someone new.

## Manifest

- At prepare, a marked person who would otherwise be a target becomes an exclusion with the new reason `chosen`. The classification order becomes: `self`, `not_found`, `no_longer_matches`, `no_membership`, `already_revoked`, `chosen`. Every reason before the last states a fact that holds whatever the caller chose; `chosen` is therefore exactly the number of people who keep their access only because the caller removed them.
- The module refuses a prepare in which a marked person is among the targets. No path leads there, because commands run one at a time; the check is a second barrier.
- Targets and exclusions still add up to everyone matched (fact 2). The digest already covers every exclusion with its reason (fact 3), so a list prepared after a removal has a different digest from one prepared before, and execution needs the digest that was shown.
- The administrator rule of fact 4 keeps looking at the targets only: it refuses only when every administrator who can restore access would leave, and it is checked again at execution. Removing one of them from the list keeps that administrator, which is what the rule protects, and lets the list be prepared.
- Execution records a person removed by choice as an excluded item with the reason `chosen`, like every other exclusion.

## Reads

- The row of a filter list in the list of bulk revocations gains `chosen`, the number of people marked, in every state of the list; like the other counts of a prepared or executed list it is visible to the administrators who can read that list. The exclusions counted by reason gain the key `chosen` for filter lists only, as `no_longer_matches` is today. The excluded people read returns the reason `chosen`. These are a new field and new values in existing reads.
- No read of a draft's people is added. Who was removed is shown after prepare, in the excluded people of the preview.

## Screen

- While a filter list is fixed, the bar gains "Remove people from the list". It turns the table's selection column on for that purpose only. The people selected for removal are kept apart from the selection used for revoking chosen people, with their own limit, and are cleared when the list is discarded, prepared, closed, expired or fixed again, so that people chosen to keep their access can never become a selection to revoke.
- The bar shows how many are selected and "Remove N from the list", which asks for the reason and sends the command. Afterwards it says how many were marked and, when any were ignored, that so many were not in the list and were left as they are. The bar always shows the number marked so far.
- The number on the bar is the number marked. The preview's number for "removed by you" can be smaller, because a marked person who is excluded anyway is shown under that other reason.
- When prepare is refused by the administrator rule, the text for a filter list says that the list would leave nobody who can restore access, and that removing one such administrator from the list, or narrowing the filter, resolves it. The screen does not list those administrators: it has no read of a draft's people, and the refusal carries no names.
- The preview shows the count of people removed by the caller worded as people who keep their access, in the counts and again in the dialog that confirms execution, and lists them among the excluded with that reason. The preview's list of reasons gains the new one so that the counts by reason add up to the excluded total.
- The note under the bar no longer says that people can only be chosen one by one after discarding.
- Copy and composition go to the ui-designer; the owner alone accepts the screen.

# Limits and open points

- A removal cannot be undone (see the command). Open for the owner: whether a restore is wanted.
- Before prepare the screen shows only the number marked, not who; the people are listed after prepare. A read of a draft's marked people could show them earlier, and an answer naming the administrators that block a prepare would let the screen offer them directly; neither is proposed, because each adds to the public contract.
- The command walks no population and reads nothing but the draft.
- Nothing here changes what the filter matched or re-runs it at execution; that residual of HYB-323 stays (`docs/features/HYB-323/plan.md:329`).
- Not verified here: behaviour on operating systems other than the development machine.

# Steps

1. Module and server: the removal in the draft, the classification, the command with its refusals, the read values, the public description of the command, the trust hold entry, tests (including that the digest differs, that totals still add up, that another person's draft and an explicit draft are refused, that identifiers outside the list and the caller's own are ignored and counted, that a marked person is never a target, and that the administrator rule passes once one such administrator is removed). Independent change review (permissions, public contract).
2. Console: the bar, the selection for removal, the administrator dead end, the preview reason, tests, a rendered check in three languages, ui-designer review, independent change review.
3. Binding documents with each step. The earlier documents that say this control does not exist are updated in step 2: `docs/features/HYB-323/visual-spec.md:64`, `:200`, `:372`, `:452`; `docs/features/HYB-323/plan.md:152`; `docs/features/HYB-294/plan.md:311`, `:328-332` (the open question becomes built); and the cross-check of `ui-design.md:421`, which moves from partial to covered.

# Design review

Independent design review (AI, Claude Opus 5.5, by reading; nothing executed), 2026-11-25: approve with conditions. All seven facts confirmed, with two corrections taken. It judged the core sound: a removal can only turn targets into exclusions, the digest covers it, only the owner can use it, the administrator rule keeps its meaning, and a self-exclusion cannot be hidden. Four blocking findings, incorporated above: the command needs its trust hold entry; the screen has no data to list the blocking administrators for a filter list (the list is dropped and the text names the remedy); refusing a whole request for one person outside the list would be the normal case and could not be fixed by the person (such identifiers are now ignored and counted); the people selected for removal must be separate state so that they cannot turn into a selection to revoke. Taken as well: the new reason is classified last, the caller's own identifier is ignored, the mark adds no entries, the module's second barrier, the lifetime rule, the wording of the count as people who keep their access, the wider list of documents to update. Not verified by the reviewer: whether the walk that fixes a filter list can include people the caller's own users read hides. The corrected text has not been read again by the reviewer; the change reviews of the two steps cover it.
