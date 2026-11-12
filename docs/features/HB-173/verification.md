---
type: VerificationRecord
title: Identity and permission binding verification
description: Records what was executed and reviewed for the binding of the identity registry to permission decisions, with results, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-173](https://linear.app/hybrid-billing/issue/HB-173/bind-identity-state-sessions-and-memberships-to-permission-decisions) binds `modules/identity` to `modules/auth` from main `b86dcd3`:

- `modules/auth` can be constructed with a source of subjects and memberships. With a source it reads subject kind, state and epoch from it, refuses its own subject writes, binds subject grants, delegations and support sessions to the membership they were registered under, and denies a decision or fails a registration during which the source changed.
- `modules/identity` offers a read-only view (subject, membership with a `joined` value, service owner and creator, a generation counter) and exposes its installation identifier and clock.
- `modules/gate` is new. It constructs the permission registry itself, never returns it, and reaches a decision only with an identity access answer: the answer is confirmed, the request's workspace must be the answer's, the permission registry decides, and an allow is confirmed once more.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review, which records its result). Machine: mac_mini. See the [plan](plan.md).

Nothing calls the gate yet. The customer usage path (`modules/customeraccess`) is not under it ([HB-174](https://linear.app/hybrid-billing/issue/HB-174/bring-the-customer-usage-path-under-identity-state-and-answer-based)) and must not be exposed. No acceptance case is run at system level.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All nine packages `ok` |

`modules/auth`: the tests that existed before this change were not edited and pass. New tests in `source_test.go` use a hand-written source: a nil source refused; a subject write refused without any change; the principal, the delegation chain, grant registration, the approval author, and the support actor and approver all read from the source; grants refused for non-members, unusable during a membership suspension, usable after reinstatement of the same membership, dead after removal and re-creation; the membership rule at the delegation root, the backing check, the delegate's workspace grant, the approver and the support approver; delegations and support sessions refused for non-members and not coming back after removal, re-invitation and a new grant; a stamp failing after a generation change with unchanged values; a decision, an approval check, a recheck and each of the three registrations during which the source changes, and the same registrations succeeding when tried again; a change at every single source read of a delegated decision, an approval check, a grant, a delegation with a parent and a support session, each noticed; the changed-source error before the identifier conflict in all three registrations; an approver without the grant learning nothing about the actor; a chain of two delegations ended by either delegator's membership.

`modules/identity`: the tests that existed before were not edited and pass. `view_test.go` walks one history through the operations of the three handles, successful and refused, and asserts for each whether the generation moved by exactly one or not at all, together with the subject, membership, `joined` and service answers of the view at each stage and the answers of an unconstructed view.

`modules/gate`, through real registries: a decision allowed for a member with a grant; ten changes that each deny the held answer and its stamp with `identity_not_current`; the zero answer, the zero stamp, and an answer and a stamp of another registry; new answers after reinstatement, a role change and a membership suspension keeping the grant; a removed and re-invited member needing a new grant; a request and a recheck naming another workspace or none; grants refused for an unknown user, a non-member and another workspace; service credentials (allowed, with a client, human-only actions, as approver, in a support session, at expiry, suspended, reinstated, revoked); clients; support sessions (the actor, an empty identifier, another user's session, rechecks, another workspace, an approver without the grant, a service approver, the zero answer, a non-member actor, the actor removed and invited again, a stale approver answer); rechecks of an approval and of a support decision after a membership change, a service stamp in a support recheck, and a second gate over the same identity registry having no grants; approvals (independent, self, unknown author, missing grant, another action, the ordinary decision refusing approve, rechecks, another workspace, a service author owned or created by the approver and one of somebody else, a departed author); delegations following the delegator's membership and user state, seven cases with the stamp taken before; zero values and construction; the administration's operations taking effect through the gate.

Not constructible, so not tested: a delegation or a grant in a second workspace, because only the bootstrap workspace can have members (HB-172); the corresponding checks rest on the permission package's own tests with the hand-written source.

Changes during a call, deterministically (`TestChangesDuringADecision`): a test clock runs a change once, on the second clock read of a call, which is the permission registry's. A session end, a role change, a credential revocation, and a session end during a recheck and during an approval check are each denied with `identity_not_current` and with the actor named, which only the second confirmation can produce. A suspension and a membership removal at the same read are denied by the permission registry itself (`principal_not_current`, `no_matching_grant`). The same clock ends an approver's session during `AddSupportSession`: the registration is undone, the session is unusable and its identifier consumed.

Import scan (`TestNoOtherPackageImportsThePermissionPackage`): every non-test Go file of the module outside the directories `modules/auth` and `modules/gate` is parsed for imports; none imports the permission package.

Property test (fixed seed): 80 histories of 120 steps over four members with grants, delegations, a client, user and membership state changes, removals, re-invitations and departures. A reference model written in the test keeps its own records (user state and epoch, membership state, `joined` and version, grants with the `joined` they were registered under, delegations with both values, the client) and never calls a recheck. After every step, six sampled recent answers are decided three ways each — alone, through the client, and as an approval of a random author — and reason and allow must equal the model's, including whether an allow came through a delegation; grant and delegation registrations must succeed exactly when the model says so. Counts of the final run: 20,833 decisions allowed, 1,772 of them through a delegation, 22,459 denied with `identity_not_current` and 6,262 with `no_matching_grant`; through the client 4,188 allowed and 16,645 denied for the client; approvals 10,574 allowed, 3,461 self-approvals, 13,060 without the grant; 1,865 grants and 439 delegations registered, 388 and 930 refused, 183 and 147 revoked; 79 removals, 20 re-invitations, 43 departures. The test fails when any of 21 categories is reached fewer than 20 times.

Limits of the reference model: one workspace; no support sessions, no service users, no rechecks of stamps, no chains of several delegations, no role changes, identity links, expiry or movement of time, no fields, and no change during a call; its rule for a delegated allow has the same structure as the package's; the generator has a growth phase of 45 steps and prefers recent answers.

Concurrency under the race detector (`TestDecisionsThatStartAfterAChangeNeverAllow`): six readers each decide once successfully, then keep deciding while the acting user is suspended, the membership is removed, the delegator is suspended, or the delegator's membership is suspended; 200 decisions of each reader that start after the change has returned are all denied. This is an opportunity for the race detector, not evidence for the second confirmation or the generation rule: the decisions after the change would be denied without them, and an overlap with the change is not guaranteed. The evidence for those two rules is the deterministic test above and the test that changes the source at every read.

# Mutation check

77 single-line mutations of the changed code in `modules/auth`, `modules/identity` and `modules/gate` were applied one at a time by a script, and the tests of the three packages were run after each. The table is one complete run on the code as it was after the change review, with rows 15 and 68 taken from a rerun after the re-review added the cases that catch them.

History of the earlier, partial runs:

- A first run of 70 mutations caught 49; 5 were not caught and 15 did not compile, because removing a check left a variable unused, and one more was not caught after being rewritten. The 15 were rewritten so that they compile. The complete result file of that run was overwritten by mistake with its summary, which is why the table below comes from a later run.
- Two of the rows not caught showed missing tests: a delegation recording the two membership values the wrong way round (all tests had equal values; a case with a delegate who joined later was added) and the adapter mapping every state to active (the acting user's suspension is caught by the answer and everyone else's by the membership, so a direct test of the adapter was added).
- After the change review the code changed and the list was brought up to it, with 7 mutations added for the review's findings (the position of both generation reads, the unavailable result, the order of the changed-source error and the conflict, the support actor's user state).
- Complete run: 71 of 77 caught, two of them (rows 61 and 77) because the mutated code made the tests hang (the test clock then acts inside the identity registry's lock), 6 not caught.
- The re-review showed that two of those six were not equivalences but missing assertions: a delegation to a non-member changed the error class when the delegator was not backed either, and a service answer in a support session was refused with the actor named. Cases were added and rows 15 and 68 are now caught. A case was also added behind row 77, which before was caught only by the hang.
- Result: 73 of 77 caught, 4 not caught.

The 4 that are not caught:

| Mutation | Why no test distinguishes it |
| --- | --- |
| 58, an unknown user mapped to a subject | The adapter then meets an empty kind and answers unknown anyway |
| 59, an unconstructed identity registry accepted by `gate.New` | Its empty installation identifier is refused by the permission registry |
| 73, the approval recheck skipping the service-author rule | An approval of a service author owned or created by the approver never gets a stamp, and ownership never changes |
| 74, a support session for another workspace than the approver's answer | Not reachable while only one workspace can have members: the approver then holds no approval grant in the other workspace and the permission registry refuses. With several workspaces this check is the only one that ties the session to the workspace the approver was admitted to, and it needs a test when a second workspace can exist, which [HB-175](https://linear.app/hybrid-billing/issue/HB-175/seed-and-recover-workspace-administrators-and-add-the-cross-workspace) records (a stated gap, not an equivalence) |

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Nil source accepted | Caught |
| 2 | Source not used | Caught |
| 3 | Subjects read from the registry's own records | Caught |
| 4 | Memberships never consulted | Caught |
| 5 | Joined value ignored | Caught |
| 6 | Inactive membership accepted | Caught |
| 7 | Source generation never read | Caught |
| 8 | Subject written beside a source | Caught |
| 9 | Grant registered for a non-member | Caught |
| 10 | Grant registered across a change | Caught |
| 11 | Grant does not record the membership | Caught |
| 12 | Backing check ignores the membership | Caught |
| 13 | Delegate's workspace grant ignores the membership | Caught |
| 14 | Delegation from a non-member | Caught |
| 15 | Delegation to a non-member | Caught |
| 16 | Delegation registered across a change | Caught |
| 17 | Delegation records the memberships swapped | Caught |
| 18 | Support session for a non-member actor | Caught |
| 19 | Support session registered across a change | Caught |
| 20 | Second generation read before the last source read of a support session | Caught |
| 21 | First generation read after the source reads of a decision | Caught |
| 22 | An unavailable result overridden by a change | Caught |
| 23 | Support actor's user state not required | Caught |
| 24 | Existing support session reported before a change | Caught |
| 25 | Existing delegation reported before a change | Caught |
| 26 | Existing grant reported before a change | Caught |
| 27 | Support session does not record the membership | Caught |
| 28 | Support approver epoch not read from the source | Caught |
| 29 | Grant matches without the membership | Caught |
| 30 | Client grants bound to a membership | Caught |
| 31 | Delegation ignores the delegator's membership | Caught |
| 32 | Delegation ignores the delegate's membership | Caught |
| 33 | Support session ignores the actor's membership | Caught |
| 34 | Unknown author accepted | Caught |
| 35 | Decision across a change accepted | Caught |
| 36 | Stamp without the source generation | Caught |
| 37 | Suspended principal current | Caught |
| 38 | Suspended user is an active member | Caught |
| 39 | Service user is a member of every workspace | Caught |
| 40 | Suspended membership is active | Caught |
| 41 | Human has a service record | Caught |
| 42 | Creator and owner swapped | Caught |
| 43 | Creation keeps the generation | Caught |
| 44 | Suspension keeps the generation | Caught |
| 45 | Reinstatement keeps the generation | Caught |
| 46 | Departure keeps the generation | Caught |
| 47 | Bootstrap keeps the generation | Caught |
| 48 | First administrator has no joined value | Caught |
| 49 | Linking keeps the generation | Caught |
| 50 | Unlinking keeps the generation | Caught |
| 51 | Acceptance keeps the generation | Caught |
| 52 | Re-created membership keeps its joined value | Caught |
| 53 | Membership change keeps the generation | Caught |
| 54 | Service creation keeps the generation | Caught |
| 55 | Service state change keeps the generation | Caught |
| 56 | Service user mapped to a human subject | Caught |
| 57 | Every state mapped to active | Caught |
| 58 | Unknown user mapped to a subject | Not caught |
| 59 | Unconstructed identity registry accepted | Not caught |
| 60 | Answer never confirmed | Caught |
| 61 | First confirmation skipped | Caught (the tests did not finish) |
| 62 | Workspace of the request not compared | Caught |
| 63 | Service answer through a client | Caught |
| 64 | Client dropped from the principal | Caught |
| 65 | Second confirmation skipped | Caught |
| 66 | Stamp without its answer | Caught |
| 67 | Empty support session accepted | Caught |
| 68 | Service answer in a support session | Caught |
| 69 | Zero stamp reaches the recheck | Caught |
| 70 | Recheck decides anew | Caught |
| 71 | Service author owned by the approver accepted | Caught |
| 72 | Service author created by the approver accepted | Caught |
| 73 | Approval recheck skips the service author rule | Not caught |
| 74 | Support session for another workspace than the approver's | Not caught |
| 75 | Support session kept although its approver was invalidated | Caught |
| 76 | Undone support session stays usable | Caught |
| 77 | Support approver's answer not confirmed | Caught (the tests did not finish) |

# Deviations from the plan

- `gate.New` takes the catalogue and the identity registry and reads installation identifier and clock from the registry, as planned; it refuses an unconstructed registry by its missing clock.
- A registration overtaken by a change of the source fails with the new error `ErrChanged`, checked before the existing-identifier conflict in all three registrations (after the change review; before it this held for grants only).
- The support-session registration checks the approver before the actor; an unavailable result is not replaced by the changed-source denial.
- `DecideInSupport` and `RecheckInSupport` refuse an empty identifier and a service answer before the answer is confirmed, with no actor named.
- The denial for a service-authored approval names the approver as actor and is produced inside the gate after the permission registry allowed.
- Two workspaces cannot be built through the identity registry, so the gate tests have one workspace; the cross-workspace cases are in the permission package's tests with a hand-written source.

# Provisional choices and undecided matters

The owner has not decided these; the implementation uses the labelled defaults of the plan: a grant dies when its membership is removed and does not return on re-invitation (an administrator must grant again), and delegations and support sessions likewise; a support actor must be a member of the session's workspace; an approver may not approve work authored by a service account it created or owns.

# Limits

- `modules/customeraccess` is not under the gate (HB-174).
- `Administration` is the root of all authority and is trusted from its holder; it proves neither the actor nor a delegator's consent.
- The client identifier and the author identifier are supplied by the caller's server-side code; the gate cannot check where they come from.
- `identity.Recheck` ignores idle time, so an interactive request must obtain its answer within the request.
- The generation is installation-wide: any change of any user or membership fails the outstanding stamps of all workspaces, and a failed recheck needs a new decision.
- An allow is true at one instant inside the call; a change that completes while the call is returning can be followed by that call's allow.
- Both constructors of the permission package remain exported; the import scan is what keeps other packages away from them.
- A second gate over the same identity registry would have its own grants; nothing prevents constructing one.
- State is in memory; no audit record is written.

# Independent review

The design review of 2026-11-13 returned FAIL (narrow: 3 HIGH, 9 MEDIUM, 5 LOW) and the plan was rewritten. A first attempt at the re-review was started against the previous text by mistake, because the rewrite had not been saved; it was stopped and discarded before it reported. The re-review of the saved text returned PASS with findings (5 MEDIUM, 6 LOW, all wording), which were folded in; the folded text was not reviewed a third time. All were read-only and executed nothing.

The independent change review of 2026-11-13 read the changed source of the three packages, the new tests, the plan and this record, executed nothing, and returned PASS with findings: no blocker or high finding, 4 MEDIUM and 12 LOW. It found no sequence in which a gate operation allows or a registration succeeds against the plan. Its findings and their handling are in the plan's change review resolution; two of the MEDIUM findings were claims of this record that the code or tests did not support, and one was the missing mutation table. It could not read the mutation justifications, which did not exist then.

The narrow re-review of 2026-11-13 read the corrected source, the tests, the plan and this record, executed nothing, and returned PASS with findings: no blocker, high or medium finding, six LOW. It walked every return of the two reordered registrations and found no wrong success, no leak and no commit without the second generation read; it found the test that changes the source at every read sound, and the order test distinguishing what it claims. It read the six justifications of mutations not caught and found three sound, one correctly declared a gap (row 74), and two that were not equivalences (rows 15 and 68). Its findings were handled afterwards: cases added for rows 15, 68 and 77; the count of added mutations corrected; the second-line-of-defence checks named in the plan; HB-175 created for the gap of row 74. These last corrections are three test cases and wording; they were verified by the checks above and a rerun of the three rows, and were not reviewed again.
