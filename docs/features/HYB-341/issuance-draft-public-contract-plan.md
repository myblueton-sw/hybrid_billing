---
type: ImplementationPlan
title: B09 draft public command and disclosure contract
description: Proposed bounded discovery and actor-owned draft operations for the existing approval inbox.
status: draft
---

# Scope and ownership

HYB-341, machine:mac_mini; root implements and integrates, refusal_review independently reviews financial/disclosure behavior, ui-designer and rendered review cover the later B09 composition. This proposal addresses the remaining public contract findings in [the journey plan](issuance-draft-journey-plan.md). The executable backend design was independently accepted after the corrective review recorded below; implementation and UI acceptance remain separately scoped. Reusable private application-backed drafts now support actual A/B approval with unchanged setup after A expires. The user outcome remains selecting an approved request, creating its derived draft and letting an independent colleague find and approve it without typing internal identifiers.

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

IDs and digests travel between server context and client state; they are never user-entered form fields. All three public mutations expose only configured origins; legacy records remain inspectable and discoverable. Before cache/effects, prepare_approval and approve reject legacy draft targets with UnsupportedCapability after current operation/inspection admission; submit against an already-owned legacy ID refuses EvidenceStale under shared identity ownership. Legacy records have no public recovery descriptors. No conversion or adoption occurs. Preparing approval is a mutation, not a harmless preview. The UI states that saving the review reserves the decision for that actor. There is no cancel, reassignment or takeover action in this contract.

# Admission and historical reads

Add issuanceDraftBoundary before s.commands.Execute, using actual current session, workspace, ordinary human identity, console rights and owned grants. All six commands require current Inspect; mutations additionally require Submit or Approve. Cached success must not disclose data after either required grant is lost. For a retained mutation identity, verify pinned actor and exact immutable request fields before cached replay; a different actor cannot replay or recover it. Resolving an approval target must not use the financial source author as the sole exclusion: retained draft checks also exclude submitter, final requester and all actual source contributors.

Read admission is separate from mutation-oriented draftCaller: it retains all session/gate/owned-grant checks but does not refuse solely because the installation is held. Explicitly allow only the three read commands through holdAllows. While held, context reports current eligibility as unchecked and readiness as installation_held; it does not refresh live source or imply a mutation is allowed. Mutations remain refused before cache under the existing hold boundary.

Create a pure retained-draft ownership check for submitted, submission_unknown and submission_refused. Validate origin/inner hashes, outer intent, map/reverse ownership and complete review chain without current financial/configuration eligibility. Submitted records use exact phase-aware module comparison. An unresolved record with no observed module accepts actual absence as an observation only; an unobserved actual module is conflict, not adoption. An observed snapshot requires full actual equality. Missing previously observed module is conflict. No read settles state, retries a module call, writes a review, or changes service revision.

Unknown retained status and current module observation are separate: missing, matching_unapproved, matching_approved, conflict or unavailable. Missing is expected only for never-observed submission uncertainty/refusal. Conflict/unavailable disable actions. An unknown approval with observed matching success remains unknown until its original actor explicitly reconciles it. Historical inspection always returns current_eligibility=unchecked, even when the recorded state is approved.

# Projection, ordering and exact bounds

Return map[string]string fields through the existing command outcome. Serialize each structured row/line separately with JSON; do not expose complete nested source/configuration graphs. Require at least 64 configured fields and 4096 bytes per value for this public adapter; otherwise return UnsupportedCapability before effects. Apply actual configured field/value limits and a proposed 65536-byte complete serialized fields-object cap on every response. Serialization/projection errors return an empty outcome.

All successful reads contain mode=synthetic, current_eligibility and snapshot. Cursor names are command-specific: next for list, next_line for inspection and next_binding for context. Identifiers retain their existing request-ID validation; digests are 64-character validated hashes; wire amount strings remain canonical exact decimals (for example 16), with an explicit source currency scale. The client formats them losslessly for display (for example 16.00 at scale 2), never through binary floating point. Enums are closed. Historical customer identity comes from retained source evidence. Optional names may come only from that retained configuration snapshot, with a maximum 256 UTF-8 bytes; historical reads do not require current organization/setup activity. For live context, names come from its validated scoped evidence; absent names are labelled identifiers, never invented business names. No credentials, raw instructions or confidential channel evidence are projected.

