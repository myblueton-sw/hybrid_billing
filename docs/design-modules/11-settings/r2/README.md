---
type: DesignSpecification
title: Administration and operations planning revision
description: Defines the HB-59 draft scope shared interaction rules module ownership and remaining adoption gates.
status: draft
---

# Scope and authority

[HB-59](https://linear.app/hybrid-billing/issue/HB-59) develops the [HB-58 findings at b87e723](https://github.com/myblueton-sw/hybrid_billing/blob/b87e723c79650e9e558caa82633239d6252c2a8a/docs/reviews/HB-58-planning-coverage/report.md) into textual screen planning. Root is the sole writer; accountable owner is Seung Woo Park; machine is mac_mini. Independent administration/security, financial and operations reviewers check the plan and actual documents. These are proposed screen behaviors constrained by existing contracts, not deployed controls or owner acceptance of a replacement edition.

Baseline: frozen v10 at `e955befffa54a9cefb392984ab7716a67c8e05b0`; previous revision is the corresponding v10 module, not a prior r2 implementation. Original screen IDs below remain navigation identities; section names describe states, not new registered screens. No frozen pages, original local specifications, generators or runtime files are replaced. Exact replacement/deletion manifests are required on later owner acceptance. HB-24 companion adoption remains open.

| Owned revision | Screens / responsibility | Detailed specification |
| --- | --- | --- |
| 02-access/r2 | M01–M05, L03; membership, identity, sessions, service identities | [Access administration](../../02-access/r2/administration.md) |
| 04-work/r2 | O01–O03, U01–U04; customer relations, protected settings, lifecycle and work | [Customer administration](../../04-work/r2/administration.md), [work management](../../04-work/r2/work-management.md) |
| 06-contracts/r2 supplement | B05 field validation only; B09 approval handoff | [FX settings](../../06-contracts/r2/fx-settings.md) |
| 11-settings/r2 | P01–P07, R01/R02 and A01 policy entry; installation, recovery, retention, evidence and policy | [Operations](operations.md), [evidence and policy](evidence-and-policy.md) |

Other modules retain their ownership: onboarding S01/S02 → O03 setup readiness; O03 → B05 pricing and B09 approvals; O03/U02 → I01 issuance and I04 delivery; every operation → R02 audit and U03 job/inbox; reports link retained history in module 09. Entry links carry canonical IDs, fixed revision and permitted return context, never customer names as keys. The [onboarding r2 proposal](../../03-onboarding/r2/README.md) remains independently draft/deferred.

# Shared screen contract

Header: workspace and permitted customer/group scope, screen purpose, as-of, fixed/current revision and current role capabilities. Then filters, status summary, selectable rows, detail panel, impact preview, decision and result. Scope changes clear selections and reload authorization; no cached cross-scope result flashes. Empty means no authorized matching records, not permission to disclose hidden counts. Loading preserves a labelled stale view without enabling mutation. Denied/unavailable hides forbidden identity; error offers request ID and permitted recovery. Zero, missing, stale and unknown are distinct.

Required mutation preview: exact target manifest, old/new values masked by field permission, effective instant and timezone, dependencies, reversible versus irreversible effects, proposer/approver, rationale and revision. Selection across pages explicitly states fixed target count; newly matching records are not added. Bulk results partition authorized targets into completed/held/failed/pending without leaking denied target names. No implicit select-all or partial-success-as-complete banner.

Use the [canonical command contract](../../../contracts/commands/planning-actions.md) for current authority, scope, epoch, expected revision, idempotency, independent approvals and generic denial. Recheck authority at commit, background execution and protected result reads; an earlier preview or approval is not current authority. Conflict requires reinspection and a newly reviewed intent. An accepted job is not business completion; an unknown external result requires status/reconciliation, not blind replay. Every section inherits these rules and supplies its additional validation and recovery.

Technical job states are `queued`, `running`, `waiting`, `succeeded`, `failed`, `cancelled`; external unknown is a reason/business state, not a seventh technical enum. A succeeded job may report partial business outcomes and must show that distinction. Mutation binding marked **unresolved** is an implementation gate: do not invent an endpoint or wire the action to a generic save. Existing command IDs below are proposed contracts, not live APIs.

All controls have visible labels and keyboard order. Validation lists link to fields; focus returns to the initiating control after a dialog, and to the result heading after submission. Status uses text in addition to color; asynchronously updated status is announced without stealing focus. Sensitive values use references/masked displays, never copyable secrets. Actual layout, keyboard and screen-reader checks remain NOT_RUN.

# Decisions still required

Named implementation tickets must bind the unresolved commands and approved role/ceiling matrix; select identity methods, independent recovery custodians, session bounds, business calendars/DST resolution, retention durations/restore targets, notification channel policy and support versions. Product profiles do not grant permissions. No commercial prices, vendor, AI model or legal retention duration is selected by this revision.

[Coverage and review](verification.md) distinguish textual coverage from visual, contract-binding and runtime acceptance. Separate HB-54 financial, VMware, full AI workflow and disclosure gaps remain outside this revision; this is not a whole-product completeness claim.
