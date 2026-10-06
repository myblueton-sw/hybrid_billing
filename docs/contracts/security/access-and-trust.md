---
type: Security Contract
title: Analytical access trust changes and conditional execution boundaries
description: Specifies proposed query enforcement, authentication-root change, engine access, fetch and trained-artifact withdrawal contracts.
status: draft
---

# Access and trust contracts

Part of [HB-11 supplementation](../../reviews/HB-11-refinements/resolution.md); refines PC-03 and ED-01/02/03/06. These are proposed product contracts, not deployed controls, technology adoption or approved scope exclusions. Existing [identity](../../architecture/security-boundaries/identity-session-permissions.md), [collection](../../architecture/data-collection.md) and [intelligence](../../architecture/intelligence-layer.md) boundaries remain applicable. Infrastructure OS/DB root is an explicit customer-operated trust boundary, not a bounded product role.

## ED-01: mandatory query boundary

All product reads, including worker analytics, search/joins/counts/aggregates, caches, exports/recall and model retrieval, pass through an authorization-aware query boundary. The proposed default is a server query gateway with typed, approved query shapes; a database-policy alternative requires independent equivalence evidence before adoption. Caller SQL or a submitted tenant filter is never authority.

The server binds current actor/client, workspace/customer/resource/field grants, policy revision, fixed data revision, period, currency and amount basis. Validate permitted select/filter/group/order/join fields and evidence references; filter eligible inputs before aggregating. Errors, counts, names and hidden operands/denominators cannot disclose forbidden facts. Economic allocation denominators remain fixed; disclosure must use approved safe explanations instead of recomputing charges over visible customers.

Gateway credentials are server-held read credentials separated from projection writers. Product clients, export/model workers and ordinary administrators cannot directly query protected analytics stores or reuse writer credentials to bypass the gateway. Setup privilege is temporary and separate. Initialize context for every pooled checkout and verify cleanup; discard an unverifiably clean connection. Immutable economic revision does not freeze old grants. Sensitive disclosure fails closed when current authority is unavailable/stale; any low-risk cache exception requires separately approved bounded policy.

Bind jobs to fixed target manifests and current recipient authority. Recheck before work, at defined execution checkpoints and before download/output release. Reduced authority invalidates/restarts the original job or yields an explicitly reauthorized partial manifest; it cannot retain the original complete label. Exact execution mechanism/checkpoint intervals and runtime versions remain decisions. Protected output staging and current-authority download checks prevent a completed background job from bypassing revocation.

## ED-02: authentication-root change

Verified provider key rotation is distinct from issuer/key trust-root replacement, identity namespace/subject migration or MFA-policy weakening. Ordinary bounded membership administration authorizes none of those changes. Define a separate trust-policy proposer and independently authorized approver/activator; self-approval is prohibited. Installation-root custody and emergency recovery are separately governed, never silent finance approval.

Proposed states: Draft → Validated → IndependentlyApproved → Activated; Rejected, Cancelled and Superseded are terminal alternatives. Validation freezes old/new configuration digests, target scope, identity mapping, affected principals, authority/session impact, proof references, expected current revision and recovery evidence. A changed digest or mapping invalidates approval. Current authority and separation are rechecked at activation.

Prove legitimate new authority and each privileged identity linkage using existing trusted reauthentication plus new proof, or separately verified recovery when the old authority is unavailable. Email/name matching, or evidence vouched for solely by the proposed root, is insufficient. Named security/finance owners must approve proof and recovery procedures for the actual profile before activation.

Activation atomically advances configuration and authority epoch with durable invalidation intent. Old affected sessions, refresh families, step-up evidence and queued privileged commands cannot proceed without approved revalidation against the new epoch. Cross-node propagation failure cannot allow privileged stale use; fail closed. Rollback is another governed activation with a new epoch and cannot revive revoked authority. Activation approval already exists in local M02; this specifies trust replacement beyond ordinary policy/rotation.

## PC-03: external billing engine, if included

Only registered platform adapter service identities may reach approved engine APIs over bounded routes; engine-side caller authentication/authorization is required in addition to network restrictions. Credentials remain server-held and absent from browser bundles/storage, downloads and telemetry. Browser/customer/unregistered workers cannot directly invoke engine APIs.

Every invocation derives from a current authorized platform command with fixed input/policy revision and bounded scope. Return candidate results to platform validation/reconciliation; an engine cannot independently approve, consume Claims, issue documents or mutate ERP/financial authority. Credential/caller revocation stops new authorized dispatch; external uncertainty after dispatch requires reconciliation, not a blind retry. Engine selection and scoped exclusion are still owner decisions.

