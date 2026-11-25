---
type: ImplementationPlan
title: Result authority provisioning through M03 and P02
description: Connect explicit result authority proposals and independent decisions without broadening generic grants.
status: draft
---

# Scope and evidence

HYB-341, machine:mac_mini, branch feat/HYB-341-erp-result after bb72a86. Root owns planning, implementation and integration; refusal_review independently reviews rights and protocol boundaries; platform_refusals reviews terminology/evidence; actual Claude CLI ui-designer leads visual composition and rendered review. The original product goal remains active. No merge, deployment, real-account grant or external ERP authorization is implied.

B09 now consumes separately governed result capabilities. M03 and P02 currently display localized result grants but do not create the dedicated authority proposals. Existing generic access request forms must continue excluding this domain. Six public result_authority commands already implement propose_console, propose_human, approve, reject, withdraw and inspect through inputAuthorityCommand with fixed domain 6. They require current installation governance, exact session/console checks and existing mutation protections. The response is historical_decision, not a statement that the grant is currently usable.

Inspection requires a known proposal ID. There is no public discovery/context command for proposals, available owned result-domain ceilings or eligible recipients. The shared registry already exposes sorted, defensive Records(workspace); serverGrantAuthority owns exact bindings and ceilings and verifies actual grant effects through Lookup/Current. These existing stores are authoritative. Do not derive ownership from a prefix or expose another authority registry.

# Intended user flow

M03 provides a visibly separate result-authority panel within its grants workflow. It must work from installation governance independently of ordinary generic-grant offers. A governor can review existing proposals, prepare an explicit console ceiling or human grant and inspect historical decisions. The five capabilities remain separate: propose, confirm, revoke, internal apply and protected Cost inspection. No combined role or implicit grant is offered.

Console recipient is fixed to the actual console client. Human proposals require an independently selected active human recipient, an exact owned active matching result ceiling and validity contained within that ceiling. A recipient cannot propose or approve their own human grant; a distinct current governor approves. Review shows subject, capability, workspace-wide scope, validity, ceiling identity and rationale before dispatch. Values are not granted merely by selecting them. Rejection and withdrawal do not revoke an already-issued grant; existing explicit grant-revocation flows remain separate.

P02 clients view links its relevant console ceiling context to the M03 workflow. B09's permission-management link reaches that panel. Navigation grants no authority; copied IDs are validated against current server reads. URLs carry only nonsecret stable identifiers and navigation state, never source packets, reasons, credentials or amounts. Root will confirm whether the existing grants view needs an explicit subview selector after visual design review; do not silently repurpose generic grant fields.

# Read contract to review before execution

Add result-domain-only governor-protected reads. Supply bounded sorted proposal discovery and current provisioning context from the actual registry, identity facts and owned console ceiling effects. Use exact target workspace, closed collection/filter schemas, ten rows per page, a coherent revision/time snapshot and fail-closed continuations. Changing scope, capability, snapshot or session clears prior candidates and decisions. Validate output bounds before serializing; do not truncate a decision rationale or publish a partial mixed collection.

Keep proposals and current options distinct. A historical applied receipt does not mean an active grant; historical revoked/expired effects remain readable. Unknown decision effects remain unresolved and must not be replaced by a new proposal. Inspection alone must never issue a grant or reconcile an unknown effect. Capability choices and lifetime/validity limits must come from the server contract, not frontend guesses.

Current action eligibility is deliberately unchecked by these read APIs. History-derived controls open a review only; commands revalidate current authority and pinned facts. No advisory evaluator or mutation simulation is introduced.

# Execution and verification boundary

Expected root-owned changes: dedicated server result-authority read handler/tests and narrow command/hold/output integration; isolated frontend result-authority contract/state/forms under identity/access; M03 composition and P02/B09 navigation; scoped tests/runners and this ticket evidence. Avoid changing shared domain semantics or another authority's public contract without an explicit reviewed need. The read contract and visual composition remain under review; no implementation acceptance is claimed by this plan.

Verify current governor loss, foreign workspace/domain/ceiling, expired or revoked ceiling, recipient/proposer exclusion, stale revision/snapshot, hold and reauthentication, output and paging limits, changed session and late results. Exercise actual console proposal/second-governor approval, human proposal/independent approval, usable result rights, rejection/withdrawal and interrupted decision outcomes. Preserve immutable attempt identities and distinguish historical decisions from current effective rights. Render EN/KO/AR desktop/tablet/mobile, keyboard/review/error states and navigation through actual service entry. Independent source, terminology and Claude rendered corrective reviews precede the scoped commit/PR update. Default-runtime and other-OS claims require their own evidence.

# Independent investigation and selected contract

refusal_review identified two critical constraints: pending records can project expiry without a service-revision change, and accurate approval readiness depends on hidden pinned registry fingerprints. Root accepts both findings. Pagination will bind a deterministic projection fingerprint, not only the service revision. The first implementation will explicitly defer current decision checks to submission rather than add an approximate readiness evaluator or expand the shared authority engine.

Add two read-only commands, governed by actual current installation authority before any lookup:

| Command | Target and request | Header fields | Rows |
| --- | --- | --- | --- |
| result_authority.history | Exact workspace; optional paired after/snapshot | mode, workspace, actor, current_eligibility=unchecked, snapshot, next | proposal, digest, kind, capability, recipient, state, proposer, proposed_at, expires_at, decision actor if present |
| result_authority.context | Exact workspace; required collection=recipients or ceilings and one of the five capabilities; optional paired after/snapshot | mode, workspace, actor, collection, capability, policy, current_eligibility=unchecked, snapshot, next | Minimal active human identity excluding current proposer, or exact active owned console ceiling with capability and validity |

