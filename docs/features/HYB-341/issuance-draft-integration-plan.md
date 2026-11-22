---
type: ImplementationPlan
title: Governed issuance profile and retained draft integration
description: Compose actual protected and receivable registries with current final-approved source evidence without issuing a document.
status: draft
---

# Outcome and ownership

HYB-341, machine:mac_mini. Root owns service integration and tests; refusal_review is independent read-only financial/security reviewer. Deliver a retained issuance draft whose monetary source, document mode/profile, protected references and payment coverage come from actual owned registries. This extends the source adapter at3eea122 and module admission atc1a08a5. It is not satisfied by another detached calculation or a mock collaborator. No real customer data, external issuance, delivery or automatic posting is authorized.

Owned implementation scope: new apps/api/server/financialissuance*.go adapter/config/profile/draft files and focused tests, the necessary Server field/initialization and scoped resource/authority integration, plus this plan and verification. Public command and screen contracts require their own concrete review before exposure. No UI composition is part of the private integration slice.

# Registry ownership and construction

A service-owned adapter holds actual protected.Registry, receivables.Registry and issuance.Registry instances. Initialize empty within SeedSample; SeedNone remains unsupported until production policy/persistence exists. Constructors receive bounded limits and the existing named exact decimal profile; constructors do not create rules, channels, protected changes, terms, kinds, profiles, drafts or grants. Module collaborators resolve only these owned registries. Rendering is disabled in this slice, and the adapter exposes no Issue, RecordExternal, Render, Download or delivery method. All adapter calls and provider changes use the server domain mutex; module locks remain internal.

An unavailable producer must refuse before publication. Private adapter handles are not an authorization mechanism and must not escape through a public response. Actual runtime modules may return historical replay; service wrappers recheck current session, governance and owned input identity before accepting it.

# Profile proposal and independent confirmation

A typed profile proposal explicitly selects issuance mode, document kind/region/tax/zone/language, applicable contract/billing revisions and document-party scope. It includes a rationale and exact canonical digest. Do not infer mode from ExternalInvoice, ERP availability or a result application timestamp. The first source path currently supports only synthetic ERP invoice evidence; self issuance, accounting reference and internal statements must remain distinct later paths and cannot be silently substituted. A synthetic non-statutory ERP profile is labelled as such; it establishes no legal/tax readiness.

Use the existing issuance kind registry for Define/ApproveKind and preserve service-owned proposal bindings. Actual installation governance is required for both proposer and independent confirmer, including current proposer checks before confirmation/replay. Never adopt an unrelated module kind or seed by ID. Check scope, capacity, closed mode/profile shape and all text/array bounds before the first module write. Deterministic owned IDs and retained proposal states must make partial registration recoverable rather than implicitly successful. Language has no effect on mode, jurisdiction, currency or amounts.

Initially permit one confirmed immutable mode/profile per exact unissued economic scope. A second conflicting proposal cannot become a shortcut for switching mode. Later profile changes need an explicit effective-date/unissued-scope protocol and must preserve issued history; do not claim that future support here. Profile approval grants no issuance permission.

# Protected and receivable inputs

Compose actual protected rules, independently admitted verification channels, requested changes, independent verification and approvals for recipient/template and applicable payment/tax references. Verification evidence is a separately supplied synthetic test fact in fixtures, never generated from an LLM assertion or the request itself. Production service wrappers must check actual callers and scope at each governance step. Store reference IDs/fingerprints only; do not place contacts, accounts or credentials in repository artifacts or tickets.

Use actual receivable RecordTerm/ApproveTerm and Bind/ApproveBinding/Cover. Missing, pending, conflicting, unsupported-mode or stale coverage refuses draft approval. No fallback net days or inferred timezone. The issuance module resolves references and terms at approval and again at current admission. Tests must exercise those actual registries rather than a resolver stub that always returns a successful value.

# Retained draft and independent approval

Separate draft submission and decision authority from final financial approval. Define a dedicated resource kind and exact current gate actions before registering any operation; no inherited final-request grant or installation-admin fallback may authorize draft submission/approval. No default human or console grants. Actual-session checks precede producer reads; decision actors must be independently authorized and excluded from self/contributor approval as defined in the concrete command design. Read access to monetary draft data requires separately declared Cost fields and scoped permission.

