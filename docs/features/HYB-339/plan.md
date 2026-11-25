---
type: ImplementationPlan
title: Index state-pending invitations without changing quota decisions
description: Removes terminal invitation history from pending-limit scans while retaining canonical records and existing authority/time semantics.
status: draft
---

# Evidence and ownership

[HYB-339](https://linear.app/myblueton/issue/HYB-339) is assigned to Seung Woo Park, Bug, In Progress, machine:mac_mini. Root owns implementation/integration; independent security reviewer is read-only. Branch fix/HYB-339-pending-invitations was created from verified origin/main 0f9d9a2 then rebased onto reviewed dependency fix/HYB-338-membership-guard at 87f45f3. Its eventual draft PR is based on that dependency branch to isolate this change. Neither dependency is merged; the same Mac owns each isolated ticket. Missing companion files retain the owner's no-restoration authorization.

Noninvasive sampling of the unchanged API scale test on HYB-338 locates fixture population in InviteUntil/countable, alongside sorted insertion and session operations. InviteUntil currently scans all historical invitations, including accepted records, for each new invitation. A pending-state index avoids rescanning terminal history; it does not claim to solve every scale cost.

# Design

Maintain a workspace-local set of state-pending invitation IDs, with canonical records remaining in r.invitations. Initialize it with the registry and add only after successful InviteUntil creation. Under the same registry mutex, remove an ID exactly when its canonical record successfully becomes accepted, revoked or locked. Keep historical records and global invitation-ID conflicts unchanged. Do not delete on failed acceptance, version overflow, expiry, future NotBefore, authority loss or any nonterminal condition.

Quota checks iterate indexed canonical records and invoke the unchanged countable predicate. Do not cache an effective count or substitute validity.at for the current expiry/authority semantics. Pending records that are expired or ineffective remain indexed and may accumulate; this bounded correction addresses terminal history only. All supported records currently enter through registry construction and InviteUntil; no persistence/import reconstruction is added or claimed.

Allowed edits: modules/identity/registry.go constructor/field, workspace.go invitation paths, focused new invitation-index/property/scale tests and docs/features/HYB-339. Preserve HYB-338 guard lines unchanged. No public contract, authentication, quota, installation, backend seed or frontend change. Root is the sole writer.

# Verification and review

Use the existing standard-library model/property pattern. Compare index membership and countable counts against full canonical scans after public operation sequences, including accepted/revoked/locked histories and pending failures. Verify exact quota boundaries, multiple workspaces, historical ID nonreuse, wrong verifier before/at lock threshold, future/expired validity, authority/time changes, version overflow without removal, and concurrent creations at quota. Meaningful assertions must cover both accepted and refused operations.

Execute existing identity race/model/source tests and vet. Measure long accepted/revoked terminal histories while keeping the same public quota semantics. Run the original TestScaleFilterListsOverHTTP unchanged, with the full 100,000-person population and HYB-338 dependency. Preserve test populations and assertions; do not call a benchmark full API readiness. Obtain independent design/change/evidence review and fix blockers before PR progression. No merge, deployment, real accounts or external effects are authorized.
