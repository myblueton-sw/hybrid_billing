---
type: ImplementationPlan
title: HYB-304 retained session record lifecycle contracts
description: Verifies idempotent publication, index removal and exact record accounting using actual accepted operations and separate component inputs.
status: stable
sources:
  - resource: ./m04-admission-consistency.md
---

# Final status

Checkpoint record. The work it names as remaining or not yet built was completed later in HYB-304, which closed on 2026-11-27 ([closing audit](closing-audit.md)).

# Scope and reviewed design

HYB-304 remains In Progress, Seung Woo Park, machine mac_mini, after pushed/read-back 5e53bb189dfa860670008151c6420c2095e51c27. Root is sole writer. Separate read-only /root/hyb310_review approved the following staged component contract with explicit limits. Create real single-target and others operations through authenticated HTTP producers, then call existing lifecycle helpers with those retained records. Repeated publication must preserve one list entry and current generation. A foreign local copy with the same reference must not delete the registered object. Index removal must be idempotent and preserve unaffected operations and exact units. Known-ID access remains available after index removal alone; only forgetting the record removes that mapping. Repeated forget must not decrement unrelated reservations. Others-tag claims require an actual kindOthers record and a later surviving session, not a kindOne operation.

No registry maps or store entries are rewritten. These are lifecycle-helper contracts, not a claimed HTTP interleaving or reproduced exploit. No production change is planned. Compare actual list/known-ID/self reads and independently accounted units with nonempty unrelated controls. New meaningful invariants against existing correct behavior need not fail before a product fix; a semantic mutation should demonstrate that the critical identity/idempotence assertion detects a broken helper. Existing cleanup, retention, tags and operation suites remain applicable. Record scoped/race results and source restoration; only a new full profile establishes a later package measurement. Required100%, aggregate M04 review and authorized merge/read-back remain open.

# Additional reviewed failure contracts

The same independent design investigation approved real registered-selection wrong digest/count refusals, an explicitly staged captured-caller logout before acceptance, authenticated oversized preview continuation and malformed key inputs, server-signer robustness using only resealed position copies of actual emitted cursors, and pending-cleanup retention. It also approved normal store put/bind/deny/release generation checks and fail-closed local captured projection inputs. These contracts preserve all source, ownership, reservation and authority guards. Inter-call service-detail failures, cryptographic collisions, missing-owner-index contradictions and internal reservation-undercharge panic remain unverified; no invented seam or map corruption substitutes for them.

# Observed scoped verification

All eleven new groups passed in1.484 seconds against unchanged production. Twenty-one new and related lifecycle/index/cleanup/retention groups passed under the race detector in5.493 seconds. Server-package vet passed. Separate read-only /root/hyb310_review inspected the actual three test files and plan and reported PASS, with no tests or mutations executed by the reviewer. The source uses legitimate producers and explicitly staged component inputs, never direct registry-map or stored-record rewrites.

Two semantic mutations were assertion-killed: bypassing registered-object identity failed the same-reference-copy assertion, and bypassing duplicate-publication protection failed the retained-list/generation assertion. Source bytes were restored exactly after each mutation and the private cache was removed before current validation. Production source, Console product inputs/build and prior browser evidence remain unchanged. No fresh browser, foreign runtime or full all-package execution is claimed for this test-only increment.

The full server short suite then passed in187.448 seconds with99.5% statement coverage:11,783 weighted statements and59 uncovered. This is the current server-only measurement, replacing the latest99.3%/79 result while preserving that75-package checkpoint historically. Required100% remains unmet. The two documentation tests passed but check frontmatter status only; root separately checked changed metadata and links. [The record manifest](record-lifecycle-evidence.json) binds exact source, mutation, profile and log artifacts. Bounded M04 aggregate source/TW/evidence review also passed independently, as recorded in [the finite checklist](m04-completion.md). Package verification, PR readiness and authorized merge/read-back remain pending.
