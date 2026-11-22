---
type: ImplementationPlan
title: B09 approved request to independently approved draft
description: Connect the existing approval inbox to the retained draft workflow through user-readable context and recovery.
status: draft
---

# User outcome and ownership

HYB-341, machine:mac_mini. The owner requires additions to earn their place through user necessity. The task is to select a final-approved request, review its derived invoice facts, create its draft, and let an independent authorized colleague find and approve it. Users must not copy internal IDs, digests, monetary values or reviewer identities between screens. Recurring-period continuity is supported by the actual private two-invoice acceptance, not by an external ERP or production claim.

Root owns implementation, product scope and integration. Actual Claude CLI ui-designer owns the material visual proposal; refusal_review independently reviews financial/disclosure contracts. Platform investigation checks existing prerequisite setup ownership. Technical-writing and rendered re-review remain required before screen acceptance. No owner FIX is implied.

# Evidence and existing surface

[B09](../../ui/screens/B09/spec.md) defines fixed-result review and independent approval. The existing approval-inbox result detail already creates final-approval requests. Extend that existing work area rather than creating an unrelated dashboard. The separate review-issue screen remains a local synthetic demonstration and is not evidence of a connected user journey.

[Private integration](issuance-draft-integration-plan.md) implements immutable source-derived drafts, separately owned approval attempts, full reviewed references/payment coverage, current execution checks and historical receipt disclosure. Public operations and screen composition require the concrete review in this plan before implementation.

# Minimum usable flow

1. Open a final-approved request in the existing approval inbox. Load current draft context scoped to that exact request and digest; discard stale replies on selection/session/workspace changes.
2. Show customer, billing period, derived gross/currency, document language and meaningful readiness. Context resolves the exact applicable reusable configuration internally. No matching configuration is a missing setup condition, not permission to silently create one.
3. The submitter chooses a supported language and confirms creation of the draft. The source determines amounts/parties. The UI generates an opaque operation identity, retains the immutable submitted intent across response loss, and reads authoritative state after navigation.
4. An independently authorized reviewer finds the submitted draft in a bounded list within B09. Show relevant period/customer/amount/state rather than exposing only technical identifiers. Inspection and operation permissions remain separate.
5. The reviewer chooses from actually applicable, independently approved payment bindings presented with readable terms and time zone. Preparing the review stores an immutable attempt; label it as an action, not a harmless preview. The subsequent review shows exact references and calendar coverage before approval.
6. Approval records the pinned facts. An approved historical receipt explicitly does not establish today's issuance eligibility. Changed references/calendar require a fresh explicit review; unresolved effects retain their original actor and immutable intent.

# Supporting contract proposal — review pending

Reuse the existing command HTTP envelope, revision checks, error handling and before-cache boundary. Required mutations are submit, prepare approval and approve. Required reads are selected-final context, bounded draft discovery and selected-draft inspection including its latest review. A separate generic history API is not required by this user task.

All reads require current actual inspection rights before reading source or retained cost evidence. Context must separate historical final facts, current source eligibility, setup readiness and caller permission; never infer readiness from recorded approval state alone. Mutation rights also require inspection. Reads must not invoke recovery or settle unknown outcomes.

Context should return typed prerequisite states and readable binding options while retaining exact configuration/binding IDs and digests internally in the client. No raw nested source/profile graph belongs in a response. Bound list size, option count, returned lines and exact serialized bytes using existing command limits; refuse oversized outcomes before mutation. Stable selected-draft inspection validates immutable ownership and actual phase-aware module readback. It supports submitted, submission-unknown/refused and approval-prepared/unknown/refused/approved states without conflating them.

Before cached mutation replay, enforce current caller rights and pinned actor ownership. A new-envelope recovery attempt is distinct from same-envelope retry. Classify uncertainty from this attempt's possible effect; an unrelated already-unknown state cannot turn a clean refusal into an uncertain result. Unknown readback is evidence, not authorization for another actor to recover.

