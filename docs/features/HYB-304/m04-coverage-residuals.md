---
type: Review
title: HYB-304 remaining server coverage and merge criterion decision
description: Classifies unexercised defensive statements and records the unresolved whole-package coverage criterion without removing protections or claiming reachability proofs.
status: stable
sources:
  - resource: ./m04-create-clock-contracts.md
  - resource: ./m04-completion.md
---

# Final status

Decided: on 2026-11-26 the owner accepted the remaining 28 uncovered weighted statements instead of the 100 percent gate ([main integration](main-integration.md)). HYB-304 closed on 2026-11-27 ([closing audit](closing-audit.md)).

# Facts and independent assessment

The final native server short run passed199.241s at99.8%,11,779 weighted statements/28 uncovered. The separate focused create sweep hits the new ServiceCount refusal unit; native profiles are not combined. Bounded M04 source/TW/render reconciliation is PASS within its finite checklist. Whole-package100% remains unmet. The owner authorized Git merge; that authorization has not been recorded as a waiver of this separate coverage criterion.

Separate read-only /root/hyb310_review assessed the remaining guards after the legitimate producer, clock, resource-preservation, dependency and activation batches. No additional meaningful bounded producer is established under the current constraints. This is UNVERIFIED reachability, not proof that every guard is permanently unreachable. The reviewer executed no tests and did not approve merge readiness.

| Source guard | Current evidence and limitation |
| --- | --- |
| accesshistory.go83 | Examined persistence-open failures carry ErrConfig/ErrLocked, including damaged data. No concrete alternative generic ErrStart producer established. |
| servicequery_choices.go127; servicequery_detail.go295 | Retained party IDs survive supported state transitions; ineligibility returns the separately handled ErrSuspended. Other dependency errors lack a current public producer. |
| servicequery_cursor.go127/133 | Sticky generation exhaustion is legitimate, but practical bounded public generation or supported initialization is not established. |
| servicequery_cursor.go165; servicequery_detail.go57 | No genuine foreign-workspace membership admission producer in the single-bootstrap construction; arbitrary foreign cookies fail earlier. |
| session.go261 | Public unlink permits a plausible OpenSession/Resolve race, but no deterministic existing synchronization point is established. Invoking unlink inside the identity clock callback deadlocks its mutex. |
| session.go266–271 | Resolved users remain retained; User is untimed. No public deletion or constructed-registry unavailable transition established. |
| session.go277–280 | Concurrent app-session removal could defeat bind, but no bounded existing seam establishes that timing. Rollback remains protected. |
| sessionaccept.go117; sessionoperationitems.go28 | Applying state is protected by the domain write lock; ordinary reads/replays cannot observe it. No supported interruption/recovery producer exposes it. |
| sessionending.go109 | Present bound caller and owner index share session-mutex invariants; no independent missing-index producer established. |
| sessionindex.go99 | Distinct verifier collision in a truncated SHA handle has no feasible bounded construction; repeated entropy creates the same verifier. |
| sessionquery.go156 | Registry-unready outcome needs an absent/unbuilt handle; no public transition from the constructed registry established. Expiry returns another outcome. |
| sessionrecords.go144 | Direct nil call is possible but supplies only an excluded nil no-op test, without a substantive resource contract. |
| sessionrecords.go151 | Undercharge contradicts maintained reservation invariants; no legitimate producer established without fabricating stored state. |
| sessionselection.go190 | Domain locking protects preparation/version/indexed lookup; no existing callback supplies a competing mutation in the remaining interval. |

# Unresolved owner criterion

The handoff requires coverage not to drop, recording server100% as its baseline. Current branch coverage remains below that baseline despite meaningful new contract verification. Readiness must stay unconfirmed until the owner explicitly chooses a permitted meaningful verification mechanism or changes the merge coverage criterion. No exclusion, protection removal, fabricated state, coverage waiver or replacement requirement is authorized by this assessment.

The concrete policy question is whether the observed passing behavior/independent defensive review at99.8% may satisfy this increment's merge criterion, or whether literal100% must remain required with an explicitly reviewed additional verification approach. This is a quality-criterion decision, not another request for permission to execute or merge. The existing merge authorization remains valid. Authorized main merge/read-back, parent M03/M02 and the complete planning goal remain open.
