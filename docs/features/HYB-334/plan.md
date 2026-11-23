---
type: DesignPlan
title: M04 credential rotation command
description: Design of a command that replaces a service account credential by issuing a new one and bounding the old one in a single decision, with an overlap window that is an installation default.
status: draft
---

# Scope and ownership

HYB-334, machine:mac_mini. This is a design for review and for the owner's questions at the end. It changes no code. By the owner's direction of 2026-11-19 a new implementation ticket is built by Codex; root owns this design and reviews the change. A security design review by a reviewer separate from the writer is required before implementation, because the command issues and ends credentials.

Not in scope: credentials of CSP or ERP connections, which planning keeps separate; client registration on P02; a distinct rotation permission; reading where a credential is used.

# Planning text and cross-check

Quotations are translated from Korean except P4, whose source is English. The planning files and screen specifications are in the authorized local workspace.

| Ref | Source | Text | How this design answers it |
| --- | --- | --- | --- |
| P1 | `docs/ui/screens/M04/spec.md:73` | Action "request credential rotation": permission "owner or rotation permission"; precondition "old and new versions and where they are used are known"; result "a bounded replacement work plan"; confirmation "effect on work in progress" | One command issues the new credential and fixes the end of the old one. The review step shows both versions and the end time. Where a credential is used is not known to the service, and no check of work in progress exists, because the service records neither the use of a credential nor the jobs that hold one; both are stated to the person and listed under Residuals; see Q3 |
| P2 | `docs/ui/screens/M04/spec.md:53`, `:54` | Credential reference and version are changed by "issue or rotate commands"; expiry and rotation state are "policy based", with "the order of revocation and rotation and a check of work in progress" | The list shows a rotation state on both credentials. The order is fixed: the new credential exists before the old one ends. The specification's "check of work in progress" is not met by this design; see Q3 |
| P3 | `docs/planning/v1.16/retention-access.md:24` | API keys and worker credentials have "owning organization, responsible manager, scope, expiry, rotation and revocation state" | Rotation state is added to the credential facts |
| P4 | `docs/planning/v1.16/domain-contracts.md:81` | "Credential rotation or new request IDs cannot reopen billing" | Rotation changes no claim identity and reopens nothing; it touches the identity registry and the pickup directory only |
| P5 | `docs/ui/screens/M04/spec.md:133` | Open values include "rotation and expiry periods" | The overlap window ships as an installation default that can be changed, by the owner's instruction of 2026-11-17 on open values |
| P6 | `docs/ui/screens/M04/spec.md:97` | The screen has an error state for rotation, and the new secret is not written to logs | The console shows a refused or unknown rotation with the existing attempt handling; the copy for it goes through terminology review. The secret is handled only by the existing one-time pickup; the command result, the attempt store and the logs carry the pickup reference and never the secret |

# Observed baseline

Read from main at 8345aaf.

- `service.issue_credential` (target: service account; field `days`) adds a credential valid from now for `days`, bounded by the installation's credential validity (30 days), and creates a one-time pickup of the secret bound to the issuing session for 10 minutes. It is refused when the account is suspended or already holds the installation's limit of live credentials (10) (`apps/api/server/services.go:404` to `:438`; the 30 days and the limit of 10 are configuration defaults, `server.go:203`).
- `service.revoke_credential` (target: credential) marks the credential revoked at once and drops its pickup (`services.go:440` to `:448`).
- The identity module keeps a credential as owner, verifier, validity and a revoked flag. `AddCredential` and `RevokeCredential` are separate operations; nothing changes a validity after creation and nothing links two credentials (`modules/identity/workspace.go:419` to `:454`, `:478` to `:500`, `registry.go:91` to `:97`).
- An answer issued to a service from a credential copies the credential's end, and the later recheck of that answer tests the answer's own end and whether the credential is revoked, not the credential's current validity (`modules/identity/boundary.go:186`, `:198` to `:207`). No production code obtains such an answer today; only tests do.
- HYB-333 added a pickup state per credential to the M04 list (open, collected, not collected) and an issue-attempt store in the console that keeps an unknown outcome on screen (`docs/features/HYB-333/plan.md`).
- No rotation command exists. Today the next step is revoke and then issue, which leaves the account without a working credential in between, or issue and then revoke, which is two decisions and leaves the old credential valid if the second is forgotten.
- The service does not record where or when a credential was last used in a form the M04 list returns.

# Design

## Command

`service.rotate_credential`, target: the credential being replaced; the command is routed by a credential target like revoke, and the published command description is updated with it. Fields: `days` (validity of the new credential, same bounds as issue) and `overlap_minutes` (how long the old credential stays valid; optional, defaults to the installation value, bounded as below). It carries expected revision, idempotency key and a rationale like the other service commands and needs the same standing as issue and revoke today.

Effect, in one write under the domain lock:

