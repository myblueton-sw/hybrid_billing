---
type: ImplementationPlan
title: Explicit synthetic document basis and retained ERP preparation
description: Connect governed charge evidence to real internal ERP preparation without inventing invoice tax or quantity semantics.
status: draft
---

# Outcome and ownership

HYB-340, machine:mac_mini, follows ef6ebd5. Root owns implementation and integration; refusal_review is the independent financial/security reviewer and platform_refusals reviews evidence. The intended outcome is a real erpledger.Registry.Prepare record derived from immutable explicit synthetic document evidence and a currently valid owned charge approval. It is not ERP transmission, posting, issuance, approval consumption or production tax policy. No new default grants or public endpoint are included in the first internal adapter.

# Discovered producer gap

Independent source review confirms rating.Charge records rule/kind/source-line/amount, not invoice net/tax/quantity. Aggregate adjustments need not map to one source line. Current seeded USD15 input has no quantity or tax. Organization profiles provide parties/path/currency, not tax or monetary interpretation. Existing component statements belong to another provenance and cannot substitute. Preserve the quantity-less candidate as a refusal case. Introduce a separate explicitly attributed synthetic fixture rather than backfill historical records.

# Document basis

Retain immutable workspace-scoped document evidence binding exact request/candidate/digests and organization DocumentKey to a named synthetic profile/version. The closed initial profile explicitly declares rated positive individual charge amounts as document net amounts; this is selected fixture evidence, never a default interpretation. Each covered charge component requires an explicit product, positive quantity, quantity unit, tax code, tax amount and evidence identity. Require complete one-to-one coverage of the supported individually attributable charges, no duplicate/omitted source identities, exact currency and lossless scale. Aggregate adjustments, negative/zero charges, missing quantity/tax, implicit allocation, mixed currencies and unsupported periods refuse. Independently stated expected net/tax totals must reconcile exactly. A preparation may not claim that the original rated-total approval approved invoice gross or tax.

The fixture must carry explicit source and document evidence; construction is local synthetic assembly and does not grant users a tax-policy management capability. Later public ingestion/governance is a separate reviewed interface. Retain canonical serialized evidence and decode defensive copies; reuse of an identity with changed content refuses.

# ERP evidence and mutation

The adapter owns the synthetic ERP registry, connection, customer/product/tax mappings and survey identity. Add bounded read-only MappingOf and SurveyOf APIs to erpledger so preparation can compare actual retained current records without repeating Map/Survey mutations or accessing private state. Both return immutable value copies and distinguish missing from malformed/foreign keys. ConnectionOf already supplies current connection facts. These reads are a prerequisite for the adapter, not completion of this plan.

Under the shared server domain lock, authenticate the actual consumer, require current owned charge approval through currentFinancialChargeLocked, validate the explicit document basis/current organization attribution and compare the exact retained connection/mapping/survey evidence. Derive erpledger.Submission from those facts and call Prepare. The service keeps deterministic ownership and exact expected submission before the module mutation, retains the returned module digest, and reconciles an interrupted result only against that same owned submission and actor. Current prerequisite/evidence checks precede any historical Prepare replay. Source identity must survive new request/key attempts so a second preparation cannot evade module coverage protection.

Prepare retains a submission and source coverage; it does not send, post, consume an approval or certify an ERP final amount. Never call CommandSubmit, CommandPost, Execute, Report, Issue or delivery APIs in this increment. An eventual public action requires its own reviewed user/console authorization, revision, step-up, hold and idempotency contract; this internal fixture adapter does not supply it.

# Scope and acceptance

Root may edit new financialdocument/financialerpprepare files and tests, narrow synthetic assembly/Server state, erpledger evidence read APIs/tests and ticket records. No external dependencies, connections or customer data. Independently review concrete schema, monetary mapping, ownership/recovery and caller boundary before adapter implementation.

