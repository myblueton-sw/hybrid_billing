---
type: ImplementationPlan
title: B09 draft public command and disclosure contract
description: Proposed bounded discovery and actor-owned draft operations for the existing approval inbox.
status: draft
---

# Scope and ownership

HYB-341, machine:mac_mini; root implements and integrates, refusal_review independently reviews financial/disclosure behavior, ui-designer and rendered review cover the later B09 composition. This proposal addresses the remaining public contract findings in [the journey plan](issuance-draft-journey-plan.md). It is not implementation approval. Reusable private application-backed drafts now support actual A/B approval with unchanged setup after A expires. The user outcome remains selecting an approved request, creating its derived draft and letting an independent colleague find and approve it without typing internal identifiers.

# Commands and immutable inputs

Reuse POST /api/v1/commands/{command}, existing CSRF/session/console validation, command envelope and command limits. All mutations require rationale, idempotency key and expected revision. Read requests retain the existing read envelope. No new generic dashboard/history endpoint is proposed.

| Command | Target | Body fields | Effect |
| --- | --- | --- | --- |
| issuance_draft.context | final request ID | digest; optional after_binding, snapshot | Read current context for exact final identity |
| issuance_draft.list | current workspace ID | optional after, snapshot | Read retained drafts, including unresolved submissions |
| issuance_draft.inspect | draft ID | optional after_line, snapshot | Read frozen draft and latest review; bounded line page |
| issuance_draft.submit | new draft ID | configuration, configuration_digest, final_request, final_digest, language | Submit application-backed draft |
| issuance_draft.prepare_approval | new review ID | draft, draft_digest, binding, binding_digest | Persist an exclusive actor-owned review |
| issuance_draft.approve | review ID | digest | Approve the exact prepared review or reconcile its owned interruption |

IDs and digests travel between server context and client state; they are never user-entered form fields. Submission exposes only the configured origin; legacy records remain inspectable and discoverable. No conversion or adoption occurs. Preparing approval is a mutation, not a harmless preview. The UI states that saving the review reserves the decision for that actor. There is no cancel, reassignment or takeover action in this contract.

# Admission and historical reads

Add issuanceDraftBoundary before s.commands.Execute, using actual current session, workspace, ordinary human identity, console rights and owned grants. All six commands require current Inspect; mutations additionally require Submit or Approve. Cached success must not disclose data after either required grant is lost. For a retained mutation identity, verify pinned actor and exact immutable request fields before cached replay; a different actor cannot replay or recover it. Resolving an approval target must not use the financial source author as the sole exclusion: retained draft checks also exclude submitter, final requester and all actual source contributors.

Read admission is separate from mutation-oriented draftCaller: it retains all session/gate/owned-grant checks but does not refuse solely because the installation is held. Explicitly allow only the three read commands through holdAllows. While held, context reports current eligibility as unchecked and readiness as installation_held; it does not refresh live source or imply a mutation is allowed. Mutations remain refused before cache under the existing hold boundary.

Create a pure retained-draft ownership check for submitted, submission_unknown and submission_refused. Validate origin/inner hashes, outer intent, map/reverse ownership and complete review chain without current financial/configuration eligibility. Submitted records use exact phase-aware module comparison. An unresolved record with no observed module accepts actual absence as an observation only; an unobserved actual module is conflict, not adoption. An observed snapshot requires full actual equality. Missing previously observed module is conflict. No read settles state, retries a module call, writes a review, or changes service revision.

Unknown retained status and current module observation are separate: missing, matching_unapproved, matching_approved, conflict or unavailable. Missing is expected only for never-observed submission uncertainty/refusal. Conflict/unavailable disable actions. An unknown approval with observed matching success remains unknown until its original actor explicitly reconciles it. Historical inspection always returns current_eligibility=unchecked, even when the recorded state is approved.

# Projection, ordering and exact bounds

Return map[string]string fields through the existing command outcome. Serialize each structured row/line separately with JSON; do not expose complete nested source/configuration graphs. Require at least 64 configured fields and 4096 bytes per value for this public adapter; otherwise return UnsupportedCapability before effects. Apply actual configured field/value limits and a proposed 65536-byte complete serialized fields-object cap on every response. Serialization/projection errors return an empty outcome.

