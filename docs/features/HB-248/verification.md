---
type: VerificationRecord
title: HB-248 simulation core verification
description: Actual checks run for W-38 increment 1 (modules/simulation), plus review outcomes and residuals.
status: stable
---

# Scope

[HB-248](https://linear.app/hybrid-billing/issue/HB-248) adds `modules/simulation`, following [plan.md](plan.md):
- **Manifests.** Dataset manifests record origin, digest, source, licence notice and lineage (SIM01). Scenario manifests fix seed, virtual clock, versions, datasets and the event list. Public or converted data can never be used in a billing scenario.
- **Expectations.** Expected values are fixed under dual control before the first run of the same engine inputs (the fingerprint rule). They can never equal a recorded run output.
- **Runs.** Only registered harnesses record runs, and never one set up by the expectation's author or approver. Each run is compared check by check (SIM03), and a reproducibility key covers the output and results digests (SIM02). The current verdict is the worst across rule sets at the highest engine version, and runs of a revoked harness fail closed.
- **DB-09 matrix.** Each commitment event is linked to a tested scenario, an unsupported source or an open decision, with history and a permanent failure count. `unsupported` and `undecided` are never counted as passed.
- **Integration proof (SIM10).** Proof is kept apart from the demo result and reads only as `proven_unclassified` until environments are classified.

Base: main `a67549f`. Machine: mac_mini.

# Checks run (2026-11-15, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/simulation/` | clean |
| `go test -race -count=10 ./modules/simulation/` | pass, including concurrent `Record`, `Propose`, `Approve`, `RevokeHarness`, `Matrix` and `Summary` |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/simulation/` | pass |
| `go test -race -count=1 ./...` | every package passes |
| Statement coverage | 100.0% |
| Property test | 150 seeds against a separate model of verdicts, `MarkedBy`, the run that decides `Current`, cell status and failure count, with copied outputs, revocation and coverage changes. In one measured run, the model reached `pass` 195 times and `cell-passed` 23 times, and every other verdict and cell status hundreds of times |

The fingerprint and results-digest known-answer values were cross-checked with an independent Python encoder in the root session; that encoder is not in the repository. The change reviewer could not check this cross-check, because it reads only the repository.

# Mutation testing

| Run | Mutants | Killed | Did not build | Survived |
| --- | --- | --- | --- | --- |
| First | 746 | 659 | 49 | 38 |
| Final code | 728 | 674 | 46 | 8 |

Between the runs:
- Boundary tests were added for the validators, the pending and done caps, history rollover, and result caps and order.
- Redundant conditions were removed:
  - the empty-decimal check;
  - the status upper bound;
  - the month-tens digit checks;
  - the empty-result check;
  - the `len(s.runs)` check, which is implied by the fingerprint set.
- The `sort` of fingerprint digests and the other targeted mutants were also checked by hand.

The 8 final survivors are equivalent:
- `<` → `<=` twice in `semverLess`, after an inequality guard;
- a seed of 21 digits, which `ParseUint` refuses anyway;
- `digits(text[:4], 5)` on a 4-byte slice;
- `Round(0)` → `Round(1)`, twice;
- `eventIndex` returning `-2` instead of `-1`;
- a pass rank of 5 instead of 4, which keeps the same order.

# Reviews

- **Design review** (independent reviewer, read-only):
  - round 1 FAIL: H1 engine output as the answer, H2 human recording, and M1–M6;
  - round 2 FAIL: N1 the correction path, N2 `self_verification`, N3 masking by rule set;
  - round 3 FAIL: R1 the fingerprint fields, R2 the run cap;
  - round 4 PASS, with lows folded in.
- **Change review** (a separate independent reviewer, read-only):
  - round 1 FAIL: M1 and M2 were test gaps, plus L1–L7;
  - round 2 PASS, with N1 (digest order in the fingerprint) fixed afterwards.

  Details are in plan.md.

# Residuals and exposure blockers

- QA-10 demo isolation (increment 2):
  - credentials;
  - egress;
  - environment registration and classification;
  - reset (SIM08).
- Generators at scale (W-39), and the replay, mock ERP, mailbox, on-premises and permission-failure drivers.
- SIM04–SIM07 and SIM09, which need a harness wired to the engine.
- Voiding runs, correcting expectations for inputs that have already run, and rebuilding keys without revoked runs.
- Engine runs outside the register, and manifests changed only to escape the fingerprint.
- `origin: synthetic` is declared, not proven.
- Transferring notice texts, dataset bytes and calculation sheets into `modules/evidence`.
- Persistence.
- Screens.
- HB-38-V system-level cases and AUD-DB-09 / AUD-QA-10 are NOT_RUN.
- Owner decisions not confirmed by the owner:
  - the public sample adoption commit;
  - who approves synthetic expectations;
  - the performance hardware;
  - the closed event set and bounds.
