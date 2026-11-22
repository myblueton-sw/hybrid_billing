---
type: ImplementationPlan
title: Reusable issuance configuration and per-invoice evidence
description: Correct the monthly setup burden before exposing the draft workflow.
status: draft
---

# User necessity and evidence

HYB-341, machine:mac_mini. Root owns implementation; platform_refusals supplied the read-only corrective design and refusal_review owns independent design/change review. The owner requires extensions to be necessary from the user's perspective. An unchanged contract must not require new profile governance and template-channel maturation every month.

[FA-12](../../architecture/data-ownership/source-relationship-bindings.md) identifies scoped IssuanceProfile revisions and effective-dated contract-item mode bindings. [EI-01](../../contracts/billing/source-erp-and-issuance-obligations.md) governs changes for unissued scope. The [B09 journey findings](issuance-draft-journey-plan.md) identify the current source-bound profile coupling. This correction precedes public draft submission/context; it does not change financial approval independence or imply issuance/delivery readiness.

# Immutable format boundary

Introduce a separately versioned reusable configuration record and per-invoice application. Existing source-bound profiles, reservations and digests retain their historical meaning; never replace their Source, reinterpret their hashes or silently adopt their module kinds. New records need a separate store or explicit format discriminator checked before every read/write. The existing source-bound path remains legacy evidence, not the public recurring-billing model.

A configuration contains workspace, configuration identity/revision, mode/profile/region/tax/zone/languages, exact authoritative contract and billing-profile revisions, customer/issuer and relevant party snapshots, currency/document path, actual kind, independent governors, reason/digest/state. It excludes invoice period, final request, result, amount and financial source digest. Select contract and billing profile; derive scope from actual organization records. Exact field/version APIs and applicability digest require independent review before code.

A per-invoice application contains configuration identity/revision/digest and the complete fresh financial issuance source, with its own digest. Ordinary draft submission creates it after current draft operation plus inspection rights, confirmed configuration integrity/current governor standing, actual kind readback, fresh final-source validation and exact applicability comparison. This is not a second configuration approval. Different billing periods are permitted; contract/profile revision, party, currency, document path, mode and zone mismatches refuse rather than infer compatibility.

# Configuration governance and stable kind

Use independent configuration proposal/confirmation and actual Define/ApproveKind. Deterministic kind identity belongs to the configuration identity/revision, not an invoice. Reserve bounded identity and applicable scope before possible module mutation, preserve exact same-actor uncertainty recovery and forbid foreign-kind adoption. Incorporate prospective identity reservation into this new configuration proposal rather than creating another user ceremony. Existing legacy reservation rules remain unchanged.

Confirmed configuration anchors the template verification channel before invoice approval. Reuse the mature channel and protected template change while configuration remains applicable; never manufacture channel age. Changed configuration must have an explicit governed revision/effective-date rule and must not alter already approved/issued history. The exact selector, overlap and activation rules are open design items; do not implement implicit latest-wins or retroactive activation.

# Consumer changes

New-path draft generation consumes configuration kind/languages plus application source amounts/parties. New approval checks the application's own financial source. Template resolution uses the stable configuration kind. Recipient/account checks and reusable payment bindings use stable authoritative applicability, not the original invoice's current financial approval. Frozen reads retain the exact configuration/application snapshots and never refresh A from B.

Do not create a synthetic financialIssuanceProfile by replacing Source while retaining an old digest. Extend protected-channel subject selection and protected/term consumers with explicit new-format applicability adapters. The configuration approval grants no draft operation right and the application grants no financial approval.

# Owned scope and acceptance

Root-owned changes: configuration/applicability/application types and governance, bounded adapter stores/initialization, new configuration-kind channel selector, protected/term new-format consumers, draft construction/submission/review path and focused actual fixtures. No public API or screen implementation until the corrected contract is independently reviewed. No explanatory source comments, foreign data, generated artifacts or unrelated refactors.