Submission reads financialIssuanceSourceLocked, resolves the service-owned confirmed mode/profile, and derives issuance.Draft customer/entity/contract/currency/scale/lines/total from the frozen source. Preserve each reported row and tax association in a retained sidecar; do not flatten away line identity or substitute a caller amount. Display labels must come from governed metadata or explicit identifiers, not invented product names. Convert decimal representation only through the named exact profile and prove equality to the approved gross. Include selected profile ID/digest, source digest/final request/digest/result receipt and exact generated draft digest in one owned binding.

The issuance module Submit can revise drafts and returns historical matches. The wrapper must forbid ID adoption, reject different immutable source/profile reuse, and retain explicit revision ownership before any allowed change. Approval revalidates current source/profile/submitter standing and current independent decider rights, then invokes actual module Approve so protected references/term coverage are captured. Bind the resulting references, coverage and approval identity into the service review digest. A later reference/term change invalidates current issuance admission; it must not silently rewrite a previously approved service binding.

# Publication and recovery boundary

Before exposing a draft as retained or approved, reconcile the exact module record and service binding. Capacity, canonical input and output-size checks precede writes. A write whose outcome cannot be established remains owned/unknown and blocks a competing ID or mode; no optimistic receipt. The concrete implementation must list failure points between module insertion/approval and service binding publication and specify exact same-actor recovery.

This slice does not invoke approvals.Execute or issuance effects. The subsequent transaction must reserve one immutable service execution identity, run complete current financial/issuance checks and CheckIssue/CheckExternal, then Execute → chosen effect → exact document read-back → Report. Interrupted effects retain the original owner and identity. An external number/time requires independent external issuance provenance and cannot be manufactured from the retained result. Document rendering, disclosure packages, delivery channels and delivery attempts remain separate follow-ups.

# Acceptance

Verify actual authenticated actors and real registries through profile proposal/independent confirmation, protected change verification/approval, approved term coverage, source-derived draft submission and independent draft approval. Refuse foreign scope, unowned matching IDs, missing role/console ceilings, changed profile/source/terms/references, revoked proposer/decider rights, self/contributor approval and capacity before mutation. Exercise each publication interruption and same-owner recovery; assert no number, issued document, execution reservation, rendering or external call occurs. Current rendered work remains B09; new user controls require actual Claude design and EN/KO/AR rendered review after backend contracts are settled.

# Review boundary

Independent reviews now cover source-profile configuration, prospective kind reservation, private governed terms and protected rule/channel/change writes. See payment-terms-service-plan.md and protected-issuance-service-plan.md for actual verification and limits. Service-level reference consumption and retained issuance drafts remain unimplemented. Their separate permission actions, independent actor exclusions and frozen approval recovery require concrete review before implementation.

# First approved implementation slice

Independent review permits empty real-registry assembly and narrowly scoped profile proposal/confirmation after recording these conditions. First profile is explicitly synthetic_erp_document_v1, mode erp_issuance, non-statutory ClassERP. It never establishes tax readiness. Define/ApproveKind reuse the actual issuance registry, with deterministic proposal-owned kind IDs and no adoption of existing unowned kinds. The profile binds exact current source digest plus workspace/contract/billing revisions, document parties/currency and stable economic target. Conflicting profiles lock by economic target, not proposal ID or changed revision.

Installation governance is allowed only for this configuration slice: caller must be an actual current human governor, confirmation must be independent, and the original proposer/current confirmer are rechecked on replay. This grants no protected-field/term/draft/issuance authority. No public command is registered here. Rendering and detail collaborators explicitly return unavailable; no successful renderer stub is installed.

Reserve profile capacity, economic scope and exact immutable intent before Define. States are registration_unknown, proposed, confirmation_unknown and confirmed. Before-write injection produces no reservation; after-module-write interruption retains the unknown state and ownership; after-publication interruption retains the published state but returns failure. Same original proposer may reconcile exact proposed kind after registration uncertainty. Confirmation reserves the intended independent confirmer before ApproveKind; only that actor may reconcile confirmation uncertainty. Any mismatch, changed source/current governor or competing identity refuses while retaining the original lock. A known unchanged module replay does not overwrite scope or references. Unknown operations cannot be cleared or re-created under another ID.

