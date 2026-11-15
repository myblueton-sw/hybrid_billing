---
type: ImplementationPlan
title: Component register with pinned candidates, evidence digests, adoption rules for offline operation, telemetry and license boundaries, and release manifests checked for exact versions and notices
description: Second increment of work item W-40. A register records third-party candidates and components pinned to an exact version and commit, with licence, notice, SBOM and test evidence digests, offline and telemetry behaviour, and modification and distribution modes. Adoption requires passing tests, offline operation, no required telemetry and a licence allowed for the declared distribution. A release manifest is checked so that every shipped component is adopted at its exact pinned version with its notice, and it gets a deterministic notice bundle digest.
status: final
---

# Scope and ownership

[HB-253](https://linear.app/hybrid-billing/issue/HB-253) is the second increment of work item W-40, after HB-252. It covers:
- the HB-40-I candidate record: pinned version or commit, LICENSE, dependencies or SBOM, offline and telemetry behaviour, modification and distribution method, and test results;
- the notice and licence-boundary checks of HB-40-V;
- the policy that the product's source-available terms are never mixed with third-party licences ([LICENSING.md](../../../LICENSING.md), [THIRD_PARTY_NOTICES.md](../../../THIRD_PARTY_NOTICES.md)).

It does not cover:
- scanning packages or producing SBOMs;
- legal review, since HB-40 states that legal suitability is not settled by this analysis;
- persistence and screens.

Base: main `cefde87`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because the register decides which third-party material may ship. Machine: mac_mini.

Allowed files: `modules/thirdparty/*` and `docs/features/HB-253/*`.

# Observed baseline

- **HB-40.** A small own core is compared with candidate OSS on correctness, isolation, operation and rights. The product's source-available commercial policy is not mixed with third-party licences.
  - Each candidate records its pinned version or commit, LICENSE, dependencies or SBOM, offline and telemetry behaviour, modification and distribution, and test results.
  - Adopting OSS for the whole core is considered only when a real gap is proven.
  - Acceptance verifies distribution notices, licence boundaries, history access after expiry, and upgrades and data migration. Legal suitability is not settled.
- **LICENSING.md and THIRD_PARTY_NOTICES.md.**
  - Third-party libraries keep their own licences and notices, and the root licence does not relicense them.
  - Before packaging a release, the components and versions actually shipped are inventoried and carry their required notices.
  - A package name in a candidate list does not prove adoption.
- **product-policy.md.** No automatic external reporting or enforcement runtime. A component that requires telemetry would add automatic external reporting.

# Decision

A new package `modules/thirdparty` follows the sibling conventions:
- one mutex and an injected clock that never goes backward;
- the shared error set and `Refusal{Reason}`;
- check order: form, lookup, repeat, rule, room;
- human actors only, with assistants refused as `ai_not_allowed` and system actors as `not_human`;
- numbered proposals approved by a second human with a recheck, withdrawal by the proposer, register-wide pending caps (1,000 in total, 250 per proposer), and 1,000 applied proposals kept.

It imports only the standard library.

## Components

A `component` proposal registers `Component{ID, Name, Version, Commit, Source, License, LicenseText, Notice, SBOM, Artifact, Patch, Dependencies, Supersedes, Offline, Telemetry, TelemetryEvidence, Modification, Distribution, Tests, Passed}`:
- **`ID`** identifies one exact pinned version. A new version is a new ID with the same `Name`.
- **`Name`** is compared in lower case everywhere.
- **`Version`** is the exact upstream version text, an identifier. The identifier rule already refuses `^`, `~`, `>`, `<`, `=`, `*` and spaces. These are also refused, because they are not pins:
  - a whole dot-separated segment equal to `x` or `X`;
  - the floating labels `latest`, `main`, `master`, `head`, `stable`, `nightly`, `dev` and `trunk`, in any letter case.

  These are refused with `unpinned_version`.
- **Uniqueness.** One (`Name`, `Version`) pair and one (`Name`, `Commit`) pair each exist at most once among records that are not rejected or revoked. A duplicate is `ErrConflict`.
- **Corrections.** A record for a pair that already has records is allowed only when:
  - every earlier record with that pair is rejected or revoked;
  - the new record names in `Supersedes` the most recently registered of all earlier records that match either pair.

  `Supersedes` must be empty when there is no earlier record, and anything else is `ErrConflict`.

  It still needs the normal second-human approval, and the `Supersedes` link stays readable. A rejected version therefore cannot be re-registered silently, but a revoked one can be corrected explicitly, for example with a fixed licence, a rebuilt `Artifact` or a replaced dependency. To replace a revoked dependency of a parent, the parent is revoked first and then corrected, because a live parent keeps its pair.
- **`Commit`** is 40 or 64 lowercase hex characters.
- **`Source`** is an `https://` URL of at most 200 bytes, printable ASCII, with a non-empty host and no user information, query or fragment. An `@` in the path is allowed, as in npm scopes and Go module proxy paths.
- **`License`** is an SPDX identifier from the closed set below. Unknown licences are refused with `unknown_license`, and an expression that combines licences is out of scope for this increment.
- **Evidence digests.** These are hex SHA-256 digests:
  - `LicenseText`: the licence text;
  - `Notice`: the retained notice text;
  - `SBOM`: the dependency inventory;
  - `Tests`: the test report;
  - `Artifact`: the shipped bytes (package or binary). It is required for `bundled` and `separate`, and empty otherwise.
  - `Patch`: the modification record. It is required for `patched` and `forked`, and empty for `unmodified`.

  The source pin (`Commit`) and the shipped-bytes pin (`Artifact`) are therefore both recorded.
- **`Dependencies`** lists 0–256 component IDs, strictly ascending, that ship with this component at run time. The transitive licence check works through them.
  - A dependency that is not registered is `ErrNotFound` at registration.
  - When the parent is `bundled` or `separate`, a dependency that is `tool_only` or `not_shipped` is refused with `tool_dependency`. Build and test tools are therefore registered on their own, not as dependencies, and never block a shipped parent.
- **`Offline`** says whether the component works without network access.
- **`Telemetry`** is `none`, `optional_off` (present but off by default and switchable) or `required`. `optional_off` needs `TelemetryEvidence`, the digest of the configuration evidence that keeps it off; it is empty otherwise.
- **`Modification`** is `unmodified`, `patched` or `forked`.
- **`Distribution`** is `bundled` (shipped inside the product), `separate` (installed beside it from its own package), `tool_only` (build or test tooling, never shipped) or `not_shipped`.
- **`Passed`** says whether the recorded tests passed.

A component is registered as a `candidate`. Records are immutable, and the ID is never reused.

## Licence boundaries

The licence set is closed. Each licence is allowed for the distributions shown:

| Licence | `bundled` | `separate` | `tool_only` |
| --- | --- | --- | --- |
| `MIT`, `Apache-2.0`, `BSD-2-Clause`, `BSD-3-Clause`, `ISC`, `Zlib` | yes | yes | yes |
| `OFL-1.1`, `CC-BY-4.0` | yes | yes | yes |
| `MPL-2.0` | `unmodified` only | yes | yes |
| `CC-BY-SA-4.0` | no | yes | yes |
| `LGPL-2.1-only`, `LGPL-3.0-only` | no | yes | yes |
| `GPL-2.0-only`, `GPL-3.0-only`, `AGPL-3.0-only` | no | no | yes |
| `PolyForm-Noncommercial-1.0.0` | no | no | no |

- **`not_shipped`** allows every licence, because nothing ships.
- **Refusal.** A combination outside the table is refused at adoption with `license_boundary`.
- **The product's own licence.** PolyForm is refused for any shipped or tooling third-party component, because the product's own terms must not be mixed into third-party material.
- **Status of the table.** It is an engineering boundary for this register, an owner decision not confirmed by the owner, and not legal advice.

## Adoption

An `adopt{Component}` proposal moves a candidate to `adopted` when every rule holds:
- `Passed` is true; otherwise `tests_failed`.
- `Offline` is true unless `Distribution` is `tool_only` or `not_shipped`; otherwise `not_offline`.
- `Telemetry` is not `required`; otherwise `telemetry_required`.
- The licence is allowed for the distribution and modification; otherwise `license_boundary`.
- Every dependency is `adopted` (otherwise `dependency_not_adopted`). The boundary check below is defence in depth: a shipped parent's dependencies have already passed a check at least as strict at their own adoption, and in practice it is reached by a `tool_only` parent with a `not_shipped` dependency. Its licence must be allowed for the distribution it ships in, with its own modification (otherwise `dependency_boundary`). That distribution is the dependency's own `separate` when it is installed separately, and the parent's distribution otherwise. Transitive material therefore cannot ship under a stricter licence through a permissive parent, while a separately installed LGPL dependency stays allowed beside a bundled parent.

- **Rejection.** A `reject{Component, Reason}` proposal moves a candidate to `rejected`, with a reason text.
- **Revocation.** A `revoke{Component, Reason}` proposal moves an adopted component to `revoked`, for example when its licence was recorded wrongly or its notice was wrong. Releases already recorded stay as history. Every later release check reports `revoked`.
- **Finality.** Rejected and revoked records are final, and a new version is a new record.

## Release manifests

`CheckRelease(Release{Version, Commit, Components})` is a read:
- **Form.** `Version` is an identifier, `Commit` is the product's own 40- or 64-character commit, and `Components` lists 1–1,000 component IDs, strictly ascending.
- **Problems.** It returns every problem, not only the first:

  | Problem | When |
  | --- | --- |
  | `unknown` | the ID is not registered |
  | `not_adopted` | the component is a candidate or rejected |
  | `revoked` | the component was revoked |
  | `missing_dependency` | a `bundled` or `separate` component has a dependency missing from the manifest. Other components are already reported as `not_shipped` |
  | `not_shipped` | its distribution is `tool_only` or `not_shipped`, so it must not be in a release |
  | `duplicate_name` | two components share a `Name` in lower case, which would mean two versions of one component |

- **Result.** `Ready` is true only when there are no problems.
- **Notice bundle.** When the release is ready, `Bundle` is the SHA-256 of the encoding in which every field is prefixed with its length as an 8-byte big-endian unsigned integer:
  - `hybrid-billing/notice-bundle/v1`;
  - the release version and the product commit;
  - the component count;
  - then, for each component in ID order: `ID`, `Name` as stored, `Version`, `Commit`, `License`, `LicenseText`, `Notice`, `Artifact`, `Patch`, `Distribution` and `Modification`.

  When the release is not ready, `Bundle` is empty.

  A release package carries the notice texts whose digests are in that bundle, so the shipped notices can be checked against the register.

`RecordRelease(Release, actor)` needs a human actor, and stores a ready release once by version, with the actor, the time, the product commit and the bundle. Its repeat check comes before the readiness rule, so repeating the same release is idempotent even after a later revocation. A different list or commit under the same version is `ErrConflict`. Recorded releases are immutable and readable, so that history and upgrades can be traced.

## Reads

Every read returns a copy:
- `Component(id)`;
- `ByName(name)`, the versions of one component in ID order;
- `Release(version)`;
- `Proposal`.

## Bounds

| Bound | Limit |
| --- | --- |
| Components | 10,000, permanently, because records are never removed so that history stays traceable |
| Recorded releases | 10,000 |
| Components per release | 1,000 |
| Identifiers | 1–128 bytes of `[A-Za-z0-9._:-]` |
| Reason text | 1–200 bytes of UTF-8, with no control characters |

## Residuals

- The HB-40 comparison of a small own core with candidate OSS, and the proof of a real gap before adopting OSS for the whole core. These are recorded as evidence outside this register (owner decision and engineering report), not here.
- Two dependencies that need different versions of one component cannot ship together (`duplicate_name`). That is intended: nested versions are a residual.
- Dependencies are only as complete as the declared list. Producing SBOMs and scanning the packages actually shipped, so that they are compared with a manifest. A manifest that leaves out a shipped component cannot be detected here.
- Licence expressions combining several licences, and licences outside the closed set.
- Legal review of the boundary table.
- Persistence and screens.
- The HB-40-V verification of history access after expiry is covered by HB-252 and the residuals of other modules. Upgrades and data migration are covered by HB-245 and HB-247.
- Owner decisions not confirmed by the owner:
  - the licence table;
  - the offline rule;
  - the telemetry rule;
  - the floating-label list.

# Verification plan

Literal tests:
- **Forms and bounds.** Every form rule and bound, including version ranges refused and commits of 40 and 64 characters.
- **Proposals.** Proposal rules and actor kinds.
- **Adoption.** Every adoption refusal, including dependency rules. Every cell of the licence table, at both modification values for `MPL-2.0`. Revocation.
- **Uniqueness.** Name/version and name/commit uniqueness, with mixed letter case.
- **Releases.** Every problem, several problems at once, a ready release, the bundle known-answer test cross-checked with independent Python, `RecordRelease` repeats and conflicts.
- **Copies.** Reads return copies.
- **Concurrency.** Concurrent calls under `-race`.

Property test: random components, adoptions and release lists. An independent model of the licence table and release problems must match.

Checks: `-race -count=10`, the 386 build, gofmt, vet, `go test -race ./...`, 100% statement coverage and mutation testing.

# Review history

- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1. `revoke`.
  - H2. `Dependencies`, with adoption and release checks.
  - M1. Name/version and name/commit uniqueness.
  - M2. `Artifact` and `Patch` digests.
  - M3. `LicenseText`, and a fuller bundle.
  - L1. Floating labels and the `x` segment.
  - L2. Lower-case names.
  - L3. An empty bundle when not ready, and a human, timed, commit-bound `RecordRelease`.
  - L4. The bound is permanent.
  - L5. `TelemetryEvidence`.
  - The HB-40 core comparison is a stated residual.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - M1. Explicit `Supersedes` corrections after rejection or revocation.
  - M2. Tool dependencies are refused for shipped parents (`tool_dependency`).
  - L1. Nested versions are a stated residual.
  - L2. A separately installed dependency is checked against its own distribution.
  - L3. The bundle uses `Name` as stored, and the `RecordRelease` repeat check comes first.

The package was renamed to `modules/thirdparty`, because `modules/components` already holds the data-centre components register, which this ticket does not touch.
- Round 3 (same reviewer): PASS. Low items folded in:
  - L1. `Supersedes` must name the most recent earlier record across both pairs, or be empty.
  - L2. The order for replacing a dependency is stated.
  - L3. The placeholder is removed.

# Change review resolution

The change review (independent reviewer, read-only) returned PASS with lows. All are folded in:
- L1. `Source` is parsed with `net/url`. It needs a non-empty host and no user information, query or fragment, and allows `@` in the path.
- L2. `missing_dependency` applies only to shipped components.
- L3. The 8-byte big-endian prefix is stated. The Python cross-check is recorded in verification.md.
- L4. The boundary check is stated as defence in depth, and a literal test reaches it through the API.
- L5. Copy tests read components that have dependencies, through both `Component` and `ByName`.
- L6. The property model compares the problem multiset, adds revocation, and recomputes the bundle with its own encoder.
- L7. Withdrawn proposals are deleted at once, as in the siblings. A stray `Decision` on a component change is refused.
- The out-of-scope check was run in the root session: `git status` and `git diff origin/main` show no change under `modules/components`.
- Mutation survivors were closed with edge tests for text and character boundaries, the first character of hex and source, and duplicate dependencies and components.
