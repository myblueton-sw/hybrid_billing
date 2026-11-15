---
type: VerificationRecord
title: HB-247 migration import verification
description: Actual checks run for W-37 increment 2 (modules/migration), plus review outcomes and residuals.
status: final
---

# Scope

[HB-247](https://linear.app/hybrid-billing/issue/HB-247) adds `modules/migration`, following [plan.md](plan.md):
- **Sources.** Sources, importer identities, workspace allow-lists and immutable series mappings are dual-controlled proposals.
- **Batches.** A batch is bound to the exported item count and digest. It is sealed by its importer, checked for evidence presence, and accepted under dual control against an effect digest.
- **History.** Accepted documents and parsed public keys or certificates become read-only history. Secret key material is refused.
- **Re-import.** Re-import is idempotent by source identity. Changed content and natural-key collisions are quarantined with both digests and never overwrite history.
- **Fences.** Accepted batches produce tenant-keyed high-water marks, printed-number fences and delivery fences, all exported through reads.

Base: main `e360c86`. Machine: mac_mini.

# Checks run (2026-11-15, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/migration/` | clean |
| `go test -race -count=10 ./modules/migration/` | pass |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/migration/` | pass |
| `go test -race -count=1 ./...` | every package passes |
| Statement coverage | 100.0% |
| Property test | 150 seeds against an independent model. It covers two sources of one tenant, independently varied printed numbers, rejects, quarantine with deduplication, and exact equality of the exported delivery and printed fence sets and the high-water mark |

The known-answer tests were cross-checked against an independent Python encoder using the canonical length-prefixed encoding. The values checked were the batch digest, the effect digest, and the `src-`, `ext-` and `dsp-` fence keys.

# Mutation testing

| Run | Mutants | Killed | Did not build | Survived |
| --- | --- | --- | --- | --- |
| First | 513 | 404 | 63 | 46 |
| Final code | 513 | 444 | 63 | 6 |

Between the runs, tests and simplifications closed the first run's survivors. In the final run, 10 mutants survived. New tests in `survivor_test.go` killed 4 of them, each confirmed by applying the mutant by hand:
- the certificate size cap tested at exactly `maxDER`;
- a re-import at the high-water mark adding no series;
- the series counter on an existing series;
- the value returned with an invalid-series error.

The 6 remaining survivors are equivalent:
- `Round(0)` → `Round(1)`, twice;
- `>` → `>=` when comparing with the plan's own maximum, which assigns the same value;
- `<` → `<=` in two sorts over unique keys;
- `&&` → `||` in accept retirement. A source has at most one unfinished batch, so every pending accept proposal in a scope names that batch.

# Reviews

- **Design review** (independent reviewer, read-only):
  - round 1 FAIL, H1–H3, M1–M7 and L1–L4;
  - round 2 FAIL, N1–N3 and lows;
  - round 3 PASS, with R3-1 and R3-2 folded in.
- **Change review** (a separate independent reviewer, read-only):
  - round 1 FAIL, M1 and M2 plus lows;
  - round 2 FAIL, a self-retirement regression from the L3 fix;
  - round 3 PASS.

  Details are in plan.md.

# Residuals and exposure blockers

- Persistence.
- Source adapters and exporters that implement the canonical encoding.
- Evidence artifact transfer into `modules/evidence`.
- Binding workspaces to tenants in `modules/auth`. Until then:
  - workspace claims are first come, first served;
  - proposal and fence reads reveal whether a tenant exists.
- Feeding the fences into `modules/recovery`, issuance, delivery, webhooks and erpadapters.
- Whether migration should pause while recovery is locked.
- Screens.
- HB-37-V system-level cases are NOT_RUN.
- Owner decisions not confirmed by the owner:
  - the document and key kinds;
  - the bounds.
