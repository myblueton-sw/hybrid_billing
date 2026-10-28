---
type: VerificationReport
title: HB-34 held-source usage projection verification
description: Records synthetic Go-to-Python-to-engine usage disclosure tests, source protection evidence and remaining customer-access limits.
---

# Change and ownership

HB-34; machine mac_mini; accountable owner Seung Woo Park. This increment extends open PR31 after source `42794e9bf6ea323574e04fd4382b420f564b91a9`. Root wrote the code/tests and ran actual engine integration. Separate read-only `query_auth_review` performed design and implementation/test review and independently ran the Go race suite and Python projection tests. Source version is the commit containing this report.

The [design and integration boundary](../../../work/HB-34-customer-access/usage-projection.md) defines a usage-only disclosure profile and a held-source subprocess bridge. It preserves existing source lifecycle/SQL enforcement and adds Go current-grant revalidation before release. Supplier costs, prices, payer/account hierarchy, raw source JSON, raw source service/unit names and internal query totals are not passed across the projection bridge.

# Executed evidence

| Check | Observed result |
| --- | --- |
| Root `go test -race -count=1 ./...` | PASS; customeraccess 1.589s, customerusage 5.290s, no race diagnostic. |
| Independent `go test -race -count=1 ./...` | PASS; customeraccess 1.444s, customerusage 4.582s, no race diagnostic. Unconfigured actual-engine test skips in this standard run. |
| Root Python projection pytest | 19 passed in 0.40s. |
| Independent `python -m pytest -q tests/focus_import/test_customer_usage.py` | 19 passed in 0.06s using the engine-probes virtual environment. Direct script import is not pytest execution and is not counted. |
| Root projection/model/reader-revocation pytest | 56 passed in 16.94s. No destructive `stack` fixture or engine reset used. |
| `go vet ./...` | PASS. |
| Actual Go-to-Python-to-PostgreSQL/ClickHouse integration | PASS; first Go engine test 43.788s; owner-requested rerun after Docker interruption 22.106s. Durations cover multiple separate queries. |

The initial sandboxed engine attempt could not connect to local PostgreSQL; it was not a pass. An authorized retry and the owner-requested rerun passed. The reviewer inspected engine harness and root evidence but did not independently execute engine integration.

Independent design review initially requested full-selection consumer inputs, pre-read archive rejection and explicit delivery-window semantics. Those changes were accepted before implementation. Final independent actual-change review and scoped runtime QA: PASS, no blocking finding. This is a local independent review, not a GitHub approval or remote CI result.

# Scenario outcomes

- Strict six-field projection preserves `2.000000000000000000` and approved service/unit labels from the actual synthetic source; the independent fixture expectation is quantity two, unit hours and service Compute. Long exact decimals, numeric zero and missing quantity/unit cases pass unit tests without float conversion or invented monetary values.
- Source rows containing private payer/resource markers and supplier costs produce no corresponding fields, source metadata or raw labels in the safe response. Unmapped labels, foreign tenant, incompatible charge interval and duplicate row identity fail the whole projection, including invalid rows outside the page.
- Source preflight checks installation, company, tenant, actor/source binding, active state and hot slices. The hook executes before source row/archive reads. The actual source rejects a stale snapshot, revoked source actor and unmapped profile. A caller-selected foreign scope or monetary field is denied by Go.
- Paused-child tests prove a Go grant removal prevents result release. Oversized/malformed output, wrong revision, foreign/extra fields, early child exit, wrong acknowledgment, trailing bytes, failed exit, deadline and cancellation return no page. Successful responses carry explicit usage-only, no-currency, fixed snapshot/profile and bounded-page metadata.
- A real bridge subprocess holds the existing source coordinator lock after the ready envelope: an independent nonblocking lock attempt fails. The lock becomes available after release acknowledgment and normal exit. This is source-lock evidence, not a production revocation latency claim.

# Engine environment and preservation

Observed PostgreSQL `18.6 (Debian 18.6-1.pgdg13+2)` and ClickHouse `26.8.33.10`. Integration uses the retained HB-33-r4 synthetic namespace read-only: PostgreSQL database `hb33_reader_revocation_20261028_r4`, ClickHouse loopback port18129 and bucket `hb33-reader-revocation-20261028-r4`. Source actor is the active synthetic tenant-A2 reader; revoked A1 is a denial case. No schema, role, source data, publication, policy or lifecycle mutation is performed. Existing default and dedicated binding/control file hashes match before and after.

Run `tests/engine_probes/.venv/bin/python tests/customer_usage/verify.py` with the authorized retained synthetic environment available. It verifies namespace ownership, projects source rows, checks the held-lock protocol and runs the explicitly configured Go engine test. `tests/customer_usage/bridge.py` changes only in-process transport/path bindings and directly executes the production bridge. It does not spawn child processes or install a public endpoint. The ordinary Go suite skips `TestEngineGateway` without its explicit profile environment; a skip is not engine verification.

Non-sensitive local results and synthetic profile are under `.local/runs/HB-34-usage-projection/`, owned by the mac_mini operator until review/handoff. They are not promised as permanent remote artifacts. This Git report retains reproducible commands, independent expected values, engine versions and outcome scope; credentials and raw source responses are excluded.

# Limits and next work

This is one configured tenant and one fixed source snapshot/range per gateway. It returns a bounded page, never a complete customer statement or export. Only explicitly mapped source-reported usage is supported; customer selling/allocation amounts, estimated-versus-actual classification and contractual disclosure approval are not inferred from FOCUS costs. Source interval containment rejects incompatible data rather than prorating it.

Go authority remains process-local. External authentication, public portal/handler, durable/distributed policy administration, source-scale performance, actual commercial disclosure approval and protected export/download remain incomplete. I18/I19 external end-to-end certification remains NOT_RUN. The trusted direct Python command must not spawn descendants; arbitrary wrapper/process-tree execution is unsupported. Source changes serialize behind existing source locks; Go grants can change during reads, and updates before the final authorization check invalidate results. Revocation after that internal linearization point cannot recall released data.
