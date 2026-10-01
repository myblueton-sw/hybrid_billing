---
type: Operations Contract
title: Independent dispute holds onboarding outcomes and personal favorites
description: Defines proposed commands and observable states for dispute holds, evidenced onboarding metrics and permission-safe preferences.
status: draft
---

# Holds onboarding and favorites

Part of [HB-11](../../reviews/HB-11-refinements/resolution.md); supplements PC-05/06/07 from source B1488/B1110/B1359, local B11/S02/U01 and HB-10. Proposed English contract semantics; detailed API/schema/work/screen trace integration remains PP-08. No running workflows or persisted preferences are claimed.

## PC-05: independent holds

| Hold type | Request / release and affected outcome |
| --- | --- |
| Unissued billing | BillingHoldRequest / BillingHoldRelease targets permitted unissued calculation/approval/issuance scope. Serialize enforcement with affected transitions; an issued document is not retroactively made unissued |
| Delivery | DeliveryHoldRequest / DeliveryHoldRelease blocks affected future delivery dispatch; already dispatched/delivered copies stay recorded. Release permits a separately authorized action and sends nothing itself |
| External collections | CollectionsHoldRequest / CollectionsHoldRelease targets explicitly supported ERP collections capability; local request acceptance is not external effectiveness |

Every command binds immutable target manifest/partial scope, hold/case ID, reason, current actor authority, expected target/hold revision, required approval and idempotency key/input digest. Duplicate identical command returns the same operation; same key with changed payload conflicts. Scope changes create a separately reviewed revision. A customer inquiry/case resolution requests review and grants no hold authority.

Separate command-processing states from effective hold state. Local effective records are active/released for identified scope; external state is unsupported, requested, pending, confirmed-active, release-pending, confirmed-released, failed or unknown with as-of evidence. Failed/unknown release cannot be treated as released; failed request cannot be treated as active. Distinguish desired local state from actual external state; retain last confirmed state and honest uncertainty. Adapter support and reconciliation/lookup authority must be evidenced.

Serialize local request/release against affected billing/delivery dispatch. Check all active independent holds at the final controllable dispatch/transition boundary. Delayed acknowledgment must match operation/revision and cannot clear a newer or independent hold. Dispatch already beyond that boundary is irreversible/unknown as applicable and is recorded rather than falsely recalled. Releasing one hold removes only that hold; remaining holds still block. No automatic billing resume/delivery follows release.

Commands, inquiry resolution and release cannot implicitly cancel issued documents, waive debt, change Payment Terms/due dates, fabricate receipts or remove retention/legal holds. Financial amendment requires its own governed command. Exact role ceilings, approvals, in-flight boundaries and external escalation policy remain named finance/security/adapter decisions.

## PC-06: onboarding outcome metrics

Define a versioned cohort of canonical onboarding attempt units bound to workspace/customer/source profile, start event, eligibility revision and observation cutoff. Freeze the eligible denominator; retries/reconnections are events within that unit rather than additional successes/denominators. Exclusions require explicit reason/rule and remain visible; abandonment/incomplete/censored attempts are retained in the cutoff report.

Report connection rate = connected eligible units / cohort eligible units, separately from first verified billing rate = units with evidenced first verified billing outcome / cohort eligible units. Optional verified/connected conversion must be labeled with its different denominator. For denominator zero show not-applicable, not zero percent. All cohorts/counts/exports use current scoped authority and declared as-of/cutoff.

Choose the verified outcome by supported profile before metric activation: for example an approved internal allocation versus authoritative supported external issuance. Authentication, received data, complete calculation, accepted ERP request and document delivery are distinct intermediate milestones; none automatically supplies the chosen verification evidence. No-use/partial-source outcomes require explicit policy rather than automatic success. Store evidence references and first valid milestone timestamps without replacing history on retry.

Record user-work, external-approval, source-generation and unknown intervals with start/end and evidence. Compute elapsed classified waiting as interval union; overlapping category unions are not disjoint and must not be summed as total. Preserve unfinished/censored and missing-timestamp intervals; show unknown duration rather than inventing zero. Total journey elapsed and active work/wait categories remain separately labeled.

Cohort start/unit, eligibility/exclusions, window, supported milestone and instrumentation/privacy retention remain product/data decisions, not chosen production values.

## PC-07: personal favorites

Own a preference by `(user, workspace, canonical customer)`; provide idempotent AddFavorite, RemoveFavorite, indicator and authorized favorites filter. Add validates current access; duplicate add produces one record. Persist allowed preferences across navigation/login independently of another user; rename changes display only. Preference IDs confer no customer authority.

Every list/count/search/cache response and mutation checks current workspace/customer permissions. Revocation immediately removes visible names/identifiers/counts under the current-authority disclosure contract; stored stale references cannot leak existence or enable guessed cross-workspace access. Use a generic denied/unavailable outcome that reveals no forbidden identity. Read paths must not bypass the [query boundary](../security/access-and-trust.md).

Profile policy must select purge or bounded hidden-reference retention and explicit regrant restoration. Neither option is selected here: until approved, no automatic resurrection is assumed. Account termination uses the approved preference retention/deletion policy and never transfers preferences to a recreated/replacement identity. Removing a favorite changes only personal preference, never customer records or grants. Conflict/retry handling preserves stable canonical identity.

## Planned acceptance

All cases are NOT_RUN; synthetic expected results guide later implementation and independent verification.

| Case | Observable expected result |
| --- | --- |
| DH-01 | Partial billing hold blocks only its authorized unissued target; unrelated work and issued obligations remain intact |
| DH-02 | Delivery hold prevents subsequent dispatch; release sends nothing and cannot undo already delivered copies |
| DH-03 | External accepted request/timeout/unsupported adapter displays pending/unknown/unsupported, never inferred confirmed effectiveness |
| DH-04 | Identical retry returns existing operation; stale revision/changed payload cannot apply another effect |
| DH-05 | Concurrent request/release or delayed external confirmation cannot clear another hold; unauthorized inquiry changes no hold |
| OB-01 | Three eligible cohort units, two connected, one verified: separate rates 2/3 and 1/3; conditional 1/2 explicitly labeled; connected-only affects no billing numerator |
| OB-02 | Retry preserves one unit; abandoned/incomplete attempts remain visible at cutoff; zero denominator is not-applicable |
| OB-03 | Parallel intervals `[0,10)` and `[5,15)` have total union 15, not 20; overlap labeled |
| OB-04 | Missing timestamps/no-use/partial coverage are unknown or explicit policy outcomes, not fabricated verified success |
| OB-05 | Each profile requires its own chosen milestone evidence; unauthorized cohort identities/counts remain hidden |
| FV-01 | Two users have independent favorites; duplicate add gives one preference; remove changes no customer/grant |
| FV-02 | Rename, navigation and new login retain permitted canonical favorite identity |
| FV-03 | Guessed cross-workspace customer reveals no existence and creates no unauthorized favorite |
| FV-04 | Revocation removes cache/list/count disclosure; regrant obeys approved restoration policy |
| FV-05 | Termination/recreated identity does not inherit prior preferences |

Supplementary contracts are supplied; canonical requirements/work/API/screen/acceptance reconciliation and remaining owner policy choices still prevent final PC closure. See [resolution and decisions](../../reviews/HB-11-refinements/resolution.md).
