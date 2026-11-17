---
type: VerificationRecord
title: HB-265 verification of the M02 authentication policy API and live screen
description: Go tests, coverage and mutation results for the policy read, the three policy commands and the maximum-age check on every request, plus console unit tests and browser QA on the production build against a live apps/api. Run on mac_mini on 2026-11-16.
status: final
---

# Scope

This record verifies [HB-265](https://linear.app/hybrid-billing/issue/HB-265) against its [plan](plan.md). The design review passed in round 4.

The run used:
- Go 1.25.7;
- Node with `tsx --test`;
- Playwright with chrome-headless-shell 1243, on macOS arm64;
- the console as a Vite preview of the production build on port 4175, proxying `/api/v1` to an `apps/api` built from this branch with `--seed sample`.

# Results

| Check | Result |
| --- | --- |
| `go test -race ./...` in `apps/api` | pass |
| Coverage of `apps/api/server` | 100.0% of statements |
| `GOOS=linux GOARCH=386` and `GOOS=windows` builds | pass |
| `go vet`, `gofmt -l` | clean |
| Mutation testing of `policy.go`, `commands.go`, `seed.go` and `session.go` | final run 116 of 118 killed; the 2 survivors are equivalent |
| `tsc --noEmit`, `npm run build` | pass |
| `npm test` | 349 pass, 0 fail |
| Browser QA (`scratchpad/qa265.mjs`) | 38 PASS, 0 FAIL, no console errors (against an API rebuilt after the review fixes) |

## API tests (`policy_test.go` and updated HB-263 tests)

**Read**
- The read shape covers:
  - the seed policy, the proposal record and its rationale;
  - `now`, `signed_in_at` and `act_until`;
  - `null` policy when none exists (`--seed=none`).
- Admission: no session, bad CSRF, and a member.

**Propose, approve and withdraw**
- Propose, approve by the other holder, and withdraw by the proposer.
- Refused: self-approval, and a withdraw by another administrator.
- Approving discards the other pending proposals; an applied proposal cannot be approved by another account.
- The proposer and approver flags are correct, and `act_until` is limited by the current window.
- Records of withdrawn proposals are pruned at the next change.

**Field validation**
- `max_age_minutes` and `step_up_minutes` boundaries: -1, 0, 4, 5, 720, 721, an overflowing value, and the step equal to the maximum age.
- Malformed targets are refused: "1", "12", "proposal-", "proposal-01", "proposal-x", unknown and unrecorded numbers, and overflow.
- A rationale over 256 bytes is refused.

**Freshness and lockout**
- A step-up beyond 10 minutes is refused until the caller signs in again.
- `approver_locked_out`.
- The maximum age ends sessions on every path: `current`, a reader, a command, `listUsers` and `collect`. A fresh sign-in passes. A future sign-in time also ends the session.
- A wall-clock step back larger than the maximum age refuses sign-in (the declared residual).

**Actor and error mapping**
- A missing session, an ended identity session, and an identity session without a table entry all give `permission_denied`.
- Error mapping.
- `window` with another action's rule.
- The 200-record cap, with no duplicate records.

**Trust boundary and structure**
- The trust boundary is pinned: the second holder can suspend the first.
- OpenAPI's `required` list for `policy.propose`.
- Only `signIn` opens production sessions (source test).
- Concurrent policy reads with dead records, alongside commands, pass under `-race`.

**HB-263 tests updated for the second holder.** Where the old assumption applied, the tests revoke the seeded grant with `withoutSecondHolder`. The protection unit cases now cover "the other holder while both hold the grant".

## Mutation testing

The first run killed 111 of 125.

**Removed as redundant:**
- the `ChangeRevision` kind check in the read (only revision proposals are recorded);
- the 19-digit bound (`ParseUint` already rejects overflow);
- the `ok` result of `proposalNumber` (an unparsable target maps to number 0, which is never recorded).

**Killed by new tests:**
- the workspace check in `Resolve`;
- an unrecorded proposal created directly in the module;
- a session missing from the table while its identity session is alive;
- a `window` with a non-policy rule after the policy rule;
- a dead record that must not be listed;
- the OpenAPI `required` fields.

The final run killed 116 of 118. The two survivors are in `policyPage`:
- `limit < act` becomes `<=`: equal values give the same result.
- `limit >= 0` becomes `> 0`: the limit is either −1 (no policy) or at least 5 minutes, never 0.

## Browser QA cases

**Policies view**
- The live phase.
- The current facts in words: "Password only", "720 minutes (12 hours)", and the approver.
- The propose form in the panel, with form validation.

**Propose**
- Propose 60/10 with a reason.
- The proposal lists "(changed)" in words.
- The proposer sees no Approve button, only the "another administrator" note.
- A second proposal.
- The proposal block has a single neutral 1px border on its block-start edge.

**Approve, as `sample-admin-2` in another browser session**
- The approve step has focus, and shows the discard warning for proposal 3.
- Cancel returns focus to the Approve button.
- Approve applies the policy ("60 minutes (1 hour)", "Within 10 minutes"), and proposal 3 disappears.

**Withdraw**
- After a reload, propose and withdraw.

**Other views**
- Providers: four methods.
- Test, keys and sync: unavailable.
- Recovery: unavailable, and "Go to Users and access" opens M01.

**Access and layout**
- A member gets the denied state, with no buttons.
- EN, KO and AR at 1440 and 390:
  - `dir` is correct, with no horizontal scroll;
  - the propose step at 390;
  - Back returns focus to the Propose button.

Renders are in `scratchpad/hb265qa/`.

# Review rounds

**Change review, round 1: FAIL.**

| Finding | Severity | Resolution |
| --- | --- | --- |
| M1: focus was lost after a proposal at narrow width | Medium | The focus target is now applied after the reloaded page renders, falling back to the pending heading. QA checks that focus lands on the new proposal and that the form is cleared. |
| L1: duplicate propose and repeated approve reported success | Low | A duplicate proposal returns `invalid_stage`; approve requires a pending proposal; the form is cleared after success. |
| L2: missing tests | Low | Added: the takeover route, and concurrent stale requests on all paths under `-race`. The leftover import was removed. |
| L3: Arabic gender agreement | Low | The step-up line uses the masculine form, with a unit test. |

**Design comparison, round 1: FAIL.**

| Finding | Severity | Resolution |
| --- | --- | --- |
| H1: the comparison was clipped in the approve step | High | The approve and withdraw steps use a stacked comparison. QA checks that it is not clipped at 1440 or 390. |
| M1: the approver was unlabelled in mobile history | Medium | Separate "Proposed by" and "Approved by" columns, each labelled on mobile. |
| M2: missing renders | Medium | Added: pending and approve at 390, the withdraw step, and the stale state at 1440 and 390 (stubbed read). Loading and error are covered by unit tests only. |
| L1–L4 | Low | Regular-weight radio labels; a gap under step headings; history column widths; the values-match note moved under the fields and shown only after an edit. |

**Change review, round 2: PASS.** N1 is fixed: a failed reload clears the pending focus target.

**Design comparison, round 2: PASS.**
- L5: the proposal cell no longer wraps (12%, `nowrap`).
- L6: the reference URLs and the module attribution are recorded in the plan.

**Still not rendered:** loading, load error and denied states; the approve and withdraw steps in KO and AR; the stale state in KO and AR; a refusal inside the approve step. These are covered by unit tests only.

# Not verified

- **Stale sign-in state against the real server time.** The browser renders it with a stubbed read (`now` after `act_until`) at 1440 and 390: approve, withdraw and the form are replaced by the re-sign-in guidance. The server behaviour is covered by `TestStepUpAndApproverLockout`.
- **Copy.** All copy is a draft for the owner's language and terminology review.
- **Owner FIX** of M02.
- **Console component tests.** As in HB-264, focus restoration is covered only by browser QA.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check.

## Requirement trace

Source: `docs/ui/screens/M02/spec.md:24-26`. Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-03 | deferred | Addressed within pre-launch scope for two-person proposal and approval of the authentication policy; registry-wide and leakage acceptance not built. | HB-265/plan.md |
| HB-36 | deferred | Providers, test login, keys, sync and recovery are shown as not available in pre-launch. | HB-265/plan.md:177-178, :200-208 |
| HB-42 | deferred | The draft, validated, independently approved, activated trust flow is collapsed into propose then approve-applies. | Planning cross-check residuals below |
| IAM-01 | unclassified – owner review | M02 adds no creation or invitation path; the spec links IAM-01 for the whole range. | this section |
| IAM-02 | deferred | Only local password accounts exist; SAML, OIDC and LDAP sign-in are residuals (HB-257 out of scope). | HB-265/plan.md:177 |
| IAM-03 | unclassified – owner review | M02 has no identity-provider connection that could recreate an account. | this section |
| IAM-04 | deferred | Federated token checks are not built. | HB-265/plan.md:177 |
| IAM-05 | unclassified – owner review | M02 has no suspension or departure path. | this section |
| IAM-06 | deferred | MFA and recovery settings are not available. | HB-265/plan.md:200-208 |
| IAM-07 | deferred | Directory sync is not available. | HB-265/plan.md:177-178 |
| IAM-08 | deferred | Maximum age is enforced and policy changes need a fresh sign-in; MFA evidence per method is not built; HYB-293 covers step-up on other commands. | HB-265/verification.md |

## Planning cross-check residuals (HYB-291)

- **Trust flow simplified.** `M02/spec.md:96-131` specifies draft, validated, independently approved and activated stages. HB-265 collapses them into propose, then approve, which applies. This was not recorded before.
