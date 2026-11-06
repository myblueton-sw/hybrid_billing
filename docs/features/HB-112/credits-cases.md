---
type: ScreenDesign
title: Credits and billing inquiry evidence
description: Designs B11 as separate credit, case, correction and payment-term evidence views without inventing financial or public-message authority.
status: draft
---

# Scope and ownership

HB-112 follows B10 on open PR87 at5a22c9835d9bfb6054103063fe29b9b8e711eed1; separately fetched origin/main is recorded during execution. Root owns PM/UI/integration/QA; an isolated writer may own B11 contract/client/tests/fixtures. Independent finance/TW and security reviewers inspect design/actual changes, and Claude CLI ui-designer supplies composition/final rendered comparison. No previous leaf/shared style/backend edit. Minimal App wiring must remain below1500physical lines. Marketplace remains held. No merge, deployment or owner FIX.

B11 spec, payment-and-recognition and Claim numeric contracts require separate balance authority/freshness, available/reserved, conservation components, inquiries/internal notes/public replies, original-linked corrections, consent, holds and payment due dates. Existing discovery found no credit/case/receivable producer. Normal view is unconnected/countunknown/no fixtures. This increment is read-only evidence UI, not full B11 write workflow: no new inquiry, reply, adjustment draft, resolution save, credit application/reservation/reversal, hold/release, payment/receipt/date change or export.

# Consumer-only read boundary

Proposed fixed same-origin GET `/api/credit-case-review`, no backend producer implemented. Actual404/501→unconnected; live payloads fail producer_unbound pending separately reviewed producer/auth/disclosure binding. Only isolated QA intercepts supply always-labelled synthetic records; no URL/window/storage fixture switch. Strict nested allowlists, bounded normalized q, view and nullable exact totals,25rows, snapshot+permission_revision cursor pins, auth_scope digests, exact view/filter/cursor response identity, redirecterror/no-store/trusted local-client header, streamed2MiB cap/fatalUTF8/abort. All scope/view/filter/cursor/error changes discard old record/count/detail before response; sequence gating and current-page ID selection; no raw external URLs/HTML/errors. Full evidence/error/selection identity enters owner review digest. No current authorization is proven by hashes or UI role previews.

Views keep balances/reservations/cases/adjustments/payment-terms. Use a discriminated read record, not one row pretending to be both credit and receivable. Query has view and bounded normalized q only (no unverified option lists). Common id/revision/label, nullable entity/customer/contract/period, updated_at, authority {kind platform/external/unknown,reference,observed_at,freshness current/stale/unknown}. All source evidence is declarative, never a grant.

Draft record shapes for independent design review:

- Balance kind balance: dimension {kind money/quantity,currency orunit}; opening/increases/applied/expired/reversed/closing/available/reserved decimalstrings|null; migration_ref and included_history_ref. Display source components without client conservation/availability computation, reconciliation badge or second ledger. Negative monetary components retain signs; quantity values use declared source unit/sign semantics. Missing authority/freshness is not usable balance. Opening migration/history references do not prove duplicate prevention.
- Reservation kind reservation: same dimension, credit_ref/document_ref/original_ref, source state reserved/applied/released/outcome_unknown/reversed/unknown and value decimal|null. Status is not financial execution or release authority.
- Case kind case: state open/in_review/resolved/closed/unknown, original_document_ref, disputed amounts by explicit currency, owner_label, received_at/deadline, separate internal_note/public_reply source text, resolution confirmed_normal/correction_requested/corrected/unresolved/unknown and resolution_ref; consent state recorded/declined/not_recorded/not_applicable/unknown with evidence_ref (recorded requires evidence). No editing/sending; a correction link never implies customer consent, exemption or settled balance.
- Adjustment kind adjustment: original_document_ref/original_claim_ref/correction_ref, reason/evidence_ref/approval_ref, source kind credit/refund/quantity_reversal/other/unknown, source state draft/pending/applied/rejected/unknown, separate per-currency values. No computed SLA discount or automatic fee pass-through, no quantity reopening or replacement authority.
- Payment kind payment: original_document_ref, issuance issued/not_issued/internal/unknown, term_ref/term_label/time_zone, issue_date/original_due_date/effective_due_date as actual ISOdates|null; receivable money {currency,value:null|string}; receipt {state confirmed/unknown/not_provided,reference,observed_at,freshness current/stale/unknown}; overdue on_time/due_today/overdue/fully_paid/unknown/not_applicable. Source clock/calendar/receipt projection owns status; no client Net30/45/date/aging calculation. Known overdue/paid/timely status needs issued/dated document, nonunknown authority/current receipt evidence, explicit currency and remaining value; stale/missing receipts require unknown. Overdue requires positive remaining; fully_paid requires0. Internal/unissued do not become confirmed overdue. No confirmed overdue aggregate.

Case/payment records expose bounded hold observations {type billing/delivery/collections,effect requested/effective/released/pending/unknown,reference,revision,target_ref,observed_at,external_effect confirmed/pending/unknown/not_applicable}. These are source-declared effects, not commands; requested is not effective, unknown is not released, no automatic resume/date/balance change. Consent and public/private text remain separate; private text is never merged into public reply or customer routing.

