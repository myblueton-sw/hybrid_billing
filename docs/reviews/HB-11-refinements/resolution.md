---
type: Planning Supplement
title: Contract supplementation and remaining planning decisions
description: Tracks concrete contracts supplied for thirteen findings and the profile policy and canonical trace work still needed for final closure.
status: draft
---

# Planning supplementation

Tracking: [HB-11](https://linear.app/hybrid-billing/issue/HB-11/specify-missing-planning-contracts-for-the-thirteen-refinements). Baseline: `d03252bb0d52db7e15a501b57507ca11bac779d6`, 2026-10-01. Accountable owner: Seung Woo Park. Root owns PM/planning/writing; separate read-only architecture, DBA and QA agents reviewed drafting proposals and concrete contracts. Worked machine: `mac_mini`.

The owner authorized supplementation after [HB-10](../HB-10-unresolved/findings.md). This package supplies proposed English contracts rather than another gap inventory. It does not select actual rates, tax/rounding values, products, retention durations, specialist assignments or scope exclusions. It preserves original source and historical review verdicts. All product execution remains NOT_RUN; owner planning-complete declaration is absent.

## Current finding disposition

**Documentary progress: all thirteen have a supplied, independently reviewed supplemental contract. Final finding closure: OPEN pending necessary profile/policy decisions and PP-08 canonical trace reconciliation/re-review.** Contract review PASS means the supplement is coherent within its scope; it is not acceptance of every product decision or implementation enforcement. No new finding IDs are added. Requirements/work/screen/test integration is not claimed complete by architecture entry links.

| Finding | Supplied contract / acceptance | Remaining closure dependency |
| --- | --- | --- |
| PC-01 | [Input/financial contract](../../contracts/billing/inputs-pricing-and-erp.md), default order and revisions; H11-M01–03 | Actual component/quantity/FX/tax/numeric choices and approved profile exceptions; PP-02/07/08 trace |
| PC-02 | [Input/financial contract](../../contracts/billing/inputs-pricing-and-erp.md), evidence-bound warnings; H11-W01–02 | Eligible warning policy and named acknowledgment authority/ceilings; PP-02/07/08 |
| PC-03 | [Access/trust contract](../../contracts/security/access-and-trust.md), conditional platform-only engine; BE-01 | Included engine profile or explicitly reviewed exclusion; approved enforcement contract/profile and PP-03/07/08; runtime enforcement verified later |
| PC-04 | [Input/financial contract](../../contracts/billing/inputs-pricing-and-erp.md), dated ERP profiles and distinct outcomes; H11-E01–03 | First actual ERP product/version/deployment/method and vendor/sandbox evidence; PP-04/07/08 |
| PC-05 | [Operations contract](../../contracts/operations/holds-onboarding-and-favorites.md), separate hold commands/states; DH-01–05 | Finance/security roles, external hold support and in-flight boundary; PP-05/07/08 B11/API trace |
| PC-06 | [Operations contract](../../contracts/operations/holds-onboarding-and-favorites.md), distinct outcomes/cohorts/waits; OB-01–05 | Cohort/milestone/window evidence and product/data owner; PP-06/07/08 S02 trace |
| PC-07 | [Operations contract](../../contracts/operations/holds-onboarding-and-favorites.md), personal permission-safe favorites; FV-01–05 | Preference retention/regrant/termination policy; PP-06/07/08 U01 trace |
| ED-01 | [Access/trust contract](../../contracts/security/access-and-trust.md), mandatory query boundary; QG-01–04 | Initial read/tenancy profile and independently reviewed enforcement contract/mechanism; PP-07/08; runtime enforcement verified later |
| ED-02 | [Access/trust contract](../../contracts/security/access-and-trust.md), governed trust replacement and epoch; TR-01–02 | Named independent authorities, proof/recovery and first auth profile; PP-07/08 M02 trace |
| ED-03 | [Access/trust contract](../../contracts/security/access-and-trust.md), every retrieval destination/credential policy; FP-01–02 | Source/network endpoint registry and approved budgets; PP-07/08 AR-02/QA-05 trace |
| ED-04 | [Input/financial contract](../../contracts/billing/inputs-pricing-and-erp.md), exact analytical profiles; H11-A01–03 | Numeric policy, selected profile/version and independent expected cases; PP-07/08 |
| ED-05 | [Input/financial contract](../../contracts/billing/inputs-pricing-and-erp.md), delivery modes/atomic publication; H11-S01–04 | Actual stable source identity, scope/mode/revision/precedence per supported profile; PP-07/08 |
| ED-06 | [Access/trust contract](../../contracts/security/access-and-trust.md), conditional trained-version withdrawal; TW-01 | Actual training inclusion or explicitly reviewed exclusion, audiences/privacy lineage policy; PP-07/08 |

## Detail retention and policy decision packet

The latest [owner requests](../../planning/recent-owner-requirements.md) require complete authorized detail, durable maintenance and definitive pricing. [Usage lifecycle](../../architecture/usage-detail-lifecycle.md) remains the canonical lifecycle design; these decision packets make its pending choices actionable. D references are local decision references, not Linear tickets. Each is currently UNRESOLVED and Seung Woo Park remains accountable for obtaining a named responsible person; role names are requested responsibility only.

Each approved packet must record decision ID, named decision owner, alternatives/evidence, selected profile/scope, validation references, approver/date, revision/effective period and downstream blocking effect. Draft → EvidenceReady → IndependentlyReviewed → Approved → Effective is the proposed lifecycle, with rejected/superseded alternatives. Changes create new revisions and invalidate affected approvals. Do not label a still-incomplete packet approved because its document or PR merged.

| Decision | Required concrete fields / requested role | Blocking effect until resolved |
| --- | --- | --- |
| D-01 first support and authority | Operating/isolation profile, providers/ERP/engine/analytics/intelligence scope; named finance/security/data/operations owners, authority separation and initial deployment budget; owner PM | Cannot assume support combinations, automatic engine/training inclusion or named specialist sign-off |
| D-02 source/detail/disclosure | Source/schema/version/method, actual resolution, scope/identity/mode/precedence/coverage, available versus disclosed fields, unit/time/currency/basis, historical window and non-usage component explanations; source/data/security | Cannot promise unsupported fine-grained detail or complete eligible current input; hidden raw/margin fields remain protected |
| D-03 pricing and warnings | Components, quantity/allowance/tier/reset/proration pools, full sequence/exceptions/conflicts, exact precision/bounds/rounding/residuals, FX authority/date/fallback and tax-policy owner, independent expected amounts; warning predicates/roles/evidence; commercial/finance/tax/DBA/security | No policy activation or final charge approval while arithmetic/eligibility/acknowledgment is undefined; no real rate or jurisdictional tax rule inferred |
| D-04 storage and retention | Per stored class/scope: authority versus derived/temp, basis date and justification, duration, hot/archive windows, dependencies/shared references/holds, grace/method, keys/schema dependencies, object versions/replicas/caches/backup expiry and minimal control/Claim replay horizon; finance/security/privacy/data/ops | Cannot claim historical availability/deletion completed or retain indefinitely by default. Tier/copy/replay cannot reset retention clocks; external downloaded copies distinguished |
| D-05 capacity and recovery | Workload/expansion/skew, query/export/recall budgets, full package format/chunks, archive verification/recall targets, closing windows, RPO/RTO, dependency/key availability and latest independent deny/deletion/withdrawal watermark; ops/DBA/security | Cannot certify sizing/SLO or reopen restore with missing latest controls; rebuild/recall creates no Claims/issuance/ERP effects |
| D-06 identity and conditional models | Auth/MFA/session/revocation profile, trust proof/recovery and independent activation authority, service credentials/fetch destinations; training inclusion/audiences, learned artifact lineage/withdrawal/backup and evaluation thresholds; security/privacy/intelligence | Sensitive operations fail closed without approved current authority; tenant-data training remains blocked until its distinct approved contract |
| D-07 operating/product policies | Hold authority/capabilities/boundaries, onboarding cohort/eligibility/window/milestone, preference retention/regrant/termination and scoped UI/API behavior; finance/product/data/security/QA | Detailed B11/S02/U01/API/work trace and final PC-05/06/07 closure remain pending |

The storage inventory includes permitted raw evidence, normalized detail, approved calculation/document components, derived customer/analytical projections, temporary exports/recall copies, audit/control records and backups. Approved dependencies and shared payer evidence cannot be removed by one customer's offboarding. Hold/reference/deletion races require serialized eligibility checks. Archive transfer verifies bytes/digest, counts/unit/currency/basis totals, decryption and required dependencies before switching references; failure retains a valid source or explicit unavailable state. Backup expiry and active-store deletion have separate observed states.

Restore isolates old writers, verifies coherent metadata/evidence, applies latest independent deny/deletion/withdrawal controls and reconciles external uncertainty before reopening. Unknown latest controls quarantine data; no resurrection through cache, model or backup. These obligations supplement the decision packet, not numeric retention or legal certification.

## Detail and lifecycle acceptance additions

All cases are NOT_RUN. They refine existing UD-01–12 rather than claim extra original source coverage.

| Case | Independent expected outcome |
| --- | --- |
| H11-D01 | Summary/full authorized detail/full export use identical revision/scope/filter/currency/basis; component bridges distinguish filtered/page totals and fixed fees/credits/tax/rounding |
| H11-D02 | Source granularity/limited fields/no-use/estimate/unknown/non-usage basis are honest; customer-safe explanation exposes no hidden supplier operand or shared payer locator |
| H11-R01 | Archive interrupted/corrupt/key/dependency failure retains valid history or explicit unavailable state; recall changes no financial outcome |
| H11-R02 | New hold/shared reference versus deletion race preserves eligible evidence; active deletion and pending backup expiry are separately recorded |
| H11-R03 | Restore/rebuild honors latest deny/deletion/withdrawal controls with no new Claim/document/ERP action; missing controls leave data quarantined |

## Remaining integration and execution order

1. Start the PP-07 decision register with D-01 profiles/owners and evidence; finish actual domain choices using content-package inputs. No staffing, cost or date is invented.
2. Reconcile supplied security and source contracts first, then numeric/order/arithmetic/warnings and relevant ERP/holds. Complete profile-specific detail/retention/recovery choices. Metrics/preferences can be integrated independently where prerequisites exist.
3. Actual PP-02–06 remediation/PP-08 child tickets remain to be created for complete canonical reconciliation. HB-11 is one integrated supplementation package, not proof those packages were executed. Preserve local originals and reconcile English requirements/work/API/screens/test records with stable IDs under one writer.
4. Independently re-review actual decision/trace closure under PP-09; explicitly accept/defer conditional capabilities without dropping required customer outcomes. Owner separately declares planning complete before any product implementation or experiment.

[Verification](verification.md) records design/change review and document checks. Only document/source-contract analysis and synthetic reasoning were performed. No implementation, benchmark, provider/ERP connection, retention job or model was executed; no external human finance/legal/security approval is asserted.
