---
type: ImplementationPlan
title: Initial reusable issuance setup within the selected contract
description: Evidence-backed scope for completing operator prerequisites without repeated invoice setup or typed internal identifiers.
status: draft
---

# Outcome and ownership

HYB-341, machine:mac_mini. Root owns scope, implementation and integration; independent financial/security review precedes new public disclosure or mutation contracts, and actual Claude CLI ui-designer leads changed screen composition. This plan continues the complete issuance journey rather than treating the preconfigured B09 test as operator readiness.

An operator selects an existing named contract, establishes reusable issuance configuration and protected references, agrees payment terms through the required independent approvals, and returns to B09. Later applicable invoices reuse that setup. No monthly recreation of configuration, channels, references or terms is required. No real issuance, delivery, payment, merge or deployment authority is added.

# Verified existing ownership

O03 already has live contract list/detail in `apps/console/src/modules/organizations/pages/contracts/Live.tsx`. Its `LiveDetail.tsx` explicitly marks documents, payment-terms and recognition views unconnected. The earlier blanket statement that O03 is synthetic-only was too broad. The documents and payment-terms views own this increment; recognition is not needed to establish these prerequisites.

The existing contract DTO supplies human-readable contract/profile identities, parties, currency, timezone and lifecycle/term information. It does not supply the billing-profile revision needed by issuance configuration. The DTO lifecycle revision comes from the lifecycle position and references contract terms; it must be resolved and checked against the exact organization contract revision on the server. It is distinct from lifecycle journal sequence, page concurrency revision and billing-profile version. Resolve authoritative revisions server-side; never ask operators to copy their IDs/revisions into another form.

The detailed historical O03 specification remains unavailable in the authorized trees. This plan proposes a separately reviewed extension grounded in the existing live screen, the owner-authorized planning contracts and actual service boundaries. It does not reconstruct or claim acceptance of the missing baseline.

# Minimum required workflow

| Existing owner | User task | Existing private boundary | Required behavior |
| --- | --- | --- | --- |
| O03 documents | Select supported issuance settings and document languages for the selected contract | proposeConfiguration / confirmConfiguration | Derive contract/profile/party/currency scope; independent confirmation once for unchanged setup |
| O03 documents | Discover existing protected rules and establish them if absent | proposeProtectedRules / confirmProtectedRules | Expose actual governance; no seeded default policy or administrator bypass |
| O03 documents | Register and independently admit applicable verification channels | registerChannel / admitChannel | Show channel type, provenance, independent admission and actual maturity; do not shorten the waiting period |
| O03 documents | Request, verify and approve protected references | requestProtectedChange / verifyProtectedChange / existing decision operation | Derive subjects from configuration; preserve evidence, effective time and independent actors |
| O03 payment-terms | Record and independently approve calendar net days | recordTerm / approveTerm | Explicit days and rationale; no Net30/45 default or invented business-day calendar |
| O03 payment-terms | Select an approved term and independently bind its applicability | bindTerm / approveTermBinding | Derive contract/timezone, show effective dates and exact term revision; reuse later |
| M03 access | Establish missing governed business rights | Existing authority domains, public gaps to map | Separate current Inspect, Submit/Approve and setup rights; administrator role is insufficient |
| B09 | Create and approve the next invoice draft | Existing six public draft commands | Consume completed reusable setup and fresh source; preserve independent review |

Configuration region/tax/timezone choices must come from actual supported contracts and profile applicability, not guessed legal defaults. Protected fields remain immutable references/fingerprints; the UI must not present them as actual bank account or email contents.

# Discovery before public writes

The next executable design must define a bounded, contract-scoped setup read with exact server-resolved contract/profile versions, retained configuration and pending work, protected-rule availability, applicable channels and maturity, protected-change states, approved terms/bindings, and caller-specific action readiness. Historical facts and current action eligibility stay separate. Reads never reconcile unknown effects or change ownership.

Use existing session, CSRF, permission, step-up, installation hold, revision/idempotency and actor-owned recovery boundaries. Define disclosure rights for each field before implementation; a contract-view grant alone must not disclose protected evidence. Fresh recovery of retained unknown work must be possible for the original actor without rewriting its business identity. Reuse B09 client behavior only where the actual setup contract matches.

