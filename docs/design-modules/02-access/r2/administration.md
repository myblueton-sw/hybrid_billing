---
type: DesignSpecification
title: Access administration detailed planning
description: Defines membership trust service identity and personal security workflows with explicit authority and recovery boundaries.
status: draft
---

# Context

This draft inherits the [HB-59 shared contract](../../11-settings/r2/README.md). Evidence: HB-58 ADM01–03/ADM08; v1.16 B0386, B0388 and B0418; [identity boundaries](../../../architecture/security-boundaries/identity-session-permissions.md), [trust contract](../../../contracts/security/access-and-trust.md) and [commands](../../../contracts/commands/planning-actions.md). Role names below describe capabilities, not approved default role grants. Authentication products and methods remain deferred.

# M01 and M03 — members, invitations and grants (ADM01)

Entry: Settings → Access → Members; scoped membership administrators see their permitted workspace/customer set. Tabs separate active/suspended memberships, pending invitations and role definitions. A global identity and one workspace membership are distinct records.

Fields: member/invitation ID, masked destination, status, inviter, sent/expiry time, intended workspace/customer set, role revision, action and field grants, permitted periods, effective dates, approval ceiling, owner and last change. Dates must be ordered; target and grant scope cannot exceed the administrator's own delegated ceiling. An expired invitation cannot be accepted. A role name alone is not a grant preview.

Flow: select targets → edit scoped proposal → inspect permission and session impact → confirm/obtain policy-required approval → apply → inspect per-target results and audit. Invitation reissue invalidates the old acceptance token and records a new expiry; revoke prevents later acceptance. No token is displayed in the list. Bulk invitation/reissue/removal previews the exact target set and reports partial outcomes. Remove membership does not delete the global identity or financial history. Last authorized administrator/custodian protection must use the approved recovery policy; absent policy blocks the destructive transition.

Walkthrough A01: an administrator proposes a member with read-only customer A access and no cost fields; preview shows that exact grant, acceptance links the proved identity, and A-only access is shown. An invitation accepted after revocation is rejected without recreating membership. A concurrent role edit makes the old preview stale; reload, compare and submit a new intent. A bulk batch with one invalid destination preserves other outcomes and retries only the corrected eligible item. Command bindings for invite/reissue/revoke, membership and role mutation are **unresolved**; policy approval mapping is required before implementation.

# M02 and L03 — trust, health and sessions (ADM02)

Entry: Access → Authentication for separately authorized trust operators; personal active sessions remain in M05. Fields: connection ID, configured method reference, issuer/audience identifiers where applicable, configuration digest/revision, current epoch, proof status, custodian references, last attempt, last verified success, observation age, detection-delay bound or unknown, affected principals/session count and propagation status. Secrets remain protected references. A green configuration check is not current login availability or billing readiness.

Trust replacement: `trust.propose` → `trust.validate` → independent `trust.approve` → authorized `trust.activate`; rejection/cancellation use `trust.reject`/`trust.cancel`. Show old/new digest, fixed affected scope, legitimate-authority evidence, recovery evidence and session impact at approval. Changed proof/configuration invalidates approval. Activation advances epoch and durable invalidation intent; old sessions, refresh families, step-up and queued privileged work cannot act under the old epoch. Show publication separately from consumer propagation. Rollback requires another governed change with a newer epoch.

Session rows: session ID, principal, client/device label, issued/last activity/expiry, method and revocation state. Session revoke requires the appropriate personal or administrative scope. Display request pending, durable deny confirmed, propagation/cleanup pending and outcome unknown separately; do not report revocation successful if durable commit is unknown. Fail closed when current authority cannot be verified. Actual session read/revoke and health-check bindings are **unresolved**.

Walkthrough A02: a validated replacement is independently approved and activated; the new epoch is visible while one consumer is pending and cannot serve old privileged sessions. If the identity provider is unreachable, show last success and current unknown, preserve authorized recovery access under the separately approved custodian procedure, and do not bypass proof or separation. If activation times out, reconcile the operation before submitting another change. Recovery method, custodian enrollment and measured propagation bounds remain undecided.

# M04 — service identities (ADM03)

Entry: Access → Service identities; scoped service administrators. Required fields: stable identity ID, responsible human/team owner, purpose, customer/action/field/period scope, credential reference and version, audience, issued/expiry time, rotation policy reference, dependent jobs/connectors, last observed use and status. Unknown owner or missing audience/scope blocks activation. Human login, service credentials, source collection and ERP credentials have separate purpose and audience.

Flow: draft identity → scope/dependency preview → approved activation → rotation/revocation/owner transfer → read-back. Rotation shows old/new version, overlap policy and consumer migration results; never reveal raw credentials in ordinary views or logs. Revocation immediately follows the durable deny contract, while dependent jobs are held and cleanup tracked. Owner transfer requires the recipient to be eligible and preserves the old owner/history. Service identities cannot satisfy human financial or trust approval requirements.

Walkthrough A03: rotating a service credential updates two consumers; one rejects the new version. The page shows the failed dependency and keeps that work held rather than reporting full rotation. Retry checks the exact intended version and current authority. Emergency revoke does not wait for migration. Mutation/credential-provisioning bindings and overlap limits are **unresolved**; no credential is issued by this document.

# M05 — personal security and preferences (ADM08)

Entry: avatar → My account; the authenticated person sees only their own identity links, sessions and preferences. Fields: proved identity links, available approved authentication/recovery methods, last successful verification, session references, quiet hours with timezone, digest/snooze and permitted personal alternate. Actual MFA/recovery method selection remains deferred.

Flow: prove control of both sides before linking identities; unlink only after checking another usable approved login/recovery path and policy. Do not merge identities merely because email matches. Revoke selected own sessions with a separate result from unlinking. Membership exit is scoped to that workspace and preserves the global identity and other memberships. Last-administrator and last-method safeguards apply. Recovery material is never written into ordinary profile history.

Walkthrough A04: a user removes an old identity link after proving a replacement; if replacement proof fails, the old link remains. A last-method removal is blocked with the approved recovery path, not a new bypass. Quiet hours delay only allowed personal notifications; mandatory operational escalation, alternate ownership, billing recipients and payment data remain unchanged. Personal security/preference mutations are **unresolved**; U03 links personal preferences here and operational rules to U02 according to capability.
