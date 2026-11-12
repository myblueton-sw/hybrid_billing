---
type: VerificationRecord
title: Identity registry verification
description: Records what was executed and reviewed for the first identity model of work item W-42, with results, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-172](https://linear.app/hybrid-billing/issue/HB-172/w-42-user-identities-invitations-memberships-sessions-and-service) adds the Go package `modules/identity` from main `4dfb5fd`: users with an immutable kind, a state and an epoch; external identities keyed by connection, issuer and subject; a one-time first administrator; pinned single-use invitations; workspace memberships with a last-administrator rule; sessions, access answers and rechecks; service users with expiring credential verifiers; three handles (Installation, Boundary, Workspace). Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review, which records its result). Machine: mac_mini. See the [plan](plan.md).

Nothing calls the package yet. It authenticates nobody, stores no secret and nothing durably, and is not wired to `modules/auth` ([HB-173](https://linear.app/hybrid-billing/issue/HB-173/bind-identity-state-sessions-and-memberships-to-permission-decisions)); neither package may be exposed alone. It does not complete W-42: the acceptance cases SEC01, EXT02, REV05, AUD-AR-09, AUD-DB-08 and AUD-QA-04 remain NOT_RUN at system level.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All eight packages `ok`, including `modules/identity` |

Literal table tests (`lifecycle_test.go`, `workspace_test.go`), with synthetic identifiers and verifiers and with every refusal asserted to show only the common text:

- identifier, triple and verifier rules; every limit required;
- bootstrap: before arming, malformed arguments, arming twice, wrong verifiers, the right verifier on the last allowed attempt and after the lock, both ends of the validity on the exact instant, a second bootstrap, user creation before it, eight concurrent calls of which exactly one succeeds;
- users: operator reference and reason, stale and repeated suspension, reinstatement and departure, sessions, answers, identities and memberships ended by a departure, identifiers and triples never reusable;
- identities: an unknown subject with a known email, another connection, linking with and without proof, by a dead and by an old session, a triple of another user, the window on the exact instant, the epoch increase ending the session used, unlinking through a later identity, of another user's identity, of the last identity, relinking;
- sessions: rejected lengths, a reused verifier, every refusal of the predicate, idle renewal forwards only and capped, both expiries on the exact instant, a clock before the opening, ending; the session limit counting current sessions only, also after suspension and reinstatement, and expired records dropped;
- recheck: thirteen changes that each make a held answer fail, the last valid instant, the zero answer, an answer of another registry with the same history, answers of the unlinking session and through the unlinked identity;
- invitations: malformed input, reuse, another workspace, another user, a dead session, guesses that do not count, the two allowed wrong verifiers and the lock, both ends of the validity, an existing member, revocation, nine ways in which the inviter's authority changes, and the pending limit counting what it should;
- memberships: every operation by an administrator, a member, an administrator elsewhere, on an absent target, with a stale version, to the same role and from the wrong state; versions across removal and re-invitation; listing fields, order, cursor and bounds;
- the last administrator: demotion, membership suspension, removal and departure, alone, with a second administrator, and with a second administrator suspended as a user or as a member; the sole member; an empty workspace; a remaining service user; a security suspension, the report, the refused departure afterwards, and reinstatement;
- service accounts: creation and owner rules, credentials (validity, limits, both ends on the exact instant, wrong verifiers never locking, revocation, rotation), suspension, reinstatement and retirement with expected epochs, the operator block, listing, and an account outliving its creator and a blocked administrator until the operator blocks it;
- sequences that must not recover what was closed (re-arming, a second bootstrap, relinking, reactivating a departed user, a membership outside bootstrap and acceptance);
- an unconstructed registry and nil or empty handles refusing all 31 operations without a panic;
- concurrency under the race detector: six sessions calling `Access` during a suspension, each answered at least once before it and none after it returned (300 calls each afterwards);
- a suspension followed by a departure while the other administrator's membership is suspended (refused) and while it is active (allowed); the operator blocking a service user an administrator suspended; credential slots freed by expiry at the exact instant and by revocation; a user of two workspaces; the epoch guard of a suspension and the version guard of a membership change at their maximum (the other counter guards are not tested) and a page as large as the largest integer; dead session records deleted after suspension, ending, linking, unlinking and departure.

Two workspaces: no operation can give a second workspace a member in this increment, so `TestWorkspacesAreSeparate` seeds a second workspace's first administrator directly in the registry's state. With it the tests cover an invitation of one workspace refused in the other, limits counted per workspace, memberships, service users and credentials unreachable and unlisted across workspaces, and the report for a workspace that holds only a service user. Tests write registry state directly only for this seeding (two tests), for a workspace that holds only a service user, and to set counters to their maximum; two tests read the session maps to count records and two read membership state.

Property test (`property_test.go`, fixed seed): 120 generated histories of 160 operations each over one workspace, with a reference model written in the test that keeps plain records, decides by scanning, and decides the last-administrator rule by copying its state, applying the change and comparing before and after. After every operation the outcome (success, refusal, last-administrator refusal) equalled the model's, and then `Access` of every session, `AccessByCredential` of every credential, `Recheck` of up to 150 held answers and `WithoutAdministrator` equalled the model; whenever the workspace had no administrator, an active administrator membership of a suspended user existed, which is the way back. The generator draws from sorted keys only, and three consecutive runs gave identical counts: 46,460 accesses allowed and 142,642 refused; 10,020 credential accesses allowed and 23,143 refused; 261,028 rechecks allowed and 494,136 refused; 199 acceptances; last-administrator refusals of 110 role changes, 117 suspensions, 113 removals and 11 departures; the workspace without administrator after 35 operations. The test fails when any of 50 categories, successes and refusals, is reached fewer than 5 times.

Limits of the reference model: its seat rule, inviter rule and session predicate have the same structure as the package's, so a misreading shared with the plan is invisible to it (the change review's HIGH finding was of this kind); one workspace; no clock moving backwards; no listings; a one-in-six sample of answers capped at 150; it compares success and failure and the last-administrator refusal, not the audit reason; limits other than attempts are set high and not exercised; it never opens a session with a reused verifier; the generator is steered (a growth phase of 40 operations, administrators preferred as actors) so that histories stay alive.

# Mutation check

181 single-line mutations of the five source files were applied one at a time by a script, and the package tests were run after each.

- First run, 170 mutations: 141 caught, 22 not caught, 7 did not compile.
- Of the 22: one showed dead code (an administrator count that was increased again for a target that becomes an administrator could never change the outcome), which was removed together with its mutation; one showed a test that did not test its rule (the idle-above-absolute case was refused by another check) and was corrected; six concerned checks between workspaces that a single workspace cannot exercise, for which the seeded two-workspace test was written. The 7 were rewritten so that they compile. Second run of 169: 155 caught, 14 not caught.
- After the change review the code changed (seat rule, operator block, session records deleted, counters, paging bound, bootstrap validity) and the list was brought up to it: mutations for removed code dropped, 15 added. Two runs in between showed two more untested rules, an inverted validity lying in the future and the paging bound with a cursor, for which cases were added.
- Final run: 165 of 181 caught, 16 not caught.

The 16 that are not caught each remove one of two checks that produce the same result, so no test through the package can tell them apart. The change review read the fourteen of the second run against the code and found thirteen justifications sound and one wrongly worded; the re-review read all sixteen of the final run, including the rows that are new or changed since (26 to 28 and 45), found them sound and asked for the first row to be limited to realistic settings, as it now is:

| Mutation | Why no test distinguishes it |
| --- | --- |
| 8, a validity without a start | Every caller also refuses a validity whose end is not in the future, and a zero start with a future end exceeds the maximum length for any limit below the saturation value of a duration and a clock that is not within about 292 years of the zero time |
| 19, a service user counted as an active human | A service user has no session and no membership, so every caller refuses earlier |
| 23 and 25, the absolute expiry ignored or inclusive in the session predicate; 104, the idle expiry not capped | The idle expiry is capped at the absolute expiry, so either check alone is enough; removing both would be caught |
| 26, 27, 28, a session of an unlinked identity, of a suspended user or of an older epoch current | Unlinking, suspension and every other epoch change delete the user's session records |
| 112, 115, 117, a recheck for a suspended user, a suspended membership or an unlinked identity | The epoch and the membership version change with each of these |
| 42, the seat rule applied to non-administrators | A workspace with members and no administrator membership at all cannot be reached |
| 45, a departed administrator counted as a seat | A departure removes the user's memberships |
| 70, the operator block not cleared by the operator's reinstatement | The block matters only while suspended, and every later suspension sets it again |
| 74, a departure not increasing the membership version | A departed user's epoch already fails every answer and the user can never return |
| 165, a human treated as a service user | A human has no owning workspace, so the workspace comparison refuses |

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Empty-secret verifier accepted | Caught |
| 2 | Zero verifier accepted | Caught |
| 3 | Verifier never compared | Caught |
| 4 | Identifier length unbounded | Caught |
| 5 | Identifier alphabet widened | Caught |
| 6 | Triple part length unbounded | Caught |
| 7 | Empty triple part accepted | Caught |
| 8 | Validity without start | Not caught |
| 9 | Inverted validity | Caught |
| 10 | Validity above its maximum | Caught |
| 11 | Validity end inclusive | Caught |
| 12 | Validity start ignored | Caught |
| 13 | Operator reference optional | Caught |
| 14 | Operator reason optional | Caught |
| 15 | Zero duration limits accepted | Caught |
| 16 | Zero count limits accepted | Caught |
| 17 | Invalid installation accepted | Caught |
| 18 | Unconstructed registry used | Caught |
| 19 | Service user counted as an active human | Not caught |
| 20 | Suspended user counted as active | Caught |
| 21 | Session current before its opening | Caught |
| 22 | Idle expiry ignored | Caught |
| 23 | Absolute expiry ignored | Not caught |
| 24 | Idle expiry inclusive | Caught |
| 25 | Absolute expiry inclusive | Not caught |
| 26 | Session of an unlinked identity current | Not caught |
| 27 | Session of a suspended user current | Not caught |
| 28 | Session of an older epoch current | Not caught |
| 29 | Removed membership present | Caught |
| 30 | Member counted as administrator | Caught |
| 31 | Suspended membership counted as administrator | Caught |
| 32 | Administrator of a suspended user counted | Caught |
| 33 | Actor need not be an administrator | Caught |
| 34 | Service users of any workspace counted | Caught |
| 35 | Retired service users counted | Caught |
| 36 | Strand rule applies without administrators | Caught |
| 37 | Target not subtracted from administrators | Caught |
| 38 | Last administrator rule disabled | Caught |
| 39 | Target's own remaining membership ignored | Caught |
| 40 | Other memberships ignored | Caught |
| 41 | Remaining service users ignored | Caught |
| 42 | Seat rule for non-administrators | Not caught |
| 43 | Any other member counts as a seat | Caught |
| 44 | A suspended administrator membership counts as a seat | Caught |
| 45 | A departed administrator counts as a seat | Not caught |
| 46 | Report ignores service users | Caught |
| 47 | Report lists workspaces with administrators | Caught |
| 48 | Page repeats the cursor | Caught |
| 49 | Page size unbounded | Caught |
| 50 | Page size zero | Caught |
| 51 | Bootstrap armed twice | Caught |
| 52 | Bootstrap validity unchecked | Caught |
| 53 | Bootstrap armed with a validity already over | Caught |
| 54 | User created before the bootstrap | Caught |
| 55 | Created user takes an existing triple | Caught |
| 56 | User identifier reused | Caught |
| 57 | Expected epoch ignored for users | Caught |
| 58 | Suspension from any state | Caught |
| 59 | Operator cannot block a suspended service user | Caught |
| 60 | A suspended human suspended again | Caught |
| 61 | A blocked service user blocked again | Caught |
| 62 | Sessions kept after suspension | Caught |
| 63 | Sessions kept after departure | Caught |
| 64 | Sessions kept after linking | Caught |
| 65 | Sessions kept after unlinking | Caught |
| 66 | Dropped sessions keep their serial entries | Caught |
| 67 | Page bound overflows | Caught |
| 68 | Operator block not set | Caught |
| 69 | Reinstatement from any state | Caught |
| 70 | Operator block not lifted by the operator | Not caught |
| 71 | Departure repeated | Caught |
| 72 | Departure ignores the strand rule | Caught |
| 73 | Departure ignores the seat rule | Caught |
| 74 | Departure keeps the membership version | Not caught |
| 75 | Departure keeps memberships | Caught |
| 76 | Departure keeps identities linked | Caught |
| 77 | Departed user stays in its state | Caught |
| 78 | Bootstrap consumed twice | Caught |
| 79 | Locked bootstrap accepted | Caught |
| 80 | Bootstrap without arming | Caught |
| 81 | Bootstrap validity ignored | Caught |
| 82 | Bootstrap verifier ignored | Caught |
| 83 | Bootstrap locks one attempt late | Caught |
| 84 | Bootstrap never locks | Caught |
| 85 | Bootstrap not consumed | Caught |
| 86 | First user is not an administrator | Caught |
| 87 | Unlinked identity resolves | Caught |
| 88 | Zero idle length | Caught |
| 89 | Idle above absolute | Caught |
| 90 | Idle above its maximum | Caught |
| 91 | Absolute above its maximum | Caught |
| 92 | Login through an unlinked identity | Caught |
| 93 | Login of a suspended user | Caught |
| 94 | Session verifier reused | Caught |
| 95 | Expired session records kept | Caught |
| 96 | Dead sessions counted against the limit | Caught |
| 97 | Sessions per user unbounded | Caught |
| 98 | Ended session stays usable | Caught |
| 99 | Ended session keeps its serial entry | Caught |
| 100 | Ended session keeps its index entry | Caught |
| 101 | Access with a suspended membership | Caught |
| 102 | Idle expiry never renewed | Caught |
| 103 | Idle expiry moves backwards | Caught |
| 104 | Idle expiry beyond the absolute expiry | Not caught |
| 105 | Revoked credential answers | Caught |
| 106 | Credential validity ignored | Caught |
| 107 | Credential verifier ignored | Caught |
| 108 | Suspended service user answers | Caught |
| 109 | Answer of another instance accepted | Caught |
| 110 | Recheck before the answer's start | Caught |
| 111 | Recheck past the answer's end | Caught |
| 112 | Recheck for a suspended user | Not caught |
| 113 | Recheck ignores the epoch | Caught |
| 114 | Recheck of a revoked credential | Caught |
| 115 | Recheck with a suspended membership | Not caught |
| 116 | Recheck ignores the membership version | Caught |
| 117 | Recheck through an unlinked identity | Not caught |
| 118 | Recheck after the session ended | Caught |
| 119 | Linking with an old session | Caught |
| 120 | Reauthentication window exclusive | Caught |
| 121 | Linking without proof | Caught |
| 122 | Triple linked twice | Caught |
| 123 | Linking keeps the epoch | Caught |
| 124 | Unlinking another user's identity | Caught |
| 125 | Last identity unlinked | Caught |
| 126 | Older identity unlinked through a later one | Caught |
| 127 | Unlinking keeps the epoch | Caught |
| 128 | Invitations of a changed inviter counted | Caught |
| 129 | Closed invitations counted | Caught |
| 130 | Invitations not started yet not counted | Caught |
| 131 | Inviter epoch ignored | Caught |
| 132 | Inviter membership version ignored | Caught |
| 133 | Invitation with any role | Caught |
| 134 | Invitation validity unchecked | Caught |
| 135 | Pending invitations unbounded | Caught |
| 136 | Invitations of other workspaces counted | Caught |
| 137 | Invitation identifier reused | Caught |
| 138 | Closed invitation revoked | Caught |
| 139 | Invitation accepted in another workspace | Caught |
| 140 | Invitation accepted twice | Caught |
| 141 | Invitation validity ignored at acceptance | Caught |
| 142 | Invitation accepted by another user | Caught |
| 143 | Inviter authority ignored at acceptance | Caught |
| 144 | Existing member accepts an invitation | Caught |
| 145 | Invitation verifier ignored | Caught |
| 146 | Invitation never locks | Caught |
| 147 | Invitation locks one attempt late | Caught |
| 148 | Invitation stays pending after acceptance | Caught |
| 149 | Re-created membership restarts its version | Caught |
| 150 | Invited member becomes administrator | Caught |
| 151 | Member changes its own membership | Caught |
| 152 | Member removes another member | Caught |
| 153 | Suspended member removes itself | Caught |
| 154 | Membership version ignored | Caught |
| 155 | Membership transition from any state | Caught |
| 156 | Membership operations ignore the last administrator | Caught |
| 157 | Membership version not increased | Caught |
| 158 | Change to the same role accepted | Caught |
| 159 | Suspending a suspended membership | Caught |
| 160 | Reinstating an active membership | Caught |
| 161 | Removed members listed | Caught |
| 162 | Owner with a suspended membership | Caught |
| 163 | Any owner accepted | Caught |
| 164 | Service identifier reused | Caught |
| 165 | Human treated as a service user | Not caught |
| 166 | Service user of another workspace reachable | Caught |
| 167 | Expected epoch ignored for service users | Caught |
| 168 | Service state change keeps the epoch | Caught |
| 169 | Administrator lifts the operator block | Caught |
| 170 | Suspending a suspended service user | Caught |
| 171 | Reinstating an active or retired service user | Caught |
| 172 | Retiring twice | Caught |
| 173 | Service users of other workspaces listed | Caught |
| 174 | Credential validity unchecked | Caught |
| 175 | Credential for a retired service user | Caught |
| 176 | Credential identifier reused | Caught |
| 177 | Revoked credentials counted | Caught |
| 178 | Credentials not started yet not counted | Caught |
| 179 | Credentials unbounded | Caught |
| 180 | Credential of another workspace revoked | Caught |
| 181 | Credential revoked twice | Caught |

# Deviations from the plan

- The registry uses one mutual-exclusion lock for every operation, because `Access` writes (it renews the idle expiry).
- A change to the role already held is refused with the common refusal and a state reason.
- Operator reference and reason are validated and not stored; the plan says kept for the audit record, which does not exist yet.
- The service-user listing also returns the epoch, which the expected-epoch calls need.
- A reused session verifier is a conflict, not the common refusal, and only while its record exists: after an end, a suspension or another deletion the verifier could be opened again.
- `Invite`, `AddCredential` and `ArmBootstrap` refuse a validity that is already over.
- A wrong bootstrap verifier outside the validity is refused without counting an attempt.
- `Resolve` returns the email attribute to the Boundary, and `OpenSession` and `Resolve` take the triple as one value.
- The second workspace needed by the cross-workspace tests is seeded directly, as described above.

# Provisional choices and undecided matters

The owner has not decided these; the implementation uses the labelled defaults of the plan and nothing depends on them yet: two roles without a ceiling between administrators; invitations always for a named user and dying with any change of the inviter's authority; a triple never linkable twice, also after departure, so a returning person is a new user; linking and unlinking by a recent session with a proof reference, each ending all sessions; rechecks that fail on an explicit session end and ignore idle time; refusal at the session, invitation and credential limits; one workspace per service user with a recorded owner; a bootstrap that locks for good; an installation-wide suspension that overrides the last-administrator rule; an operator block on service users; only the bootstrap workspace having members. Open decisions of HB-42 are unchanged: SSO/IdP product, allowed authentication methods, LDAP path, login identifier scope, MFA, self-signup, recovery and external collaborator policy.

# Limits

- The package trusts the Installation and Boundary holders: `OpenSession` is login as anyone whose triple is known. The layer that exposes them must be authorized and must have authenticated the person.
- A verifier is the credential at this boundary; the secrets behind verifiers, their entropy and their derivation are the boundary's.
- No recovery exists: a locked or expired bootstrap, a workspace without an administrator and a user without a usable identity need paths that are the owner's decision.
- A second workspace cannot get its first administrator.
- A stolen recent session together with a proof the boundary accepts can add an identity.
- A suspension here does not deny a decision in `modules/auth` until HB-173.
- State is in memory and lost on restart; terminal invitation and credential records are kept, as are expired session records of a user who never logs in again.
- Creating a record under a used identifier is a conflict wherever the identifier is used, which reveals its existence to a workspace administrator; identifiers must be generated by the server layer.
- An administrator can retire a service account the operator blocked and create another.
- No audit record is written; operator reference and reason are validated and not stored.

# Independent review

The design review of 2026-11-13 returned FAIL (narrow: 8 HIGH, 11 MEDIUM, 4 LOW) and the plan was rewritten. The re-review of the same day returned PASS with findings (2 HIGH, 8 MEDIUM and smaller items, all wording), which were folded in; the folded text was not reviewed a third time. Both were read-only and executed nothing.

The independent change review of 2026-11-13 read the five source files, the three test files, the plan and this record, executed nothing, and returned PASS with findings: no blocker, 1 HIGH, 4 MEDIUM, 9 LOW. The HIGH finding was real: after `SuspendMembership` of the second administrator and a security suspension of the first, the first could depart and leave a workspace that no operation could ever give an administrator; the code followed the plan's wording and the reference model had copied it, so neither the tests nor the mutation run could see it. It is corrected in code, plan, model and a literal test. The MEDIUM findings (the operator unable to block an already suspended service user; conflicts revealing identifiers; a property test that was not reproducible; a concurrency test that could pass without concurrency) and the LOW findings are handled as listed in the plan's change review resolution.

The narrow re-review of 2026-11-13 read the corrected source, the three test files, the plan and this record, executed nothing, and returned PASS with findings: no blocker, no high or medium finding, six LOW. It checked the seat rule against every mutating operation and found no sequence that strands a workspace and no user who becomes undeletable; it found the operator block, the deleted session records (serials are never reused, a missing serial fails a recheck), the paging bound and the bootstrap validity correct, and C1 to C13 resolved or documented (C14 it could not map). Its six findings were handled afterwards: a deleted session's verifier can be opened again, now stated and pinned by a case (R1); the wording of the first not-caught row (R2); two inaccurate sentences here (R3); the claim about counter guards narrowed to what is tested (R4); a case relabelled (R5); which review read which not-caught rows (R6). These last corrections are wording and one test case; they were verified by the checks above and not reviewed again.
