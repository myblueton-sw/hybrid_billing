---
type: Lifecycle Protocol Design
title: First FOCUS path deletion fencing and policy outages
description: Defines the proposed irreversible deletion boundary, actual writer fencing and action-specific policy outage behavior required before implementing the first FOCUS data path.
status: draft
---

# Lifecycle prerequisites

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Owner: Seung Woo Park. Machine: `mac_mini`. This draft specifies three prerequisites from [current decisions](../current-design-decisions.md) and [build design](../platform-data-build-design.md). Read with the [physical table contract](focus-physical-schema.md). None of these protocols is implemented or runtime-certified by the engine probes.

## Common authority and serialization

One lifecycle domain owns the complete physical target/dependency closure for an operation. Shared-payer objects, mixed-retention partitions and backups require the union of all affected domains; if closure is incomplete, stop. Every new hold, dependency pin, reader pin, publication and deletion acquires the same PostgreSQL domain rows in canonical order, then scope/target rows in canonical order. Retention policy, current independently verified control watermark, authority epoch and exact immutable input digest are checked under that serialization. Retry must revalidate the complete command, never shrink its closure.

Only a mutation coordinator holds CH/object write access. Workers receive no direct storage write credentials. The coordinator checks installation epoch, domain generation, exact operation/input digest, allowed state and current policy immediately before each side effect. This check alone does not fence an already running external request; drain protocol below is mandatory. Durable domain state/unique active operation reserves ownership beyond any lease/transaction, including unknown outcomes. Store administrators and restore executors are separate authorized paths that must obey the same fence; operator custody is not tenant finance authority.

## 1. Irreversible deletion boundary

| Phase | Preconditions and durable result | Concurrent hold/pin/publication |
| --- | --- | --- |
| Prepare | Resolve exact CH row/partition rewrite and S3 object VERSION targets, exports/backups and retained dependencies; verify retention eligibility, origin, no holds/pins/readers/unknown operations; reserve domain operation; create conditional immutable journal intent | Preparation alone authorizes no deletion; new restrictive intent stops progress |
| Quiesce | Block new mutation admission; drain writers/readers for the target closure; prove current controls and journal completeness, recheck dependency/hold set | New hold before boundary is durably recorded and wins; new pins/publications wait or conflict |
| Commit boundary | With domain locks, require no active/pending restrictive control, all old effects terminal and exact policy/target/input digest unchanged; write durable deletion authorization evidence, logical tombstone and `delete_committed` operation reservation | This PG transition is the advertised irreversible authorization boundary; new holds return explicit too_late/conflict and residual status, never a false guarantee that bytes remain protected |
| Dispatch | Fenced coordinator executes only reserved exact targets. Journal authorization is durable before destructive request; commit marker binds PG decision. An uncertain PG/journal/storage result becomes unknown and blocks reuse | New readers/pins/publications reject against tombstone; restrictions can still prevent remaining deletions where physically possible, with explicit partial outcome |
| Reconcile | Query exact CH operations and object version identifiers; confirm target absence and preserved unaffected rows; do not blindly reissue unknown effects | Unknown remains logically denied and reserved; timer/lease expiry cannot mark success |
| Complete | Record per-store actual removal proof and residuals (detached parts, versions, release exports, backups, key dependencies) | Physically removed only when every required target is proven removed; otherwise explicit residual/pending state |

A hold submission first reserves a restrictive pending intent in the domain serialization, then writes the independent immutable journal and commits the active hold with its journal reference. Deletion sees pending intents and stops. A crash before acknowledgment cannot turn an accepted-looking hold into permission to delete. Acknowledgment requires durable journal plus PG state; missing either returns unavailable/pending. A hold release/permissive policy change never becomes effective from an uncommitted journal intent. The same protocol covers revocation and deletion denials.

Deletion authorization must similarly reserve pending state before its journal intent; restrictive changes arriving before the boundary invalidate that authorization. Revalidate under domain lock and mark the exact authorized intent committed at the boundary. Journal entries without commit markers are conservative on restore: replay restrictions; reconcile permissive/deletion authorization without dispatch until its committed decision is proven. No distributed atomic transaction is claimed between PG and object storage.

Physical object locks/versioning can defer removal: report blocked/residual, never success. Mixed-retention CH partitions require verified rewrite preserving every retained row and controls before removing the old bytes. Ordinary object age/CH TTL cannot bypass domains, retention holds or evidence dependencies. A deletion denied before boundary leaves original evidence intact; an unknown operation leaves scope closed pending reconciliation.

## 2. Writer fencing and unknown effects

1. Root operations authority transitions the domain (or installation on restore) to quiescing and closes new operation admission. Retain durable ownership of any active/unknown operation.
2. Block the old coordinator's actual storage route: revoke old credentials/sessions or demonstrably isolate its network/process access. Stop/cancel all registered CH query IDs and object transfers. A DB token, expired lease or new PG generation alone cannot invalidate an already running CH/S3 request.
3. Establish terminal evidence for every admitted operation and outstanding request on the old route. CH query disappearance is not sufficient for remote object transfers; verify remote effects and exact object versions. If a disconnected executor can still mutate, fail closed and keep domain quiescing/unknown.
4. Reconcile rows and object versions against the fixed operation/manifest. Stop retries/cleanup/publication until outcome is established. A sealed attempt requires proven writer completion/drain and no path able to append later.
5. Advance monotonic installation epoch/domain generation transactionally only after old access is demonstrably fenced and operations reconciled; create replacement coordinator route with least privilege. Register admission state and policy acknowledgment, then reopen. All old commands conflict by generation/digest.
6. For a load retry, now synchronously delete stale attempt rows and verify absence, allocate the next PG attempt and load. For deletion, retain the exact committed reservation. A replacement does not reinterpret a correction or create a fresh economic identity.

