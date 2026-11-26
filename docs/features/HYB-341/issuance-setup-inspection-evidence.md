---
type: VerificationRecord
title: Separate authority for issuance setup inspection
description: Scoped implementation and logic evidence for protected-reference and payment-term inspection prerequisites.
status: verified
---

# Scope

HYB-341, machine:mac_mini. Root implemented separate Inspect capabilities in the existing protected and terms authority domains. Each requires its own governed human grant and console ceiling, exact resource kind and Cost evidence field. Neither administrator status nor a write grant supplies Inspect. The change creates no default grants or public setup endpoint and does not complete the O03 operator journey.

Existing write mappings, domain versions and grant prefixes remain unchanged. New evidence fields cannot be downgraded through the caller catalog or granted through generic permission commands. Private read callers retain current session, identity, gate, nondelegation and owned-grant checks; they skip only the installation-hold mutation refusal. No effect reconciliation occurs in these callers.

# Review and logic evidence

The independent refusal_review reviewer inspected the capability design and resulting production diff and returned scoped PASS. Required corrections identified in design review were included before that source review: forwarding exact fields through Actor/Caller checks, forcing Cost classification after catalog overrides, generic field exclusions and separate held-read wrappers. The reviewer did not execute tests or approve an O03 disclosure DTO. Passed production scope was not sent through repeated reviews.

Executed checks:

- `go test ./apps/api/server -run '^(Test(Term|Protected)Authority|TestFinancialSetup)' -count=1`: PASS, 3.186 seconds at the initial implementation.
- `go test -race ./apps/api/server -run '^(Test(Term|Protected)Authority|TestFinancialSetup|TestFinancialIssuanceTerm|TestFinancialIssuanceProtected)' -count=1`: PASS, 69.860 seconds, including uncertain grant recovery and mutation-only refusal.
- `go test -race ./apps/api/server -run '^TestFinancialSetup' -count=1`: PASS, 7.269 seconds after adding ineffective membership and actual delegated-inspection cases.
- `go test -race ./modules/internal/grantauthority ./modules/termsauthority ./modules/protectedauthority`: shared registry tests PASS, 1.320 seconds; the two facade packages have no package-local tests. Expansion regression compares every preexisting capability's proposal digest and grant effect identity using the legacy and expanded capability sets; legacy registries continue to reject Inspect.

Actual-grant cases cover independent approval, absence of seeded rights, separate capabilities, cross-domain refusal, generic field bypass rejection, catalog downgrade resistance, revoked human/ceiling grants, missing owned bindings/ceilings, ended sessions, actor/workspace mismatch, unknown effects before and after governed recovery, held reads and held write-wrapper refusal. Additional edge cases prove that generally gate-allowed delegation and active-but-ineffective prior-membership grants cannot satisfy owned inspection.

The first delegation edge run failed in fixture setup: AddDelegation requires the recipient to hold a direct workspace grant. The fixture now supplies an unrelated term mutation grant, then proves that inspection is available through delegation at the general gate and refused by the private inspection boundary. No production policy was weakened to make the fixture pass.

# Limits and next integration

These are synthetic in-memory logic tests. No new browser, screenshot or screen-verification run was performed; the owner performs screen verification and design re-review remains required for changed designs. Compatibility evidence covers existing grant identities, not interoperability of new Inspect records with old binaries. Exact contract-to-organization scope resolution, bounded setup disclosure, provisioning routes, evidence capture and the ordinary operator journey remain pending. Required public-contract review must cover those new boundaries separately.
