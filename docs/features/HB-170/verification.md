---
type: VerificationRecord
title: Relationship revisions verification
description: Records what was executed and reviewed for the first relationship model of work item W-02, with results, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-170](https://linear.app/hybrid-billing/issue/HB-170/w-02-relationship-revisions-with-effective-intervals-change-plans-and) adds the Go package `modules/relationships` from main `ea7dbba`: five separate relationship kinds as dated timelines, immutable numbered revisions per workspace, change plans with chunked upload, whole-candidate validation, a manifest digest, a preview and atomic activation, and queries pinned to a revision. Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review, which records its result). Machine: mac_mini. See the [plan](plan.md).

Nothing calls the package yet. It authorizes nothing, computes no amount and stores nothing durably. It does not complete W-02: the acceptance cases GH03 to GH06 remain NOT_RUN at system level.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All seven packages `ok`, including `modules/relationships` |

Literal table tests (`registry_test.go`, `rules_test.go`) cover: real and unreal dates, the year range and day order; invalid intervals; the identifier rule with its 128-byte bound; every limit required and the depth maximum; per-timeline rules refused at `AddChunk` (overlap in exclusive kinds including one day, same-target touching, overlapping and nested entries, an open entry before a later one) with touching different targets accepted; a mid-month move with `TargetOn` on the day before, the day and the day after, `Segments` with maximal pieces, gaps, clipping and refused open queries; a retroactive change with the preview, the pinned answers of the old revision and the revision records; viewing-group union, release, removal by an empty timeline and a later role change; two plans on one base, wrong digest, wrong expected revision, a stale plan, sealing on a stale base, stale plans counting as open, and revision numbers without gaps; the plan life cycle (chunk resume in another order, an index with other content, a member named in two chunks, a refused chunk replaced at the same index, missing and surplus chunks leaving the plan in draft, completion and sealing afterwards, a negative and an excessive chunk count, read-back in every state, paging, chunk, seal, activation, discard and change reads refused on discarded, rejected and activated plans as listed in the test, identifier reuse, a refused draft leaving nothing behind, a no-op plan); plan limits at and above their value; the digest (a golden value computed separately with a short script from the written encoding, equality under another chunking and order, inequality for another workspace, another base number, another base content under the same number, another day, end, target, kind, count and an added empty timeline, and for a boundary shift between member and target); input and output copies; queries that do not suit a kind, revision 0 and a missing revision; workspace separation; hierarchy cycles (direct, indirect, excluding the start member, on some days, on one day), depth at three and four ancestors and on some days only, a ring of five reported as depth, a violation naming a member the plan did not touch, a parent with the maximum number of entries; payer chains with touching and overlapping days, mutual payers and a later plan; role disjointness in the three kinds regardless of days and across plans, and free use across kinds; the order and bound of violations and the truncation flag above and exactly at the bound; invalid identifiers, days and cursors on every query.

Concurrency under the race detector: six readers call `Active` and read both pools at the returned revision while 60 plans move 40 members between two pools; every read matched exactly that revision, readers observed at least two distinct revisions, and at least one read was overtaken by an activation between obtaining the revision number and finishing the read (the test keeps publishing, up to a bound, until that has happened). Two goroutines activating two sealed plans on one base: exactly one succeeded.

Property test (`property_test.go`, fixed seed): 60 registries with 25 generated plans each. A reference model written in the test keeps the plain entry lists and answers by scanning every entry on each of 13 days (the last standing for all later days); it owns validity (payer chain, cycle, depth and role disjointness, each decided per day). `Seal` accepted exactly the candidates the model accepts and returned exactly the model's violations: 963 accepted; rejected candidates contained 203 cycle, 21 depth, 327 payer-chain and 247 role violations and 82 no-op plans; the test fails if a category is reached fewer than 15 times. After every fifth generated plan every kind, identifier and day of the universe was compared, and after the others and again for every earlier revision at the end a random twentieth: `TargetOn`, `TargetsOn`, `MembersOn`, `Segments` (coverage and maximality), `Chain` and `Union` equalled the model, and the preview's changed members, differing days, affected targets and earliest day equalled the model's difference.

Limits of the reference model: it generates per-timeline valid input only (the per-timeline rules rest on the literal tests); it uses a universe of 17 identifiers and 13 days; it does not exercise limits, several workspaces, the digest, chunk conflicts or stale plans; it does not check the target sets and entry counts of the preview; it shares the kind list, the constructors and the day-containment test with the implementation.

Scale: 100,000 timelines in 20 chunks were sealed and activated twice (a load, then a move of every member on the 15th); the test took between about 0.8 and 2.0 s without and about 7.6 s with the race detector on this machine while other workloads ran (load average between 14 and 24). No duration is asserted.

# Mutation check

135 single-line mutations of `types.go`, `validate.go`, `registry.go` and `plan.go` were applied one at a time by a script, and the package tests (without the scale test) were run after each. Each row names the fault the mutation introduces.

- First run, 126 mutations: 121 caught, 5 did not compile; the 5 were rewritten and were caught.
- The independent change review named faults that no test would have caught and that the list did not contain: a truncation flag set at exactly the bound, a base digest replaced by zeros, and several unchecked arguments, copies and state transitions. Tests were added first, then 9 mutations.
- Final run on the corrected code: 135 of 135 caught. One row was run separately because its search text had gone stale after a rename.

A complete result of this kind says that the listed faults are detected; it does not say that no other fault exists.

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Interval end inclusive | Caught |
| 2 | Interval start exclusive | Caught |
| 3 | Touching intervals overlap | Caught |
| 4 | Empty intersection accepted | Caught |
| 5 | Empty interval valid | Caught |
| 6 | Zero start valid | Caught |
| 7 | Unreal dates accepted | Caught |
| 8 | Year above 9999 accepted | Caught |
| 9 | Year 0 accepted | Caught |
| 10 | Identifier length unbounded | Caught |
| 11 | Identifier alphabet widened | Caught |
| 12 | Contract pool not exclusive | Caught |
| 13 | View group roles not disjoint | Caught |
| 14 | Contract pool roles not disjoint | Caught |
| 15 | Allocation pool roles not disjoint | Caught |
| 16 | Roles disjoint in every kind | Caught |
| 17 | Depth above package maximum | Caught |
| 18 | Zero limits accepted | Caught |
| 19 | Timelines per chunk unbounded | Caught |
| 20 | Member twice in a chunk | Caught |
| 21 | Unknown kind accepted | Caught |
| 22 | Invalid member accepted | Caught |
| 23 | Entries per timeline unbounded | Caught |
| 24 | Input entries aliased | Caught |
| 25 | Invalid target accepted | Caught |
| 26 | Member equal to target | Caught |
| 27 | Invalid interval accepted | Caught |
| 28 | Same target touching accepted | Caught |
| 29 | Same target overlapping accepted | Caught |
| 30 | Exclusive overlap of one day accepted | Caught |
| 31 | Exclusive overlap accepted | Caught |
| 32 | Chunk timelines not put in order | Caught |
| 33 | Entries of equal start not ordered by target | Caught |
| 34 | Empty timeline stored | Caught |
| 35 | Member not removed from target index | Caught |
| 36 | Member not added to target index | Caught |
| 37 | Empty target kept in index | Caught |
| 38 | Target index not sorted | Caught |
| 39 | Cycle only when the start member recurs | Caught |
| 40 | Depth bound off by one upward | Caught |
| 41 | Depth bound off by one downward | Caught |
| 42 | Walk does not narrow the interval | Caught |
| 43 | Walk follows non-overlapping parents | Caught |
| 44 | Role check removed | Caught |
| 45 | Payer chain check removed | Caught |
| 46 | Payer chain ignores days | Caught |
| 47 | Configured depth ignored | Caught |
| 48 | Violations not ordered by kind | Caught |
| 49 | Violations unbounded | Caught |
| 50 | Truncation flag not set | Caught |
| 51 | Difference intervals not merged | Caught |
| 52 | Unchanged days reported as different | Caught |
| 53 | Revision above the latest | Caught |
| 54 | Revision 0 missing for an untouched workspace | Caught |
| 55 | TargetOn on a view group | Caught |
| 56 | Segments on a view group | Caught |
| 57 | Segments with an open query | Caught |
| 58 | Segments with an invalid query | Caught |
| 59 | Gap before an entry not reported | Caught |
| 60 | Gap after the last entry not reported | Caught |
| 61 | Segment not clipped to the query | Caught |
| 62 | Members ignore the day | Caught |
| 63 | Members ignore the target | Caught |
| 64 | MembersOn with an unknown kind | Caught |
| 65 | TargetsOn with an unknown kind | Caught |
| 66 | Targets not sorted | Caught |
| 67 | Union not sorted | Caught |
| 68 | Union reads another kind | Caught |
| 69 | Union with an invalid group | Caught |
| 70 | Digest without version tag | Caught |
| 71 | Digest without workspace | Caught |
| 72 | Digest without base number | Caught |
| 73 | Digest without base digest | Caught |
| 74 | Strings without length prefix | Caught |
| 75 | Digest without timeline count | Caught |
| 76 | Digest without kind | Caught |
| 77 | Digest without member | Caught |
| 78 | Digest without entry count | Caught |
| 79 | Digest without target | Caught |
| 80 | Digest without start day | Caught |
| 81 | Digest without end day | Caught |
| 82 | Digest end marker not distinct | Caught |
| 83 | Plan identifier reused | Caught |
| 84 | Draft on a base that is not active | Caught |
| 85 | Draft on a revision of a workspace without any | Caught |
| 86 | Digest with a zero base digest | Caught |
| 87 | Truncation flag at exactly the bound | Caught |
| 88 | Chunk count above the limit at seal | Caught |
| 89 | Negative chunk count at seal | Caught |
| 90 | Invalid page cursor accepted | Caught |
| 91 | Returned target page aliases the plan | Caught |
| 92 | Returned violations alias the plan | Caught |
| 93 | Read-back violations alias the plan | Caught |
| 94 | Open plans unbounded | Caught |
| 95 | Closed plans keep their slot | Caught |
| 96 | Chunk added outside draft | Caught |
| 97 | Chunk index unbounded | Caught |
| 98 | Index reused with other content | Caught |
| 99 | Identical chunk refused or applied twice | Caught |
| 100 | Member named in two chunks | Caught |
| 101 | Seal outside draft | Caught |
| 102 | Seal with a missing chunk | Caught |
| 103 | Seal with a stored index above the count | Caught |
| 104 | Seal on a stale base | Caught |
| 105 | Timelines per plan unbounded | Caught |
| 106 | Entries per plan unbounded | Caught |
| 107 | Manifest not ordered before the digest | Caught |
| 108 | Timelines equal to the base counted as changed | Caught |
| 109 | No-op plan sealed | Caught |
| 110 | Violations do not reject | Caught |
| 111 | Rejected plan stays in draft | Caught |
| 112 | Violations not stored for read-back | Caught |
| 113 | Latest day reported as earliest | Caught |
| 114 | Target with changed days not reported | Caught |
| 115 | Target that gains a member not reported | Caught |
| 116 | Activation outside sealed | Caught |
| 117 | Activation ignores the digest | Caught |
| 118 | Activation ignores expected | Caught |
| 119 | Activation of a stale plan | Caught |
| 120 | Active pointer not switched | Caught |
| 121 | Revision without digest | Caught |
| 122 | Activated plan stays sealed | Caught |
| 123 | Plan does not record its revision | Caught |
| 124 | Discard in a terminal state | Caught |
| 125 | Discarded plan keeps its preview | Caught |
| 126 | Plan read-back aliases the summary | Caught |
| 127 | Preview aliases the summary | Caught |
| 128 | Page size unbounded | Caught |
| 129 | Page size zero | Caught |
| 130 | Changes readable in any state | Caught |
| 131 | Member page repeats the cursor | Caught |
| 132 | Target page repeats the cursor | Caught |
| 133 | Returned change aliases the plan | Caught |
| 134 | Plans looked up in a fixed workspace | Caught |
| 135 | Revisions looked up in a fixed workspace | Caught |

# Deviations from the plan as first written

- The hierarchy walk reports every rule a member breaks on some day, not only the first found; the plan was amended. This makes the result equal to a day-by-day reading, which the reference model checks.
- The per-kind summary counts the entries of the changed timelines, not of the whole revision; the plan was amended.
- No fuzz target was written; the plan did not ask for one.

# Provisional choices and undecided matters

The owner has not decided these; the implementation uses the labelled defaults of the plan and nothing depends on them yet: at most one parent, one payer, one contract pool and one allocation pool per member per day; payer chains of length one, which rejects the documented multi-level payment structures; no nested viewing groups; a role that is permanent across a kind's history; civil-day granularity; the workspace-wide switch, which invalidates every other open plan of the workspace on activation; rejection for good after a validation failure. Open decisions of HB-02 are unchanged: organization depth (a required parameter here), attribution priority of shared accounts, and contract handling when mid-month history is missing (a gap is answered as "none" and the package cannot tell it from missing history).

# Limits

- The package trusts its caller for the workspace and authorizes no write. Queries, previews and violations can name any identifier of the workspace, including another customer's.
- `Party.ParentID` in `modules/organizations` stays the only authority for the legal entity of a billing document; the two hierarchies are not reconciled or checked against each other.
- Identifiers are not checked for existence and have no type; de-duplication in `Union` is by identifier only.
- Validation runs over the whole candidate under the registry lock; a slow seal delays every workspace.
- State is in memory and lost on restart. Old revisions and the preview lists of activated plans are kept forever.
- Actor and reason of a change are not recorded.

# Independent review

The design review of 2026-11-13 returned FAIL (narrow: 4 HIGH, 12 MEDIUM, 8 LOW) and the plan was rewritten. The re-review of the same day returned PASS with findings (5 MEDIUM, 7 LOW), which were folded in; the folded text was not reviewed a third time. Both were read-only and executed nothing.

The independent change review of 2026-11-13 read the four source files, the three test files, the plan and this record, executed nothing, and returned PASS with findings: no blocker or high finding, 2 MEDIUM and 14 LOW. It reported that it could not construct a wrong query answer, an altered pinned answer, a candidate that passes sealing while breaking a rule, a published revision that differs from the sealed one, a lost update, a digest ambiguity, a cross-workspace read or write, a panic, race, deadlock or unbounded walk. The two MEDIUM findings were in tests: the atomicity test could pass without a read being overtaken by an activation, and the truncation boundary was untested while one test passed on a silently truncated list. Both were corrected, as were the LOW findings: a refused draft no longer creates a workspace record, an excessive chunk count is a limit error, dead code removed, a vacuous assertion and an order-dependent case fixed, the missing argument, copy and state cases added, the plan text aligned with what a sealed plan stores and with the reference model's sampling, and this record's claims reworded. Two naming remarks on receiver names were not acted on. The corrections were verified by the checks and the mutation rows above and were not reviewed again. The review did not read the rules files or the mutation script and verified no execution.