Larger draft work is not yet approved: it still needs distinct direct-human draft submit/approve permissions, consumption checks separate from the implemented protected/term write governance, explicit exclusion of submitter/source contributors/final requester, frozen reference/coverage approval recovery and issuance-time term rechecks. Module draft Approve can rebind references on a later call, so the service must reconcile exact frozen approval facts rather than invoke it blindly on changed inputs.

# Profile implementation and independent review

The first private configuration slice now composes the actual empty protected, receivables and issuance registries during sample initialization. Non-sample initialization leaves the adapter absent. Profile proposal and independent confirmation retain the approved source and immutable configuration digest, bind the economic target, and reconcile exact owned module kinds after interrupted registration or confirmation. There is no new public operation, default permission, issuance draft, document issue or delivery effect.

Independent source review found the initial constructor's decimal import outside the existing server source allowlist. Root moved constructor assembly into the already-allowed approvalseed.go, retaining the candidate decimal profile rather than substituting the distinct approval profile. The guard was not widened. Corrective independent source review passed for this slice. Review did not certify runtime tests or approve the larger draft contract. Public exposure must distinguish post-write uncertainty from a clean refusal before any effect.

Focused profile tests passed1.484s. Combined issuance/source-rule tests passed2.137s, and race verification with empty-registry/SeedNone coverage passed12.199s. Final focused tests passed2.217s after adding exact changed-source/governor refusal assertions and wrong-confirmer refusal during uncertain recovery. Server vet passed. The full server regression passed198.856s, covering the changed sample startup assembly and existing server behavior.

The tests exercise actual public final request/approval, separate human governors, defensive copies, unchanged financial approval execution records, six interruption boundaries, exact module kind states, foreign-kind refusal, economic-scope locking, revoked original proposer governance, changed result receipt and same-actor recovery. They do not establish crash durability across process restart, production installation readiness, actual customer contacts, statutory readiness or completed issuance/delivery. Protected and term governance, distinct draft permissions, retained draft approval and execution remain required next work.


# Concrete frozen-draft investigation

Read-only investigation found no existing dedicated draft Submit/Approve actions or authority domain. The next reviewed composition must add ordinary ceiling-plus-human grants without financial/protected/term permission substitution. Current change and term write validators reopen their original source/profile or require write capability, so they cannot directly serve draft consumption. Separate the new draft caller's current rights and financial source from retained configuration integrity, historical actors' current standing and stable source applicability. Reference reuse does not keep old financial approval current.

For current noncredit nonstatutory ERP drafts, derive kind/customer/issuer/contract/currency/scale and exact gross from the current confirmed source/profile. Require explicit supported language. Retain complete result-row identity and quantity/net/tax sidecar because module Line exposes only Text/Amount/Tax. Emit deterministic positive canonical net/tax components; preserve zero components in retained evidence without emitting invalid zero lines. Do not invent a Detail artifact or Original. Module Draft has no explicit date/timezone; actual coverage provides calendar semantics at approval.

The service resolver must compare actual Resolve change ID/reference/effective against the applicable owned scheduled full trail, current original protected actors/rules/channels and exact stable contract/customer, billing-profile/issuer or reserved-kind provenance of the new source. New kinds cannot inherit old-kind templates; equal customer IDs cannot erase differing contracts/revisions. The analogous term consumer must verify actual owned approved term/binding history, original actor standing and current contract/profile-zone coverage without requiring the draft caller to approve payment bindings.

Before Submit, retain immutable service intent and economic/ID ownership; refuse module ID adoption or revision under a changed spec. Before Approve, pin independent actor and expected complete draft/reference/coverage evidence. Freeze complete actual DraftView including revision/digest/actors/references/coverage/timestamp after exact readback. Already-approved replay reads that frozen result and must not invoke module Approve again, because the module may rebind without changing the draft digest. Reference or calendar-midnight drift between preflight and actual approval requires an explicit uncertainty/refusal protocol; never silently accept changed facts. This investigation is not executable draft-design acceptance or evidence of implemented draft consumption. Number reservation, actual issuance, rendering and delivery remain separate.
