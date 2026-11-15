---
type: VerificationRecord
title: HB-242 plugin registry verification
description: Actual checks run for the first increment of W-35 (modules/plugins), plus review outcomes and residuals.
status: final
---

# Scope

[HB-242](https://linear.app/hybrid-billing/issue/HB-242) adds `modules/plugins`, implementing the plan in [plan.md](plan.md):
- dual-controlled publishers, keys, versions, workspaces and installations, approved through numbered proposals;
- signed canonical manifests with declarative screens;
- the `runnable` rule, with register-wide marks;
- lifecycle isolation per workspace;
- tagged hashed tokens bound to installation revisions;
- endpoints with fingerprint tombstones;
- egress checks re-entered after the `Destinations` call.

Base: main `83da313`. Machine: mac_mini.

# Checks run (2026-11-15, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l modules/plugins` | no output |
| `go vet ./modules/plugins/` | pass |
| `go test -race -count=10 ./modules/plugins/` | pass (24.0 s) |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/plugins/` | pass |
| `go test -race -count=1 ./...` | every package passes |
| Statement coverage of `modules/plugins` | 100.0% |
| Source rules (`TestSourceRules`) | pass: no comments, files under 1,500 lines, only standard-library imports |
| Property test | 120 seeded sequences × 160 steps against an independent model of the transition table, `runnable` and token validity, covering three workspaces (one of them a verification workspace) |

Known-answer tests pin three encodings: the key challenge, the manifest encoding and the tagged token hash. The token hash was cross-checked against an independent Python computation.

# Mutation testing

- First run: 677 mutants, of which 599 were killed, 33 did not build and 45 survived. Tests were added for constant pins, form boundaries, a duplicate-free check of the capability order, full screen and scope lists, applied-history pruning and storage reinstalls.
- Final run, after the review fixes: 682 mutants, of which 643 were killed (one of them by a manual re-run after the version-drift test was split), 33 did not build and 6 are equivalent:
  - 2 are `<` → `<=` in sort comparators over unique keys;
  - 3 are `Round(0)` → `Round(1)`, which also strips the monotonic reading and keeps nanosecond precision;
  - 1 is a `[2]int` → `[3]int` pair array whose third element is unused.

# Reviews

- **Design review** (independent reviewer, read-only): FAIL in rounds 1 to 3, then PASS in round 4. The history is in the plan.
- **Change review** (a separate independent reviewer, read-only): PASS with M1 and L1–L8. The fixes were re-reviewed and returned PASS. The resolutions are in the plan.

# Residuals

- **Plugin runtime.** The process or container runtime that enforces timeouts, memory and isolation. HB-35-V is met only partially: one workspace's actions never change another, but resource isolation is not enforced here.
- **Not yet built:**
  - checking the content of deployment evidence;
  - expiring fingerprint tombstones;
  - raising the cap of 16 runtime identities;
  - indexing installations by version for automatic stops (L5);
  - the core-version upgrade path.
- **Plugin actor kind.** The later command adapter must use a distinct plugin actor kind that never satisfies a human or dual-control check (M8).
- **Other work items.** Persistence, the separate-origin host and the screens (P02, S05) are not part of this increment.
- **Owner decisions.** These items are not confirmed by the owner:
  - the closed sets of capabilities, components and migration rules;
  - the numeric bounds;
  - single-human tightening actions with second-human confirmation of marks;
  - accepting register-wide fingerprint ownership.
