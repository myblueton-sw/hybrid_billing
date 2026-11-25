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
| Connected screen workflow remains incomplete | Context/list/inspection and three mutations | Blocking for connected user journey | Six public commands implemented; backend source/test review passed | Connect B09 and prerequisite setup, then verify rendered journeys; see [public contract](issuance-draft-public-contract-plan.md) |

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


# Current context checkpoint

The selected-final context API now separates frozen final amounts from current source eligibility, caller permissions, applicable configuration/binding choices and action readiness. Actual Inspect-only callers can see missing setup without receiving Submit rights. Current facts never settle retained unknown operations; saved review ownership and explicit fact-recheck requirements are reported separately. Held/expired source behavior, service binding identities, scoped cursors and current authority checks have executable HTTP coverage. Independent source/test review accepted this boundary after the future-binding applicability correction described in the public contract. Three mutation wrappers and their output/effect semantics, prerequisite setup UI and B09 rendered acceptance remain pending.


# Public mutation checkpoint

The configured-only submit, save-review and approve commands now reuse the existing session/CSRF/step-up/hold/revision/idempotency envelope. Current operation plus Inspect rights and pinned business identity are checked before cached disclosure. Response admission covers every projected line page and possible receipt/review state before effects. Invocation-local tracking distinguishes actual retained changes and module attempts from unchanged prior uncertainty. Exact historical business replay preserves current caller rights without reopening expired financial source eligibility. Backend implementation and independent source/test review are complete for these commands; integrated regression evidence is recorded in the public contract. The prerequisite setup and rendered B09 journey remain incomplete.


# B09 discovery scope correction and client checkpoint

The subsequent actual Claude CLI ui-designer read-only delta review completed successfully. It inspected prior KO1440 and AR1440 renders and the supplied public API summary; access to the contract file was denied, so it did not independently inspect that file. This was a proposal review, not rendered acceptance of new UI. The proposal places retained drafts in a separate Issuance drafts tab inside B09, following Decision history. Existing approval kinds and decision history remain unchanged because the draft API exposes only the latest review, not a complete decision ledger. Draft creation stays on the selected approved final request; its receipt opens the draft detail.

The tab uses the existing full-width list/detail pattern, ten-row snapshot-bound pages and complete line pagination. It labels identifiers honestly, preserves source-scale monetary strings, requires explicit binding selection, and distinguishes saved review from approval and actual issuance. Original-actor recovery cannot become takeover or cancellation. Stable review items are B09:draft:list-scope, list-paging, list-row, create-entry, action-state, term-choice, inspect, owned-recovery and flow. EN/KO/AR terminology and new 390/768/1440 rendered review remain pending.

The initial read client implements list, selected-final context and inspection with current-session checks, exact response correlation and scoped cursors. Six targeted client tests passed; TypeScript typecheck passed. These checks cover read-session isolation, envelope identity, snapshot validation and exact money display, not mutation or connected-screen acceptance.

Context now probes the actual pure configured draft builder before reporting source readiness. A targeted helper test passed for supported scale four and rejected scales five, seven and eight, without retaining drafts. Independent refusal_review accepted the source/test change and explicitly ran no tests. This is helper-level representability coverage, not a high-scale producer HTTP journey.


# Draft mutation client checkpoint

The client now sends configured submission, review preparation and approval through the existing command envelope. It freezes exact arguments, rationale, revision, actor/workspace and request/idempotency keys. Current-session validation precedes sending; malformed or effect-bearing failures remain uncertain. Exact no-effect dependency/authority unavailability is distinguished from uncertain mutation failure. Receipt correlation includes target, draft digest and approval review digest. Submission success must be submitted; approval success must be approved or refused. Prepare replay retains supported historical review states.

Independent refusal_review found overly broad success-state acceptance in the initial client. Root corrected action-specific terminal-state validation and added tests for all four impossible success states; the reviewer accepted the correction through source/test inspection without running tests. Sixteen focused read/mutation/intent tests passed. Intent factories generate new business identifiers from the selected source and accept only explicit returned binding choices. Reload recovery preserves the server's original actor, business identity and immutable arguments. These are client-level checks; the connected screen, persistent in-session uncertain-operation handling and rendered journey remain pending.


