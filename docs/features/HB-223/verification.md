---
type: VerificationRecord
title: Protected payment, recipient, tax identifier and template change verification
description: Records what was executed and reviewed for the QA-01 increment of work item W-23, the register in which changes to payment accounts, document recipients, tax identifiers and templates need existing-channel confirmation by a separate human, separate approval bound to that confirmation, and apply only from a forward effective time, with results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-223](https://linear.app/hybrid-billing/issue/HB-223) adds the Go package `modules/protected` from main `ee60476`. Its non-test sources use only the standard library. A registry with an injected clock and one mutex keeps, per workspace:

- **Rules under dual control.** `Propose`, `Confirm` and `Discard` set the number of approvers, the confirmation validity, the maximum lead and the minimum channel age. Every proposal takes a new revision, the proposer cannot confirm, and the history of every proposal is readable. Proposals are capped at 1,000 per workspace.
- **Channels under dual control.** `Register` and `Admit` (by a different human) record a subject's contact channel with evidence; `Retire` ends it. The registrar and admitter are custodians and are excluded from requesting, confirming or approving through that channel.
- **Requests over protected references.** A change names the subject, field, new reference (vault identifier and fingerprint), provenance, instruction (kind, channel, sender fingerprint, evidence) and effective time. Assistant-generated provenance is refused (`llm_generated`). An instruction from an address matching a known channel must name it. One open change per subject and field; effective times are forward-only and strictly increasing.
- **Existing-channel confirmation.** A human other than the requester confirms through an admitted, unretired channel of the subject that is older than the channel age and is neither the instruction's channel nor the sender's address. Denial rejects the change. The confirmation digest binds verifier, channel, evidence, outcome and time to the request digest.
- **Separate approval.** One to three humans who are not the requester, verifier or channel custodians approve the confirmation digest. A rejection ends the change.
- **Lapse, withdrawal and resolution.** Open changes lapse at their effective time or when the confirmation expires. A scheduled change can be withdrawn only before it takes effect and when no later change depends on it. `Resolve` returns the reference in force at a given time, so documents dated before an effective time keep the earlier reference.
- **Actors.** Every write needs a human; an assistant gets `ai_not_allowed` and a system `not_human`, including on repeats.

# Commands and results

All runs were made in the worktree `/private/tmp/hb223` on 2026-11-14 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/protected` | no output |
| `go vet ./modules/protected` | pass |
| `go test -race -count=10 ./modules/protected/` | pass (23.6 s) |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null` in the package | builds |
| `go test -race -count=1 ./...` | pass, 37 packages with tests |
| statement coverage of `modules/protected` | 100.0% |
| property test against an independent state-machine model | 600 seeds × 100 steps pass; every state reached at least 20 times |
| mutation testing (`mutate223.py`, 414 mutants) | 391 killed, 21 did not build, 2 survived (both equivalent) |

The property model is a separate implementation of the request, confirmation, approval, withdrawal, lapse and resolution rules. It covers human, assistant and system actors, wrong digests, denied confirmations, a retired channel and two subjects and fields, and compares every refusal, the state, `Before`, `Ended`, `EndedBy`, `Scheduled` and `Resolve` at four offsets after each step. Channel admission and age are covered by example tests only.

The two surviving mutants replace `Round(0)` with `Round(1)` on the clock reading and on the effective time. Both strip the monotonic reading and leave a nanosecond-precision time unchanged, so they are equivalent. The 21 mutants that did not build were rejected by the compiler or `go vet` (duplicate map keys, duplicate switch cases and character literals such as `'10'`).

# Acceptance AUD-QA-01

| Criterion | Evidence |
| --- | --- |
| An account change from a new email alone is refused | `TestNewEmailAloneNeverApplies`: approval before confirmation is `not_confirmed`, the change lapses and never resolves |
| An LLM-generated account is refused | `TestLLMGeneratedAccountRefused`: `assistant_generated` provenance and assistant or system actors are refused |
| Existing-channel confirmation | `TestConfirmationChannelControls`, `TestSenderFingerprintMatchesChannel`, `TestVerifyRefusalPairs`, `TestChannelAgeBoundary` |
| Separate approval | `TestApprovalSeparation`, `TestApproveRefusalPairs`, `TestRetiredChannelBlocksApproval` |
| Applied only from the effective date | `TestConfirmedAndApprovedChangeAppliesFromEffectiveOnly`: the old reference resolves 1 ns before the effective time and the new one from it |

# Reviews

- Design review: three rounds by an independent read-only reviewer: CONDITIONAL PASS (3 high, 6 medium, 4 low), CONDITIONAL PASS (3 medium, 4 low), then PASS with one low finding applied. See the plan's review history.
- Change review: two rounds by a separate independent read-only reviewer: PASS with 2 medium and 7 low findings, all fixed; then PASS with one low finding, applied.

# Provisional choices

These are mine, not the owner's, and need owner confirmation:

- the rule ranges (1–3 approvers, 1 hour–30 days validity, 1 hour–366 days lead, 1–90 days channel age) and the 1,000-proposal cap;
- the field, provenance, channel and instruction kinds and the rejection reasons;
- reading QA-01's validity period as how long a confirmation stays usable for approval;
- the separation sets for requester, verifier, approvers and channel custodians;
- any human may withdraw or discard; a request snapshots the rules in force;
- a retired confirming channel blocks further approvals without ending the change.

# Not verified or not built

- The integration layer: deriving provenance and the sender fingerprint from system metadata, `modules/auth` permissions, storing the resolved change on each document and converting document dates.
- The vault holding protected values, persistence, screens O03, I01 and B09, and DBA and QA sign-off.
- The ERP recognition cadence (ERPC-01 to ERPC-08), which remains in W-23.
