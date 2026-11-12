---
type: VerificationRecord
title: Permission registry verification
description: Records what was executed and reviewed for the first authorization model of work item W-03, with results, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-169](https://linear.app/hybrid-billing/issue/HB-169/w-03-permission-registry-with-scoped-grants-delegation-and-support) adds the Go package `modules/auth` from main `8dccf40`: a closed action set, scoped grants with explicit selectors and periods, field classes, delegation chains evaluated at decision time, support sessions, client intersection, an approver-independence check, three external codes with audit reasons, and an authority stamp with recheck, held in an in-memory registry with expected versions. Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini. See the [plan](plan.md).

Nothing calls the package yet. It authenticates nobody, stores nothing durably and guards no entry point. It does not complete W-03: the acceptance cases SEC01, GH04, EX02, EXT02, REV05, AUD-AR-09, AUD-QA-04 and AUD-QA-08 remain NOT_RUN at system level.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All six packages `ok`, including `modules/auth` |
| `go test -fuzz FuzzDecideNeverAllowsWithoutAGrant -fuzztime 20s ./modules/auth/` | PASS, about 4.3 million executions in the final run, no failure. The target only exercises a registry without grants; its value is panic detection |

Literal table tests cover: each action needing its own grant and kind; invalid requests; scope, field, period and validity bounds including the exact instants and the exclusive end month; malformed registrations and record identity; principals (installation, epoch, suspension, reactivation) and unavailability for `Decide`, `Recheck`, `CheckApproval`, `RecheckApproval` and session registration; the download conjunction; client intersection; delegation chains with the AUD-AR-09 refusals at registration (re-delegation not allowed; a widened customer, legal entity, contract, field set, period in both directions and validity in both directions; another action or kind; a revoked or expired parent; a suspended or service delegator) and at decision (another kind, month or contract than delegated; a revoked parent); the delegations used listed in sorted order; delegations ending with their backing (root grant lost and granted again, suspended delegator in the middle and at the root, expiry by clock, delegated download, mixed direct and delegated fields, chain length); support sessions and their registration, including the approver's and actor's state, narrowed approver grants and a second-level delegate of the actor; approver independence and approval recheck; stamps for direct, delegated and session principals (revocation of the delegation or its parent, expiry by clock, session end, epoch change, unrelated writes, reuse for another request, another instance); records that cannot be closed twice and epochs that cannot move back; an unconstructed registry failing closed on decisions and writes; repeating the current availability keeping stamps; stored terms not aliasing the caller; and a concurrent revocation under the race detector, where each of 8 readers makes 500 decisions after the revocation has returned.

The tests are in `registry_test.go` and `rules_test.go`.

Property tests in `property_test.go` use fixed seeds and fail when a category is rarely reached: 24,000 decisions over 600 generated registries compared with a reference model written in the test (1,526 allowed, 231 of them through a delegation), with the single-field subset property on every allow; removal of authority never adding an allow over at most 300 registries (rounds that draw a delegation revocation without a delegation are skipped; 92 allows were lost to a removal); grants in another workspace never changing the allow bit over 300 registries; and client decisions equal to the conjunction of the subject's reference decision and the client's own grants over 300 registries (229 allowed).

Limits of the reference model, as the change review noted: it is independent for the decision rule but takes the set of delegations from whatever registration accepted; the generator varies action, kind, customer, fields and workspace but keeps legal entity, contract, period and validity constant, and those dimensions rest on the literal tests; the other-workspace property writes grants only.

The reference model failed once during development on a correct implementation: it required both download conjuncts to come from the same source. The plan's rule evaluates each conjunct on its own, so the model was corrected, not the implementation.

# Mutation check

118 single-line mutations of `types.go`, `decide.go` and `registry.go` were applied one at a time by a script, and the package tests were run after each. Each row names the fault the mutation introduces. Rows were renumbered as the list grew.

History, including what the first attempts missed:

- First run, 84 mutations: 66 caught, 13 not caught, 5 did not compile. Test cases were added for 9 of the 13, the 5 were rewritten so that they compile, and 2 aliasing mutations were added. Second run of 86: 82 caught.
- The independent change review then showed that the list was narrower than the areas this record claimed: it had no mutation for the delegation kind and period at decision time, legal-entity and contract containment, bounded-in-bounded periods, a child's start of validity, a revoked or expired parent at registration, the delegator's state, or the epoch rule, and no test distinguished those faults. The tests were added first; then one aliasing mutation that the code change made obsolete was removed and 29 mutations were added (114).
- After the re-review 4 mutations for the unconstructed registry and for repeating the current availability were added (118).
- Final run on the corrected code: 114 caught, 4 not caught. The final run was made in two parts, rows 1 to 99 and rows 100 to 118, because the script process was interrupted during row 100 and left that mutation in `registry.go`; the line was restored by hand, the package tests were confirmed to pass, and the remaining rows were run.

The 4 that are not caught remove a second line of defence whose effect another check already produces, so no test can tell them apart:

| Mutation | Why no test distinguishes it |
| --- | --- |
| 1, a `one` selector matching an empty resource identifier | A registered selector identifier is never empty |
| 25, the decision ignoring the re-delegation flag of an ancestor | Registration refuses a child under a parent without the flag, and delegations are immutable |
| 44, `Recheck` not testing the approval flag separately | The flag is part of the compared stamp value |
| 63, registration not testing delegator equals delegate | The chain check refuses a repeated subject with the same error |

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Selector matches an empty resource id | Not caught |
| 2 | Selector within lets all narrow under one | Caught |
| 3 | Period end inclusive | Caught |
| 4 | Period start exclusive | Caught |
| 5 | Period ignores a missing month | Caught |
| 6 | Unbounded period within bounded parent | Caught |
| 7 | Validity end inclusive | Caught |
| 8 | Validity start exclusive | Caught |
| 9 | Child validity may outlive parent | Caught |
| 10 | Scope ignores workspace | Caught |
| 11 | Scope ignores legal entity | Caught |
| 12 | Scope ignores contract | Caught |
| 13 | Direct ignores kind | Caught |
| 14 | Direct ignores revocation | Caught |
| 15 | Direct ignores field | Caught |
| 16 | Direct ignores client id | Caught |
| 17 | Covers ignores delegation revocation | Caught |
| 18 | Covers ignores delegation expiry | Caught |
| 19 | Covers ignores suspended delegator | Caught |
| 20 | Covers ignores delegate workspace grant | Caught |
| 21 | Covers ignores delegation fields | Caught |
| 22 | Covers ignores delegation scope | Caught |
| 23 | Root delegator need not hold the grant | Caught |
| 24 | Root need not hold view for delegated download | Caught |
| 25 | Decision ignores re-delegation flag | Not caught |
| 26 | Subject download without view conjunct | Caught |
| 27 | Client download without view conjunct | Caught |
| 28 | Client grants not required | Caught |
| 29 | Suspended client accepted | Caught |
| 30 | Suspended subject accepted | Caught |
| 31 | Stale epoch accepted | Caught |
| 32 | Installation not checked | Caught |
| 33 | Decide may decide approve | Caught |
| 34 | Client plus session accepted | Caught |
| 35 | Session allows other actions | Caught |
| 36 | Session ignores kind | Caught |
| 37 | Session ignores customer | Caught |
| 38 | Session ignores workspace | Caught |
| 39 | Session ignores fields | Caught |
| 40 | Session usable by another subject | Caught |
| 41 | Session ignores expiry | Caught |
| 42 | Session ignores ending | Caught |
| 43 | Recheck ignores stamp equality | Caught |
| 44 | Recheck accepts approval stamp | Not caught |
| 45 | Approval recheck ignores stamp equality | Caught |
| 46 | Stamp without generation | Caught |
| 47 | Stamp without instance | Caught |
| 48 | Stamp without author | Caught |
| 49 | Stamp without customer | Caught |
| 50 | Stamp without fields | Caught |
| 51 | Self approval allowed | Caught |
| 52 | Approval through client | Caught |
| 53 | Approval inside session | Caught |
| 54 | Service approver | Caught |
| 55 | Unknown author accepted | Caught |
| 56 | Approval check for other actions | Caught |
| 57 | Decide ignores unavailability | Caught |
| 58 | CheckApproval ignores unavailability | Caught |
| 59 | Approve grant to a service subject | Caught |
| 60 | Approve grant to a client | Caught |
| 61 | Grant with subject and client | Caught |
| 62 | Approve delegable | Caught |
| 63 | Self delegation | Not caught |
| 64 | Service delegate | Caught |
| 65 | Parentless delegation unbacked | Caught |
| 66 | Backing ignores scope | Caught |
| 67 | Backing ignores period | Caught |
| 68 | Registration ignores re-delegation flag | Caught |
| 69 | Registration ignores parent delegate | Caught |
| 70 | Child action differs from parent | Caught |
| 71 | Child scope wider than parent | Caught |
| 72 | Child period wider than parent | Caught |
| 73 | Child validity beyond parent | Caught |
| 74 | Child fields wider than parent | Caught |
| 75 | Chain length unbounded | Caught |
| 76 | Subject repeated in chain | Caught |
| 77 | Delegate without workspace grant | Caught |
| 78 | Actor approves own session | Caught |
| 79 | Actor's delegate approves session | Caught |
| 80 | Session approver without grant | Caught |
| 81 | Session approver stale epoch | Caught |
| 82 | Session approver through client | Caught |
| 83 | Session with cost or no fields | Caught |
| 84 | Stored session fields alias the caller | Caught |
| 85 | Session without customer | Caught |
| 86 | Delegation kind ignored at decision | Caught |
| 87 | Delegation period ignored at decision | Caught |
| 88 | Chain of a non-field action not sorted | Caught |
| 89 | Unconstructed registry calls a nil clock | Caught |
| 90 | Stale stamp reported as principal | Caught |
| 91 | Backing ignores kind | Caught |
| 92 | Child kind differs from parent | Caught |
| 93 | Containment ignores legal entity | Caught |
| 94 | Containment ignores contract | Caught |
| 95 | Child period may start earlier | Caught |
| 96 | Child period may end later | Caught |
| 97 | Child validity may start earlier | Caught |
| 98 | Child under a revoked parent | Caught |
| 99 | Child under an expired parent | Caught |
| 100 | Suspended or service delegator | Caught |
| 101 | Epoch moved back | Caught |
| 102 | Reactivation without a new epoch | Caught |
| 103 | Subject kind changed | Caught |
| 104 | Grant revoked twice | Caught |
| 105 | Delegation revoked twice | Caught |
| 106 | Session ended twice | Caught |
| 107 | Revoked delegation still disqualifies approver | Caught |
| 108 | Second-level delegate of the actor approves | Caught |
| 109 | Session approver from another installation | Caught |
| 110 | Session approver inside a session | Caught |
| 111 | Suspended or service session approver | Caught |
| 112 | Suspended or service session actor | Caught |
| 113 | Stored delegation terms alias the caller | Caught |
| 114 | Stored grant terms alias the caller | Caught |
| 115 | Repeating the current availability changes the generation | Caught |
| 116 | An unconstructed registry can be made available | Caught |
| 117 | Subject write on an unconstructed registry | Caught |
| 118 | Client write on an unconstructed registry | Caught |

# Provisional choices and undecided matters

The owner has not decided these; the implementation uses the labelled defaults of the plan and nothing depends on them yet: the action names and the two policy actions; the download conjunction and its root rule under delegation; `approve` not delegable (the re-review named this the more conservative option; AR-09's substitute approver needs an own direct grant under it); human-only delegation; the same-workspace delegate rule for the open cross-group question; clients never approving and otherwise grantable any action; support limited to `view` of general fields with point-in-time approval; chain length four; month granularity; the registry-wide generation in the stamp. Open decisions of HB-03 are unchanged: SSO provider, approval ceilings, separation-of-duties exceptions and cross-group viewing delegation.

# Limits

- The package trusts its caller for the authenticated principal, the resolved resource and the author of an approval subject. Per-dimension matching does not verify that a contract belongs to the named customer.
- Registry writes are not themselves authorized; the layer that exposes them must be.
- One subject per human is assumed until identities exist (W-42).
- Any write invalidates every stamp in flight, which is fail-closed and a liveness risk under frequent writes.
- State is in memory and lost on restart. No database, row-level security or connection handling exists.
- `modules/customeraccess` remains the sole authority for the customer usage path; the two models are not combined.
- The field catalogue is not keyed by resource kind, and test catalogues are synthetic.
- The rewritten rules after the design re-review were not sent for a third design review before implementation.

# Independent review

The design review of 2026-11-12 returned FAIL (7 HIGH, 7 MEDIUM, 4 LOW) and the plan was rewritten. The re-review of the same day found those resolved and returned FAIL (narrow: 1 HIGH, 9 MEDIUM, 6 LOW); its corrected rules were adopted as listed in the plan. Both were read-only and executed nothing.

The independent change review of 2026-11-13 read the three source files, both test files of that time, the plan and this record, executed nothing, and returned PASS with findings: no blocker or high finding, 5 MEDIUM, 9 LOW. It reported that it could not construct a forbidden allow, a wrongly accepted stamp, a leak across workspaces or customers, a deadlock or an unbounded loop. The MEDIUM findings were: delegation IDs unsorted for non-field actions (corrected); eight delegation rules without a distinguishing test (tests added); stamp cases promised by the plan but absent (added); the epoch case not testing the epoch rule (added); and claims in this record that the tests did not support (the review sentence, the mutation areas and the AUD-AR-09 list, all corrected above). The LOW findings and their handling are in the plan's change review resolution. It did not read the mutation script, the rules files or the planning sources, and verified no execution.

The narrow re-review of 2026-11-13 read the corrected source, the three test files, the plan and this record, executed nothing, and returned PASS with findings: no high or medium finding, seven LOW. It checked the seven code corrections and found no wrong allow, no wrongly accepted stamp and no panic on an exported read path; it judged C1 to C7, C9, C11 and C14 resolved, C5, C10 and C12 partly resolved, and could not map C8 and C13 because the first report was not among its inputs. Its findings were corrected afterwards: writes on an unconstructed registry (R1), a missing test for the availability change (R2), wording and mutation history in this record (R3, R4), two test names (R5, R6) and unchecked setup errors (R7). Those last corrections were verified by the checks above and the mutation rows 115 to 118 and were not reviewed again.
