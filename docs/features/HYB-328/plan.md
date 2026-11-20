---
type: ImplementationPlan
title: P07 service binding and runtime authority prerequisites
description: Evidence-based boundary design for product rights, plugin license identifiers and original distribution notices.
status: draft
---

# Scope and ownership

[HYB-328](https://linear.app/myblueton/issue/HYB-328), owner Seung Woo Park, machine mac_mini; base main d7b6a3d. Root owns implementation and integration. Independent reviewer owns read-only boundary review; Claude CLI ui-designer owns material visual composition before screen changes. Preserve the running FOCUS preview and all unrelated branches. No merge, deployment, real entitlement issuance or external message is authorized.

# Evidence and missing inputs

Available: [license scope](../../../LICENSING.md), [product policy](../../licensing/product-policy.md), [owner mode requirements](../../licensing/usage-modes-requirements.md), [HB-252 implementation](../HB-252/plan.md), [HB-253 implementation](../HB-253/plan.md), [HYB-309](../HYB-309/plan.md), current P07 preview and the service binding register. HYB-309 supplies signed manifest license identifiers and installation allowlist state, not original notice contents.

The original P07 specification and referenced planning requirements/ui-design files are absent from this checkout and available worktrees; no history for the P07 specification was found in local Git refs. The owner's prior instruction permits continuation without restoration. It does not make absent source available or certify a reconstructed specification. This design uses the available owner requirements and identifies uncovered contracts explicitly.

# Findings

1. P07 is excluded from the live service switch. It renders a preview and makes no licensing API request.
2. License mode and third-party registries exist as tested in-memory domain modules. Neither is instantiated by the API server.
3. Reading license status changes high-water time, expiry latches and last-verification evidence. A nominal GET is therefore not sufficient to avoid durability requirements.
4. Third-party `LicenseText` and `Notice` fields are hashes, not original text or artifact locations. Plugin manifests supply license identifiers only.
5. Current P07 bundle links cover seven frontend dependencies. Their original LICENSE files are emitted at build time; development serving must be checked separately.
6. Existing preview text implies operation restrictions after expiry, contradicting the owner's nonblocking expiry requirement. Correct this when the screen is revised.

# Binding boundary

Keep five independent facts: public project license, declared product-use mode and verified entitlement, support agreement, plugin installation policy, and original third-party/distribution notices. Never infer a grant from a selected mode, company/email, allowed plugin identifier, valid signature alone or missing data.

A bounded first read surface may report available plugin identifiers and original build artifacts, with explicit `not_connected` for product entitlement authority and unavailable original plugin notices. It must not be called complete product-license binding. Missing inventories must not be represented by a verified empty list. The existing seven frontend notices remain separately scoped to that bundle.

Use existing authenticated session, installation-level authorization and workspace admission. Do not expose all plugin or installation facts to an ordinary workspace member. Reject obsolete responses after session/permission changes. Preserve loading, denied, failed, empty, unavailable and unknown-contract states separately. Never turn an unknown server enum into active or licensed.

# Prerequisites before entitlement mutations

- Stable installation/product identity bound server-side; never use a workspace ID or a client-supplied company as a replacement.
- Durable transactional state for mode, accepted signed documents, grants, activation interval, key trust/compromise, high-water time, expiry latches and last verified evidence. Status mutations require the same durable commit boundary as explicit commands.
- A separately established time authority, explicit offline uncertainty and a reviewed restore/replay protocol. `time.Now()` alone does not satisfy the accepted requirement. Unavailable time means unverified, never expired, newly active or a restarted evaluation.
- Trusted issuer provenance and public-key custody/rotation. No signing private key enters the installation, client, repository or evidence.
- Authenticated human actor mapping, idempotency, revision conflict handling and no implicit activation from selecting a mode.
- Artifact storage joining verified digests to bounded original LICENSE/NOTICE bytes, release version and source provenance. Preserve originals, disable HTML execution and avoid arbitrary server filesystem paths or remote URL fetching.

The 336-hour evaluation starts only from an express accepted grant. Expiry warnings stay nonblocking; historical access, export and normal workflows continue subject to their existing permissions. Contact remains user-initiated and does not grant rights. Support and marketplace activation remain distinct unavailable contracts until real producers exist.

# Verification and independent review

Required future checks: authorized/unauthorized and second-workspace reads; source failure versus empty inventories; exact original notice bytes in development and built serving; tampered/missing artifact rejection; malformed enums/collections; stale-session responses; public versus product/plugin rights copy; EN/KO/AR desktop/mobile and keyboard/RTL; no expiry-based workflow block. Activation requires additional restart/restore/rollback/replay and time-authority tests before claiming LM-01 through LM-11 service coverage.

Initial independent review PASS only for bounded read-only integration. Full activation/registration is blocked until the runtime authority design above is completed. Reviewer independently confirmed the state-mutating status read, hash-only notice references and plugin metadata limitation. This is design evidence, not an implementation, runtime verification, legal review or native-language approval.

# First executable increment

Before introducing a new licensing API, repair the existing P07 foundation: serve the seven exact allowlisted original dependency LICENSE files during local development, preserve byte identity in production output, and correct expiry copy to the owner's nonblocking rule. No layout, grant, API permission, mode selection or registration mutation changes. Allowed files: console Vite configuration, a bounded notice-serving helper and focused tests, P07 Screen copy, and this ticket's evidence. No material visual composition changes require a new ui-designer layout; rendered copy remains independently reviewed. Full HYB-328 service binding stays pending after this increment.

# Durable registry increment

Continue on the existing HYB-328 branch and PR. Root owns `modules/licensemode/durable*`, checkpoint codec/validation, focused tests and ticket evidence. Independent reviewer is read-only. Existing registry API stays available for current in-memory consumers; a new encapsulated durable facade owns its registry exclusively and exposes domain operations without a raw registry escape hatch. One directory binds exactly one server-selected installation/product; explicit CreateDurable and OpenDurable operations prevent accidental initialization on recovery. Later calls omit the installation ID.

The facade serializes every operation, including Status and read methods whose local-clock watermark changes. It commits a versioned complete checkpoint before returning any domain result. Store failure returns an explicit storage-unavailable error and poisons the facade until it is closed and reopened; no uncommitted state or active entitlement is exposed. Reopening validates the complete checkpoint present before accepting calls. A failed commit has an unknown outcome: replacement may have completed before directory sync failed, so reopening can observe the old or new complete state. All accepted claims, grant references, expiry latches, key states, proposals, histories, local/high-water times and counters survive. Invalid submissions are not stored as raw input; only the registry's existing digest/outcome history persists.

Checkpoint bytes use a strict bounded schema and checksum for accidental corruption, not authentication or operator rollback resistance. Restore validates identifiers, references, list bounds, enums and claims; unknown/corrupt state is refused without reinitialization. Initialization is explicit and separate from opening an existing installation. The local file-store adapter uses exclusive ownership, private file modes, same-directory atomic replacement and sync before acknowledgment. macOS filesystem execution is tested; Windows/Linux execution and physical crash durability are unverified. Unsupported sync/replacement guarantees fail explicitly. A stale ownership marker requires explicit operational recovery, never automatic deletion.

Time remains the separately supplied Authority port; no local clock fallback is introduced. Tests use synthetic authority only. Restore of an older complete store by an operator controlling disk remains outside local rollback resistance. Actual external authority selection and API/installation authorization remain pending; persistence alone does not activate a grant or bind P07.

Validation: restart preserves mode/activation interval and repeated grant behavior; status-observed expiry and high-water survive restart; unavailable/backward authority stays unverified; storage faults prevent all further reads/writes; corruption/unknown schema/wrong identity fail closed; one writer; invalid raw submission absent; race tests. Existing domain tests remain authoritative for signature and rights rules.

# API installation and status increment

Continue the existing ticket/branch. Root writes API server licensing adapter, CLI option, focused API tests and ticket evidence; independent reviewer reads security/design and changes. No screen layout changes. The local FOCUS preview remains untouched.

An optional `license-installation` startup setting supplies the stable installation identifier; product is fixed to `hybrid-billing`. State lives under the existing private API data directory at `license-state`. Startup opens an existing store with exact binding; malformed/wrong-bound/locked state refuses startup. Missing store is explicitly uninitialized and is never created at startup. An authenticated installation administrator may initialize through the existing revision/idempotency/step-up command boundary; the actor comes from the authenticated command context. Default omitted setting is not configured and exposes no product identity.

Add authenticated `GET /api/v1/licenses`, installation-admin only, scoped to the installation workspace. Return explicit not_configured/uninitialized/configured states, actual durable status, and time-authority connection state; no fabricated grant. Status failure returns dependency_unavailable with no retained rights. Read under the server domain lock; the durable facade serializes state-changing reads. Return only explicit allowlisted fields, never verification keys, rejected payloads or filesystem paths.

Add commands `licenses.initialize`, `licenses.select_mode`, `licenses.register` and `licenses.activate`, through the existing command envelope. Every target is the stable installation resource `license-installation`; foreign workspace/target is absent. Canonical standard-base64 body is split into required `body` (at most 4096 characters) and optional `body_tail` (at most 1368 characters, only after a full 4096-character first chunk). Combined decoded bytes are bounded at 4096; signature is exactly 64 decoded bytes. Exact evaluation terms are validated by the domain. The existing command value limit is unchanged. Selection never activates evaluation. Signed registration returns its actual domain outcome; activation refusals retain their distinct stable outcome without reporting active rights. Successful activation or domain replay returns `activation_recorded`, not a claim that the interval is currently active. Mutations require the existing humanActor check before constructing a human domain actor. Raw storage failures follow the command framework unknown-operation path, not a settled dependency refusal. No issuer key is auto-provisioned and no grant is seeded. Trusted issuer management remains a separate required connection; a valid-key fixture is confined to tests.

Authority is an explicitly supplied server configuration port; nil uses an unavailable authority. Do not map the API's wall/server clock to authoritative time. The CLI currently has no real authority provider; status/registration remains explicit about that limitation. Production provider selection, issuer management and live P07 remain pending, not falsely certified by an injected test clock.

Validate authorization, CSRF/reauthentication, wrong workspace/target, initialization/restart binding, second writer/corruption failure, selection without grant, invalid/unknown-key inputs, actual synthetic signed registration/activation with injected authority, missing/backward authority, storage failure, command replay and OpenAPI. Registration/activation domain replay survives restart; generic command-envelope receipts and identity sessions remain the existing service's lifecycle and are not newly claimed durable.

# Issuer verification-key management increment

Expose the existing dual-controlled license key lifecycle through the authenticated installation command boundary: propose_key, propose_retirement, propose_compromise, approve_key_change and withdraw_key_change. Root owns API licensing files/tests/docs; independent reviewer is read-only. No private signing material, key generation, real grants, remote authority or rights ownership determination is introduced.

Proposals require an authenticated human installation administrator, current revision, rationale and step-up. Public Ed25519 key/proof and integer UTC validity bounds use the existing domain validation; issuer identity is an explicit operator assertion, not inferred from email. Key possession proof does not prove legal commercial licensing authority. Domain self-approval, non-proposer withdrawal, retired/compromised and authoritative-time checks remain authoritative at approval time. Proposal numbers use stable license-change-N targets; only the configured installation workspace resolves them. Missing/unavailable store retains existing fail-closed behavior. Raw storage errors keep outcome unknown.

Add scoped reads by known key ID and proposal number. No unbounded inventory or list completeness claim. Return key identity, issuer, SHA-256 public-key fingerprint, validity and retirement/compromise state. Proposal read includes kind/state/proposer/approver and exact requested public-key fingerprint/issuer/bounds or target key/retirement time. Do not expose private material, raw possession proofs or arbitrary filesystem paths. Successful command responses return proposal number and stable target for review. Approval reads the durable domain proposal and does not recreate a proposal from browser fields.

Validation: real HTTP two-admin key proposal/approval followed by signed entitlement registration; self-approval and non-proposer withdrawal refused; invalid possession proof and private-key-shaped input rejected without persistence; compromise invalidates accepted rights; retirement refuses unavailable time; permission/CSRF/step-up/workspace and canonical numeric target checks; proposal retention after restart and unknown commit handling. No product source change to identity membership scanning is bundled with licensing. The 100000-person scale timeout remains separately recorded; standard server regression excludes explicit scale tests and is not presented as a full scale PASS.

Design refinements: target resolution performs syntax/workspace checks only, without a durable read before authorization. Key/proposal reads authenticate before touching storage. Self-approval maps to the declared self_approval_denied response; non-proposer withdrawal maps to permission_denied. Other domain refusals keep explicit outcome names. Missing authorized lookup returns state:not_found; malformed identifiers return invalid_request. Proposal reads are addressed by canonical positive decimal numbers and expose the exact proposed action, not reconstructed browser input.

General regression found the four previous license commands missing explicit trust-hold rows, although the default already refused them. Add explicit refusal rows for all eight state-changing license commands and allow only proposer withdrawal of a pending key change during hold. Shared approval remains refused because it can add a key. Keep authorization, human identity, step-up and revision checks. Independent policy-extension review passed before implementation. Replace the previous numeric test assertion to comply with the repository source rule; no amount arithmetic or source-rule relaxation is introduced.

The next non-scale regression exposed the pending-proposal inventory invariant: license requests must participate in bulk revocation impact. Extend the durable facade with a bounded pending-proposer read (existing maximum 1000), commit before returning, and expose only proposer IDs internally. Add a fallible store catalog beside the existing in-memory stores. Bulk preparation obtains all counts before freezing its impact; any unavailable license store refuses preparation rather than claiming zero requests. License entries use rechecked:false because the existing domain does not reauthorize the original proposer on approval; this increment does not claim cancellation by suspension. Extend root-owned files to durable.go, bulkimpact.go, bulk.go and focused integration tests. The namespace completeness invariant must enumerate both catalogs, without exempting licenses.

Include the existing M01 bulk-impact copy map and rendered markup test in scope so the new row has EN/KO/AR labels instead of a machine identifier. Root owns routine copy; independent reviewer checks terminology in rendered context. No layout, interaction, approval semantics or native-language acceptance is changed.

# P07 live screen increment

Root owns service/licenses.ts, P07 Live/components/copy/CSS/tests, operations routing, four command type additions and ticket evidence. Independent reviewer checks permission/rights semantics and technical writing; Claude CLI ui-designer supplies visual guidance. Reuse existing full-width shell, review controls and neutral separators. Consulted GOV.UK Summary list and Error summary: adopt semantic labelled facts and focused error summaries; reject colored decorative edges and fixed page-width containers. Exact sources: https://design-system.service.gov.uk/components/summary-list/ and https://design-system.service.gov.uk/components/error-summary/.

Connect authenticated P07 to strict GET /licenses parsing, explicit not_configured/uninitialized/configured states, actual status/reason/authority and entitlement/evaluation metadata. Clear old data at reload/failure; scope all results and commands to session/workspace/CSRF, ignore late responses on replacement/unmount. Notices remain independently accessible. Support/Marketplace sources remain unavailable. No status inferred from local time and no access blocked by expiry.

Initialize and select_mode use the existing command boundary, current read revision, rationale, human checks and step-up. Offline registration accepts a bounded JSON envelope containing canonical base64 body/signature and optional evaluation terms; reject unknown keys and malformed content before submission, never persist it in browser storage or URLs. Choosing evaluation does not activate a grant. Activation requires express terms acceptance; successful registration/activation is displayed as a recorded outcome followed by a fresh status read, not inferred rights. On uncertain mutation, disable fresh mutation submission in the in-memory login session across remounts and show inspection guidance; do not silently retry with a new idempotency key. Key administration remains API-only in this increment.

Stable P07 items: data:license-status, state:loading/denied/unavailable/not-configured/uninitialized/configured, action:reload/initialize/select-mode/register/activate, field:rationale/signed-envelope/terms-acceptance; existing notices/support items preserved. EN/KO/AR and desktop/mobile verification covers real app route, error/late response, permission refusal, initialization and mode changes, invalid/synthetic registration and independent notices. No owner FIX, real grant, merge or deployment inferred.

The independent remount-fix design review approved a Session-object-keyed WeakMap controller with immutable external-store snapshots. Pending acquisition precedes dispatch; settlement always targets the captured original controller, and the current view refreshes after settlement. Full-document reload is outside this in-memory guarantee. Root also maintains the scoped P07 glossary entries; AI semantic review does not imply native-language acceptance.

# P07 verification-key screen increment

Root owns the P07 trust components/copy/style, strict service trust parser/tests, shared license command controller, command IDs and scoped documentation. Independent reviewer owns read-only contract/TW review; actual Claude CLI ui-designer owns visual guidance/re-review. Continue existing PR and branch after merging current main; conflicting test inventories, notice serving and local login boundary are retained. Combined baseline: 927 console tests and build PASS.

Within Offline, separate signed entitlement submission from progressive public verification-key administration. Known-ID reads never imply an inventory. Key records show fingerprint, issuer assertion, signing bounds and distinct acceptance deadline. Proposal reads show exact action and actor identities. New-key input is a bounded JSON envelope with public material only; no private-key creation, signing or issuer-authority certification. Retirement and compromise are separate, explicit proposals. Approval requires a different human; withdrawal belongs to the proposer. The server remains authoritative for permissions, trust hold, revisions and time eligibility.

Strict parsing binds response identities to requested IDs, canonical decimal uint64 proposal numbers and exact kind-specific shapes. Public-key hex, possession-proof hex and integer UTC bounds must match the producer contract. Confirmation binds the displayed proposal/action and read revision; editing lookup input, reload, another command or session replacement invalidates it. An unknown result locks all nine license commands using the existing session controller. Command success is validated per operation, including returned proposal target. Refresh affected reads after settlement and preserve diagnostics when refresh fails. Withdrawn proposals may read not_found; absence does not prove withdrawal.

Verify double-control refusals, file bounds, malformed responses, changed lookup/revision, pending navigation, cross-command uncertainty, missing records and EN/KO/AR desktop/tablet/mobile interaction. External time authority and legal issuer provenance remain pending. No user-runtime replacement or owner FIX is inferred.