Acceptance: create and independently confirm one configuration and admit one template channel; process A; expire A's financial approval; produce B's fresh final evidence; submit and independently approve B with the exact same configuration digest, kind, template channel/change and payment binding. Assert no additional configuration governance/kind registration, current B source validity and unchanged A historical records. Refuse applicability changes, foreign/corrupt ownership and lost rights. Exercise interrupted configuration and draft operations without changing legacy records.

# Current review state

Read-only corrective proposal received. Exact authoritative applicability fields, format-discriminated consumer API, revision/effective-date overlap policy and uncertainty boundaries remain pending executable independent design review. No implementation or acceptance is claimed by this document.

Independent document review accepts this as a safe draft checkpoint, not implementation approval. Authoritative applicability inputs are Workspace with active state/version; exact ContractRevision; BillingProfile with active state/version; sorted deduplicated active Party snapshots/versions for referenced parties and applicable legal parents; and server-derived DocumentKey. DocumentKey alone does not establish active-party eligibility. Preserve the correspondence checks already present in approvalnormalized.go: customer/buyer, seller/issuer, billing revision and compatible currency/path. Mode/zone/languages remain explicitly governed configuration, not inferred organization policy.

For executable review, the smallest proposed initial restriction is one confirmed configuration revision per exact applicability scope, effective for new applications from actual confirmation time. No backdating, automatic latest selection, overlapping activation or silent replacement. Invoice period is not activation time. This is a proposed implementation boundary rather than a source policy; scope key, digest schema, initial-revision validation, confirmation timestamp and collision/unknown recovery still require concrete agreement before code.

# Executable initial-configuration proposal

Root proposes the following bounded private contract for independent review. This is a first reusable configuration version, not a complete replacement/effective-date editor.

`financialConfigurationSpec` contains ID, Revision, Contract, ContractRevision, BillingProfile, BillingRevision and the existing governed mode/profile/region/tax/zone/languages. Revision must equal 1 in this increment; unsupported replacement/revision requests refuse before reservation. The scope key is a versioned digest of workspace, contract identity/revision and billing-profile identity/revision. Mode, language, currency or party differences cannot create another competing configuration for that same scope. No automatic selection across revisions is introduced.

`financialConfigurationScope` contains the actual Workspace/version, immutable ContractRevision, BillingProfile/version, DocumentKey and sorted deduplicated Party/version snapshots for customer, seller, issuer, buyer, payer, recipient and their legal parents. Every referenced party must be active in the same workspace; legal parents must be active legal entities. Missing optional payer/recipient values are preserved rather than replaced. Require exact contract customer/buyer and seller/issuer correspondence, supported external-invoice path and valid currency. Proposals derive these facts from organization APIs under the server domain lock. No final approval is needed to configure them.

`financialIssuanceConfiguration` retains schema version, workspace/spec/scope, derived Kind, proposer/reason/digest, mutable state/confirmer and full actual module snapshots. Its immutable digest excludes mutable state, confirmer, module snapshots and activation timestamp. Kind identity uses a separate configuration identity/revision digest namespace, avoiding collision with legacy profile IDs. Store bounded configuration and reverse-scope ownership separately from legacy profiles/reservations; retain no references to caller-owned slices.

Proposal authorizes the live-session protectedRuleCaller wrapper and profileCaller, validates all inputs/current scope, checks capacity/reverse ownership and actual KindOf absence before reserving an unknown intent. Actual Define success is copied into the retained record before any post-module fault. Same-actor exact retry accepts only complete retained module readback; an absent module permits retry only when no module snapshot was ever observed. Unknown lookup/foreign equal kind/missing observed module refuse without adoption or recreation. Known no-write failures retain their reserved intent; a new ID cannot escape scope ownership.

