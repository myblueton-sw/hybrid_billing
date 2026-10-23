---
type: Validation Report
title: Six-provider sample storage and authorized company query
description: Records actual sample ingestion and scoped company queries with official AWS Azure GCP OCI rows and clearly synthetic VMware Alibaba additions.
status: draft
---

# Six-provider sample execution

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Owner: Seung Woo Park. Machine: mac_mini. Root owns implementation and runtime execution; independent DBA/security/QA review is separate. [Design](../../../work/HB-27-six-provider/brief.md), [runbook](../../../ops/installation/six-provider-samples.md), [evidence](evidence.json). MIT PR25 was merged and verified on main `0e5533801d201820264eadc1afe50714a254d8da` before this feature publication; third-party dataset attribution remains applicable.

## Actual source coverage

| Provider requested | Source label | Stored rows | Evidence class |
| --- | --- | ---: | --- |
| AWS | AWS | 942 | Original official public sample values |
| Azure | Microsoft | 51 | Original official public sample values |
| GCP | Google Cloud | 115 | Selected official full-source rows, lexical values preserved |
| OCI | Oracle | 7 | Original official public sample values |
| VMware | VMware | 4 | Explicit synthetic contract fixture |
| Alibaba Cloud | Alibaba Cloud | 4 | Explicit synthetic contract fixture |
| Total | Six providers | 1123 | Mixed public and synthetic, query-only |

The official 1000-row file is pinned by SHA-256 `e91e5ac7edf01ed2c9d926f37ef7dc1ae2aae97956fea8da6c9ee488b1c2839e`. GCP extraction scanned all 5488359 rows of the exact full gzip, SHA-256 `c198cda74cbde4b11b9be289815bda7e9ac62042f1d023ad9bc8fbf7bff1018c`, 496305507 bytes. All 115 GCP source ordinals and the pinned subset/ordinal digest are preserved. The selected CSV SHA-256 is `c602b6871d0a0a8bae3a80daa8b3615c7ba948f304cd1f7213706776e6f5acde`. This is a reserialized subset, not an unchanged compressed-byte segment or whole-dataset ingestion.

The combined 1123-row CSV adds four per-row origin columns: evidence class, source artifact, source digest and original source ordinal. Original official lexical cells are unchanged. The eight synthetic rows have synthetic account/resource identifiers, deliberately chosen USD/CNY values and no claimed vendor tariff. Combined SHA-256: `986a297e5476f4747a2921f690745309256ffc5d117cf487ff8a1c2d7286b9c2`. Input remains declared FOCUS 1.0, never relabelled 1.2/1.3. Native VMware/Alibaba export compatibility and real-account collection are NOT_RUN.

## Storage and company authorization

CLI run `20261023T082547Z-2413fb`: **PASS**, all 42 commands exit zero. Fresh public fetch, verified pack build, explicit configuration, ingest, digest confirmation, publication, identical same-batch replay, complete pagination and individual/provider/company filters ran against actual PG/CH/S3 services. Published batch: `eace0377-ffe5-4d17-bfd0-c1b9bb111637`; manifest SHA-256 `51a0a507075f92bcd468fae92c7b18129153e85cac600c8ff61ee1949c506d5e`.

| Authorized scope | Rows | Meaning |
| --- | ---: | --- |
| reader-north-a | 1112 | AWS/Azure/GCP plus two VMware and two Alibaba rows |
| reader-north-b | 9 | OCI plus one VMware and one Alibaba row |
| reader-company-north | 1121 | Exact disjoint union of north-a and north-b |
| reader-foreign | 2 | Separate company, one synthetic row per added provider |

The foreign company's display label intentionally equals the north label; its immutable company ID differs. Labels, uploaded tenant fields, query filters and local binding hints grant no membership. The company actor has an explicit server-owned two-tenant allowlist and separate role-bound PG/CH policies. Direct unrestricted SQL under its runtime role excludes foreign rows; it cannot edit membership or read shared source manifests/original keys.

