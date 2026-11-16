---
type: ImplementationPlan
title: M04 service-account API with service users, credential metadata, and one-time credential secrets
description: Phase 3g of the pre-launch plan (HB-257). Adds a read-only credential metadata listing to modules/identity and, in apps/api, a service-account read and commands to create, suspend, reinstate and retire service users and to issue and revoke their credentials. A secret is shown once through the existing pickup path and only its verifier is kept.
status: final
---

# Scope and ownership

[HB-268](https://linear.app/hybrid-billing/issue/HB-268) adds the API behind M04 (service accounts). Base: main `393c927`. The M04 screen follows in HB-269.

**Ownership**
- Root (Claude Code) owns the plan, the implementation and the tests.
- An independent read-only reviewer reviews the design and the change, with a security focus (credentials and permissions).
- Machine: mac_mini.

**Allowed files**
- `modules/identity`: a new read method and its tests.
- `apps/api/server`: a new `services.go` and its tests; route, command registration and pickup changes in `boundary.go`, `admin.go`, `directory.go`, `users.go` and `openapi.go`.
- `docs/features/HB-268/*`.

# Observed baseline

- `modules/identity` already holds service users and credentials:
  - `CreateServiceUser`, `SuspendServiceUser`, `ReinstateServiceUser`, `RetireServiceUser`, `ServiceUsers`;
  - `AddCredential` and `RevokeCredential`.
  - Every one checks that the actor is an administrator of the workspace.
  - A service user's owner must be an active human member, and the module refuses otherwise.
  - A credential stores a 32-byte verifier, a validity of at most `CredentialValidity` (30 days in apps/api), and a revoked flag. At most `CredentialsPerServiceUser` (10) unrevoked, unexpired credentials exist per service user.
- The module has no way to list a service user's credentials. `ServiceInfo` carries ID, state, epoch, creator and owner only.
- `AccessByCredential` exists, but `apps/api` has no request path authenticated by a service credential.
- M01 hands out one-time set-password links through a pickup: the command returns an opaque `link_ref`, and `GET /api/v1/identity/links/{ref}` returns the URL once to the same session within 10 minutes. The secret never enters the command outcome, the idempotency store or the audit journal.

# Design

## Module: credential metadata, a direct lookup and a bound on ended credentials

- `func (h *Workspace) Credentials(actor Verifier, workspace, serviceID string) ([]CredentialInfo, error)`:
  - uses the same check as the other service methods (`r.service`): the actor must be a current administrator of the workspace, and the service user must belong to that workspace;
  - returns `CredentialInfo{ID, Validity, Revoked}` sorted by ID;
  - never returns the verifier.
- `func (h *Workspace) Credential(actor Verifier, workspace, credentialID string) (string, CredentialInfo, error)`:
  - is a direct lookup with the same administrator and workspace checks;
  - returns the owning service user's ID;
  - is used to resolve `revoke_credential` without scanning every service user.
- **Bound.** The module previously kept revoked and expired credentials forever.
  - After `AddCredential` and `RevokeCredential`, it now keeps only the newest `EndedCredentialsKept` (20) ended credentials per service user, by `NotAfter` then ID, and deletes older ones.
  - A pruned credential then answers `ReasonUnknown` instead of `ReasonExpired`; both are refusals.
  - Pruning removes the record that stopped an ID being reused, so each credential now carries a registry serial (the session serial counter). `AccessByCredential` puts it in the Answer, and `Recheck` refuses an Answer whose serial differs. An old Answer therefore cannot pass again after its ID is pruned and reused.
  - Each service user therefore holds at most 10 live plus 20 ended credentials.

## API read: `GET /api/v1/identity/services`

- Admitted like the other identity reads: a current session, the installation administration action, and the domain read lock.
- Returns:
  - `revision`;
  - `services`: each with `id`, `state` (`active`, `suspended`, `retired`), `version` (epoch), `creator` and `owner` with their logins, and `credentials`;
  - each credential: `id`, `not_before`, `not_after`, `state` (`active`, `expired` or `revoked`), and `usable`, which is true only for an active credential whose service user is active (a suspended or retired service user's credentials cannot authenticate);
  - `owners`: active human members with an active membership (`user_id`, `login`), the candidates for owner;
  - `limits`: `credential_days`, `credentials_per_service` and `services`, derived from `IdentityLimits` (30 and 10) and the service cap (200);
  - `now`.
- Credentials are listed live first, then ended, newest `not_after` first.
- Service users are read through `ServiceUsers` pages of the configured page size until exhausted.
- **Bound.** `service.create` refuses at 200 service users (retired ones included, because their IDs are not reused). The read is therefore at most 200 × 30 credential rows.

## Commands

All use the installation administration permission, are mutating, and require a reason (rationale), like M01–M03.

| Command | Target | Fields | Effect |
| --- | --- | --- | --- |
| `service.create` | `services` | `name` (text), `owner` (ID) | Creates `svc-<name>` owned by `owner`. |
| `service.suspend` | service ID | — | Suspends it; its credentials stop authenticating. |
| `service.reinstate` | service ID | — | Reinstates a suspended service user. |
| `service.retire` | service ID | — | Retires it and revokes every unrevoked credential. |
| `service.issue_credential` | service ID | `days` (integer 1–30) | Issues a credential and returns a pickup reference. |
| `service.revoke_credential` | credential ID | — | Revokes it. |

**Rules**
- `name`:
  - is 3–48 characters, not counting the `svc-` prefix, of lowercase letters, digits and single inner hyphens;
  - is prefixed with `svc-`, so it cannot collide with a human user ID or `sample-*`;
  - an existing ID, including a retired service user's, gives `invalid_request` with field `name`: the module's `ErrConflict` from `CreateServiceUser` is mapped to it explicitly.
- `owner` must be in `owners`, otherwise `invalid_request` with field `owner`. An administrator may own a service user, including one they create.
- **Resolve:**
  - checks the boot workspace;
  - looks targets up with the caller's session through the boot-workspace calls (`ServiceUsers`, `Credential`), not `findUser`;
  - an identity refusal resolves to `NotFound`, which the framework reports as `permission_denied`;
  - only `ErrUnavailable` resolves to `Unavailable`.
- State commands pass the epoch read under the same domain lock, so concurrent changes are caught by the framework's `expected_revision` check (`revision_conflict`), not by the module's epoch.
- The handler counts live credentials first. At the limit it returns `prerequisite_failed`, rather than the module's `ErrLimit`, which maps to "unavailable".
- `retire`:
  - revokes only active credentials (unrevoked and unexpired) after the state change, inside the same domain write, so pruning triggered by a revocation never removes an item the loop still has to visit;
  - a revoke failure after retirement returns `internal_error` ("partial"), like M01's `errPartial`, and the framework stores that for replay;
  - retirement alone already stops authentication, because `AccessByCredential` requires an active service user;
  - recovery is `revoke_credential` on each remaining credential, which the module allows on a retired service user.
- `issue_credential`:
  - creates the credential ID as `cred-` plus 16 random hex characters;
  - generates a 32-byte secret from `crypto/rand`;
  - stores the verifier SHA-256(secret), the same construction as sessions;
  - sets the validity from now to now + days × 24 h;
  - refuses a suspended service user with `prerequisite_failed`, and a retired one through the module's state refusal (`invalid_stage`);
  - runs every check before `AddCredential`, and nothing can fail after it, so a named failure, which is not stored for replay, never follows a created credential.
- `revoke_credential`:
  - resolves the credential in the boot workspace through the direct `Credential` lookup;
  - refuses an already revoked one through the module's state refusal (`invalid_stage`);
  - revokes an expired one without error;
  - drops that credential's unread pickup.

**Secret handling**
- The secret is written as `hbsk_` + base64url (no padding) into a pickup bound to the issuing session, for 10 minutes. The command outcome carries only `credential` (the ID) and `secret_ref`.
- `GET /api/v1/identity/links/{ref}` returns `{"secret": ..., "credential": <ID>}` for a credential pickup and `{"url": ...}` for a link pickup. A pickup is deleted on first read, as today.
- **The verifier input** is the raw 32 random bytes, not the `hbsk_` string. `AccessByCredential(credentialID, verifier)` needs the credential ID, which is why the pickup returns it. A later authentication path must decode the base64url text after `hbsk_` and hash those bytes.
- The secret never enters the outcome, the idempotency record, the audit journal, the logs or the listing.
- A replayed attempt returns the same outcome. A pickup that was already read is gone, so the secret cannot be read twice.
- Pickups for a service user:
  - several can coexist, one per issued credential, so issuing again leaves no credential whose secret nobody can read;
  - suspend and retire drop all of them.

**Faults:** a `services` fault hook drives the dependency-unavailable path in tests, like `roster`.

## What is not included

- Authenticating requests with a service credential (`AccessByCredential` in a request path). Credentials issued in pre-launch therefore cannot be used yet. This is stated as a residual, and in HB-269's screen copy.
- Transferring ownership, because the module has no such operation.
- A last-used time or use history, because the module records none.
- Grants for service users. The M03 grant command only offers human members.
- Purpose, audience or scope metadata from the preview form.
- Seeding a service user: the module requires an administrator session, so the list starts empty.

# Review history

- **Change review (PASS):** five Lows.
  - L1, reuse of pruned IDs: fixed with the credential serial.
  - L2, unread secrets kept in memory past the pickup window: fixed by pruning pickups at the start of every service command.
  - L3, the partial-retire window: residual.
  - L4, test gaps: covered by tests for another workspace's administrator, replay, the stale revision, and the recovery revoke.
  - L5, read cost: residual.
- **Round 2 (PASS):** two Lows. N1: retire revoked expired credentials too, and pruning could turn that into a false "partial"; it now revokes active ones only, with a test. N2: stale plan text, corrected.
- **Round 1 (FAIL):**
  - F1 (Medium): the read was unbounded. Fixed with module pruning, a direct lookup and the service cap.
  - F2–F9 (Low): addressed above. They covered `usable`, the retire recovery, multiple pickups, check ordering, the pickup returning the credential ID and the hash input, derived limits, Resolve mapping, the self-owned rule and the residuals.

# Tests and verification

- **Module:**
  - the listing and the lookup refuse a non-administrator, an unknown or foreign target, an invalid ID, and an unavailable registry;
  - neither exposes the verifier (the returned struct has three fields);
  - pruning keeps the newest 20 ended credentials, ordered by `NotAfter` then ID;
  - a pruned credential is refused.
- **API:**
  - each command: success, gate refusal, a self-owned or ineligible owner, name rules, the limit, suspended and retired refusals, stale epochs, faults and the partial retire;
  - the read: owners, the order, `usable`, and the states;
  - the service cap;
  - the pickup: once only, the same session only, expiry, and the secret absent from the outcome, the audit and the listing;
  - a credential's verifier matches the picked-up secret (checked through `AccessByCredential` in a test only).
- **Checks:** race with count=40; 386 and Windows cross builds; 100% statement coverage of the changed packages; mutation testing of the changed lines; gofmt; vet.

# Residuals for the owner

- Service credentials cannot authenticate any request in pre-launch.
- No ownership transfer and no use history.
- If the human owner leaves, the service user keeps working; neither `AccessByCredential` nor `Recheck` checks the owner.
- Reinstating a suspended service user brings back all its unrevoked credentials, including any suspected of leaking. Revoke those before reinstating.
- If the issuing session ends before the pickup is read, the credential stays live for up to its validity and nobody holds its secret. Nothing leaks; the listing does not mark it, so HB-269's copy advises revoking credentials whose secret was not saved.
- A retire can report "partial" in a microsecond window if a credential expires between the API's clock read and a pruning revocation. The service user is still retired; the result is cosmetic and is not covered by a test.
- One services read at the cap walks about 200 × 6,000 credential records under the registry lock. It is bounded, and acceptable for pre-launch.
- State resets on restart, as in every pre-launch phase.
