---
type: VerificationReport
title: HB-34 protected usage export and download verification
description: Records current-authority export jobs, full-selection manifests, protected download revalidation and actual synthetic engine evidence.
---

# Scope and provenance

HB-34; machine mac_mini; accountable owner Seung Woo Park. This increment extends open PR31 after `9831a60c83817af26921df35e2f2d27df1e4013d`. Root implemented the changes and executed synthetic engine integration. Separate read-only `query_auth_review` performed design, actual-change review and independent Go/Python QA. Source version is the commit containing this report.

The [accepted design](../../../work/HB-34-customer-access/protected-export.md) implements `Exports.Create`, `Run`, `Download` and `PurgeExpired` around the existing held-source usage gateway. Create freezes an opaque authority checkpoint and recipient; Run is a separate explicit worker operation. Jobs and artifact bytes are process-local and bounded by supplied job/row/byte/TTL limits. There is no durable scheduler, customer HTTP endpoint or public download link.

# Implemented behavior

`usage.export` is distinct from `usage.read`. Original `NewService` continues to require read authority; the action-specific service is trusted backend configuration. Checkpoints bind exact store, recipient installation/subject/epoch, action, selection and scoped versions. Wrong/zero checkpoints, changed target, another store/action, epoch change, revoke/regrant and availability ABA deny. Equivalent relevant updates invalidate old jobs; unrelated subject updates preserve them.

Python export mode validates all selected source rows, checks the full source row count and rejects row-budget overflow. Its complete-selection marker and count are mandatory at the Go bridge. This is complete supported selected usage, not proof of upstream completeness, finalized financial amounts or a complete invoice. Source provenance/range/disclosure restrictions remain those of the preceding usage profile.

The JSON package contains safe usage rows and a fixed manifest with profile revision, source snapshot, scope/range, six fields, row count, payload SHA256 and explicit usage-only/no-currency basis. Exact decimals and missing values remain strings with measurement state. Formula-like labels remain inert JSON strings; no CSV/XLSX/PDF support is claimed. Download includes a fixed `usage.json` name, `application/json`, package digest and copied bytes.

Generation serialization and download's canonical package comparison occur while the source child holds existing source protections. Final Go checkpoint validation precedes the release handshake. Download re-queries the fixed source and never trusts cached artifact bytes alone. Failure, corruption, changed source, expiry or failed handshake yields no bytes. Invalid jobs discard payloads; expiry purges jobs. Wrong-recipient attempts cannot invalidate an owner's job. Concurrent/duplicate Run cannot create another artifact.

# Executed checks

| Check | Result |
| --- | --- |
| Root race suite after initial cleanup fix | `go test -race -count=1 ./...`: PASS; customeraccess 2.060s, customerusage 20.051s. |
| Independent final race suite, including same-time expiry cleanup refinement | PASS; customeraccess 1.597s, customerusage 19.965s; no race diagnostic. |
| Independent Python projection/export pytest | 21 passed in 0.34s. Includes 101 selected rows beyond a one-row page, count mismatch and row-ceiling rejection without truncation. |
| Root projection/model/reader-revocation pytest | 58 passed in 17.59s; no engine-reset fixture. |
| `go vet ./...` | PASS. |
| Actual engine runner | `tests/engine_probes/.venv/bin/python -u tests/customer_usage/verify.py`: PASS; explicitly configured Go engine test 42.817s. |

The actual engine test covers existing usage queries plus export Create→Run→Download, payload/manifest verification, hidden-marker exclusion and export-authority removal preventing another download. Observed engines: PostgreSQL `18.6 (Debian 18.6-1.pgdg13+2)` and ClickHouse `26.8.33.10`. It reuses the existing HB-33-r4 synthetic namespace read-only. Default/prior binding and control fingerprints remained unchanged. No database/schema/role/source/lifecycle mutation was performed. The reviewer inspected this harness but did not independently run it. Ordinary Go tests skip the engine case unless its explicit synthetic profile environment is provided.

The initial sandboxed Go build attempt failed on build-cache permissions; an authorized retry succeeded. This failure is not recorded as a test pass. Local non-sensitive integration results remain under `.local/runs/HB-34-usage-projection/`, owned by mac_mini until handoff/review; no permanent remote artifact retention is promised.

# Found defect and resolution

Independent static review found that expiry during download revalidation denied bytes but retained the cached payload until a later purge. `TestExpiryDuringPausedDownloadDiscardsBytes` reproduced FAIL before correction: `expired artifact bytes retained after final check`.

The correction purges expired jobs before and after Run/Download under the job mutex, sampling one time for cleanup and the associated validity check. The paused-download regression then passed in the race suite. The independent reviewer re-reviewed the actual correction and tests and returned PASS with no remaining blocking finding. Source security, recipient isolation, deterministic manifests, concurrency and complete-selection count checks were also reviewed.

# Remaining limits

Authentication selection/integration stays deferred. HB-34 remains open for public application integration, durable/distributed authority and jobs/artifacts, approved customer-facing formats and production-scale verification. The bounded package is source-reported usage only; selling/allocation prices and commercial disclosure approval are not inferred from supplier costs. It supports up to the configured limit within the existing 10,000-row exploratory source ceiling and byte budget; this does not certify a 100,000+ export service.

No saved recipient credential is used: Run and Download need a currently verified recipient context supplied by the trusted host. A process restart loses jobs rather than restoring stale grants. TTL cleanup clears references; it is not cryptographic memory erasure. The direct Python bridge must not spawn descendants. Source updates serialize behind existing source locks; permission changes before final authorization deny, while updates after internal release linearization cannot recall prior bytes. External I18/I19 end-to-end certification remains NOT_RUN.