# Initial draft tab and asynchronous state checkpoint

The B09 Issuance drafts tab now renders retained list/detail and complete snapshot-bound row/line paging. This initial screen does not yet expose creation, saved-review actions, protected reference/coverage presentation or retained mutation outcomes; those remain required before user-journey acceptance. A session-bound store retains uncertain attempts across selection changes, prevents duplicate pending retries, and invalidates reads before and after mutation settlement. Independent review found and root corrected the during-mutation stale-read race; both completion orders have regression coverage and source re-review passed. Twenty-one focused client/store tests passed.

Initial browser verification used an isolated synthetic session and intercepted draft APIs against the actual Vite entry on port 8898. EN/KO/AR at widths 390 and 1440 passed list selection, exact large monetary display, unavailable current-context feedback, no page exceptions and no viewport overflow. Root inspected KO1440 and AR390 screenshots. During that check root found and corrected non-JSON review-state reporting that crashed the shell. Browser evidence is local at /private/tmp/hyb341-draft-screen-smoke; this is client fixture evidence, not live backend or complete rendered acceptance. Claude rendered review and width 768, actions, keyboard traversal and full asynchronous journey coverage remain pending.

Terminology source review required synthetic/historical eligibility qualification and distinguishing a refused approval attempt from a reviewer rejection. Root updated those labels and the Korean workspace term. The terminology reviewer inspected the guide/glossary but the tech-writer skill was unavailable in both local trees; no native-language acceptance is claimed.


# Connected draft actions checkpoint

The selected approved final request now exposes draft creation with explicit supported document language and rationale; IDs, configuration and source digest are derived rather than typed. The drafts tab presents saved references/coverage, explicit payment-term choice, review saving, approval, identical retry and original-owned recovery. Outcome records remain visible independently of selection. Saved review is clearly separate from approval and issuance. Prerequisite setup screens remain outside this increment.

Independent UI source review found three integration defects: newer context could accompany an older inspection; prerequisite refresh erased creation inputs; and a retained unknown prevented fresh-envelope recovery of the original business operation. Root corrected all three. Prepare/approve require matching inspection/context revisions. Creation inputs survive prerequisite refresh; selection/navigation guards require explicit discard. Fresh recovery requires a current inspected allowed descriptor, matching original actor/business arguments/revision, and no competing pending or unrelated unknown operation. Successful reconciliation keeps the original error and marks its business result reconciled, rather than claiming the original HTTP response succeeded. The reviewer accepted these scoped corrections without executing tests or claiming full UI acceptance.

Twenty-two focused tests passed. Actual isolated browser fixtures against Vite confirmed selected-final creation, preserved language/rationale across prerequisite refresh, explicit payment-term selection, saved reference/date presentation, approval after a lost response using the identical request, and retained approved inspection. A separate scenario returned cached internal-error uncertainty on identical retry, then verified fresh original-owned recovery of the same review without reload; original business target/arguments stayed unchanged while HTTP request identity changed. These are synthetic intercepted-API client journeys, not live backend end-to-end proof.

Terminology review corrected reviewed issue-date basis, Korean payment terms/deadline wording, billing-target ownership, exclusive saved-review ownership and refusal-versus-rejection wording; no native acceptance is claimed. Final multilingual action layouts, keyboard coverage, actual Claude rendered review, scoped full integration checks and Git/PR checkpoint remain pending.


# Corrected rendered and server-contract verification

Actual Claude CLI ui-designer inspection found five composition blockers: centered list content, broken radio/label layout, empty list-track space after mutation invalidation, offscreen active phone tab, and clipped mobile line-table content. Root corrected these within B09. Post-mutation detail now spans the available workspace while invalidated list data stays removed. At widths up to 480px, bounded lines use stacked labelled values. Scoped mobile tabs wrap so the active view remains visible. The designer independently inspected corrected KO1440, EN768, AR390 and AR1440 renders and returned scoped PASS.

