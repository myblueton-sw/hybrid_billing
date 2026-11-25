---
type: Investigation
title: Current final approval to issuance and delivery integration boundary
description: Existing module evidence and unresolved producer prerequisites for the next original stage-chain increment.
status: draft
---

# Objective and ownership

Continue HYB-341 under HYB-225 after integrated commit24a481e. Root owns planning and implementation; refusal_review independently investigates financial boundaries, read-only. This is the next original stage-chain dependency, not a replacement of the full product goal. Machine:mac_mini remains active. No issuance, external confirmation, posting, approval consumption or delivery is performed by this investigation.

# Authoritative requirements

[HYB-225 plan](../HYB-225/plan.md) requires immutable producer evidence through charge result, applicable ERP-final reconciliation/amount, issuance confirmation and delivery. [EI-01 through EI-07](../../contracts/billing/source-erp-and-issuance-obligations.md) forbid changing issuer/mode for already-issued scope, fallback self-issuance on ERP failure, inferred posting success and a second customer document for accounting-only integration. Automatic posting starts disabled. Posted results require read-back against the approved snapshot; a mismatch enters correction review without rewriting history.

[FA-12](../../architecture/data-ownership/source-relationship-bindings.md) distinguishes self issuance, ERP issuance, accounting reference and internal chargeback modes. Mode/profile binding, protected customer/entity references, numbering, immutable snapshot and fixed delivery package are separate owned facts. Module availability does not establish a service-owned mode, current policy or verified external capability.

# Observed implementation

- The current service owns exact final approval subjects and current approved-consumer validation. That helper performs CheckCurrent, not Execute or issuance. Public result application records synthetic internal ledger evidence; it is not external issuance confirmation.
- modules/issuance requires Fields, Terms, Details, Renderer and Zones collaborators. Draft submission/independent approval bind protected references and receivable coverage. Issue allocates a number and records an immutable snapshot. RecordExternal separately accepts an external number/time but its module API alone cannot certify where those facts came from. A service caller must provide authoritative provenance and current permissions.
- modules/approvals Execute reserves a stage/target execution identity and keeps historical replay semantics. A service must revalidate current rights/evidence before its call and preserve an owned interrupted-operation boundary; CheckCurrent is not a substitute for recording the effect execution.
- modules/delivery requires a live disclosure package whose recipient matches a governed channel. Request and Approve are separate; duplicate package/channel and uncertain history remain explicit. Issuance PDF availability alone does not prove that customer disclosure and recipient authorization are satisfied.
- Current apps/api/server source search found no direct issuance, delivery, protected or receivables registry integration. Existing integration/recovery seeds describe other synthetic workflows and are not evidence of a completed final-approval-to-issuance chain.

# Next design questions

Map actual module APIs and source-owned mode/profile/protected-field/term prerequisites before selecting an implementable slice. Preserve ERP issuance versus internally generated documents; never fill missing protected fields, terms, templates or external confirmations with client assertions or unrelated seeds. Define the exact stage execution reservation, service-owned correlation, immutable artifact identity and recovery/read-back protocol together. Separate module prerequisites, actual sample-runtime integration, public permission governance and screen acceptance.

Independent architecture investigation completed with the findings below. No concrete downstream command contract or implementation is approved by this draft. Missing production tax/jurisdiction/vendor decisions remain explicit; they do not justify silently substituting an internal statement for the requested customer issuance path.

# Independent investigation findings

refusal_review recommends a governed retained issuance draft derived from owned final approval as the next coherent slice, followed by a separately designed execution/issuance transaction. It inspected issuance Submit/Approve, Issue/RecordExternal, protected Request/Verify/Approve/Resolve, receivables RecordTerm/ApproveTerm/Bind/ApproveBinding/Cover, approvals Execute/Report and disclosure-backed delivery. No tests or effects were performed, and the recommendation is not implementation approval.

The current final subject does not bind a prospective template, recipient, payment account, term coverage, language or issuance mode. Compose those actual governed registries and bind their resolved references to a separately reviewed issuance draft. Module human actors do not replace actual authenticated service permissions. Keep draft preparation non-consuming; a later effect boundary needs nonmutating admission/capacity checks and retained ownership across Execute, chosen issuance operation and Report. A domain mutex cannot roll back partial cross-registry effects.

Required acceptance includes changed references/terms, replaced results, revoked final approver rights/limits, independent actors, foreign identical IDs, each interrupted execution/issuance boundary, exact retry owner, capacity before effects and immutable issued history. External number/time must come from independently evidenced issuance, never the internal application timestamp. Rendering, disclosure and delivery remain separate later work. Root will turn this investigation into a concrete reviewed contract before implementation.

The first concrete implementation dependency is [nonmutating issuance admission](issuance-preflight-plan.md), independently design/source reviewed. It preserves all remaining producer and service boundaries; its module-level PASS is not approval to issue a document or consume a financial approval.