1. The old credential must exist in the caller's workspace, be live (not revoked, not expired) and belong to a service account that is not suspended or departed; otherwise the command is refused and nothing changes.
2. The account must hold fewer live credentials than the limit, counting the old one. At the limit the command is refused; the guidance is to revoke an unused credential first. The limit is not raised for the overlap.
3. The old credential must not already be bounded by an earlier rotation; a second rotation of the same credential is refused. The new credential may itself be rotated later.
4. A new credential is added, valid from now for `days`, with a pickup of its secret exactly as for issue.
5. The old credential's end becomes the earlier of its present end and now plus the overlap. It is never extended.
6. Each credential records the other's identifier, so the list can show "replaces" and "replaced by, ends at".

The result returns the new credential, the pickup reference and the old credential's new end. The secret is collected through the existing one-time pickup.

## Why the old end is fixed at the decision

Alternative B was to fix the old credential's end only when the new secret is collected, so that an uncollected secret could not lead to an outage. It is not chosen:

- B fails open: when the pickup lapses uncollected, the old credential stays unbounded, which is the state this command exists to remove.
- B does not remove the outage either: a collection can have an unknown outcome for the person (HYB-333, N5), where the service records the secret as collected while the browser never showed it, and B would then end the old credential although nobody holds the new secret.
- B makes the pickup, a read-like step, change credential validity, and it leaves a rotation that is neither done nor refused when the pickup lapses.
- With the end fixed at the decision, the state after the command is fully determined and the review step can show the exact end time before the person confirms.

The chosen design does not remove the outage risk; it makes it visible and leaves time to act. The overlap may not be shorter than the pickup validity, so the old credential outlives the window in which the new secret can be collected, unless it was already due to end sooner. The M04 list keeps showing the pickup state of the new credential and the end time of the old one. While the old credential is still valid the person can issue another credential in the ordinary way. None of this bounds the outage by itself: it is avoided only by deploying a working secret before the old end. A rotation is not undone: the old end is not moved back, because lengthening a credential that was told to end is a larger risk than issuing a fresh one.

## Overlap window

| Value | Provisional default | Bounds | Where it lives |
| --- | --- | --- | --- |
| Overlap when the request names none | 24 hours | from the pickup validity (10 minutes) to the maximum below | installation configuration, shown in the M04 limits the list already returns |
| Maximum overlap a request may name | 7 days | at least the default, at most the credential validity | installation configuration |

An immediate cutover is not a rotation: it is revoke followed by issue, which already exists. The numbers are provisional values under the owner's instruction on open values; see Q1.

## Module change

One new operation in the identity module, `ReplaceCredential(actor, workspace, oldID, newID, verifier, validity, oldEnd)`, performing steps 1 to 6 atomically with the same administrator check as `RevokeCredential` and the same shape checks as `AddCredential`. The module refuses an `oldEnd` that is not after its own clock, and it applies the refusal for a suspended account itself, which today lives only in the server. The credential record gains two optional identifiers (replaces, replaced by). `CredentialInfo` returns them; a link whose counterpart no longer exists is omitted. Pruning of ended credentials keeps a bounded credential while its replacement is live, so that the pair stays readable, and prunes it afterwards as today; the ended credentials kept stay within the existing bound plus the limit per account.

One existing operation changes behaviour: the recheck of a service answer additionally refuses when the credential's current validity does not hold at the time of the recheck. Without this, an answer obtained before a rotation would pass until the credential's original end, and the old end fixed by the command would not be enforced. The change is latent today, because no production path obtains a service answer, and it is required before one does. An answer still reports its original end, so a future consumer must not treat that reported end as a deadline without a recheck.

A service-level sequence of `AddCredential` and a separate bounding call is rejected: a failure between the two would leave a new credential and an unbounded old one, which is the two-decision state this command exists to remove.

## Read change

`GET /api/v1/identity/services` adds to each credential row, only when set: `replaces`, `replaced_by` and, for a bounded credential, nothing new, because its `not_after` already carries the new end. A derived `rotation` value is returned: `replacing` on the new credential while the old one is live, `ending` on the old one while it is live, absent otherwise. The limits object gains the overlap default and maximum. The integrations read is unchanged, as in HYB-333.

## Failure and recovery

| Case | Behaviour |
| --- | --- |
| Refused (stale revision, limit, suspended account, old credential not live, already rotated, bounds) | nothing changes; the refusal code is the guidance |
| Outcome unknown to the person | the same exact request returns the stored outcome; the list shows whether a credential with `replaces` set exists. If it does and its secret can no longer be collected, the person issues another credential while the old one is valid. The old end stays as set |
| New secret never collected | the new credential is live but unusable; the list shows it; the person revokes it and issues another. The old end stays as set |
| Account suspended or retired during the overlap | existing behaviour: pickups are dropped; both credentials follow the account's state |
| Account suspended and reinstated during the overlap | the pickup of the new secret was dropped at suspension and is not restored; the old end is unchanged; the person issues another credential if the new secret was not collected |
| Old credential with less time left than the overlap or the pickup validity | its end stays as it was; the review step shows that end |
| Old credential revoked during the overlap | allowed; it ends at once; the new credential is unaffected |
| New credential revoked during the overlap | allowed; the old credential keeps its bounded end and is not restored |
| Bulk revocation (M01) of the responsible person | unchanged; a rotation creates no pending work for another person |

