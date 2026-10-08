---
type: Design Review Report
title: HB-20 comprehensive architecture re-review
description: Records source coverage, architectural corrections and unresolved implementation and operational evidence.
status: draft
---

# Comprehensive re-review

Tracking: [HB-20](https://linear.app/hybrid-billing/issue/c04558aa-62af-4914-a71e-785049be5fcb). Owner: Seung Woo Park. Root owns planning, integration and all document writes. Separate read-only architect, DBA and QA reviewers inspected the proposal and relevant inputs. Branch: `docs/HB-20-cost-data-review`; machine: `mac_mini` during execution. No product implementation is authorized by this report.

## Verdict and evidence limits

The previous narrow proposal was insufficient as a complete architecture: FAIL for comprehensive design readiness. The revised [object storage architecture](../../architecture/data-ownership/object-storage-data-architecture.md) received independent architect, DBA and QA PASS for its logical proposal only. Physical data design, engine/version selection, operational feasibility and performance remain OPEN / UNVERIFIED. [Comparative research](../../architecture/performance-operations-comparison.md) defines alternatives and unrun acceptance profiles.

The [input inventory](input-inventory.json) records 48 authorized local inputs with byte hashes. Inventory is not proof of reading every source sentence. QA mapped all 43 requirement groups and work items and inspected 35 detailed contracts and 210 acceptance references (119 original, 91 added). All 210 acceptance references and the 35 detailed-contract execution statuses remain NOT_RUN; implementation work remains NOT_STARTED. Cross-reference consistency is not functional verification.

Original DOCX identity and structure were inspected (SHA-256 `e7772927bc521ee035d50a7762b2f75611b80b2162e385878dc4c36c6c0db84d`, 1,443 XML paragraphs and 102 tables; the source inventory separately lists 123 chapters). This does not assert a new word-for-word semantic audit of the entire original document. Reviewers read relevant domain contracts and targeted original requirement material. Local planning/HB-16 inputs may be unpublished or unmerged; reviewing them does not certify their adoption. The inventory captures that distinction.

## Material findings and resolution

| Finding | Baseline severity and consequence | Revised logical resolution | Remaining evidence |
|---|---|---|---|
| File storage treated as sufficient table management | HIGH: ambiguous concurrency and snapshots | Explicit format/catalog ownership and pinned release manifests | Versioned commit, fence and pin protocol |
| Physical source identity and financial overlap undefined | HIGH: duplicate or omitted money | Receipt, source identity, revisions and economic Claims separated | Provider keys, multiplicity, physical constraints and cardinality |
| Internal cost and Marketplace roles incomplete | HIGH: cost/sales double counting | Internal cost facts and distinct supplier, customer, fee, settlement and eligibility measures | Source-specific mappings and independent oracles |
| CSP amortization implied as new engine | HIGH: exceeds original scope | Consume provider RI/SP/CUD results; internal depreciation separate | Profile capability/readiness contracts |
| FOCUS, measurement and billing-basis eligibility underspecified | HIGH: invalid sums or unnecessary blocking | Distinct cost bases; event/counter/gauge/interval profiles; basis-specific completeness | Exact source schema and unit/reset rules |
| Numeric, FX and full-pool behavior insufficient | HIGH: engine-dependent monetary results | Exact coefficients/scales, explicit purpose and complete pools | Numeric bounds, schema/FX policies and compatibility tests |
| Lake retention and recovery missing maintenance races | HIGH: deleted evidence or stale eligibility | Pins, holds, unknown-commit reconciliation and independent restore fences | Fault injection, cleanup and recovery exercises |
| Lake/catalog access bypass absent from access model | HIGH: customer/field disclosure | All routes enforce current authorization; metadata/functions/objects covered | Negative access and revocation tests |
| ClickHouse assumed to be canonical collection destination | MEDIUM: unnecessary copies and correction burden | Objects are intake evidence; lake is analytical foundation; hot serving measured separately | Representative route comparison |
| Catalog and day-two costs omitted | MEDIUM: understated operational burden | Catalog shortlist, maintenance ownership and BM01–17 | Measured operator hours and whole-system cost |

## Full requirement-group coverage

This is a review map, not implementation acceptance. Group numbers refer to planning requirement IDs HB-01 through HB-43, not Linear ticket IDs. PG means transactional business authority; lake means raw/canonical/calculated objects and metadata; serving is an optional reproducible projection; control covers authorization/jobs/lifecycle. Conditional and later scope remains conditional and later.

| Group | Area | Data and control review focus |
|---|---|---|
| 01 | Operating profiles and parties | Distinct workspace, party, customer and direct/resale responsibilities in PG |
| 02 | Organization/account changes | Effective and recorded revisions; atomic full-pool treatment and deduplication |
| 03 | Permissions/delegation | Current row/field authorization before aggregation and export |
| 04 | Provider readiness | Separate connection, source coverage, normalization and billing eligibility |
| 05 | Connector profiles | Original providers and source authority/completeness; deferred integrations preserved |
| 06 | Receipts and original evidence | Object receipt registry, conflicts, replacement boundaries and hostile files |
| 07 | FOCUS | Versioned semantics and distinct billed/effective/list/contracted bases |
| 08 | Evidence reproducibility | Pinned schema/engine/policy and sparse contribution graph |
| 09 | Reconciliation | Independent source-document counts and basis-specific totals |
| 10 | HawkEye | Deferred integration, existing contract retained |
| 11 | TokenMeter | Deferred integration, existing contract retained |
| 12 | On-premises cost | Internal depreciation, power, shared pools, idle cost and residuals |
| 13 | VMware history | Time intervals, counter resets, units and unknown measurements |
| 14 | Allocation | Complete denominator, fixed revision and exact residual distribution |
| 15 | Contract lifecycle | Terms and termination distinct from settlement state |
| 16 | Selling rates | Complete allowances, tiers, minimums and pricing pools |
| 17 | Formulas | Bounded typed rules, policy collisions and approval |
| 18 | FX | Purpose, quotation denominator, fees, fallback and exact arithmetic |
| 19 | Commitments | Provider-supplied results; no new CSP amortization/optimizer engine |
| 20 | Marketplace | Buyer, infrastructure, seller fee, settlement and benefit eligibility separate |
| 21 | Claims | Exact global economic coverage, concurrency and mixed-obligation atomicity |
| 22 | Prepaid/credits/disputes | Transactional balances, holds and receipts |
| 23 | Approvals | Digest and revision-bound decisions |
| 24 | ERP intents | Purpose-scoped idempotency, unknown outcomes and monthly separation |
| 25 | ERP profiles | Odoo/SAP profile guards retained |
| 26 | Common XLSX | Initial delivery requirement preserved, exact values and partial failures |
| 27 | Disclosure/export | Allowed fields, metadata safety and current access |
| 28 | Own documents | Immutable issued revision and numbering |
| 29 | Correction delivery | Linked corrections, caps/holds and no historical overwrite |
| 30 | Administrative views | Current PG state, reproducible analytical releases, zero/unknown/stale distinctions |
| 31 | Schedules/alerts | Target manifests, independent deadlines and exemptions |
| 32 | AI assistant | Phased API scope, no approval by conversation, retention boundaries |
| 33 | Common commands | Bounded cursors, actor/client permission intersection and expected revision |
| 34 | Jobs/events | Leases, fencing, outbox references and partial versus unknown outcomes |
| 35 | Plugins | Phase boundaries, isolated pinned modules and retained execution history |
| 36 | Self-hosted | All selected stores/catalogs, offline/proxy/CA and operator ownership |
| 37 | Recovery | Coherent data recovery plus independent latest control fences and external uncertainty |
| 38 | Simulator | Independent synthetic numerical oracles; future execution scope remains gated |
| 39 | Scale | >100M source/day, >3B at 30 days and concurrent operational load |
| 40 | Licensing | Exact component versions, rights/notices, SBOM and expiry |
| 41 | Release evidence | Real profiles, money/security/scale and two-period pilot evidence |
| 42 | Identity/access | Deny-first behavior and independently current revocation |
| 43 | Retention | Shared Parquet files, snapshots, references, backups and tombstones |

## Review provenance and next gates

Architect reviewed the complete revised logical proposal and found prior storage/publication/lifecycle gaps addressed. DBA reviewed the revised proposal against local Claim/numeric and relevant billing/domain contracts; physical identity, exact numeric profiles and Claim enforcement remain mandatory design work. QA reviewed group coverage and revised logical boundaries; it did not execute the acceptance cases. Root integrated the findings and authored the documents. No reviewer wrote the reviewed files.

Required next decisions: maximum daily/burst envelope, freshness and close deadlines; source keys/multiplicity; canonical schemas and numeric profiles; catalog commit/pinning/cleanup protocol; exact financial overlap constraints; physical partition/sort/file/index choices; writer/reader versions; retention/RPO/RTO; operational and cost budgets. The benchmark plan is a proposal for later authorized validation, not permission to begin product work.

Some authorized local v1.16 planning files still contain the superseded >1M/day premise. Current scoped architecture documents point to >100M/day. Full canonical planning synchronization remains explicit follow-up; untracked local inputs were not silently staged or declared synchronized.

GitHub PR creation previously failed with `must be a collaborator (createPullRequest)`. A pushed branch is not a PR, merge, planning closure or completed ticket. Authorized PR integration and merge read-back remain pending.
