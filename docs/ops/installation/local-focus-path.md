---
type: Runbook
title: Run the synthetic local FOCUS path
description: Initializes and exercises the bounded HB-27 PostgreSQL, ClickHouse and S3 path with dynamic version contracts and explicit local-only identity and numeric limits.
status: draft
---

# Local FOCUS execution

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Owner: Seung Woo Park. Machine during implementation: `mac_mini`. This is a local CLI scaffold, not a running Go/React product or authenticated customer API. Its [plan](../../work/HB-27-local-path/brief.md) and [verification](../../reviews/HB-27-local-path/verification.md) distinguish actual execution from remaining work.

## Prerequisites

Use the existing [verification Compose stack](../../../deploy/local/verification/compose.yaml), its generated ignored `.env` and pinned [Python environment](../../../tests/engine_probes/pyproject.toml). Install the lock with `uv sync --project tests/engine_probes --frozen` if the environment is absent. Start the stack with `docker compose -f deploy/local/verification/compose.yaml up -d --wait`. Secrets stay in the ignored environment file; never print `docker compose config` or environment contents. Commands below run from the repository root.

The coordinator uses isolated PG/CH namespace `hb_local` and S3 bucket `hb-local-focus`; it does not modify the old `hb` probe tables or other projects. Initialization creates local nonowner runtime roles and verifies security configuration. Incompatible existing elevated roles, grants, settings or extra row policies cause failure. The expected scope row policies are repaired transactionally/in place; local policy activation is not reset by repeated init. Migration digest mismatch fails; there is no automatic destructive migration/reset command.

## Execute

```sh
tests/engine_probes/.venv/bin/python tools/local_focus.py init
tests/engine_probes/.venv/bin/python tools/local_focus.py versions
tests/engine_probes/.venv/bin/python tools/local_focus.py demo --version 1.3 --file tests/fixtures/focus/synthetic-1.3.csv
```

The demo executes original preservation → exact validation → CH insert and independent CH aggregate checks → explicit synthetic approver confirmation → PG publication CAS → scoped read → verified Parquet export. It selects the requested registry version, not a default version branch. Expected synthetic result: two rows, each of four cost measures totals `0.200000 USD`; coefficients are 200000 at scale 6. Consumed quantity totals 5 Hours and pricing quantity totals 2 Hours; unit prices are never totaled. JSON reports exact release/registry/export digests and one current scope revision. Repeating the identical demo returns the same sealed release/export without inserting rows again. The automatic approver in `demo` is a labelled synthetic scenario, not a production approval bypass.

The standalone commands keep steps independently observable:

```sh
tests/engine_probes/.venv/bin/python tools/local_focus.py ingest --actor loader-a --version 1.3 --revision 1 --file tests/fixtures/focus/synthetic-1.3.csv
tests/engine_probes/.venv/bin/python tools/local_focus.py confirm --actor approver-a --release <release-id> --manifest-digest <manifest-digest>
tests/engine_probes/.venv/bin/python tools/local_focus.py publish --actor approver-a --release <release-id>
tests/engine_probes/.venv/bin/python tools/local_focus.py query --actor reader-a
tests/engine_probes/.venv/bin/python tools/local_focus.py query --actor reader-b
tests/engine_probes/.venv/bin/python tools/local_focus.py export --actor approver-a --release <release-id>
```

Replace placeholders with ingest output. To switch the already published synthetic source to another supported fixture, use a strictly newer source revision, for example `demo --version 1.2 --revision 2 --file tests/fixtures/focus/synthetic-1.2.csv`. The old release stays immutable. Same source revision with changed bytes or contract conflicts. An earlier sealed release replay never moves a newer current pointer backwards.

## Dynamic supported versions

[Registry entries](../../../packages/contracts/focus/) declare exact version/revision, sources, field mappings/types, header/value predicates, enum rules, sum/nonadditive behavior and exact row-product checks. `versions` enumerates installed valid entries. The generic evaluator never executes Python/SQL from a contract. A compatible new version needs a reviewed entry and independent fixtures; new semantics outside the allowlisted evaluator need a reviewed evaluator change. There is no guessed version ordering, inference from headers, or automatic fallback.

Each release pins the entire selected entry/digest. Changed/missing current entry invalidates unsealed confirmation/publication; sealed reads/exports use pinned historical semantics. Tests install a test-only `9.9` entry to prove dynamic dispatch; **9.9 is not bundled product support**. The [official demo inventory](../../validation/reports/HB-27-focus-demo-inventory/summary.md) remains version 1.0 exploratory evidence, not supported ingestion or a relabelled provider export.

The local synthetic profile is Decimal(18,6), <=10,000 rows and 8 MiB, with bounded lexical/exponent syntax and positive source revision <2^63. Per aggregate absolute coefficient is below 10^22 and each request reads one current stable source scope. No production/provider precision, tolerance, currency conversion or tariff is chosen. Whole-file account/subaccount/billing/charge-period admission uses server-owned connection metadata before any storage, even for unknown versions. The profile's price/host applicability is a conservative synthetic Compute subset, not full FOCUS certification.

## Identity, quarantine and failure behavior

`loader-a`, `approver-a`, `reader-a` and equivalent `-b` identities are seeded synthetic local personas selected by the trusted local operator. Selection is not authentication; never expose this CLI or its credentials as an end-user service. A loader cannot confirm; a reader cannot mutate; PG forced RLS/composite references and separate CH scope-bound readers defend tenant boundaries. Runtime roles cannot alter authoritative policy/rights. No pricing, customer selling price, Claim or issuance is authorized by input publication.

Out-of-scope/undecodable admission or exceeded envelope limits reject before storing. Authorized but unusable input (including an unsupported version whose source envelope can be established) is preserved once in quarantine with digest/reason and a durable hold. It is never loaded or published. Exit 2 reports quarantine/rejection, exit 3 dependency/unknown-operation failure. Raw dependency errors and credentials are suppressed.

A pending/unknown storage operation survives process/PG session loss and blocks reuse; no lease expiry authorizes another writer. Quarantine holds also block confirmation/publication/query/export after upload completion. This increment has **no quarantine-resolution, re-request dispatch, automatic takeover, retry cleanup or destructive delete command**. Do not manually erase those records to force a successful result. Correct source/profile issues and design explicit authorized resolution/fencing before expanding the implementation. Policy withdrawal or changed capability before an effect stops progress. Local PG locking replaces neither a deployed policy server nor an independently durable restore-control journal.

## Test and retain

```sh
tests/engine_probes/.venv/bin/python -m pytest tests/local_focus -q
```

Integration tests reset the disposable `hb_local` PG/CH namespaces before and after each case; **running them removes the local demo state**. They do not touch the historical `hb` probes, other projects or existing S3 objects. Re-run `demo` afterwards to leave an inspectable published sample. Test resets are fixture setup/teardown, not a product recovery mechanism. S3 originals/exports from tests remain synthetic local objects without a cleanup claim.

Leave the four-service stack running for continuing work. `docker compose -f deploy/local/verification/compose.yaml down` stops it without deleting volumes. Go access API, React UI, actual provider profiles/collectors, user authentication, complete customer attribution, control-journal recovery, quarantine resolution, safe deletion, actual rebuild import, HA and capacity acceptance remain future work.
