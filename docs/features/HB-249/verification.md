---
type: VerificationRecord
title: HB-249 demo isolation verification
description: Actual checks run for W-38 increment 2 (modules/sandbox), plus review outcomes and residuals.
status: stable
---

# Scope

[HB-249](https://linear.app/hybrid-billing/issue/HB-249) adds `modules/sandbox`, following [plan.md](plan.md):
- **Environments.** Demo and production environments are registered under dual control, with namespaces that are disjoint across all environments: database, key ring, evidence path by segments, and number prefix. Demo prefixes start with `DEMO`, and a demo needs a registered production environment first.
- **Destinations and egress.** Mock destinations use reserved names only (`.test`, `.invalid`, `.example`), never loopback. They are unique register-wide, and can be disabled at once. The `Allow` egress choke point limits demo to matching enabled mocks and refuses mocks from production.
- **Configuration.** A mock-only typed configuration schema has no free-text, secret or secret-store kind. It refuses sensitive and connection-path keys, and its enum values cannot be host names.
- **Reset.** Resets are confined to their own demo environment's namespaces. IDs are allocated by the register, optional request keys make the start idempotent, each target has a completion flag, and old resets are evicted below a watermark.
- **Admission and labels.** `Admit` refuses cross-environment inputs, non-real origins in production and foreign numbers. `Label` returns the DEMO marking with the origin and source.

Base: main `b91a135`. Machine: mac_mini.

# Checks run (2026-11-15, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/sandbox/` | clean |
| `go test -race -count=10 ./modules/sandbox/` | pass, including concurrent calls of every operation |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/sandbox/` | pass |
| `go test -race -count=1 ./...` | every package passes |
| Statement coverage | 100.0% |
| Property test | 150 seeds against an independent model of namespace registration, destination refusal reasons, `Allow`, multi-target reset confinement and `Admit` with numbers. Every listed outcome is reached; one measured run is in the test log |

# Mutation testing

| Run | Mutants | Killed | Did not build | Survived |
| --- | --- | --- | --- | --- |
| First | 545 | 488 | 36 | 21 |
| Final tests | 545 | 506 | 36 | 3 |

Boundary tests for the identifier, destination ID, number prefix and host grammars, the latest reset record and label origins killed 18 survivors. The 3 remaining are equivalent:
- the encoder's length field of 9 bytes instead of 8, which only changes the internal proposal identity code consistently;
- `Round(0)` → `Round(1)`;
- the enum loop starting at 1, because the first character is already checked as a letter.

# Reviews

- **Design review** (independent reviewer, read-only):
  - round 1 FAIL: H1 the environment ID prefix, H2 database names, H3 loopback, and M1–M4;
  - round 2 FAIL: N-M1 the reset watermark;
  - round 3 PASS, with lows folded in.
- **Change review** (a separate independent reviewer, read-only):
  - round 1 FAIL: M1 and M2 were test gaps, with no isolation hole found;
  - round 2 PASS.

  Details are in plan.md.

# Residuals and exposure blockers

- The Assistant conversation retention class of QA-10, planned for W-43.
- **Wiring callers.**
  - The egress layer, configuration writers, reset tooling and the issuance entry calling this register.
  - Output writers applying labels.
  - Feeding environment classes into `modules/simulation` proof.
- **Deployment preconditions.**
  - The demo runs in its own network namespace, where reserved names resolve only to mock containers.
  - Target actions are idempotent.
- **Undetectable here.**
  - Sockets, resolvers and symlinks.
  - Unregistered real systems.
  - Relabelled provenance.
  - Replicas at admission.
- Packaging (`compose.demo.yaml` and the demo commands), persistence and screens.
- SIM08 and AUD-QA-10 system-level cases are NOT_RUN.
- Owner decisions not confirmed by the owner:
  - the reserved suffixes;
  - the `DEMO` prefix rule;
  - the refused key fragments;
  - the destination kinds;
  - the bounds.