The live B09 test exposed another required permission: its submitter also needs approval-view access to select an approved final request. Draft-submit permission alone is not sufficient. The test now explicitly provisions that read grant. Operator guidance must explain the necessary combination rather than hiding the request list or silently widening grants.

# User necessity and unresolved evidence input

New IDs are generated internally; existing contracts, profiles, terms and eligible channels are selected with meaningful labels. Preserve user input during prerequisite reads. Explain one actual blocking prerequisite and its real destination at a time; do not add a separate administration dashboard.

A meaningful source for selecting provenance/evidence references is still missing from the inspected public workflow. Replacing that gap with unexplained internal-ID/fingerprint boxes would not complete ordinary operator work. Before exposing protected setup writes, inspect existing evidence ownership and either integrate a verified source or design explicit human evidence capture with its real verification semantics. Do not equate an uploaded file, computed hash or typed URL with successful independent verification.

Configuration replacement/effective revision support is separately constrained by the current initial-revision implementation. Do not silently imply that an existing confirmed configuration is editable. Required later revisions must preserve issued history and receive their own concrete contract review.

# Acceptance and limits

The complete operator journey must start without fixture-seeded issuance configuration, protected rules/channels/references or payment terms and finish with B09 ready for the appropriate independent actors. Its second applicable invoice must reuse unchanged setup. Verify missing rights, actor separation, maturity, stale revisions, interrupted writes/recovery, hold behavior and absence of fabricated current eligibility.

Public contract review, concrete O03 design, implementation, logic verification and independent design re-review are pending for this plan. The owner will perform screen verification; root must retain design re-review, as clarified after the owner assigned browser verification to themselves. Do not report source/design review as rendered verification or owner acceptance. The already passed B09 live test deliberately supplies governed setup beforehand; it does not satisfy this operator acceptance criterion.

# Explicit discovery authority prerequisite

Source inspection found no Inspect capability in `modules/termsauthority` or `modules/protectedauthority`; their current capability sets authorize writes only. Contract-view, administrator status, draft Inspect and a write grant must not become substitutes for a new setup disclosure right.

The proposed prerequisite adds separate Inspect capabilities to those existing authority domains, with distinct resource kinds and evidence fields, using the existing independently approved human grant and console ceiling. It adds no seeded grants, implicit migration or new disclosure endpoint. Existing mutation mappings and immutable proposal digests must remain compatible. Independent financial/security review must approve the capability and generic-grant exclusion changes before implementation.

Logic verification must exercise actual independently approved ceilings and human grants, absence of default access, cross-domain and cross-capability denial, rejection of generic grant bypasses, and immediate loss of access after revocation. A later contract-scoped read must separately prove current session and field authority, bounded output, scope resolution, pure held reads, unknown-effect preservation and absence of sensitive evidence for unauthorized sections. This prerequisite alone does not establish operator readiness.

The owner requested find-skills discovery for design re-review. The installed Vercel web-design-guidelines skill was matched through the Skills CLI and its official source; use it alongside the existing Hybrid Billing completion loop and Claude ui-designer requirement. Apply the project-pinned guideline baseline and review changed scope only. No new skill installation or completed design review is claimed by discovery.

# Selected-contract scope resolution

Source discovery by platform_refusals, confirmed against `contracts.go` and the configuration scope validator, establishes an existing server-owned association: contract metadata selects the billing profile, and `contractRead` resolves the lifecycle position for an explicit date. Its position revision selects the exact organization contract revision. Do not select the largest revision or infer a profile by matching party names.

The proposed scope resolver accepts a selected contract ID and explicit as-of date. Under the domain lock and the eventual endpoint's current authorization, require scoped contract metadata, a nonempty linked profile, an active lifecycle position with nonzero terms revision, the exact organization revision and the current linked billing-profile version. Use `configurationScope` to validate their correspondence, active workspace/profile/parties, external-invoice path and document key. Missing or mismatched source evidence refuses setup; no repair or guessed default is permitted.

The returned context digest must bind the selected date, lifecycle sequence/position, exact contract/profile versions and authoritative configuration scope. A subsequent proposal re-resolves and compares that digest before effects. Names and metadata timezone are presentation context, not proof of issuance eligibility or a substitute for the separately governed document zone. A historical selection must not silently turn into today's setup. These are proposed new integration requirements, not a claim of implemented disclosure or completed contract review.
