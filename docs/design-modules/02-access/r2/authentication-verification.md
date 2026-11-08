---
type: VerificationPlan
title: Authentication integration acceptance and review
description: Maps login directory federation and recovery requirements to future negative tests and actual document review evidence.
status: draft
---

# Scope and evidence

[HB-60](https://linear.app/hybrid-billing/issue/HB-60) adds the [authentication integration proposal](authentication-integration.md) at the owner's request. Root is sole writer; security and operations QA reviewers are independent/read-only. Base is fetched origin/main `b48b23c`; HB-59 is referenced at immutable `adf77cf` and is not imported or declared merged. Existing v10 remains active. User scope is planning documents only.

The linked specification cites primary IETF RFCs, OASIS SAML Core/Profiles/Bindings and OpenID Foundation Core/Discovery/logout standards. Each protocol section distinguishes standards from narrower proposed product choices. No online IdP, directory, credentials or customer data was used. Vendor/adapter/version selection, actual limits and executable endpoint bindings remain gates.

# Future acceptance cases

Every case is NOT_RUN. Pass requires observing the actual enforced outcome across UI/backend/workers and retaining non-sensitive versioned evidence; a written scenario or synthetic mock alone is insufficient for external interoperability claims. Each tested profile records provider/directory and adapter versions, configuration digest, browser/deployment mode, input class, expected/observed result, safe audit reference and actual times. Assertions, tokens, passwords and customer data never enter the report.

| ID | Scenario | Required outcome and recovery evidence |
| --- | --- | --- |
| AUTH01 | Approved method authenticates invited active member | Fresh rotated session; only current authorized workspace/customer scope; request/provider/configuration lineage |
| AUTH02 | Valid identity has no membership; same email exists elsewhere | Pending/access request only; no merge, membership, role or account-existence disclosure |
| AUTH03 | Workspace/connection/return URL tampered or callback from another browser | Reject bound transaction; no session or redirect to unapproved destination; safe restart |
| AUTH04 | Expired/replayed transaction or concurrent identical callbacks | At most one session creation; stale replay denied; no second consumed-proof session |
| AUTH05 | LDAP StartTLS refusal, bad certificate or unapproved failover/referral | No credentials sent before verified TLS, no downgrade/credential forwarding; connection unavailable |
| AUTH06 | LDAP empty password/DN, injection, zero/multiple search matches | Generic login failure; no anonymous success or widened search; safe diagnostic reason only |
| AUTH07 | LDAP rename versus object deletion/recreation | Rename retains immutable identity; new object ID cannot inherit old grants or identity links |
| AUTH08 | Directory disabled/locked state or incomplete groups | No eligible authentication/derived grant where proof is missing; state-specific admin evidence without user enumeration |
| AUTH09 | Paged directory scan fails halfway or reaches limit | Incomplete scan, no inferred mass removal or freshness renewal; retry from verified checkpoint under current rights |
| AUTH10 | Signed SAML with wrong issuer/audience/destination/recipient/InResponseTo | Reject before session; neither valid signature nor email overrides target mismatch |
| AUTH11 | SAML unsigned assertion, duplicate XML ID, wrapping, DTD or extra auth assertion | Reject consumed-node ambiguity/unsafe XML; no network entity fetch; bounded parse work |
| AUTH12 | SAML expired/future/replayed assertion or unsolicited login | Deny according to approved skew and request profile; one-use replay store persists across relevant nodes |
| AUTH13 | SAML POST returns without ordinary Lax cookie | Reviewed browser-correlation mechanism works or rejects safely; no disablement of global CSRF/session controls |
| AUTH14 | OIDC wrong state/nonce/issuer/aud/azp, missing PKCE or changed redirect | Reject transaction/token; code exchanged only against configured issuer with correct verifier |
| AUTH15 | OIDC unknown kid/untrusted key URL/discovery mismatch/UserInfo sub mismatch | Bounded trusted-key refresh only; no arbitrary outbound fetch or identity switch; otherwise reject |
| AUTH16 | ID token presented to product API; upstream scope or group claims imply admin | Deny wrong credential class and unauthorized power; application authority stays separately scoped |
| AUTH17 | MFA required but weak/missing/stale assurance or changed epoch | Block sensitive command; fresh verified step-up under approved provider mapping; no self-approval bypass |
| AUTH18 | Linking by matching email, already-linked subject or unavailable old provider | Reject shortcut; require existing/new proof or separately governed recovery; preserve last usable method |
| AUTH19 | Member/group reduction during API query, export/download or queued job | Enforce current deny at documented checkpoints; no old-authority output or stronger service-account fallback |
| AUTH20 | Local logout succeeds but IdP SLO times out/unsupported | Product session remains invalid; upstream status unknown/unavailable; never global-success claim |
| AUTH21 | Forged/replayed/expired or missing-exp/cross-issuer logout callback | No unrelated session termination; validate profile and scope, legitimate matched session revoke only |
| AUTH22 | Routine signing-key rotation versus unproven root/subject migration | Bounded proven rotation only; root change needs independent trust approval/new epoch; no resurrected sessions |
| AUTH23 | Metadata/JWKS/LDAP destination redirects to prohibited network | Reject under ED-03 before credential disclosure; approved private endpoint still requires reviewed scope |
| AUTH24 | IdP outage/directory stale evidence and attempted local fallback | New affected login blocked; sensitive actions fail closed; no ordinary local fallback or infinite cache |
| AUTH25 | Emergency access and backup restore after revocation | Independently verified bounded custody; no implicit financial authority; latest deny/epoch applied before restored access |
| AUTH26 | Activation timeout or changed preview/configuration revision | Reconcile original operation; no duplicate blind activation; changed digest requires new validation/approval |
| AUTH27 | CSRF/fixation/idle or absolute expiry and trusted-proxy header spoof | Reject unauthorized mutation/session/forged scope; rotate/expire server-side; safe fresh login |

# Open implementation decisions

Accountable owner assigns actual security/operations custodians before enabling profiles. They must resolve: maintained library/broker and supported providers, local authenticator choice, group/disable semantics and sync bounds, assurance mappings, role/permission binding, session/clock/rate/network budgets, same-site callback/proxy topology, key provenance/overlap, directory account limits, optional SLO, recovery proof and retention of audit/replay/session data. Missing values block dependent activation, not document review. No unsupported protocol is silently enabled for compatibility.

# Actual review and document QA

Pre-write security and operations QA plan reviews: PASS, with constraints incorporated in the draft. Actual-document security review: PASS; scoped re-review also passed after explicit SAML session cap, untrusted OIDC additional-audience rejection and logout-token expiry requirements were added from primary-source checking. Actual-document operations QA: PASS. Reviewers read the full draft independently and made no file edits; their verdicts are textual contract review, not executed protocol tests. Root metadata/local-link checks for four scoped documents, exactly 27 unique AUTH cases and six exact canonical trust-command references: PASS using a local read-only assertion script. `git diff --check` and scoped staged review: PASS. Primary standards were opened by root; no external compatibility claim is inferred. Runtime security, provider/directory interoperability, browser/cookie/CSRF behavior, logout delivery, scale and failure injection: UNVERIFIED / NOT_RUN.

This planning revision neither closes earlier runtime findings nor accepts a vendor or supersedes the entire v10 module. PR/merge and machine release are tracked in HB-60; ticket completion requires authorized merge and remote main read-back.