List returns at most ten row_00 through row_09 fields, sorted by immutable draft ID. Each row has draft, digest, customer_id, customer_name, period, currency, scale, total, submitter, submission_state, review_state, review_actor and submitted_at. Omit submitted_at when no actual module submission was observed; do not invent a timestamp for an intent. Include unresolved/refused records so reload can find their original immutable operation. Validate every scanned record before filtering. Snapshot is a digest of the sorted projected retained records and observed module phases, scoped to workspace; after must name a member of the same snapshot. Changed records or observations invalidate the cursor with EvidenceStale. Do not use service revision alone because private paths can update retained maps without advancing it.

Inspection returns draft identity/digest, configured or legacy origin kind, source final identity, customer/period/currency/scale/total/language, submitter/final requester, submission state, module observation, latest review identity/digest/state/actor/refusal, recorded receipt and current eligibility. A compact exact configuration reference is required for configured records; full governance graphs are not. Up to ten line_00 through line_09 fields contain index, text, exact amount and tax flag from the module draft; do not invent line IDs or quantity/unit fields absent from issuance.Line. Stable ordering follows immutable draft lines. after_line is the last returned zero-based decimal line index (0 through 199, canonical decimal encoding) and snapshot pins the complete inspection evidence and line array; unknown cursor or changed evidence refuses. All lines remain reachable across pages, up to the private 200-line bound. Page metadata reports line_count and next_line; never silently omit lines.

The three protected references use separately bounded reference_00 through reference_02 fields: field, subject, change ID, reference ID/fingerprint and effective timestamp, taken from the frozen review references. These are explicitly reference identifiers, not fabricated account numbers, email addresses or template names. Coverage is a bounded structured field containing binding/term identity and revision, contract, issue date, due date, zone, net days and calendar mode. Readable business labels and final translated copy require the later terminology review.

Context separates historical final summary from current source eligibility, configuration readiness, applicable binding choices and caller permissions. A current Inspect denial yields no context. Missing Submit/Approve is an explicit permission=denied, not proof that setup is missing. Resolve configuration against the exact current source contract/billing/document scope; zero matches yields configuration_missing, multiple applicable matches yields configuration_ambiguous. Never choose an arbitrary match. Validate the selected configuration's current standing and fresh source before ready. Missing or inapplicable terms yield terms_missing; adapter/evidence failure yields unavailable; expired source yields source_ineligible; an existing economic draft yields existing_draft with its exact identity. Existing unresolved ownership is not replaced.

Context returns at most ten binding_00 through binding_09 choices per page with ID/digest, term revision, net days, calendar mode, zone and effective dates; no silent truncation. Sort by immutable binding ID; after_binding is the last returned ID. snapshot hashes the exact context and all applicable choices. A changed context, unknown cursor or stale snapshot refuses with EvidenceStale; next_binding exposes remaining choices within the existing private 100-binding capacity. Choices require exact configuration, contract and actual current calendar coverage. Context exposes no payment reference amounts beyond the authorized final summary. Missing setup destinations are not linked to synthetic-only screens.

# Reload recovery descriptors

Inspection includes the bounded recovery_submit/recovery_prepare/recovery_approve fields defined in the exact table below, only after current Inspect admission. Each contains operation, original_actor, business_id, state, permission and exact frozen arguments, never an original HTTP key, session token or CSRF value. For configured submission: configuration/configuration_digest, final_request/final_digest and language, with business_id equal to the draft ID. For prepare_approval: draft/draft_digest and the service binding ID/binding_digest from financialDraftReview, with business_id equal to the review ID. For approve: digest and the review business ID. Coverage's module binding ID must not substitute for the service binding ID. All descriptor fields are taken from validated retained origins/review identity, not refreshed context or current choices. Legacy records are explicitly legacy and receive no recovery descriptors for any operation.

Return descriptors for the relevant original actors' unresolved steps; each descriptor's permission is allowed, denied or unavailable and requires exact actor identity plus current operation rights, independent of the Inspect permission that permits observation. The UI obtains descriptors by selecting a discovered record. While the original HTTP envelope remains in session, identical-envelope retry remains available under existing command semantics. After reload, explicit recovery uses a new envelope/idempotency key with the same retained business identity and arguments. The UI never invents new business IDs to bypass a retained uncertainty. Prepared review ownership is visible even when no recovery action is available to the current viewer.

# Mutation output and uncertainty

