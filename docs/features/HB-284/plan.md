---
type: ImplementationPlan
title: P02 integrations API over modules/plugins, modules/webhooks and modules/events with a synthetic sample publisher, plugin, subscriptions and deliveries
description: Phase 5c of the pre-launch plan. Adds an integrations read for the four P02 tabs (API clients, modules, subscriptions and channels, events and redelivery) and administrator commands with a reason for plugin proposals, plugin stop and disable, webhook subscription approval, pause, resume and revoke, and dead-letter redelivery and dismissal. Nothing calls a network, dispatches a delivery, issues a plugin token or holds a signing key after the seed.
status: final
---

# Scope and ownership

[HB-284](https://linear.app/hybrid-billing/issue/HB-284) adds the API behind P02 (API, plugins and channels). Base: main `1907411`. The P02 screen follows in a separate ticket. The plan follows the merged [HB-282 plan](../HB-282/plan.md) and its [verification record](../HB-282/verification.md).

**Ownership**
- Root (Claude Code) owns the plan and integration. Implementation may be delegated to one writer in this worktree.
- An independent read-only reviewer reviews the design and the change, with a security focus (keys, signatures, tokens, approvals, actor kinds, no network effects).
- Machine: mac_mini.

**Allowed files**
- `modules/plugins/signing.go` (new): three exported wrappers, plus `signing_test.go`. No rule changes:
  - `Manifest.SigningMessage()` returns `encode()`;
  - `KeyChallenge(publisher, version)` returns `challenge(publisher, version)`;
  - `(*Registry).Recheck(number)` is a dry run of a pending proposal's checks (described under Design).
- `apps/api/server`:
  - new `integrations.go` (read), `integrationcmds.go` (commands and error mapping), `integrationseed.go` (seed and server store), `integrationdeps.go` (module dependency stand-ins), and their tests;
  - registration in `admin.go` (`registerUsers`), the route in `boundary.go`, the registries and limits in `server.go`, the seed step in `seed.go`;
  - the seed step count in `world_test.go` (13 to 14).
- `docs/features/HB-284/*`.
- No other module, `modules/trust`, `modules/connectors` or console file changes.

# Observed baseline

## `modules/plugins`

- **Registry.** `NewRegistry(Config{Core, Destinations, Random}, Limits, clock)` needs:
  - a semver `Core`, non-nil `Destinations` and `Random`, and limits inside the module bounds (`registry.go:110-129`, `registry.go:9-27`).
  - Bounds: Publishers ≤ 1000, Keys ≤ 16, KeyLife 1 hour to 3650 days, Plugins ≤ 10000, Versions ≤ 100000, Workspaces ≤ 100000, Installations ≤ 1000, Tokens ≤ 1000, TokenLife 1 second to 24 hours, Fingerprints ≤ 10000, Endpoints ≤ 16, Proposals ≤ 1000.
- **Changes are proposals.** `Propose(change, human)` checks the change, then `Approve(number, other human)` re-checks and applies it (`registry.go:179-249`).
  - Assistants are refused with `ai_not_allowed` and system actors with `not_human` (`registry.go:148-158`).
  - Self-approval is refused (`self_approval`, `registry.go:233-234`). A non-pending proposal is refused (`not_pending`, `registry.go:231-232`). A repeat approval by the same approver returns the applied view (`registry.go:229-230`).
  - The same proposer proposing the same change returns the existing proposal (`registry.go:194-198`). One proposer may hold at most `(Proposals+3)/4` pending proposals per scope (`registry.go:202-210`).
  - `Withdraw` is limited to the proposer (`not_proposer`) and deletes the proposal (`registry.go:251-271`). Applied proposals beyond `Proposals` per scope are pruned (`registry.go:244-247`).
  - `Proposal(number)` is the only proposal read; there is no list (`registry.go:273-281`).
- **Change kinds** (`types.go:23-33`, `changes.go:9-42`):
  - Registry scope: `publisher`; `key` (an ed25519 key with a proof over `challenge(publisher, version)`, `changes.go:155`); `version` (a manifest signed over `encode()` by an active publisher key, `changes.go:185`, compatible with `Core`, `changes.go:102-104,187`); `workspace` (ID, runtime system identity, verification flag).
  - Workspace scope: `runtime`; `install`, `upgrade` and `rollback` with a version; `enable`; `remove`; `endpoint` (an ID declared by the installed manifest, with a 64-hex certificate fingerprint and evidence, `changes.go:37-39,317-335`).
  - A workspace-scoped proposal needs the workspace to exist (`registry.go:189-191`).
- **Lifecycle rules.**
  - Install is refused while an installation is live, with a revoked publisher or compromised key, a vulnerable version outside a verification workspace, or a taken namespace or changed storage contract (`changes.go:250-278`).
  - Upgrade and rollback need a live installation, the right direction (`direction`) and the same storage contract (`storage_changed`, `changes.go:280-301`).
  - Enable needs `installed` or `stopped` (`changes.go:303-315`).
- **Manifest form** (`manifest.go:146-221`):
  - 64-hex digest, semver version and core range;
  - capabilities (`call_command`, `read_events`, `serve_screen`) sorted by kind, then by target;
  - ascending egress and endpoint IDs;
  - runtime limits of 1 second to 5 minutes, 16 to 4096 MiB and concurrency 1 to 64;
  - a storage contract (namespace, owner, `delete_on_removal` with 0 days or `retain_for_evidence` with 1 to 3650 days, a migration kind);
  - screens whose fields read the plugin's namespace or a declared `read_events` target.
  - IDs use `[A-Za-z0-9._:-]`, at most 128 bytes (`manifest.go:48-59`).
  - Weak and small-order keys are refused (`weak_key`, `manifest.go:313-328`).
- **Direct operations.**
  - `Disable(workspace, plugin, human)` moves `enabled` or `stopped` to `installed`. On an `installed` plugin it returns the view unchanged without an error (`operations.go:30-48`).
  - `Stop(workspace, plugin, reason, actor)` accepts a human, or the workspace runtime system actor. It moves `enabled` to `stopped` and needs an ID-form reason (`operations.go:50-74`). On a `stopped` plugin it returns the view unchanged without an error (`operations.go:66-73`). `ReasonOperator = "operator"` exists (`types.go:96`).
  - Marks (vulnerable, compromise, revocation) with confirm and clear, and `RetireKey`, are human operations (`operations.go:168-307`).
- **Reads.** `Installation`, `Workspace`, `Publisher` and `Version` read one named item (`operations.go:76-155`). There is no list of publishers, plugins or installations.
  - `Publisher` returns key public bytes (`types.go:255-262`).
  - `Version` returns the manifest signature (`types.go:264-268`).
  - Proposal views carry the key proof and the version signature (`types.go:142-149,215-233`).
- **Runtime paths.** `IssueToken`, `CheckToken`, `CheckEndpoint` and `CheckEgress` need the workspace runtime system actor (`tokens.go:43-66,77-203`). `CheckEgress` calls `Destinations.Active` (`tokens.go:190`).
- **Errors:** `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrLimit`, `ErrStale` (only from `CheckEgress`, `tokens.go:199-201`), and `Refusal{Reason}` wrapping `ErrRejected` (`types.go:9-16,99-105`).
- **Fixtures** (`world_test.go:142-170`):
  - keys from `ed25519.NewKeyFromSeed(sha256("synthetic-plugin-seed-"+name))`;
  - the proof is `ed25519.Sign(private, challenge(publisher, version))`;
  - the version signature is `ed25519.Sign(private, m.encode())`.
  - `encode` and `challenge` are unexported, so the server cannot build a valid key or version without wrappers.

## `modules/webhooks`

- **Registry.** `NewRegistry(Config{Outbox, Destinations, Permits, Signer, Redirects}, Limits, clock)` needs every dependency (`subscriptions.go:44-49`). Bounds (`subscriptions.go:51-61`, `types.go:19-39`):
  - Subscriptions ≤ 10000, Types ≤ 32, Attempts ≤ 100, Dead ≤ 10000, Batch and Work ≤ 10000, Keys ≤ 16;
  - Lease 1 second to 1 hour; Backoff ≤ MaxBackoff ≤ 24 hours; Backoff ≤ MaxAge ≤ 30 days; KeyLife 1 hour to 3650 days.
- **Subscriptions.**
  - `ProposeSubscription(workspace, Subscription{ID, Subscriber, Destination, Types}, human)`: the caller supplies the ID, and the same proposer and content returns the existing proposal; otherwise a duplicate ID is `ErrConflict` (`subscriptions.go:155-193`).
  - `ApproveSubscription` refuses self-approval and a non-proposed state. It then calls `Destinations.State` and `Permits.Subscribe` outside the module lock (`admit`) and `Outbox.Bounds`, and starts the cursor at the outbox head (`subscriptions.go:209-262`). A repeat by the same approver returns the view whatever the current state (`subscriptions.go:235-237`).
  - `PauseSubscription` needs `active` (`not_active`). `RequestResume` records the requester and overwrites an earlier one (`subscriptions.go:279-291`). `ApproveResume` refuses the requester (`self_approval`), re-admits, and records a gap when the reason is `gap` (`subscriptions.go:293-332`).
  - `RevokeSubscription` works from any state, cancels the open delivery, and is a silent no-op on a revoked subscription (`subscriptions.go:334-360`).
  - `SubscriptionOf` is the only subscription read; there is no list (`subscriptions.go:362-373`).
  - Human checks: `find` refuses an assistant or system actor (`subscriptions.go:93-101,195-207`).
- **Keys.**
  - `ProposeKey` is by a human (`keys.go:75-115`).
  - `ApproveKey` refuses self-approval, then asks the configured `Signer.Sign` to sign the module's own challenge, and verifies it against the proposed public key (`keys.go:117-168`). Approval therefore needs a live signer that holds the private key.
  - `Keys` lists only active keys, with public bytes (`keys.go:191-209`), through `listed`, which refuses only assistants (`deliveries.go:426-438`).
- **Deliveries.** `Dispatch`, `Confirm`, `Redirect` and `Report` need a system actor (`deliveries.go:126-132,226-238`).
  - `Dispatch` reads the outbox. `Confirm` calls `Destinations.State`, `Permits.Allowed`, `Signer.Current` and `Signer.Sign`, and returns the body, signature and headers (`deliveries.go:240-334`).
  - Lease tokens are `lease-N` (`deliveries.go:120-124`).
  - `Report` with `delivered` closes the delivery and drops it from every list (`deliveries.go:86-89,376-379`); `rejected` buries it as a dead letter (`deliveries.go:62-74,380-381`); `failed` retries with backoff (`deliveries.go:391-396`).
  - Job states are `queued`, `running`, `waiting`, `succeeded`, `failed` and `cancelled` (`deliveries.go:403-419`). `Deliveries(workspace, id, after, limit, actor)` pages the open, queued, dead and cancelled deliveries (`deliveries.go:440-460`).
  - Delivery IDs are `delivery-N` from one registry-wide counter (`subscriptions.go:110-113`).
- **Dead letters.**
  - `RequestRedeliver` and `RequestDismiss` record the requester, overwriting an earlier one, and return only an error (`deliveries.go:478-487,515-524`; the writes are at `deliveries.go:485,522`). `letter` returns `ErrNotFound` for a delivery that is not a dead letter (`deliveries.go:462-476`).
  - `ApproveRedeliver` and `ApproveDismiss` each delete both the redeliver and the dismiss request for the letter (`deliveries.go:508-509,541-542`).
  - `ApproveRedeliver` refuses the requester and a revoked subscription, then queues a fresh redelivery (`deliveries.go:489-513`).
  - `ApproveDismiss` refuses the requester and removes the letter (`deliveries.go:526-547`).
  - There is no read of pending requests.
- **Errors:** `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrLimit`, `ErrStale` (state changed while a dependency was consulted, `subscriptions.go:256-258,319-321`, `keys.go:153-155`), and `Refusal{Reason}`, including `dependency_unavailable` for any dependency error (`types.go:76-101`).
- **Fixtures** (`world_test.go:14-262`):
  - fakes for every dependency and an `events.Registry` outbox read with a system relay actor;
  - keys from `NewKeyFromSeed(sha256("synthetic-webhook-seed-"+version))`;
  - subscriptions by `ann` and approval by `bob`, then key publication;
  - dispatch, confirm and report with the system `relay`.

## `modules/events`

- `ProposeType` and `ApproveType` are a human two-person flow (`registry.go:207-260`).
- `Publish`, `Read` and `Bounds` need a system actor (`registry.go:320-416`). `Read` returns `Expired` before the floor (`registry.go:391-392`).
- Limits are in `types.go:92-103`. There is no type list.

## Server and commands

- **Commands.** Commands run under the server's domain write lock (`commands.go:196-206`).
  - Only `invalid_request` carries fields, and only declared field names (`modules/commands/execute.go:399-413`).
  - Handler codes are `unsupported_capability`, `invalid_stage`, `revision_conflict`, `prerequisite_failed`, `dependency_unavailable`, `permission_denied` and `invalid_request` (`modules/commands/types.go:71-74`). `self_approval_denied` is not a handler code, so HB-282 maps self-approval to `prerequisite_failed` (`installation.go:458-474`).
  - Enum fields are validated against their values (`execute.go:42-48`).
- **Clients and service users.**
  - The only gate client is `console`, registered by `server.New` (`server.go:190-192`). The gate has no client list read (`modules/gate/gate.go:218-290`), but `Grants()` returns client grants (`access.go:162-166`).
  - Service users and their credentials are read through `identity` (`services.go:75-124`) and managed on M04 (HB-276).
  - The access read hides `AdministerInstallation` grants (`access.go:160-163`).
- **Resolve.** `Resolve(workspace, target)` receives no actor; only `Apply` receives the user (`installation.go:481-493,524-529`).
- **Admin reads.** Admin reads use `admitted`, which requires `AdministerInstallation` for the console client (`users.go:42-67`).
- **Sample seed.** The sample seed runs one counted step per domain (`seed.go:94-112`), and the test asserts 13 steps (`world_test.go:57-59`).
- **HB-282 actor test.** The AST test allows system actors only in the seed function (`installation_edges_test.go:193-215`).

# Design

## Registries and limits

- **Plugins.** `plugins.NewRegistry(Config{Core: pluginCore, Destinations: integrationDestinations, Random: crypto/rand.Reader}, PluginLimits(), clock)`.
  - Limits: Publishers 8, Keys 4, KeyLife 400 days, Plugins 16, Versions 64, Workspaces 4, Installations 16, Tokens 1, Fingerprints 16, TokenLife 1 hour, Endpoints 4, Proposals 64.
  - Tokens must be at least 1, but no token is ever issued.
  - `pluginCore = "1.0.0"` is a named constant: the pre-launch core API version that manifests declare compatibility with.
- **Webhooks.** `webhooks.NewRegistry(Config{Outbox: outboxReader, Destinations: integrationDestinations, Permits: integrationPermits, Signer: seedSigner, Redirects: noRedirects}, WebhookLimits(), clock)`.
  - Limits: Workspaces 2, Subscriptions 16, Types 8, Lease 1 minute, Backoff 1 minute, MaxBackoff 1 hour, Attempts 5, MaxAge 24 hours, Dead 100, Batch 10, Work 4, Keys 4, KeyLife 400 days.
- **Events.** `events.NewRegistry(EventLimits(), clock)`.
  - Limits: Workspaces 2, Types 16, Events 1000, Retention 30 days, Page 100, Sources 1, Seen 1, InboxRetention 1 hour, Revisions 1, Reservation 1 minute.
- All values are inside the module bounds. The three registries are built in `server.New`, next to the installer.

## Module dependency stand-ins (`integrationdeps.go`)

These are server values, not modules. They hold no secrets except the seed signer while it is armed. They never take the server's domain lock, because the webhooks module calls them while `execute` already holds it.

- **`integrationDestinations`** answers both interfaces from a fixed sample allowlist:
  - `sample-erp-webhook` `active`, `sample-payments-webhook` `active` and `sample-legacy-webhook` `inactive`;
  - for plugins, `sample-erp` `active`.
  - Unknown IDs and other workspaces are `inactive` or `false`. No address is resolved and no network is used. `modules/trust` destinations are a residual.
- **`integrationPermits`.** `Subscribe` and `Allowed` are true only for the boot workspace, the sample subscriber `sample-erp-connector`, and event types approved in the server's event type record.
- **`outboxReader`** adapts `events.Read` and `events.Bounds`. It holds one fixed system actor (`integration-outbox`), built once at construction. That actor could publish events if passed to `events.Publish`, so it is used only inside the adapter's `Read` and `Bounds` methods. The AST test enforces this. The adapter is a read-only dependency of the webhooks module, not the requester.
- **`seedSigner`** maps a subscription to its current key version and, only while armed, to an ed25519 private key.
  - While armed, it signs only for the triple (boot workspace, `sample-erp-invoices`, `sample-webhook-key-1`) and refuses any other triple with an error.
  - The seed arms it before the webhook key approval and disarms it right after the last `Confirm`, with a `defer` that also disarms it when the seed fails. Disarming zeroes the held private key slice with `clear` and drops the map. Zeroing is best effort: it clears only the slice the signer holds, not copies the runtime or `crypto/ed25519` may have made.
  - Once disarmed, `Sign` returns an error, so the module answers `dependency_unavailable` (`keys.go:144-147`, `deliveries.go:329-332`). `Current` returns only the version name.
  - This deviates from HB-282, where the key stayed a local variable: the webhooks module calls the signer itself (`keys.go:144`, `deliveries.go:329`), so the key must be reachable through the signer while those seed calls run. It is never reachable from a request.
- **`noRedirects.Check`** always returns an error. It is unreachable, because `Redirect` needs a system actor.

## Sample seed (only with `--seed sample`; one counted step `integrations`)

**Actors and seed-only identities**
- The first sample administrator proposes and the second approves, unless noted.
- System actors built only inside `seedIntegrations`: `sample-billing` publishes events, and `sample-relay` dispatches, confirms and reports.
- The plugin workspace runtime `sample-plugin-runtime` is an identifier only and is never used as an actor.

**Event types:** `invoice.issued` v1 (field `invoice_id`) and `payment.received` v1 (field `payment_id`), each proposed and approved. The server records each type (name, version, fields, proposer, approver).

**Plugins**
- Publisher `sample-publisher` ("Sample Publisher").
- Key `sample-plugin-key-1`: an ed25519 pair generated from `crypto/rand`, with the private key a local variable of the seed function, cleared with `clear` (best effort) once both versions are signed. Proof over `plugins.KeyChallenge`; valid from the seed time for 180 days.
- Plugin `sample-ledger-export`, versions `1.0.0` and `1.1.0`, signed over `Manifest.SigningMessage()`:
  - digest: SHA-256 hex of `sample-ledger-export-<version>`;
  - core range `1.0.0` to `1.9.9`;
  - capabilities `read_events:invoice.issued` and `serve_screen:overview` (no `call_command`);
  - egress `sample-erp`, endpoint `api`;
  - limits 30 seconds, 128 MiB, concurrency 2;
  - storage: namespace `sample-ledger-export-data`, owner `sample-publisher`, `retain_for_evidence` for 30 days, `additive`;
  - screen `overview` (table, label "Overview") with fields `total` (label "Total", source the namespace) and `event` (label "Event", source `invoice.issued`).
  - Both versions keep the storage contract, so an upgrade passes `changes.go:296-299`.
- Workspace: the boot workspace, runtime `sample-plugin-runtime`, not a verification workspace.
- Install `1.0.0` and enable it.
- Two pending proposals:
  - (a) upgrade to `1.1.0`, proposed by the second administrator (the first can approve);
  - (b) endpoint `api` with fingerprint SHA-256 hex of `sample-certificate-api` and evidence SHA-256 hex of `sample-evidence-api`, proposed by the first administrator (the second can approve).

**Webhooks**
- Subscription `sample-erp-invoices` (subscriber `sample-erp-connector`, destination `sample-erp-webhook`, types `invoice.issued`), approved.
- Key `sample-webhook-key-1`: an ed25519 pair from `crypto/rand`, handed to the armed signer only. Valid from the seed time for 180 days. Proposed and approved; approval makes the module ask the signer for its challenge.
- Publish three `invoice.issued` events (`sample-invoice-1` to `-3`, source `urn:sample-billing`), after approval, because approval starts the cursor at the head. Then three rounds of dispatch, confirm and report by `sample-relay`:
  - `delivered` (closed; it leaves the lists);
  - `rejected` (a dead letter);
  - `failed` (waiting, 1 attempt).
- The signer is disarmed right after the third `Confirm`; `Report` does not sign.
- The seed discards every lease token, body, signature and header. Nothing is sent: the outcomes are synthetic and labelled as sample data.
- The first administrator requests redelivery of the dead letter (the second can approve).
- Subscription `sample-payments-feed` (destination `sample-payments-webhook`, types `payment.received`), proposed by the second administrator and left pending.

## Server store (`integrationseed.go`)

The modules have no list reads, so the server keeps, separate from the HB-282 store:

- **Plugin side:**
  - publisher IDs;
  - plugin IDs with their registered version strings in approval order;
  - the plugin workspace ID;
  - proposal entries (number, kind, created time, proposer, rationale, closed by, closing rationale, and the last module view). These are kept like `installStore.keep`: at most 200, pending ones never dropped, newest 50 shown.
- **Webhooks side:**
  - subscription entries (ID, created time, proposer, rationale, per-command actor and rationale history);
  - the dead-letter delivery IDs per stored subscription;
  - the redeliver and dismiss requester and rationale per delivery ID, mirrored from the commands because the module has no read of them.
- **Mirror rules.** The server updates a mirror only after the module call succeeds:
  - the seed's `Report` with `rejected` adds the delivery to its subscription's dead letters;
  - `ApproveRedeliver` and `ApproveDismiss` remove the letter and clear both its redeliver and dismiss requests, as the module does;
  - `RevokeSubscription` keeps the letters (the module keeps them), and the resume requester is never mirrored: the read and the prechecks take it from `SubscriptionOf().Resumer`, which revocation and resume approval clear (`subscriptions.go:330,350`).
- **Events side:** the event type records.
- A counter for server-generated subscription IDs (`subscription-<n>`).

Commands accept only targets found in this store.

## Read (admin only, boot workspace only)

`GET /api/v1/integrations` is admitted like the installation read. For another workspace it returns `{revision, now, state: not_configured}`. Otherwise it returns `revision` and `now`, and one section per tab, each with its own `state` (`configured`, or `not_configured` when nothing is registered, for example with `--seed none`).

**`clients`**
- `clients`: each gate client the server registered (only `console`, kind `first_party_console`, `active`), with its grants from `Grants()` filtered by client ID: ID, action, kind, scope, not before, not after, state. As in the access read, `AdministerInstallation` grants are hidden, so `installation-console` is not listed.
- `service_users`: ID, state, owner (ID and login), and the count of usable credentials, from the same identity reads as M04. Credential IDs are not repeated here. `manage_on` is `M04`.
- `oauth_clients`: `not_connected`. There is no OAuth client registry yet.

**`modules`**
- `workspace`: ID, runtime identifier, verification flag.
- `publishers`: ID, name, revocation mark, and keys (version, not before, not after, retired, compromise mark). No public key or proof.
- `plugins`: for each version:
  - version, publisher, key version, digest, core range, capabilities, egress, endpoint IDs, runtime limits, storage contract, screens (ID, component, label and fields as plain text), and the vulnerable mark (by and confirmed by, with logins and reason). No signature.
  - The digest is shown on purpose: P02 lists "version/digest" as a column, and a package digest identifies a package; it is not a credential. This differs from HB-282, which hid artifact digests, and is a design-review point.
- `installations`: plugin, version, state, reason, revision, endpoints (ID, fingerprint, approver and login; no evidence), and the disposal contract when removed.
  - `can_stop` (blocker `not_enabled`) and `can_disable` (blocker `not_allowed_state`), derived from the state.
  - `tokens`: `not_issued`; `runtime`: `not_connected`.
- `proposals`:
  - number, target, kind, state (`pending`, `applied`, `withdrawn`), created time;
  - proposer and approver (ID and login), rationale, closed by and closing rationale;
  - a summary: publisher ID and name; key version and validity; plugin and version; workspace and runtime; kind, plugin and version; endpoint ID and fingerprint;
  - `can_approve` and `can_withdraw` with blockers:
    - `not_pending`, derived from the stored view. The read never calls `Proposal` or `Recheck` for a non-pending entry, because the module deletes withdrawn proposals and prunes applied ones.
    - `own_proposal` and `not_proposer`.
    - For other pending proposals, the result of `Recheck`: a refusal reason (for example `direction`, `storage_changed`, `not_allowed_state`, `namespace_taken`, `not_declared`, `key_inactive`, `signature_invalid`, `revoked`, `vulnerable`), or `not_found`, `conflict` or `limit` for those errors.

**`channels`**
- `destinations`: ID, state, and `source: sample_allowlist`.
- `subscribers`: the sample subscriber.
- `subscriptions`:
  - ID, subscriber, destination, types, state, reason, cursor, lag (`head - cursor`), gaps, proposer, approver and resumer (ID and login);
  - active keys from `Keys`, called with `{ID: reader, Kind: human}`: version, not before, not after, state, proposer, approver. No public key;
  - `signer`: `not_connected`;
  - per-action availability with blockers:
    - approve: `own_proposal`, `not_pending`, `destination_inactive`, `subscribe_refused`, computed from the stand-ins;
    - pause: `not_active`;
    - request resume: `not_pending` (not paused or suspended), `already_requested` (`SubscriptionOf().Resumer` is set, by anyone);
    - approve resume: `not_pending` (no requester), `own_request`, `destination_inactive`, `subscribe_refused`;
    - revoke: `not_pending` when already revoked.

**`events`**
- `types`: name, schema version, fields, state, proposer and approver.
- `outbox`: floor and head from the adapter.
- `deliveries`: per subscription, up to 50 rows from `Deliveries`, called with `{ID: reader, Kind: human}`:
  - ID, sequence, source, event ID, kind, job state, reason, attempts, outcome unknown;
  - for dead letters, the pending redeliver and dismiss requests (requester and login, rationale) and `can_request_*` / `can_approve_*` with blockers:
    - request: `already_requested` when a request of that kind is pending, by anyone;
    - approve: `not_requested`, `own_request`, and `revoked` for redelivery on a revoked subscription;
    - `not_dead` for rows that are not dead letters.
- `dispatcher`: `not_connected`. Nothing dispatches after the seed, so `queued` and `waiting` rows stay as they are.

No row contains a public key, proof, signature, private key, lease token, plugin token, webhook body or signature header.

## Exported plugin wrappers (`modules/plugins/signing.go`)

- `func (m Manifest) SigningMessage() []byte { return m.encode() }`
- `func KeyChallenge(publisher, version string) []byte { return challenge(publisher, version) }`
- `func (r *Registry) Recheck(number uint64) error` behaves as follows:
  - under the module lock, it returns `ErrNotFound` for an unknown number and `refuse(not_pending)` for a non-pending proposal;
  - otherwise it returns `r.check(p.view.Change, now)`, the same check `Approve` runs before applying (`registry.go:236-238`), without changing state.
- This is a reviewed module change with tests showing that:
  - a signature over the wrapper output passes `Propose` and `Approve`;
  - `Recheck` returns what `Approve` would return, and changes nothing (the same `Proposal` view and the same installation revision before and after).

## Commands (`integrations.*`, reason required, actor = the reader as a human)

| Command | Target | Fields | Module call |
| --- | --- | --- | --- |
| `integrations.plugin_approve` | `plugin-proposal-<n>` | none | `plugins.Approve(n, reader)` |
| `integrations.plugin_withdraw` | `plugin-proposal-<n>` | none | `plugins.Withdraw(n, reader)` |
| `integrations.plugin_propose` | `plugin:<plugin>` | `kind` (required enum `install`, `upgrade`, `rollback`, `enable`, `remove`); `version` (text, `Required: false`) | `plugins.Propose({kind, workspace, plugin, version}, reader)` |
| `integrations.plugin_stop` | `plugin:<plugin>` | none | `plugins.Stop(workspace, plugin, "operator", reader)` |
| `integrations.plugin_disable` | `plugin:<plugin>` | none | `plugins.Disable(workspace, plugin, reader)` |
| `integrations.subscription_propose` | `webhooks` | `destination`, `types` (comma-separated) | `webhooks.ProposeSubscription(workspace, {subscription-<n>, sample-erp-connector, destination, types}, reader)` |
| `integrations.subscription_approve` | `webhook-subscription:<id>` | none | `ApproveSubscription` |
| `integrations.subscription_pause` | `webhook-subscription:<id>` | none | `PauseSubscription` |
| `integrations.subscription_request_resume` | `webhook-subscription:<id>` | none | `RequestResume` |
| `integrations.subscription_approve_resume` | `webhook-subscription:<id>` | none | `ApproveResume` |
| `integrations.subscription_revoke` | `webhook-subscription:<id>` | none | `RevokeSubscription` |
| `integrations.letter_request_redeliver` | `webhook-letter:<delivery-id>` | none | `RequestRedeliver` |
| `integrations.letter_approve_redeliver` | `webhook-letter:<delivery-id>` | none | `ApproveRedeliver` |
| `integrations.letter_request_dismiss` | `webhook-letter:<delivery-id>` | none | `RequestDismiss` |
| `integrations.letter_approve_dismiss` | `webhook-letter:<delivery-id>` | none | `ApproveDismiss` |

**Permission (a deviation from P02).** Every command requires `AdministerInstallation` on the installation kind, the same permission as the other admin commands.
- This deviates from P02, which keeps API, module and channel authority separate (`docs/ui/screens/P02/spec.md:33`) and names separate extension, verification, approval, stop and reprocess permissions (`spec.md:70-76`). The gate has no such actions yet.
- The definitions are built in three groups (`plugin_*`, `subscription_*`, `letter_*`), each from its own permission value. Splitting the permission later is a one-line change per group. The split itself is a residual.

**Resolve.** `Resolve` receives no actor, so it reads only the server store and calls no module. It returns not found unless the workspace is the boot workspace and the target is in the store:
- the proposal number, in canonical decimal;
- the plugin ID;
- `webhooks`, once at least one event type is approved;
- the subscription ID;
- a delivery ID in the store's dead letters of a stored subscription.

`Apply` then calls the module with the human reader. A letter that is no longer a dead letter makes `letter` return `ErrNotFound`, which maps to `invalid_stage`. So does a revoked subscription that the module has purged (`subscriptions.go:384-390`).

**Actors.** Every module actor is `{ID: ctx.User, Kind: human}`. No command builds a system or assistant actor, and a test enforces it. There is no command that signs, dispatches, confirms, reports, redirects, issues or checks tokens, or checks egress.

**Server prechecks** (fields only on `invalid_request`). A precheck that reads module state uses the human reader or an actor-free read (`SubscriptionOf`, `Installation`). It never calls `Proposal` or `Recheck` for a store entry that is not pending.
- `plugin_propose`:
  - `version` must be a registered version of the plugin for `install`, `upgrade` and `rollback`, and empty for `enable` and `remove` (`invalid_request` on `version`);
  - the plugin workspace must exist (`invalid_stage`).
- `subscription_propose`:
  - `destination` must be in the sample allowlist (`invalid_request` on `destination`);
  - `types` must be 1 to 8 distinct, approved event type names (`invalid_request` on `types`);
  - a stored subscription that is `proposed` or `active` with the same subscriber, destination and set of types, from another proposer, is refused with `invalid_stage`. The same proposer gets the existing proposal back (see Duplicates).
- **Pending requests.** The module overwrites a pending requester, which would let a second administrator take over a request. So a request while one of the same kind is pending, by anyone, is refused with `invalid_stage` (blocker `already_requested`):
  - `subscription_request_resume` when `SubscriptionOf().Resumer` is set;
  - `letter_request_redeliver` and `letter_request_dismiss` when the store holds a pending request of that kind for the letter.
- **Silent no-ops.** The modules succeed silently on five calls. The server refuses them first with `invalid_stage`, so the revision does not move:
  - `plugin_approve` on a non-pending proposal (`registry.go:229-230`), decided from the stored view;
  - `plugin_disable` on an `installed` plugin (`operations.go:40-47`);
  - `plugin_stop` unless the plugin is `enabled`; on `stopped` the module is silent (`operations.go:66-73`);
  - `subscription_approve` on a non-proposed subscription (`subscriptions.go:235-237`);
  - `subscription_revoke` on a revoked one (`subscriptions.go:346-348`).
- **Rationale.** The rationale is capped at `maxRationale` (256 bytes) and kept in the server store with the actor.

**Error mapping**

| Module result | Code |
| --- | --- |
| `Refusal` `self_approval` or `not_proposer` | `prerequisite_failed` |
| `Refusal` `destination_inactive` or `subscribe_refused` | `prerequisite_failed` |
| `Refusal` `dependency_unavailable` | `dependency_unavailable` (retry later) |
| `Refusal` `ai_not_allowed` or `not_human` | `internal_error` (the server built a non-human actor; unreachable) |
| Any other `Refusal` | `invalid_stage` |
| `ErrNotFound` after the prechecks | `invalid_stage` |
| `ErrConflict` (for example, a fingerprint already in use) | `invalid_stage` |
| `ErrStale` (state changed during a dependency call) | `invalid_stage` |
| `ErrLimit` | `prerequisite_failed` |
| `ErrInvalid` | `internal_error` (the server built a bad input) |

These codes carry no fields, so the read's blockers explain them.

**Duplicates.** A duplicate `plugin_propose` or `subscription_propose` by the same administrator returns the existing proposal (module behaviour). The server reuses the stored entry and does not generate a new subscription ID for the same content from the same proposer.

**Results.** Command results return `fields` with `proposal`, `subscription` or `delivery` and the `target`, and `business_state` is the module state.

# Out of scope (residuals)

- **Network and runtime.**
  - Dispatch, confirm, redirect and report after the seed; nothing sends a webhook, and approved redeliveries stay `queued`.
  - Plugin tokens (`IssueToken`, `CheckToken`), endpoint and egress checks, and runtime stop by the runtime system actor. These need a non-human runtime identity and a real runtime.
- **Signing.**
  - Webhook key proposal, approval and retirement: approval needs a connected signer, and the seed signer is disarmed. A real signer (KMS or HSM) is a separate design.
  - No endpoint may ever sign caller-supplied data.
- **Plugin registration and containment.**
  - Registering publishers, keys and versions from the console; this needs a package and signature intake.
  - Proposing endpoints from the console; this needs certificate evidence intake.
  - Marks (vulnerable, compromise, revocation, confirm, clear), publisher key retirement, runtime changes and verification workspaces.
  - Without marks, a compromised publisher or key is contained only by a single administrator stopping or disabling each installation, and nothing blocks a later re-enable proposal from being approved.
- **Permissions.** Separate gate actions for API, module and channel administration, stop and reprocess, as P02 requires. Until then every command uses `AdministerInstallation`.
- **Destinations, permits and clients.**
  - `modules/trust` destinations with address and redirect checks; the sample allowlist is a stand-in.
  - Real subscriber permissions; the permits are a stand-in.
  - OAuth or API client registration and token issuance.
  - Service user commands stay on M04.
- **Other.**
  - Event publication and inbox reception (`events.Receive`, source bindings) from the console.
  - Delivery history of closed deliveries; the module drops them.
  - Persistence. External application origins (QA-06) are a later design.
- **Cursor resync.** P02's expired cursor resync is shown only through the subscription `gap` reason and recorded gaps. No command forces a resync.

# Verification plan

- **Module:** `signing_test.go` for the three wrappers:
  - a signature over the wrapper output passes `Propose` and `Approve`;
  - `Recheck` matches `Approve` for each refusal used by the read, and changes no state;
  - 100% statement coverage of `modules/plugins` remains.
- **API read:**
  - the read shape with the sample seed and with `--seed none` (clients present, the other sections `not_configured`);
  - `clients.grants` hides `AdministerInstallation` grants;
  - an administrator of another workspace gets `not_configured`, and a non-admin member gets `permission_denied`;
  - every blocker above, including `already_requested` and the `not_pending` rows built without a module call.
- **API commands:**
  - each command's success, self-approval, not proposer or requester, not pending, approve after withdraw;
  - the five silent no-ops refused as `invalid_stage`, including `plugin_stop` on a stopped plugin, with the revision unchanged;
  - pending requests: a repeat request by the same administrator and a request by the second administrator, for resume, redelivery and dismissal, are each refused with `invalid_stage` and leave the requester unchanged;
  - mirror clearing after redelivery and dismissal approval, and no mirror change when the module call fails;
  - an identical subscription from another proposer refused;
  - unknown targets (non-canonical numbers, unknown plugin, subscription or delivery), a letter no longer dead, and a revoked subscription purged from the module, each giving `invalid_stage` or not found as specified;
  - a bad `version`, `destination` or `types` with the field name;
  - a duplicate proposal returning the same number or ID;
  - `dependency_unavailable` from a stand-in fault, and `ErrStale` through a hook that changes state during `admit`;
  - every command refused with `permission_denied` for a non-admin member and for an administrator of another workspace;
  - idempotent replay and a stale revision.
- **Leak test.** The JSON of the read and of every command result contains none of the following, in hex, base64 or raw form:
  - plugin or webhook public keys, proofs or manifest signatures;
  - `lease-` tokens or `hbp_` tokens;
  - webhook signatures or bodies;
  - private key bytes.
- **Signer test.**
  - While armed, the signer refuses any triple other than (boot workspace, `sample-erp-invoices`, `sample-webhook-key-1`).
  - After `Start`, the seed signer holds no key and the slice it held is zeroed.
  - `Sign` fails, and a module `ApproveKey` attempt returns `dependency_unavailable`.
  - The signer is disarmed when the seed fails partway.
- **AST tests** over all non-test files in `apps/api/server`:
  - `KindSystem` and `KindAssistant` from `plugins`, `webhooks` and `events` appear only in `seedIntegrations` and the `outboxReader` constructor. `integrations.go` and `integrationcmds.go` build no non-human actor.
  - The adapter's system actor field is used only in `outboxReader.Read` and `outboxReader.Bounds`.
  - Calls to these appear only in `seedIntegrations`:
    - `Dispatch`, `Confirm`, `Redirect`, `Report`;
    - `IssueToken`, `CheckToken`, `CheckEndpoint`, `CheckEgress`;
    - webhooks `ProposeKey`, `ApproveKey`, `RetireKey`;
    - events `Publish`.
  - No `"system"` or `"assistant"` string literal appears in request-path files.
  - `integrationdeps.go` imports no `net`, `net/*` or `crypto/tls` package.
  - The registered `integrations.*` command IDs are exactly the fifteen above.
- **Fault hooks.**
  - `integrations-read`, `plugins-<call>`, `webhooks-<call>` and `events-<call>` for every module call;
  - seed failure paths for each step;
  - concurrent reads and commands under the race detector.
- **Runs:**
  - `go test -race -count=40` on the new tests;
  - cross builds for linux/amd64, windows/amd64 and darwin/arm64;
  - 100% coverage of the new server files;
  - mutation testing of the four new server files and `signing.go`. Survivors are killed by a test or shown to be equivalent, with a reason.
- **Checks:** `gofmt` and `go vet` clean; the 1,500-line limit per file holds.
- **Reviews:** independent design and change review; `verification.md`.

# Design review round 2 clarifications

- The seed's `RequestRedeliver` records its requester and rationale in the mirror, and the seed read test asserts the pending request is shown and blocks a second request.
- Identical subscription content that is `proposed` or `active` is refused with `invalid_stage` for anyone, except the original proposer while it is still `proposed`, who gets the existing subscription back.
- The string-literal AST rule covers `integrations.go`, `integrationcmds.go` and `integrationdeps.go`; the KindSystem/KindAssistant check matches the `plugins`, `webhooks` and `events` package qualifiers only.
- The manifest digest is shown as an accepted public integrity value.