Each page contains at most ten numbered row fields. Policy is a closed JSON object with the five capabilities and server-configured proposal lifetime/max grant validity durations. Headers plus rows fit the configured output limits or the entire read is refused. History row summaries omit rationale; the existing inspect command supplies the complete protected record without truncation. Continuations bind actor/workspace/domain/collection/capability and a canonical hash of the full bounded projection and policy. Time-dependent expiration/ceiling eligibility must alter that projection even without a revision change. Reauthorize every page. A different snapshot invalidates the collection; no partial mixed rows are published.

Recipient discovery must use active human/effective membership facts, not accessFacts.eligible, which is restricted to installation administrators and would omit valid ordinary recipients. Ceiling discovery verifies the exact result registry ceiling, bindings, actual sameGrant/Lookup effect and active status; unrelated domain grants never become choices. Malformed/orphan associations fail closed. Lists are historical/options evidence, not mutation permission or assurance of approval.

For pending records, show review controls only where historical actor/state rules permit and label current checks as deferred to submission. Current server checks remain authoritative. Unknown reconciliation uses a distinct explicit review path for the original decision actor, preserving proposal ID/digest and the identical recorded decision rationale; create a fresh HTTP attempt only after current inspection. Do not offer reject/withdraw on unknown outcomes. The frontend must distinguish identical HTTP retry from fresh-key reconciliation and must not infer active grant status from the recorded historical receipt. Exact found revoked/expired effects may settle history without recreating rights.

# External composition references

Root inspected the official [GOV.UK check answers pattern](https://design-system.service.gov.uk/patterns/check-answers/) and [select guidance](https://design-system.service.gov.uk/components/select/) on 2026-11-22. Adopt a separate readable summary before committing a grant intent, associated field hints and deliberate unselected choices. Prefer explicit capability choices and a bounded selectable recipient/ceiling list over a long dropdown. Reject the example's fixed content width, branding and implied ability to edit historical decisions. No third-party assets/code are imported. Actual Claude composition will place these patterns within existing M03/P02 IA and the full-width, neutral, local-Plex design constraints.


# Closed wire and construction bounds

The independent reviewer conditionally accepted the design and required explicit wire types, projection bounds and snapshot inputs. Root resolves those conditions as follows before implementation:

- Recipient rows have exactly user and login string keys. Only active Human users with effective active workspace membership are returned; current proposer is excluded. Login is display identity, never an authentication credential.
- Ceiling rows have exactly ceiling, capability, not_before and not_after strings. Time strings use canonical server RFC3339Nano. The domain-6 registry and actual grant identity must agree before inclusion.
- History rows have exactly proposal, digest, kind, capability, recipient, state, proposer, proposed_at, expires_at and decision_actor strings. decision_actor is an empty string when no decision exists; optional keys are prohibited.
- Policy has exactly capabilities (the ordered five-element string array), proposal_lifetime_seconds and max_grant_validity_seconds (canonical positive decimal integer strings). Retain the immutable constructor limits in the server wrapper; expose no inferred frontend policy.
- Snapshot is a lowercase 64-hex SHA-256 digest of canonical JSON containing fixed domain version, workspace, actor, command/collection/capability, policy and the entire sorted projected rows. Observation time and the service revision are excluded from the digest. Actual current authorization is rechecked separately on every page; the envelope service revision remains available for later mutations. Projected states/row inclusion change on time-driven expiry.
- after is the last row ID; a continuation requires both after and snapshot, validates exact membership of after in the current collection and rejects any snapshot mismatch. A first page supplies neither. No arbitrary ordinal offset or cross-filter cursor is accepted.
- Maximum complete projected history/ceilings/recipients is 1,000 rows. Reject over-bound registry/ceiling/member collections and stop user enumeration after detecting more than 1,000 users. Do not truncate a recipient population or claim it complete. Existing governor authentication retains its existing identity-read behavior; this limit bounds the additional provisioning projection, not an unimplemented global identity optimization.
- Each page has at most ten rows and a closed header; each JSON row and header must fit configured value/field limits. Oversized output is refused before any response fields are released. Missing, corrupt, foreign or orphan owned bindings fail closed. These reads never settle unknown outcomes or allocate new grants.

# Implementation and design checkpoint

Root implemented the two protected result-only reads and retained constructor policy limits. Initial HTTP tests exposed fixture activation/null-body mistakes and a missing declared EvidenceStale refusal; those were corrected without weakening the guards. Independent source review identified fractional timestamp loss through the shared seconds-only stamp helper. The new reads now use RFC3339Nano and an actual `.500`-second ceiling-to-human-proposal regression verifies exact containment. Applied console records and ceiling maps are checked in both directions. Corrective source review passed. Focused HTTP tests passed in 0.845 seconds before the final precision test; final result authority/permission race tests passed in 4.828 seconds. Final server vet passed. Full Go short regression passed all 83 packages, including server 147.650 seconds.

Actual Claude CLI initial design received an incorrect capture directory from root and explicitly did not inspect M03/P02 renders. Root corrected the evidence path to the existing result-rights-final captures; a completed follow-up inspected all four actual M03/P02 renders and retained the design with concrete layout revisions. Neither invocation is acceptance of the unimplemented provisioning screens.

Root accepts a separate `area=result` beneath M03 `view=grants`, with a separately mounted component that does not run generic access reads or mount GrantForm. P02 links only the actual console client; B09 links to the result area. The new area uses full-width list/detail composition, neutral separators, plain historical states, readable pre-send summaries and mobile stacked facts. Exact server validity boundaries remain visible and editable as UTC text including nanoseconds; localized display never changes the command instant. Proposal timestamps may have a secondary localized display, but exact values remain available in detail/review. Context observation time comes from the actual command envelope, not the snapshot hash. No unsupported total/page count or search is advertised. Frontend implementation and rendered acceptance remain pending.