Before any retained or module effect, construct and bound the possible success receipt and every inspection page from the proposed immutable draft/review. Bound maximum eventual actor/timestamp/state fields as in financialdocumentoutput.go. Reject too-large data before reservation. Submit receipt fields are draft, digest, submission_state, receipt=submission_record, current_eligibility=unchecked. Review receipts add review, review_digest, review_state, receipt=review_attempt or historical_approval. Receipts do not embed complete rows/references; inspection returns those.

Introduce invocation-local effect tracking around private submission/review operations: no_effect, retained_intent_changed, module_attempted, observed_publication_changed. Set it at actual write/call boundaries, never by inspecting unrelated pre-existing unknown state. Typed clean refusals remain clean when no effect was possible. A possible-write failure becomes the existing non-typed uncertain handler error, retaining immutable actor-owned evidence and the command registry's uncertain operation behavior. This includes module errors after reservation. Same-envelope retry follows existing registry semantics; a new envelope may reconcile the same business identity through explicit server readback. Never allocate a new business identity to evade uncertainty.

A purely historical successful replay may use the frozen receipt but still requires current operation/inspection and pinned identity. If actual module observation fails before any new effect, return the clean unavailable observation error and preserve prior uncertainty; do not claim it is newly created or settled. Private tests keep their current error semantics through a wrapper or explicit internal effect result, not a shared mutable server flag.

# Required acceptance and remaining review

HTTP tests must prove ordinary caller discovery, selected context, paged complete line inspection, actual submit/prepare/approve, both formats in reads, current grant loss before cached replay, ended session, wrong workspace and actor, read-only uncertainty observation, historical read while held/after expiry, stale/forged cursors, output cap rejection before mutation, all post-effect failure boundaries and clean refusal beside an older unknown attempt. Existing private A/B evidence is reused, not replaced by seeded module records.

Browser acceptance remains selecting without internal ID entry, authoritative reload recovery, independent reviewer discovery, explicit saved-review ownership, async selection/session invalidation, and source-scale amount/calendar rendering in EN/KO/AR. Setup authoring and governed draft grants remain necessary separate missing UI work; these six commands alone do not complete the user journey. Independent executable review accepted these backend contracts after the corrective tables below; boundary implementation and browser verification remain required.

# Independent review ledger

The initial independent refusal_review found the public contract not yet executable. The findings below were subsequently resolved by the accepted corrective tables. Private implementation acceptance remains separate.

| Finding | Required next correction | State |
| --- | --- | --- |
| Reload recovery omitted exact frozen service binding/submission inputs | Bounded pinned-actor descriptors; distinguish fresh-envelope recovery from in-session retry | Closed by corrective executable review |
| Response fields/enums/readiness precedence and cursor pairs remain prose-level | Complete required/optional DTO/state tables, including held/missing/unknown/unavailable responses and next_binding naming | Closed by corrective executable review |
| Inspection is not action admission | Specify current permissions versus source/setup readiness, exclusions, hold/step-up and fresh applicable choices before preparing a review | Closed by corrective executable review |
| Effect tracking lacks exact propagation and revision table | Define successful retained changes, known refusal publication, observed recovery, possible module effect and projection-error handling per boundary | Closed by corrective executable review |
| Canonical amount strings were described as fixed scale | Explicit canonical decimal plus scale and lossless display | Closed by corrective executable review |
| Historical names could depend on active current setup | Use retained source/configuration labels and independent module observation | Closed by corrective executable review |

The initial reviewer accepted held read-only access and conservative output admission as directions. The later corrective review accepted the executable tables below. This document records the actual review result rather than claiming the user journey or screen design is complete.

# Executable response and action tables

These tables supersede earlier descriptive wording for field names, descriptor schema, states and cursor pairs. Independent corrective review subsequently accepted this executable backend design; implementation verification remains scoped separately below. Wire scalar values are strings; structured values use JSON objects with the exact keys below. Omit optional keys rather than emitting ambiguous empty objects. A present cursor terminator is the empty string. All counts/indices/scales/revisions use canonical nonnegative decimal strings. No ad hoc keys are allowed.

