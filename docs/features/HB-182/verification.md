---
type: VerificationRecord
title: Onboarding readiness verification
description: Records what was executed and reviewed for the first model of work item W-04, with results, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-182](https://linear.app/hybrid-billing/issue/HB-182/w-04-onboarding-projects-readiness-stages-with-bound-evidence) adds the Go package `modules/onboarding` from main `29c73eb`: onboarding projects per workspace on one of three paths; diagnostic runs with a closed list of distinguished outcomes, each bound when it is recorded to the configuration generation, to the grant of the role it ran under and to whether setup access existed; readiness stages stored separately, each resting on one run; configuration changes that invalidate from a named stage on; three access roles as separate grants; and one predicate that decides both whether billing may be opened and whether it reads as open, with every accepted write invalidating an opening that it finds not billable. See the [plan](plan.md).

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan four times before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package: it connects to no provider, holds no credential and runs no diagnosis, and it must not be exposed directly. Capability evidence and source coverage, provider profiles (DB-02), handoff links and consent (DB-08) are not part of this ticket. The source cases of HB-04, AUD-DB-02, AUD-DB-08 and AUD-QA-07 remain NOT_RUN at system level, and W-04 is not complete.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All thirteen packages `ok`, including `modules/onboarding`; run again after the last correction |

What the acceptance asks, shown by literal tests with a test clock (`registry_test.go`):

| Asked | What the test shows |
| --- | --- |
| Insufficient permission, normal no-usage, waiting for the first export, an unsupported period and a partial file are distinguished | Each is its own outcome, accepted only on the stages where it can occur, on connected and file paths; only a passing outcome reaches a stage, and no-usage reaches first data and coverage as empty |
| Operational read after the setup permission is withdrawn | Withdrawing or expiring setup changes no stage and leaves the export writer active; billing is refused while setup is active and on a read recorded during it, also when the run and the withdrawal carry the same clock reading, and opens after a read recorded without it |
| A stale diagnosis cannot open billing (DB-08) | Opening is refused at the exact instant evidence expires, with every stale stage listed; billing that was open reads as not open from that instant |
| A changed configuration cannot open billing (DB-08) | A change from each of the seven stages invalidates that stage and the later ones only; a run from before the change is refused afterwards, also after a return to the old hash |
| Withdrawing the collector's access | Every stage becomes stale; a new grant revives nothing; old runs are refused |
| Billing never comes back by itself | After a lapse, a setup grant that then expires or is withdrawn, or a read replaced by one recorded under setup and replaced again, billing stays closed until it is opened again |

Further literal tests: projects, repeats and conflicts of creation, the three paths and their stages, a file-path project opening on four evidence stages with no roles; fourteen malformed runs, the validity limit, another product version, a role without a grant; stages reached out of order with the prerequisite flag and the gap-free progress; every refusal of `Reach`, among them a run of another stage or generation, an expired run, a run under setup or the writer's access, a run under an ended grant, a run older than the current one, a spare run recorded before an upstream stage was replaced, for a stage that had been reached and for one that had not, and coverage disagreeing with first data on emptiness; ten malformed grants, the setup duration limit, each withdrawal condition, a grant while one is active, withdrawals of roles that are not active or expired; empty data blocking billing and a change of emptiness closing it; repeats of every kind of write with identical and other content and across kinds; the order of refusals when two causes apply; the write limit, the reserve for the two invalidating writes, its exhaustion closing billing, and the first reason kept when a change is the last reserve write; the registry's time moved by creations, reads and refused writes and never moving back; the tenant boundary; a registry that is not built.

Property test (`property_test.go`, fixed seed): 400 histories of 110 steps over the three paths, a quarter of them with a write limit small enough to be reached, half of the steps steered towards readiness so that billing is actually opened. A reference model written in the test decides, from each attempted write, whether it is accepted and what the readiness then is; after every step the registry's acceptance and its whole readiness equal the model's. Independently of the model, from what the registry reports: billing goes from not open to open only through an accepted opening and reads open right after one; no stage holds again without an accepted `Reach` of that stage; no holding stage rests on an expired run or on a run recorded before its floor, with the floors kept by the test; while billing reads open, no stage is missing, stale or empty, no run rests on a grant that is not active, no setup grant is active, the operational read was not recorded under setup, and the reserve is not used up; no read and no refused write changes the sequence. Reached: 7,532 runs and 4,723 reaches accepted, 980 configuration changes, 1,124 grants and 700 withdrawals, 609 openings and 7,719 refused ones, 10,830 reads with billing open, 512 closings of billing that was open, 838 steps with the reserve used up.

Concurrency under the race detector: twenty-four writes with the same expected sequence, of which exactly one is written.

Limits of these tests: the model is the same rules written a second time by the same author and shares the stage order and the required stages with the package, so a shared misreading is invisible to it; the property looks at no error kind, no blocker list, no repeat and only one project; the case of the reserve used up while billing was open was not reached at random and is shown only by the literal test; expectations in the tables were worked out by hand.

# Mutation check

111 single-line mutations of `modules/onboarding` were applied one at a time by a Python script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten.

History:

- First run, before the change review: 104 caught, 4 not caught, 3 did not compile. Two of the 4 showed missing tests, which were added: a creation reading the clock, and a duration or a condition on a role that takes none. The 3 were rewritten so that they compile and were caught.
- After the change review's corrections, 12 rows were rewritten for the changed code and all 111 were run again from the start.
- Result: 109 of 111 caught, 2 not caught.

The 2 that are not caught cannot change a result; the change review confirmed the first by reading:

| Mutation | Why no test distinguishes it |
| --- | --- |
| 46, a stored write identifier answered as a repeat for another kind of write | The content of every kind of write has its own type, so the comparison of the content already tells the kinds apart; the comparison of the kind is kept as defence against a later kind that reuses a type |
| 69, no check of the opening before a run is stored | Recording a run changes nothing the predicate reads, so the check after it finds exactly what the check before it would have found, with the same reason |

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | A file path with all seven stages | Caught |
| 2 | Any path accepted | Caught |
| 3 | The parent path treated as a file path | Caught |
| 4 | An outcome that did not pass reaches a stage | Caught |
| 5 | Normal no-usage reaches nothing | Caught |
| 6 | No-usage on any stage | Caught |
| 7 | Insufficient permission on a file path | Caught |
| 8 | Awaiting a first export on a file path | Caught |
| 9 | Awaiting a first export on any stage | Caught |
| 10 | An unsupported period on any stage | Caught |
| 11 | A partial file on any stage | Caught |
| 12 | An unsupported scope on any stage | Caught |
| 13 | Any role accepted | Caught |
| 14 | Any withdrawal condition | Caught |
| 15 | No reserve required | Caught |
| 16 | No longest validity required | Caught |
| 17 | Identifier length unbounded | Caught |
| 18 | A withdrawn grant stays active | Caught |
| 19 | A setup grant never expires | Caught |
| 20 | A setup grant active at its expiry | Caught |
| 21 | The registry's time goes backwards | Caught |
| 22 | A creation does not read the clock | Caught |
| 23 | A repeated creation with another source | Caught |
| 24 | A repeated creation with another path | Caught |
| 25 | A repeated creation with another hash | Caught |
| 26 | Projects per workspace unbounded | Caught |
| 27 | Projects counted across workspaces | Caught |
| 28 | An invalidated stage holds | Caught |
| 29 | Expired evidence holds | Caught |
| 30 | Evidence under an ended grant holds | Caught |
| 31 | Evidence holds at its expiry | Caught |
| 32 | Billing on empty data | Caught |
| 33 | Billing with a missing stage | Caught |
| 34 | Billing with a stale stage | Caught |
| 35 | Billing while setup is active | Caught |
| 36 | Billing on a read recorded under setup | Caught |
| 37 | Billing when the reserve is used up | Caught |
| 38 | Prerequisites always shown as held | Caught |
| 39 | The reason of an invalidation not shown first | Caught |
| 40 | An ended grant shown as an expiry | Caught |
| 41 | Empty stages not flagged | Caught |
| 42 | The reserve never counted as used up | Caught |
| 43 | A later reason replaces the first | Caught |
| 44 | The opening survives a moment without its conditions | Caught |
| 45 | A stored identifier repeats any content | Caught |
| 46 | A stored identifier of another kind | Not caught |
| 47 | A conflicting identifier written again | Caught |
| 48 | Expected sequence unchecked | Caught |
| 49 | Every write admitted to the reserve | Caught |
| 50 | No reserve for the writes that invalidate | Caught |
| 51 | Writes unbounded | Caught |
| 52 | The reserve not named as the reason | Caught |
| 53 | No check of the opening after a write | Caught |
| 54 | A run that examines the opening | Caught |
| 55 | Outcomes on any stage | Caught |
| 56 | A validity that is not positive | Caught |
| 57 | A detail that is text | Caught |
| 58 | A failure without a code | Caught |
| 59 | A failure code that is text | Caught |
| 60 | The own path as the alternative | Caught |
| 61 | An unknown alternative | Caught |
| 62 | An alternative on another outcome | Caught |
| 63 | Roles of runs unchecked | Caught |
| 64 | Validity unbounded | Caught |
| 65 | A run by another product version | Caught |
| 66 | Every run valid for the longest time | Caught |
| 67 | A run under a role without an active grant | Caught |
| 68 | Runs never marked as recorded under setup | Caught |
| 69 | No check of the opening before a run is stored | Not caught |
| 70 | The opening reached like a stage | Caught |
| 71 | A run of another stage as evidence | Caught |
| 72 | A failing run as evidence | Caught |
| 73 | A run of an earlier generation as evidence | Caught |
| 74 | An expired run as evidence | Caught |
| 75 | Evidence under the setup or the writer's access | Caught |
| 76 | Evidence under an ended grant | Caught |
| 77 | An older run replaces the current evidence | Caught |
| 78 | A run from before the floor | Caught |
| 79 | Coverage and first data disagree on emptiness | Caught |
| 80 | Empty coverage while first data does not hold | Caught |
| 81 | Billing comes back when a lapsed stage is reached | Caught |
| 82 | A change of emptiness treated as a renewal | Caught |
| 83 | Reaching a stale stage treated as a renewal | Caught |
| 84 | No floor after an upstream replacement | Caught |
| 85 | Later stages survive an upstream replacement | Caught |
| 86 | The opening survives an upstream replacement | Caught |
| 87 | A change naming a stage outside the path | Caught |
| 88 | A change without a new generation | Caught |
| 89 | The named stage survives a change | Caught |
| 90 | A change invalidates the earlier stages too | Caught |
| 91 | The wrong reason after a change | Caught |
| 92 | A change leaves billing open | Caught |
| 93 | The new hash not stored | Caught |
| 94 | Setup without a duration | Caught |
| 95 | Setup without a condition | Caught |
| 96 | A duration or condition on another role | Caught |
| 97 | Setup duration unbounded | Caught |
| 98 | A role on a file path | Caught |
| 99 | A second grant while one is active | Caught |
| 100 | Setup grants without an expiry | Caught |
| 101 | The wrong reason after a setup grant | Caught |
| 102 | A withdrawal of a role that is not active | Caught |
| 103 | A withdrawal that withdraws nothing | Caught |
| 104 | Billing opened whatever the evidence | Caught |
| 105 | Billing opened twice | Caught |
| 106 | Billing reads open on evidence that lapsed by the clock | Caught |
| 107 | Billing reads open on an invalidated opening | Caught |
| 108 | The reason of the opening's invalidation not shown | Caught |
| 109 | Progress counts stages beyond a gap | Caught |
| 110 | An expired setup grant shown as active | Caught |
| 111 | The grant of a run not shown | Caught |

# Deviations from the plan

- The order of refusals was implemented differently from the reviewed text (identifiers, stored identifier, expected sequence and limit before the operation's own input); the plan was changed to the implemented order after the change review.
- Replaced stage records, earlier openings and earlier configuration hashes are not kept; the plan now says so.
- An operation refused for a malformed identifier does not read the clock.

# Provisional choices and undecided matters

The owner has not decided these; the implementation uses the labelled rules of the plan: the stage set of each path, with all seven on the parent path; evidence counted only from runs under the collector's access; billing refused on empty first data or coverage, which blocks a customer with legitimately no usage; billing reading as not open as soon as evidence expires, with a renewal needed before the lapse; billing refused while setup access is active and on a read recorded during it; every configuration change closing billing; the stage a change affects taken from the caller; the validity of a diagnosis stated per run within a limit; no read-time check of the product version.

A consequence worth the owner's attention: a stage reached for the first time, or again after a lapse, invalidates every stage reached after it in the order, so onboarding has to be shown in order, and a missed renewal of an early stage costs a new diagnosis of every later one and a new opening.

# Limits

- The package authorizes nothing and is reachable only in process; it must not be exposed before commands exist for it.
- Exposure blocker: nothing is evicted; a project that keeps renewing its evidence reaches its write limit, after which billing reads as not open and cannot be reopened in this registry.
- A run is dated when it is reported, not when the diagnosis was made.
- Which stage a configuration change affects is the caller's word; a caller that names only the opening can open billing again at once.
- The withdrawal condition of a setup grant is recorded and not enforced.
- On the parent path the parent's own access is not linked; on a file path evidence is bound to no access.
- A file-import project standing in for a missing original carries no mark of it (PO09 not met).
- References and holders are not checked against the workspace; an identifier can still be a secret or a name of the allowed shape.
- The clock is called while the registry's lock is held.
- State is in memory and lost on restart.
- The per-row mutation results are in the session's scratch files, not in the repository.

# Independent review

The plan was reviewed four times on 2026-11-13, each pass read-only and executing nothing. The first returned FAIL (6 HIGH, 10 MEDIUM, 1 LOW): as written, billing could read as open or be reopened on evidence the plan claimed to exclude, in five sequences; the plan was rewritten. The second returned FAIL on a narrower scope (2 HIGH, 4 MEDIUM, 3 LOW): billing came back without an opening when a setup grant ended, and one rule compared times that can tie. The third returned FAIL on one sequence (1 HIGH, 3 MEDIUM, 2 LOW): billing reopening through two renewals of the operational read; the case-by-case rules were replaced by one rule. The fourth, limited to that change, returned PASS with findings (2 MEDIUM, 4 LOW) and said implementation may start.

The change review of the same day returned PASS with findings: no blocker, no HIGH, 2 MEDIUM and 8 LOW. By case analysis of the code it found no sequence in which billing reads as open or is opened on excluded evidence or comes back without an opening, no refused path that changes a project, and no path to another workspace. All ten findings were handled as the plan's change review resolution lists. The corrections made after it (one function for a stage's status, one for the write bound, a declared reason, tests and the property's added oracles) were not reviewed again; they are covered by the tests and by the full mutation run after them.

Every executed result in this record is the author's.
