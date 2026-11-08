---
type: DesignSpecification
title: Customer relationships and protected configuration
description: Specifies organizational changes financial contact protection lifecycle and profile aware overviews.
status: draft
---

# Context

Inherits [shared interaction rules](../../11-settings/r2/README.md). Evidence: HB-58 ADM04–07; v1.16 B1197.R002–R007, B1206.R001–R005, B1209–B1215, B1468–B1476, B1493 and B1500; [pricing obligations](../../../contracts/billing/source-pricing-and-cost-obligations.md), [payment contract](../../../contracts/billing/payment-and-recognition.md) and [screen bindings](../../../ui/screen-binding-register.md). All mutations below have **unresolved bindings** unless a canonical command is explicitly named. Generic administration never grants monetary approval.

# O01/O02 — hierarchy and relationship changes (ADM04)

Entry: Customers → organization/group detail → Relationships. Scoped relationship administrators see separate tabs for legal entity hierarchy, reporting membership, provider account/payer association, attribution, contract/allowance pool and allocation pool. None implies any other; paying agent is separate from buyer. A reporting-group move does not move a CSP account, change the legal invoice bundle or reset consumed allowance.

Fields: entity IDs and relation type, old/new parent/group, provider/account reference where applicable, contract and complete pool IDs, valid-from/to, effective timezone, reason, immutable target manifest, affected periods, issuer/buyer/currency/tax/ERP route, current grants and export/report impact. Reject cycles, conflicting or duplicate intervals for the same exclusive relationship, incompatible complete attribution and ambiguous relations. Legitimate overlapping ReportingGroup memberships remain allowed under B1197.R003; aggregate views deduplicate the same canonical entity/charge at the declared basis rather than summing membership copies. Reporting membership is not exclusive legal ownership or financial attribution. Dependency traversal must be complete; split pool selection is blocked.

Flow: select → freeze targets → impact preview across access/financial/report/export dependencies → required independent approval of digest → schedule/apply → dependency-unit read-back. Each indivisible contract/allowance/allocation dependency unit applies atomically or stays unchanged; independent units report separate results. Recheck current authority and revisions at execution. No silent partial unit or historical reattribution. A retroactive change uses the separate governed correction path.

Walkthrough W01: moving a company into a reporting group changes future reporting membership, retains original period attribution and its consumed contract allowance, and does not change provider ownership. If the move would alter access or financial scope, require the corresponding independently approved decision. A concurrent pool revision invalidates preview; rebuild the complete manifest. Retry a failed independent unit only after reinspection, never repeat completed units. A move does not revoke already downloaded bytes; current download links and caches must recheck new permissions.

# O03/I01 — protected billing settings (ADM05)

Entry: Customer → Billing settings; view and edit permissions are distinct and field-scoped. Sections: legal/tax identity, payment account, billing recipients, document template, issuer/issuance mode. Fields: canonical owner/entity references, masked bank details, recipient reference and purpose, tax identifier, template revision, old/new values, evidence reference, requested effective date, eligible future documents and proposer/approver. No free-text recipient list silently replaces approved contacts.

Flow: draft → verify changes through an existing registered independent channel (never solely the newly supplied channel) → impact preview → separate authorized approval bound to exact revision/evidence → apply at effective instant → read-back. Changing bank/recipient/evidence invalidates prior approval. Issued or externally pinned documents retain original snapshots; unissued affected approvals become stale and require appropriate recomputation/reapproval. Mode switching is permitted only for verified capabilities, never automatic self-issuance on ERP failure. This flow performs no payment and changes no issued payment due date.

Walkthrough W02: a proposed bank change is verified with the existing registered contact, independently approved and used for eligible future documents only. If the registered channel cannot verify it, hold the change and show the evidence owner; no send-to-new-contact shortcut. If issuance wins a race, that issued snapshot remains unchanged and the change applies only to the remaining eligible set. Unknown dispatch is pinned for reconciliation rather than treated as unissued.

# O03 — contract and service lifecycle (ADM06)

Entry: Customer → Contract → Lifecycle; authorized contract operators draft and finance/security reviewers approve their separate impacts. Fields: event start/change/suspend/resume/end, effective bounds/timezone, contract revision, recurring and one-time charge policy, proration method and basis (amount/quantity), allowance/minimum/discount rules, currency, billing cadence, service period, last collected interval, final evidence requirement, late-source window policy, responsible owner and access-removal plan. Missing proration policy or overlapping/unbilled intervals blocks final calculation; do not invent proportional days or a recognition date.

Show service lifecycle independently from financial state: ended may remain pending_final; settled requires reconciled final evidence and authorized financial decisions. Suspension/resumption does not imply debt waiver, deletion or automatic rerating. Final collection should precede routine source-access removal where possible; unavailable evidence records reason, owner and follow-up instead of indefinitely retaining access. Urgent security revocation always proceeds immediately and records resulting evidence limitations. No VM destruction, provider subscription cancellation or customer data deletion is implied.

Walkthrough W03: end mid-period → preview policy-based intervals → approve effective end → collect eligible final evidence → confirm inputs via `input.confirm_publish` → compute/reconcile a fixed complete snapshot and resolve required blockers/warnings under their approved policy → separate final amount decision via `settlement.approve_final` → issued/history handoff. Late credit retains original Claim/document lineage through a correction; it never reopens an immutable invoice. If final collection fails, ended/pending_final remains visible with owner and next action. Security revoke proceeds regardless, and later recovery uses authorized replacement evidence, not restored old credentials.

# U01/O01 — profile-aware overview (ADM07)

Entry: Work → Overview. Fields/filters: operating profile, permitted entity/group/account/period/currency, fixed attribution basis, data freshness, collection readiness, cost, effective selling amount, internal allocation, receivable status and blockers. MSP, standalone enterprise, corporate group and internal-chargeback views change labels/navigation and relevant summaries; they grant no rights and never combine internal allocation with external revenue. Missing data is not zero; mixed currencies are separate unless an approved conversion basis is explicitly selected and labelled.

Flow: choose authorized profile/scope → inspect summary with denominator/completeness → drill into O03/S02/B09/I04 or R01 at the same canonical scope → return preserving filters. Keep the existing large customer-list/favorites workflow with pagination, stable sorting and bounded queries; no measured 5,000-company performance claim is made.

Walkthrough W04: a group controller drills from a reporting total into one entity and sees historical attribution and current permitted detail. An inaccessible entity does not appear through counts, search or exports. Changing profile cannot expose cost fields; missing source data displays an incomplete state with S02 action. `favorites.add/remove/list` use current customer read and personal scope; overview query bindings remain unresolved.