The first creation render review found stretched controls and unlabelled source amounts. Root added explicit request/approved-source-amount facts and readiness, capped the language selector at 20rem and the rationale at 72ch, and separated field labels. The actual designer re-read current KO1440, EN768 and AR390 creation images and returned scoped PASS. Korean rationale and document language in EN/AR are explicit synthetic test inputs, not built-in defaults; document language starts unselected and is independent of interface language. These are rendered design passes, not owner FIX, native-language or assistive-technology acceptance. Local CLI results: /private/tmp/hyb341-draft-render-review.txt, /private/tmp/hyb341-draft-render-rereview.txt and /private/tmp/hyb341-draft-create-rereview.txt.

Root verified creation and review/approval/retry journeys in all nine EN/KO/AR by390/768/1440 combinations after relevant corrections. Keyboard radio selection, Tab traversal to reason and save, Enter activation and visible focus passed in KO1440. No screen-reader or cross-OS certification follows.

The new Go fixture executes actual authenticated synthetic HTTP handlers for creation context, submission, list/inspection, review context/preparation, approval context/approval and approved inspection. Its credential-free responses are accepted by the TypeScript decoders and mutation receipt client. The client transport in that check is intercepted: it is captured-wire compatibility proof, not a live browser-to-server end-to-end test. Independent parser review found and root corrected truncated initial list/binding-page acceptance, coercion of array-valued enums, and review evidence on non-submitted drafts. Corrective source/test review passed; 25 focused tests including actual server wire passed.

Final console suite: 1,163 passed, two skipped, zero failed (1,165 total). Typecheck and production build passed; the build retains the existing large-chunk warning. Backend context regression passed in 4.161s and actual HTTP fixture generation in 0.799s. Full backend race evidence at fd6f403 remains the prior checkpoint; this increment does not relabel that earlier check as a new full backend run. Native-language/assistive-technology acceptance, live browser/backend integration and operator prerequisite setup remain pending.


# Live local browser/API verification plan

Next verification uses the actual built console and production HTTP handlers on one isolated loopback test listener. Existing configuredDraftWorld supplies synthetic independently governed setup and an approved source before the browser starts. Bind admission and the exact loopback origin remain enforced. Synthetic session cookies are delivered to an isolated browser process through an in-memory stdin pipe, never files/logs/tickets; session/CSRF and all command checks execute normally. No real account, customer data, external delivery, runtime bypass endpoint or production configuration change is introduced.

Root owns test-only Go harness and console browser runner. Acceptance: submitter selects the actual approved final request and creates a draft through the UI; a separate reviewer session discovers it, chooses actual returned terms, saves review and approves; Go then reads the actual module and verifies its independent approver and coverage. This closes a live network integration gap but does not certify login, operator setup entry, persistence, native language or production deployment. Tests are explicit opt-in and require an existing built console and installed browser runner.


# Live browser/API result and next operator scope

The opt-in live harness now serves the actual rebuilt console and production API handlers from one isolated loopback listener. No browser requests are intercepted. The initial run correctly refused approval-list access because the synthetic draft submitter lacked approval-view permission. Root made that read-only human/client grant an explicit test prerequisite; production rights were not changed.

The rebuilt console then completed actual selected-final creation, separate reviewer discovery, explicit term selection, review saving, approval and inspection. Go read back exactly one draft/review, the selected configuration/source, USD 16 at scale 2, the configured independent reviewer and exact module coverage. The final live test passed with race instrumentation in 9.319s. The harness passes synthetic session cookies through stdin only and adds no authorization bypass endpoint. Independent source review accepted the isolation and assertions; it did not run the test. This establishes one EN desktop live network journey, not login, initial operator setup, persistence or issuance execution.

Next scope is the [initial reusable operator setup](issuance-operator-setup-plan.md). Inspection corrected the earlier O03 note: its contract list/detail are live; documents/payment-terms/recognition views are explicitly unconnected. Extend the first two existing views, derive authoritative contract/profile versions and avoid monthly reconfiguration. Meaningful evidence provenance input and public setup disclosure/recovery still require concrete design review.