Individual and company pagination covers all expected distinct batch ordinals, with no duplication or loss. The north union and foreign set are disjoint and cover all 1123 rows. Company filters return AWS942/Azure51/GCP115/OCI7/VMware3/Alibaba3; the remaining two synthetic rows belong only to foreign. All additive measures were independently compared to Fraction sums of original lexical values per provider/service/currency/quantity-unit dimensions. USD and CNY remain separate; NULL/present/missing controls are preserved and unit prices are excluded from sums. Exact per-scope currency cost controls are in evidence.

## Verification and review

Final root suite `20261023T082253Z-71fb94`: **49 passed**, zero failures/errors/skips. This includes 28 pure/model/configuration cases and 21 real PG/CH/S3 integration scenarios. Existing official-only regression tests remain passing. New cases verify exact six-provider counts/provenance, same-company union, foreign exclusion despite equal labels, role denials, membership/binding tamper, inactive members, audience revocation, currency/unit/Fraction controls, full paging, mixed publication batches and membership INSERT/DELETE/UPDATE concurrency.

Checksummed migration002 leaves migration001 unchanged. Membership mutations lock the parent audience FOR UPDATE; a company query holds parent FOR SHARE through every member's materialization. Tests observe those admin mutations blocked by the lock and require subsequent reads to reject the changed membership. The complete expected member set is pinned; no discovered-row-only phantom grant or silently partial company response is accepted. Unknown storage reservations and whole-source atomic publication retain their previous behavior.

Independent DBA/security design and actual-code reviews: PASS before the licensing interruption. Final independent DBA/security/QA runtime and source evidence read-back: PASS, including all 16 source hashes, 49-test JUnit, 42 successful CLI commands, complete disjoint ordinal union and independent exact controls. DBA independently matched 972 dimensional measure checks with Fraction arithmetic and NULL counts. Reviewers did not mutate DBs or rerun integration tests. Root verified all recorded source/fixture hashes and raw-command/JUnit digests on resume; incorporating MIT main changed no tested source bytes, so the suite was not rerun for the documentation-only merge. Raw CLI rows/logs and identifier-bearing generated configurations remain ignored/local-only without a persistence guarantee. Only original synthetic fixtures, aggregate/hash evidence and documents are committed.

Final independent staged-scope/document review: PASS, exactly 16 scoped files; all recorded source hashes match index bytes and MIT LICENSE matches main. Metadata, links, whitespace and secret scans pass. No unrelated assets, official source rows, raw logs, environment files or generated datasets are staged.

## Limits and handoff

Actual sample collection/storage/query works locally. This is a bounded 16 MiB/10000-row complete-snapshot profile with trusted local personas and synthetic company assignments, not deployed user authentication or proof of real customer ownership. The full 5.48M-row source is scanned for selection, not wholly loaded. Official/public observation and synthetic fixture evidence stay distinct. No native VMware/Alibaba translator, live cloud collector, production API/UI, full FOCUS certification, financial approval, FX conversion, retention execution, destructive recovery or capacity acceptance is claimed.

The documented disposable hb_focus test fixtures reset only their sample namespace, then the final CLI run reinstalled the richer pack containing every prior official1000 row plus GCP115 and eight labelled additions. Historical S3 originals and the separate hb_local path remain untouched. The final local sample is left queryable. Configuration replacement is not a general migration/recovery feature; existing incompatible profiles still reject initialization. HB-27 remains In Progress for broader unfinished product scope; this report does not claim a merge.

Attribution: FOCUS Sample Data by FinOps Open Cost and Usage Specification, a Series of Joint Development Foundation Projects, LLC; [pinned revision](https://github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS-Sample-Data/tree/adbdd17a132984d6e8583c149c236d2199c3f5bc), [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/). Public lexical values are retained in derived local datasets; row-level origins identify additions and selection. Project-authored synthetic fixture and source code follow the root MIT license.
