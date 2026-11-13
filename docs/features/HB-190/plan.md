---
type: ImplementationPlan
title: Evidence artifacts, sealed bundles, retention, legal holds and deletion that checks dependencies
description: Defines the first model of work item W-08, an evidence registry in which artifacts are held in the product store or at a verified remote location, a confirmed calculation is sealed as a bundle only when everything it rests on is held for its retention period, and retention, holds and deletion check which bundles depend on an artifact.
status: draft
---

# Scope and ownership

[HB-190](https://linear.app/hybrid-billing/issue/HB-190/w-08-evidence-artifacts-sealed-bundles-retention-holds-and-dependency) is the first increment of work item W-08 (requirement HB-08, with parts of QA-02 and QA-06). It is the fourteenth increment built under the owner's instruction of 2026-11-12. Base: main `7f4362f`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model decides whether the evidence of a confirmed billing amount may be deleted and holds records per tenant. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-08: the sources, mapping, attribution, policy, FX, engine and module versions and the result of a confirmed calculation are preserved so that they can be reached again. A URL and a hash alone do not guarantee reproduction. Proposed direction: an input manifest fixes the list of input files with revision, hash, coverage, row count and currency sums; where the immutability and retention of a customer's store cannot be verified, the needed evidence is copied into the product store; retention, legal hold and deletion check artifact dependencies. Scale: the row-level basis of a large manifest is kept apart and the header holds a digest and counts; evidence is shared as versioned artifacts and originals are not copied once per customer.
- Acceptance of HB-08: for the approved retention period and while a valid hold or dependency exists, a preserved billing is reproduced after the source URL expired, was deleted or lost its permission, and a past confirmed download returns the same file. Linked source cases DAT01, EX01, EX09, REV04 are NOT_RUN; their text is not in the planning files read.
- Open decisions of HB-08, not made: the retention period, who is responsible for encryption keys, the conditions of an immutable remote store.
- QA-02: before a source permission is withdrawn the permitted final evidence is secured, and where that is impossible the reason, the owner and the path of a later capture are recorded; an emergency security withdrawal is not delayed for evidence and is recorded as incomplete. AUD-QA-02 is NOT_RUN.
- QA-06: a vulnerable old version needed to reproduce the past runs only in an isolated verification environment, and reading confirmed evidence does not depend on running an old version; after a module is removed a confirmed document can be read. AUD-QA-06 is NOT_RUN. The external-app parts of QA-06 concern other work items.
- AR-03 names HB-08 as well. Its eligibility part is carried by `modules/intake` (HB-186), where an unverified extraction cannot be sealed. Its preservation part, keeping the original scan, the extraction candidate and the verification evidence together so that a value can be checked again after the source is gone, is not modelled here and is deferred.
- QA-06 also asks for the registration of storage, ownership, migration and deletion rules per module; that is not covered here.
- Existing code: `modules/intake` publishes input manifests with digests, counts and sums; `modules/normalize` (HB-188) binds a run to a mapping revision and a decimal profile. Neither stores bytes. The preservation store is not selected.

# Decision

One Go package, `modules/evidence`, standard library only, with an in-memory registry. It stores no bytes, copies nothing, reads no store and has no entry point wired in this increment: it records what a caller says is held where, and decides what is sealed, what is still held and what may be deleted.

What this can and cannot show: that a bundle is sealed only when every artifact it names is held under the rule below, that no bundle that still exists names an artifact the registry gave up, and that a bundle's verdict follows what was reported about its artifacts. It cannot know that a copy exists, that a remote store is immutable or that bytes match a digest; those are the caller's reports, and a later increment with a real store has to verify them. Nor does it know that the artifacts a bundle names are enough to reproduce a calculation: nothing ties a named manifest to the named originals. The verdict is therefore called `held`, not reproducible (stated limits).

Alternative considered: keeping evidence references inside each producing module (manifests in `modules/intake`, revisions in `modules/normalize`). It was not taken because deletion has to see every dependant of an artifact in one place.

Every caller-supplied string is an identifier under the project's identifier rule, of which the package has its own copy, a lowercase hexadecimal digest of 8 to 128 characters, or a value of a closed list. A location is an opaque identifier and never a URL, so that no credential of a signed address is stored or returned (stated obligation of the caller).

## Artifacts

An artifact is one versioned piece of evidence in a workspace, registered once and immutable in what it is: an identifier unique in the workspace; a kind from a closed list (`source_original`, `input_manifest`, `mapping`, `attribution`, `policy`, `fx`, `engine`, `module`, `result`, `document`); a version; a digest of its bytes; its size, positive; and its first custody. Registering the same identifier again with identical content, the first custody included, writes nothing; with other content it is a conflict. The decimal profile of a run, a target schema and contract terms have no kind of their own; a caller registers them as `policy` artifacts (assumption, listed below).

Custodies say where an artifact is held. They are instances that are only ever added, each identified by its kind and location:

- `product`: a copy in the product store, with the store's receipt;
- `remote`: a location the product does not control, with, optionally, a verification: the time until which immutability and retention were verified, and the evidence of that verification. Without a verification, or with one that has ended, a remote custody is a location and a hash and counts for nothing below.

`AddCustody(artifact, custody)` adds an instance. For an instance that exists: identical writes nothing; a remote verification that ends later, with its evidence, replaces the stored one; one that ends at the same time or earlier writes nothing and succeeds, and the call returns the stored end, which the caller must read; a product instance with another receipt is a conflict. `ReportLoss(artifact, kind, location, reason)` ends one instance, with a reason from a closed list (`expired`, `deleted`, `access_revoked`, `corrupted`, and for a remote instance `verification_withdrawn`). It is never refused for a dependency, because it reports what happened. A lost instance does not come back, but the same artifact can be held again by a new instance at another location: a restored copy, a new store, a restored access under a new reference.

One predicate decides everything else. At the present moment `now`, an artifact is held until a time T when it was not deleted through the registry and has an instance that was not lost and is either `product` or `remote` with a verification that ends at or after the later of `now` and T. It is held now when that is true for T = `now`. The edges, in one place: a verification that ends exactly now still holds; a bundle is protected while `now` is before its retention time and no longer at that time.

## Bundles

A bundle is the evidence of one confirmed calculation in a workspace. `Confirm(bundle)` creates it, immutable: an identifier unique in the workspace and never used again; what it is about, a customer and a period; the time until which it must be retained; whether FX was used and whether modules were used; and its items, each a role and an artifact.

It is refused, with every reason listed and nothing written, unless all of these hold. Each reason has a code, and where it concerns one, a role and an artifact; reasons are listed in the order of this list, then by role in the order of the kind list, then by artifact identifier:

- `unknown_artifact`: every item names an artifact of the workspace (one of another workspace is unknown in the same way);
- `deleted_artifact`: none was deleted through the registry;
- `wrong_kind`: the artifact's kind is the item's role;
- `named_twice`: no artifact is named twice;
- `role_missing`: there is at least one item of each of `source_original`, `input_manifest`, `mapping`, `attribution`, `policy`, `engine` and `result`. That these seven are what every confirmed calculation has, and that several of each may occur, is an assumption taken from the list in the requirement, not a decided contract;
- `fx_mismatch`, `module_mismatch`: `fx` items are present exactly when FX was used, `module` items exactly when modules were used; `document` items are optional;
- `retention_not_ahead`: the retention time is after `now`;
- `not_held`: every item's artifact is held until the retention time. So a remote location that was verified for a shorter time, or not at all, is not enough, and the caller copies the artifact into the product store first.

An engine or module version that is restricted (below) may be named; the bundle then says that it can only be run again in isolation.

`Confirm` also binds, at that moment, the identifiers of the open gaps of its customer and period (below) into the bundle. It does not refuse for them (not the conservative choice, listed below): a calculation may be confirmed knowingly without a source that could not be secured, and the bundle says so for good.

A bundle stores a header and its items. The header holds the number of items per role, the bound gap identifiers, and a digest: SHA-256 over the workspace, the bundle identifier, the customer, the period, the two flags as `fx` or `no_fx` and `modules` or `no_modules`, and then every item as role, artifact identifier and artifact digest, items sorted by role in the order of the kind list and then by artifact identifier, compared bytewise, every field preceded by its length as eight bytes, big-endian. The retention time is not in the digest, because it can be extended. Items are read in pages in that order. Artifacts are shared: many bundles name one artifact, and nothing is copied per bundle or per customer.

A document produced or reissued after confirmation cannot join the bundle; it is protected by a further bundle, and nothing here links the two. Several bundles may exist for one customer and period, and nothing says which is current (stated limits).

`Extend(bundle, until)` moves the retention time later; a time at or before the stored one writes nothing and succeeds, and the call returns the stored time. It is accepted after the retention time has passed, as long as the bundle was not purged, and refused for a purged bundle. It is not refused when an item is then not held until the new time: retention can be an obligation; the item is reported as at risk. A bundle confirmed by mistake, with a wrong retention time or wrong items, cannot be shortened or corrected and pins its artifacts until that time (stated limit, no exit here).

## Holds

`PlaceHold(hold)` records a legal hold in a workspace: an identifier, a reason reference, and at least one bundle; each must exist and not be purged, otherwise the hold is refused, since it would hold nothing. A hold is never refused because of the state of custody, and has no end time. `ReleaseHold(hold, reason)` ends it, with a reason from a closed list (`matter_closed`, `placed_in_error`, `operator`).

A hold names a fixed list of bundles. A bundle confirmed later for the same customer and period, and an artifact that no bundle names yet, are not covered (stated limit; a hold by scope is listed as undecided).

A bundle is protected while `now` is before its retention time or an active hold names it.

## Purge and deletion

`Purge(bundle, reason)` ends a bundle that is not protected, with a reason from a closed list (`retention_ended`, `operator`): its items are dropped and its header stays. It is refused while the bundle is protected.

`DeleteArtifact(artifact, reason)` gives up an artifact through the registry, finally, with a reason from a closed list (`retention_ended`, `wrong_registration`, `operator`): every instance ends, no custody can be added and no bundle can name it. It is refused, naming the bundles up to a bound with their total, while any bundle that was not purged names the artifact, protected or not. So the order is always purge, then delete, and this holds at every moment, not only at the moment of a call: no bundle that was not purged names an artifact deleted through the registry. A hold or an extension that arrives after the retention time has passed therefore still finds everything the registry was responsible for.

The caller removes bytes only after `DeleteArtifact` succeeded (stated obligation). `ReportLoss` remains the report of a loss that happened elsewhere; the registry cannot prevent it and shows it (below).

## Verdicts and reading

`Inspect(bundle)` reads, without writing, a bundle that was not purged:

- `held` when every item's artifact is held now; otherwise `broken`, listing the items that are not, in item order up to a bound with the total, each with the cause: `lost`, with the reason of the instance lost last, or `verification_ended`;
- `at_risk`, for a protected bundle only: the items that are held now but not until the end of its protection, which under an active hold has no end, so that every item without a `product` instance is at risk;
- how it can be run again: `operational`, or `isolated_only` when an `engine` or `module` item is of a restricted version;
- the gaps bound at confirmation, and the gaps of its customer and period that are open now.

`Exposed(workspace)` lists the bundles that were not purged and are `broken` or have an item at risk, by identifier, up to a bound with the total. `ReportLoss` returns the same for the bundles that name the artifact concerned, so that the caller of an emergency withdrawal learns at once what it broke.

`Download(bundle, role, artifact)` returns for an item of a bundle that was not purged the artifact's identifier, version, digest, size and the instances through which it is held now. Digest and size never change, so a past confirmed download is the same file as long as the artifact is held; when it is not held now the call is refused and nothing is offered in its place: a remote instance that is not verified is not offered even if it might still answer. Through a verified remote instance alone, "the same file" rests on that verification.

Reading an item does not depend on any other item: a `document` or `result` is downloaded whether or not an `engine` or `module` item is still held or may be run.

## Restricted versions

`Restrict(kind, version, reason)` marks a version of an `engine` or `module` as not to be run in operation, finally, with a reason reference. It belongs to the installation, not to a workspace, like the targets of `modules/normalize`: it applies to every artifact of that kind and version in every workspace, also to ones registered later, because a vulnerable version is vulnerable for every tenant. It changes no custody and no verdict; it changes only how a bundle can be run again. Whether anything is run, and where, is outside this package.

## Evidence gaps

`RecordGap(gap)` records, for a customer and period of a workspace, that evidence of a role is not secured: an identifier, the role, a reason from a closed list (`capture_failed`: it could not be captured before access ended; `emergency_revocation`: access was withdrawn at once for security; `received_after_end`: a source arrived after the end of the contract and is not yet secured), the owner of the follow-up and the path of a later capture. `CloseGap(gap, artifact)` closes it with an artifact of that kind that is held now. Gaps are never deleted. Open gaps are bound into a bundle at confirmation and shown by `Inspect`, as said above; that is their only effect.

Limits, stated: the artifact that closes a gap is not tied to the customer or period and can be deleted afterwards if no bundle names it; `ReportLoss` records no gap by itself, the caller does, with what `ReportLoss` returned; and the order of QA-02, securing evidence before access is withdrawn, cannot be enforced here because no operation here withdraws access.

## Repeats, error kinds, tenant boundary, time and limits

Every operation is its own key and first tests for an identical repeat, before state and limits: a repeat with the same content writes nothing and succeeds in every state reached since, and returns that state, which the caller must read: `Register` of a deleted artifact, `Confirm` of a purged bundle (compared with what was confirmed, not with a later retention time), `PlaceHold` of a released hold, `RecordGap` of a closed gap. The same identifier with other content is a conflict. `Extend` and a remote verification follow the rule for values that only move forward, given above.

Error kinds: a malformed request is invalid; an unknown artifact, instance, bundle, item, hold or gap is not found, and so is one of another workspace; the same identifier with other content, or a second loss, purge, deletion, release or closing with another reason or artifact, is a conflict; no room is a limit error; everything a stored state refuses is rejected: `Confirm` with reasons, `DeleteArtifact` while named, `Purge` while protected, a hold on a purged bundle, `Extend`, `Inspect` and `Download` of a purged bundle, `Download` of an item not held, `AddCustody` on a deleted artifact or a lost instance, `CloseGap` with an artifact of another kind or not held, `verification_withdrawn` for a product instance. `Restrict` of another kind is invalid. No error repeats a string of the refused request.

Every operation but `Restrict` takes the workspace and reaches only that workspace's artifacts, bundles, holds and gaps. The registry authorizes nothing and must not be exposed directly (stated limit).

The clock is injected and read once per operation; the registry keeps the highest reading it has seen, and `now` in every rule is that. Because time alone ends protection, one wrong reading far ahead ends the protection of every bundle whose retention time it passes, for good (stated limit, an exposure blocker).

Required and positive at construction, without defaults: artifacts, bundles, holds and gaps per workspace, instances per artifact, items per bundle, bundles per hold, restricted versions, the page size and the bound of lists. Artifacts, bundles, holds and gaps are counted for good, deleted, purged, released and closed ones included; workspaces are created by first use and not limited in number; one lock serves every workspace; none has an exit here (stated limits, exposure blockers; retention of the registry's own records is undecided).

## What is not decided here

The retention period, who holds encryption keys and what makes a remote store verified and who may say so: these are the decisions the design part of W-08 asks for, they remain the owner's, and the design part stays open. Also: whether a hold should force a copy into the product store; a hold by customer and period instead of a list; which roles a confirmed calculation must have and how many of each; under which kind a decimal profile, a target schema and contract terms are kept; whether a calculation may be confirmed with open gaps or with a restricted version; how a mistaken bundle is corrected; which bundle of a customer and period is current; who may delete, purge, hold, release and restrict. Not the conservative choices, for the owner: `Extend` and `PlaceHold` succeed although evidence is then at risk; `Confirm` succeeds with open gaps and binds them.

# Allowed files

`modules/evidence/*` (new); `docs/features/HB-190/*`; and one sentence of `docs/features/HB-188/verification.md`, which named 2026-11-14 as a day of execution although every check ran on 2026-11-13 (a correction of a recorded fact, found while this plan was reviewed).

# Verification

- The acceptance at package level, against a store simulated in the test, whose objects really expire, are removed or lose access and drive the reports: a bundle whose artifacts have a product instance stays `held`, and each download returns the same identifier, version, digest and size, after every remote instance expired, was deleted and lost access; a bundle that rests on a remote instance alone cannot be confirmed unless it was verified until the retention time, and is `broken` once that instance is lost; the simulation removes bytes only after `DeleteArtifact` succeeded, and `held` always implies that the simulated bytes exist.
- Literal table tests with a test clock: every refusal of an artifact and of a custody; a verification replaced by one ending earlier, at the same time and later, and with other evidence; custody added after a loss at the same and at another location, after every instance was lost and after deletion; a product copy added again after `corrupted`; loss of an instance that is absent, already lost, and lost with another reason; the predicate at the edges of a verification and under a clock that moves; every reason `Confirm` is refused, alone and listed together in order, with a retention time at and before now, a deleted artifact and a restricted version; open gaps bound at confirmation; the header digest by a second writing of the encoding; shared artifacts; deletion of an artifact that no bundle names, that only purged bundles name, that an unprotected bundle names, and that one protected and one unprotected bundle name; purge refused under retention, under a hold and under both, and allowed at the moment protection ends; retention lapsing and then a hold or an extension, with every artifact still there; a hold on a missing, a purged and no bundle; `Extend` earlier, equal, later, after the lapse and on a purged bundle; extension and holds putting items at risk; `Inspect` in each verdict and `at_risk` for a bundle that is no longer protected; `Exposed` and what `ReportLoss` returns; `Download` through a product instance, a verified and an unverified remote instance, of an item not held and of a purged bundle; restriction across workspaces, before and after registration, of another kind; reading that does not depend on an engine or module; gaps recorded, closed with the right, the wrong and an unheld artifact; repeats of every operation after every state and what they return; paging and bounded lists with totals and order; the tenant boundary; limits; errors that repeat nothing.
- Properties with a fixed seed over random histories in which the clock advances across retention and verification times, against a reference model written in the test that decides acceptance and every verdict, and, independent of the model, invariants read from the registry after every step: no bundle that was not purged names an artifact deleted through the registry; a `held` bundle has no item that is not held; a protected bundle is not purged; held until T implies held until every earlier T; a download never changes.
- Concurrency under the race detector: deletion against confirmation of a bundle that names the artifact, of which exactly one succeeds; a hold against a purge; an extension against a purge.
- The package's tests run many times in a row before any mutation run; a mutation check by a script in Python with results appended to files that are never overwritten.
- `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level after this ticket: DAT01, EX01, EX09, REV04, AUD-AR-03, AUD-QA-02, AUD-QA-06. This ticket does not complete W-08.

# Deferred

A store, copying and verifying bytes; the preservation part of AR-03; the module storage rules of QA-06; a hold by scope; encryption and keys; verified links to the manifests of `modules/intake` and the revisions of `modules/normalize`; running an old version in isolation; the external-app parts of QA-06; contract and settlement states of QA-02; commands, permissions and screens; persistence.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL without a blocker and without asking for a redesign: 4 HIGH, 11 MEDIUM, 6 LOW. It judged the predicate and the refusal of unverified remote custody sound, and found the guarantee stronger than the rules where protection was a momentary condition, gaps were free-standing notes and a custody was a slot. All were adopted above.

| Finding | Resolution |
| --- | --- |
| F1 protection lapsing and resuming over a deleted artifact | Deletion is refused while any bundle that was not purged names the artifact; purge is refused while protected; the invariant holds at every moment |
| F2 a hold on a purged bundle; hold scope | Refused; at least one bundle; the fixed list stated as a limit |
| F3 gaps that block and show nothing | Open gaps are bound at confirmation and shown by `Inspect`; `ReportLoss` returns what it exposed; the reasons defined; limits stated |
| F4 a lost custody stranding bundles | Custodies are instances that are only added; a deleted artifact is final |
| F5 edges of the predicate and time | Held until the later of now and T; a retention time must be ahead; `verification_withdrawn`; the edges and the clock stated |
| F6 repeats of values that only move forward | One rule, returning the stored value; repeats after purge, deletion, release and closing return the state |
| F7 roles guessed; what the verdict means | At least one of seven roles, marked as an assumption; modules declared like FX; later documents, missing kinds and the name `held` stated |
| F8 stored strings | Typed; a location is never a URL; reasons from closed lists |
| F9 restriction per workspace | By kind and version for the installation |
| F10 AR-03 overclaimed | Corrected in the baseline; the preservation part deferred |
| F11 the unguarded report | `Exposed`; the calling contract of deletion; one meaning of `deleted` |
| F12 error kinds and lists | Stated |
| F13 the header digest | Algorithm, encoding, order and coverage stated |
| F14 no correction of a mistaken bundle | Stated as a limit |
| F15 to F21 | Limits, the date, field rules, remote downloads, `at_risk` without protection, the order of QA-02 and the open design part stated |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: sixteen of the twenty-one first findings resolved and five partly; fifteen new findings, 1 HIGH, 8 MEDIUM and 6 LOW. It said implementation may start after text fixes and that no further design pass is needed. The rules below settle them and are part of the decision; where they are more precise than a sentence above, they govern. The change review reads them against the code.

- N1, a product location belongs to one artifact. The registry keeps, for the installation, which artifact of which workspace a `product` location was first used by, and a `product` instance at a location that another artifact used, in any workspace and whether or not that instance was lost since, is a conflict. So `DeleteArtifact` of one artifact never licenses the removal of bytes another artifact rests on. Remote locations are not exclusive: the product does not delete there. The conflict tells a caller that a location is taken and nothing else (stated; locations are the product's own, not a tenant's).
- N2, reasons of `Confirm`. An item yields only the first that applies of `unknown_artifact`, `deleted_artifact`, `wrong_kind`, `not_held`. `named_twice` is listed once per artifact, without a role. The counts for `role_missing`, `fx_mismatch` and `module_mismatch` use the role an item states, whatever the artifact's kind. `unknown_artifact` is a reason inside a rejected confirmation, not a not-found error.
- N3. `AddCustody` on a deleted artifact or for an instance that was lost, and `Extend` of a purged bundle, are rejected also when the request repeats an earlier one: success would read as held or retained.
- N4 and N5. An artifact keeps a digest of what was registered and a bundle of what was confirmed, the retention time as confirmed included; a repeat of `Register` or `Confirm` is compared with that, not with the present instances or retention time, and so also after deletion and purge.
- N6. Lost instances count towards the limit of instances per artifact. An artifact that used them up can never be held again (stated limit, an exposure blocker).
- N7. The version of an `engine` or `module` artifact is one identifier that names the component and its version together; that it does is the caller's obligation. `Restrict` returns nothing and says nothing about what it matched. A mistaken restriction has no exit (stated limit).
- N8, gaps. A gap concerns a bundle when its customer and period equal the bundle's, whatever its role. `Confirm` returns the gaps it bound. `Inspect` shows the bound gaps with their present state, and for a closed one the artifact that closed it, and the gaps of the customer and period that are open now; both sorted by identifier up to the bound of lists, with totals.
- N9. An item that is not held has the cause `lost` when every instance of its artifact is lost, with the reason of the one reported last, and `verification_ended` otherwise, which includes a remote instance that was never verified. Every operation, reads included, moves the registry's time forward.
- N10, stated obligations and limits: a location names one stored object, so a copy made again after a loss has a new location; a `remote` and a `product` instance with the same location text are two instances; a loss reported in error cannot be taken back, only a new instance added.
- N11. The header digest starts with the field `evidence-bundle-v1`; an artifact's digest enters it as its hexadecimal text; the bound gaps are in the header and outside the digest. Items, pages, the digest and the reasons of `Confirm` all use one order of roles, that of the kind list.
- N12, error kinds. A request is judged in this order: malformed, which is invalid and includes a hold that names a bundle twice; more items than a bundle or more bundles than a hold may have, a limit error; an identifier that exists, which is a repeat when identical and a conflict otherwise; something unknown, not found; what a stored state refuses, rejected; no room, a limit error. `ReportLoss` for an artifact that was deleted through the registry is rejected; its instances ended without a loss reason. A hold that is refused returns the positions, in the request, of the bundles its error is about: the missing ones when any is missing, otherwise the purged ones.
- N13. `ReportLoss` returns, of the bundles that name the artifact and were not purged, those in which that artifact's item is now not held or at risk, by identifier up to the bound with the total.
- N14. `Extend` returns, with the stored time, how many items of the bundle are not held and how many are at risk; `PlaceHold` returns the named bundles that are `broken` or have an item at risk.
- N15. `Bundle(workspace, bundle)` reads a header in every state, purged included: subject, flags, retention time as confirmed and now, counts per role, digest, bound gaps, state. `Download` returns the instances through which the artifact is held now, each with kind, location, receipt, verification end and evidence, sorted by kind and location.
- Scale. The registry keeps, per artifact, the bundles that name it, so deletion and `ReportLoss` do not scan; `Exposed` reads every bundle of a workspace while the single lock is held (stated with the other exposure blockers).
- Tests added: the shared product location, within and across workspaces, against the simulated store; and, as a property over random histories, that a `held` bundle implies that the simulated bytes of every item exist.

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker or HIGH, 2 MEDIUM and 9 LOW, with one note. It found no path by which an artifact that a bundle still names is deleted through the registry, or a protected bundle is purged, other than the stated limit of the clock, and confirmed by reading the predicate, the exclusivity of product locations, the reasons of `Confirm`, the reverse index and the absence of shared state in views.

| Finding | Resolution |
| --- | --- |
| 1 positions of a refused hold | The rule says which positions are returned; a test with both kinds added |
| 2 lists bounded by their own limits | Stated below |
| 3 the registry's time kept in the clock's zone | Kept in UTC without a monotonic reading |
| 4 instances of a deleted artifact shown as they were | Stated below |
| 5 a reason read through a pointer that could be nil | Rewritten without the pointer |
| 7 MEDIUM an invariant in the property test that could not fail | It reads the holds of a purged bundle directly |
| 8 MEDIUM the simulated store followed the reports | Its remote objects now expire at the end of their verification without any report, an unverified remote object is not kept at all, and a held bundle must have bytes that still exist at that moment |
| 9 a repeated registration putting bytes back | Only a first registration stores bytes |
| 10 what the model does not cover | Recorded in the verification record |
| 11 missing literal cases | Added: purge under retention and a hold, purge at the moment protection ends, a hold on a missing and a purged bundle, a loss of an unknown instance of a deleted artifact, a refused deletion cut at the bound, one hold identifier in two workspaces, and the shared product location against the store |
| 12 dead code in a test | Removed |

Stated with this review, as the code behaves:

- No error value repeats a string of the request. The reasons of a refused `Confirm` are structured and name the roles and artifacts of its items, which were checked to be identifiers before.
- `ReportLoss` returns the exposed bundles also when the loss was reported before. An unknown workspace gives an empty list from `Exposed` and `Gaps`, like one in which nothing happened.
- For a deleted artifact or a lost instance the refusal comes before any other test: `AddCustody` is rejected before a receipt or location conflict is looked at, and `ReportLoss` on a deleted artifact is rejected whatever instance it names.
- The state of an artifact governs: the instances of a deleted artifact are shown as they were last recorded and are not marked lost; its product locations stay taken.
- All times are kept and returned in UTC.
- Bounded by their own limits and not by the bound of lists: the gaps in a bundle's header, the instances of an artifact and of a download, the reasons of a refused `Confirm`, the positions of a refused hold.
- The order of items in `Confirm` and of bundles in `PlaceHold` does not matter for a repeat. An item given twice yields its own reason twice and `named_twice` once. A page beyond the end is empty; a page larger than the limit is invalid. Holds and gaps carry no times, and a hold does not change a bundle's time of last change.
- Two tests in the code were removed because they could change no result: `at_risk` asks only whether an item is held until the retention time or, under a hold, has a copy, since a bundle that is not protected has no hold and a retention time that is not ahead; and the lists of exposed bundles do not ask whether a bundle was purged, since a purged bundle has no items.