| Response | Always present fields | Conditional fields |
| --- | --- | --- |
| context | mode, final_request, final_digest, final_state, customer_id, period, currency, scale, total, current_eligibility, readiness, submit_permission, approve_permission, submit_action, prepare_action, approve_action, step_up, snapshot, next_binding, binding_count | customer_name if retained label exists; configuration/configuration_digest/languages JSON array only for one validated applicable configuration; existing_draft/existing_digest only when an owned economic draft exists; row fields binding_00..09 for current applicable choices |
| list | mode, current_eligibility=unchecked, snapshot, next, count | row_00..09 for this page |
| inspect | mode, current_eligibility=unchecked, snapshot, next_line, line_count, draft, digest, origin, final_request, final_digest, customer_id, period, currency, scale, total, language, submitter, final_requester, submission_state, module_observation, receipt | customer_name when retained; submitted_at only if module observed; configuration/configuration_digest for configured origin; review JSON and reference_00..02/coverage only if latest review exists; recovery_submit/recovery_prepare/recovery_approve as specified below; line_00..09 |
| submit receipt | mode, draft, digest, submission_state, receipt=submission_record, current_eligibility=unchecked | none |
| prepare/approve receipt | mode, draft, digest, submission_state, review, review_digest, review_state, receipt, current_eligibility=unchecked | none |

context historical summary uses the exact final record and its retained candidate evidence; it does not require current eligibility. If that retained identity/integrity cannot be verified, return EvidenceStale with zero fields instead of an invented partial summary. Scale and canonical total come from retained validated financial evidence. customer_name is optional in all rows. The required summary never depends on a current active organization lookup.

| Structured object | Exact required keys | Optional keys |
| --- | --- | --- |
| list row | draft, digest, customer_id, period, currency, scale, total, submitter, submission_state, review_state, module_observation | customer_name; review_actor when review_state is not none; submitted_at when observed |
| binding option | binding, binding_digest, term_revision, net_days, calendar_mode, zone, from, to | none |
| line | index, text, amount, tax (JSON boolean) | none |
| review | id, digest, actor, state, reviewed_at | refusal only in refused state |
| protected reference | field, subject, change, ref_id, fingerprint, effective | none |
| coverage | contract, binding, term, revision, net_days, calendar_mode, zone, issue_date, due_date | none |
| recovery descriptor | operation, original_actor, business_id, state, permission, arguments (object) | none |

recovery_submit is present only for configured submission_unknown/submission_refused and contains the exact five submit argument keys. recovery_prepare is present only for a configured draft with the latest prepared or approval_unknown review and contains the exact four prepare argument keys; replaying prepare observes the saved attempt but does not execute approval. recovery_approve is present for that same review and contains only digest. Each descriptor carries its own permission=allowed/denied/unavailable key, replacing the aggregate recovery_permission field in inspect. No aggregate recovery permission field is returned. Historical approved/refused reviews expose no unresolved-action descriptor. A caller may inspect another actor's descriptor but permission is denied. Even allowed is current authority only, not current business eligibility or permission to change arguments. Client recovery still uses the normal mutation boundary.

| Enum | Values |
| --- | --- |
| origin | configured, legacy |
| submission_state | submitted, submission_unknown, submission_refused |
| review state | none (list only), prepared, approval_unknown, approved, refused |
| module_observation | missing, matching_unapproved, matching_approved, conflict, unavailable |
| current_eligibility | unchecked, eligible, ineligible, unavailable |
| permission | allowed, denied, unavailable |
| action | ready, permission_denied, authority_unavailable, installation_held, reauthentication_required, authentication_policy_unsatisfied, source_ineligible, setup_missing, setup_ambiguous, evidence_unavailable, existing_draft, draft_required, independent_reviewer_required, owned_by_other, unresolved_operation, review_required, recheck_required, completed, legacy_read_only |
| readiness | installation_held, unavailable, source_ineligible, configuration_missing, configuration_ambiguous, terms_missing, existing_draft, ready |
| step_up | satisfied, reauthentication_required, authentication_policy_unsatisfied |
| receipt | submission_record, review_attempt, historical_approval |

Inspection receipt is historical_approval only when retained latest review.state is approved; otherwise review_attempt if a review exists, else submission_record. A matching approved module behind an unknown review does not change this value. Integrity failure is an empty EvidenceStale outcome; module lookup failure is a valid historical projection with module_observation=unavailable. Retained hashes/chain are always checked first.

For every paged command, its input cursor and snapshot must be both absent or both present; any one-sided pair is InvalidRequest. An initial call computes a snapshot even for zero rows. snapshot excludes cursor/current page and includes complete retained projection, ordered full rows/lines/choices, module observations and, for context, current permissions/actions/readiness. Snapshot includes workspace and caller identity; no cross-viewer cursor reuse. Stale or unknown cursor is EvidenceStale. Context next_binding, inspection next_line and list next are always returned, including empty terminal values. Snapshot does not include request IDs or a clock sampled per page; derived date/eligibility changes still invalidate it.

