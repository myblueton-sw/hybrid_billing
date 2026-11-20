---
type: VerificationRecord
title: Pinned FOCUS sample reload and local console read-back
description: Documents isolated sample provisioning, exact scope totals and authenticated browser verification.
status: draft
---

# Result

The owner-authorized reload succeeded. The local console on port 8830 reads the published data through the Go session gate. Overview and collected source-data screens loaded successfully after normal sign-in; the reported `service_unavailable` is resolved for these paths.

# Data and scope evidence

The source pack SHA-256 is `986a297e5476f4747a2921f690745309256ffc5d117cf487ff8a1c2d7286b9c2`, reconstructed with the existing pinned fetch/build functions. The official 1000-row file and the full 496305507-byte gzip passed their pinned digests. All 115 GCP rows were extracted with the pinned original-ordinal digest. No original dates, lexical amounts, currency or provenance values were changed. Derived monthly CSV files preserve each occurrence and provenance; they are not claimed byte-identical to the original combined CSV.

| Evidence | Observed |
| --- | --- |
| Combined source | 1123 rows: 1115 public plus 8 explicit VMware/Alibaba synthetic fixtures |
| Published monthly partitions | 2024-03:115; 2024-09:1007; 2024-10:1 |
| Authorized company | 1121 rows; north-a 1112 and north-b 9 |
| Separate foreign company | 2 rows, excluded from company-north |
| Authorized provenance | official-public 1000; official-public-subset 115; synthetic-contract-fixture 6 |
| Providers | AWS, Microsoft, Google Cloud, Oracle, VMware, Alibaba Cloud |
| USD billed/effective | 27.37875272899 / 18.72651418586 across 1119 rows |
| CNY billed/effective | 11 / 11 across 2 rows; no currency conversion |
| FOCUS version | 1.0, exploratory query-only; financial_authority=false |

Initialization used the six-provider configuration and an empty lifecycle journal. Monthly partitions went through describe, register, collect, confirm and publish using the documented local scenario roles. No legacy snapshot was imported first. Source finalization/completeness is an assertion about this sample scenario, not a provider feed or financial approval. Synthetic received-time retention uses hot 30/total 365/grace 0/backup 30 days; no archive/deletion/retirement operation was run.

# Runtime and verification

- Three healthy storage services in the new, explicitly named `hb336-preview` project: PostgreSQL 18.6, ClickHouse 26.8.33.10, SeaweedFS 4.48, using repository-pinned image digests. Loopback ports 25432/28123/28333. Other project containers and volumes were untouched.
- Locked `uv sync --frozen --project tests/engine_probes` installed the existing Python dependencies. Runtime data/tool code matches the ticket's main base d7b6a3d.
- Synthetic runtime credentials are in the ignored, mode 600 local configuration. Downloaded data, derived partitions, control journal, bindings and manifests remain ignored under the authorized root `.local/`; no credentials or account/resource rows are committed.
- Readers 8806/8810 bind `reader-company-north` and require the existing API-generated shared-secret file. The current Vite process uses `HB_GATED=1`; all collected requests pass through the Go session/permission gate. No second workspace was granted data access.
- PASS: exact company and individual scope counts through the real lifecycle query.
- PASS: authenticated HTTP 200 from overview, receipts, connections and collection-status; collection-status also reports 1121 rows.
- PASS: unauthenticated collected request 401; direct reader request without gate secret 403.
- PASS: actual Korean browser sign-in → overview → source-data, 1121 rows/six providers, canonical font load, no page errors and no `service_unavailable`. Chromium 153.0.8010.12 on macOS, 1440×1100. Screenshots and runner are local `/private/tmp/hyb336-qa`; non-sensitive results are committed in evidence.json.
- No destructive integration tests were run against the loaded environment. This operational task changes no application code. It does not establish current live-provider collection, financial acceptance, all screen integrations, native-language acceptance or Windows/Linux operation.

# Independent final review

Separate read-only reviewer PASS: independently checked source SHA-256, 1123 source rows, exact occurrence equality across monthly partitions, public/synthetic provenance and scoped counts. Reviewed the QA runner/results and both rendered screenshots. This was artifact and source-data review, not a second provisioning or browser run; runtime isolation and health remain executor-recorded evidence. No blockers.

# Reuse and handoff

The user-facing URL remains `http://127.0.0.1:8830/?service=api&lang=ko#/home?lang=ko`. The local sample autofill control from HYB-335 is active. API, both readers and Vite remain running for inspection.

The data survives storage-container stops in the named project volumes. The trusted configuration, `.local/focus-import-bindings.json` and `.local/focus-lifecycle` journal must stay with this installation. Do not rerun initialization or delete control state to bypass a failure. To restart the existing storage only, run from `/Users/sw/project/hybrid_billing`:

```sh
docker compose -p hb336-preview -f deploy/local/verification/compose.yaml -f deploy/local/verification/overview-ports.yaml up -d --wait postgres clickhouse seaweedfs
```

Run `tests/engine_probes/.venv/bin/python tools/focus_overview.py --local-operator --actor reader-company-north --port 8806 --ui-port 8830 --pg-port 25432 --ch-port 28123 --s3-port 28333 --gate-secret-file /private/tmp/hyb330-api/reader-secret` and the same command with `--port 8810` for the second reader. Use the active Go API's secret-file path if its data directory changes. Restarting the synthetic API resets its accounts; local autofill configuration must match the newly initialized sample account. Keep Vite's gated proxy enabled. Runtime process paths are current local handoff facts, not portable deployment instructions.

Ticket HYB-336; owner Seung Woo Park; machine mac_mini. Reviewed Git evidence is handed off through a PR; no merge or deployment performed. Runtime remains available while the active work claim is released at handoff.
