---
type: ImplementationPlan
title: Installer register with signed releases, pre-install impact previews, bounded TLS diagnostics, checkpointed runs and one-time administrator registration
description: First increment of work item W-36. A register verifies signed release manifests with pinned artifact digests against dual-controlled release keys. From a fresh host observation it computes an impact preview that names services, ports, volumes, permission changes and the effect on existing data, and it blocks every conflict instead of changing it. It limits diagnostics to approved destinations and answers TLS failures only with correct trust-setting guidance. It runs installations as checkpointed steps bound to the accepted preview, so they resume after interruption and are invalidated by any change. It finishes with a one-time, expiring administrator registration that it stores only as a hash.
status: draft
---

# Scope and ownership

[HB-243](https://linear.app/hybrid-billing/issue/HB-243) starts work item W-36 (requirement HB-36; HB-36-D/I/V). The planning baseline:
- AR-09 is implemented in `modules/auth` (HB-169).
- DB-08 is implemented in `modules/onboarding` (HB-182).

This increment covers:
- DB-13;
- the HB-36-I items for signatures and hashes, fixed artifacts, pre-diagnostics, checkpoints and one-time administrator registration.

Base: main `2a091eb`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because the register decides what an installer may change on a customer host and who becomes the first administrator. Machine: mac_mini.

Allowed files: `modules/installer/*` and `docs/features/HB-243/*`.

Out of scope, recorded as residuals:
- upgrade and restore procedures;
- support bundles with de-identification, preview and disabled external upload (increment 2, HB-36-I);
- the host probe and the runner that actually inspect and change a host;
- persistence;
- the P01 screen;
- scale measurement (HB-36-V).

# Observed baseline

- **HB-36-D.** Independent installation in the customer environment is the default. The Linux VM container path is validated first. Closed networks, proxies, internal certificate authorities and external databases or storage are each judged per support scope.
- **DB-13.**
  - Services, ports, volumes and permission changes, and their impact on existing data, are shown before installation.
  - Connection troubleshooting never advises disabling TLS verification or granting broad administrator rights.
  - The diagnostic network is limited to an approved destination range.
  - AUD-DB-13 requires that an existing volume or port conflict is never changed without authorization, and that certificate errors get the cause and the correct trust setting instead of disabling verification.
- **HB-36-I.** Signatures and hashes, fixed images, pre-diagnostics, checkpoints, one-time administrator registration, and upgrade and restore procedures.
- **Platform independence.** `.agents/rules/platform-independence.md` applies. Platforms are data in the release manifest (portability targets), never assumptions of the register. Tested combinations are recorded separately and are not claimed here.
- **Sibling conventions.** `modules/plugins` provides the patterns used here:
  - numbered proposals;
  - Ed25519 keys with proof and weak-key refusal;
  - register-wide marks;
  - tagged hashed one-time secrets.

# Decision

A new package `modules/installer` follows the sibling conventions:
- one mutex and an injected clock that never goes backward;
- `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (with `Refusal{Reason}`) and `ErrLimit`;
- check order: form, lookup, repeat, rule, room;
- assistants refused as `ai_not_allowed` and system actors as `not_human` where a human is required;
- only standard-library imports.

It is constructed with `NewRegistry(Config{Random}, Limits{...}, clock)`.

## Proposals

Configuration that grants or widens what an installer may do is a numbered proposal:
- `Propose(Change, actor)` by a human;
- `Approve(number, actor)` by a different human, which rechecks every rule and applies the change;
- `Withdraw(number, actor)`, by the proposer only.

The proposal rules are the same as in `modules/plugins`:
- an identical pending proposal from the same proposer is returned again;
- every new proposal gets a new number;
- pending proposals are capped per scope, where a scope is a site or the register, and one proposer may hold at most a quarter of a scope's cap, rounded up;
- the register keeps at most `Proposals` applied proposals per scope;
- withdrawn and purged proposals read as `ErrNotFound`.

| Kind | Payload | Effect |
| --- | --- | --- |
| `key` | `ReleaseKey{Version, Public, NotBefore, NotAfter, Proof}` | Active release key. The proof is a signature over `hybrid-billing/release-key/v1 ‖ version` |
| `release` | `Release`, `Signature` | Published release |
| `site` | `Site{ID, Tenant, Platform, Probe}` | Registered site, bound to one tenant. `Probe` is the one system actor that reports observations and runs steps for the site |
| `probe` | `Site{ID, Probe}` | Replaces the probe identity |
| `config` | `SiteConfig{Site, Version, Ports, Volumes}` | Replaces the site's single override set, which belongs to one release version, and increments the configuration revision |
| `destination` | `Destination{Site, ID, Host, Port, Ranges}` | Approved diagnostic destination |
| `plan` | `Acceptance{Site, Version, Hash, Admin}` | Accepted preview, naming the human who will register as first administrator |

Probe identities follow the same rules as the plugin runtime identities:
- a probe serves exactly one site;
- an identity is never reassigned and is tombstoned register-wide;
- a site may use at most 16 probe identities over its lifetime.

Applying a `probe` or `config` change:
- invalidates every active (`open` or `interrupted`) run of the site, with `probe_changed` or `config_changed`;
- deletes the site's live bootstrap codes;
- retires the site's accepted plans;
- discards the site's latest observation, so the new state must be observed again.

## Release keys and releases

- **Keys.** Release keys follow the rules of `modules/plugins` publisher keys:
  - Ed25519, with weak and small-order keys refused;
  - an inclusive window of at most `KeyLife`;
  - names never reused, with at most `Keys × 64` names;
  - at most `Keys` active keys;
  - `RetireKey` by one human, which takes effect at once.
- **Release fields.** `Release`:

  | Field | Rule |
  | --- | --- |
  | `Version` | `MAJOR.MINOR.PATCH`, compared numerically |
  | `Key` | Release key name |
  | `Platforms` | 1–16 entries `{OS, Arch}` of identifiers, strictly ascending. These are portability targets, not tested combinations, and the register does not interpret them |
  | `Artifacts` | 1–32 entries `{Name, Kind, Digest}`, strictly ascending by name. `Kind` is `container_image`, `archive` or `package`, so containers are one option, not the only path. `Digest` is 64 lowercase hex characters of SHA-256 |
  | `Services` | 1–32 entries `{Name, Artifact, Ports, Volumes}`, strictly ascending by name. `Artifact` names a listed artifact. `Ports` are 0–8 entries `{Number 1–65535, Protocol tcp\|udp}`, strictly ascending. `Volumes` are 0–8 release volume names, strictly ascending |
  | `Volumes` | 0–32 entries `{Name, Data}`, strictly ascending. `Data` marks a volume that holds customer data. Every service volume must be listed, and every listed volume must be used |

- **Derived permissions.** Permissions are derived from the contents, never declared:
  - `run_service` and `write_config`, always;
  - `listen_port`, when any port is declared;
  - `create_volume`, when any volume is declared.

  `run_service` means only that a service is started, through whatever adapter the site uses. It implies no particular OS service manager.
- **Release validity.**
  - Lists are refused if they are out of order or contain duplicates, never re-sorted.
  - The signature covers `hybrid-billing/release/v1` and then every field in table order, length-prefixed and count-prefixed. A known-answer test pins the encoding.
  - At proposal and at approval, the signature must verify against an active, unretired key whose window contains the current time (`signature_invalid` or `key_inactive`).
  - A version is immutable (`ErrConflict`).
- **Revocation.**
  - `RevokeRelease(version, reason)` and `CompromiseKey(key, reason)` create marks with the `modules/plugins` semantics: the mark takes effect at once, a second human confirms it to make it permanent, and only the marker may withdraw it while it is unconfirmed.
  - While a release's own mark, or its key's compromise mark, is active:
    - the release cannot be previewed, accepted or started (`revoked`);
    - every active run of it, on every site, is invalidated (`release_revoked`);
    - every accepted plan for it is retired.
  - Withdrawing a mark revives neither.
  - A compromise mark also retires the key.

## Observations

- **Reporting.** `Observe(site, Observation, actor)` is called by the site's probe only (`not_probe`). It replaces the site's single latest observation, records the observation time, and assigns it an increasing sequence.
- **Observation fields.**
  - `Platform`, which must equal the site platform (`platform_mismatch`);
  - `Ports`: up to 1,024 entries `{Number, Protocol}` in use;
  - `Volumes`: up to 1,024 existing volume names;
  - `Services`: up to 1,024 existing service names;
  - `Granted`: the permissions already granted;
  - `Complete`, which must be true (`incomplete`). A probe that cannot list everything must not report a partial view.
  - Every list is strictly ascending and unique.
- **Freshness.** An observation is fresh for `ObservationLife` (inclusive) after it is recorded.
- **Ownership labels.** Observations carry no ownership labels, because labels on a host can be forged. In this increment, every existing resource the release would need is a conflict.

## Previews

This increment covers first installation only. Installing over an existing installation needs the upgrade contract of increment 2.

`Preview(site, version)` requires:
- the release to be published and not revoked;
- the site platform to be listed in `Platforms` (`unsupported_platform`);
- a fresh observation (`observation_stale`, or `ErrNotFound` without one);
- the site to have no first administrator and no completed run (`already_installed`).

It computes a deterministic `PreviewView`:

| Section | Content |
| --- | --- |
| `Services` | Each release service with action `create`, or conflict `service_exists` when the observation lists it |
| `Ports` | Each service port after overrides, with action `bind`. Conflicts: `port_in_use`, when observed in use; `port_duplicate`, when two services resolve to the same port and protocol |
| `Volumes` | Each release volume after overrides, with action `create`. Conflicts: `existing_volume`, when observed; `volume_duplicate`, when two release volumes resolve to one host name |
| `Data` | Each release data volume, with effect `new`. Each conflicting existing volume, with effect `untouched` |
| `Permissions` | Each derived permission not yet in `Granted`, as a change to grant |
| `Conflicts` | Every applicable conflict (a resource can have several, for example `existing_volume` and `volume_duplicate`), sorted by conflict code and then target. Overrides configured for another release version give conflict `config_mismatch` |

- **Overrides.** Overrides come only from the dual-controlled `config` change for that version:
  - each port override maps `service/port/protocol` to a new port;
  - each volume override maps a release volume to a host volume name.

  At proposal and at approval, overrides must name the version's existing services, ports and volumes (`not_declared`), port targets must be 1–65535, and volume targets must be valid identifiers. There are at most 64 port overrides and 64 volume overrides.
- **Hash.** The preview hash is the SHA-256 of `hybrid-billing/install-preview/v1` and the canonical encoding of these inputs and results:
  - the site and its tenant;
  - the probe identity;
  - the release version and its signature digest;
  - a canonical digest of the observation's content, but not its sequence or time;
  - the configuration revision;
  - every section.

  A host refresh that changes nothing therefore keeps the hash.
- **No side effects.** A preview never changes anything. The register has no operation that adopts, deletes, rebinds or wipes an existing resource. A conflict is resolved only by a reviewed configuration change, or by an operator outside the product, after which a new observation is reported.

## Acceptance and runs

- **Acceptance.** A `plan` proposal names the version, the preview hash and the nominee `Admin`, who is the human expected to register as first administrator. Proposing and approving both recompute the current preview, and require that:
  - the hashes are equal (`plan_changed`);
  - the preview has no conflicts (`conflicts`).

  An applied acceptance is usable once.
- **Starting a run.** `StartRun(site, number, actor)` by a human:
  - uses an unretired applied `plan` proposal of that site (`not_approved`, or `acceptance_used`), consuming it only on success;
  - requires the recomputed preview to have the accepted hash;
  - requires no active run of the site (`run_open`).

  It creates a run whose ID is `run-<n>`, with a counter per site. The run records the version, the hash, the configuration revision, the probe and the steps.
- **Steps.** Steps are derived from the preview, in order:
  1. `preflight`;
  2. one `artifact:<name>` for each artifact;
  3. one `volume:<host name>` for each volume, using the override name;
  4. one `service:<name>` for each service;
  5. `register_admin`.
- **Completing steps.** `Complete(site, run, step, Evidence{Digest}, actor)` is called by the run's probe only. Every step carries an evidence digest of 64 lowercase hex characters: for an `artifact:` step, the artifact digest; for any other step, the probe's opaque record of the step. It completes the next open step. An identical repeat of an already completed step returns the view, and any other step is `out_of_order`. An `artifact:` step requires its digest to equal the pinned digest. Otherwise the run fails with `digest_mismatch`, and a failed run can never be resumed. A digest the probe reports itself is a consistency check against the signed manifest, not an independent verification.
- **Interruption.** `Interrupt(site, run, reason, actor)` is called by the probe or a human. It keeps every completed checkpoint.
- **Resuming.** `Resume(site, run, actor)` by a human resumes from the first incomplete step. It requires:
  - the release is not revoked;
  - the configuration revision and the probe are unchanged;
  - no other active run exists;
  - a fresh observation.

  The preview is recomputed against that observation. Conflicts on resources that this run's completed `volume:` and `service:` steps created, including their ports, count as satisfied. Any other conflict invalidates the run (`host_changed`).
- **Cancelling.** `Cancel(site, run, actor)` by a human closes the run.

Run transitions:

| From | Operation | To |
| --- | --- | --- |
| `open` | `Complete` of the last step | `completed` |
| `open` | digest mismatch | `failed` |
| `open` | `Interrupt` | `interrupted` |
| `interrupted` | `Resume` | `open`, or `invalidated` |
| `open` / `interrupted` | `Cancel` | `cancelled` |
| `open` / `interrupted` | probe, configuration or release change | `invalidated` |

- **Repeats.** `Interrupt` of an `interrupted` run and `Cancel` of a `cancelled` run return the view.
- **Terminal states.** Every other operation on a terminal run is refused (`run_closed`).
- **Wrong state.** On an active run, an operation that the table does not allow from its state is refused (`not_allowed_state`). Examples are `Resume` of an `open` run, and `Complete` or `IssueBootstrap` on an `interrupted` run.
- **Partial installs.** A run that ends `cancelled`, `failed` or `invalidated` after some `volume:` or `service:` steps leaves those resources on the host. The next preview reports them as conflicts. The register never adopts them, so an operator removes them outside the product or waits for the upgrade contract of increment 2. This is safe under AUD-DB-13 and is tested.
- **Active runs.** `open` and `interrupted` runs are active. At most one active run exists per site.
- **Cross-site effects.** Every operation changes one site only, except release and key marks.

## One-time administrator registration

- **Issuing a code.** When `register_admin` is the next step of an `open` run, `IssueBootstrap(site, run, ttl, actor)` by the run's probe:
  - creates the code `hbi_` followed by 43 base64url characters of 32 random bytes, returned once;
  - stores only the SHA-256 of `hybrid-billing/install-bootstrap/v1 ‖ code`, the site, the run and the expiry;
  - accepts a `ttl` from 1 minute to `BootstrapLife`;
  - keeps at most one live code per site, so a new code replaces the old one;
  - refuses with `dependency_unavailable` when entropy fails or when the hash is already present.
- **Registering.** `RegisterAdmin(code, actor)` by a human:
  1. refuses a malformed code as `ErrInvalid`;
  2. requires a known hash (`unknown_code`) and no expiry, where the code is expired once the current time reaches the expiry (`expired`; the code is deleted);
  3. requires the run still `open`, with `register_admin` as its next step, and its probe and revision unchanged (`stale_code`; the code is deleted);
  4. requires the actor to be the accepted nominee (`not_nominee`; the code is kept).

  On success, it:
  - consumes the code;
  - records the actor as the site's first administrator;
  - completes the step and therefore the run.
- **Repeats and scope.**
  - A repeated or replayed code is refused (`unknown_code`).
  - `Administrator(site)` reads the record.
  - The record grants no role by itself. Granting the role through `modules/auth`, and checking that the actor has authority over the site's tenant, are upstream responsibilities and are recorded as residuals.

## Diagnostics

- **Destinations.** `Destination` entries are approved per site under dual control:
  - `Host` is a canonical IP literal (parsed with `net/netip`, unzoned, and equal to its `String()` form) or a lowercase DNS name of LDH labels without a trailing dot.
  - `Port` is 1–65535.
  - `Ranges` are 1–8 canonical address prefixes, strictly ascending, that the resolved address must fall into.
    - Each prefix is at least /16 for IPv4 and at least /48 for IPv6 (`ErrInvalid`), so no single approval opens a broad network.
    - An IP literal host must lie within its own ranges.
  - Destinations are unique per site, with at most `Destinations` per site.
- **CheckTarget.** `CheckTarget(site, destination, actor)` by the probe returns the approved host, port and ranges, or refuses (`not_approved`). The probe must connect only after this check, and only to an address within the ranges. The probe enforces this, which is recorded as a residual.
- **Report.** `Report(site, destination, Result, actor)` by the probe records the result in a ring of the last `Reports` results per site and returns `Guidance{Cause, Steps}`.
  - `Result` has these fields:
    - `Kind`: `dns`, `tcp`, `proxy` or `tls`;
    - `Failure`: a code valid for the kind;
    - `Address`: the resolved address, which must lie in the ranges (`address_not_approved`), except for a `dns` failure;
    - `Via`: an optional proxy, which must be another approved destination of the site (`not_approved`).
  - The guidance comes from a closed table:

    | Kind | Failure | Cause | Steps |
    | --- | --- | --- | --- |
    | tls | `unknown_authority` | The issuing CA is not in the product trust store | `add_internal_ca_to_product_trust`, `verify_chain` |
    | tls | `hostname_mismatch` | The certificate does not name the host | `use_certificate_hostname`, `reissue_certificate` |
    | tls | `expired` / `not_yet_valid` | Validity window | `check_clock`, `renew_certificate` |
    | tls | `weak_signature` | Weak algorithm | `reissue_certificate` |
    | proxy | `proxy_auth` | Proxy credentials | `configure_proxy_credentials` |
    | proxy | `proxy_refused` | Proxy policy | `configure_proxy_allowlist` |
    | dns | `dns_failure` | Name resolution | `check_dns` |
    | tcp | `refused` / `timeout` | Network path | `check_firewall`, `check_service` |
    | any | `unknown` | Unclassified | `collect_details` |
    | any | `ok` | No problem | none |

- **No unsafe guidance.** No guidance step disables verification, skips certificate checks or grants broad administrator rights. A test asserts that the closed step set contains no such step and that every pair maps only into it. Unknown codes and pairs that do not match their kind are `ErrInvalid`.
- **Copy.** The rendered copy for each step (P01) needs terminology review, which is recorded as a residual.

## Bounds

| Limit | Scope | Maximum |
| --- | --- | --- |
| `Keys` | register, active | 16 |
| `KeyLife` | — | 1 h to 3650 d |
| `Releases` | register | 10,000 |
| `Sites` | register | 100,000 |
| `Destinations` | per site | 64 |
| `Reports` | per site ring | 1,000 |
| `Runs` | per site, history | 100 (oldest closed runs purged) |
| `ObservationLife` | — | 1 min to 24 h |
| `BootstrapLife` | — | 1 min to 24 h |
| `Proposals` | pending per scope | 1,000 |

Other bounds:
- Identifiers are 1–128 bytes, and reasons are plain text of 1–200 bytes.
- The list sizes are those given above.
- The product of the maximums is an accepted theoretical bound. Operators size the limits to their memory budget, as in `modules/plugins`.

## Residuals

- **Increment 2:** upgrade and restore of an existing installation, including ownership of existing resources; support bundles.
- **Host probe and runner:** the probe and runner that inspect and change the host, including enforcing the connect-only-after-check rule and the ranges.
- **First administrator:** granting the role through `modules/auth`, and checking tenant authority upstream.
- **Elsewhere:** persistence; P01 screen copy review; scale measurement.

# Verification plan

Literal tests cover:
- every form rule and its boundaries;
- known-answer tests for the release encoding, the key challenge, the preview hash and the bootstrap hash;
- release signatures: tampering with any field, a retired key, a key outside its window, a compromised key, and a revoked release;
- proposals: repeats, withdrawal, self-approval, assistant and system refusals, the per-scope and per-proposer caps, and history pruning;
- previews: every action and conflict kind, overrides and `config_mismatch`, unsupported platforms, stale, incomplete or mismatched observations, and determinism of the hash under refreshes that change nothing;
- derived permissions;
- that no operation changes a conflicting resource;
- acceptance: changes after acceptance (observation, configuration, revocation) refused with `plan_changed` or `revoked`;
- runs: every row of the transition table; idempotent repeats; resume with a host change and with resources the run itself created; invalidation of open and interrupted runs by configuration, probe or release change; single use of acceptances, with no revival after a mark is withdrawn; `already_installed`; one active run per site; history bounds;
- wrong-state refusals, refused `StartRun` calls that leave the acceptance unused, and a preview after a partial run;
- bootstrap codes: issue, replacement, expiry boundaries, replay, a stale run, refusal of a code issued before a probe change, refusal of anyone but the nominee, and entropy failure and collision;
- diagnostics: canonical hosts and ranges, unapproved destinations, resolved addresses outside the ranges, an unapproved proxy, failure codes that do not match their kind, the full guidance table, and the absence of unsafe steps;
- probe binding: the probe of site A cannot act on site B;
- cross-site isolation.

Property test: random observations, configuration and probe changes, previews, acceptances, runs, interruptions, resumes, completions and marks across two sites, checked against an independent model of preview conflicts and run states. The test asserts these invariants:
- at most one active run per site;
- no `Complete` under a probe or revision other than the run's;
- no live code outside an open run whose next step is `register_admin`.

Checks:
- `-race -count=10`, a 386 build, gofmt and vet;
- the whole repository with `-race`;
- 100% coverage and mutation testing.

# Review history
- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1. Probe and configuration changes now invalidate active runs, codes, acceptances and observations, and the probe is part of the hash.
  - M1. Acceptances now carry the version.
  - M2 and M11. This increment is first installation only, with `already_installed`, and existing resources are conflicts.
  - M3. Active runs include interrupted runs, and the plan has a transition table.
  - M4. Resume now rechecks the host.
  - M5. The hash now uses the observation content, not its sequence.
  - M6. Permissions are now derived.
  - M7. Added `volume_duplicate`.
  - M8. Configuration is now bound to a version, with `config_mismatch` and override bounds.
  - M9. Hosts are now canonical, with ranges, resolved addresses, an approved proxy, kind/failure pairing, and the probe enforcement recorded as a residual.
  - M10. Releases now use artifacts of several kinds.
  - M12. `Administrator(site)` is added, sites are bound to a tenant, and the auth integration is a residual.
  - L1. Acceptances are single use and retired on invalidation.
  - L2. Registration needs the nominee, and the consumption rules are specified.
  - L3. Ownership labels were dropped.
  - L4. Observations must be complete and match the platform.
  - L5. Repeats of `Complete` are idempotent, with defined evidence.
  - L6. Added the `unknown` failure and a copy residual.
  - L7. Run identifiers are defined and the total bound is recorded.
  - L8. Interrupted runs are now included in invalidation.
- Round 2 (same reviewer): PASS. Low items and what changed:
  - L5 remainder. Every step now has defined evidence.
  - N1. Partial installs are documented and tested.
  - N2. Added `not_allowed_state`.
  - N3. An acceptance is consumed only on success.
  - N4. Wording fixed, a single override set per site, and `ErrStale` removed.
  - N5. Minimum prefix lengths added.
  - N6. The matching tests are added to the plan.

# Change review resolution

The change review (independent reviewer, read-only) returned FAIL with M1 to M3 and L1 to L7. Resolved as follows:
- M1. `Interrupt` records the observation sequence, and `Resume` refuses with `observation_stale` until a newer observation exists. The property model tracks the same rule.
- M2. Ranges overlapping `::/96`, `64:ff9b::/96` or `64:ff9b:1::/48` are refused. Ranges whose address is IPv4-mapped are refused, and so are ranges that would overlap `::ffff:0:0/96`, because such a range either starts inside it or also overlaps `::/96`. `Report` refuses IPv4-mapped addresses.
- M3. Known-answer tests now cover the bootstrap hash (checked against an independent Python computation) and the preview hash of the standard fixture (a regression pin computed by the implementation). A test also changes each preview section (data, permissions, services, actions, conflicts, volumes) on its own and shows that the hash changes.
- L1. The plan now states the actual order: conflict code, then target. Kind adds nothing, because each code belongs to one kind.
- L2. Every applicable conflict is listed: `port_in_use` together with `port_duplicate`, and `existing_volume` together with `volume_duplicate`. Data effects no longer depend on duplicates.
- L3. Interrupt reasons are plain text kept in `Note`. `Reason` holds only system codes (`interrupted`, `host_changed` and others).
- L4. The property test now issues and registers codes in separate steps, so the live-code invariant sees live codes. It also tries `Complete` under a retired probe, which must give `not_probe`. Refusal reasons beyond `not_probe` remain checked by the literal tests.
- L5. Removed `compare` and the write-only `run.acceptance`.
- L6. A probe or configuration change retires only the plans in that site's scope. Release marks still scan every scope.
- L7. A refused `Resume` returns an empty view.
- H1. A `proxy` result requires `Via`, and its address is checked against the proxy destination's ranges.
- H2. Port ownership on resume is accepted as a limit while observations carry no ownership labels.

The re-review (same reviewer, read-only) returned PASS. Its low items were applied:
- N1. The property model now computes code expiry itself.
- N2. Cancelling and invalidation clear the interrupt `Note`.
- N3. The NAT64 local-use prefix `64:ff9b:1::/48` is refused as well.
