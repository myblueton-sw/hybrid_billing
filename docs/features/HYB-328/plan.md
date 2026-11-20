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
