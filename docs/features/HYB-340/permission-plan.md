---
type: ImplementationPlan
title: Direct-human financial approval permission
description: Shared recoverable permission engine mode and narrowly governed server grant integration.
status: draft
---

# Design and ownership

Continue active HYB-340, machine:mac_mini, branch feat/HYB-340-financial-integration after7c75d99. Root is sole writer and integration owner. Independent refusal_review accepted the pinned engine design before code. Engine/module ownership covers modules/internal/grantauthority closed mode, modules/financialpermission wrapper and focused tests; later server adapter/commands/guards/impact and actual screens receive their own scoped verification. No second financial approvals registry is introduced.

# Closed direct-human engine

Domain mode0 remains the existing console-ceiling/human-grant contract, including fixed legacy digests. Mode1 is immutable after construction and admits only direct_human_grant with empty CeilingID. Unknown modes, legacy kinds in direct mode, direct kind in legacy mode, ceilings and unknown capabilities refuse. Recipient exclusion applies to proposing, approving and rejecting; withdrawal remains proposer-only.

The financialpermission wrapper fixes mode1, synthetic-financial-permission-v1, financial-permission- deterministic grant prefix, approve capability and charge_approval resource kind. Existing candidateKind charge_candidate remains Cost read authority; approvalKind approval_review remains B09 General read authority. Neither becomes permission to approve a charge.

Reuse existing unknown-effect reconciliation without relaxing it: immutable decision actor/reason/digest, fresh governor checks, exact deterministic effect lookup, no unknown-to-rejected/withdrawn transition, no recreation of matching revoked/expired grants. Historical receipt status is not current permission. The module supplies proposal mechanics, not authenticated identities or actual gate grants.

# Server integration design

A closed server domain must bind action gate.Approve, subject-only, no client/fields, dedicated resource kind, explicit bounded validity and honestly whole-workspace scope. Two current installation governors must differ from the recipient. Current active human membership is required before new application; reconciliation must distinguish a historical revoked/expired effect from current eligibility. Exact matching includes all scope selectors, action/kind/fields, recipient and validity. Generic grant-add paths must not provision this dedicated kind; separately authorized grant revocation remains available.

The adapter must preflight output capacity and reserve intent before effect, preserve before/after-effect fault behavior and count pending/unresolved permission proposals separately. RequestApproval provisioning and actual human decision transport remain to be resolved under explicit reviewed scope. Do not strip ConsoleClient globally: approval commands deliberately reject client calls and require CheckApproval/RecheckApproval with the actual original capture author and current module amount limits. No default grant, amount decision, production permission, deployment or merge is included.

# Reviewed server adapter conditions

Independent review accepted a third closed serverGrantAuthority domain with fixed direct-human shape, action/kind/scope and its own prefix/fault namespace. Both Current and final grant construction must reject every other shape; legacy domains still reject DirectHuman. All financial permission commands, including inspect, require current financialGovernor checks. Stored proposer checks include a canonical sorted module-admin-list fingerprint as well as existing person/installation authorization evidence.

Before a new effect or an authoritatively absent unknown retry, validate recipient membership. An exact existing revoked/expired effect may resolve uncertainty as a historical receipt without recreating the grant or claiming current eligibility. A twenty-field receipt budget includes the existing nineteen-field maximum and an explicit historical receipt marker; value capacity must cover the engine's1024-byte rationale bound. No read/action/scope fields come from request data.

Exclude the dedicated resource kind at generic M03 catalog, request validation, retained-request approval and final grant construction. Preserve and verify independent M03 revocation of the actual subject grant. Hold permits inspect/reject/withdraw and refuses propose/approve; existing step-up remains. Pending, applying and unknown records need their own fallible impact category. Current financial governance must be checked before same-key command receipt replay, not only in the effect handler. Implementation and verification are recorded separately in permission-server-verification.md.
