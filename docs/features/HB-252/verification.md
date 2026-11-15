---
type: VerificationRecord
title: HB-252 license mode register verification
description: Actual checks run for W-40 increment 1 (modules/licensemode), plus review outcomes and residuals.
status: final
---

# Scope

[HB-252](https://linear.app/hybrid-billing/issue/HB-252) adds `modules/licensemode`, following [plan.md](plan.md), [usage-modes-requirements.md](../../licensing/usage-modes-requirements.md) and [product-policy.md](../../licensing/product-policy.md):
- **Modes.** Mode selection grants nothing.
- **Keys.** ed25519 verification keys are dual-controlled, with proof of possession, separate signing and acceptance windows, rotation and compromise.
- **Signed documents.** Decoding is strict, and the verification order is fixed, with distinct outcomes.
- **Evaluation.** One 336-hour evaluation per installation and per grant, started on authoritative time.
- **Time.** A high-water mark, clock anomalies, latched expiry and a dual-controlled reset.
- **Commercial entitlements.** Current entitlements and effective ends are computed on read from uncompromised, acceptance-ordered entitlements, with `superseded`, `bad_supersedes` and DB-14 `duplicate_right`.
- **Status.** A nonblocking status read. No call blocks, locks, sends or deletes anything.

Base: main `d7a88cf`. Machine: mac_mini.

# Checks run (2026-11-16, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/licensemode/` | clean |
| `go test -race -count=10 ./modules/licensemode/` | pass, including concurrent registration and status reads |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/licensemode/` | pass |
| `go test -race -count=1 ./...` | every package passes |
| Statement coverage | 100.0% |
| LM-01 to LM-11 | one named unit test each, all passing. System-level runs are NOT_RUN |
| Property test | 150 seeds, every listed outcome and state reached |

The property test uses an independent model of acceptance-ordered supersession, verification outcomes, compromise and clock anomalies. Agreement IDs and `Supersedes` targets are reused.

Known answers were cross-checked in this session against Python's `cryptography` 41.0.7 and an independent length-prefixed encoder:
- the claims body digest;
- the ed25519 public key from a fixed seed;
- the document signature;
- the key proof of possession.

All four match.

# Mutation testing

| Run | Mutants | Killed | Did not build | Survived |
| --- | --- | --- | --- | --- |
| First (before the change review fixes) | 448 | 378 | 14 | 56 |
| Final code | 449 | 419 | 14 | 16 |

After the final run, boundary tests with literal values killed 4 survivors, each checked by hand: three caps and the reset edge.

The 12 remaining are equivalent:
- **Rounding.** `Round(0)` → `Round(1)`, twice.
- **Values already refused by other checks.** A 13-digit time bound (any 13-digit time exceeds the maximum), the bad-path return value, a 3-digit scope count (any 3-digit count exceeds 32), and the `d.bad` grouping, which the re-encode comparison covers.
- **The perpetual sentinel**, `+2` instead of `+1`.
- **Effective end and tie-breaking.** `<=` at the effective end, because latching and later-issued supersession decide that case anyway, and `>=` in two equal-`IssuedAt` tie-breaks.
- **An unreachable adjacency.** A new entitlement that ends at an accepted one's start has already expired.
- **A compromised end bound** that the latch already covers.

# Reviews

- **Design review** (independent reviewer, read-only):
  - round 1 FAIL: H1 compromised-key backdating, H2 clock rollback revival, and M1–M5;
  - round 2 FAIL: compromise recovery, key windows, the high-water mark, renewal chains;
  - round 3 FAIL: `Supersedes` with compromise, reset revival;
  - round 4 FAIL: latch against effective end;
  - round 5 PASS.
- **Change review** (a separate independent reviewer, read-only):
  - round 1 FAIL: H1 same-agreement renewal chains cut to empty, letting a duplicate paid right through, plus M1 and L1–L5;
  - round 2 PASS.

  Details are in plan.md.

# Residuals and exposure blockers

- Restoring or replacing the register itself, by an operator who controls code and storage, cannot be detected.
- Small-order public keys are not rejected with the standard library alone.
- Key issuance, custody and signing; online verification and revocation lists; shortening `AcceptUntil`.
- Evaluation extensions, renewals registered before their `Start`, and exhausting the caps.
- Checking actual usage against each entitlement's contracted metric (AUD-DB-14 usage half). Marketplace sales activation remains a later scope.
- The OSS candidate register (increment 2), persistence, screens, and wiring `Status` into the UI with the contact action. `Status` must never switch features.
- LM-01 to LM-11 and AUD-DB-14 at system level are NOT_RUN. Legal suitability is not established here.
- Owner decisions not confirmed by the owner:
  - the evaluation terms;
  - the issuer;
  - the metric set;
  - the bounds;
  - ed25519.