Verify actual retained ERP preparation and exact independently expected net/tax totals; missing quantity/tax; duplicate/omitted/aggregate charge coverage; stale approval/organization/mapping/survey; foreign binding; scale loss; immutable history; stable source duplicate protection; and recovery after module preparation before service receipt retention. Run module/server unit/race/vet and relevant full regression. No UI change is planned at this stage; keep screen acceptance distinct. Production tax policy, vendor capability verification, final ERP amount approval and subsequent effects remain unresolved.

# Reviewed evidence reads and quantity meaning

Independent refusal_review passed the MappingOf/SurveyOf contract and concrete source/tests. Both use the existing registry lock, validate canonical keys, return current value copies and distinguish malformed from missing/foreign keys. Collection counts and revisions remain unchanged; normal enter clock-watermark advancement remains possible. Full erpledger race suite passed 2.529s and module vet passed. These APIs are implemented prerequisites, not the assembled adapter.

Independent review also permits explicitly declared document units independent of normalized usage quantity for this named synthetic profile. Document quantity/unit never claim provider usage, recompute price or change the bound net amount. Preserve product/quantity/unit/evidence in the immutable basis digest and service provenance because erpledger.Line has no unit field. Different quantity/unit evidence cannot reuse a preparation identity. No default quantity, tax amount or backfill is allowed.

# Concrete adapter schema for review

Use a private synthetic ERP adapter with one actual erpledger.Registry, a bounded immutable document-evidence store and a bounded preparation-ownership store. A document evidence record contains ID, mode=synthetic, profile=synthetic_individual_net_v1, evidence reference, workspace, charge request/digest, candidate/digest, exact organization DocumentKey, connection, explicit direct ERP path, complete current connection/mapping/survey snapshots, expected net/tax and a line array. Each line contains source rating line ID, exact rule/kind identity, explicit product, document quantity/unit, tax code/amount and evidence reference. Canonical serialized bytes bind every field; retain/decode defensive copies and refuse conflicting IDs.

Only one positive individually attributable charge per input line is supported initially, with no omitted input lines, exclusions or aggregate adjustments. Match rule/kind/line, bind net directly to exact charge amount and independently reconcile net/tax totals. Product and document quantity are explicit profile evidence, never silently copied from a missing SKU/quantity. Stable ERP Source identities derive the economic financial target and original source RowKey, not request/candidate/preparation IDs or rule versions; prohibit ambiguous/duplicate original keys. This preserves duplicate-source protection if a request or candidate is recreated. Require exact external-invoice document attribution, currency and full calendar month, explicit direct path and lossless connection scale; do not infer an ERP route from language or billing profile.

Preparation owns a deterministic submission key plus immutable basis digest, exact expected module submission and actor before calling Prepare. Store a bounded pending ownership record first; on identical retry, repeat current charge/producer/document/connection/mapping/survey checks and reconcile only a matching SubmissionOf result with the same author. A different key may not create another live preparation of the same stable source coverage. A module success followed by missing service receipt remains recoverable under that owned pending record. No fallback to a foreign pre-existing module key. An internal fault hook may test the after-Prepare/before-receipt boundary; no external transport or automatic posting is involved.

Independent refusal_review passed the concrete schema with binding implementation conditions: require a bijection across input lines, normalized row origins, an explicit supported charge-kind allowlist and document lines; positivity does not make aggregate adjustments supported. Bound all text/counts/serialized evidence before arithmetic. Quantities must be positive; zero tax is valid only when explicitly supplied under this synthetic profile. Hash structured versioned workspace/economic-target/complete-RowKey identities, never row positions or preparation IDs. First ownership creation must reject any pre-existing module submission, even identical; only an already-owned pending record can reconcile the full matching submission/actor/evidence. Lookup errors are not absence. Preflight capacity and representation before ownership/effect creation; retain uncertain pending ownership and recheck all current evidence before every replay. Survey comparison uses the full current view/author/time and cannot claim detection of every intervening rewrite. Concrete implementation still requires independent review.
