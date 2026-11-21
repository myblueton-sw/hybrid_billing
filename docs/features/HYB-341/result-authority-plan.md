---
type: ImplementationPlan
title: Separate authority for synthetic ERP result evidence
description: Provision explicit independent result capabilities without broadening existing document or financial grants.
status: draft
---

# Decision and ownership

HYB-341, machine:mac_mini. Root is the sole writer; refusal_review independently reviews financial/security boundaries and platform_refusals reviews evidence/terminology. The [parent plan](plan.md) retains full result ingestion, reconciliation and final approval scope. Independent source investigation recommends a separate fixed domain 6 rather than expanding domain 5. Existing document grants and digests remain unchanged.

Implement modules/resultauthority as a fixed wrapper over the existing internal grantauthority engine. Version synthetic-erp-result-authority-v1, grant prefix result-grant-, server fault prefix result-authority-, resource kind synthetic_erp_result_evidence, Cost field erp_result_evidence. Closed capabilities propose, confirm, revoke, apply and inspect map respectively to ProposeERPResult, ConfirmERPResult, RevokeERPResult, ApplyERPResult and dedicated ViewCost. Apply means ingest confirmed synthetic result evidence into the internal ledger; it grants no external ERP command or financial approval.

The four actions are direct-human, nondelegable auth actions. Provision both an explicit console ceiling and a human grant constrained to that same capability/namespace/current validity. Actual result consumers must additionally verify actual current human session, console intersection and no delegation. No default result grants or evidence. SeedNone has no registry. Reuse the single installation governance engine with a separately initialized instance/maps; never alias document-authority records or ceilings.

# Service provisioning contract

Expose result_authority.propose_console, propose_human, approve, reject, withdraw and inspect using the existing bounded shared provisioning handler. All commands require current installation governance; recheck actual governor authority before cached replay. Mutations require existing CSRF, recent authentication, expected revision, rationale and immutable idempotency. Require at least 20 output fields and 1024 bytes per value before mutation, retaining the historical_decision receipt marker. Approval uses a distinct currently authorized governor, rechecks original proposer and recipient and enforces exact active ceiling containment. Existing rejected/withdrawn/expired/superseded/unknown effect behavior is retained.

Integrate fixed domain dispatch, current-target checks, effects, command boundaries, sample initialization and domain-specific hold/step-up inventories. Exclude any result action, resource kind or Cost field from generic grant-add paths, even when mixed into another namespace; existing explicit revocation remains available. Include pending result-authority proposals in fallible impact tallies so bulk user operations cannot silently omit them. Pending result-evidence proposals are a later separate store, not fabricated by this provisioning increment.

# Scope and verification

Root may edit the new fixed authority wrapper/tests, auth/gate declarations and relevant tests, narrow server domain/constructor/shape/dispatch/initialization/provisioning/bypass/impact/hold/step-up/catalog integration and these ticket records. No ERP result packet or ledger mutation is introduced by provisioned rights alone. Provisioning UI integration and its actual EN/KO/AR renders must be separately completed before claiming a usable end-to-end result workflow; backend grants are not screen acceptance.

Verify all five capabilities, fixed namespace/version/prefix, old domain grant digests unchanged, cross-domain ceiling/record replay refusal, unsupported domain refusal, no default grants, SeedNone, self-approval/recipient exclusion, changed proposer/governor/recipient status, revoked/expired ceiling, direct-human/console intersection and delegated denial. Exercise actual authenticated command issuance, lost-decision recovery, stale revisions/digests, maximum output reasons, hold/step-up and generic bypass. Run auth/shared authority/module/server scoped race/source/vet and appropriate regression. Independent design review precedes edits; corrective review follows actual results. UI and production ERP certification remain unclaimed.

# Independent design review and visible impact

refusal_review passed this backend contract before implementation, requiring independently closed final grant shape, historical revoked/expired unknown-effect reconciliation, mandatory Cost classification despite configured overrides and independent expected capability mappings. The pending tally adds one visible M01 category; that localized label must receive terminology review and actual rendered EN/KO/AR verification. This does not implement result provisioning controls or result ingestion screens.

The same new grant namespace also appears in existing M03 human-grant and P02 console-ceiling rows. Add localized action/kind/field labels there without adding generic grant offers or result provisioning controls. Render the actual issued rights in EN/KO/AR and preserve the existing layout and independent design review.
