---
type: SecurityImplementationPlan
title: Reader SQL revocation and restore fencing
description: Defines durable reader denial, scoped engine enforcement and isolated regression evidence for HB-33.
---

# Scope and ownership

[HB-33](https://linear.app/hybrid-billing/issue/HB-33/enforce-reader-sql-revocation-and-preserve-denial-through-restore) follows merged PR28 at `79866f4388b8052d64443b974e4d1eec9cedacf9`. Accountable owner Seung Woo Park; machine mac_mini. Root owns implementation and QA integration. An independent DBA/security reviewer reviews design and changes. Owned files: `modules/focus_import/{revocation,authority,warehouse,lifecycle,service}.py`, focused tests under `tests/focus_import/`, and this ticket's work/validation documents. Preserve all unrelated files and the default 1123-row runtime.

# Design

The I17b failure is caused by changing product flags without changing engine privileges. Derive denial from validated configuration plus durable revoked actors: tenant readers, directly revoked audiences and every audience containing a revoked tenant. Affected audiences are denied completely; unrelated member readers stay authorized.

Revocation uses a dedicated coordinator under the existing lifecycle lock, not the general lifecycle transaction. Persist a pending operation and revoked-actor intent with an autocommit checkpoint, apply PostgreSQL object privilege removal and ClickHouse table SELECT removal/false row policy, then persist current metadata and clear the pending operation. Resume only the same pending revocation; unrelated pending operations and inconsistent checkpoints remain denied. Return success only after both engines finish. An unavailable engine leaves a visible pending operation and is not claimed fenced.

PostgreSQL enforcement is limited to this database's `hb_focus` schema and generated reader roles. Terminate affected sessions only in the current database before and after committed privilege removal, including prepared/long transactions; no role deletion, global NOLOGIN or role membership changes. Bound lock waits and leave failed attempts pending. ClickHouse enforcement removes only `hb_focus.fact` access. Its supported cancellation profile uses exclusively provisioned internal reader identities: inspect their grants, reject any privilege beyond this fact table, then cancel their active queries synchronously and confirm no active query remains. This does not support arbitrary shared users or prove the historical authority of queries started before an administrator changed their grants. Fresh namespace/identity ownership is verified in the test harness. Already delivered data cannot be recalled.

Both engine initializers accept the denial closure and must never grant protected access to denied readers, including ordinary reinitialization and missing-schema restore. PostgreSQL changes remain transactional. ClickHouse creates/updates denied policies without granting SELECT. Service initialization and both restore paths pass durable revoked actors before any provisioning or restoration. Existing active-state checks continue to govern product access.

# Verification

Use a fresh dedicated PostgreSQL database and synthetic bucket, unique tenant/view identities and a verified empty separate ClickHouse service/database. Do not reset HB-32 or default namespaces. Verify prior failure, then existing/new PG sessions, prepared statements and long transactions; CH reused/new clients and in-flight behavior; full dependent-audience denial; unchanged A2/B data; direct audience revocation; duplicate/retried revocation and injected partial failures; reinitialization and old-backup/missing-schema restore. Record actual engine versions, source hashes, exact results and default-runtime/role preservation. Unit tests cover denial closure and ordering; they do not substitute for real engine evidence.

External authentication, customer-safe projection and portal/export paths remain separate work. The owner was asked which authentication profile to implement first because the existing design leaves the provider and session policy unresolved. No production deployment or real-account access is included.

Independent DBA/security design review accepted this bounded plan; actual change review identified and required an additional existing-schema restore fence before transaction locks. That correction and the denial-aware missing-schema path received source re-review. Runtime evidence and remaining boundaries are recorded in the ticket's validation report.

Documentation reconciliation also updates `docs/architecture/data-ownership/focus-lifecycle-protocols.md` with the bounded implemented status and `docs/ops/installation/focus-collection-lifecycle.md` with revocation/retry use, preserving broader unverified coordinator contracts.
