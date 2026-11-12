---
type: ImplementationPlan
title: Onboarding projects, readiness stages with bound evidence, diagnostic outcomes and separated access roles
description: Defines the first model of work item W-04, an onboarding project per workspace and path whose readiness stages are stored separately, each backed by a diagnostic run bound to a configuration generation, a validity period and the access grant it ran under, with one predicate that decides both whether billing may be opened and whether it reads as open.
status: draft
---

# Scope and ownership

[HB-182](https://linear.app/hybrid-billing/issue/HB-182/w-04-onboarding-projects-readiness-stages-with-bound-evidence) is the first increment of work item W-04 (requirement HB-04, with the evidence-validity part of DB-08 and the role separation of QA-07). It is the tenth increment built under the owner's instruction of 2026-11-12. Base: main `29c73eb`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model decides when billing may open and is keyed by tenant. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-04: authentication success, operational read, first data, full coverage, FOCUS conformance, reconciliation and billing activation are different states. Support per provider, contract, account role and data kind is shown by evidence. Proposed direction: `OnboardingProject`, `SetupStep`, `DiagnosticRun`, `CapabilityEvidence`, `SourceCoverage`; branches for an existing account, a parent operator and a file path, with only the required steps in the progress denominator; setup permission, export-service write and collector read are separate. Scale: discovery by cursor; diagnostic results are stored and a revisit does not query 100,000 accounts or rows again.
- Acceptance of HB-04: insufficient permission, normal no-usage, waiting for the first export, an unsupported period and a partial file are distinguished, and operational read is tested after the setup permission is withdrawn. Linked source cases, all NOT_RUN, among them ON03 (after authentication, tell insufficient permission, an empty export and late arrival apart), ON09 (a configuration change or a permission withdrawal re-validates the affected steps and blocks reuse of past approvals), PO09 (an alternative path by a parent's data or a file import, and never showing a missing original as held), FD01 (file read and query succeed after the setup permission is withdrawn).
- Open decisions of HB-04, not made: the first provider, contract and account roles to prove, real samples, and who holds access. The work breakdown fixes the state and permission contract after that decision, so the contract below is provisional until then.
- DB-08: step evidence is bound to a configuration hash, a product version and a validity period; a change invalidates only the affected later steps; created resources, extra cost and policy changes are shown for consent beforehand; a handoff link is limited to a task, expires, and holds no secret; the setup permission has a withdrawal condition. AUD-DB-08 (NOT_RUN): a stale diagnosis or a changed configuration cannot open billing, an expired or foreign handoff link is refused, and after the setup permission is withdrawn only operational read remains.
- QA-07: the setup owner's temporary permission, the export service's operating writer and the collector's reader are separate in purpose and in withdrawal condition. AUD-QA-07 (NOT_RUN): withdrawing the setup permission does not stop the export writer, and a scope outside support branches to an unsupported path instead of asking for authentication again.
- DB-02 (provider profiles and their branches) is a mandatory detail contract of W-04 and depends on the owner's choice of the first provider.
- Existing code: Python modules under `modules/focus_import` and `modules/focus_local` hold a local FOCUS import path with its own connection and collection status; no Go package models onboarding. `modules/contracts` (HB-178) uses one sequence per record with caller-assigned write identifiers. Everything is in memory or local files; the business database is not selected.
- The planning sources are untracked local files of the primary checkout and were read there.

# Decision

One Go package, `modules/onboarding`, standard library only, with an in-memory registry. It records what was found and decides whether the recorded evidence still holds; it connects to no provider, stores no credential, runs no diagnosis and has no entry point wired in this increment. A diagnostic run is something a caller reports to it.

What is built of the proposed direction: `OnboardingProject`; the proposal's `SetupStep` as the readiness stage; `DiagnosticRun`. Not built at all, each mandatory for W-04 and left to follow-up tickets: `CapabilityEvidence` and `SourceCoverage` (support per provider, contract, role and data kind shown by evidence, the second sentence of HB-04); provider profiles (DB-02); handoff links and the consent for created resources and cost (DB-08); discovery; the relation to the Python import path.

## Projects and paths

An onboarding project belongs to one workspace and has a caller-assigned identifier, an opaque reference to the source it onboards, a path fixed at creation, and the hash of its configuration at creation. `Create` with an identifier that exists in the workspace writes nothing and succeeds when source, path and hash are identical, and is a conflict otherwise. A new project has sequence 0, and its creation is not one of its counted writes.

| Path | Meaning | Required stages |
| --- | --- | --- |
| `existing_account` | The workspace connects to the source with its own access | all seven |
| `parent_operator` | Data comes through a parent operator's connection | all seven |
| `file_import` | Files are brought in; nothing is connected | first data, full coverage, FOCUS conformance, reconciliation, billing |

Only the required stages exist for a project: a stage outside its path cannot be recorded, named in a configuration change, or counted. The stage sets are provisional and the owner's to decide; in particular what "authenticated" should mean on the parent path.

Stated limits of the paths: two projects may name the same source, and nothing reconciles them. On the parent path the access belongs to the parent and is not linked to anything here: if the parent's access ends in the parent's own workspace, this project learns it only through a failing run or an explicit withdrawal. A file-import project that stands in for an original that could not be obtained carries no mark that the original is absent, so PO09's "never show a missing original as held" is not met by this increment.

## Writes, time and identifiers

Every project has one sequence that increases by one with each write. Every write carries a caller-assigned write identifier, unique in the project across all kinds of write: a write whose identifier is stored and whose content is identical writes nothing and answers with the sequence it produced, marked as a repeat, whatever expected sequence the repeat carries, since that is not part of the content, and whatever happened since, so the caller reads the state separately; a stored identifier with other content or of another kind is a conflict; a refused write stores nothing. The order in which a write is judged, and so which refusal it gets when several apply: a malformed workspace, project or write identifier; an unknown project; a stored write identifier (repeat or conflict); the expected sequence; the write limit; then the operation's own input and its rule. A write with a stale expected sequence and a malformed body is therefore a conflict, not malformed; nothing is written either way.

Writes that record progress (`RecordRun`, `Reach`, `GrantRole`, `OpenBilling`) take the caller's expected sequence and conflict on a mismatch. `GrantRole` is one of them also for `setup`, although granting `setup` closes billing (below): when it is refused, no setup access is recorded and nothing needed closing. Writes that take something away (`ChangeConfig`, `WithdrawRole`) take no expected sequence, so that they cannot lose against a busy project.

Every time is the registry's own: the injected clock is read once by every operation, reads included, under the registry's lock (except an operation refused for a malformed workspace, project or write identifier or on a registry that is not built, which reads nothing), and the registry's time is the highest reading so far, so a reading earlier than an earlier one is replaced by it and nothing that lapsed can come back by the clock. A read that moves the registry's time forward is not a write. No caller states a time, and every duration a caller states must be positive. A run is therefore dated when it is reported, and a caller that reports late dates it late, which shortens nothing but its remaining validity (stated limit: the registry cannot know when a diagnosis was really made).

Whatever a rule below needs to know about the moment of a write, which grant was active, whether setup access existed, which generation was current, is decided from the stored state when the write is made and stored with the record; it is never worked out afterwards by comparing times, because two writes can carry the same time.

Runs and grants are appended and never deleted. A stage keeps its current record only, the opening its current state only, and the configuration its current hash and generation number: replaced records are not kept in this increment, since nothing reads them, and history belongs with persistence (stated).

## Configuration generations

The project's configuration has a generation number, 1 at creation. `ChangeConfig(project, write, hash, from)` starts the next generation with a hash, which may equal an earlier one, and names the first required stage the change affects. Every reached stage from that stage onward is invalidated with the reason `config_changed`, and every required stage from that stage onward, reached or not, gets a floor: the sequence of this write. `Reach` accepts only a run recorded after the stage's floor. Stages before it are untouched: their evidence belongs to an older generation and keeps holding, which is what "only the affected later steps" means.

A run belongs to the generation in which it was recorded. `Reach` refuses a run of an earlier generation, whatever its hash, so that returning to an old hash does not bring old evidence back.

`billing_open` is the last stage and is therefore invalidated by every configuration change (provisional and stricter than DB-08 asks; the owner's to decide). Which stage a change affects is stated by the caller, later the command that knows the configuration, and the registry cannot check it: a caller that names `billing_open` for a change that affected coverage keeps stale evidence alive and can open billing again at once. AUD-DB-08's "a changed configuration cannot open billing" is therefore shown at package level only for the stages the caller names (stated limit).

## Access grants, separate per role

A project on a connected path has three roles, and no operation on one touches another:

| Role | Purpose | Granting |
| --- | --- | --- |
| `setup` | The setup owner's temporary access | Needs a duration, at most a limit given at construction, and a withdrawal condition from a closed list (`after_setup_complete`, `on_expiry`, `on_request`) |
| `export_writer` | The export service's operating write access | No expiry |
| `collector_reader` | The collector's read access | No expiry |

`GrantRole` creates a grant: a numbered record with the role, the holder as an opaque reference, never a credential, and the time. A grant is active from the write that made it until its end, which is its withdrawal or, for `setup`, the first moment the registry's time is at or after its expiry. Granting a role that has an active grant is refused, and so is withdrawing a role that has none, with nothing written; each role has at most one active grant.

The withdrawal condition of `setup` is recorded and shown and is not enforced: nothing here knows when setup is complete (stated limit). A file path has no roles: granting and withdrawing are refused there.

## Diagnostic runs

`RecordRun(project, expected, write, run)` stores what a diagnosis found. The caller states: the stage it examined; the outcome; on a connected path, the role it ran under; how long the result is valid, at most a limit given at construction; the product version; an opaque reference to its stored detail, so that a revisit reads the stored result and queries nothing again; and for some outcomes the fields below. The registry adds the time, the configuration generation and, on a connected path, two things decided at the write: the grant, which is the active grant of the stated role, without which the run is refused; and whether a `setup` grant was active at that write. On a file path a run names no role and is bound to no grant. A run is valid from its time until its time plus its validity, the end excluded. A run cannot examine `billing_open`.

The product version must be the registry's, given at construction: a diagnosis by another version of the product is refused as evidence for this one. Because records do not outlive the registry in this increment, a stored run can never carry another version, so there is no check at read time yet; with persistence, loading records of another version must make their stages stale, and how fine a version difference counts is the owner's decision (stated, not built).

The outcome is one of a closed list, each allowed only for the stages where it can mean something:

| Outcome | Meaning | Stages |
| --- | --- | --- |
| `passed` | The stage's condition was observed | any |
| `passed_no_usage` | The source was read and legitimately holds nothing for the period: normal no-usage | first data, full coverage |
| `insufficient_permission` | The access exists and lacks a permission | any, on a connected path |
| `awaiting_first_export` | The export is set up and its first delivery has not arrived | first data, full coverage, on a connected path |
| `period_unsupported` | The source does not provide the period asked for | first data, full coverage, reconciliation |
| `partial_file` | A delivery is incomplete | first data, full coverage, FOCUS conformance |
| `unsupported_scope` | The scope is outside what is supported. The run names the alternative: one of the two other paths, or none | authenticated, operational read, first data |
| `failed` | Anything else, with a code that is an identifier | any |

Only `passed` and `passed_no_usage` can reach a stage; every other outcome is stored and shown and reaches nothing. Branching to an alternative path means creating another project; this increment does not link the two.

## Stages, stored separately

The stages, in their order: `authenticated`, `operational_read`, `first_data`, `full_coverage`, `focus_conformance`, `reconciled`, `billing_open`.

Each stage has its own stored record and none is derived from another: reaching a later stage records nothing about an earlier one. `Reach(project, expected, write, stage, run)` makes a stored run the stage's current evidence when all of these hold, and is refused otherwise with nothing written:

- the run examined that stage and its outcome can reach it;
- the run is of the current configuration generation, has not expired, and on a connected path ran under `collector_reader` with its grant still active;
- the run was recorded, by sequence, after the run of the stage's current record, if there is one, and after the stage's floor, if it has one, so that a run made before its inputs were replaced cannot reach the stage, whether or not the stage had been reached before;
- for `full_coverage`, when `first_data` currently holds, the two agree on emptiness, and when it does not hold, the run is not `passed_no_usage`.

A stage is empty when the run of its current record is `passed_no_usage`.

On a connected path a run under `setup` or `export_writer` is stored and shown and reaches nothing: evidence is what the collector's own access can do.

What a new record does to later stages: if the stage held just before with the same emptiness, the new record is a renewal and nothing else changes, except what the rule under Opening billing does to an opening. Otherwise, the stage was missing, stale, or changes between empty and not empty, and every reached stage after it is invalidated with the reason `upstream_renewed`, `billing_open` among them, and every required stage after it gets the sequence of this write as its floor (a floor on `billing_open` has no effect, since no run reaches it), because what they showed, or what a spare run for them showed, rested on evidence that has been replaced.

`billing_open` is reached only by `OpenBilling`.

## When a stage holds, and what a reader sees

A reached stage before `billing_open` holds at the registry's time when it has not been invalidated, its run has not expired, and on a connected path the grant its run ran under is active. This is evaluated when it is read and never stored.

`Readiness(project)` returns, for each required stage: `missing`; `holding`; or `stale` with one reason, the first that applies of `config_changed`, `upstream_renewed`, `role_ended`, `expired`. Beside the status it returns whether the stage is empty, and whether every required stage before it holds, so that a `reconciled` that holds while `operational_read` is missing reads as "evidence recorded, prerequisites not held" and not as progress.

Progress is the number of stages, counted from the first required stage, that hold without a gap, over the number of required stages; a stage that holds beyond a gap does not count. Empty stages count as holding in the progress and are flagged, since they block billing below.

## Opening billing: one predicate

`billable` is true at the registry's time when every required stage before `billing_open` holds, neither `first_data` nor `full_coverage` is empty, on a connected path no `setup` grant is active and the run behind `operational_read` carries the mark that no `setup` grant was active when it was recorded, and the project can still accept an invalidating write, that is, its reserve is not used up.

`OpenBilling(project, expected, write)` records `billing_open` when `billable` is true and is refused otherwise, listing every reason: the stages that are missing, stale or empty, and the two setup conditions. Opening when billing is open is refused.

Billing reads as open exactly when a `billing_open` record exists, has not been invalidated, and `billable` is true now. The same predicate decides both, so the two cannot disagree. The status of the `billing_open` stage in `Readiness` is this answer: `missing` when it was never opened; `holding` when open; otherwise `stale` with one reason: the stored reason of its invalidation (`config_changed`, `upstream_renewed`, `setup_granted`, `capacity_exhausted`, `evidence_lapsed`), or, when it is not invalidated and `billable` is false, which only the clock can bring about, `evidence_lapsed`.

The opening does not survive a moment in which `billable` is false. Every accepted write, of any kind, looks at the opening twice, before it is applied and after: if an opening exists, is not invalidated, and `billable` is false at that moment, the opening is invalidated. The reason is `evidence_lapsed` at the check before; at the check after it is `setup_granted` when the write is a grant of `setup`, `capacity_exhausted` when the write used up the reserve, and `evidence_lapsed` otherwise. A record is invalidated once: the first reason is kept, also when the write that invalidates it is the one that uses up the reserve, so a configuration change that is the last reserve write leaves `config_changed`, so a write whose own rule would also invalidate the opening (a configuration change, a `Reach` that is not a renewal) leaves the reason of the check before it when that check already acted. A refused write, a repeat of a stored write and a read perform neither check and store nothing; the claim below does not depend on them, because none of them changes anything `billable` reads. This one rule replaces a list of cases, and it is why billing goes from not open to open only through an accepted `OpenBilling`: `billable` can turn false by the clock alone only when evidence expires, and it can turn true again only through a write, a `Reach`, which finds it false before it is applied and invalidates the opening; it can turn false by a write, for example a grant of `setup`, a `Reach` that replaces the operational read by one recorded under setup or makes a stage empty, a withdrawal, a configuration change, the last write of the reserve, and that same write finds it false after it is applied. A `setup` grant that then expires by the clock therefore finds the opening already invalid.

The setup conditions are this model's reading of the acceptance "operational read is tested after the setup permission is withdrawn": billing opens only on a read that was shown when the temporary access was gone. Withdrawing or expiring `setup` changes no stage before `billing_open`, and does not touch `export_writer`, because nothing rests on it.

Withdrawing `collector_reader` makes every reached stage before `billing_open` `stale` with `role_ended`, and invalidates the opening by the rule above; granting it again creates another grant and revives nothing, because each run is bound to the grant it ran under.

Provisional and conservative, the owner's to decide:

- billing cannot open on empty first data or coverage. This blocks a customer with legitimately no usage for as long as that lasts, non-usage charges included;
- billing that was open reads as not open as soon as any required evidence expires, which may be stricter than the business wants for a running customer, and means evidence must be renewed before it lapses;
- billing needs the setup access to be gone; an alternative would let it open while setup is still active;
- evidence counts only from runs under the collector's access;
- a stage renewed an instant before its evidence expires changes nothing else, while the same stage reached an instant after invalidates every later stage and the opening, so a missed renewal costs a diagnosis of each later stage and a new opening;
- on a file path evidence is bound to no access at all, and billing opens on four evidence stages.

## Tenant boundary and limits

Every operation takes the workspace and addresses only that workspace's projects; a project of another workspace is not found. The registry authorizes nothing and must not be exposed directly (stated limit). The source reference, holders, detail references, failure codes, the configuration hash and the product version are identifiers under the project's identifier rule (letters, digits and `. _ : -`, at most 128 bytes), so an address, a provider resource name with a slash, free text or a secret of another shape is refused; that an identifier is not itself a secret or a personal name is the caller's obligation. The package has its own copy of the rule (stated). References are not checked against the workspace.

Required and positive at construction, without defaults: projects per workspace; writes per project; a reserve of further writes per project; the longest validity of a run; the longest duration of a setup grant. The count is of accepted writes of every kind, which is the project's sequence, so polling runs that find `awaiting_first_export` use it up. While the sequence is below the limit every write is accepted; from the limit on, only `ChangeConfig` and `WithdrawRole`; when the sequence equals the limit plus the reserve, none, and the reserve is used up. Repeats of stored writes are answered at any time. At the limit only `ChangeConfig` and `WithdrawRole` are accepted, from the reserve, because the writes that take evidence or access away must not be the ones refused; when the reserve is used up they are refused too, and from that moment `billable` is false and an opening that was valid is invalidated with the reason `capacity_exhausted` by the write that used the reserve up, so that a project that can no longer record a change or a withdrawal never keeps billing open on what it can no longer correct.

Stated limit, an exposure blocker like those of the other registries: nothing is evicted, so a project that renews its evidence for long enough reaches its limit, its evidence then expires, and it cannot be renewed in this registry. Retention belongs with persistence. The number of workspaces is not bounded here.

## What is not decided here

The first provider and its profile; the stage set of each path; which stages a given configuration change affects; the validity period of each kind of diagnosis (the caller states it per run, within the limit); how fine a product version difference invalidates evidence; whether billing may open on empty data, while setup access is active, or stay open through an expiry; whether any configuration change must close billing; whether evidence under other roles may count; who may record runs, change configuration, grant roles and open billing.

# Allowed files

`modules/onboarding/*` (new); `docs/features/HB-182/*`.

# Verification

- Literal table tests with a test clock: the three paths and their stages; each outcome on each stage, allowed or refused by the table; only the two passing outcomes reaching a stage; stages reached out of order, with the prerequisite flag and the gap-free progress; every refusal of `Reach` (another stage, an outcome that cannot reach, an earlier generation also after a return to the same hash, an expired run, a run under another role, a run whose grant ended also after a new grant, a run older than the current one, a run recorded before the stage's floor, coverage and first data disagreeing on emptiness); every refusal of `RecordRun` (no active grant, another product version, a validity that is not positive or above the limit, an outcome outside its stages, `billing_open`, a role on a file path); the refusals of a setup grant (a duration above the limit, a condition outside the list).
- The acceptance at package level: the five distinguished outcomes; `setup` withdrawn and expired changing no stage and leaving `export_writer` active; `collector_reader` withdrawn making stages stale, a new grant reviving nothing; billing refused while `setup` is active and on an operational read recorded during it, opened after a read recorded without it; a run and the withdrawal of `setup` written at the same clock reading, with the run still marked as recorded under setup; `setup` granted after the opening and then expired or withdrawn, with billing staying closed until it is opened again; the operational read replaced after the opening by one recorded under setup and then by one recorded without it, with billing staying closed until it is opened again; a file-path project opening with no roles, and refusing every role operation.
- DB-08: expiry at the exact boundary; a change from each stage invalidating that stage and the later ones only; a change naming a stage outside the path refused; a same-hash change starting a generation; billing refused with each reason listed, opened when billable, and reading as not open after an expiry, a change, a withdrawal and an emptiness change; billing not restored by reaching the lapsed stage again, and opened again explicitly; a renewal just before and a reach just after an expiry; a renewal while an earlier stage is stale; a spare run recorded before an upstream stage was replaced, refused afterwards, for a stage that had been reached and for one that had not.
- Writes: repeated identifiers with identical and other content for every kind; expected-sequence conflicts; invalidating writes accepted without one; a grant while active and a withdrawal while inactive refused with nothing written; the write limit, the reserve for invalidating writes, and its exhaustion closing billing that was open; a clock that goes backwards between two reads and between a read and a write reviving nothing; the tenant boundary; nothing written by a refused write or a read.
- Properties with a fixed seed over random histories of attempted writes and clock advances, against a reference model written in the test that decides from the attempts which are accepted and what holds: the same writes are accepted; the readiness equals the model's; billing reads open only when the model's own predicate says so; immediately after a successful opening billing reads open; billing goes from not open to open only through an accepted `OpenBilling`; no stage before `billing_open` goes from stale or missing to holding without a `Reach`; billing is never open while a required run is of an earlier generation than the last change that named its stage or an earlier one, is expired, or rests on an ended grant, while first data or coverage is empty, while a `setup` grant is active, while the operational read was recorded under one, or once the reserve is used up; no stage holds on a run recorded before its floor, with the floors kept in the test from the accepted writes; no read changes the sequence.
- Concurrency under the race detector: concurrent writes with the same expected sequence, of which exactly one is written.
- Mutation check by a script in Python, which runs wherever the toolchain does, with per-row results appended to files that are never overwritten.
- `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level after this ticket: the source cases of HB-04 (ON03 to ON06, ON09, PO01 to PO10, FD01, FD02, FD08), AUD-DB-02, AUD-DB-08 and AUD-QA-07. This ticket does not complete W-04.

# Deferred

`CapabilityEvidence` and `SourceCoverage`; provider profiles and their branches (DB-02); handoff links, and consent for created resources, cost and policy changes (DB-08); a link between a project and the project it branched from, and a mark for an absent original (PO09); the parent's access as a linked record; discovery by cursor; the relation to the Python import path; a read-time check of the product version; retention; commands, permissions and screens S01, S02; persistence.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL, not implementable as written, without asking for a redesign: 6 HIGH, 10 MEDIUM and 1 LOW, among them five sequences in which billing would read as open, or could be reopened, on evidence the plan claimed to exclude. All seventeen were adopted in the sections above.

| Finding | Resolution |
| --- | --- |
| D1 a role granted again revived evidence | A run is bound to the grant it ran under; `Reach` refuses a run whose grant ended |
| D2 a return to an old hash revived old runs | Runs belong to a configuration generation; `Reach` refuses earlier generations |
| D3 billing open on empty data | One predicate for opening and for reading, covering both stages that can be empty |
| D4 re-reaching undefined; billing restored by itself | The latest record is current under stated conditions; a record that is not a renewal invalidates later stages, the opening included |
| D5 the write limit refusing the write that invalidates | The reserve covers every invalidating write; what happens at the limit stated |
| D6 caller-stated, unbounded times | Every time is the registry's; validity and setup duration bounded by required limits |
| D7 the stored opening against the computed answer | The stage status of `billing_open` is the computed answer; progress without gaps; empty stages flagged |
| D8 the version check vacuous | Checked when a run is recorded; the read-time check stated as not built, with its reason |
| D9 the setup acceptance shown only by construction | Billing needs setup to be gone and the operational read recorded without it; the recorded-only condition stated |
| D10 role operations underdefined | Grants as numbered records, half-open intervals, refusals with nothing written |
| D11 the parent path | Its limits and the provisional stage set stated |
| D12 the trusted first stage; any change closing billing | Stages outside the path refused; the initial hash; the limit on AUD-DB-08 and the provisional rule stated |
| D13 withdrawal against a busy project; retries | Invalidating writes take no expected sequence; write identifiers for every write |
| D14 the outcome matrix | Stage by outcome table; the alternative never the own path, or none; the failure code an identifier; the PO09 gap stated |
| D15 baseline and deferral wording | What is and is not built of the proposed direction stated; the contract called provisional |
| D16 verification | The model decides acceptance from attempts; the properties and literal sequences added |
| D17 smaller items | Reason precedence, identifier formats, duplicate sources, unbounded workspaces and the script's portability stated |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned FAIL, again without asking for a redesign: the five unsafe sequences of the first review closed, eleven of the seventeen findings resolved and six partly, and nine new findings, 2 HIGH, 4 MEDIUM and 3 LOW. It found billing coming back without an opening after a setup grant ended, and the test of "recorded without setup" unsafe when two writes carry the same time. All nine were adopted in the sections above.

| Finding | Resolution |
| --- | --- |
| E1 billing coming back when a setup grant ends | Granting `setup` invalidates the opening; the property that billing opens only through `OpenBilling` added |
| E2 the setup test decided by times that can tie | Decided at the write and stored on the run, as every such fact now is |
| E3 a run made before an upstream replacement reaching the stage again | `Reach` needs a run recorded after the stage's latest invalidation |
| E4 the reserve failing open | `billable` is false once the reserve is used up |
| E5 the status of `billing_open` not total | One ordered list of reasons |
| E6 verification | The contradicted property corrected; the missing properties and sequences added |
| E7 the outcome table | `billing_open` excluded from runs; empty defined; coverage and first data must agree |
| E8 undefined small cases | Creation, repeats, validity interval, positive durations, the file path and the setup grant as a progress write stated |
| E9 the renewal cliff | Listed among the provisional rules |

# Design third review resolution

The third narrow review of 2026-11-13 (read-only, nothing executed) returned FAIL again, on one sequence: no case of billing open on excluded evidence, but billing still going from not open to open without an opening, by replacing the operational read with one recorded under setup and then with one recorded without it. Six of the nine second-round findings were resolved and three partly; six new findings, 1 HIGH, 3 MEDIUM and 2 LOW. All were adopted, the first by replacing case-by-case invalidation with one rule.

| Finding | Resolution |
| --- | --- |
| F1 billing reopening through renewals of the operational read | Every write invalidates an opening that it finds not billable, before or after it is applied |
| F2 the registry's time undefined for reads | The highest reading of any operation, under the lock |
| F3 a spare run for a stage never reached escaping the cascade | Floors on every later required stage, reached or not |
| F4 verification | The reserve clause, a floor property and the literal sequence added |
| F5 the reserve formula | Stated |
| F6 small items | The order of refusals, roles on a file path and the wording for the opening stated |

A fourth pass, limited to this round's changes, returned PASS with findings on 2026-11-13: implementation may start; no blocker or HIGH, 2 MEDIUM and 4 LOW clarifications, all adopted above (only accepted writes perform the two checks; the stored reason is the first and names the reserve when that is the cause; the wording of renewals, floors and the explanation). By case analysis it found no way for billing to go from not open to open without an accepted `OpenBilling`. That property is also checked on every random history, with the model's time advanced on every attempted operation, reads and refusals included.

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, no HIGH, 2 MEDIUM and 8 LOW. By case analysis of the code it found no sequence in which billing reads as open or is opened on expired, ended, invalidated or empty evidence, under an active setup grant, on a read recorded under setup or with the reserve used up, none in which billing goes from not open to open without an accepted `OpenBilling`, no refused path that changes a project, and no path to another workspace's project.

| Finding | Resolution |
| --- | --- |
| C1 the order of refusals differed from the plan | The plan now states the implemented order; a table test with two causes per case added |
| C2 generic oracles weaker than claimed | A quarter of the random histories run with a small write limit so that the reserve is reached, with a minimum count; every open read checks that each stage's run rests on an active grant, from the grants and runs the registry reports |
| C3 "nothing is deleted" not true of stage records, the opening and generations | The sentence narrowed to what is kept |
| C4 the clock not read by every operation | The exception stated |
| C5 the reason when a change uses up the reserve | Stated: the first reason is kept; literal test added |
| C6 weak or vacuous assertions | The empty assertion removed, discarded openings asserted, the parent path added to the outcome table test |
| C7 missing literal tests | Added: refused writes and a read of the runs moving the time, expected-sequence conflicts on a grant and an opening, a malformed run name, every role on a file path, a creation repeated after a configuration change, a withdrawal of an expired setup grant, a strictly older run |
| C8 the kind comparison of repeats cannot be distinguished | Kept as defence and recorded as such in the mutation check |
| C9 "holds" written twice | One function gives the status of a stage; holding and the view both use it |
| C10 the write bound computed in two ways; an undeclared reason | One function for the bound; the reason declared |

The empty-coverage case of the outcome table stays unasserted there, because whether it reaches depends on what first data holds at that moment; it is asserted in the test of empty data.
