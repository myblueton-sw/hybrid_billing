---
type: DesignNote
title: HYB-304 frozen session selection integration
description: Specifies the approved selection lifecycle's concrete integration with legacy termination and retry retention.
status: draft
sources:
  - resource: ./plan.md
  - resource: ./query-server-progress.md
---

# Record and authority

Root is sole writer on machine mac_mini. Reuse one server-owned record from building through sealed to accepted operation. A global reference table and operation-ID index point to that record; its frozen target rows and accepted result/retry references are charged once per owned allocation. Existing selfEnds lists may reference the operation for compatibility; dropping an entry from that list does not release an active record or retry budget.

Bind every selection to canonical owner/user epoch/workspace/caller session and fixed creation/reference/expiry. No login-only target reconstruction. New target extraction and application use the canonical per-owner index under the session mutex, with the server domain lock protecting preparation/acceptance. Target reads remain read-only. Newly created sessions outside the immutable set are reported, never added; missing recorded targets remain gone results.

# Wire and construction

- POST `/me/sessions/selections` starts with `{}` or resumes with `{selection, after}`; strict object keys/types/duplicate detection and bounded input. Preparation is an authorized, CSRF-protected transient read snapshot, not a domain authority change.
- Scan at most session_query_scan source rows per call. During construction bind index/domain revisions and eligibility deadline; a changed/expired construction requires a new preparation. Retrying the same continuation returns the current record without duplicate append. Repeating an initial request for the same live source reuses its record.
- Building replies expose observed progress with count null, not a final count or percentage. Sealing establishes the exact target count/digest and makes the record confirmation-ready; applying and accepted records preserve that same immutable count/digest. A GET summary/preview on `/me/sessions/selections/{reference}` pages the immutable rows at the existing bounded page size; reference/continuation scope is authenticated.
- New termination input on `/me/sessions/end` is `{selection, digest, count}`; it is mutually exclusive with the existing `{session,generation,handles}` and one-handle variants. A zero-target selection cannot produce a termination operation. The current admission/step-up gate remains authoritative.
- Resolve the same accepted record and supplied fingerprint before preparation-expiry/live-generation comparison. Retained operations replay after preparation expiry. Missing, evicted, foreign or incompatible references cannot reconstruct a selection; foreign/missing lookups disclose no existence distinction.

# Capacity and legacy paths

New changeable limits use the existing AccessSettings/CLI mechanism: prepared views per caller2 (1–8), targets per view10000 (64–10000), global retained record/reference budget50000 (256–1000000). The existing query lifetime controls building/sealed expiry. Startup validates budget >=1+3×target maximum. Resource admission still checks occupied capacity and actual not-included allocations.

Charge one record plus each frozen target, accepted result, not-included row and outstanding retry. Fixed row fields and the existing eight cleanup-item categories remain bounded. Before effects, reserve result rows + exact not-included rows + one possible retry per target. Acceptance transfers existing charges. Release unused reservation after construction; completing/sweeping retries releases their charges exactly once.

The same global accounting applies to legacy one/≤64-handle operations; they cannot bypass capacity by avoiding the new selection route. Preserve their explicit confirmed set and existing replay behavior. Canonical workspace extraction is required for any termination source; an input referring to another scope never authorizes its termination or discloses its metadata. Keep legacy generation tokens for legacy responses; new records use their canonical scoped generation consistently.

Expired unaccepted views release all charges. Oldest settled operation records may be evicted for capacity, making later references explicitly unavailable. Active retries, including final failed denial cleanup awaiting release, keep the minimal record state and charge; they are not evictable merely because the visible operation is outside the compatibility list's five entries. Admission failure occurs before side effects and leaves the confirmed set unchanged.

# Operation recovery and bounded reads

Selection preparation, preview, acceptance and replay remain bound to the original caller. Accepted operation summaries and result/not-included pages permit another currently authorized session of the same canonical owner, user epoch and workspace for recovery. Missing or foreign operation IDs return 404. Unconfirmed intent is visible only to its original caller, with outcome, cleanup and current-generation fields null; its result rows remain unavailable. Operation lists count and page accepted records only.

Accepted operations use an owner-scoped ordered index. Page and item cursors bind the caller, owner, page size, view and operation revision; acceptance, eviction and cleanup transitions invalidate previous continuations. Summary counters are maintained during result construction and cleanup transitions so that a summary never scans retained result rows. Result and not-included pages copy at most 100 rows.

# Remaining integration

The server now implements bounded operation summaries and result/not-included pages. The typed client methods are present; migration of `/me` and rendered console consumers remains unfinished, and the server foundation still requires final verification evidence before increment completion. This design note does not authorize a partial-feature merge or declare UI-04 complete. Independent bounded integration-design review passed; actual-change/security/coverage/replay/resource and rendered checks follow implementation.