# Fresh action readiness

Inspection is historical only. The client must request context using the inspected final_request/final_digest before showing a new saved-review action. Context resolves existing draft ownership by economic target. For an existing configured draft, use its exact retained configuration and application identity; do not reselect another configuration. Current source/configuration must equal that retained application's snapshots. A legacy draft returns legacy_read_only for new public mutations. Submitted state and exact module observation are prerequisites for new review preparation.

Context computes submit_permission and approve_permission using current session/console/owned grants independently of source/configuration readiness. A missing grant is denied; unavailable authority is unavailable, never denied. Inspect admission failure returns an empty error. The existing stepUpCheck is read-only and these new commands are ordinary sensitive actions, with no trust-candidate target-dependent policy. Evaluate it for the three command names; report the most restrictive result. Mutations always recheck it; context is advisory, not a reservation or authorization token.

Action precedence is deterministic: operation permission denied/unavailable; installation hold; failed step-up; retained integrity/module conflict/unavailable; caller-independent source eligibility; setup missing/ambiguous; then workflow rules below. Readiness is separate from permissions: hold first, unavailable integrity/dependency second, source_ineligible third, configuration missing/ambiguous fourth, terms_missing fifth, existing_draft sixth, otherwise ready. While held, eligibility=unchecked and omit live configuration/choices; retain only validated historical summary and owned draft reference. When source validation fails due to expired approval or lost source-governor standing, eligibility=ineligible, readiness=source_ineligible. Genuine dependency/authority uncertainty produces unavailable; never label it as missing configuration. Valid source yields eligible even if terms/configuration are missing, because source eligibility and setup readiness are separate.

| Operation | Workflow rule after common current admission checks |
| --- | --- |
| submit | No existing economic draft and one current applicable configuration: ready. Existing draft: existing_draft; recover it through inspection. Terms are not required to submit an amount-derived draft, so terms_missing does not disable submit_action. |
| prepare | No submitted draft: draft_required. Current caller equals submitter/final requester/source author/contributor: independent_reviewer_required. Latest prepared/unknown review: owned_by_other or unresolved_operation; latest approved: completed. No review or latest refused, current three references and at least one applicable binding: ready. Missing terms/references: setup_missing. |
| approve | No latest prepared/unknown review: review_required or completed. Different actor: owned_by_other. Original actor with matching prepared facts and live source/setup: ready. Original actor with resolvable current facts that differ from the saved references/coverage: recheck_required. Original actor with observed unknown module success: unresolved_operation, use explicit recovery descriptor; do not relabel as a new approval. |

For missing configuration/terms and unavailable/held context, binding_count=0, next_binding="" and no binding rows; submit/prepare/approve actions explain the applicable constraint independently. Context with a prepared review pins its exact binding when checking approval; it must not silently substitute a newly applicable option. Never refresh retained review facts on reads.

# Invocation effect and service revision table

Pass an invocation-local financialDraftEffect pointer into tracked internal variants of submitDraft, prepareDraftApproval, approveDraft and refusal publication; existing private entrypoints delegate without changing their error contract. The tracker has monotonic booleans retained_changed, module_attempted and observed_publication_changed. Set retained_changed only after an actual owned map/chain value differs and is stored; mark module_attempted immediately before invoking the authoritative module. Mark observed_publication_changed when recording or reconciling an actual returned module snapshot/state. No global tracker and no classification from unrelated existing unknown state. A known reference/coverage refusal is a successful retained state change and must set retained_changed.

| Invocation result | Public outcome | Revision handling |
| --- | --- | --- |
| Clean validation/admission/lookup/preflight error before any effect | Empty typed refusal; preserve existing uncertainty | No Next call and existing changed() returns false |
| Successful new intent, module effect, observed reconciliation or known refused-review publication | Bounded receipt, business state, ResultRevision=s.Next() | Existing execute changed() advances once |
| Successful same-business replay in a fresh envelope with no new private effect | Same frozen bounded receipt with ResultRevision=s.Next() | Existing command semantics advance once for the accepted new command; do not claim a new business effect |
| Same-envelope cached replay | Existing registry replay receipt after current boundary checks | changed() excludes Replay; no new revision |
| Error after any tracked retained change/module attempt/publication | Empty non-typed errFinancialDraftUncertain | Registry retains operation ID/NextOperation; execute changed() advances once, without an extra Next call |
| Unexpected projection failure after private effect | Same uncertain error; retain all owned evidence | Same once-only uncertain revision path |
| Read success/failure, including missing/conflicting/unavailable module observation | Bounded frozen projection or empty typed error | Never Next and never private recovery |

