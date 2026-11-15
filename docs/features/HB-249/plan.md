---
type: ImplementationPlan
title: Demo isolation with disjoint environment namespaces, reserved mock destinations at an egress choke point, a mock-only typed configuration schema, confined idempotent reset and production admission
description: Second increment of work item W-38. A register keeps demo and production environments apart by disjoint database, key ring, evidence path and document number namespaces. Demo traffic may reach only registered mock destinations on reserved names. Demo configuration is checked against a typed schema that has no secret or secret-store kind. Reset touches only its own demo environment and is idempotent by reset ID. Production admits only real-origin inputs from itself, never synthetic, public or demo inputs or demo document numbers.
status: final
---

# Scope and ownership

[HB-249](https://linear.app/hybrid-billing/issue/HB-249) is the second increment of work item W-38, after HB-248. It covers:
- detail contract QA-10, except the Assistant conversation retention class, which belongs to W-43;
- source case SIM08: real external issuance is blocked, and reset has no effect on production data;
- the source plan's chapter 103 rule. That rule separates the demo environment ID, database, keys, evidence paths and document numbers. It blocks real credentials and external issuance, mail and payment by default, allows only mock destinations, and confines reset.

It does not cover:
- deployment packaging (`compose.demo.yaml` and the `demo init/reset/run/verify` commands);
- the screens and file output marking themselves;
- persistence;
- the drivers.

Base: main `b91a135`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because the register decides what may leave a demo and what may enter production issuance. Machine: mac_mini.

Allowed files: `modules/sandbox/*` and `docs/features/HB-249/*`.

# Observed baseline

- **QA-10.** Demo refuses entry and storage of real credentials, and its network allows only registered mock destinations. The environment ID, database, keys, evidence paths and document numbers are separated from production. Detecting every real secret by its shape is not guaranteed; a mock-only configuration schema, blocked secret-store connections and egress isolation are used together. Acceptance AUD-QA-10 requires these refusals:
  - real credential settings or connection paths;
  - destinations other than mocks;
  - a reset that touches production data.
- **Source plan, chapter 103.** The demo environment ID, database, keys, evidence paths and document numbers are separated from production. Real credential entry and external issuance, mail and payment are blocked by default, and the network allows only mock destinations. Screens, PDF, CSV, XLSX and API outputs show DEMO and the data source, with synthetic data marked. Reset initialises only that run under demo permission, and can never target production connections or volumes. Interruption and resumption reuse the same event ID without duplicates.
- **Chapter 70.** Moving to production needs new credentials, sources and policy approval. Synthetic evidence, virtual contracts and demo document numbers are never migrated.
- **Chapter 108.** The same data can never enter real issuance input in production.
- **Siblings.**
  - `modules/installer` has a canonical host rule.
  - `modules/support` refuses sensitive key fragments in schemas.
  - `modules/simulation` (HB-248) records which proof is `proven_unclassified` until environments are classified. Feeding this register's classification into it is a residual.

# Decision

A new package `modules/sandbox` follows the sibling conventions:
- one mutex and an injected clock that never goes backward;
- `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (with `Refusal{Reason}`) and `ErrLimit`;
- check order: form, lookup, repeat, rule, room;
- assistants refused as `ai_not_allowed` and system actors as `not_human` where a human is required;
- numbered proposals approved by a second human with a full recheck, and withdrawal by the proposer;
- register-wide pending caps: a total, and a quarter of it per proposer;
- bounded history of applied proposals. Withdrawn proposals are deleted at once.

It imports only the standard library, including `net/netip`.

The register cannot see a process's real sockets or files. It is the decision point that the egress layer, configuration writers, reset tooling and the issuance entry must call. Wiring those callers is a residual.

## Proposals

| Kind | Payload | Effect |
| --- | --- | --- |
| `environment` | `Environment` | A new environment with its namespaces |
| `destination` | `Destination` | A mock destination of a demo environment |
| `schema` | `Schema` | A mock-only configuration schema, register-wide |

`DisableDestination(id, actor)` lets one human disable a destination at once and permanently. Its ID and its host and port stay recorded.

## Environments

`Environment{ID, Mode, Database, KeyRing, EvidencePath, NumberPrefix}`:
- **`ID`** is 1–32 bytes of `[a-z0-9]`. It has no separator characters, so a volume name `<id>-<suffix>` belongs to exactly one environment.
- **`Mode`** is `demo` or `production`. It never changes.
- **`Database` and `KeyRing`** match `[a-z][a-z0-9_]{0,62}`: lowercase only, with no `.` or `:`. Case folding and qualified names therefore cannot make two names refer to the same object.
- **`EvidencePath`** is an absolute path: `/` followed by 1–16 segments of `[a-z0-9_-]`, each 1–64 bytes, joined by `/`, at most 200 bytes, with no trailing `/`.
- **`NumberPrefix`** is 2–16 bytes of `[A-Z0-9]` followed by `-`.
  - A demo prefix must start with `DEMO`.
  - A production prefix must not start with `DEMO`.
  - A violation of either rule is refused with `demo_prefix`.
- **Disjoint namespaces.** Across all environments ever registered, the following are refused with `namespace_shared`:
  - a `Database` or `KeyRing` equal to another environment's;
  - an `EvidencePath` equal to another's, or a segment prefix of it, in either direction;
  - a `NumberPrefix` equal to another's. Because a prefix is letters and digits followed by a single `-`, one prefix can be a string prefix of another only when they are equal, so equality covers the prefix case.
- **Permanence.** Environments are never changed or removed, and a repeated ID is `ErrConflict`.
- **Production first.** A demo environment is refused with `no_production` while no production environment is registered. Every real production system that a demo could collide with must be registered as a production environment first. Real systems that were never registered cannot be compared with, and this is a residual.

## Mock destinations

`Destination{ID, Environment, Kind, Host, Port}`:
- **`Kind`** is one of `erp`, `mailbox`, `webhook`, `payment`, `provider` or `notification`.
- **`Host`** follows the canonical host rule of `modules/installer`.
- **`Port`** is from 1 to 65535.
- **Demo only.** The environment must be `demo`. Otherwise the destination is refused with `not_demo`.
- **Reserved hosts.** The host must be a DNS name ending in `.test`, `.invalid` or `.example` (RFC 2606), bound to mock containers. Otherwise it is refused with `not_reserved`.
  - IP literals are refused.
  - Loopback, `localhost` and `.localhost` are also refused, because on a shared host they can reach production services.
  - The registered domains `example.com`, `example.net` and `example.org` are not reserved.
- **`ID`** is 1–64 bytes of `[a-z0-9_-]`. There is no `.` or `:`, so a destination ID used as a setting value can never read as an address.
- **Uniqueness.** A host and port pair is unique across the whole register (`ErrConflict`), so one demo cannot reach another demo's mock. Destination IDs are register-wide and never reused.

## Egress choke point

`Allow(Request{Environment, Kind, Host, Port}) error` is a read and changes no state. A host that fails the canonical host rule, or a port outside 1–65535, is `ErrInvalid` in every mode. This catches uppercase, trailing dots, IPv4-mapped and other non-canonical forms.
- **Demo.** The request is allowed only when it matches an enabled destination of that environment with the same kind, host and port. Otherwise it is refused with `not_mock`. A disabled destination is refused with `not_mock` too.
- **Production.**
  - A reserved host, or a host and port pair registered as any demo destination, is refused with `mock_destination`, so production never treats a mock as real. Every registered mock has a reserved host, so the reserved-host check alone covers this case.
  - Any other request returns `nil`. This only means that this register does not refuse it. Production egress policy belongs to the callers' own registers, such as `modules/installer` and `modules/support`.

## Mock-only configuration

`Schema{ID, Fields}`:
- **Fields.** 1–128 `FieldSpec{Key, Kind, Values}`, strictly ascending by key.
- **Keys** match `[a-z][a-z0-9_]{0,63}`. A key is refused with `sensitive_key` when it contains a fragment from either list:
  - the `modules/support` sensitive list;
  - connection-path fragments: `url`, `uri`, `host`, `endpoint`, `server`, `vault`, `bucket`, `path`, `addr`, `domain`, `port`, `ip`, `target`, `proxy`, `smtp`, `remote`, `callback`.

  Over-refusal is accepted.
- **Kinds** form a closed set: `enum`, `integer`, `flag` and `destination`. No kind can hold free text, a secret, a credential or a secret-store reference. Such a setting therefore cannot be declared, entered or stored for a demo.
- **Values.**
  - `enum` needs 1–64 strictly ascending `Values` matching `[a-z][a-z0-9_-]{0,63}`. There is no `.` or `:`, so no value can be a host name or URL.
  - `destination` needs exactly one value: the destination `Kind` it accepts, which is one of the six kinds. Any other value is `ErrInvalid`.
  - Configuration writers never read an `enum` or `integer` value as an address. Demo egress is limited by `Allow` in any case.
  - `integer` and `flag` take no values.

`CheckConfig(Config{Environment, Schema, Settings}) error` is the gate a configuration writer calls before it stores demo settings:
- **Environment.** The environment must be `demo`. Otherwise the call is refused with `not_demo`, because production configuration with real credentials is handled by the secret store, not here.
- **Settings.** Settings are 0–128 `Setting{Key, Value}`, strictly ascending by key. Every key must be declared in the schema; otherwise the call is refused with `undeclared`. Each value must fit its kind; otherwise the call is refused with `ErrInvalid`:
  - `enum`: one of the declared values;
  - `integer`: canonical decimal `int64`;
  - `flag`: `true` or `false`;
  - `destination`: the ID of an enabled destination of the same environment, of the declared kind.

## Reset

`BeginReset(Plan{Environment, Key, Targets}, actor) (ResetView, error)`:
- **Actor.** The actor is a human or a system actor, under demo permission, which the caller enforces. Assistants are refused with `ai_not_allowed`.
- **Environment.** It must be `demo`. Otherwise the reset is refused with `not_demo`.
- **ID.** The register allocates the reset `ID`: the next number in a per-environment sequence that starts at 1. The tooling saves the ID before acting, and resumes with it.
- **`Key`** is an optional request key (an identifier) that makes the start idempotent too. A repeated key with the same targets returns the existing reset. A repeated key with different targets is `ErrConflict`. A lost reply or a crash before the ID is saved therefore does not leave an orphaned reset. The key is forgotten when its reset is evicted.
- **Targets.** 1–64 `Target{Kind, Name}`, strictly ascending by kind and then name. `evidence` names follow the `EvidencePath` grammar, which excludes `.` and `..`. Every other name is an identifier. Each target must lie inside the environment; otherwise the reset is refused with `outside_env`:

  | Kind | Inside when `Name` is |
  | --- | --- |
  | `database` | the environment's `Database` |
  | `keyring` | the environment's `KeyRing` |
  | `evidence` | the `EvidencePath`, or a path below it by segments |
  | `numbers` | the `NumberPrefix` |
  | `connection` | a destination ID of this environment |
  | `volume` | the environment ID, or the environment ID followed by `-` and more identifier characters |

- **Progress.** A reset records the actor and the time it began, plus a completion flag for each target.
- **`CompleteTarget(environment, id, target, actor)`** marks one target done.
  - It is idempotent.
  - It refuses assistants (`ai_not_allowed`).
  - A target that is not in the reset is `ErrNotFound`.
  - When every target is done, the reset is `completed`.
- **Resuming.** `ResetRecord(environment, id)` returns the reset with its completion flags. The tooling re-runs only the targets that are not done, so a resumed reset has neither duplicates nor gaps.
- **Caller precondition.** Each target action is idempotent. Two tooling instances that resume the same reset may both run an unfinished target, so this register does not claim targets. This is stated.
- **Eviction.** Each environment keeps its 10,000 most recent resets, by ID. Because the register allocates IDs in order, the oldest is always the lowest, and the highest evicted ID is a watermark below every live ID. Resuming or completing an ID at or below the watermark is refused with `stale`. `stale` means "begin a new reset", which is safe because every reset is confined and its target actions are idempotent. An ID above the highest allocated is `ErrNotFound`.

## Production admission

`Admit(Input{Environment, Source, Database, Origin, Number}) error` is the check the issuance entry calls for each input:
- `Source` is the environment the input was produced in.
- `Database` is the database the input was read from. Inputs must be read from the environment's own primary `Database`; reading through replicas or other databases is a residual.
- `Origin` is `real`, `public`, `converted` or `synthetic`.
- `Number` is optional.

The rules:
- **Source.** `Source` must equal `Environment`, and `Database` must equal that environment's `Database`. Otherwise the input is refused with `cross_environment`, so demo data, virtual contracts and demo evidence can never be carried into production.
- **Declared provenance.** These fields are declared by the caller. Data copied into the production database and relabelled `real` at its origin cannot be detected here, and this is a residual.
- **Demo.** A demo environment admits any origin. A `Number` must start with its `NumberPrefix` and be longer than it; otherwise the input is refused with `foreign_number`.
- **Production.**
  - Only `real` is admitted. Every other origin is refused with `not_real`, so neither synthetic nor public sample data can be issued.
  - A `Number` must start with the environment's `NumberPrefix` and be longer than it; otherwise the input is refused with `foreign_number`. A production prefix never starts with `DEMO`, so this also refuses demo numbers.

## Labels

`Label(environment, origin, source) (LabelView, error)` returns the marking every output must carry. `source` is the identifier of the data source, such as a dataset or provider:
- **Demo.** `Marking` is `DEMO`. The view also carries the origin, shown as `synthetic` when synthetic, and the source.
- **Production.** `Marking` is empty.

The screens, PDF, CSV, XLSX and API writers apply it. That wiring is a residual.

## Reads

Every read returns a copy:
- `Environment`;
- `Destinations(environment)`;
- `Schema`;
- `ResetRecord(environment, id)`;
- `Proposal`.

## Bounds

| Bound | Limit |
| --- | --- |
| Environments | 1,000 |
| Destinations per environment | 256 |
| Schemas | 1,000 |
| Fields per schema, settings per config | 128 |
| Values per enum | 64 |
| Targets per reset | 64 |
| Resets kept per environment | 10,000. The oldest (lowest ID) is evicted and the watermark advances |
| Pending proposals | 1,000 register-wide, 250 per proposer |
| Applied proposals kept | 1,000 |
| Identifiers | 1–128 bytes of `[A-Za-z0-9._:-]` |

## Residuals

- **The Assistant conversation retention class of QA-10.** It is planned for W-43.
- **Wiring the callers.**
  - The egress layer, configuration writers, reset tooling and the issuance entry calling this register.
  - The output writers applying labels.
  - Feeding environment classes into `modules/simulation` proof.
- **Real sockets.** This register cannot see sockets, DNS answers or files.
  - A caller that skips `Allow` is not blocked.
  - A reserved name that resolves to a real address through a local resolver is not detected. The demo must run in its own network namespace, where reserved names resolve only to mock containers. This is a deployment precondition.
  - Symlinks below an evidence path are not detected.
- **Unregistered systems and declared provenance.** Real systems never registered as production environments cannot be compared with. One placeholder production environment satisfies `no_production`, so that guard is weak. Relabelled provenance at `Admit` cannot be detected.
- **Admission databases.** Admission from replicas or several databases per environment.
- **Packaging.** Deployment packaging and the demo commands.
- **Persistence and screens.**
- **System-level cases.** SIM08 and AUD-QA-10 are NOT_RUN.
- **Owner decisions not confirmed by the owner:**
  - the reserved host suffixes, and the exclusion of loopback;
  - the refused key fragments;
  - the `DEMO` prefix rule;
  - the destination kinds;
  - the bounds.

# Verification plan

Literal tests:
- **Form and bounds.** Every form rule and every bound, at the exact limit and one past it.
- **Proposals.** Proposal rules, actor kinds and recheck at approval.
- **Namespaces.**
  - Every `namespace_shared` case, both directions of each prefix.
  - The `DEMO` prefix rules.
  - Sibling paths that share a string prefix but not a segment prefix, such as `/a/b` and `/a/bc`.
- **Destinations.**
  - Every reserved suffix, and near misses: `test.com`, `example.com`, `localhost`, `a.localhost`, `127.0.0.1`, `::1`.
  - Uniqueness of host and port across environments.
  - `no_production`.
  - `not_demo`, uniqueness and disabling.
- **Egress.** Matching on kind, host and port. Disabled destinations. Production refusal of reserved and registered mock hosts. Non-canonical forms in both modes.
- **Configuration.** Every kind, `undeclared`, `sensitive_key`, foreign and disabled destinations, and `not_demo`.
- **Reset.**
  - Every target kind inside and outside, including:
    - another demo environment's names and production names;
    - evidence paths that share a string prefix but not a segment prefix;
    - volumes named after another environment.
  - Allocated IDs, request keys, completion flags and resumption.
  - `CompleteTarget` actor and target rules.
  - Eviction and `stale` at the exact cap.
  - `not_demo`.
- **Admission.** Every origin, `cross_environment` both ways, and number prefixes.
- **Labels.** Both modes.
- **Concurrency.** Concurrent calls under `-race`.

Property test: random environments, destinations and requests. An independent model of namespaces and allowed destinations must agree with `Allow`, `BeginReset` and `Admit` on every request.

Checks: `-race -count=10`, the 386 build, gofmt, vet, `go test -race ./...`, 100% statement coverage and mutation testing.

# Review history

- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1. Environment IDs are `[a-z0-9]`, so volume names cannot cross environments.
  - H2. `Database` and `KeyRing` are lowercase, with no qualifiers.
  - H3. Loopback, `localhost` and `.localhost` are no longer reserved. A separate network namespace is a stated precondition.
  - M1. Connection-path key fragments are refused, and enum values cannot contain `.` or `:`.
  - M2. Per-target completion and resumption.
  - M3. `Database` is checked at `Admit`. Relabelled provenance is a residual.
  - M4. `no_production`, plus a residual for unregistered systems.
  - L1. `Allow` validates host form.
  - L2. The evidence target grammar is stated. Symlinks are a residual.
  - L3. Eviction with a watermark.
  - L4. Destinations are unique register-wide. A `destination` field declares its kind. `Label` carries the source.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - N-M1. The register allocates reset IDs per environment, so the watermark can never block new resets.
  - N-L1. `CompleteTarget` rules are stated. Idempotent target actions are a caller precondition.
  - N-L2. A `destination` value must be one of the six kinds.
  - N-L3. Admission reads from the primary `Database`. Replicas are a residual.
  - N-L4. More fragments, and writers never read values as addresses.
  - N-L5. The weak guard is noted in the residuals.
- Round 3 (same reviewer): PASS. Low items folded in:
  - R3-L1. An optional request `Key` makes `BeginReset` idempotent.
  - R3-L2. The meaning of `stale` is stated.
  - R3-L3. The names in the verification plan are fixed.

# Change review resolution

The change review (independent reviewer, read-only) found no isolation hole in the code. It returned FAIL for test gaps. Resolved:
- M1. The target, identifier and path bounds are tested at the limit and one past it.
- M2. The property model now covers:
  - `keyring`, `numbers` and two-target plans;
  - `Admit` with numbers and the `converted` origin;
  - destination refusal reasons.

  A literal test refuses `DEMO-1` as a numbers target.
- L1. Destination IDs are `[a-z0-9_-]`.
- L2. The schema rule now runs before the room check.
- L3. `demo_prefix` is named in the plan.
- L4. Two checks could never decide an outcome and were removed: the production `DEMO` number clause and the production endpoint lookup in `Allow`. The test was renamed. A number must now be longer than its prefix.
- L5.
  - Tests cover the approval recheck for `namespace_shared`, and a sensitive key that is not the first field.
  - `clean` validates the stored copy.
  - The concurrency test covers every call.
  - The progress wording is corrected.

The re-review (same reviewer, read-only) returned PASS. Its one wording low (R1) is folded into the egress section.
