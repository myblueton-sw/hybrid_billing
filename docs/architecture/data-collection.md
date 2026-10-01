---
type: Ingestion Design
title: Data collection authority grouping and ingestion methods
description: Defines provider and customer grouping, API push export and upload contracts, and completeness before billing eligibility.
status: draft
---

# Data collection and grouping

Part of [HB-7](architecture-readiness.md). Owner requests explicit collection methods and grouping criteria. Proposed contracts refine local v1.16 `provider-profiles.md`, `domain-contracts.md` and `detailed-requirements.md`; provider/API capabilities are not certified here.

## Canonical grouping basis

Use independent dimensions rather than one provider bucket:

| Dimension | Purpose | Boundary |
| --- | --- | --- |
| Installation/workspace | Deployment and authenticated tenant policy | Primary security/ownership context; provider is not a tenant |
| Customer/Party/contract | Servicing relationship, economic responsibility, sale/internal allocation | Customer grants and contractual attribution; one account is not necessarily one customer |
| Provider/product/source authority | Source family, contract/account role, actual cost/usage/invoice issuer and kind | Adapter and comparison axis; authoritative costs differ from reseller selling data |
| Source billing scope/account | Payer/billing account/subaccount/project/resource/site/cluster identity | Collect at authorized authoritative scope; retain temporal lineage |
| Period/revision | Usage/billing period, arrival/as-of, source correction and mapping revision | Completeness and reproducibility; late arrivals do not overwrite approved input |
| Org/service/resource | Department/project/cost center and service/resource dimensions | Derived approved effective-dated attribution; no implied access grant |

Canonical envelope proposal: installation/workspace ownership; source provider/product/authority/kind; scope/account; stable source identity/revision; event/usage period and received/as-of time; unit/currency/amount basis; mapping version; original artifact/row locator; coverage/validation state. Customer/org/contract assignment is explicit versioned attribution, not an inferred folder name or uploader identity.

Collect shared payer evidence once in an authorized ownership context, then derive customer-scoped manifests/views with permitted fields. Cross-workspace sharing requires explicit authorization; being a customer cannot grant raw shared payer access. Provider grouping serves connector health/source reconciliation; customer/contract grouping serves billing/disclosure. Reports pivot either axis without changing ownership or economic uniqueness.

Keep ProviderCost, MeteredUsage, ProviderInvoice, ResellerSale, CustomerCharge, Allocation and Forecast distinct. A new API/upload route cannot duplicate a source already received through another route. Stable source/authority/coverage identity and approved precedence rules resolve overlap; different payloads under one identity are conflicts/corrections, not silent last-write wins. Exact profile identity/precedence keys remain unresolved.

## Method contracts

| Method | Intended use | Controls and failure handling |
| --- | --- | --- |
| API pull | Approved cost/usage/invoice queries and ERP receipt/result reads | Least-privilege credential reference; version/account scope; page/cursor completeness, quota/backoff, snapshot/as-of and durable checkpoint; incomplete pages remain partial |
| Export/store read | Periodic files or queryable provider export datasets | Separate setup writer/collector reader; generation/list manifest, hashes/counts, availability/late correction, retention/query cost and bounded reads |
| Authenticated push/batch | Approved on-prem/AI/platform metering | Service identity and tenant/scope binding, version/digest/replay/idempotency and durable receipt; conflicting payload rejected; no submitted tenant trust |
| Webhook/notification | New evidence, external status or correction signals | Signature/identity/time/replay validation, duplicate/out-of-order handling; durable receipt then authoritative detail lookup; notification alone is not invoice confirmation |
| Manual upload | Approved missing-source, reseller statement, historical or ERP result evidence | Uploader scope; source/account/period/authority declaration, mapping preview/provenance; content/size/encoding/formula/malware/archive checks, quarantine and reviewed publication |
| Agent/collector | VM/Kubernetes/on-prem measurements | Stable resource/time/unit and sequence/reset/gap contracts, secure transport/offline buffering and coverage; observation is not independently authoritative cost |
| OCR/parsing assistance | Candidate extraction from approved invoices/documents | Candidate separated from original, evidence locators and verification; uncertain currency/decimal/identity blocks final input |

API/upload describes transport, not credibility. Successful receipt requires further authority, identity, semantics, coverage and reconciliation checks before billing eligibility. Overrides need approved reasons/evidence and revisions; no null-to-zero or hidden source mutation.

## Common lifecycle

1. Register/diagnose profile: owner, provider/authority, account role/scope, schema/mapping, period/availability, permissions and secret references.
2. Receive into bounded protected staging; authenticate scope and validate permitted fields. Unexpected secrets or original AI prompt/response bodies go through a controlled exclusion/rejection path, not ordinary raw billing storage.
3. Persist eligible immutable bytes and receipt identity/digest. Couple durable output with acknowledgment/checkpoint; reconcile object/DB writes rather than assuming atomicity.
4. Normalize using a fixed mapping revision; classify accepted/quarantined/duplicate/conflicting records and preserve locators/units/currency. Dropping invalid rows does not establish complete coverage.
5. Verify coverage/pages/files, counts/bytes and currency-specific totals. Reconcile source invoice/cost where applicable to the actual billing basis; do not fabricate an invoice for internal allocation/independent pricing.
6. Publish complete verified input/attribution manifests. Corrections create new revisions and impact previews, never overwrite approved inputs.
7. Monitor lag/missing scope/retry/quarantine/expiry. Replay checks authorized scope, retention eligibility and existing Claim/deletion/revocation controls.

Receipt IDs, source identities, business Claims and ERP submission keys are separate. Kafka can carry validated envelopes/artifact references but is optional for uploads/batches. OpenTelemetry is operational; an OTLP-based approved meter still needs the full collection contract.

## Kubernetes and on-prem lineage

Separate authoritative infrastructure cost → VM/node/resource observations → workload/namespace/pod attribution → approved customer selling/allocation rules. Preserve ownership/resource history and explicit idle/shared-cost policy. Avoid billing both a VM/node cost and its workload allocation as additional independent costs. VMware is optional; actual telemetry source/allocator and supported deployment remain unresolved. No OpenCost adoption or adapter certification is implied.

## Profile acceptance

Each real profile records provider/product/API/schema version, contract/account role, region/site, kind/authority, method, permissions, expected coverage/generation delay, quota/cost, stable IDs, correction/precedence rules and evidence. Supported/limited/unsupported/unverified is profile-specific, never inferred from provider name.

Planned cases: incomplete pagination; late correction; identical API/upload source; conflicting revision; shared payer disclosure; wrong-customer/malicious upload; webhook replay; resource move/reset/gap; broker replay; unknown authority; no usage versus unavailable data; OCR decimal error; forecasted gap. Integration/security/coverage/performance tests are **NOT_RUN**. Exact endpoints/formats and first support combinations remain unresolved.