Common scalar fields: mode=synthetic, current_eligibility, snapshot, next (or next_line). Identifiers retain their existing request-ID validation; digests are 64-character validated hashes; wire amount strings remain canonical exact decimals (for example 16), with an explicit source currency scale. The client formats them losslessly for display (for example 16.00 at scale 2), never through binary floating point. Enums are closed. Historical customer identity comes from retained source evidence. Optional names may come only from that retained configuration snapshot, with a maximum 256 UTF-8 bytes; historical reads do not require current organization/setup activity. For live context, names come from its validated scoped evidence; absent names are labelled identifiers, never invented business names. No credentials, raw instructions or confidential channel evidence are projected.

List returns at most ten row_00 through row_09 fields, sorted by immutable draft ID. Each row has draft, digest, customer_id, customer_name, period, currency, scale, total, submitter, submission_state, review_state, review_actor and submitted_at. Omit submitted_at when no actual module submission was observed; do not invent a timestamp for an intent. Include unresolved/refused records so reload can find their original immutable operation. Validate every scanned record before filtering. Snapshot is a digest of the sorted projected retained records and observed module phases, scoped to workspace; after must name a member of the same snapshot. Changed records or observations invalidate the cursor with EvidenceStale. Do not use service revision alone because private paths can update retained maps without advancing it.

Inspection returns draft identity/digest, configured or legacy origin kind, source final identity, customer/period/currency/scale/total/language, submitter/final requester, submission state, module observation, latest review identity/digest/state/actor/refusal, recorded receipt and current eligibility. A compact exact configuration reference is required for configured records; full governance graphs are not. Up to ten line_00 through line_09 fields contain index, text, exact amount and tax flag from the module draft; do not invent line IDs or quantity/unit fields absent from issuance.Line. Stable ordering follows immutable draft lines. after_line is the last returned zero-based decimal line index (0 through 199, canonical decimal encoding) and snapshot pins the complete inspection evidence and line array; unknown cursor or changed evidence refuses. All lines remain reachable across pages, up to the private 200-line bound. Page metadata reports line_count and next_line; never silently omit lines.

The three protected references use separately bounded reference_00 through reference_02 fields: field, change ID, reference ID/fingerprint and channel ID/provenance. These are explicitly reference identifiers, not fabricated account numbers, email addresses or template names. Coverage is a bounded structured field containing binding/term identity and revision, contract, issue date, due date, zone, net days and calendar mode. Readable business labels and final translated copy require the later terminology review.

Context separates historical final summary from current source eligibility, configuration readiness, applicable binding choices and caller permissions. A current Inspect denial yields no context. Missing Submit/Approve is an explicit capability=false, not proof that setup is missing. Resolve configuration against the exact current source contract/billing/document scope; zero matches yields configuration_missing, multiple applicable matches yields configuration_ambiguous. Never choose an arbitrary match. Validate the selected configuration's current standing and fresh source before ready. Missing or inapplicable terms yield terms_missing; adapter/evidence failure yields unavailable; expired source yields source_ineligible; an existing economic draft yields existing_draft with its exact identity. Existing unresolved ownership is not replaced.

Context returns at most ten binding_00 through binding_09 choices per page with ID/digest, term revision, net days, calendar mode, zone and effective dates; no silent truncation. Sort by immutable binding ID; after_binding is the last returned ID. snapshot hashes the exact context and all applicable choices. A changed context, unknown cursor or stale snapshot refuses with EvidenceStale; next_binding exposes remaining choices within the existing private 100-binding capacity. Choices require exact configuration, contract and actual current calendar coverage. Context exposes no payment reference amounts beyond the authorized final summary. Missing setup destinations are not linked to synthetic-only screens.

# Reload recovery descriptors

Inspection includes a bounded recovery field only after current Inspect admission. It contains operation, original_actor, business_id, eligibility (unchecked) and exact frozen arguments, never an original HTTP key, session token or CSRF value. For configured submission: configuration/configuration_digest, final_request/final_digest and language, with business_id equal to the draft ID. For prepare_approval: draft/draft_digest and the service binding ID/binding_digest from financialDraftReview, with business_id equal to the review ID. For approve: digest and the review business ID. Coverage's module binding ID must not substitute for the service binding ID. All descriptor fields are taken from validated retained origins/review identity, not refreshed context or current choices. Legacy records are explicitly legacy and do not receive a configured submission recovery descriptor.

