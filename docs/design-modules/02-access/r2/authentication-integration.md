---
type: IntegrationSpecification
title: Login LDAP SAML and OIDC integration specification
description: Defines proposed protocol profiles configuration identity mapping session enforcement and recovery for enterprise login and SSO.
status: draft
---

# Scope and decision

[HB-60](https://linear.app/hybrid-billing/issue/HB-60) follows the owner's request to add login, LDAP (correcting OLTP), SAML and SSO integration specifications. This is planning only. The request expands specification coverage beyond the earlier method-selection deferral; it does not select an IdP vendor, broker/library, deployed connection or production support claim. The proposed profiles below require owner adoption and implementation interoperability evidence. Local credentials remain a conditional method, not a new outage fallback.

Baseline is module 02 of frozen v10 at `e955befffa54a9cefb392984ab7716a67c8e05b0`. This is an additive r2 authentication specification, not a replacement of the whole module. The separate [HB-59 access administration draft](https://github.com/myblueton-sw/hybrid_billing/blob/adf77cfce977e241e6f0c6f4aff53d3936376ba8/docs/design-modules/02-access/r2/administration.md) remains unmerged evidence. Canonical [identity/session policy](../../../architecture/security-boundaries/identity-session-permissions.md), [access/trust ED-02/03](../../../contracts/security/access-and-trust.md) and [trust commands](../../../contracts/commands/planning-actions.md#authentication-trust-replacement) govern authority and publication. This document supplies protocol detail without granting permissions. Root writes; separate security and operations reviewers review; accountable owner is Seung Woo Park, machine mac_mini.

| Capability | Proposed integration profile | Explicit boundary |
| --- | --- | --- |
| Product login | Same-origin backend with opaque server session; an enabled verified authentication method supplies identity | No default public administrator; method success alone creates no membership |
| LDAP directory login | LDAPv3 search then user bind over verified TLS; approved implicit TLS or mandatory StartTLS profile | LDAP is not browser federation or automatic desktop SSO; AD compatibility requires separate tested directory schema/profile |
| SAML SSO | SAML 2.0 SP-initiated Web Browser SSO; signed Redirect AuthnRequest and POST Response with signed consumed assertion | No unsolicited/IdP-initiated login, artifact binding or arbitrary assertion acceptance in this first profile |
| OIDC SSO | OpenID Connect Authorization Code, PKCE S256, confidential backend client and exact redirect URI | OAuth authorization alone is not identity; no implicit flow or password grant; no upstream token as product API credential |
| Local login, if enabled | Explicit installation policy, reviewed password verifier/MFA/recovery design | Not enabled by protocol failure; exact authenticator and enrollment method remain decisions |

SSO describes reuse of an identity-provider login across services; SAML and OIDC are federation profiles. A directory may back the organization's IdP, but this specification does not make LDAP bind equivalent to SAML/OIDC. Protocol labels do not prove MFA. Standards references below identify their base behavior; stricter exclusions, control flows and support gates are proposed product choices.

# Common connection and exchange contract

M02 Authentication connections owns configuration; L01/L03 supply login/return/error; M01/M03 own membership/grants; M05 owns personal links/sessions. Ordinary membership administrators cannot activate trust roots. Public pages show only installation-approved connection labels, generic failures and a request ID; tenant discovery must not enumerate customer membership from email/domain.

Every connection requires: canonical installation/workspace/connection ID, protocol/profile version, display label, owner, allowed membership scope, approved network destinations, TLS trust references, secret/key references and versions, identity-attribute mapping, grant-mapping revision, enrollment policy, assurance policy, time/skew/size/retry limits, current configuration digest/authority epoch, independent activation evidence, last attempt/success and detection freshness. Unknown required limits or mapping policy blocks production activation; this document invents no deployment values. UI previews old/new digests, identity/membership/session impact and exact affected scope before approval.

Backend creates a single-use login transaction bound to intended installation, connection, workspace context, configuration revision/epoch, permitted return route, browser binding, creation/expiry and protocol correlation fields. The public request may select a registered connection, never supply an issuer/directory URL or authoritative tenant. At callback recheck the transaction, current connection/epoch, proof and current account/membership before atomically consuming it and issuing a fresh product session. A concurrent replay cannot create another session. Return destinations use server-owned route IDs or an explicit allowlist, never untrusted RelayState/state as a URL. Trust changes in flight invalidate old transactions.

Network controls cover LDAP hosts, metadata, discovery, JWKS, token/UserInfo and logout destinations. ED-03 applies to DNS, redirects, failover, TLS hostname validation, approved private networks and credential audience. Private enterprise endpoints need explicit approved scope; loopback, metadata and unapproved management destinations remain denied. No user-controlled token header or metadata URL can initiate unrestricted fetches. Client errors never expose directory DN, tokens, full assertions, credentials or forbidden account existence.

Protocol wire names in this document are standard fields, not registered product API routes. Product start/callback/logout/session-revoke/config-test/mapping-preview and directory-sync endpoint paths, request schemas, rate budgets and permission bindings are **UNRESOLVED** implementation gates. Existing `trust.propose`, `trust.validate`, `trust.approve`, `trust.activate`, `trust.reject` and `trust.cancel` remain the sole named trust-change command family. Validation and test-login do not activate a connection or create grants.

# Login and first-party session

Flow: approved login entry → select permitted method → verify identity and transaction → resolve stable principal → check current membership and assurance → create rotated session → authorized landing page. No membership yields an access-request/pending state, never automatic customer access. Cancellation or expired transaction restarts a fresh attempt. Failed attempts use generic responses with bounded per-source/connection/account defenses; provider lockouts must not trigger uncontrolled retries.

Sessions are server-held and opaque, with Secure/HttpOnly cookies, host/path isolation, idle/absolute expiry and CSRF/origin checks for mutations. Rotate identifiers after authentication, linking and privilege changes. Browser localStorage holds no bearer credential. SameSite and callback correlation must be tested per binding: SAML cross-site POST may not carry a Lax cookie. Use a dedicated narrowly scoped, short-lived correlation design compatible with POST, or a validated equivalent, without weakening all session-cookie defenses. State/RelayState alone is not proof the callback belongs to the initiating browser. Incomplete browser binding rejects the login and provides safe restart.

Approved local login, if included, requires adaptive one-way password verification, throttling, one-time bound reset/invite material, atomic token consumption and session invalidation policy. Neither LDAP user passwords nor IdP tokens become a local password. SSO-enforced members cannot choose local login to bypass policy. Bootstrap and emergency custody remain separate operator-authorized processes.

MFA/step-up policy binds required assurance, trusted evidence interpretation, authentication age, principal, connection and epoch to the sensitive operation. Verified OIDC `acr`/`amr`/`auth_time` or SAML AuthnContext/AuthnInstant must satisfy an approved per-provider mapping; presence of a claim or successful SSO is insufficient. Requested stronger authentication without verified returned evidence is blocked. LDAP password bind alone does not meet a multi-factor requirement. Recent proof does not remove independent financial/trust approval or current permission checks.

# LDAPv3 directory profile

The profile uses LDAP bind/search semantics from [RFC 4511](https://www.rfc-editor.org/rfc/rfc4511.html) and authentication/TLS rules from [RFC 4513](https://www.rfc-editor.org/rfc/rfc4513.html). Product choices are read-only directory access and no password writeback. TLS 1.2 or later is the proposed minimum, with TLS 1.3 preferred when supported and reviewed suites; legacy TLS and insecure certificate bypass are excluded following [RFC 9325](https://www.rfc-editor.org/rfc/rfc9325.html).

| Configuration | Required validation / meaning |
| --- | --- |
| Servers | Explicit host, port and implicit-TLS or StartTLS mode; reviewed failover list belonging to the same identity namespace; no automatic insecure fallback |
| TLS | Expected server name, CA bundle reference/version and approved certificate policy; hostname/chain validity before sending credentials |
| Search identity | Read-only service bind DN and secret reference/version; minimum base/attribute rights; not the end user's stored password |
| User search | Base DN, scope, administrator-controlled filter template, login attribute and immutable identifier attribute; escaping is syntax-specific |
| Groups | Group base/filter, group identifier and membership attribute semantics; direct versus nested traversal policy, cycle/depth/page limits and completeness |
| Account state | Directory-specific disabled/locked/expired semantics and freshness/detection policy; no invented universal LDAP flag |
| Runtime | Connection/search/bind timeouts, size/page limits, bounded retries, sync cadence and authoritative full-scan checkpoint; values require profile approval |

Authentication sequence: establish/verify TLS → bind search identity → search using an escaped supplied login value → require exactly one eligible user with immutable ID → open an isolated TLS connection and bind the returned user DN with the supplied nonempty password → check account eligibility and complete required mapping → discard user password and bind connection → issue a product session only after common gates. Empty DN/password, anonymous or unauthenticated bind success never authenticates a person. StartTLS refusal/timeout stops before any credentials; no plaintext retry. Search and user bind identities never share an ambiguously rebound pool connection. A race that changes object identity/state requires fresh verification, not a cached DN login.

Filter assertion values follow [RFC 4515](https://www.rfc-editor.org/rfc/rfc4515.html); do not concatenate login text into filter syntax. DN construction uses a separately reviewed DN encoder, not filter escaping. Zero/multiple results, missing immutable ID or incomplete required group enumeration fail closed. Generic user errors conceal whether search, bind or eligibility failed; authorized diagnostics retain safe reason codes.

Principal key is installation + connection + immutable directory object ID; DN/login/email are mutable attributes. [RFC 4530 entryUUID](https://www.rfc-editor.org/rfc/rfc4530.html) is one standardized identifier; a vendor-specific equivalent requires tested encoding/stability rules. Rename preserves identity; delete/recreate with a new object ID never inherits old links/grants. Referrals are not followed in this first profile; a separate approved referral/credential-boundary design is needed. Nested groups are unavailable until their complete traversal policy is selected and verified.

Synchronization is distinct from authentication. A full bounded scan publishes only after pagination/completeness and source checkpoint are verified. Partial/failed scans never mean all missing members were removed, nor renew stale grant freshness. Apply observed explicit disables promptly; unresolved records remain subject to freshness/deny policy. LDAP provides no universal product-session logout or push-revocation signal here. Show last attempt, last complete verified observation and detection deadline; after the approved bound, new login and sensitive operations needing that evidence are blocked. No cached password fallback.

# SAML 2.0 Web Browser SSO profile

The first product profile uses SP-initiated Redirect request and POST response, described by [OASIS bindings](https://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf) and [Web Browser SSO profile §4.1](https://docs.oasis-open.org/security/saml/v2.0/saml-profiles-2.0-os.pdf). This is a deliberately narrower product proposal than all SAML options.

| Configuration | Required validation / meaning |
| --- | --- |
| SP | Stable SP entityID, exact HTTPS ACS URL, request-signing key reference/certificate, expected bindings and optional decryption-key reference |
| IdP | Expected entityID, approved metadata source/import digest, SSO URL/binding and trusted signing certificate set with provenance |
| Subject | Persistent non-reassignable subject contract, NameID format/qualifiers or approved signed stable attribute; transient/email-only identity cannot create a permanent link |
| Assertions | Allowed signature/digest algorithms, required signed consumed assertion, bounded message/XML size and time/skew policy; SHA-1/unsigned profile excluded |
| Claims | Explicit signed display/contact/group mappings and assurance mapping; absent or truncated required attributes block dependent grants |
| Logout | Optional supported SLO endpoint/binding, session identifiers and validation profile; unavailable until separately configured/tested |

Request contains fresh ID, IssueInstant, SP Issuer, registered Destination/ACS and requested NameID/assurance policy. Sign the Redirect request using the vetted library's binding rules; RelayState is an opaque transaction reference with integrity/browser binding, not authority or an open redirect. Response uses HTTPS POST `SAMLResponse` and bound RelayState. No arbitrary alternate ACS or unknown connection is selected from incoming data.

Accept only successful, correlated responses with expected issuer, destination, assertion audience, recipient, validity and subject confirmation. Validate every signature present and require a signature on the exact assertion consumed; reject ambiguous duplicate IDs, extra authentication assertions and mismatched signed/used nodes. Enforce response and bearer confirmation InResponseTo against the one-use request and retain response/assertion replay records for the accepted validity window plus approved skew. Subject, AuthnStatement and Conditions follow [SAML Core](https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf); a valid signature alone is insufficient. Reject expired/future/unsupported conditions and missing required subject/session evidence. If AuthnStatement supplies SessionNotOnOrAfter, cap the product session accordingly or reauthenticate before continuing.

XML is processed by maintained validated libraries with external entities/DTD/network schema retrieval disabled and bounded decompression/parse work. Encrypted assertions, if supported, require approved decryption plus signature/semantic verification; encryption is not authenticity. Browser-controlled KeyInfo cannot establish trust. XML security library, algorithms and certificate rollover policy must pass implementation review before any compatibility claim. An IdP supporting only an unsigned assertion is incompatible with this proposed profile, not grounds to disable checks.

# OIDC Authorization Code profile

Identity claims and token validation follow [OIDC Core §3.1](https://openid.net/specs/openid-connect-core-1_0.html). The product uses a backend client and explicit state/nonce in addition to [PKCE S256](https://www.rfc-editor.org/rfc/rfc7636.html); code flow and redirect protection follow the [OAuth security BCP](https://www.rfc-editor.org/rfc/rfc9700.html).

| Configuration | Required validation / meaning |
| --- | --- |
| Provider | Exact issuer; approved discovery URL or pinned metadata; authorization/token/JWKS/UserInfo endpoints and allowed origins |
| Client | Registered client_id, approved backend authentication method and protected secret/private-key reference/version; no browser client secret |
| Redirect | Exact registered HTTPS callback and optional post-logout redirect; no wildcard or caller-controlled host |
| Claims/scopes | `openid` required; minimum additional claims/scopes; stable sub, approved group/assurance mapping; offline access disabled unless separately needed/approved |
| Validation | Approved signing algorithms, token/clock bounds, key-cache freshness and bounded unknown-kid refresh; `none` and attacker-controlled key URLs rejected |

Start: response_type=code, scope including openid, client_id, exact redirect_uri, random single-use state/nonce and code_challenge with S256. Backend keeps the transaction-bound verifier. Callback requires the bound state/browser/provider context; exchange code only at the configured token endpoint with identical redirect, verifier and registered client authentication. No code or token in application logs/history/analytics; callback landing strips protocol parameters after consumption.

Verify signature using trusted issuer keys and check exact iss, aud containing this client with no untrusted additional audience, applicable azp, exp, issued-time policy, nonce and required auth_time/assurance. This product requires azp=this client for multi-audience tokens and checks it whenever present. UserInfo, if used, must return the same sub as the validated ID token. Bind identity to connection + iss + sub. Email/email_verified does not authorize account merging. ID tokens never authorize product API requests. Upstream access/refresh tokens stay server-side, only for approved purposes; product session expiry/revocation remains independent.

Discovery issuer must exactly match configured issuer; endpoint/key retrieval is bounded by the approved profile and ED-03, not trusted solely because metadata names it. Use [OIDC Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html) metadata validation. Unknown key IDs may trigger a bounded refresh from the already approved JWKS endpoint; never fetch a token-supplied URL. Failure to verify a new token rejects that login. Existing sessions still require current epoch/authority and approved evidence freshness, not indefinite stale keys or grants.

# Account linking, membership and group mapping

Authentication resolves a principal; membership is a separate installation/workspace grant. Initial design uses invited/pre-provisioned memberships. Just-in-time enrollment is conditional on a separately approved restricted policy and is not enabled by a successful login. SCIM, Kerberos/NTLM desktop SSO, password writeback and provisioning deletion APIs are outside this profile; their absence must not be presented as working deprovisioning.

Mapping fields: connection/group immutable ID or exact trusted claim value, eligible workspace/customer set, permitted action/field/period ceiling, effective bounds, source freshness, policy revision, proposer/approver and removal semantics. Group names are not globally unique. Unknown/truncated/overage claims are incomplete evidence, not proof of no groups or permission to fetch arbitrary links. Privileged administrative, trust and financial approval grants require their separately approved authority path; directory groups cannot silently confer them. Mapping preview shows additions/reductions and conflicts, but cannot publish until approved.

Effective grants intersect active membership, approved delegation/role ceiling, current external eligibility where required, resource scope and command conditions. A mapping may reduce its own derived grants, not erase independently justified grants or mutate historical approvals. Denies take precedence according to the approved policy. Missing source-specific removal/precedence policy blocks activation; no assumed union of all roles. A role reduction revokes affected sessions/step-up or forces current re-evaluation across API, query, jobs and export, not only the menu.

Linking requires current trusted reauthentication plus proof of the new identity, with transaction-bound intent and uniqueness checks. A destination identity already attached elsewhere is blocked for governed review. If the old method is unavailable, separately verified recovery proves legitimate authority; the proposed new IdP cannot vouch for its own takeover. Unlink checks last usable method/custodian and does not delete global identity or other memberships. Migration preserves audit lineage and never reuses revoked identity keys.

# Trust rollout, rotation, logout and recovery

M02 workflow: draft → validate network/schema/identity/mapping and synthetic test → independently approve exact digest → activate with current authority → observe new epoch and invalidation propagation. `trust.*` commands apply to root/issuer/namespace/assurance weakening changes. Test success is neither activation nor authorization. Unknown activation result reconciles the original operation; no second blind activation. Rollback is a fresh governed change with a newer epoch, never restoring revoked sessions.

Verified routine signing-key rotation is distinct from root replacement. A preapproved rotation policy must bind unchanged issuer/identity namespace, authentic key provenance, allowed algorithms, overlap/retirement times, cache behavior and rollback constraints. Unproven metadata/key changes go through root-change review. Never accept a new signing certificate simply because it arrived inside an assertion. TLS CA changes, LDAP identity-attribute migrations and changes that weaken assurance receive independent impact review. Expired/unverifiable required keys make affected verification unavailable; health shows attempts and success separately.

Local logout first invalidates the current product session server-side, then clears its cookie. Optional all-product-session revocation is separately scoped and records durable deny versus pending cleanup. Upstream logout is an additional best-effort, capability-specific operation; UI can say 'Product session ended; identity-provider session unknown' and offer a fresh login that still checks current membership. External logout failure cannot resurrect the local session. No guarantee of logout from all other enterprise applications is made.

If configured, [OIDC RP-Initiated Logout](https://openid.net/specs/openid-connect-rpinitiated-1_0.html) uses the registered endpoint, permitted ID-token hint and exact registered post-logout redirect with correlated state. Local logout never depends on its callback. Optional [OIDC Back-Channel Logout](https://openid.net/specs/openid-connect-backchannel-1_0.html) validates signature, issuer/audience, required iat and exp with freshness/expiry checks, required event, jti replay and sid/sub binding; nonce must be absent. It revokes only matched issuer/client sessions. SAML SLO requires trusted signed messages, issuer/destination/time/request/replay checks and mapped NameID/SessionIndex; it is conditional until separately tested. Unsolicited SLO messages do not use the login callback. LDAP unbind closes the directory connection, not all product sessions.

On IdP/directory outage: block new affected login; preserve last verified evidence visibly; enforce current product denies and evidence freshness for existing sessions. Sensitive operations fail closed when current authority cannot be established. An approved bounded low-risk read policy, if any, must be explicitly documented; no implicit fail-open cache. Ordinary local login is never switched on automatically. Named recovery custodians use independently verified proof, time/scope-limited emergency access and post-use revocation/audit; recovery cannot self-approve finance or weaken independent trust approval. Restores remain isolated until latest authority epochs and deny/deletion controls are applied.

# Required implementation decisions and acceptance

Before enabling any profile, security/operations owners select maintained broker/adapters and versions, tested provider/directory combinations, exact routes and schemas, TLS/signature algorithms, timeout/skew/session/rate limits, directory nested-group and disable semantics, MFA enrollment/assurance mapping, group removal policy, recovery custody and logout support. Secrets remain only in approved storage. Configuration export includes references and public metadata under current permissions, never secret values. No test user credentials or real directory contents were requested or used here.

[Verification and scenarios](authentication-verification.md) define acceptance and record actual document review. All protocol interoperability, browser, security-enforcement and failure-injection tests remain NOT_RUN. Source references were checked as primary standards; this is not certification or a full security implementation.
