---
type: DesignSpecification
title: Installation recovery and retention administration
description: Specifies installation support restore migration and retention workflows with durable evidence and safe resume gates.
status: draft
---

# Context

Inherits [shared rules](README.md). Sources: HB-58 OPS04–06; v1.16 B0913–B0920, B1505–B1516; [usage lifecycle](../../../architecture/usage-detail-lifecycle.md), [identity boundaries](../../../architecture/security-boundaries/identity-session-permissions.md), [ERP obligations](../../../contracts/billing/source-erp-and-issuance-obligations.md). Every action in this file is a proposed behavior with **unresolved command binding** except explicitly referenced ERP commands. Approval matrices, retention durations, RPO/RTO and supported deployment combinations are not selected here.

# P01 — installation, upgrade and support (OPS04)

Entry: Settings → Installation; authorized installation operators see system health without automatic access to customer financial detail. Fields: installation ID, product/schema versions, supported profile, existing-data detection, services/ports, volume and capacity plan, credential references, proxy/CA references, approved outbound destinations, offline mode, backup manifest, migration compatibility, maintenance window, responsible owner and checkpoint. Preflight validates required services, conflicts, storage/decryption and offline package integrity. Never initialize over detected data implicitly.

Flow: choose supported installation/upgrade plan → preflight → inspect downtime/data migration/backup effects → approve required change → execute checkpointed stages → verify service, schema and data → resume permitted work. State separates preflight blocked, ready, running, waiting, verification failed and verified result. Changed target/version requires a new plan. Upgrade rollback must include compatible schema/data restore and fencing; changing an image alone is not a data rollback. Reentry inspects durable checkpoint and actual service state before resuming.

Support panel fields: issue reference, requesting owner, bounded actor/scope, permitted actions, start/end, approval and audit reference. Access expires/revokes independently from the issue status; no permanent remote administrator default. Diagnostic bundle preview lists categories, redactions, target scope and retention/expiry. User explicitly exports a masked bundle; no automatic upload or external support connection. Secret values, raw customer bodies and license keys are excluded.

Walkthrough O01: an upgrade preflight finds existing data and an incompatible migration path; execution stays blocked until a reviewed compatible plan exists. A failure after migration resumes only from verified checkpoint or the approved coherent recovery set. An expired support grant is denied even while the issue remains open; bundle export still rechecks current rights. An operator without financial read sees operational counts only, with no customer existence leakage.

# P03 — backup, isolated restore and migration (OPS05)

Entry: Settings → Recovery; separately authorized recovery operators and finance/security reviewers. Fields: backup set ID/time, schema/product versions, evidence/metadata manifest, counts/checksums, encryption-key references and readability, dependency completeness, source installation, isolated target, latest independent revocation/deletion control watermark, old-writer fencing evidence, external submission/delivery ledger, validation report, owner and approved resumption scope. Missing dependencies, unreadable keys or unverifiable latest controls leave the restore quarantined.

Flow: select coherent set → validate compatibility and targets → approve isolated restore → fence old workers/credentials and lock target external writes, notification dispatch and issuance → restore → apply latest deny/deletion controls → verify counts/totals/lineage/readability → reconcile external effects → approve bounded resume. Never enable services just because database restore succeeded. Rebuilt read models use only eligible retained evidence. Historic invoices remain read-only and cannot enter an issuance queue.

ERP recovery uses `erp_route.lock_dispatch`, `erp_route.sync_after_recovery` and `erp_route.unlock_dispatch` under their existing gates; manual assignments require the exact independent commands in the registry. Unknown outcomes remain assigned for reconciliation; completed manual/ERP numbers are synchronized before new dispatch. Notifications/delivery also need duplicate/outcome reconciliation, not blind outbox replay. Distinct domain dispatch gates remain locked until their evidence passes.

Migration checklist: final source close or explicit pending-final owner; export manifest and signed/authorized receipt evidence where policy requires; destination count/total/hash/readability verification; new destination credentials; old installation fencing; current rights/control watermark; retained history access; unresolved job/late-credit owners. Copying credentials is not credential provisioning. Cutover never replays historical economic actions. A failed validation retains source history and isolated target for governed recovery, not two active writers.

Walkthrough O02: restoring a backup predating a member revocation and deletion decision does not resurrect that access/data. Apply latest controls before serving any restored view. If the control source is unavailable, stay quarantined. An ERP submission with an unknown response remains blocked while an operator reconciles its actual number; no duplicate issuance is offered. A failed target read-back prevents cutover and displays failed partition plus owner/checkpoint.

# P04 — storage, archive, holds and deletion (OPS06)

Entry: Settings → Data lifecycle; scoped data operators, policy approvers and authorized hold owners have separate capabilities. Fields by data class/customer/period: policy revision, clock basis/date, duration, hot/archive windows, recall target, deletion grace/method, owner/approval, current store/bytes/rows, hold/legal basis, shared references, financial dependencies, active jobs, export copies, backups and expiry. Recopy/archive/restore cannot reset clocks. Durations and legal applicability require decisions; absence blocks automated disposal.

Flow: choose class/scope → preview exact candidates and dependency/hold exclusions → approve policy-bound request → durable decision → logical access denial → per-store cleanup → backup expiry → policy-complete reconciliation. UI separately shows requested, decision-confirmed, deny effective, cleanup pending/failed, backup pending and policy-complete evidence. Unknown durable commit is not decision-confirmed; reconcile the original operation first. Do not call data fully erased while a governed managed copy remains.

Before every removal, serialize with new hold/reference creation and recheck financial dependencies/active jobs/current policy. Shared evidence cannot be removed by one customer's offboarding. Legal holds are not billing/delivery/collections holds and generic case resolution cannot release them. Partial cleanup lists managed stores with succeeded/held/failed/pending counts and owner; eligible failed partitions retry without removing protected dependencies. Backup expiry is separately evidenced; unmanaged exports cannot be promised remotely erased. Preserve only justified minimal controls/financial lineage under approved horizons, not indefinite retention by default.

Archive recall fields: retained object manifest, authorized purpose/scope, current policy/rights, recall job, expiry and destination. Recall provides eligible retained data only, rechecks rights on resume/download and does not reset retention clocks. Expired/denied requests require a new authorized request; they do not revive deleted objects.

Walkthrough O03: a deletion preview includes an object newly placed on hold before commit; revalidation excludes it and explains the authorized reason. Logical denial succeeds but one projection cleanup fails; UI shows that exact residual and does not report all copies erased. A restored older backup applies the latest durable deletion watermark before read-model publication. A recall completed after membership revocation provides no download or cached data.
