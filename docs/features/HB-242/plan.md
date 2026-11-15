---
type: ImplementationPlan
title: Plugin registry with signed versioned manifests, dual-controlled installation per workspace, isolated lifecycle, scoped hashed tokens, approved remote endpoints and vulnerability gates
description: Defines the first increment of work item W-35. A register records publishers and their Ed25519 keys under dual control. It accepts versioned plugin manifests whose package digest is signed by a published key. It installs approved versions per workspace under dual control, and lets operators enable, disable, stop and roll back without touching other workspaces. It issues scoped expiring plugin tokens that it stores only as hashes. It approves remote endpoints by server identity and deployment evidence. Versions marked vulnerable are refused in production. Removal keeps confirmed evidence readable.
status: draft
---

# Scope and ownership

[HB-242](https://linear.app/hybrid-billing/issue/HB-242) starts work item W-35 (requirement HB-35; HB-35-D/I/V; QA-06) after W-34. Base: main `83da313`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the plan and the change, because the register decides which external code may act for a workspace. Machine: mac_mini.

Allowed files: `modules/plugins/*`, `docs/features/HB-242/*`.

Out of scope, recorded as residuals:
- the process or container runtime that runs plugins, together with its timeouts, isolation and resource enforcement (the register records limits and lifecycle decisions, and a runtime adapter enforces them);
- the host of the separate origin for external apps;
- iframes, which QA-06 defers;
- calling commands with a plugin token (the token's command scopes are checked by a later adapter on `modules/commands`);
- storage migration;
- persistence;
- screens (P02, S05).

# Observed baseline

- HB-35-D: versioned extension contracts and examples are provided from the start. The core owns originals, calculation results, approvals and ledgers. External code must never write to the database directly or run in the admin DOM. Distribution trust, package signing, remote endpoint approval and version compatibility are decisions still open.
- HB-35-I: a PluginManifest, capabilities, restricted tokens, an egress allowlist, and digest and compatibility checks, run as a separate process or container. Screen extensions are only declarative configuration rendered by the core. A remote endpoint has its server identity and deployment evidence checked.
- HB-35-V: a module timeout, isolation, stop or rollback does not block other customers, and confirmed evidence stays readable after removal.
- QA-06:
  - External apps open on a separate origin and never receive core admin tokens.
  - Storage, ownership, migration and deletion rules are registered per module.
  - Vulnerable old versions run only in isolated verification environments.
  - Reading confirmed evidence never depends on running an old version.
- Sibling registers:
  - `modules/trust` holds approved destinations and weak-key checks.
  - `modules/webhooks` shows the dual-control and key patterns.

# Decision

A new package `modules/plugins` follows the sibling conventions:
- an injected clock that never goes backward, and one mutex;
- the standard errors, plus `ErrStale` and `Refusal{Reason}`;
- check order: form, lookup, repeat, rule, room;
- assistant actors refused as `ai_not_allowed` wherever a human is required, and system actors refused as `not_human`.

It imports only the standard library. It is constructed with:

```
NewRegistry(Config{Core, Destinations, Random}, Limits{...}, clock)
```

- `Core` is the running core version.
- `Destinations` answers whether a destination is active in a workspace of `modules/trust`.
- `Random` is the token entropy source.

## Proposals and dual control

Every change that grants or widens authority is a numbered proposal.

- **Propose.** `Propose(Change, actor)` by a human validates form, lookup and rule, then returns `ProposalView{Number, Change, Proposer}`. Proposing the identical change again while it is still pending returns the same proposal.
- **Approve.** `Approve(number, actor)` by a different human:
  - rechecks every rule against the current state;
  - applies the change;
  - is idempotent for the same approver;
  - refuses self-approval (`self_approval`), withdrawn or applied proposals (`not_pending`), and proposals whose rules no longer hold, with the specific reason.
- **Withdraw.** `Withdraw(number, actor)` by the proposer removes the proposal. A new proposal always gets a new number, so an approver acts only on the exact change they inspected.
- **Pending bound.** Pending proposals are capped per scope, where a scope is a workspace or the installation-wide register. Filling one workspace's cap never blocks another.

Change kinds:

| Kind | Payload | Applied effect |
| --- | --- | --- |
| `publisher` | `Publisher{ID, Name}` | Approved publisher |
| `key` | `PublisherKey{Publisher, Version, Public, NotBefore, NotAfter, Proof}` | Active publisher key |
| `version` | `Manifest`, `Signature` | Approved plugin version |
| `workspace` | `Workspace{ID, Verification, Runtime}` | Registered workspace |
| `runtime` | `Workspace{ID, Runtime}` | Replaces the runtime identity; every token of the workspace becomes invalid |
| `install` | `Workspace, Plugin, Version` | Installation in state `installed` |
| `upgrade` / `rollback` | `Workspace, Plugin, Version` | Version switch to a strictly higher / lower version |
| `enable` | `Workspace, Plugin` | State `enabled` |
| `remove` | `Workspace, Plugin` | State `removed`, with the storage decision |
| `endpoint` | `Workspace, Plugin, Endpoint{ID, Fingerprint, Evidence}` | Approved or rotated remote endpoint |

Tightening actions take effect at once, by a single human, without a proposal, because they never grant authority:
- `Disable`;
- `Stop`;
- `RetireKey` (normal rotation).

`Stop` may also come from the workspace's registered runtime.

Register-wide marks:
- **Kinds.** `MarkVulnerable(plugin, version, reason)`, `CompromiseKey(publisher, key, reason)` and `RevokePublisher(publisher, reason)` create a mark.
- **Effect.** A mark takes effect at once, stopping where the target is no longer runnable. It records the human and the reason.
- **Confirmation.** A second human confirms it with `ConfirmMark(target)`, which makes it permanent. The marker cannot confirm their own mark (`self_approval`).
- **Withdrawal.** Only the marker may withdraw an unconfirmed mark, with `ClearMark(target)`. Anyone else is refused (`not_proposer`). A confirmed mark can never be cleared (`not_pending`), so no single human can lift another human's mark. Withdrawal lifts the mark but never re-enables what it stopped: humans re-enable under dual control.
- **Repeats.** Marking an already marked target returns the existing mark.
- **Scope.** A mark is active while it is unconfirmed or confirmed, so one mistaken account can stop work but cannot make the stop permanent.

## Publishers and keys

- **Key form.** Publisher keys are Ed25519. Weak or small-order keys are refused (`weak_key`), and windows are inclusive and at most `KeyLife` (`key_life`).
- **Proof of possession.** A key proposal carries a proof: an Ed25519 signature over a length-prefixed challenge `hybrid-billing/plugin-key/v1 ‖ publisher ‖ version`. A key without valid proof is refused (`key_mismatch`).
- **Key names.** Names are never reused within a publisher. At most `Keys × 64` names are remembered (`ErrLimit`).
- **Retirement.** `RetireKey` blocks new versions signed by the key. Versions already approved stay runnable.
- **Compromise and revocation.** An active compromise mark revokes every version signed by the key, and an active revocation mark revokes every version of the publisher. Either one:
  - stops every enabled installation of those versions in every workspace, with reason `revoked`;
  - invalidates their tokens;
  - refuses pending proposals that depend on them when they are approved.

## Manifests and canonical encoding

A `Manifest` holds the following fields:

| Field | Rule |
| --- | --- |
| `Plugin` | Identifier |
| `Version` | `MAJOR.MINOR.PATCH`: digits only, no leading zeros, each component at most six digits, no pre-release or build suffix. Compared numerically |
| `Publisher`, `Key` | A publisher key |
| `Digest` | 64 lowercase hex characters (SHA-256 of the package) |
| `Core` | `CoreRange{Min, Max}` of versions, `Min ≤ Max` |
| `Capabilities` | 1–32 entries `{Kind, Target}`. Kind is `read_events`, `call_command` or `serve_screen`. Strictly ascending and unique |
| `Egress` | 0–16 destination identifiers, strictly ascending |
| `Endpoints` | 0–16 endpoint identifiers, strictly ascending |
| `Limits` | Timeout 1 s–5 min, memory 16–4096 MiB, concurrency 1–64 |
| `Storage` | `{Namespace, Owner, RetentionDays, Deletion, Migration}`. `Deletion` is `delete_on_removal` (retention 0) or `retain_for_evidence` (retention 1–3650). `Migration` is `none`, `additive` or `copy_forward` |
| `Screens` | 0–16 entries, strictly ascending by ID. Each is `{ID, Component, Label, Fields}`. `Component` is `table`, `form`, `detail` or `chart`. `Label` is plain UTF-8 text of 1–200 bytes with no control characters, which the core renders as escaped text. `Fields` are 1–32 entries `{Name, Label, Source}` with unique names, kept in the given order. `Source` is the plugin's own storage namespace or the target of one of its `read_events` capabilities. Each screen needs a `serve_screen` capability whose target is the screen ID. Screens hold no scripts, markup or URLs |

- **Ordering.** Out-of-order or duplicate lists are refused as `ErrInvalid`, never re-sorted. The stored manifest is therefore exactly the signed one.
- **Signed message.** The signature covers the tag `hybrid-billing/plugin/v1` and then every field in table order. Each field is a length-prefixed string. Numbers are decimal strings, and each list is prefixed by its count. Field order within a screen is part of the signed bytes. Known-answer tests pin the encoding.
- **Validity at proposal and at approval:**
  - the signature verifies against the named key;
  - the key is active and not compromised;
  - the publisher is approved and not revoked;
  - the key window contains the current time;
  - the core version lies in `Core` (`incompatible`).
- **Immutability.** A plugin version is immutable, and the same plugin and version with any other content is `ErrConflict`.
- **Vulnerability.** A confirmed vulnerability mark is permanent. A fix ships as a new version.
- **Plugin ownership.** A plugin ID belongs to the publisher of its first approved version. A `version` from any other publisher is refused (`publisher_mismatch`), so `upgrade` and `rollback` never change the publisher.

## Workspaces and isolation

- **Registration.** A workspace is registered under dual control.
- **Verification flag.** Its `Verification` flag is fixed at registration and can never change.
- **Runtime identity.** `Runtime` names the one system actor that runs plugins for that workspace. A runtime identity serves at most one workspace and is never reassigned. Rotated-out identities are tombstoned register-wide, with at most 16 identities per workspace over its lifetime (`ErrLimit`). If a runtime is compromised after the cap is reached, the operator path is `Disable` of every installation, by one human and immediate, which invalidates every token. Raising the cap is a residual.
- **Verification workspaces.** These are isolated verification environments. Their installations may not hold:
  - `call_command` token scopes (`isolated`);
  - endpoint approvals (`isolated`);
  - egress (`CheckEgress` refuses `isolated`).

One rule decides whether a version may run in a workspace. `runnable(workspace, version)` holds when:
- the version is approved;
- its publisher is not revoked;
- its key is not compromised;
- the core is compatible;
- the version is not vulnerable, or the workspace is a verification workspace.

The rule is applied by:
- `install`, `upgrade`, `rollback` and `enable` approval;
- `IssueToken`, `CheckToken`, `CheckEndpoint` and `CheckEgress`;
- the automatic stops of `MarkVulnerable` and of revocation. These stop installations only where the version is no longer runnable, so vulnerable versions keep running in verification workspaces.

## Installation lifecycle

Each (workspace, plugin) pair has its own installation record, with a revision that increases on every state or version change.

| From → to | Operation | Actor |
| --- | --- | --- |
| none / `removed` → `installed` | `install` | two humans |
| `installed` / `stopped` → `enabled` | `enable` | two humans; requires `runnable` |
| `enabled` / `stopped` → `installed` | `Disable` | one human |
| `enabled` → `stopped` | `Stop` | one human, or the workspace runtime with a reason such as `timeout` |
| `enabled` → `stopped` | automatic | creating a vulnerability, compromise or revocation mark; the stop holds while the mark is active, whether unconfirmed or confirmed |
| `installed` / `enabled` / `stopped` → same state, new version | `upgrade` (higher) / `rollback` (lower) | two humans; requires `runnable` |
| any but `removed` → `removed` | `remove` | two humans |

Other transitions are refused (`not_allowed_state`).

Repeating an operation is idempotent:
- `Disable` of `installed` returns the view;
- `Stop` of `stopped` returns the view.

At install, a storage namespace must not be held by another plugin in the workspace (`namespace_taken`). A namespace is held by:
- a live installation;
- a removed `retain_for_evidence` installation, until its retention days have passed since removal.

Only the same plugin may reinstall into its retained namespace. While the namespace is still held, the reinstall must keep:
- the same namespace;
- `retain_for_evidence`;
- retention that is not shorter (`storage_changed`).

A version switch:
- keeps only those endpoint approvals whose IDs the new manifest lists;
- requires the same storage namespace and deletion rule;
- requires retention that is not shorter (`storage_changed`), so evidence protection and namespace ownership never weaken. Changing storage needs removal and a new install.

An installation's revision never resets, including across removal and reinstallation.

`remove`:
- deletes the installation's tokens and endpoints;
- records the storage decision in `Installation().Disposal`: the namespace and its deletion rule, `delete_on_removal` or `retain_for_evidence`.

The register keeps every manifest and version record readable after removal, revocation or vulnerability marking. Confirmed evidence lives in core modules and is read without running the plugin.

## Tokens

`IssueToken(workspace, plugin, scopes, ttl, actor)` is called by the workspace's registered runtime only (`not_runtime`). It requires:
- an enabled installation whose version is runnable;
- scopes that exactly match manifest capabilities `{Kind, Target}`, with no prefix or wildcard meaning;
- no `call_command` scope in verification workspaces;
- `1 s ≤ ttl ≤ TokenLife`.

The token:
- is `hbp_` followed by 43 base64url characters of 32 random bytes;
- is returned once, and a retried call issues a new token;
- is stored only as SHA-256(`hybrid-billing/plugin-token/v1` ‖ token), together with (workspace, plugin, version, digest, revision, scopes, expiry).

If entropy fails, the call is refused (`dependency_unavailable`). Issuing first prunes the installation's expired and stale tokens, then requires fewer than `Tokens` live tokens.

`CheckToken(token, scope)`:
- refuses a malformed token as `ErrInvalid`. Any value without the `hbp_` form, such as an admin token, is malformed.
- returns `Grant{Workspace, Plugin, Version, Digest, Revision, Scopes}` only when all of these hold:
  - the hash is known;
  - the token has not expired;
  - the installation is enabled at the bound revision;
  - the version is runnable;
  - the scope is granted.
- otherwise refuses with a reason, and drops the expired or stale entry.

The returned digest lets the runtime keep work that is in flight during a swap on the digest it started with. Plugin tokens never carry user or admin authority.

## Remote endpoints and egress

- **Endpoint proposals.** The `endpoint` proposal ID must appear in the manifest (`not_declared`), and the workspace must not be a verification workspace.
- **Endpoint fields.**
  - `Fingerprint` is the SHA-256 hex of the server certificate.
  - `Evidence` is the SHA-256 hex of the deployment evidence. Checking the evidence content is a residual.
- **Fingerprint reuse.** Fingerprints are unique register-wide (`ErrConflict`). A rotated-out or removed fingerprint is kept as a tombstone that survives removal and is never reused. Live and tombstoned fingerprints together are capped by `Fingerprints` per workspace (`ErrLimit`), so one workspace's rotations never use up another's. A workspace that reaches the cap cannot approve new fingerprints. That bound is accepted, and expiring tombstones is a residual.
- **CheckEndpoint.** `CheckEndpoint(workspace, plugin, endpoint, fingerprint, actor)` is called by the workspace runtime. It requires an approved endpoint of an enabled, runnable installation whose fingerprint matches (`fingerprint_mismatch`).
- **CheckEgress.** `CheckEgress(workspace, plugin, destination, actor)` is called by the workspace runtime. It requires an enabled, runnable installation that lists the destination, in a workspace that is not a verification workspace. It captures the installation revision, then asks `Destinations` after leaving the lock. The destination must be reported active (`destination_inactive`), and a dependency failure refuses (`dependency_unavailable`). It then re-enters the lock and refuses with `ErrStale` unless the revision, the enabled state and `runnable` are unchanged.

## Bounds

Every `Limits` value has a hard maximum and is validated at construction:

| Limit | Scope | Maximum |
| --- | --- | --- |
| `Publishers` | register | 1,000 |
| `Keys` | per publisher | 16 |
| `KeyLife` | — | 1 h to 3650 d |
| `Plugins` | register | 10,000 |
| `Versions` | register | 100,000 |
| `Workspaces` | register | 100,000 |
| `Installations` | per workspace | 1,000 |
| `Tokens` | per installation; no register-wide token cap, so no workspace can block another's tokens | 1,000 |
| `Fingerprints` | per workspace, live plus tombstones | 10,000 |

The product of the maximums, for example about 10^11 token hashes in total, is an accepted theoretical bound. Operators size `Workspaces`, `Installations` and `Tokens` to their memory budget, and each token lives at most 24 hours.

Register-wide fingerprint ownership follows `modules/trust`. A workspace that first claims a public fingerprint keeps it, and that is accepted.
| `TokenLife` | — | 1 s to 24 h |
| `Endpoints` | per installation | 16 |
| `Proposals` | pending, per scope; one proposer holds at most a quarter (rounded up) | 1,000 |

Other bounds:
- Identifiers are 1–128 bytes.
- Labels and reasons are at most 200 bytes.
- List sizes are fixed as in the manifest table.

## Residuals

- The process or container runtime that enforces timeouts, memory and isolation. HB-35-V is therefore met only partially in this increment: the register guarantees that one workspace's stop, rollback or removal never changes another workspace.
- Checking deployment-evidence content.
- Command adapter. The later adapter that executes commands with a plugin token must use a distinct plugin actor kind that never satisfies a human or dual-control check.
- The core version upgrade path.
- Persistence.
- The separate-origin host.
- Screens.

# Verification plan

Literal tests:
- **Form and encoding.**
  - Every form rule, including semantic version edges, ordering and duplicate refusal, screen sources and labels, storage rules, and list bounds.
  - Known-answer tests for the challenge and manifest encodings, and a signature failure for every tampered field.
- **Proposals.** Proposal numbering, repeats, withdrawal, stale approval after withdrawal and re-proposal, self-approval, refusal of assistant and system actors, and the pending cap per scope.
- **Publishers and keys.** Key proof, weak keys, key life, name reuse, retirement versus compromise, and publisher revocation stopping installations in two workspaces while verification workspaces keep vulnerable versions.
- **Verification workspaces.** The vulnerable-version path in a verification workspace for every operation that applies `runnable`, and refusal in production.
- **Runtime binding.** Runtime A cannot issue, stop or check for workspace B. Rotating the runtime invalidates tokens.
- **Lifecycle.** Every transition table row and each refused transition, plus isolation between workspaces.
- **Tokens.**
  - Exact scopes, TTL bounds, the count bound with pruning, entropy failure, admin-form refusal, and storage of hashes only.
  - The token is invalid after stop, disable, upgrade, rollback, removal and runtime rotation.
- **Endpoints and egress.** Endpoint declaration, fingerprint uniqueness and tombstones, and egress through a fake `Destinations`.
- **Bounds.** Every limit boundary.

Additional tests from round 2:
- a foreign publisher's version is refused;
- storage-weakening switches are refused;
- marks: immediate effect, confirmation, clearing, self-confirmation refusal;
- a stop during the `CheckEgress` dependency call gives `ErrStale`;
- tombstones for fingerprints and runtimes;
- reserved retained namespaces: refused for another plugin, allowed for the same plugin, and released once retention has passed;
- per-workspace token and fingerprint caps that never block another workspace;
- marks withdrawn only by the marker, with confirmed marks never cleared, and a clear that restarts nothing;
- the per-proposer cap.

Property test: random operations across two production workspaces and one verification workspace, with interleaved vulnerability marks, key compromises and publisher revocations. It runs against an independent model of the transition table, `runnable`, and token validity.

Checks:
- `-race -count=10`, a 386 build, gofmt and vet;
- the whole repository with `-race`;
- 100% coverage and mutation testing.

# Review history

- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1. Compromise and revocation now stop running versions.
  - H2. Workspace runtimes are now bound.
  - H3. The verification flag is fixed and isolated.
  - H4. A single `runnable` rule now applies everywhere.
  - H5. Tokens are bound to the revision and digest, and upgrade is explicit.
  - M1. Approvals now name a proposal number.
  - M2. The plan now has a transition table, and every approval rechecks its rules.
  - M3. Screen sources are now limited to grants, and labels are bounded.
  - M4. Endpoints and egress are now cross-checked, with tombstones.
  - M5. Every limit is now bounded.
  - M6. The canonical, signed order is now specified.
  - M7. The plan now registers migration rules and namespaces.
  - M8 and L5. Recorded as residuals.
  - L1. Keys now need a proof.
  - L2. Tightening actions take effect at once, and vulnerability marks are permanent.
  - L3. Plugin tokens now have a tagged form.
  - L4. Entropy failure is now handled.
- Round 2 (same reviewer): FAIL. All round-1 findings resolved. New findings and what changed:
  - N-H1. A plugin ID is now bound to its first publisher.
  - N-M1. Switches can no longer weaken storage.
  - N-M2. Register-wide marks now take effect immediately but need a second human to become permanent or be cleared.
  - N-M3. `CheckEgress` now rechecks the revision after the dependency call.
  - L-a. Fingerprint tombstones are now register-wide.
  - L-b. Runtime identities are now tombstoned.
  - L-c. Register-wide token and manifest caps were added.
  - L-d. A per-proposer cap was added.
  - L-e. The revision is now monotonic.
- Round 3 (same reviewer): FAIL. All round-2 findings resolved. New findings and what changed:
  - M-A. Retained namespaces are now reserved for their plugin.
  - M-B. Register-wide token and fingerprint caps were replaced by per-workspace caps.
  - M-C. Only the marker may withdraw an unconfirmed mark.
  - L-f. Accepted as a bound, with a residual.
  - L-g. The operator path is now documented.
  - L-h. Automatic stops are now tied to active marks.
- Round 4 (same reviewer): PASS. Low items and what changed:
  - R4-L1. A reinstall into a held namespace keeps the storage rule.
  - R4-L2. The total bound is recorded as accepted.
  - R4-L3. Register-wide fingerprint ownership is accepted, following `modules/trust`.
  - The round-2 note on marks is superseded by the round-3 withdrawal rule.

# Change review resolution

The change review (independent reviewer, read-only) returned PASS with no high findings. Its items were resolved as follows:

- M1, token hash collision. `IssueToken` now refuses a hash that is already stored (`dependency_unavailable`), so a repeating entropy source can never overwrite another installation's token. Tested with a repeating reader.
- L1, defensive checks.
  - `CheckToken` now also requires the bound version and `runnable`.
  - `CheckEgress` re-entry now also requires the enabled state and `runnable`.
  - The property test asserts after every step that every enabled installation is runnable.
  - The defensive branches are tested by changing internal state.
- L2, withdrawn and purged proposals. Approving one returns `ErrNotFound`, because the register keeps no tombstone for them. The register keeps at most `Proposals` applied proposals per scope, oldest first, and a purged proposal also reads as `ErrNotFound`. This amends the Proposals section.
- L3, the removal result. The storage decision is read from `Installation().Disposal`, as amended above.
- L4, compromise marks. A compromise mark also retires the key, and withdrawing the mark does not un-retire it. The key's existing versions run again, but the key signs no new versions, and a new key is proposed instead. This is accepted as a tightening.
- L5, cost of automatic stops. Accepted for this increment and recorded as a residual: index installations by version before raising the limits toward their maximums.
- L6, removed records. These are bounded by `Plugins` per workspace, not by `Installations`. Accepted and recorded in Bounds.
- L7, dedupe encoding. Key times are now encoded as Unix seconds plus nanoseconds, so there is no `UnixNano` overflow.
- L8, test gaps.
  - Added: a known-answer test for the tagged token hash (checked against an independent Python computation), and a pinned test that a second human's pending `enable` still applies after a `Disable`, as the plan allows.
  - Runtime rotation, mark confirmation and endpoints stay covered by literal tests rather than by the property model.

The re-review of these fixes (same reviewer, read-only) returned PASS. Following its low observation, the test entropy fake now writes a 64-bit counter, so collisions appear only when a test asks for them.
