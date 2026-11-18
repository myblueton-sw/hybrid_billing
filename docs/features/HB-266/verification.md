---
type: VerificationRecord
title: HB-266 verification of the M03 access API
description: Module and apps/api tests, coverage, mutation and build results for the read-only grant and delegation listing, Explain, the access read and preview, and the grant and revoke commands. Run on mac_mini on 2026-11-16.
status: stable
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

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check. Rows marked "(classified by HYB-301)" replace rows that HYB-291 left unclassified; the [HYB-301 trace classification](../../reviews/HYB-301-trace-classification/report.md) gives the evidence.

## Requirement trace

Source: `docs/ui/screens/M03/spec.md:133-159` (HB-266 is the API, HB-267 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-03 | deferred | Addressed within pre-launch scope for grants, delegation revoke and the effect preview, with self-grant refused; registry-wide and leakage acceptance not built. | HB-266/plan.md |
| HB-04 | not connected | (classified by HYB-301) M03's part of HB-04 is keeping the setup role apart from export write and collector read (`requirements.json` HB-04 implementation_proposal) and testing operational reads after setup is revoked. Built: grants with a validity window and revoke (HB-266/plan.md:25, 119); the only grantable pair is `view focus_data rows` (HB-266/plan.md:106). Not built: a setup permission. It needs provider onboarding (S02) and collectors; S02 is outside the Go-bound pre-launch scope and served ungated on 8793 (HYB-291 M9). | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| HB-23 | deferred | Grant changes apply directly, without the approval request `M03/spec.md:83` requires. 2026-11-18: the approval request was built by HYB-298 (PR #212). | Planning cross-check residuals below |
| HB-33 | deferred | Grant commands use the common command path with idempotency keys and expected revisions, but after a partial failure a second submit can create a second grant. | HB-267/verification.md:42 |
| HB-36 | deferred | Support sessions are not available. | HB-266/plan.md:153-161 |
| HB-42 | deferred | The revocation states received, deny confirmed and cleanup running (`M03/spec.md:84, 109-113`) do not exist. 2026-11-18: built by HYB-298 (PR #212); received, deny confirmed and cleanup running pass inside one command. | Planning cross-check residuals below |
| AUD-AR-09 | deferred | Delegation creation and support sessions are not available; ended delegations are listed by ID only. | HB-266/plan.md:153-161, HB-267/verification.md Residuals |
| AUD-DB-08 | deferred | (classified by HYB-301) Built: grants expire and can be revoked (HB-266/plan.md:25, 119). Not connected: the CSP-side setup permission with a revocation condition and operational-read-only access after it is revoked (external). Not built: task-scoped expiring handoff links (`M03/spec.md:143-149`), which wait on binding S02 provider onboarding (owner decision; S02 is outside the Go-bound pre-launch scope). | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| AUD-QA-04 | deferred | Idempotency keys and expected revisions are enforced; OAuth client credentials, PKCE and the client-user intersection are not built. | HB-266/plan.md |
| IAM-01–IAM-08 | deferred | Role and revocation paths exist for local users only; approval request, revocation states, bulk paths and federated checks are not built. 2026-11-18: approval request and revocation states built by HYB-298 (PR #212), bulk revocation by HYB-299 (PR #218); federated checks remain open. | Planning cross-check residuals below |

## Planning cross-check residuals (HYB-291)

- **No approval request.** `M03/spec.md:83` requires grant changes to go through an approval request at a fixed revision; HB-266 applies them directly. (2026-11-18: built by HYB-298, PR #212.)
- **No revocation states.** The received, deny confirmed and cleanup running states (`M03/spec.md:84, 109-113`) do not exist; a revoke is a single command. (2026-11-18: built by HYB-298, PR #212.)
