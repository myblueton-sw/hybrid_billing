---
type: Runbook
title: Collect six-provider samples and query authorized company views
description: Runs observed public AWS Azure GCP OCI samples and labelled synthetic VMware Alibaba fixtures through storage and explicitly authorized individual and company queries.
status: draft
---

# Six-provider sample workflow

Tracking: HB-27. Owner: Seung Woo Park. Machine during implementation: mac_mini. [Design](../../work/HB-27-six-provider/brief.md), [base local runbook](tenant-focus-data.md). This is a local trusted-operator sample workflow. It is not live provider collection, production authentication or financial acceptance.

## Source classes

| Requested provider | Stored source label | Sample evidence |
| --- | --- | --- |
| AWS | AWS | 942 original official public rows |
| Azure | Microsoft | 51 original official public rows |
| GCP | Google Cloud | All115 selected rows from the pinned official full dataset |
| OCI | Oracle | 7 original official public rows |
| VMware | VMware | 4 deliberately synthetic contract fixtures |
| Alibaba Cloud | Alibaba Cloud | 4 deliberately synthetic contract fixtures |

The original official1000-row file is freshly downloadable. The GCP subset is extracted by scanning the exact full gzip (5488359rows); SHA-256 and original source ordinals pin the selection. Public source values remain lexically unchanged in the derived union; CSV serialization and four provenance columns mean this union is not original-byte-identical. The full compressed source is496305507bytes and may be downloaded/reused locally; it is not wholly ingested. The output pack has1123 rows and remains declared FOCUS1.0.

The focused source search found provider export guidance but no verified reusable VMware/Alibaba billing sample adopted for this increment. Do not infer universal sample absence or native exporter support from the synthetic fixtures. Alibaba export guidance: [FOCUS official guide](https://focus.finops.org/docs/implementation/data-generators/alibaba-cloud/); VMware FOCUS context: [vendor article](https://blogs.vmware.com/cloud-foundation/2026/06/03/unified-cloud-value-accelerating-cloud-fin-ops-with-vcf-9-1-and-focus/). These references do not supply the fixture amounts or establish a real connection.

## Prepare and initialize

Use the running local verification stack/pinned Python environment from the base runbook. Commands run at repository root; default generated dataset/config/provenance paths are ignored under `.local/datasets/focus/`.

```sh
tests/engine_probes/.venv/bin/python tools/focus_data.py fetch-demo
tests/engine_probes/.venv/bin/python tools/focus_data.py build-samples
tests/engine_probes/.venv/bin/python tools/focus_data.py init --config .local/datasets/focus/six-provider.config.json
```

`build-samples` validates/reuses `.local/datasets/focus/full.csv.gz` or downloads the pinned gzip, then verifies full digest/size before extracting. `--full-source <existing-full-gzip>` selects a verified cache; it never trusts a filename alone. The checked GCP115 subset cache is reused only when its exact bytes and original-ordinal digest match. Synthetic rows come from the committed, clearly named fixture. Combined source/config/provenance are derived outputs; do not commit them or public account/resource identifiers.

`init` provisions a fresh disposable hb_focus sample namespace with three synthetic tenants and one explicit company audience. It does not replace an existing incompatible configuration. Existing official-only installations may apply migration002 and retain their existing config/query behavior; switching snapshots/company assignments requires a reviewed transition. The integration suite documented below resets only its disposable hb_focus namespace. The final sample execution after those tests installs the1123-row pack. This reset is test setup, not a product migration or recovery procedure; S3 historical originals and hb_local remain untouched.

## Store and approve

```sh
tests/engine_probes/.venv/bin/python tools/focus_data.py ingest --actor importer --version 1.0 --revision 1 --file .local/datasets/focus/six-provider.csv
tests/engine_probes/.venv/bin/python tools/focus_data.py confirm --actor approver --batch <batch-id> --manifest-digest <manifest-digest>
tests/engine_probes/.venv/bin/python tools/focus_data.py publish --actor approver --batch <batch-id>
```

Use exact ingest output in placeholders. These are separate local personas; they do not imply independent human or financial approval. Original/provenance columns, NULL tokens and exact decimal values are preserved. Full snapshot/dimensional controls and each tenant row digest must verify before atomic publication. Duplicate replay returns the same batch. Unknown effects continue blocking both individual and company reads; no automatic recovery/deletion or lease takeover was added.

## Individual and integrated queries

```sh
tests/engine_probes/.venv/bin/python tools/focus_data.py query --actor reader-north-a --provider GCP
tests/engine_probes/.venv/bin/python tools/focus_data.py query --actor reader-north-b --provider OCI
tests/engine_probes/.venv/bin/python tools/focus_data.py query --actor reader-company-north
tests/engine_probes/.venv/bin/python tools/focus_data.py query --actor reader-company-north --provider Alibaba
tests/engine_probes/.venv/bin/python tools/focus_data.py query --actor reader-foreign
```

North-a has1112 rows (AWS942, Azure51, GCP115, VMware2, Alibaba2). North-b has9 (OCI7, VMware1, Alibaba1). Their immutable company ID is company-north. The company audience explicitly grants only these two tenant keys and returns1121 rows across all six providers. Foreign has2 rows, a different company ID, and the same display company label to prove labels confer no access. Individual readers cannot select an arbitrary tenant list or elevate themselves into the company audience.

Optional initial configuration `views` entries require actor, company_id, active and unique nonempty tenant_keys. Company IDs must match all explicit members; duplicate aliases, cross-company membership and unknown tenant keys reject. View identity and expected members are immutable server records. Membership changes always lock the parent audience row, serializing insert/update/delete against reads; direct membership tamper causes complete view rejection. A revoked audience, inactive/missing member, mixed publication batches or unknown operation also denies the complete view.

Queries derive a separate PG/CH reader role from the audience actor and verify every authorized slice before union. Direct SQL under this role also excludes foreign data. Each returned detail row includes its tenant key. `tenant_breakdown` retains per-tenant controls; total `groups` combine only authorized rows and keep provider/service/currency/units distinct. No USD/CNY sum or FX conversion occurs. Pages use unique batch ordinals and `--after` plus `--expected-batch`; no duplicate tenant rows are silently deduplicated. Shared original object keys and other company source rows are never returned.

Aliases Azure/OCI/GCP/Alibaba are exact display-filter aliases for Microsoft/Oracle/Google Cloud/Alibaba Cloud. Original source values stay unchanged. `x_HB_EvidenceClass`, source artifact/digest and source ordinal remain in each lexical source row; synthetic rows never masquerade as observed exports. Reporting timezone and metadata-only currency/retention semantics remain as in the base runbook.

## Verify

```sh
tests/engine_probes/.venv/bin/python -m pytest tests/focus_import -q
```

Build the sample pack first. The suite includes prior official-only regressions, explicit audience/member lock races, role isolation, currency/NULL controls, full paging and mixed-batch denial. Fixtures reset hb_focus only; rerun sample initialization/intake/approval after testing to leave a queryable pack. Current acceptance is bounded sample ingestion/query, not full dataset capacity, live collectors/native VMware or Alibaba translation, deployed API/UI/authentication, pricing/issuance, FX or retention execution.

Attribution: official FOCUS Sample Data, FinOps Open Cost and Usage Specification, a Series of Joint Development Foundation Projects, LLC; [pinned revision](https://github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS-Sample-Data/tree/adbdd17a132984d6e8583c149c236d2199c3f5bc), [CC BY4.0](https://creativecommons.org/licenses/by/4.0/). Only hashes/aggregate observations and independently created synthetic fixtures are committed.
