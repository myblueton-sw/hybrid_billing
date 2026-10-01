---
type: Data Disclosure Design
title: Cost and usage detail disclosure and durable lifecycle
description: Defines complete authorized cost explanations, usage lineage, historical storage and maintained retrieval with scoped retention and recovery.
status: draft
---

# Cost and usage detail lifecycle

Tracking: [HB-9](https://linear.app/hybrid-billing/issue/HB-9/define-cost-and-usage-detail-disclosure-and-durable-lifecycle). Accountable owner: Seung Woo Park. Root owns PM/planning/writing; separate read-only DBA and QA reviewers assess data/financial and disclosure/lifecycle contracts. Worked machine: `mac_mini`.

**Owner-confirmed requirement:** costs must be accompanied by complete usage details and explanatory information; that information must be stored, managed and maintained. Connect [collection](data-collection.md), [data ownership](data-ownership/billing-data-model.md), [administration](admin-design.md) and [current permissions](security-boundaries/identity-session-permissions.md). This is documentation/design; development requires owner planning acceptance.

Local basis: v1.16 `requirements.md:485–545`, `retention-access.md` and S04/C01/C02 screen specs already propose customer-safe detail/export, snapshot totals and archive/recovery. They are not all published in Git. This refinement makes continuity/maintenance explicit; it does not assert every addition came from the original source or that any path is implemented.

## User outcome and supported detail

A permitted administrator/customer opens a cost summary, filters period/service/account/resource/org dimensions, inspects underlying usage and charge components, understands adjustments and downloads the complete authorized result. Historical approved/issued periods remain retrievable within approved retention. A bounded list page is not the whole dataset.

"Complete" means all eligible rows/fields for the requested authorized scope and supported source granularity, with coverage/exclusions stated. It never means all tenants, shared payer originals, hidden purchase costs/margins or original AI prompt/response bodies. Source capability and customer disclosure are separate: coarse data cannot yield invented hourly/pod/request detail; restricted fields cannot become available through drill-down.

For each initial source/customer profile, declare available resolution, disclosed dimensions/fields, cost basis, historical window and coverage criteria. A supported usage-based charge must provide its permitted usage explanation. If a required explanation cannot be supplied, explicitly classify that combination limited/unsupported and resolve business acceptance; do not silently omit detail or fabricate zero. Non-usage charges show contractual basis rather than invented usage.

| Detail family | Required when applicable | Meaning/disclosure |
| --- | --- | --- |
| Identity/scope | Stable detail ID, source revision, customer/contract, service/product and permitted account/resource/org references | Protected internal evidence links; display names are not keys; frozen historical attribution |
| Time/resolution | Usage start/end, time zone, period, source resolution, received/as-of and revision | Usage time differs from arrival/invoice dates; approved period-split rules; no fabricated finer measurement |
| Measure | Original/normalized quantity/unit, conversion revision, quality/completeness and actual/estimated/unknown state | Supported exact decimals; incompatible units not summed; unavailable data not zero; public fields follow disclosure |
| Rate/basis | Applicable selling/allocation rate and denominator, tier/commitment/minimum basis, pricing revision | Quantity × rate is not universal. Fixed fees/credits/tax/rounding/adjustments have distinct component types/explanations |
| Amount | Currency, cost/selling/allocation/recognition basis, net/tax/gross classification, applicable FX/rounding revision | Separate business bases; disclose permitted amounts/rates only; no currency sum without approved conversion |
| Calculation/evidence | Input/result manifest, rule/contract/allocation revisions and permitted evidence references | Usage contributions and components explain charges; internal raw locators/payer IDs are not customer download URLs |
| Lifecycle | Publication/coverage, retained/archive/recall/expired/unavailable state and reason | Missing, restricted, partial, estimated and measured zero remain distinct; safe public references |

## Lineage and immutable history

Preserve permitted `SourceArtifact/SourceRevision` and locators → normalized measure/detail and `InputManifest` → priced/allocated `CalculationPartition/ResultManifest` → approved/issued document line and correction references → permitted projection/export. Reuse [existing model families](data-ownership/billing-data-model.md), not a second financial ledger. Store many-to-many contribution links: one source row can support several allocated lines, and one line can aggregate many measures. Include approved contribution quantity/amount/weight and allocation basis where applicable without counting source cost twice.

Freeze source/input/mapping/unit/contract/org/rate/FX/allocation/rounding revisions and full eligible result manifests per approved outcome. Deliveries obey approved replacement/append rules; late changes create linked revisions/corrections. Current names/org assignments cannot rewrite historical liability/attribution. Drill-down, projection rebuild, archive recall and export cannot consume new Claims, reissue documents or resubmit ERP commands.

Economic revisions are immutable; current row/field/recipient authority is checked for queries, generation, recall and download. Retained evidence does not preserve former access. Safe explanations remain usable without inaccessible shared-payer links or formulas/identifiers/totals/denominators that reveal hidden costs.

## Summary query and export

1. Bind queries to authenticated workspace/customer authority, fixed input/result or issued revision, period/time zone, dimensions/filter, currency/basis and current disclosure revision. Separate live/provisional from approved historical views.
2. Summary, permitted counts and full-result subtotals use one eligible authorized set. Stable cursor pages need deterministic tie-breakers; expiry restarts instead of mixing revisions. Label page subtotal, filtered permitted subtotal and full-document total separately.
3. Reconcile applicable components on the fixed revision: usage charges plus fees/adjustments/credits, taxes and explicit rounding bridge to the relevant document total. Do not force filtered usage subtotal to equal the invoice. Explain safe differences. Disclosure filters do not shrink economic allocation denominators or recalculate charges.
4. Use bounded server queries/aggregation, not wholesale browser loading. Dimensions/search/joins/counts/errors obey current authority. Analytical projections need policy enforcement and query/rate budgets; [HB-8 ED-01/04](../reviews/HB-8-external-design/findings.md) remain open pending their full reviewed contracts.
5. Separate page export from all filtered authorized rows. Full export freezes a target manifest and generates a protected asynchronous package. Supported format/chunk limits remain decisions. Preserve decimal values/identifiers/units/time/basis and safe serialization; validate formulas/active content. Manifest records snapshot/filter, permitted count, currency/basis totals, parts/digests and completion.
6. Recheck authority during work/delivery. If it shrinks, invalidate/restart or explicitly produce a reauthorized partial artifact with updated manifest/totals; never claim complete original scope. Create customer-safe files from permitted fields rather than hiding raw columns/sheets. Temporary export expiry does not remove authoritative history.

Within identical revision/scope/filter/currency/basis, summary, complete permitted detail and export totals agree under approved arithmetic. Reject unsupported mixes. Different bases/disclosed scopes require a safe reconciliation bridge or explicit limitation, not misleading equality.

## Pricing policy and charge finalization

The owner also asks how definitive pricing logic is decided and finalized. Separate **approved pricing policy** from **approved calculated amount**, and both from document issuance and external ERP confirmation. A saved rate, completed job or invoice delivery does not establish all these outcomes. Actual rates, supported pricing modes and the complete order of calculation operations are not selected by this document.

| Decision bundle | Required choices/evidence | Proposed responsible roles |
| --- | --- | --- |
| Commercial basis | Contract/customer/service scope; usage-based, fixed, tiered, commitment/minimum or other supported components; supplier cost, sale and internal allocation distinct | Owner/commercial and finance, using contract/source evidence |
| Measurement and price | Meter/unit and denominator, source quality/coverage, rate currency, quantity/time boundaries, tier accumulation/reset, effective period and partial-period/proration rules | Finance with data/source owner; DBA verifies representability and lineage |
| Modifiers and conflicts | Discount eligibility/stacking/priority, fixed fees, minimum/maximum, credits, shared allocation, currency conversion and complete operation order | Finance/commercial proposes; independent billing/DBA checks boundary/collision examples; PC-01 remains OPEN |
| Numeric and tax basis | Supported exact precision, rounding points/mode/residual distribution, net/tax/gross and applicable tax treatment/FX authority/date | Named finance/tax-policy owner; DBA validates approved arithmetic, without inventing jurisdictional rules |
| Activation and control | Draft/approval roles, effective dates, contract override precedence, overlap/gap detection, amendment and historical correction policy | Owner/finance/security approve responsibility ceilings and prohibited self-approval |

**Policy finalization:** register each unresolved choice and accountable named decision owner → draft a versioned policy with source/contract evidence and proposed alternatives → simulate independent expected cases and historical impact → review finance/commercial intent and independent billing/security correctness → approve the fixed policy revision and publish its authorized scope/effective dates. Publication cannot activate a conflicting/unsigned or still-blocked rule. Treat ordinary activation prospectively; retroactive application requires an explicit impact-reviewed correction/reapproval path. A changed draft or amendment is a new revision, never an edit to an approved historical policy.

The decision packet includes representative independent expected amounts plus tier boundaries, unit denominators, no/missing usage, fixed fees, negative corrections, overlapping policies, month/contract boundaries, discount/minimum order collisions, multiple currencies and rounding residual cases. Show customer impacts and the exact chosen formula/operation sequence. A finance approval cannot substitute for unresolved deterministic semantics or failing monetary oracles. Rules may differ by approved contract/profile; do not impose one universal formula. Compare against original authority where appropriate; an approved customer selling price need not equal provider purchase cost.

**Charge finalization:** verify eligible usage/coverage and contractual applicability → bind the published effective policy and all input/unit/org/allocation/FX revisions → calculate explicit component detail → reconcile quantity/currency/basis totals and independent expected cases → resolve blocking errors and explicitly eligible warnings under approved acknowledgment policy → approve the fixed amount/evidence snapshot → separately authorize issuance/submission. The warning contract remains HB-6 PC-02 closure work. Any changed input/policy/amount invalidates the approval for the affected draft; after issuance use linked corrections instead of overwriting history. Preserve Claim uniqueness across recalculation and retries. Uncertain external outcomes remain unknown until reconciled; approval is not payment or ERP posting confirmation.

Store the policy revision, decision/approval actors and dates, effective interval, calculation/component trace, expected-case references, publication state and correction links alongside retained usage/result dependencies. Customers receive the permitted selling/allocation explanation; restricted supplier pricing and margin operands remain protected. Owner Seung Woo Park obtains named commercial/finance/tax/security responsibility; those specialists are proposed roles, not currently assigned external consultants. Exact choices stay in PP-07 and PC-01/02 until evidenced and independently reviewed.

## Durable storage and maintained availability

Persist eligible bytes and transactional ownership/publication/manifest metadata before reporting complete receipt/detail. UI caches, staging files, broker retention and sampled telemetry are not durable usage history. Physical engines remain candidates under [technology assessment](technology-assessment.md). Derived projections are replaceable; authoritative approved detail remains retrievable from retained immutable evidence/results and dependencies without relying on an expired provider API.

| Stored class | Dependencies/owner | Maintenance boundary |
| --- | --- | --- |
| Permitted raw evidence | Revision/locator/digest, ownership/coverage, encryption/key/schema references; source owner/operations | Protected quarantine separate from eligible evidence; shared payer reference checks |
| Normalized measures | Original/normalized unit/time/quality, mapping/conversion, lineage/input manifest; data owner/DBA | New correction revisions; projection deduplication cannot erase approved historical inputs |
| Approved results/documents | Complete eligible calculation detail, contribution bridge, versions/corrections; finance/DBA | Retain deterministic explanation/reproduction dependencies within policy; latest-rate recomputation is not history |
| Customer/analytical views | Data/policy revisions and permitted schema; analytics/security | Rebuild eligible retained revisions with current grants, deletion controls and publication completeness gates |
| Export/recall temporary copies | Frozen target/recipient/policy, digest/expiry/job; operations/security | Explicit shorter lifetime, bounded recall cache and download/cleanup checks; external downloaded copies outside automatic recall |
| Policy/hold/control/audit/backups | Approved retention, shared references, latest deny/deletion watermark and restore inventory; operations/security/finance | Separate active/archive/backup deletion states; durable lifecycle audit independent of telemetry |

Define revisioned policy per class/scope: retention basis/date, duration, hot/archive windows, recall availability target, deletion grace/method, responsible person, approval and dependencies. Durations/jurisdictions/SLO/RPO/RTO are unresolved; maintenance means neither indefinite retention nor a declared legal obligation. Copy/replay/tier changes cannot reset retention clocks.

Archive transition verifies target version, bytes/digest, retrieval/decryption, row counts/unit/currency/basis totals and required schema/policy/result dependencies. Switch references after verification and retain a valid source until completion. Expose retained, archived, recall-queued/running, available, failed, expired and unavailable states. Explain missing/corrupt/key-unavailable causes safely; no invented recall time or zero substitute.

Deletion checks holds, financial dependencies, active jobs and shared references immediately before removal, serialized with new references/holds. One customer's offboarding cannot destroy another customer's evidence. Preserve only justified minimal Claim/duplicate/control records for approved replay/backup horizons; no hash-anonymity or permanent-retention assumption. Apply deletion across managed stores/projections/export caches; backup expiry is separate. Local v1.16 retention-access is the detailed basis pending PP-08 English reconciliation.

Back up metadata and referenced evidence/detail/dependencies coherently. Restore verifies counts/totals, readability/decryption and lineage; isolates old writers/credentials, verifies/applies latest independent deny/deletion watermark, rebuilds only eligible projections and reconciles external uncertainty before reopening. Missing dependencies/latest controls leave data quarantined/unavailable. Rebuild is read-model recovery, not billing execution.

## Administration and capacity

Scoped administrators monitor rows/bytes/growth by class/source/customer/period, completeness/late arrivals, projection revisions/lag, archive/recall/deletion/backup work, shared-reference/hold reasons, checksum/schema/key expiry and failed partitions. Operational counters do not grant financial detail or reveal unauthorized customers. Show policy/impact previews, actual owners, job retries/checkpoints, bounded concurrency and audit. Per-store job target counts reconcile succeeded/held/failed/pending outcomes; deletion status distinguishes active deletion from pending backup expiry, with as-of/unknown states.

Size using source resolution, normalized/calculated row expansion, revisions, indexes, replicas/backups, tiers, merge/temp space, query/export/recall concurrency and correction horizon. More than one million records/day and at least thirty million/month is a planning premise, not measured capacity. Queries/maintenance cannot starve ingestion or closing. Quota/disk pressure yields visible bounded backpressure or approved degraded operation; never silent sampling/truncation, deletion of retained finance evidence or false complete history.

## Acceptance design and closure

| Case | Required outcome; all runtime cases NOT_RUN |
| --- | --- |
| UD-01 lineage | Many-to-one and one-to-many contribution links persist; workload allocation does not add duplicate infrastructure cost |
| UD-02 totals | Summary/full detail/export agree at identical revision/filter/scope/currency/basis; page subtotal distinct; fee/tax/credit/rounding bridges explicit |
| UD-03 source honesty | Coarse resolution, unknown/missing, measured zero, estimate and fixed fee remain distinct; no invented detail or universal quantity × rate |
| UD-04 disclosure | Shared payer/customer paths protect rows/fields and derived facts across counts/search/links/cache/export; inaccessible raw evidence does not prevent safe detail |
| UD-05 history | Late correction and org/rate/source-mode changes preserve historical detail/corrections; revised access never revives prior grants |
| UD-06 complete export | More than 100,000 rows page/export completely without wholesale browser loading; retries/part failure/cursor expiry/revocation cannot falsely complete |
| UD-07 archive | Transfer interruption/corruption/key/dependency failure keeps valid copy or explicit unavailable state; recall changes no financial outcome |
| UD-08 retention | Shared-reference/hold races preserve eligible evidence; temporary expiry leaves authority intact; managed backup deletion state is honest |
| UD-09 recovery | Rebuild/restore preserves eligible historical detail without new Claims/ERP effects; latest deny/deletion controls prevent resurrection |
| UD-10 capacity | Concurrent ingestion/closing/query/export/archive/deletion/recall verifies totals/counts, budgets and pressure states; no silently lost usage |
| UD-11 policy | Independent tier/unit/discount/minimum/FX/rounding examples and effective-date conflicts have approved expected amounts; published revisions cannot be silently edited |
| UD-12 final charge | Reconciliation/errors/warnings and snapshot approval precede authorized issuance; changed draft evidence invalidates approval; issued corrections and uncertain ERP outcomes stay distinct |

Named finance/security/data/operations owners choose first supported resolution/disclosure, numeric/component rules, query/export/recall targets and class-specific retention/recovery (PP-07). Reconcile existing requirements/work/contracts and S04/C01/C02/I04/B/I paths in English with these cases (PP-08), then independently review content/trace (PP-09). Existing screen families host the flows; no new screen is required solely for this document.

HB-6/HB-8 findings remain open. This supplies detail/lifecycle inputs, not complete enforcement/arithmetic/source-mode closure. [Verification](../reviews/HB-9-usage-detail/verification.md) records document checks; actual storage/retention/query/UI/export/integrity/recovery is **UNVERIFIED / NOT_RUN**.