Preflight must construct every potential future public page using bounded maximum actor/time/state/descriptor values before the corresponding mutation. Preflight of approval includes both approved and known-refused outcomes; prepare includes original binding recovery arguments. Check exact JSON serialization size for all fields/individual values/aggregate. Public wrappers translate errors using the invocation tracker; errors with no tracked effect remain mapped typed refusals even when an older unknown record is present. This table intentionally preserves existing command registry operation/revision semantics rather than adding a second revision counter.

# Saved-fact drift and legacy mutation closure

For a configured draft, context compares current resolved references/coverage with the pinned prepared review only after normal live admission. A difference returns approve_action=recheck_required for the original reviewer. The UI offers an explicit saved-review recheck action with a confirmation explaining that it runs the actual approval attempt: if current facts still differ, the server records a refused review and a fresh review can be saved; if the facts match at execution time, it can approve. This is not cancellation, a dry-run or a promise to refuse. It sends the same approve command with the exact retained review ID/digest and a fresh envelope. The existing atomic private comparison remains authoritative. If current source/references/coverage cannot be resolved, return the appropriate ineligible/unavailable action; do not fabricate a drift refusal or silently release ownership.

Legacy profiles are retained compatibility evidence only in the public boundary. List/inspect disclose them under Inspect, but no recovery descriptor or ready/recheck action is emitted. Prepare/approve reject a legacy origin before cached replay or effects, including a legacy review ID. Direct private legacy tests/entrypoints remain supported. This restriction does not convert, hide or overwrite old records; later legacy operator migration would need its own reviewed contract.

# Executable review and first read implementation

Independent corrective design review PASS for the six-command backend scope after the exact DTO/action/effect tables, original-reviewer recheck path and configured-only public mutation policy were added. This closes the proposal findings above for implementation, not for rendered UI or setup usability.

The first implementation registers only issuance_draft.list and issuance_draft.inspect. Separate current read admission permits historical inspection during hold while preserving session, console and actual owned Inspect grants. The before-cache server boundary performs that admission before command resolution. Pure retained ownership validates inner/outer evidence, shared scopes, exact derived monetary draft and review chains; module observations never adopt or reconcile an unknown operation. Frozen recovery descriptors retain exact service binding and application inputs and exclude legacy mutation descriptors. Lists and lines are paged with caller-scoped evidence digests and bounded JSON output. Context, three mutations, mutation preflight/effect tracking and B09 composition remain unfinished.

Independent source review identified incoherent failed states: refused submissions could carry observed modules, and review refusal text was not validated despite exclusion from the intent digest. Root reproduced HTTP200 disclosure in corruption tests. Corrections require a fully empty module for submission_refused and exactly the two supported review refusal reasons, with no refusal text in other states. Corrective source/test review PASS for the list/inspect scope; reviewer executed no tests.

Actual HTTP tests create 11 distinct synthetic monthly sources through normalization, independent financial approvals and draft submission in one unchanged world, including six independently declared positive net/tax source rows yielding 12 monetary lines. They verify all ordered drafts and all exact lines, cursor rejection, unresolved module observations without settlement, frozen review service identities, expired/held historical inspection and lost Inspect rights. These synthetic periods include future months; they are pagination evidence, not a claim of completed real billing periods. Test helper default two-row behavior is retained. Initial fixture failures were a null HTTP body, a canonical 2.5 amount expected as 2.50 and an incorrect expected HTTP status for EvidenceStale; fixes changed test inputs/expectations only.

Root intermediate results: public read/bounds tests PASS1.442s, initial public race PASS8.993s, legacy/configured/draft regression PASS11.554s. Final public plus same-world/expiry fixture race PASS33.686s; public admission/caller-cursor race PASS19.201s before fixture session cleanup. Broader server PASS334.317s was built before the two retained-state guards and later test-only supplements; it is not a final-source full-suite claim. The first expanded race failed at repeated fixture session creation. Helpers now reuse and close only their own sessions through actual logout, without changing limits or resetting the world; independent fixture review PASS. Final draft/configured-draft/legacy-serialization race PASS100.900s at the final source/test state. Server vet and staged whitespace checks PASS. No screen rendering, public mutation or full user-journey completion is claimed.
