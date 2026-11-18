---
type: VerificationRecord
title: HB-253 third-party component register verification
description: Actual checks run for W-40 increment 2 (modules/thirdparty), plus review outcomes and residuals.
status: stable
---

# Scope

[HB-253](https://linear.app/hybrid-billing/issue/HB-253) adds `modules/thirdparty`, following [plan.md](plan.md):
- **Records.** Each third-party component record is pinned to an exact version, source commit and shipped artefact, with licence, notice, SBOM, patch, test and telemetry-evidence digests. Name/version and name/commit pairs are unique, with explicit `Supersedes` corrections.
- **Adoption.** Adoption requires passing tests, offline operation for shipped material, and no required telemetry. The closed licence table is applied to the component and, through the "ships in" rule, to its run-time dependencies. Rejection and revocation are dual-controlled.
- **Releases.** Release checks report every problem: unknown, not adopted, revoked, missing dependency, not shipped, duplicate name. A notice bundle digest is produced only when the release is ready, and `RecordRelease` binds it to the product commit.

The ticket text named `modules/components`. That package already exists (the data-centre components register), so this work lives in `modules/thirdparty`. `git status` and `git diff origin/main` show no change under `modules/components`.

Base: main `cefde87`. Machine: mac_mini.

# Checks run (2026-11-16, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/thirdparty/` | clean |
| `go test -race -count=10 ./modules/thirdparty/` | pass, including concurrent proposals and release checks |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/thirdparty/` | pass |
| `go test -race -count=1 ./...` | every package passes |
| Statement coverage | 100.0% |
| Property test | 150 seeds against an independent model, every listed outcome reached |

The property test's independent model covers:
- the licence categories;
- adoption outcomes;
- revocation;
- the exact multiset of release problems;
- the notice bundle, recomputed with the test's own 8-byte big-endian encoder.

The release bundle known-answer value was cross-checked in this session with an independent Python encoder, and it matches: `2081a157f15280a546c1a293d614715847f1796b3c0b0305b7212c608bb6799d`.

# Mutation testing

| Run | Mutants | Killed | Did not build | Survived |
| --- | --- | --- | --- | --- |
| First (before the review lows were folded in) | 341 | 283 | 46 | 12 |
| Final code | 361 | 303 | 49 | 9 |

After the final run, character tests killed two of the survivors, each checked by hand.

The remaining 7 are equivalent:
- `Round(0)` → `Round(1)`;
- the source loop starting at 1, because the first character is always the `h` of `https://`;
- `>` instead of `>=` for DEL, which `url.Parse` already refuses;
- two `boolText` changes that only alter the internal proposal identity consistently;
- a sequence tie, which is impossible;
- a sequence offset.

# Reviews

- **Design review** (independent reviewer, read-only):
  - round 1 FAIL: H1 no revocation, H2 transitive dependencies, and M1–M3;
  - round 2 FAIL: M1 corrections after revocation, M2 tool dependencies;
  - round 3 PASS, with lows folded in.
- **Change review** (a separate independent reviewer, read-only): PASS, with lows L1–L7 folded in.

  Details are in plan.md.

# Residuals and exposure blockers

- **Inventory completeness.** Producing SBOMs and scanning the packages actually shipped. A manifest that leaves out a shipped component, or a dependency that falsely declares itself `separate`, cannot be detected here.
- **Licences.** Licence expressions, licences outside the closed set, and legal review of the boundary table.
- **Not modelled.** Nested versions of one component, and the HB-40 comparison of the own core with OSS together with its gap evidence.
- **Not built here.** Persistence and screens.
- **Owner decisions** not confirmed by the owner:
  - the licence table;
  - the offline rule;
  - the telemetry rule;
  - the floating-label list.
