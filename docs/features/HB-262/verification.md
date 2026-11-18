---
type: VerificationRecord
title: HB-262 verification of the installation administration action
description: Tests, coverage, mutation and build results for the administer_installation action in modules/auth and modules/gate and its sample seed in apps/api, run on mac_mini on 2026-11-16.
status: stable
---

# Scope

This record verifies [HB-262](https://linear.app/hybrid-billing/issue/HB-262) against its [plan](plan.md). The design review passed in round 2. Every result below was run on mac_mini on 2026-11-16 with Go 1.25.7.

# Changes

**`modules/auth`**
- New action `administer_installation`, with the fixed kind `installation` and the predicate `installationAction`.
- `SetInstallationWorkspace` is set once, under the registry lock.
- Grant validation requires a subject source, the installation workspace, full scope, any period, no fields, and a human subject.
- Delegations of the action are refused.
- The action has its own decision branch, `administer`. It sits after `current()` and before the approval and support-session branches. It uses direct grants only and refuses with `installation_scope`; a client without a grant is refused with `client_not_allowed`.

**`modules/gate`**
- Re-exports the action, the kind and the reason.
- `Administration.SetInstallationWorkspace`.
- `run` refuses the action unless the confirmed Answer is a human administrator. `run` now takes the request instead of the workspace.

**`apps/api`**
- The seed's first step, in both seed modes, sets the installation workspace to the configured bootstrap workspace.
- The sample seed grants the action to `sample-admin` and the console client.
- The catalogue uses `gate.InstallationKind`.
- The expected seed step counts in tests change from 6 to 8 and from 7 to 9.

# Results

| Check | Result |
| --- | --- |
| `go test -race ./...` in `modules` | all packages pass |
| `go test -race ./...` in `apps/api` | pass |
| `-race -count=40` on the new auth and gate tests; `-count=10` on the new apps/api tests | pass |
| `-race -count=10` on the auth installation tests, including the model test (added after the change review) | pass |
| `go vet`, `gofmt -l` | clean |
| `GOOS=linux GOARCH=386` and `GOOS=windows` builds of `modules` and `apps/api` | pass |
| Coverage of changed functions (`-coverpkg=./auth,./gate` over auth, gate and commands) | 100% |
| Coverage of `apps/api/server` | 100% |

`Registry.Describes` is at 85.7%. The uncovered statement is its existing nil-receiver branch, which this change did not touch.

## Coverage of the plan's cases

**auth: `installation_test.go`**
- set-once workspace;
- description by kind;
- grants before the workspace is set, and without a source;
- each refused grant shape;
- a refused delegation;
- decisions:
  - through the console;
  - without a client;
  - for another workspace;
  - with no grant;
  - for a client without a grant, an unknown client, or a suspended client;
  - in a support session;
  - with another kind or with fields (`invalid_request`);
  - for a service subject, a subject that became a service, a suspended subject, or a departed member;
- a recheck before and after a change;
- delegations ignored;
- a forged grant in another workspace refused by the workspace check;
- a concurrent set and decide under `-race`;
- `TestInstallationDecisionsMatchAModel`, a seeded random test of 2,000 cases against an independent model. Each case draws:
  - the installation workspace (unset, `w1` or `w2`);
  - the subject (human or service);
  - subject and client grants in either workspace, with grant acceptance asserted against the model;
  - a departure or suspension;
  - the client (none, granted or not granted);
  - a support session;
  - the request workspace.

  It checks the reason of each decision, rechecks each allowed decision, and requires every outcome to be reached at least 20 times.

**gate: `installation_test.go`**
- the role check: the bootstrap administrator is allowed and a member with a grant is refused;
- a support session;
- another workspace (`workspace_mismatch`, from the gate's workspace check, before auth);
- demotion:
  - a recheck of the earlier stamp gives `identity_not_current`;
  - a fresh Answer gives `installation_scope`;
- an invitation into another workspace fails.

**apps/api: `installation_test.go`**
- the installation workspace is the configured workspace and cannot change, with both seeds;
- `sample-admin` is allowed through the console client, and `sample-admin-2` (an administrator without a grant) is refused with `installation_scope`.

# Mutation testing

The changed lines were mutated: comparison and boolean swaps, error and reason swaps, removed negations, disabled `if` conditions, and empty-string changes. The script is `scratchpad/mutate262.py`, run in four copies.

The first run killed 91 of 99 mutants. Six of the eight survivors were resolved in code or tests:
- The `!= ""` clause in `inInstallation` was redundant, because workspace IDs are already validated as non-empty. It was removed.
- The `client.version == 0` clause in `administer` was redundant, because an unknown client has a zero state. It was removed.
- The workspace check in `administer` is defence in depth beyond grant validation. A test with a forged grant now kills it.
- Three mutants on the existing delegation validation line, which this change touched, survived. They concerned an invalid delegation ID and an invalid validity, and a new test kills them.

The final run killed 92 of 94. Both survivors are equivalent:
- `registry.go`, `!known` in `AddGrant`, an existing condition: an unknown subject has a zero state, which already fails `subject.State != Active`.
- `decide.go`, the author `""` passed to `stamp` in `administer`: decide and recheck compute the stamp digest with the same constant, so no observable decision changes.

# Change review

After the round 1 fixes, these were re-run on 2026-11-16 and passed:
- `go vet` and `go test -race ./...` in `modules` and `apps/api`;
- coverage of the changed functions (100%) and of `apps/api/server` (100%);
- the `linux/386` builds.

The mutation counts above are from before the round 1 fixes. Those fixes only add tests.

**Round 1: FAIL.** No path to the action was found outside the installation workspace or without the administrator role.

| Finding | Severity | Resolution |
| --- | --- | --- |
| M1: the random model test planned in the plan was missing | Medium | Added: `TestInstallationDecisionsMatchAModel`. |
| L1: the delegation refusal passed through `holdsInWorkspace`, not the new clause | Low | bob now holds a grant in w1, so the refusal isolates the new clause. |
| L2: the plan expected `installation_scope` for a suspended subject | Low | The plan now says `principal_not_current`, and a grant to a suspended subject is now tested. |
| L3: the armed bootstrap workspace was not asserted directly | Low | Both seed modes now assert `SetInstallationWorkspace(bootWorkspace)` returns nil. |

**Round 2: PASS.**
- N1, the model's fixed inputs, is covered by the literal tests.
- N2, a wording point, is corrected above.
- N3: the re-run is recorded above.

# Not verified

- No endpoint uses the action yet; HB-263 will.
- The gate role check's `answer.Kind() != identity.Human` clause is redundant with the role clause beside it, because a service Answer has the service role, and with auth's `activeHuman`, so no test isolates it. It is kept as defence in depth, and no surviving mutant was reported for that line because the mutation set disables only the first disjunct.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check.

## Requirement trace

Source: No screen spec. IDs derived from the phase 3 scope in `docs/features/HB-257/plan.md`. Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-03 | deferred | Addressed within pre-launch scope for the installation administration action, valid only in the once-set installation workspace; registry-wide and leakage acceptance not built. | HB-262/plan.md |
| HB-36 | deferred | Covered: installation administration is separated from workspace roles. Not covered: installation operation and support paths (P01). | HB-262/plan.md |
