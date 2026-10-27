---
type: SecurityValidationReport
title: MSP customer isolation verification
description: Records a reproduced metadata leak, its scoped correction, actual database and lifecycle scenarios, and remaining external-access gaps.
---

# Verdict and boundary

HB-32 verifies the installed MSP collection/query core using synthetic customers. The corrected explicit-connection product path passes the executed core checks. This is not an external-customer isolation certification: direct internal SQL revocation fails, and external identity, customer-safe projection, portal, exports, cache and worker access paths are not implemented or verified.

The [scenario plan](../../../work/HB-32-msp-isolation/brief.md) defines I01–I21 before implementation. Root owns implementation and integration; independent DBA and security/QA reviewers inspected both design and actual changes. Accountable owner: Seung Woo Park; machine: mac_mini. Ticket: [HB-32](https://linear.app/hybrid-billing/issue/HB-32/design-and-verify-msp-customer-data-isolation-scenarios).

# Actual defect and correction

At base main `8153975c19fe27e32569e5416ef5f9298362ec08`, I09 returned the correct company-A four rows/USD32 but also two foreign coverage entries and two foreign batch graph references. No foreign private cost row was observed. I10 then rejected A's existing cursor with `query_snapshot_changed` after a B-only correction. Both were FAIL: metadata and activity are part of the isolation boundary.

`modules/focus_import/collection.py:selected_slices` now derives eligible tenant IDs from registered connection domains and server-owned assignment configuration before reading slices or constructing coverage, graphs and snapshot pins. It rejects unavailable ownership and mismatched delivery/window connection identity. It preserves authorized zero-usage periods instead of filtering by row count. No database schema, public identity API or deployment setting changed.

Legacy windows without a connection ID retain their existing shared-window behavior. Compatibility is tested; customer metadata isolation for those legacy windows is not newly certified. Customer-facing release must establish reviewed explicit source ownership and access boundaries.

# Observed results

Final pre-phase run `20261026T233842Z-pre`: 19 executed cases PASS (17 core, one storage-adjacent, one conditional engine metadata probe). Lifecycle run `20261026T234112Z-lifecycle`: two core PASS and one conditional direct-SQL FAIL. Across these final runs, 21 cases PASS and one FAIL; I18/I19 remain NOT_RUN. Eight additional pure selector regressions PASS.

| Scenarios | Actual observation |
| --- | --- |
| I01–I04 | A1=3 rows/USD25; A2=1/USD7; B=2/USD300; explicitly authorized company-A=4/USD32. Equal company labels, shared payer and matching account strings do not grant sibling/company membership; AWS/GCP remain provider-qualified. |
| I05–I08 | PG context spoof yields zero foreign rows; five forbidden privilege operations denied. CH unfiltered and UNION reads contain only A1; foreign predicate zero. Three locked-setting changes and a zero-row INSERT denied. `readonly=0` is allowed globally, but the same query observes mode0 and zero B rows; no blanket immutable-readonly claim. |
| I09–I10 | No foreign coverage or graph references; legitimate authorized zero-period coverage remains. A snapshot/cursor stays valid after B-only correction; B independently reports USD301. |
| I11–I13 | Ambiguous attribution rejected. B identifiers inside an A-valid delivery interval rejected with `connection_domain_authority` before retaining raw bytes. Reader cannot register delivery. Cross-company view definition, missing/extra membership and inactive audience rejected. |
| I14 | Four local personas interleaved/concurrent without observed contamination. Does not certify an absent HTTP session pool. |
| I15 | Archive/recall retains scope. Product access and old cursor denied after revocation. Restoring a pre-revocation backup does not restore product-query authority; unaffected A2/B remain exact. |
| I16 | Anonymous GET and LIST against the dedicated synthetic bucket return AccessDenied. No unrelated bucket tested. |
| I17a | Reader access to physical `system.parts` counts denied with ACCESS_DENIED. |
| I17b | FAIL: PG connection opened before revocation still reads slice row count3; CH credentials still read three A1 facts after product revocation and restore. CH is stateless HTTP; this is credential reuse, not a persistent CH session. No cross-customer rows observed in this probe. |
| I18–I19 | NOT_RUN: external authentication/forged user, MSP multi-customer employee grants, department human permissions, customer field projection, portal/search/cache/download/export and queued-worker paths. Internal `source_json` is not a customer-safe projection. |
| I20 | Own zero-usage month preserves two known-zero domain attestations and coverage, with A1 original three rows unchanged; no A-zero metadata in B response. |
| I21 | Synthetic clock2300 expires only dedicated test slices. B expiry does not block A2; owned expiry fails with `period_expired_unavailable`. No physical all-copy erasure claim. |

Postchecks additionally verify that a revoked company-view member denies the whole view, and that the unchanged default legacy installation still returns tenant row counts2/9/1112 and company union1121. These checks do not certify legacy customer metadata isolation.

# Environment and evidence

PostgreSQL18.6 and ClickHouse26.8.33.10 were exercised with real policies and SQL. Dedicated PG database `hb32_msp_isolation_20261027`, alternate CH port18124 and bucket `hb32-msp-isolation-20261027` isolate synthetic writes. Process-local adapters redirect transport endpoints and binding/control/bucket paths only. Preflight verified unused namespaces and reader roles; existing shared ingest/approve role attributes and memberships remain unchanged.

Default runtime journal sequence remains58; before/after status and binding hashes match. The default1123-row installation and unrelated hb_local are not reset. Dedicated synthetic state remains locally available for review, including revoked readers and expired slices. Do not blindly rerun lifecycle against this finished namespace.

The [evidence manifest](evidence-manifest.json) retains non-sensitive final observations, source/artifact digests and local evidence locations. Raw response logs, credentials, bindings, runtime data and backups are not committed. Local evidence is owned by the mac_mini operator and retained until ticket handoff/review; no permanent remote artifact retention is promised.

# Reproduction and execution

Use the pinned `tests/engine_probes/.venv` dependencies and an authorized configured local engine environment. Scripts have no external-customer inputs. On a fresh reviewed namespace: run `preflight.py`, then `reproduce.py` against the base source, apply the reviewed correction, run `check_selector.py`, `verify.py --phase pre`, `verify.py --phase lifecycle`, then `postcheck.py`, all under `tests/focus_import/msp_isolation/`. Creation requires absent dedicated namespaces; lifecycle changes only the owned synthetic namespace. These are local integration scripts, not automatically executed CI tests.

Earlier pre-phase runs are retained honestly: `233103` exposed an incorrect test assumption that global readonly was immutable; `233334` used separate stateless requests and did not prove a setting override in the data request; `233607` used a stale fixed coverage count after a legitimate own-zero period existed. The assertions were corrected and final `233842` rerun passed. These were test-oracle gaps, not additional production fixes. Original I09/I10 failures remain recorded separately.

# Independent review and next gate

DBA and security/QA final source/evidence reviews both PASS for the narrow fix and supported product-query boundary. Both preserve I17b FAIL and I18/I19 NOT_RUN. Reviewers inspected actual diff, matching source digests and retained results; they did not run DB mutations or claim independent runtime execution.

Before external customers can access an MSP installation, implement authenticated principal-to-company/tenant grants, customer-safe projection and all customer-facing access paths, then run adversarial end-to-end isolation/revocation tests. Keep internal DB/storage credentials behind the trusted service boundary; if direct SQL is a product requirement, its independent credential/session revocation must be designed and tested. This ticket does not claim those features complete or silently convert the failed probe into PASS.

# PR 28 resumption review, 2026-10-28

Reviewed PR [#28](https://github.com/myblueton-sw/hybrid_billing/pull/28) at head `f74ed80cd641e65a750e6bb9aceb22b9e7cd5f48` against base `8153975c19fe27e32569e5416ef5f9298362ec08`. Linear workspace, team and project matched the local binding; HB-32 remained In Progress with accountable owner Seung Woo Park and the mac_mini claim at review start. Root owns evidence verification and this report update; a separate read-only reviewer owns independent source review.

- PASS: the documented selector command reran all eight tests successfully. The independent reviewer separately ran the same eight tests successfully.
- PASS: applying the same suite to the base commit's selector function, loaded in memory through its AST without changing the working tree, produced three assertion failures and two errors out of eight tests. These cover foreign-window inclusion, foreign expiry/unsealed interference, missing ownership and mismatched connection identity. This is a pure selector comparison, not a new engine reproduction.
- PASS: SHA-256 verification matched all ten source entries and all seven retained artifact entries in the evidence manifest; artifact byte lengths also matched. The manifest still records 21 PASS and one FAIL. Hash agreement verifies retained evidence integrity, not a fresh runtime execution.
- PASS: `git diff --check 8153975..f74ed80` reported no whitespace errors. Independent source review found no new blocking defect in ownership filtering, metadata/snapshot consumers, connection immutability or revocation interactions.
- UNVERIFIED: no database lifecycle or integration scripts were rerun. The completed synthetic namespace contains revoked readers and expired slices and was preserved. Separate customer connections remain the demonstrated isolation profile; correction independence within a shared multi-customer connection is not established.

At observation, GitHub reported the PR open, unmerged and conflict-free, with zero check runs, zero commit statuses and zero submitted GitHub reviews. The combined status was pending with no statuses; conflict-free does not mean CI passed. The independent review above occurred locally and is not a submitted GitHub approval. I17b remains FAIL and I18/I19 remain NOT_RUN; legacy isolation and external-customer readiness are not certified. No merge or deployment was performed. This report-only follow-up does not change the reviewed source digests.