Outbox effects use fixed operation keys and exact conditional-create/version checks. An S3 create timeout requires lookup/hash reconciliation, not assuming the first write failed; a conflicting object never gets overwritten to make the retry pass. Restore fences all old routes before replay/rebuild and requires independent journal stream completeness beyond the restore point, then reconciles external effects before reopening. The current single-node engine probes prove neither route revocation nor lifecycle concurrency; this coordinator is a proposed implementation constraint, not an installed service.

## 3. Policy-server outage matrix

“Current” requires the action's authorized policy revision/digest and complete current control watermark, not a cached timestamp. No unbounded stale-configuration fallback. Cache permission is a separately approved capability with exact scope/input, action, epoch, expiry and maximum rows/bytes; no configured bound means stop.

| Action | Policy/control authority unavailable | Recovery condition |
| --- | --- | --- |
| New collection/intake or re-request | Stop new external reads/admission; record unavailable without falsely accepting data | Recheck connection scope, current policy, quotas and registered route |
| Already authorized immutable local processing | Continue only under the exact bounded capability above, without new external reads, publication or disclosure; otherwise stop safely | Resume same input/operation after current authorization and fence checks |
| Release confirmation/publication or new dependency/pin | Stop; no pointer or financial eligibility transition | Full current authority/dependency/manifest and scope CAS revalidation |
| Interactive read, export, download or AI retrieval | Fail closed at required current-control/output checkpoint; cancel/drain; keep pins until termination known | Reauthorize exact rows/fields/recipient and current release dependencies |
| New deletion, ongoing destructive dispatch | Stop before new irreversible side effects; do not clear tombstone/reservation on outage | Current controls plus exact phase and uncertain-effect reconciliation |
| Hold/revocation restriction | Only independent authorized journal+PG restriction path may durably accept; unavailable path returns unavailable/pending, never success | Reconcile all restrictive pending intents before deletion/serving resumes |
| Hold release, permissive policy update | Stop | Current authority, independent controls and new committed decision |
| Restore/rebuild reopen | Keep executors/serving fenced; local byte verification only under bounded capability | Latest independent complete journal watermark, all restrictive controls applied, unknown effects reconciled, authorized generation |

Policy acknowledgment records command identity/digest, actor, policy revision, epoch, target closure, journal evidence where required and actual outcome. Quotas use coordinated reservations, not independent cached counters. The policy server owns lifecycle permission; it does not approve prices or confer tenant access. This matrix's conservative read/disclosure stop is a proposed first-path availability tradeoff; no SLA or offline acceptance is invented.

## Required adversarial acceptance

All below are NOT_RUN product tests, distinct from successful engine probes:

| Case | Required observable result |
| --- | --- |
| New hold races prepared deletion | Exactly one serialized boundary; preboundary hold blocks dispatch and receives durable acknowledgment |
| Hold arrives after delete_committed | Explicit too_late/conflict and true residual status; no promise of removed-byte preservation |
| Reader pin/publication races deletion | Domain serialization prevents newly authorized access to committed deletion targets |
| Stale loader remains connected after lease loss | No new generation loads/seals/deletes until old actual access and requests are fenced/drained |
| Coordinator dies after external request | Unknown operation remains reserved; reconciliation proves exact effect before retry |
| Policy outage at publication/output/delete | No new unauthorized pointer/output/destructive effect; terminal requests release pins only with evidence |
| PG point-in-time restore behind hold/revocation | Independent current restrictions replay before reopen; missing watermark fails closed |
| Shared object spans retained customer rows | Verified retained-row rewrite, exact controls and permissions; no blanket deletion |
| S3 lock or backup prevents removal | Honest pending/residual proof; no physically-removed label |

Design review can accept this protocol without claiming these tests passed. Executable implementation must review real grants/network topology, cancellation/drain proof, transaction constraints, migration recovery and every case above before product use.


## Bounded local execution status

The [HB-27 local runbook](../../ops/installation/local-focus-path.md) and [execution review](../../reviews/HB-27-local-path/verification.md) now describe a tested synthetic local increment using a dynamic supported-version registry. It implements isolated PG authority, immutable CH facts, original/quarantine preservation, confirmation/CAS/query and verified Parquet export. It does not implement this document's entire proposed product contract. Production profiles/authentication, quarantine resolution, actual fenced takeover/deletion, restore controls and capacity remain gated; historical design NOT_RUN statements are not retroactively converted into product passes.


## Bounded local implementation evidence

HB-27 subsequently implements the [trusted single-host CLI lifecycle](../../ops/installation/focus-collection-lifecycle.md) with independent DB-rollback controls, complete-window revisions, typed archive/recall, holds, residual-aware deletion and guarded backup restore. [Its execution report](../../validation/reports/HB-27-lifecycle/summary.md) identifies actual tests. This does not certify all proposed multi-host fencing, production policy-authority outages, storage-media erasure, full retention UI or production capacity scenarios in this protocol. Unknown dispatched effects remain closed and have no automatic takeover. Preserve the preceding NOT_RUN design gates for those broader paths.
