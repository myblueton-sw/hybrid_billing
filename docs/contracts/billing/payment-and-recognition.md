---
type: Billing Contract
title: Payment terms overdue and independent monthly ERP recognition
description: Carries existing owner additions into English while separating required lifecycle semantics from proposed date calculation and unresolved accounting profiles.
status: draft
---

# Payment and recognition contract

Tracking: [HB-14](https://linear.app/hybrid-billing/issue/HB-14). This English supplement preserves the authorized local v1.16 payment-terms.md and erp-recognition-cadence.md owner additions, rather than attributing them retroactively to the original DOCX. Source hashes and exact existing requirement/work/test mappings are in the [reconciliation index](../../planning/v1.16/planning-reconciliation.json). These are design contracts; runtime and ERP compatibility are NOT_RUN.

## Payment terms: required outcome and proposed calculation

Required: manage contract-specific payment within N days from invoice issuance and show due dates/overdue status in billing. Net 30 and Net 45 are examples, not universal defaults. Map this addition to source groups HB-15/22/24/27/28/29/30/31.

A proposed contract revision records term ID/revision/name, invoice-issue-date basis, nonnegative integer net days, calendar/business-day mode, contract time zone, holiday/calendar revision, effective dates, reason and approval. Calendar days are the initial design proposal, not an owner-approved global rule. Business-day/holiday adjustment requires an approved calendar and computation rule. Month-end terms, installments and early-payment discounts remain subsequent contracts outside this addition's required scope.

At issuance, preserve the confirmed issue date, contract/term revision, original and current effective due date, and applied time zone/calendar in the document snapshot. Changed contract terms do not recalculate issued history. An unissued/undated draft has an estimated date, never confirmed overdue status; internal allocation without payment obligation is not applicable. Missing or contradictory required terms block issuance.

Due-date extensions require separate current authority, approval, reason and effectivity; preserve original due date and past overdue evidence. Redispatch/download does not restart the clock. Cancellations, replacement documents and corrections remain linked; explicitly decide due-date inheritance/recalculation rather than treating every correction date as a new issue date.

Proposed calendar calculation: confirmed issue date in contract time zone plus net days, with issue date as day 0 and payment on the due date still timely until that date ends. The next local date starts overdue eligibility. Browser/UTC dates do not replace contract dates. Synthetic examples: 2026-10-01 plus 30 days is 2026-10-31; plus 45 days is 2026-11-15. These are date examples, not legal deadlines or adopted customer policy.

## Receivables, settlement and overdue authority

Outstanding is the authoritative confirmed receivable less confirmed allocated receipts/credits/cancellations/corrections, including linked reversals exactly once, in exact per-currency amounts. Unallocated cash, payment promises, transfer claims and document delivery/download are not settlement. Overpayment is a separate credit/unallocated balance, not negative overdue.

Overdue requires a passed effective due date, positive valid outstanding and sufficiently current authoritative receipt evidence. Partial payment leaves only the remaining balance overdue. Distinguish due today, not due, overdue, settled, not applicable and unknown. Calendar-day aging uses the contract time zone and handles midnight, leap-year and DST boundaries. Dispute and collections hold do not erase payment obligations or overdue facts. This requirement does not authorize late fees, interest or automatic debit.

Choose one receivable/cash authority per approved contract/installation. ERP synchronization binds document identity, entity, currency, revision and as-of time. A due-date disagreement is a reconciliation finding, never silent overwrite. Manual receipt confirmation requires explicit evidence, authority, audit and approval policy; it does not create an unauthorized second cash ledger.

Missing/stale receipt state produces payment-confirmation-needed / overdue-unknown. Last known balance, elapsed days and timestamp can be shown separately from confirmed overdue totals. Preserve economic receipt time and system arrival time. Late receipts/reversals revise the current result without erasing what prior notices actually knew. A reversed receipt reevaluates against the current approved effective due date; it cannot silently reset issue date/term or cancel an approved extension.

## Monthly ERP cost recognition: required outcome

Required: when customer billing is quarterly, transmit eligible monthly amounts to ERP for monthly cost recognition and reconcile the quarterly billing amount with the monthly amounts already sent. Map to source groups HB-15/21/23/24/25/26/28/29/30/31.

Usage/attribution period, ERP recognition period/posting date, customer billing cadence/issue date and payment due date are separate axes. Monthly recognition is not invoice issuance, receivable confirmation, delivery, due-date commencement or settlement. Payment Terms start from the applicable invoice's confirmed issue date.

Identify whose entity/books and transaction are recognized. Purchasing-entity expense/accrual, selling-entity revenue/unbilled balance and internal allocation are not the same journal. This requirement concerns cost recognition; revenue recognition, accounts, debit/credit and tax timing need approved finance/ERP mappings, not an inferred UI default.

Proposed contract configuration includes billing and monthly recognition cadence, fiscal period anchors/boundaries/time zone, entity/books/transaction path, amount basis, currency/tax/FX, provisional-input policy, approval and effective revision. Different monthly and quarterly amount bases are not automatically comparable. Handle partial months, start/end contracts and non-calendar fiscal quarters explicitly. Missing months are unknown, not zero.

A fixed RecognitionBatch binds workspace/entity/contract, recognition and attribution periods, source/calculation snapshots, row lineage, basis/currency/amount and approval/hash. Purpose-specific ERP submissions reference the batch revision, target books, idempotency key and external document/posting status/as-of. monthly_recognition and customer_invoice use distinct purposes within the existing submission ledger and are linked without creating another cash authority.

## Submission, comparison and duplicate recognition

Monthly path: eligible monthly data → validation/reconciliation → fixed batch → authorized approval → recognition submission → authoritative result lookup/reconciliation. It does not wait for the quarterly issue date. Receipt/transport success differs from actual posting. Unknown/timeout results require lookup before retry; partial failure resumes only unresolved work.

Use an official API or verified file workflow, never direct ERP DB writes. An unsupported recognition journal cannot be replaced by the invoice API. File download is not posting proof. Actual document types/reversals/lookup/deduplication need profile/version evidence.

Before quarterly reconciliation, gather all required monthly batch revisions, submission/posting states and linked adjustments/reversals. Compare only matching entity, contract, attribution scope, basis, currency and tax basis:

Quarterly comparable amount = current linked monthly recognized net amounts + approved linked adjustments not already included in those net amounts.

Net includes the original plus confirmed amendments minus confirmed cancellations/reversals, once. Adjustment identity, revision and lineage establish whether it is already in net. Approved but unconfirmed posting is only an expected comparison, not final reconciliation. Missing or unknown monthly posting prevents a completed comparison even when arithmetic difference is zero. Reconcile pre-tax/tax/post-tax separately with exact transaction-currency amounts; show display rounding separately. Functional-currency FX differences need separately approved linked FX adjustments; never add currencies directly.

Synthetic same-basis example: 100 + 120 + 80 = 300. A quarterly discount resulting in 290 requires an explained approved -10 adjustment and external handling/reconciliation evidence. If current monthly net already equals 290 after that adjustment, the additional bridge adjustment is zero, not another -10. These are synthetic expectations, not accounting entries.

Quarterly tiers/discounts/benefits, late evidence and credits can produce legitimate differences. Record cause, affected months, before/after amounts, approval and external state. Do not invent a plug balance or overwrite history. Rounding differences also need the approved policy and explanation.

Distinguish collecting, monthly posting confirmation needed, amount difference, adjustment approval pending, adjustment posting confirmation needed and reconciled. Changed hashes/months/external states invalidate affected review. Unresolved amounts/external states block final completion of that reconciliation path. Whether customer issuance must wait is an explicit approved profile gate; a permitted issuance exception cannot hide unresolved reconciliation.

Before automatic quarterly submission, select and validate the ERP-specific offset/replacement/reversal-or-reference mechanism that avoids recognizing monthly costs again. Distinct idempotency keys alone do not prevent double accounting. Undefined/unsupported treatment blocks automatic final quarterly submission; verify actual external documents and net balances.

Closed-period correction uses an approved open-period adjustment/reversal linked to the original attribution month; never overwrite the original monthly batch. Invoice cancellation does not automatically imply recognition cancellation. Re-read manual ERP changes and partial/interrupted adjustments; never modify foreign-owned journals.

## Screens, disclosure and bounded operation

These bindings preserve the source design; complete canonical screen/API/work integration remains pending. O03 shows terms and billing/recognition cadences. I01 shows term snapshot, original/effective due date, balance/as-of/aging and recognition linkage. I02 separates recognition batches, external journals, quarterly comparison and receipt synchronization. I03 distinguishes file purpose/period from confirmed posting. B08/B09 review quarterly result, monthly net and adjustments. B11 separates credit effects, disputes, overdue and holds. U02 keeps monthly and quarterly deadlines separate.

U01/R01 show per-currency balances and aging (not due, due today, 1–30, 31–60, 61–90, 91+ and unknown separately), monthly/quarterly comparison and owner/next action. U03 rechecks receipt, due-date, current permissions, disclosure, recipient and hold state immediately before notification. C01/C02 expose only the customer's permitted documents/terms/balance/as-of/inquiry path; internal journals/unbilled recognition/internal adjustments, other customers and hidden supplier/margin fields are not automatically disclosed.

Use indexed entity/customer/currency/due-date/balance/status projections, fixed snapshots and incrementally updated recognition links. Receipt/due-date events and contract-time-zone date rollover trigger updates, with reconciliation to recover missed events. Preserve idempotency and disclose projection/as-of lag. Design against the established >1M/day workload without source-wide scans or row-wise UI requests. Actual freshness/refresh SLO, receipt tolerance and notification timing/channel values remain unresolved.

## Existing acceptance IDs carried forward

Every case is NOT_RUN. These preserve the local source IDs rather than adding sixteen new product features.

| ID | Required planned result |
| --- | --- |
| PT-01 | Contract-specific Net30/45 date examples, day-0 proposed calculation, midnight/month-end/leap/DST boundaries |
| PT-02 | Issued term snapshot survives contract revision; approved extensions preserve original date and history |
| PT-03 | Partial/unallocated payments, credits, cancellations, overpayments and reversals conserve per-currency balances without double subtraction |
| PT-04 | Stale/unknown ERP receipts and date disagreement are not confirmed overdue; late receipts/reversals reevaluate correctly |
| PT-05 | Portal/search/totals/export preserve customer scope and contract/date-change authority |
| PT-06 | Aging equals the same-snapshot per-currency receivable total at 0/1/30/31/60/61/90/91-day boundaries |
| PT-07 | Unissued/internal/disputed/held/corrected documents remain distinct; no duplicate or stale incorrect notifications |
| PT-08 | Bulk date rollover, event replay, outage catch-up and projection lag are bounded and honest |
| ERPC-01 | Three monthly recognition submissions for quarterly billing create no customer invoice/delivery/due-date start |
| ERPC-02 | Monthly 100/120/80 versus 300; 290 requires -10 adjustment/posting/reconciliation, counted once |
| ERPC-03 | Missing/unknown/partial months are not zero or complete; idempotent resume and result lookup avoid duplicates |
| ERPC-04 | Monthly then quarterly handling avoids double recognition through verified offset/replacement/reversal/reference and external net reconciliation |
| ERPC-05 | Closed periods, late sources, corrections, cancellations and manual ERP changes preserve owned-history boundaries |
| ERPC-06 | Entity/customer/basis/currency/tax/FX distinctions reject invalid aggregation |
| ERPC-07 | Partial months/non-calendar quarters/cadence changes preserve snapshots and scoped approval |
| ERPC-08 | Bulk monthly/quarterly close, projection lag, recovery and official API/file combinations require actual validation |

Still unresolved: calendar/holiday/max-net-days policy, receipt authority/freshness, extension roles, notice/refresh targets; per-ERP journal/accounts, provisional recognition, fiscal calendar/tax/FX, customer-issuance gate and anti-double-recognition method. See [D-02/03/07](../../planning/v1.16/planning-decisions.md). No legal/accounting certification, messages, ERP mutation or product implementation occurred.
