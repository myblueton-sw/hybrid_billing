---
type: ImplementationPlan
title: Authenticated candidate-backed charge approval transport
description: Actual-session request and human decision commands with immutable provenance and separate review permission.
status: draft
---

# Scope and ownership

Continue HYB-340 on machine:mac_mini after the request-permission checkpoint. Root is the sole implementation writer; independent refusal_review reviews financial/security design and changes, platform_refusals reviews terminology and evidence. Owned files are new server financialcharge command/boundary/binding files and tests, narrow command registration/dispatch/assembly/hold/step-up integration and these ticket records. No new permission seed, external effect, module-policy replacement or owner screen acceptance is authorized.

# Request and immutable provenance

Expose financial_charge.request with gate.RequestApproval on charge_approval, requiring both actual human and ConsoleClient rights. Before any lookup or replay, recheck the actual caller session and permission. The target is a new financial request ID; only candidate and digest are wire fields. Rebuild financialSubjectLocked under the domain lock immediately before module Request. Require the owned current single-approval policy. The server derives original author, exact amount/currency, economic target and required evidence; caller overrides are forbidden.

Store a bounded workspace/request binding with candidate ID/digest, complete immutable subject, requester and policy revision. Preflight sidecar capacity and maximum result representation before Request, which can supersede existing requests in the same economic scope. Refuse adoption of an existing module request without service binding. Existing IDs require identical immutable provenance and requester; do not append duplicate bindings. Same-key receipts are historical and still require current session/request permission. Fresh-key calls rebuild current producers and cannot silently reuse stale evidence.

# Decision and review boundary

Expose only financial_charge.approve and financial_charge.reject as actual human financial approval commands. Verify their registered definitions use Approval=true and gate.Approve on charge_approval; only these exact commands may clear the server-supplied ConsoleClient. Preserve CSRF, session, step-up, hold and global revision boundaries and all other command clients.

Admit the actual session before a bounded internal workspace-scoped binding/original-author lookup. Missing, foreign, corrupt or unauthorized bindings have the same nondisclosing public refusal, without revision or metadata. This internal lookup is necessary to call actual CheckApproval, never a placeholder author or generic Approve. Check requester exclusion as well as original-author exclusion before replay. Resolve returns the actual immutable author; the command registry retains CheckApproval/RecheckApproval. New effects require exact retained request/subject/policy plus rebuilt current candidate provenance. Do not use CheckCurrent as a pending request validator. Module amount coverage is mandatory for approve only; reject preserves the existing reducing-decision semantics while requiring actual approval permission and separation.

Expose financial_charge.review as a nonmutating command protected independently by existing approval_review/View/approval_evidence permission plus its console intersection. Neither RequestApproval nor Approve grants this read permission. Return bounded recorded request identity, exact amount/currency, original author/requester, policy and evidence digest fields, explicitly labeled historical evidence rather than current eligibility. No raw candidate, supplier Cost or source rows are returned. Mutation receipts contain only IDs, digest and recorded state, not the review payload. Current producer checks occur on fresh mutations, not implicitly on historical review.

# Projection boundary

Independent review found that the existing B09 index accepts only gross basis, while rating.Result.Total does not prove tax-inclusive gross semantics. This increment will not append new requests to that index or label their amounts gross. Existing B09 read behavior remains unchanged; actual new requests are reachable through the separately authorized bounded review command. B09 listing/action integration and a truthful amount-basis contract remain explicit subsequent work under the same full goal. Even without indexing, retain single-approval policy scope until multi-approval review and decision behavior are separately integrated.

# Independent design review

Independent refusal_review accepted this concrete plan after the gross-basis indexing deferral. Implementation must retain a closed module rejection reason with separate human rationale, refuse fresh-key revival of terminal requests and account for pending requests in bulk-impact discovery.

# Verification and review conditions

Cover real separately provisioned requester and approver, no defaults, original-author/requester exclusion, amount/currency/date limits, stale source/input/pricing/policy, nondisclosing missing/foreign lookup, current permission on same-key replay, exact receipt history, fresh-key revalidation, supersession and concurrent decisions, preflight without partial effects, bounded provenance and no duplicate bindings, unchanged legacy command clients, SeedNone, trust hold and step-up. Use actual independently governed synthetic USD15 candidates. Run focused race, relevant shared regressions and independent re-review. Screens, downstream execution and full W-23 are not completed by this transport.