Return descriptors for the relevant original actors' unresolved steps; the caller's may_recover boolean requires exact actor identity plus current operation rights, independent of the Inspect permission that permits observation. The UI obtains descriptors by selecting a discovered record. While the original HTTP envelope remains in session, identical-envelope retry remains available under existing command semantics. After reload, explicit recovery uses a new envelope/idempotency key with the same retained business identity and arguments. The UI never invents new business IDs to bypass a retained uncertainty. Prepared review ownership is visible even when no recovery action is available to the current viewer.

# Mutation output and uncertainty

Before any retained or module effect, construct and bound the possible success receipt and every inspection page from the proposed immutable draft/review. Bound maximum eventual actor/timestamp/state fields as in financialdocumentoutput.go. Reject too-large data before reservation. Submit receipt fields are draft, digest, submission_state, receipt=submission_record, current_eligibility=unchecked. Review receipts add review, review_digest, review_state, receipt=review_attempt or historical_approval. Receipts do not embed complete rows/references; inspection returns those.

Introduce invocation-local effect tracking around private submission/review operations: no_effect, retained_intent_changed, module_attempted, observed_publication_changed. Set it at actual write/call boundaries, never by inspecting unrelated pre-existing unknown state. Typed clean refusals remain clean when no effect was possible. A possible-write failure becomes the existing non-typed uncertain handler error, retaining immutable actor-owned evidence and the command registry's uncertain operation behavior. This includes module errors after reservation. Same-envelope retry follows existing registry semantics; a new envelope may reconcile the same business identity through explicit server readback. Never allocate a new business identity to evade uncertainty.

A purely historical successful replay may use the frozen receipt but still requires current operation/inspection and pinned identity. If actual module observation fails before any new effect, return the clean unavailable observation error and preserve prior uncertainty; do not claim it is newly created or settled. Private tests keep their current error semantics through a wrapper or explicit internal effect result, not a shared mutable server flag.

# Required acceptance and remaining review

HTTP tests must prove ordinary caller discovery, selected context, paged complete line inspection, actual submit/prepare/approve, both formats in reads, current grant loss before cached replay, ended session, wrong workspace and actor, read-only uncertainty observation, historical read while held/after expiry, stale/forged cursors, output cap rejection before mutation, all post-effect failure boundaries and clean refusal beside an older unknown attempt. Existing private A/B evidence is reused, not replaced by seeded module records.

Browser acceptance remains selecting without internal ID entry, authoritative reload recovery, independent reviewer discovery, explicit saved-review ownership, async selection/session invalidation, and source-scale amount/calendar rendering in EN/KO/AR. Setup authoring and governed draft grants remain necessary separate missing UI work; these six commands alone do not complete the user journey. The proposal awaits independent review, especially hold/disclosure policy, selection limits, output preflight and invocation-local uncertainty integration.

# Independent review ledger

Independent refusal_review found the public contract not yet executable; no public implementation is authorized by that review. Private implementation acceptance is separate. Root incorporated explicit reload recovery descriptors, canonical-decimal wire/scale display wording and historical-label independence after this review; these corrections still require re-review.

| Finding | Required next correction | State |
| --- | --- | --- |
| Reload recovery omitted exact frozen service binding/submission inputs | Bounded pinned-actor descriptors; distinguish fresh-envelope recovery from in-session retry | Proposed correction added; re-review pending |
| Response fields/enums/readiness precedence and cursor pairs remain prose-level | Complete required/optional DTO/state tables, including held/missing/unknown/unavailable responses and next_binding naming | Open |
| Inspection is not action admission | Specify current permissions versus source/setup readiness, exclusions, hold/step-up and fresh applicable choices before preparing a review | Open |
| Effect tracking lacks exact propagation and revision table | Define successful retained changes, known refusal publication, observed recovery, possible module effect and projection-error handling per boundary | Open |
| Canonical amount strings were described as fixed scale | Explicit canonical decimal plus scale and lossless display | Proposed correction added; re-review pending |
| Historical names could depend on active current setup | Use retained source/configuration labels and independent module observation | Proposed correction added; re-review pending |

The reviewer accepted held read-only access and conservative output admission as directions, not finished implementation contracts. The next root action is to resolve the open tables before public code, then obtain executable review. This document records the actual review result rather than claiming the user journey or screen design is complete.
