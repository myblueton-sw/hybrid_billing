---
type: VerificationRecord
title: HB-266 verification of the M03 access API
description: Module and apps/api tests, coverage, mutation and build results for the read-only grant and delegation listing, Explain, the access read and preview, and the grant and revoke commands. Run on mac_mini on 2026-11-16.
status: final
---

# Scope

This record verifies [HB-266](https://linear.app/hybrid-billing/issue/HB-266) against its [plan](plan.md). The design review passed in round 2. Every check below was run on mac_mini with Go 1.25.7.

# Results

| Check | Result |
| --- | --- |
| `go test -race ./...` in `modules` and `apps/api` | pass |
| Coverage of `apps/api/server` | 100.0% |
| Coverage of the new module code (`listing.go` and the gate pass-throughs) | 100.0% |
| `GOOS=linux GOARCH=386` and `GOOS=windows` builds | pass |
| Mutation testing (`access.go`, `auth/listing.go`, `gate/gate.go`) | first run 147 of 162 killed; survivors resolved below |

## Module tests

**`auth/listing_test.go`**
- Grant states: active, revoked, pending, client and field grants.
- Sorted order, and copies that do not alias registry data.
- Delegation states.
- `Effective` turns false when a membership is recreated.
- The selector and period accessors.
- `Explain` matches `Decide` for allowed, out-of-scope, delegated, no-grant and unknown subjects. It changes no generation. It refuses installation and human-only actions, including for an actual installation-grant holder. A suspended subject and a generation change mid-call are both refused.
- `Explain` racing `AddGrant` and `RevokeGrant` under `-race`.

**`gate`**
- The zero administration.
- Listing pass-through, and `Explain` with and without a grant.
- The source test still shows that only gate imports auth.

## apps/api tests (`access_test.go`)

**Read**
- Installation grants are hidden.
- The reader grants appear with their state, `effective` flag and scope.
- The catalogue is the seeded console ceiling, ending at the `reader-console` end date.
- The subjects are the active members.
- No session is refused, and so is a member.

**Grant and revoke through the console**
- A member is refused on the collected-data endpoint.
- After a grant, the member can read through the console. No client grant is added.
- After the grant is revoked, the member is refused again, and a second revoke gives `invalid_stage`.

**Grant validation**
- A self-grant is refused.
- These inputs are refused, each naming its field: an unknown subject; an action, kind or fields outside the ceiling (including the `installation` kind); empty fields; a customer-scoped `focus_data` grant; a malformed customer; `days` of 0 or 31.
- Exactly 30 days is allowed.
- The declared cross-grant to another administrator is allowed.
- A rationale over 256 bytes is refused, and so is a wrong target or workspace.

**Revoke rules**
- Refused: installation grants, client grants and the caller's own grant.
- An unknown ID gives `permission_denied`.
- Another holder's grant can be revoked.
- A delegation is listed, revoked, and revoked twice (`invalid_stage`). An unknown delegation gives `permission_denied`.

**Preview**
- Decisions are given per action and per field.
- Before and after a grant; `rows` is allowed while the unrelated `state` field is refused.
- A subject outside the set is refused, as are the `installation` kind and a malformed customer.
- With a cost field configured, the preview includes `view_cost`.

**Ceiling and grantees**
- A client grant narrowed by customer, legal entity, contract or period never counts as ceiling.
- A suspended user and a suspended membership are not grantees.
- OpenAPI lists the required fields of `access.grant`.

**Failure paths**
- A module refusal maps to `prerequisite_failed` (fault hook).
- Roster failures in the read, the preview and the grant command.
- An unavailable administration on revoke.
- A grant scoped to a customer on a kind with a full ceiling succeeds.

**Other**
- States and error mapping.
- Concurrent reads, previews and grants under `-race`.

## Mutation testing

**Resolved:**
- The explicit `Closed` checks in revoke were redundant with the module's `ErrConflict`, and are removed.
- The `fullScope` and grantee conditions are now killed by the narrow-ceiling and suspended-state test.
- The `Required` flags of the grant fields are killed by the OpenAPI check.
- The installation check in `Explain` is killed by the holder test.

**Equivalent survivors:** three `sort` comparisons that change `<` to `<=`. The IDs are unique, so the order is the same.

# Change review

**Round 1: PASS** (only Low findings).
- Applied:
  - L1: the test fault now replaces the module call instead of overwriting its result.
  - L3: the latest covering ceiling sets the day limit.
  - L4: duplicate or empty fields are refused, naming the field `fields`, with tests.
  - L5: the unused parameter is removed.
  - L7: the concurrent test now asserts its responses.
- Recorded as residuals for HB-267:
  - L2: delegations have no `effective` flag.
  - L6: the preview shows `download` for general fields only.

# Not verified

- **The M03 screen.** It comes in HB-267.
- **Delegation creation and support sessions.** They are residuals.
