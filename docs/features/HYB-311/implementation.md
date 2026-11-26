---
type: Record
title: Implementation record of the guard for duplicate-prevention records
description: What each step of HYB-311 built, how it was checked, what the reviews found and what is left.
status: draft
---

# Step 1, the rule in the retention register (2026-11-25)

Built against [the design](design.md), G1 to G7, in `modules/retention` only.

- `Guards` is a third reference kind and `guarding` a ninth outcome. A guard is accepted from a claim-class artifact to an artifact of another class that shares a tenant, by the system actor, while neither side is damaged, being deleted or deleted (`records.go`).
- `guarding` (`changes.go`): a claim artifact is guarding while any artifact it guards is not deleted or its stores' backups have not expired past the deletions. The reason is checked after a hold and before a requirement, wherever the others are checked: when a plan is applied, at a lease, at a retry, in the preview, in the impact of a policy change and in the standing. `Proceed` answers no for a guarding artifact.
- `anchor` (`changes.go`): the due state and the due time of an artifact count from the later of its anchor and the latest deletion time of what it guards, and are unsettled while anything it guards is undeleted. The due test, both sides of the impact and the standing's due time use it.
- `ImpactView`, `PreviewView` and `Standing` gain a guarding value. The run counts list the new outcome.

Not reachable through the register's interface: `Proceed` answering no because of a guard. A guard cannot be written for a record that is being deleted, and gone cannot be undone, so a record that was leased cannot start guarding afterwards. The check is kept as a second barrier and its test sets the guard in the register's memory directly.

Checks on the development machine (macOS arm64):

- `go vet` clean; the whole module test package passes.
- `TestGuardShape`, `TestGuardKeepsTheRecordUntilItsSourceIsGone` (a source with two stores: existing, damaged for a long time, deleted with live backups, one store's backups expired, gone; the period counted from the deletion; the record then deleted and both identifiers refused afterwards), `TestGuardPeriodRunsFromTheLastDeletion` (two guarded sources, a record younger than the deletion, the impact of a shorter policy with its guarding count), `TestGuardReleasesAPendingRecordAtTheLease` (lease and retry), `TestGuardReasonOrder`.
- `TestGuardProperty`: 900 generated sequences of clock steps, backup expiries, guards, holds, plans for both classes, stops, leases, retries, go-aheads and store reports that succeed or fail. Whenever the register leases a record or answers a go-ahead for it, the test checks from its own bookkeeping that everything the record guards is gone and that the period has passed since the later of the record's anchor and the last deletion; after every step it compares the standing's guarding value with its own. The test fails if fewer than 100 leases or go-aheads concerned a record with a guard. Each sequence keeps one claim policy; a change of policy during a sequence is not generated.
- Each rule removed or weakened in turn (17 changes): 16 made a test fail; the remaining one, the order of the reasons, was not covered and now is (`TestGuardReasonOrder`).

Not checked: any measurement of cost with many guards per record (the guarding test walks a record's guards on every check); other operating systems.

# Step 2, the service and the console (2026-11-25, same change)

The two steps are one change, because the server's tests pin the lists of outcomes and would fail on the module alone.

- `apps/api/server/retention.go`: `guarding` in the list of standings and of run outcomes; an artifact that is not in a run is shown as guarding before no policy, not due, held and referenced, so that a record whose class has no policy is still shown as guarding; a guarding artifact counts as excluded in the per-store counts. No field is added to the read.
- `apps/api/server/retentionseed.go`: the sample claim record guards the first sample source, written before that source's run is leased. In the sample that source ends deleted with no backup expiry recorded, so the record is shown as guarding. No claim policy is added.
- Console: the two key lists and the two label tables gain the value in three languages (`service/retention.ts`, `retention/live-codes.ts`); the strips of counts have eight and nine cells.
- `apps/console/tools/retention-qa.mjs` is new: it opens the expiry and deletion views in three languages at three widths.

Checks: `go vet` clean for `apps/api`; the server's retention tests pass with their pinned lists updated (eight standings, nine outcomes, the sample record guarding instead of without a policy, two references written by the sample data); the whole `apps/api/server` package passes in 577 seconds with a thirty-minute limit, run alone after every Go change. A first run, made while the rendered check was running on the same machine, was ended by the default ten-minute limit and is not a result; the package took 366 seconds earlier the same day, so its run time on this machine is now close to the default limit, which is noted for whoever runs it next. `tsc` clean; console tests 1386, pass 1383, fail 0, skipped 3 (none added: the label tables have no test of their own, the rendered check reads the labels). Rendered check against a real local service with the sample data (`tools/retention-qa.mjs`): 82 passed, 0 failed, run again after the labels changed; it checks that no value is shown as unknown, that the guarding label appears in both views and that exactly one class counts one guarding record. Not an owner acceptance.

Not shown by anything: a record released from a run as guarding in the service (the sample record is in no run), and an input received again and refused; see "What stays blocked" in the design.

Reviews (AI, Claude Opus 5.5, by reading; neither ran anything), 2026-11-25:

- Independent change review, billing integrity and retention: approve with conditions, no blocking finding. It found no path that authorizes deleting a guarding record, confirmed that every reader of the anchor uses the effective anchor and that no pinned expectation was weakened. Three conditions, done: a sentence of the design claimed that the guarding count makes a completed source show as a changed impact, which the code does not do (corrected in the design); the property test generates no policy change and its floor counts leases as well as go-aheads (stated); a console test fixture lacked the new key (added). Left open: the cost of walking every guard on each check is not measured; the rendered check proves the outcome label only as a strip heading, since no sample run holds a guarding outcome.
- ui-designer review of the two labels: FAIL, corrected with its strings word for word and not sent back for a second reading. "Source" pointed at the class of that name although a guarded item can be of another class, and the Korean "보호" read like a hold. The labels now say that the record is kept until the item is gone. Its suggestion for the nine-cell strip at medium width (three columns) is taken. The reviewer cannot certify the Arabic as final and could not read the 390-wide Arabic rendering; both are left for the owner's screen review.
