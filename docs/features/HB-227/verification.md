---
type: VerificationRecord
title: ERP product adapters and delivery gate verification
description: Records what was executed and reviewed for the first increment of work item W-25. It covers product and version profiles, adapters with per-command and per-limit evidence where simulation never certifies, revocation, certification and capability export only at sandbox or production evidence, the per-target automatic delivery gate tied to its approved settings, chunk planning within proven row limits, and a simulated ERP round trip through the ledger. The record gives results, the mutation check, provisional choices and what remains unverified; no real ERP sandbox exists.
status: draft
---

# Delivered behavior

[HB-227](https://linear.app/hybrid-billing/issue/HB-227) adds the Go package `modules/erpadapters` from main `b7a7b6a`. Its non-test sources use only the standard library; its tests import `modules/erpledger` and `packages/primitives/decimal`.

- **Profiles and adapters.**
  - Profiles are immutable. The only vendor and product pairs are `odoo`/`odoo`, `sap`/`s4hana`, `sap`/`ecc`, `sap`/`business_one` and `other`/`other`.
  - Each adapter is bound to one profile for good. It maps the platform commands to product methods over `api` or `file`; the `database` transport is refused.
- **Evidence.**
  - Each command and each of the two limits has its own level: declared, simulated, sandbox or production.
  - A system actor may record `simulated` only. `sandbox` and `production` need a reference and a human other than the author.
  - Levels may be skipped but never lowered.
  - Revocation is terminal. It keeps who raised the evidence and records who revoked it.
- **Certification and capabilities.**
  - Certification needs every lifecycle command, `get_balance` included, at `sandbox` or `production`.
  - `Capabilities` and `Admits` work only at those levels, so simulated and revoked commands never count.
- **Delivery gate.**
  - Diagnoses are kept per target and numbered.
  - An internal cost attachment, unproven recipient control or unproven attachment control blocks every automatic channel. A manual-only template that attaches an internal file still reports it.
  - Enabling needs the latest `allowed` diagnosis and a second human. An unsafe or changed diagnosis disables delivery.
- **Planning.** Rows are split into documents of at most `MaxRows`, only while `max_rows` is proven at `sandbox` or above and not revoked.

# Commands and results

All runs were made in the worktree `/private/tmp/hb227` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/erpadapters` | no output |
| `go vet ./modules/erpadapters` | pass |
| `go test -race -count=10 ./modules/erpadapters/` | pass (2.5 s) |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null` in the package | builds |
| `go test -race -count=1 ./...` | pass, 40 packages with tests |
| statement coverage of `modules/erpadapters` | 100.0% |
| property test of `Plan` against an independent chunk model | 300 random cases pass |
| mutation testing (`mutate227.py`, 400 mutants) | 369 killed, 25 did not build, 6 survived (all equivalent) |

The full run left 10 survivors. `TestMutationGuards` was added for two cases: certification blocking that lists only the commands below sandbox, and the 100,000-document plan bound at 999 rows per document. A targeted rerun then killed four of the survivors.

The 6 that remain are equivalent:

- `>=` in the last-chunk clamp;
- a production level value of 4 instead of 3 (only the order matters);
- -2 instead of -1 for a revoked or missing level, in two places;
- `<=` when tracking the lowest level;
- `Round(1)` on the clock.

The 25 mutants that did not build were rejected by the compiler or `go vet`.

# Simulated round trip

`roundtrip_test.go` runs a simulated ERP written in the test file. It does not certify anything and no sandbox exists.

**Normal path.** The simulated ERP drives `modules/erpledger` through these steps:

1. prepare;
2. submit draft;
3. draft observation;
4. read draft;
5. draft document, with an unrelated freight row;
6. link and reconcile, recovering the amounts: 300 attributed of a 315 document;
7. post, with the final document read back;
8. a correction revision that makes the link stale.

**Timeout path.** A second test covers a timeout:

- `Checked` turns the submission `unknown`;
- a resend under a new key is refused;
- polls recover the draft and the posting;
- a manual edit reported by webhook marks the submission diverged.

The ledger connection is registered with its capabilities directly. No adapter evidence is raised in these tests. Evidence raised elsewhere in the tests uses references with the prefix `test-fixture-`.

`TestLedgerAcceptsUnadmittedCapabilities` records the open gap. `Admits` rejects `post` and `correct` for a declared adapter, yet `erpledger.Connect` still accepts them. The test fails with a pointer to this residual as soon as the ledger starts refusing them.

# Acceptance

| Criterion | Evidence |
| --- | --- |
| AUD-DB-05: an internal cost file blocks automatic delivery regardless of posting | `TestInternalAttachmentBlocksAutomaticDelivery` |
| AUD-DB-11: no direct database path | `TestAdapterGates` (`direct_database_forbidden`) |
| HB-25-D: S/4HANA, ECC, Business One and Odoo kept apart | `TestProductsStaySeparate` |
| HB-25-V: a simulated pass is not certification | `TestSimulationNeverCertifies` |
| HB-25-I: the path from approved specification to correction | Simulated only (`TestSimulatedRoundTripThroughLedger`). NOT RUN against a real ERP. |
| HB-25-V scale: call limits, maximum rows and draft latency in a sandbox | NOT RUN |

# Reviews

- **Design review:** two rounds by an independent read-only reviewer.
  1. CONDITIONAL PASS, with 3 high, 5 medium and 5 low findings.
  2. PASS, with three low wording gaps that were folded in.
- **Change review:** two rounds by a separate independent read-only reviewer.
  1. PASS, with 5 low findings and one note, all applied.
  2. PASS.

# Provisional choices

These are mine, not the owner's, and need owner confirmation:

- the lists of vendors, products, deployments, levels, lifecycle commands and attachment kinds;
- system actors may record simulated evidence only;
- `Capabilities` accepts only `sandbox` and `production`;
- the plan splits by row count alone, up to 100,000 documents;
- an enablement is tied to its exact delivery settings.

The first pilot product is the owner's decision and is not made here.

# Not verified or not built

- Calls to a real Odoo or SAP system, credentials, and sandbox evidence.
- The HB-25-V measurements and call-limit enforcement.
- Binding a ledger connection to an adapter, the transport check, and the diagnosis target check (`modules/erpledger` still accepts self-declared capabilities).
- The qualified certifier role with `modules/auth`, and approved manual delivery routes (HB-27, HB-29).
- Persistence and screens I02, I04 and S02.
