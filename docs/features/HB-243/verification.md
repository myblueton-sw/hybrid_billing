---
type: VerificationRecord
title: HB-243 installer register verification
description: Actual checks run for the first increment of W-36 (modules/installer), plus review outcomes and residuals.
status: stable
---

# Scope

[HB-243](https://linear.app/hybrid-billing/issue/HB-243) adds `modules/installer`, following [plan.md](plan.md). It provides:
- signed release manifests with pinned artifact digests and derived permissions;
- complete host observations reported by the probe;
- deterministic first-install previews that list every conflict and change nothing;
- single-use acceptances;
- checkpointed runs, with invalidation on probe, configuration and release changes and a host recheck on resume;
- one-time, nominee-bound administrator registration;
- diagnostics limited to approved destinations and ranges, with a closed table of safe guidance.

Base: main `2a091eb`. Machine: mac_mini.

# Checks run (2026-11-15, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l modules/installer` | no output |
| `go vet ./modules/installer/` | pass |
| `go test -race -count=10 ./modules/installer/` | pass (12.5 s) |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/installer/` | pass |
| `go test -race -count=1 ./...` | every package passes |
| Statement coverage | 100.0% |
| Source rules (`TestSourceRules`) | pass |
| Property test | 150 seeded sequences × 200 steps across two sites, against an independent model of conflicts, freshness, acceptances, run states, codes and probe binding |

Property test coverage:
- In one measured run of the property test, these operations succeeded:
  - 231 run starts;
  - 776 step completions;
  - 13 resumes;
  - 4 administrator registrations.
- Failed operations were checked against the model as well.
- The test checks three invariants: one active run per site; active runs only under the current probe and revision; no live code outside the `register_admin` step.

Known-answer tests:
- The release encoding and the bootstrap hash are pinned and were cross-checked against independent Python computations.
- The key challenge is pinned structurally.
- The preview hash is a regression pin computed by the implementation. A test also shows that every preview section changes the hash on its own.

# Mutation testing

| Run | Mutants | Killed | Did not build | Survived |
| --- | --- | --- | --- | --- |
| First | 786 | 725 | 26 | 35 |
| After fixes | 789 | 752 | 26 | 11 |

The 35 first-run survivors were closed with tests, or with simplifications where the code was redundant.

Of the 11 survivors in the second run:
- 5 were in the translated-prefix list. A redundant entry was removed, and the remaining three entries' mutants were re-run by hand and killed.
- The remaining 6 are equivalent:
  - `Round(0)` → `Round(1)`, three times;
  - a `[2]int` → `[3]int` pair array;
  - `<` → `<=` in two sort comparators over identical or unique keys.

# Reviews

- Design review (independent reviewer, read-only): round 1 FAIL, round 2 PASS. Its low items were folded into the plan.
- Change review (a separate independent reviewer, read-only): round 1 FAIL (M1–M3, L1–L7, H1–H2), round 2 PASS. The low items N1–N3 were applied.

# Residuals

- Upgrade and restore of an existing installation, and support bundles (increment 2).
- The host probe and runner. They must enforce connect-only-after-check and the approved address ranges.
- Granting the first administrator a role in `modules/auth`, and checking tenant authority upstream.
- Persistence, P01 copy review of the guidance steps, and scale measurement (HB-36-V).
- The supported platform combinations are untested. Releases declare portability targets only.
- Owner decisions not confirmed by the owner:
  - the closed artifact kinds and guidance table;
  - the minimum prefix lengths;
  - the numeric bounds;
  - first install only, with every existing resource treated as a conflict.