# Necessary setup boundary

Profiles/kind reservations, protected channels/party references, payment terms/bindings and draft grants currently use private governed paths. Existing organization/integration/access screen ownership must be inspected before adding any setup interaction. Display precise missing prerequisites and only real available destinations. Do not manufacture defaults, seed permissive rights, or call fixture-populated screens a complete operator journey. The minimum missing setup interactions are unresolved design inputs, not authorization to build broad administration APIs.

Source inspection identifies O03 `/contracts` (existing customer-detail body) as contract/issuance-profile and payment-terms ownership, and M03 `/access` as business-rights ownership. The [payment and recognition contract](../../contracts/billing/payment-and-recognition.md) explicitly places terms in O03. Existing O03 is synthetic-only; M03 has a live result-authority area but no draft-authority area. P02 integration/event channels are not protected verification channels. O03's detailed `docs/ui/screens/O03/spec.md` and referenced `docs/planning/v1.16/ui-design.md` are absent from both this worktree and the authorized original workspace. Do not reconstruct an accepted layout from the catalog. Dependent O03 composition remains pending verified baseline or a separately reviewed new design grounded in available owner/planning evidence; independent B09 contract/design work can continue.

# Visual references and constraints

The official [Carbon data-table guidance](https://www.carbondesignsystem.com/building-blocks/core/components/data-table/guidelines) supports bounded selection with a search/filter toolbar and progressive detail. Adopt that hierarchy, not bulk financial approval or a new component dependency. The official [GOV.UK check-answers pattern](https://design-system.service.gov.uk/patterns/check-answers/) supports a grouped human-readable confirmation before mutation. Adopt the confirmation step, not its fixed-width container or editing of source-derived amounts. Both were consulted for this design increment.

Use full viewport width, canonical assets/local IBM Plex, square controls and neutral separators. No decorative accent borders or edge strips. Preserve visible keyboard focus and scoped screen-review controls. Render EN/KO/AR at desktop/tablet/mobile; Arabic uses RTL with isolated money/dates/identifiers. The actual Claude invocation receives prior B09 renders and these references; returned design and changed-render acceptance must be recorded separately.

# Acceptance and findings ledger

Required behavior: ordinary submitter and independent reviewer complete the task without typing internal identifiers; applicable setup is discoverable; submitted/approved records survive navigation; response-loss recovery preserves exact ownership; permission/session changes clear inappropriate cached data; old async responses cannot replace a new selection; historical receipts remain distinct from current eligibility. Validate keyboard review/error/recovery paths and all locale/viewport variants.

Required boundary evidence: before-cache grant loss, wrong actor recovery, current-source failure, reference/calendar drift, unknown inspection without mutation, frozen inspection after financial expiry, exact output limits and unavailable adapter. Reuse passing private tests; add public/browser evidence for the actual new boundary rather than duplicating unrelated suites.

| Finding | Scope | Severity | Evidence | Resolution / recheck |
| --- | --- | --- | --- | --- |
| Setup ownership is not yet connected to public user actions | B09 prerequisite path | Blocking for full user journey | Private setup methods and existing integration plan | Investigate existing screen ownership; pending |
| Proposed visual composition has no changed-render verification | B09 draft states, EN/KO/AR | Blocking for UI implementation acceptance | Actual CLI proposal received; no changed render exists | Proposal received; rendered review pending, see visual-design evidence below |
| Public boundary remains incomplete | Context/list/inspection and three mutations | Blocking for connected user journey | Corrective executable design accepted; public list/inspect implemented and independently reviewed | Implement current context and three mutations; see [public contract](issuance-draft-public-contract-plan.md) |

No issuance, numbering, rendering/download, delivery, payment, generic reporting expansion, deployment or production persistence is included in this increment. Those remain distinct product requirements where the original plan requires them; excluding them here does not complete the overall goal.

# Design review findings and required correction

Independent financial/disclosure review accepts the user direction but does not approve public implementation yet. Exact DTO fields/caps/order/snapshot cursors, readiness codes, mutation targets/fields/receipts and attempt-specific effect classification must be fixed. Existing private DependencyUnavailable failures can follow actual writes; directly exposing them as clean HTTP refusals is incorrect. Historical inspection cannot simply reuse held mutation admission or submitted-only ownership validation. Preparing a review persists exclusive actor-owned state and provides no cancellation/reassignment protocol.

A more fundamental user-necessity finding blocks freezing the public submit/context contract. [FA-12](../../architecture/data-ownership/source-relationship-bindings.md) defines reusable scoped IssuanceProfile revisions and effective-dated contract-item mode bindings; [EI-01](../../contracts/billing/source-erp-and-issuance-obligations.md) limits changed mode/issuer to unissued scope. Neither requires fresh profile/kind/channel governance for every monthly invoice. Current private profiles bind an exact original financial source, consume their reservation and derive kind identity from that profile. The passing two-invoice test therefore required new configuration solely because of this implementation coupling. It proves composition after expiry, not acceptable recurring setup effort.

Before public submission exposure, separate reusable independently governed configuration from per-invoice source application/snapshot. Unchanged applicable configuration retains its kind and mature template channel. Each invoice still needs fresh final-source validation and its own independent draft approval. Preserve original immutable records; do not reinterpret their meaning, fabricate channel age or adopt foreign module objects. Exact applicability/revision rules and a migration boundary require a concrete independent design review. Revised acceptance must process B after A expires using unchanged approved configuration and mature channels, without new configuration approval, while preserving A's history. This correction is required to remove unnecessary user work, not an optional platform expansion.

# Actual visual-design proposal evidence

The actual Claude CLI `ui-designer` invocation completed with exit0. It inspected the existing B09 source and three prior final-request renders: full KO1440 and KO390/AR1440 panel crops. It returned a proposal, not a rendered PASS. Root supplied the official external-reference summaries; Claude did not browse them. The proposal keeps B09's existing list/detail layout, places draft creation on selected final-approved requests, and makes drafts discoverable to independent reviewers through the existing approval workspace. It specifies neutral readiness rows, grouped review facts, explicit saved-review action, supported-language choice, pinned terms/references, actor-owned recovery, responsive/RTL structure and stable B09 draft item IDs.

Outstanding design findings include source-scale money display, meaningful governed labels, server discovery of unresolved attempts after reload, unavailable setup destinations and terminology review. The proposal has no final Arabic copy or native acceptance. Current final-request reference input is a separate existing usability finding; it must not be copied into the new draft flow. Public contract and reusable-configuration correction take precedence before implementing this proposal. Local supporting output: `/private/tmp/hyb341-draft-journey-design.txt`; no generated screen or owner FIX is claimed.

# Reusable configuration correction evidence

The private application-backed draft correction is now implemented and independently reviewed. Root's actual two-invoice test passed: one configuration, three mature channels/changes and one approved binding precede A; after A is independently approved and its financial approval expires, B is independently approved from new November evidence with exactly the same setup. A history and setup counts remain unchanged. This closes the monthly reconfiguration concern for the private initial-revision path. It does not resolve public disclosure/recovery contracts, setup discovery or rendered user acceptance above. Public context must select the applicable reusable configuration and exact current final evidence, rather than exposing the legacy source-bound profile mechanism.

# Public read integration checkpoint

The [executable public contract](issuance-draft-public-contract-plan.md) passed independent corrective backend design review. Public draft list/inspection are implemented, including retained unresolved work, frozen review recovery arguments, held/expired historical disclosure under current Inspect rights and complete bounded pagination. Root verified actual 11-draft and 12-line HTTP traversal and corrective state-integrity cases; final relevant race PASS100.900s. Current context, three public mutations and their pre-effect output/uncertainty boundaries remain pending, followed by B09 composition and actual rendered/terminology review. No connected screen acceptance follows from the read API result.