## ED-03: every retrieval uses the same fetch policy

Apply the existing AR-02/QA-05 destination policy to initial calls and all pagination, manifests, artifact references, webhook detail lookups, OCR resources and diagnostics. Bind permitted scheme/origin/port/path/source scope and credential audience. A source response cannot change workspace, credential or authority merely by supplying a URL.

Validate resolved addresses against approved public/private destinations and ensure the actual connection uses the validated destination while preserving TLS hostname checks. Revalidate redirected/retried requests and newly resolved addresses. Never forward credentials across origins by default; a distinct registered destination needs its own approved credential binding. Explicit private on-prem endpoints are supported only within reviewed scope. Deny cloud metadata endpoints, loopback and unapproved management networks, including indirect references and redirects; ordinary capability registration or review cannot override these prohibitions. Unclassified link-local destinations fail closed; explicit destination classification must still satisfy the metadata/loopback prohibitions. This preserves original B0189/B0725/B1428 rather than introducing a network exception.

Bound response bytes, decompression, reference depth, time, pages and retries with approved profile budgets. Exceeding a budget leaves partial/quarantined evidence and cannot publish completeness. Redacted diagnostics cannot reveal credential payloads. The first source/network profile supplies actual destinations/budgets; this document creates no connection or network exception.

The [planning command contract](../commands/planning-actions.md) supplies canonical input/trust requests, errors, independent approval and atomic epoch/publication transitions. M02 and S03/S06/B09 bind to that contract; a generic activation or confirmation flag cannot bypass it. Planned FP-03: allow an approved private on-prem destination while rejecting a metadata/loopback destination reached initially, through DNS changes or through a redirect/reference; unclassified link-local remains denied. FP-03 is NOT_RUN.

## ED-06: conditional training and withdrawal

Sensitive tenant-data training/fine-tuning is blocked until an independently reviewed, explicitly approved training and withdrawal contract exists. This is a prerequisite, not a claim the owner excluded training from product scope. Permitted scoped retrieval/inference does not grant training or pooled reuse rights.

If enabled, register weights/checkpoints/adapters and derivative artifacts with training-input/policy lineage, audiences, deployed versions and backup inventory. Withdrawal first marks all affected versions ineligible, fences activation/use and stops affected deployments; only then report enforcement effective. Unknown lineage remains ineligible. Current authorization still applies to features and output retrieval.

Replacement/retraining requires new eligibility evidence and approval. Restore consults independent latest withdrawal controls before serving artifacts, and cannot revive an ineligible version from backup. Retention/deletion applies to training inputs and managed trained artifacts separately. File deletion and unlearning are never promises of complete learned-information erasure; approved privacy policy determines obligations and evidence.

## Planned acceptance

All cases are NOT_RUN. Implemented enforcement, timing and actual profile compatibility require separate evidence after owner planning acceptance.

| Case | Independent expected outcome |
| --- | --- |
| QG-01 | Missing/tampered scope, raw query and protected-store direct product access denied; authorized typed query succeeds only within current grants |
| QG-02 | Restricted fields/identities cannot leak through filter/group/order/join/count/errors or model/export evidence links |
| QG-03 | Cross-customer pooled reuse has no previous scope; failed cleanup discards connection rather than exposing data |
| QG-04 | Revocation during export/recall/model work prevents subsequent unauthorized output; original completeness invalidated |
| TR-01 | Bounded admin trust substitution, same-email linkage, self-approval and changed-digest activation denied |
| TR-02 | Legitimate approved migration creates a new epoch; old sessions/refresh/step-up/queued privilege fail; outage/rollback cannot revive them |
| BE-01 | Conditional engine rejects browser/unregistered/forged/revoked callers; output alone creates no approval, Claim or issuance |
| FP-01 | Malicious follow-on URLs, DNS changes and redirects cause no forbidden request or credential forwarding; approved private endpoint stays scoped |
| FP-02 | Oversized/decompressed/recursive/partial retrieval stops with honest incomplete status |
| TW-01 | Unapproved training or unknown lineage cannot activate; withdrawal stops affected derivatives and restore cannot revive them |

Remaining decisions: actual query/credential mechanism and supported versions; tenancy/auth profiles, separately named authority/proof/recovery owners, session/revocation bounds; engine inclusion, endpoint/budget registry; training inclusion/audiences/privacy lifecycle. See [decision register](../../reviews/HB-11-refinements/resolution.md). No runtime or security certification is asserted.