## Console

On the M04 credential row of a live credential that is not already ending: an action "Rotate credential" beside "Revoke". The form asks for the validity of the new credential and the overlap, prefilled with the installation default and showing its bounds. A review step before sending states, as facts: which credential will end and at what time in UTC; that the new credential's secret is shown once and must be collected within the pickup time; that the old end is not moved back; that this screen does not know where the credential is used and does not check work in progress; and that a suspected exposure calls for revoking the credential, not rotating it. After sending, the existing issue-attempt handling shows the outcome and the one-time reveal. The row of the old credential shows "Ends at" with the time and "Replaced by"; the row of the new one shows "Replaces". The ui-designer role leads the composition and the copy goes through terminology review in English, Korean and Arabic; no copy is fixed by this document.

# Tests and verification

- Module: the atomic replacement; refusal of each precondition with no change, including a suspended account and an `oldEnd` at or before now; the old end never extended, as a property over every operation; an old credential near expiry keeps its end; a second rotation of the same credential refused; rotating the new credential leaves the first unchanged; the limit counted with the old credential; pruning keeps the pair readable and omits a dangling link; the administrator check; a service answer obtained before the rotation is refused at recheck after the old end.
- Server: the command through its public path with expected revision and idempotent replay; overlap default, bounds, overflow and the lower bound at the pickup validity; replay from another session of the same person and replay after the old credential has ended; reinstatement after suspension during the overlap; authentication with the old credential succeeds before its new end and fails after; authentication with the new credential succeeds once collected; the read fields; the integrations read unchanged; suspension and revocation during the overlap.
- Console: unit tests of the form model and bounds; the attempt store with an unknown outcome; rendered checks in English, Korean and Arabic at the three widths.
- Independent security change review of the implementation, and the owner's screen review.

# Owner questions, with the default applied until answered

1. Q1. The overlap default (24 hours) and maximum (7 days). Default applied: these values, changeable in installation configuration.
2. Q2. Whether rotation needs its own permission, as the specification's "owner or rotation permission" suggests. Default applied: the same standing as issue and revoke, which is installation administration today; a distinct permission is a separate ticket.
3. Q3. The specification's precondition that "where it is used is known" and its "check of work in progress". The service knows neither. Default applied: the review step says so and the person confirms; recording use per credential and checking work in progress are separate tickets, without which this part of the specification stays unmet.
4. Q4. Whether a rotation may be undone by restoring the old end. Default applied: no.
5. Q5. Whether an expiring credential should prompt rotation ("policy based" in P2). Default applied: no prompt and no automatic rotation; the list already shows expiry.

# Residuals

- Nothing detects that a workload still uses the old credential when it ends, and nothing checks work in progress, which the specification asks for.
- The pickup remains bound to the issuing session; a person whose session ends within 10 minutes cannot collect the new secret and must issue another credential.
- Records live in process memory in the pre-launch service, as for all credentials.

# Not checked

- The gate internals around the recheck.
- The P02 specification's wording on client rotation, which names M04.
- Console sources.

# Review history

Round 1, independent security design review (role-based AI reviewer separate from the writer, by reading, nothing executed), 2026-11-24: CHANGES REQUIRED.

| Finding | Severity | Resolution in this revision |
| --- | --- | --- |
| H1. The old end is not enforced on a service answer issued before the rotation | High | The recheck of a service answer is changed to test the credential's current validity; stated as the one behaviour change, with a test |
| M1. The specification's check of work in progress is silently unmet | Medium | Stated in P1, P2, Q3, the review step and the residuals |
| M2. Pruning can leave dangling links and its bound was unstated | Medium | A link without its counterpart is omitted; the bound is stated |
| M3. "Always outlives the pickup window" is false near expiry, and the mitigations do not bound the outage | Medium | Reworded; the outage is avoided only by deploying a working secret before the old end; rows added for near-expiry and suspend then reinstate |
| M4. The module took the old end and the suspended check from the caller | Medium | The module refuses an end not after its own clock and a suspended account |
| L1 to L5. Argument against alternative B; P4 is English; the rotation error state and the command description; revoke for a suspected exposure; routing by credential target | Low | Each adopted |

Judged to hold by the reviewer: the baseline statements with corrected references; that rotation chains cannot exceed what issue allows and never extend an end; that a replay returns the pickup reference as issue does, session-bound and one-time; the atomic module operation; that the read change discloses nothing new and the integrations read stays unchanged; that the defaults of Q1, Q2 and Q4 are safe.

Round 2, bounded re-review of the revision, 2026-11-24: APPROVE. The reviewer confirmed that the recheck change does not affect human answers or a validity that has not started, that the five findings are resolved and that no false statement was introduced; its three small additions (the secret and logs, the reported end of an answer, a dropped unchecked item) are included.
