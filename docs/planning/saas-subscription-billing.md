---
type: Planning Supplement
title: SaaS subscription and hybrid billing management
description: Adds future SaaS subscription management to the existing billing scope with lifecycle, pricing, authority, screen and acceptance proposals.
status: draft
---

# SaaS subscription and billing planning

Tracking: [HB-18](https://linear.app/hybrid-billing/issue/HB-18/plan-saas-subscription-and-hybrid-billing-management). Owner request: 2026-10-06. Accountable owner: Seung Woo Park. Root owns PM/planning and document integration; an independent reviewer checks billing, tenant boundaries and planned acceptance. This is a future capability in planning, not implemented behavior or a declaration that planning is complete.

## Scope and provenance

The owner asked to include future SaaS service subscription and billing management. The working interpretation is **SaaS vendors managing customer subscriptions and charges**, combining recurring fees, seats and metered usage. That interpretation and the detailed behaviors below are planning proposals; the owner requested the capability, not every proposed policy value or launch sequence.

An enterprise's inventory of purchased SaaS subscriptions, license optimization and renewal procurement is a separate possible extension. Offering Hybrid Billing itself as a hosted service is also a separate delivery decision; existing self-hosted scope remains. No additional provider, payment processor, tax jurisdiction or ERP compatibility is certified. All existing provider families and conditional/later requirements remain included.

RA-15 and SB references are editorial planning identifiers, not new source requirement groups or Linear ticket numbers. Preserve the original 43 source groups and 49 screen identities. The published [reconciliation index](v1.16/planning-reconciliation.json) supplies overlap evidence, but the detailed local baseline is unavailable in this checkout; full canonical integration remains pending. Missing local rules and planning inputs are bypassed for this bounded documentation task by the owner's explicit instruction, not reconstructed or declared restored.

## Proposed capabilities and required policy fields

| Ref | Capability | Proposed contract and unresolved activation fields |
| --- | --- | --- |
| SB-01 | Product, plan and add-on catalog | Version products, prices, billable units, currency, monthly/annual interval, included allowance, tiers and effective dates. A subscription binds an approved contract and price revision; edits do not rewrite historical charges. Define grandfathering/migration and customer communication rules before activation. |
| SB-02 | Customer subscriptions | Support multiple subscriptions per authorized customer/contract with service-instance references, trial, start, renewal, pause/resume, scheduled cancellation and end. Define supported transitions and approval, renewal notice and term policies. A customer, payer and tenant are not interchangeable. |
| SB-03 | Changes and proration | Preview before/after price, remaining term, quantities, credit and effective date for upgrades, downgrades, seat/add-on changes and cadence changes. Specify immediate versus next-period treatment, denominator, time zone, billing anchor, advance/arrears and rounding; missing policies block monetary finalization. |
| SB-04 | Recurring, seat and usage charges | Combine fixed fees, billable seat history and eligible metering with allowances, tiers, minimums and overages. Define event versus cumulative quantity, seat counting rule, measurement window, late/corrected usage and complete shared-pool coverage. Do not charge both a cumulative snapshot and its component events. |
| SB-05 | Billing and adjustments | Feed the existing calculation, reconciliation, human approval, Claim, issuance and correction workflow. Separate discount, included usage, promotional credit, prepaid cash, credit document and cash refund. Every adjustment links to its original scope and approved policy. |
| SB-06 | Collection and payment evidence | Plan processor adapters, payment-method references, authorized payment attempts, provider events, retries and reconciliation. Automated debit is conditional on a separately approved payment/mandate/security contract; no processor is selected. Preserve one receivable/cash authority and manual evidence paths. |
| SB-07 | Service entitlements | Bind feature/seat/quota proposals to effective subscription revisions and explicit service policy. Separate service provisioning state, payment state and login/authorization. Define grace, restriction, suspension and reactivation; late/failed payment does not itself authorize data deletion or revoke all retained-document access. |
| SB-08 | Customer portal | Show authorized subscription, price revision, renewal/cancellation dates, included/used/overage quantities, preview versus issued charges, receipts/as-of and permitted change requests. Changes use common server commands; customer requests are not financial approval. |
| SB-09 | Finance and reporting | Keep billing cadence, Payment Terms, cash settlement and monthly ERP cost recognition independent. Define per-currency MRR/ARR, churn, unpaid balance and renewal metrics with explicit inclusion rules; operational metrics are not accounting revenue. SaaS revenue recognition requires separate approved accounting design. |
| SB-10 | Operations and audit | Track change schedules, failed/unknown jobs, payment-provider discrepancies, expiring trials and entitlement-sync lag. Preserve effective/recorded/confirmed times, evidence, actor and current permissions. Notifications and retries recheck state and holds before dispatch. |

## Ownership and state boundaries

Extend the existing [logical model](../architecture/data-ownership/billing-data-model.md); do not introduce a second customer, financial ledger or tenant model. Proposed logical families are Catalog/PriceRevision, Subscription/SubscriptionRevision, SubscriptionItem, ChangeRequest, MeterDefinition/UsageManifest, EntitlementRevision/Delivery and PaymentAttempt/ProviderEvent. These are design vocabulary, not approved table names or schema.

Every reference binds installation/workspace, customer relationship, contract and relevant service instance. A shared payer or provider customer ID grants no cross-customer access. Consolidating invoices requires the same authorized issuer/bill-to/currency/basis and explicit aggregation policy; grouping subscriptions cannot reapply allowances or duplicate economic Claims. Transfers of payer, workspace or customer are governed migrations with historical liability and grant decisions, never ID reassignment.

| State dimension | Proposed progression and authority |
| --- | --- |
| Subscription contract | Draft → trialing or active → ended; pause/resume and scheduled changes are explicit effective-dated transitions. Trial conversion needs agreed commercial terms and required approval. Scheduled cancellation remains visible while service stays active until its effective end. |
| Calculation and document | Estimated → calculated/reconciled → human-approved fixed snapshot → separately authorized issuance → authoritative confirmation. Subscription renewal jobs cannot bypass existing financial approval. |
| Payment attempt | Prepared → authorized dispatch → pending/unknown → confirmed success/failure; later reversal/dispute remains linked evidence. Provider event arrival alone is not sufficient settlement proof. |
| Receivable | Not applicable, unconfirmed, outstanding, partially settled, settled or overdue/unknown under the existing payment contract. Attempt success must be reconciled and allocated to the correct invoice before settling its balance. |
| Service entitlement | Desired revision → delivery pending → confirmed applied or failed/unknown. Actual product access uses the approved service policy and current authorization; unknown delivery must not be displayed as applied. |

Validate expected current revision at every change. Concurrent cancellation, renewal and plan changes require one ordered, auditable outcome; conflicting/stale requests are rejected for refreshed preview. An unsupported transition remains blocked. Ending service does not cancel supplier resources, erase receivables or destroy retained evidence. Cancellation reversal is a new authorized transition, not resurrection of an old command.

## Pricing and financial conservation

Apply [PC-01](../contracts/billing/inputs-pricing-and-erp.md): quantity/target → base selling price → contract discount → fees → minimum/maximum → FX → selling tax/rounding. Subscription proration and seat rules must specify which quantity/component is affected, how full coverage is determined and which approved sequence applies. Changing ordering requires a reviewed revision, not a UI toggle with silent retrospective effects.

Record interval `[from,to)`, service time zone/calendar, billing anchor, advance/arrears basis, unit, meter revision and price/contract revision. Define short months, month-end anchors, leap days, trials, pauses, mid-period changes and annual-to-monthly transitions explicitly. Seat pricing must select assigned, active, peak or time-weighted semantics; login count cannot silently substitute. Allowance reset, pooling and rollover are contract fields, not implicit monthly defaults.

Corrections retain economic period and original line linkage. An already consumed recurring or usage component cannot become chargeable again because a job, subscription revision or request key changes. Refundable credit cannot exceed eligible unrefunded value under the approved policy; pending external refunds/credit applications reserve capacity or block automation when external reservation is unsupported. Refund requests, credit notes and provider-confirmed cash refunds are different states. Disputed payments follow separate hold policy; do not silently convert them into zero charges.

Synthetic oracle, not an adopted price: a full-period fixed fee 100 plus 3 billable seats at 10 plus 120 usage units with 100 included at 2 yields `100 + 3*10 + (120-100)*2 = 170`, before tax/FX, with no other modifiers. Splitting this dataset into two jobs must still yield 170. Under a specifically chosen day-based 30-day policy, an upgrade adding 60 for exactly 15 remaining days contributes `60*15/30 = 30`; this example does not select a universal proration rule.

Annual billing must not imply dividing every amount by twelve for accounting. Preserve the existing [Payment Terms and monthly ERP cost recognition contract](../contracts/billing/payment-and-recognition.md); distinguish whose books, transaction purpose, amount basis and period apply. Customer subscription revenue, deferred revenue and tax schedules require an approved separate mapping. Where monthly cost recognition and customer billing coexist, link actual source periods and approved adjustments without double recognition.

## Payment, entitlement and security failure handling

For a future payment adapter, bind configured provider account, workspace, customer, invoice, attempt and currency/amount; verify inbound event signatures and replay bounds, deduplicate event identities and verify authoritative provider state. Reject wrong-account/wrong-currency/wrong-amount events and quarantine conflicting payloads. Event times and arrival times are distinct. Duplicate/out-of-order callbacks cannot revive superseded states or mark an unrelated invoice paid. Provider payout evidence is not customer invoice settlement.

Persist dispatch intent before an external attempt. Timeout is unknown; look up the original attempt before retrying. Fail closed on unsupported lookup/idempotency/authorization or stale authority, leaving a controlled reconciliation path. No blind retry, automatic charge without mandate, payout, bank ledger or direct ERP database write is introduced. Store server-held secret references and opaque payment-method references; raw card/security-code data must not enter product documents, logs, exports or browser storage.

Payment-policy approval, customer consent/mandate, current action authority and original fixed charge approval are separate prerequisites. Choose one invoice issuer and financial authority for each path; provider recurring billing cannot issue a second invoice or bypass platform human approval. Cash refunds require separate authorized execution and confirmed provider outcome; unknown refund results require lookup before any new dispatch. Chargebacks/reversals update linked receipt evidence without silently rewriting issued amounts. Notifications and collection retries respect collection/delivery/dispute holds as applicable; a hold does not erase the debt. Payment-method changes require verified scoped commands and audit; they cannot alter the authoritative invoice amount or tax/payment-recipient records implicitly.

Entitlement delivery uses scoped service identities, versioned desired state, idempotent commands and authoritative acknowledgement. Old callbacks cannot regrant access after a later suspension/cancellation. Reconcile target-system state after outage/restore and fence stale workers; no financial refund or invoice cancellation is inferred from service delivery failure. Define compensation/manual recovery and any grace allowance explicitly before activating the integration.

All portal, query, cache, export, job and notification paths follow the [access contract](../contracts/security/access-and-trust.md). A paid feature grant never grants access to another tenant or permission to approve charges. After service expiry, retained billing evidence remains available only through current authorized disclosure/retrieval policy; ordinary revocation still takes effect. Retention/hold/deletion duties remain independent of subscription/payment state.

## Screen and workflow integration proposal

These are proposed extensions of published screen references, not replacement specifications or additional screen IDs. Complete field/action/API/permission trace requires the original screen baseline.

| Existing screen references | Planned flow and required exceptional states |
| --- | --- |
| O03, B04 | Select customer/contract → plan/version/add-ons → term and seat/meter rules → change preview → request approval. Show unsupported policy, overlapping terms, stale revision and effective-date conflicts. |
| B08, B09 | Review full subscription-period coverage, charge components and independent totals → reconcile → human approval. Show missing usage/seat history, exhausted Claims, expired quote and changed price/input invalidation. |
| B10, B11 | Linked credits/corrections/refund requests, disputes and holds. Show reserved versus available amount and unknown external outcome; no implied cash refund from a credit note. |
| I01, I02, I03 | Issued document, due date, receipt allocation and purpose-specific ERP/payment evidence; manual handoff remains distinct from confirmed external application. |
| C01, C02, C03 | Own subscription/usage/document detail and scoped change/cancellation/inquiry requests. Preview versus confirmed amount, inaccessible data and pending entitlement delivery are explicit. |
| U01, U02, U03, R01, R02 | Renewal/trial/change tasks, collection/entitlement exceptions, permission-safe notification, per-currency metrics and audit. Show as-of/unknown, owner and next action. |

Every sensitive action previews scope/effective date/amount and approval needs, then checks current authority and expected revision at execution. Loading, empty, denied, partial, stale, unknown, failed and confirmed states remain distinct. No executable prototype, new API implementation or user-task validation is delivered here.

## Proposed overlap and unresolved decisions

Source HB identifiers in this table identify requirement groups; the task is Linear HB-18, which is unrelated to source HB-18 FX/rounding.

| Planning area | Existing source groups | Decision packets / remaining work |
| --- | --- | --- |
| Customer/subscription lifecycle | HB-01/02/03/15/42 | D-01/06/07: customer/payer/service ownership, transitions, roles and entitlement policy |
| Catalog, seats, metering and pricing | HB-06/08/15/16/17/18/21/38 | D-02/03: units, revision/Claim keys, proration, allowances, sequence, precision and independent expected values |
| Payments, corrections and ERP | HB-21/22/23/24/25/26/28/29/31 | D-02/03/07: mandate, payment authority, provider evidence, refund/hold policy, tax and finance mappings |
| Portal, events and retained evidence | HB-27/30/33/34/37/39/42/43 | D-04/05/06/07: permission/actions, service acknowledgement, recovery, storage and measured operational budgets |

Use [D-08](v1.16/planning-decisions.md) as the SaaS extension packet. Decide supported product behavior separately from customer-specific configuration and external compatibility evidence. Documentation covers the whole requested capability; delivery order does not exclude it. A proposed dependency sequence is: reviewed lifecycle/catalog/metering contracts → calculation and controlled invoice/ERP handoff → entitlement and portal flows → certified collection adapters and recovery. This is not an approved release commitment; deployment, staffing, timing and automatic collection scope remain open.

MRR/ARR/churn definitions must name eligible subscriptions, committed versus variable usage, trials, paused/canceled timing, discounts/credits, annual normalization, currency and as-of. Do not publish a single mixed-currency total, equate MRR with recognized revenue or infer churn from a failed payment alone.

## Planned acceptance

All SB-A cases are **NOT_RUN**. They are future acceptance obligations, not proof of implementation. Structural document checks and independent review are recorded separately in the [verification report](../reviews/HB-18-saas-subscription/verification.md).

| Case | Independent expected result |
| --- | --- |
| SB-A01 | Repeated/concurrent renewal jobs consume each eligible subscription-period-component once, including restarts with new request keys. |
| SB-A02 | Full-period fixed/seat/usage example totals 170; chunking, overlapping pools and replay cannot reapply the 100-unit allowance or change the result. |
| SB-A03 | Under the synthetic 30-day policy, the 15-day upgrade delta is 30; missing denominator/time-zone/rounding policy blocks finalization. |
| SB-A04 | A new price revision leaves issued history fixed; changed draft input or approval digest requires fresh calculation/review. |
| SB-A05 | Scheduled cancel versus concurrent renewal/change has one governed result; pause/trial/expiry preserve liability and required financial evidence. |
| SB-A06 | Missing seat history or late/partial meter coverage is unknown, not zero; cumulative-plus-event duplicate evidence cannot double charge. |
| SB-A07 | Duplicate/out-of-order/wrong-account payment callbacks cannot create receipts or regress state; timeout triggers lookup before retry. |
| SB-A08 | A confirmed payment for the wrong invoice/currency/amount does not settle the target; unallocated cash and partial receipts follow one cash authority. |
| SB-A09 | Concurrent refund/credit attempts cannot exceed the eligible balance; credit documents and pending/refused refunds are not confirmed cash movement. |
| SB-A10 | Collection/dispute/delivery holds retain their separate effects; retry/notification rechecks authority, hold and current receipt evidence. |
| SB-A11 | Payment failure alone does not delete data; entitlement delivery failure is visible; stale callbacks and restore cannot regrant revoked service access. |
| SB-A12 | A customer cannot enumerate another customer's subscription/usage/invoice or infer hidden supplier/margin data through search/export/notifications. |
| SB-A13 | Paid-plan feature grants do not grant financial approval or tenant access; revoked users cannot retrieve historical invoices through old export links. |
| SB-A14 | Annual billing, Payment Terms and monthly cost recognition stay distinct; source-derived recognition uses approved mappings and avoids duplicate accounting. No unapproved SaaS revenue schedule is inferred. |
| SB-A15 | Trial, month-end/leap-day, cadence-change and seat-change boundaries use the chosen policy and preserve all before/after revisions; stale previews cannot execute. |
| SB-A16 | Metric totals reproduce the same approved definition/scope/currency/as-of; variable usage, annual normalization and partial payment are not silently classified as recognized revenue or churn. |

Planning closure requires review of D-08, canonical requirement/work/API/screen/acceptance integration, explicit remaining conditions and the owner's separate planning acceptance. No monetary policy, payment mandate, provider support, runtime test or development approval is certified by this supplement.
