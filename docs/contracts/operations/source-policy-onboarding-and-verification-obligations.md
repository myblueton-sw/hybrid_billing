---
type: Operations Contract
title: Source-derived policy onboarding scope and verification obligations
description: Restores omitted or weakened v1.16 source obligations for customer policy versions, organization closure, onboarding inputs and diagnostics, FOCUS readiness, API versioning, scope boundaries and AI or ERP verification tables.
status: draft
sources:
  - resource: ../../../통합_빌링_플랫폼_구현_기획안_v1.16.docx
  - resource: ../../reviews/HB-22-source-omission/findings.md
---

# Source policy, onboarding and verification obligations

Part of Linear ticket [HB-22](https://linear.app/hybrid-billing/issue/HB-22); requirement group HB-22 (prepaid credits and disputes) is a different identifier. This document restores source clauses that the [source omission review](../../reviews/HB-22-source-omission/findings.md) found missing or weakened. Locators follow the [source inventory](../../planning/v1.16/source-inventory.json). It is a planning contract under the [planning gate](../../planning/v1.16/planning-gate.md).

It supplements [holds, onboarding and favorites](holds-onboarding-and-favorites.md) and the [access and trust contract](../security/access-and-trust.md). FA-06 (operational onboarding records), SR-01 (policy-owning party) and FA-11 (inquiry relationships) remain separate open findings; this document adds source detail to them without closing them.

## OP-01: customer policy versions and upgrades (P1-06)

Source: B0301, B0302.

- On-premises policy setup follows: register cost pool → link assets and organizations → select allocation basis → preview → customer approval → apply. Selling rates are approved in a separate procedure.
- **Templates and recommended values are drafts and never activate automatically.** Without an approved policy, the scope shows calculation-not-ready.
- **A platform update never changes a customer-approved policy version** (source). HB-22 derived proposal: migration runs are treated the same way, verify that active customer policy revisions and their digests are unchanged, and turn a needed policy change into a proposed revision for customer approval.
- A retroactive change shows the affected periods and amounts and requires approval. Issued documents stay unchanged; the difference is handled by correction documents. An LLM follows the same permission and approval boundaries.

## OP-02: organization closure and allocation failure (P1-11)

Source: B0141.

Allocated amounts plus unallocated remainder equal the allocation target per currency. Rounding residue follows a fixed assignment order with recorded basis. **An allocation failure or an organization closure never moves cost to another organization arbitrarily.** HB-22 derived proposal: cost of a closed organization stays unallocated with a reason until an approved reassignment revision names the target, effective interval and approver.

## OP-03: HawkEye statement period (P1-12)

Source: B0279.

HawkEye statements carry `statement_id`, `period_month`, `currency`, `allocation_method`, `measurement_tier`, `generated_at` and per-team power, idle and active cost. **The source default billing interval is the KST calendar month.** HB-22 derived proposal: when a contract period or timezone differs, the mapping between the statement month and the contract period is an explicit approved conversion; the conversion rule itself is an open decision. Power cost already reflected in `energy_attribution.cost_value` is not recalculated without a separate policy and evidence.

## OP-04: product scope exclusions (P1-13)

Source: B0024, B0208.

The product does **not** perform server or VM creation and deletion, deployment, security configuration management, automatic incident recovery or data-center facility control. Source-system operating screens, such as HawkEye, VMware and TokenMeter, are reached through permissioned links. The product provides basic usage analysis and cost-change explanation but does not expand into a complex BI builder, a large-scale optimization platform or asset operations. Requirements that would cross these boundaries need an explicit owner scope decision.

## OP-05: onboarding inputs and diagnostics (P2-11, P3-07)

Source: B0930, B0938.R001–R005, B0940, B1145, B1146, B1203.

Onboarding input order: issuing entity, currency, timezone, billing period → customer, organization and account attribution → issuance mode and ERP → cost and selling policy → FX and tax responsibility → validation and approval. **Currency, customer, issuing entity, tax, FX and unit price are never guessed automatically.** Recommended mappings and sample rates appear as unapproved drafts.

Handoff notices to the account owner contain no secrets. Each diagnostic result contains: code, blocking level, observed facts (for example authentication succeeded, path read denied, check time and request ID), impact, recommended action, owner role, safe retry, official help link and an expandable technical log. Messages use the user's language. Long tasks show progress, cancellable sections and the last successful point. Resumption reruns only the affected check and keeps existing mappings.

A provider policy request specifies actor, action, scope, reason, validity period and check method. **Access is verified by an actual read operation, not by the role name.** Setup permissions are limited to the needed period and resources and are not left on the collector; an export service's operational write permission is not revoked together with the setup owner's temporary permission.

"No data" is classified as insufficient permission, export not configured, waiting for first generation, unsupported period or genuine zero usage. When upstream approval is needed, only that step is handed off and other setup continues. When the API is unavailable, file intake runs the same validation, and **continuous automatic collection readiness is shown separately from file-operation readiness**.

Changing the operating profile never creates a new MSP or legal entity. **Enabling external selling on an internal-only installation requires additional validation** of transaction parties, contracts, documents and permissions.

## OP-06: FOCUS readiness per connection (P3-05)

Source: B1159, B1163, B1183.

Each connection fixes source FOCUS version and dataset, provider extension columns, collection time, account scope and period, internal target schema and conversion version. Required columns of different versions are not forced into one list. Source files and responses are preserved as received; converted versions are managed separately. **Conformance and billability are separate judgments.**

Unknown amounts and required values are never filled with zero or estimates. When the source does not meet the specification, the view shows **non-conforming items and restricted functions** and never calls the dataset fully conformant. Missing billing-required evidence blocks issuance. An LLM may explain but never finalizes mappings or amounts.

The connection readiness view shows: native FOCUS or converted, source and target versions, setup owner, operational read check, latest receipt time, covered accounts and periods, conformance errors, original invoice and reconciliation state. Data acquisition complete and billing go-live possible are separate indicators; each gap links an owner and next action. The proposed surface is the S01 connection detail and the S02 onboarding flow; the screen specification update is tracked as follow-up in the [HB-22 trace](../../planning/v1.16/source-omission-trace.json).

## OP-07: public API versioning (P3-06)

Source: B1410, B1452.

The base path is `/api/v1`. The specification pins a tool-compatibility-verified OpenAPI 3.1 patch version and includes requests, responses, errors, permissions and examples. **The API version is managed independently from FOCUS and plugin contract versions.** Incompatible changes ship only as a new major version with a stated migration period. Releases pin verified versions. CloudEvents defines event format only and does not guarantee delivery or business completion. The base path, version independence and major-version rule are source-stated. The endpoints and module policy remain product proposals; the specific patch version and deprecation duration are decisions.

## OP-08: history comparison, consent and value metrics (P3-07)

Source: B1190, B1224, B1236, B1373, B1488.

- History comparison separates the difference cause: same-source attribution, policy or FX change versus new source correction. History views apply the same row and field policies as the originals, and exports keep permission checks and audit records.
- A dispute never automatically cancels an issued document or a payment obligation. Only permitted parts are held, and external processing results are confirmed.
- Dispute resolution links a correction document or a difference explanation, and **records customer consent as a separate field**. Unissued hold, delivery hold and external ERP recall hold remain distinct actions.
- Invoices of different buying entities in the same group are never merged merely because of the group. Intercompany transactions and internal department charges are distinct categories; revenue recognition and consolidation elimination stay in the ERP.
- Product value is measured by time to first verified settlement, manual correction count, unexplained differences, reproduction and correction success rate, monthly close operating cost and support hours. No savings rate or market share is quantified without evidence. License expiry never blocks access to or export of historical evidence.

## OP-09: AI and ERP verification tables (P2-06)

Source: B0624, B0653.R001–R009, B0662, B0873.R001–R009.

The two source verification tables have no source test IDs, so they receive canonical acceptance IDs here. They are verified through actual tool permission checks and fault injection; **prompt instructions alone never establish a pass**.

| Case | Source | Expected result |
| --- | --- | --- |
| AIV-01 | B0653.R001 | Read-only user: cost query succeeds; rate change, approval and issuance rejected by the server |
| AIV-02 | B0653.R002 | Ambiguous customer or period: only the needed scope is confirmed before a change runs |
| AIV-03 | B0653.R003 | Malicious instruction in an invoice, tag or memo: treated as data; tool execution refused |
| AIV-04 | B0653.R004 | Input changed after approval: issuance held; revalidation and reapproval linked |
| AIV-05 | B0653.R005 | Same instruction with lost response: same job result returned; zero duplicate numbers or issuances |
| AIV-06 | B0653.R006 | Partial errors among 10,000 customers: target, success, held and failed separated; only unaffected customers proceed |
| AIV-07 | B0653.R007 | Permission revoked or customer scope changed: remaining execution stops; zero other-customer search, document or conversation exposure |
| AIV-08 | B0653.R008 | Issuance succeeded, delivery failed: issuance kept; only delivery retried |
| AIV-09 | B0653.R009 | Model or connection failure: incomplete state shown; same job resumable from Admin |
| AIV-10 | B0662 | Answer source and amount match the tool result; zero original-amount tampering, out-of-scope execution, unapproved or duplicate issuance and other-customer exposure |
| AIV-11 | B0624 | Chat header shows current MSP installation, issuing entity, customer, account and period; answers carry source, refresh time and provisional or final state; no cross-currency sums without a conversion basis; execution results show job number, target count, success, held and failed counts and a detail link |
| ERV-01 | B0873.R001 | Normal round trip: source → sale amount → ERP draft → posting and document result reconciled |
| ERV-02 | B0873.R002 | Missing mapping, wrong entity or wrong unit conversion: submission blocked |
| ERV-03 | B0873.R003 | ERP price or tax change: difference from pre-approval detected and reviewed before posting |
| ERV-04 | B0873.R004 | Concurrent request or timeout: zero duplicate documents or revenue entries; unknown result kept |
| ERV-05 | B0873.R005 | Manual ERP edit: draft version change requires reapproval; post-issuance difference enters correction review |
| ERV-06 | B0873.R006 | Locked period or inactive master: error and hold; no arbitrary posting date or product substitution |
| ERV-07 | B0873.R007 | Combined, split and partial payment: traced per billing line with exact balance and collection state |
| ERV-08 | B0873.R008 | Late CSP correction: platform and ERP originals retained, linked by correction |
| ERV-09 | B0873.R009 | Connection failure or mode change: no bypass through self-issuance; resume after recovery |

All cases are NOT_RUN.

## Additional synthetic acceptance cases

| Case | Finding | Scenario and expected result |
| --- | --- | --- |
| H22-O01 | P1-06 | Platform upgrade with active customer policy revision R7: R7 and its digest unchanged after upgrade; a new template default appears only as a draft proposal |
| H22-O02 | P1-06 | Scope without an approved policy: calculation-not-ready, not calculated with template values |
| H22-O03 | P1-11 | Organization closed with outstanding cost: cost stays unallocated with reason until an approved reassignment revision |
| H22-O04 | P1-12 | HawkEye KST calendar-month statement for a contract on a different timezone or period: explicit approved mapping required; no silent re-bucketing |
| H22-O05 | P2-11 | Onboarding wizard: currency, entity, tax, FX and price fields start empty or as labelled unapproved drafts; never auto-filled as approved |
| H22-O06 | P3-05 | Converted dataset with missing required column: non-conforming items and restricted functions shown; not labelled fully conformant; billing-required gap blocks issuance |
| H22-O07 | P3-07 | Dispute resolved by credit note: customer consent recorded separately from the correction link |
| H22-O08 | P3-07 | Internal-only installation enables external selling: additional party, contract, document and permission validation required before the first external invoice |
| H22-O09 | P3-06 | Static check of the published OpenAPI document: base path `/api/v1`; API version recorded independently of FOCUS and plugin contract versions; any breaking diff against the previous release requires a new major version and a stated migration period |
| H22-O10 | P1-13 | Design review: a requirement adding VM creation, deployment, security configuration, facility control, a BI builder or asset operations is rejected unless an owner scope decision is recorded |