Confirmation authorizes the current independent governor through the same live-session wrapper, verifies exact immutable configuration/reverse scope/current organization snapshots/current proposer standing, and compares actual KindOf with the retained proposed snapshot. Pin the confirmer before possible ApproveKind. Retain complete active result before post-module fault, then publish confirmed. Only the pinned confirmer may recover. Activation is the actual retained active module timestamp; new applications require a current clock at or after that instant. Confirmed replay returns the same frozen active snapshot after current governance/configuration checks; historical readback remains separate and does not mutate state. No caller can backdate activation.

Fault tests surround intent reservation, module write and publication for both phases, plus lookup unavailability. Match full KindView, including actors/state/time. Validate wrong authorized actor, changed organization revisions/party standing, foreign equal kind, reversed/corrupt ownership, missing observed module, limits and defensive copies. Setup must succeed in an otherwise real sample world before any financial candidate/final/profile/draft is created by this flow. Legacy constructor sample evidence is not misreported as new setup output.

After this governance foundation, add an explicit new-format per-invoice application (configuration identity/revision/digest plus fresh source) and new-format protected/term/draft adapters. Exact consumer changes remain independently reviewed before execution; no fake legacy profile may bridge formats. Full acceptance remains A/B with unchanged configuration and mature channels, not this foundation alone.

# Executable governance review

Independent refusal_review accepted the private initial-configuration governance and focused-test scope. KindView.At exists and is set by actual Define/ApproveKind; same-approver replay preserves the activation time for this closed nonstatutory kind. Confirmation recovery compares the full retained proposed snapshot before mutation, or the full retained active snapshot after observed success. An unobserved active result is never adopted. Missing observed state is never recreated. Proposer replay after confirmation preserves the confirmed record. Validate reverse ownership in both directions and activation no earlier than proposal time. This review does not approve consumer integration or claim the A/B acceptance test has passed.

# Private governance implementation checkpoint

The separate versioned configuration and reverse-scope stores now support initial revision 1, independent proposal/confirmation, actual organization snapshots and a stable configured-kind namespace before this flow creates any financial candidate, final approval, legacy profile or draft. The actual external-invoice organization API requires a legal-entity payer; an omitted recipient remains omitted. No fallback recipient or payer is invented. Configuration scope is independent of invoice period and financial approval digest.

All eight interruption boundaries retain ownership and observed full module snapshots. Tests cover independent wrong actors, same-scope alternate identities, foreign identical kinds, unavailable lookup versus absent observed module, unobserved active module state, changed organization versions/standing, reverse-map and timestamp corruption, copying and bounded capacity. Service-map capacity tests inject full maps deliberately; module capacity uses 100 actual Define calls. A known module capacity refusal retains its reserved intent.

Independent source/test review identified missing recovery-authority coverage. Added actual pinned-confirmer grant revocation and ended-session regressions. The session test reproduced a defect: profileCaller alone accepted the retained identity answer after transport session removal. Both new entrypoints now use the existing protectedRuleCaller wrapper to recheck the live human session and identity before governor admission; it has no protected-rules-record dependency. Revoked-governor tests assert the actual private inputauthority.ErrDenied sentinel; ended sessions assert commands.Unauthenticated. Exact retained ownership and active module snapshots remain unchanged on refusal. Independent corrective source review passed; the reviewer did not execute tests.

Final focused configuration race PASS 5.335s. Legacy issuance-profile and empty-assembly race PASS 8.692s. Server vet and diff checks passed. Earlier 2.458s/4.848s configuration results predate the added session regression and do not certify that correction. No new full-server, public API, browser or rendered design result is claimed.

Next user-visible acceptance still requires the explicit new-format per-invoice application, stable protected-template and payment-term consumers, and B using A's exact configuration/kind/channel after A's approval expires. This private foundation does not yet remove the monthly setup burden in the screen. Initial revision replacement/effective-date editing, public setup/discovery, draft transport and B09 journey integration remain open. No legacy source-bound records were reinterpreted; no issuance, numbering, delivery, merge or deployment occurred.
