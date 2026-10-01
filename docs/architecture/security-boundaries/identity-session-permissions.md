---
type: Security Design
title: Identity session token and authorization policy
description: Defines proposed self-hosted authentication, session and credential lifecycles with current scoped authorization and revocation.
status: draft
---

# Identity session and permissions

Part of [HB-7](../architecture-readiness.md), refining local v1.16 `retention-access.md` and HB-03/HB-42 requirements. Policies below are proposed design obligations; durations, IdP products, algorithms/library versions and implementation remain unselected. No credentials are created here.

## Authentication and stable identity

Support design includes local credentials, SAML/OIDC and LDAP/AD integration. Compare a maintained identity broker with validated protocol adapters; do not build cryptographic protocols. User IDs remain stable. Identity namespaces include provider connection and trusted issuer/subject; same email/domain/name cannot link accounts or grant membership. Account linking requires existing reauthentication plus new identity proof or a separately authorized verified process.

OIDC uses Authorization Code with PKCE, state/nonce and validated issuer/audience/signature/time/redirects; ID tokens are not API access tokens. SAML validates the consumed assertion, request binding, audience/recipient/time and replay with trusted keys. LDAP uses verified TLS and stable directory IDs, rejects empty/anonymous bind and escapes filters; no stored user passwords or automatic SSO fallback. Local passwords use a reviewed adaptive one-way verifier; provider secrets are encrypted in the approved secret store. Existing source conditions: local `retention-access.md:24–53`. [OIDC Core](https://openid.net/specs/openid-connect-core-1_0.html), [OAuth security BCP](https://www.rfc-editor.org/rfc/rfc9700.html), [OWASP SAML](https://cheatsheetseries.owasp.org/cheatsheets/SAML_Security_Cheat_Sheet.html), [OWASP LDAP](https://cheatsheetseries.owasp.org/cheatsheets/LDAP_Injection_Prevention_Cheat_Sheet.html).

SSO is not automatic MFA proof. Sensitive finance/access commands require approved authentication strength and sufficiently recent step-up evidence. Organization SSO enforcement must not be bypassed by ordinary local login during an IdP outage. Recovery and time-limited break-glass are separate audited, scoped paths.

## Session and token classes

| Class | Proposed policy | Durable metadata and controls |
| --- | --- | --- |
| Browser session | First candidate: opaque server-side session through same-origin backend; cookie scoped with Secure, HttpOnly and appropriate SameSite; CSRF defense for mutations | Session ID verifier, user, installation, authentication/step-up time, idle/absolute expiry, revocation version; no browser localStorage bearer secret |
| API access token | Short-lived scoped OAuth token or approved opaque credential; enforce issuer/audience/client/expiry/current grants | Client/resource scope, subject/delegation and current credential/account state; JWT signature alone never grants permanent access |
| Refresh token, if issued | Bound to client/scope; rotate or sender-constrain; atomic family transition and replay response | Family/verifier digest/expiry/revocation; define multi-tab/concurrent refresh handling without indefinite reuse |
| Service account credential | Separate from people, minimum endpoint/action/resource scope, named owner, expiry/rotation/revocation | Protected verifier for locally issued secret; upstream CSP/ERP secrets stored by secure reference; never human approval authority by default |
| Bootstrap/invite/reset | Short-lived one-time purpose and target binding; atomic consumption, reissue invalidation and bounded attempts | Verifier digest, target/workspace/role ceiling/purpose/expiry/used/revoked; no raw token in logs, audit or telemetry |
| Downloads and support sessions | Current scoped checks for immediate revocation; explicit lifetime/custody for any signed URL | Artifact/version/recipient/purpose; actual support actor, target, reason, expiry and allowed actions |

Rotate session identifiers after authentication and privilege changes; enforce idle and absolute expiration on the server and invalidate server-side state on logout. Cookie settings alone do not establish CSRF protection; verify origin/CSRF token and chosen deployment flows. Reverse-proxy trust and TLS termination must be explicit. [OWASP session management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html), [OWASP CSRF](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html).

Public-client refresh tokens require rotation or sender constraints under OAuth security guidance. If rotation is used, detect reuse and revoke the active family; document concurrent legitimate refresh behavior. The same-origin session proposal does not require exposing refresh tokens to browser JavaScript. [RFC 9700 refresh tokens](https://www.rfc-editor.org/rfc/rfc9700.html#section-4.14).

## Authorization policy

Effective authority is the intersection of current account/membership state, role/delegation grants, client scopes, resource/customer/workspace ownership, permitted fields/time and command-specific conditions. Deny by default. Derive tenant context from authenticated membership and validated resource scope; a submitted workspace ID, org parent, MSP relationship or globally unique ID is insufficient. Delegation chains cannot exceed grantor authority, expiry or explicit re-delegation limits.

Permission evaluation governs API, async workers, search/aggregates, caches, downloads, analytical queries and model retrieval. Customer portals receive a permitted projection rather than unrestricted data with hidden columns. Roles describe responsibility; they are not a universal superuser grant. Installation operators manage runtime; access administrators manage bounded memberships; billing preparers calculate; approvers review a fixed snapshot; issuers/submission workers act on authorized approval; customer viewers and auditors receive defined read scopes. Self-approval/dual-control policy must be explicit and cannot be erased by holding multiple roles.

When PostgreSQL RLS is adopted, use an application role that cannot bypass it and verify pooled tenant context cleanup. Owners/superusers and bypass roles have special behavior; RLS complements application authorization and does not constrain a customer OS/DB root administrator. [PostgreSQL RLS](https://www.postgresql.org/docs/current/ddl-rowsecurity.html). Self-hosted privileged infrastructure access is a documented trust boundary with customer governance, not a claim of tamper-proof tenant isolation.

## Revocation and outage contract

Commit authoritative deny/version before reporting revocation effective. Distinguish product enforcement from later token/cache cleanup and from external IdP termination detection. Sensitive writes, issuance, exports/downloads and resumed jobs check current authority; stale or unavailable authority fails closed. Low-risk reads may use only a specifically approved bounded cache policy; expiry and allowed staleness are unresolved owner/security decisions, not implicit defaults.

Cross-node session/cache propagation and in-flight long exports require defined revocation checkpoints and measured maximum delay. Pending human jobs recheck scope and ownership; no implicit promotion to a stronger service account. External irreversible dispatch or already downloaded files cannot be revoked retroactively. Restore stays quarantined until latest revocation/deletion decisions are verified and applied. Product database rollback cannot revive an old invitation or grant.

Bootstrap creates one initial administrator through atomically consumed operator-held material, with no public default credentials. Test last-admin protection, IdP misconfiguration recovery, backup/restore and emergency-access expiry. Emergency access requires named custody, reason, limited scope/time, independent audit and post-use revocation; it cannot silently approve or rewrite financial records.

## Decision register and acceptance

The proposed [access/trust contract](../../contracts/security/access-and-trust.md) specifies mandatory analytical enforcement, governed authentication-root replacement/epoch handling and conditional engine callers. Use its negative acceptance cases with these current-authority/session obligations; actual profiles/mechanisms/limits remain decisions.

Seung Woo Park obtains named security/operations owners for: session idle/absolute limits, step-up lifetime, issued token/refresh policy, revocation/detection bounds, SSO/MFA products, service-client flow, signing/key rotation overlap, recovery custody, rate limits and audit/profile retention. Values remain UNRESOLVED.

Planned cases: fixation; cookie/CSRF/cross-origin failures; logout and expiry; invalid issuer/audience/signature/nonce; SAML replay; LDAP injection/empty bind; same-email identities; refresh theft/concurrency; tenant guessing; MSP delegation expiry; role-reduction across API/cache/query/model/export/queued work; external outage; last-admin and backup resurrection. Acceptance requires current scoped authority and preserved financial separation. All runtime security cases are **NOT_RUN**.
