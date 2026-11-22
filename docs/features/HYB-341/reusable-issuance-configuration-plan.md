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