Money/quantity transport is finite base10 strings (no exponent/float, bounds80chars/76digits/38fraction, no activated numericprofile/rounding). Currency and unit never combine. No page/crosscurrency/credit-receivable totals or absent0. Nullable scope/values remain unknown. Future producer must close sign semantics, authority, financial conservation, receipt policy and disclosure independently; read-shape checks cannot certify them.

# Screen and verification

Fullwidth canonical IA/logo/localPlex16body/control/table20h2/14meta/AR1.8; neutral square borders/no colorededge/shadow. Five visible mobile tabs; direct-child row selectors protect inline close controls; split desktop>=1024, selected-row inline detail below1024; source textauto, complete amounts+unit/IDs/datesLTR. Default unconnected has safe B09/B10 navigation and real reread, no pretend command buttons. Loaded synthetic views have bounded kind-specific records, source scope/as-of/freshness before values, disclosed internal/public text, source holds/consent/reference evidence, page/inspect/close/Escape/focus and closed full-column table. No ordinary product checklist.

| Stable item | Purpose | Relevant states/checks |
| --- | --- | --- |
| B11:scope:read-query | Current view and reference/label search | loading/empty/error/scope invalidation |
| B11:data:credits | Authority/as-of, source balance components, no client conservation claim | zero/null/stale/money/quantity |
| B11:data:reservations | Supplied reservation/application/reversal references | pending/unknown/released |
| B11:data:cases | Dispute, deadline, internal/public source text | open/resolved/unknown, separate disclosure |
| B11:data:consent | Explicit consent/evidence, never inferred | recorded/declined/not_recorded/not_applicable/unknown |
| B11:data:adjustments | Original-linked source correction and approval evidence | draft/pending/applied/no quantity authority |
| B11:data:payment-terms | Source issue/due/remaining/receipt freshness/overdue | stale/unknown/partial/zero/notapplicable |
| B11:data:holds | Type/target/effect/external uncertainty | requested/effective/unknown/released, no resume |
| B11:action:inspect | Canonical current-page detail and focus return | desktop/mobile/tablet/history |
| B11:state:async | Unconnected/denied/stale/failure/late response removal | unit/browser/source reviews |

Plan60main variants (5views×3locales×2widths×actualunconnected/isolatedsynthetic), actual404, source-specific guards, scoped keyboard/search/paging/late success+error/denied/malformed/livefailclosed/previewnegative, tablet close widths, localfonts/neutral borders/RTL/nooverflow, existing B10 regression and full frontend tests. Native/human/otherOS/real financial and public-message workflow remain deferred.

# External evidence

[Stripe credit balance](https://docs.stripe.com/invoicing/customer/balance) informs immutable transaction references, credit-versus-cash distinction and separate internal notes. Adopt evidence hierarchy only; reject Stripe sign conventions, automatic next-invoice application, formulas/commands/assets/SDK setup. [Zendesk ticket comments](https://developer.zendesk.com/api-reference/ticketing/tickets/ticket_comments/) and [Adding comments](https://support.zendesk.com/hc/en-us/articles/4408828489370-Adding-comments-to-tickets) inform explicit public-versus-internal containers; no reply editor, sending behavior, API mutation or notification defaults are imported. The first guessed Zendesk help URL was inaccessible; official search resolved the correct article and developer reference. Product contracts govern balance, consent, receipt and hold meaning.

# Independent design conditions

Finance/TW and security design conditional PASS: exact view→record kind and money/quantity dimensions; retain signed values, no balance-conservation/availability certification. Stale authority shows qualified historical observations, not current usable balance. Known payment states require original issued document, term/date/timezone snapshot, current referenced authority and confirmed/current referenced receipts; overdue positive remaining, fullypaid exact0, negative never overdue/paid. Stale/missing receipts/authority require unknown applicable status; explicit internal/notapplicable stays separate. Effective/released holds require target/revision/asof plus confirmed external effect or explicitnonapplicability; pending/unknown external effects cannot support them. Applied adjustments require originaldocument orClaim link. Consentrecorded requires evidence. Monetary quantity-reversal observations disclose missing quantity evidence and grant no replacement authority. Private/public source text stays separate; no local send, correction, financial or public-state transition is implemented.

# Composition reconciliation

Actual Claude CLI ui-designer read the before B11 and B10 precedent captures, foundations and plan; root independently read the official Stripe/Zendesk pages (the CLI had no web tool and did not claim browsing them). Adopt7/12 bounded list and5/12 source card at>=1024, inline selected-row detail below1024, authority/freshness before values, separate closed internal/public disclosures, source consent/holds and type-specific cards. The five mobile tabs use a readable two-column grid with a full-width final tab rather than fitting long Arabic labels into three narrow columns. Canonical keys and finance-preferred labels prevail. Record heading uses h2 to remain at20px and avoid skipping a heading level under the page h1. Direct-child row selectors prevent the B10 tablet-close regression. Exact-value scroll regions remain keyboard-accessible. No external sign convention, automatic credit application, reply composer or command defaults were adopted.
