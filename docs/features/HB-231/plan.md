---
type: ImplementationPlan
title: Post-issue decreases that reference an immutable original, show the explicit difference, deduplicate by source key, bind approval to the base they were computed on and link exactly one matching credit
description: Defines the first increment of work item W-29, a correction ledger over issued documents from modules/issuance. A correction states a target amount for an original; the register computes the explicit difference against the effective amount computed from every issued credit, supports decreases only, serialises one open correction per original, deduplicates reprocessing of the same source key, binds approval to the request digest and base, applies only by linking one new issued credit whose parties, currency, scale and amount match, and lets a human explain credits issued outside the ledger. ERP final amounts are recorded and compared without overwriting billing.
status: draft
---

# Scope and ownership

[HB-231](https://linear.app/hybrid-billing/issue/HB-231) starts work item W-29 (requirement HB-29; the post-issue part of AR-07; HB-29-V) after HB-230 (`modules/issuance`). Base: main `0ab3797`. Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviews the plan and the change, because the register decides billed amounts after issue. Machine: mac_mini.

Out of scope, recorded as residuals: delivery attempts and channel capability gating (DB-05, the next W-29 increment), DB-07 (already delivered by HB-228 `modules/exports`), pre-issue adjustments (HB-222 `modules/adjustments`, where unissued retroactive changes become a new run), line-level and tax-level correction breakdowns, shared-pool propagation, generating the difference draft automatically in `modules/issuance`, feeding effective amounts to `modules/receivables` and `modules/erpledger`, persistence and screens.

# Observed baseline

- HB-29-D: the issued original is immutable; late sources and retroactive changes become difference documents that reference the original; correction types, cumulative credit limits.
- HB-29-I: Adjustment → original, line and cumulative correction, approval, issue; an issued bill is never issued again.
- HB-29-V: the same credit processed twice, partial correction limits, ERP final amount differences.
- AR-07: target-total input shown as an explicit target − current difference; reason, evidence, author and approver kept; source errors not hidden; after issue, corrections reference the original.
- `modules/issuance` issues credit documents whose `Original` names the credited document and enforces credit room; it has no debit kind that references an original.

# Decision

A new package `modules/corrections` follows the sibling conventions: injected clock, one mutex, workspace isolation, `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`), `ErrLimit`; check order form, lookup, repeat, rule, room; assistants refused everywhere (`ai_not_allowed`). `NewRegistry(Config{Documents}, Limits{Workspaces, Corrections, Records}, clock)`, where `Documents` has the `issuance.Registry.DocumentOf` signature and a new read-only `issuance.Registry.CreditsOf(workspace, original)` that lists, in recording order, every credit issued or ERP-recorded against an original (the order of `Issue`/`RecordExternal` calls, independent of ERP issue times). Both are called under this register's mutex; issuance never calls back. Allowed files: `modules/corrections/**`, the additive `CreditsOf` with tests in `modules/issuance`, `docs/features/HB-231/**`.

## Decreases only in this increment

`modules/issuance` lets only credit kinds reference an original, so an increase could not be tied to the original it corrects, and any later invoice of the same parties could be claimed as the difference. This increment therefore supports decreases only: a target above the current effective amount is refused (`debit_unsupported`). Debit (increase) documents that reference their original are a residual needing a reviewed issuance change.

## Originals and the effective amount

An original is an issued document that is not itself a credit (`not_original`). Its total, currency, scale, parties, contract, class, region, tax, statutory flag and `External` flag come from the snapshot, whose `Workspace` must equal the requested workspace. The effective amount is always computed from issuance, never from this ledger alone: the original total minus the totals of every credit `CreditsOf` lists, so credits issued directly in issuance or recorded from an ERP are never hidden. A credit that no applied correction links is an unlinked credit; the effective view lists them, and `Request` and `Approve` are refused while one exists (`unlinked_credit`). A missing document is `ErrNotFound`; any other lookup error, or a snapshot of another workspace, is `documents_unavailable`.

## Requests

`Request(workspace, Correction{ID, Original, Key, Target, Reason, Evidence, Period}, actor)` by a human. `Key` identifies the triggering source; `Target` is a canonical non-negative decimal with at most the original's scale (`scale_mismatch`); `Reason` is one of `late_source`, `retroactive_change`, `quantity_error`, `price_error`, `other`; `Evidence` is a reference identifier; `Period` `{From, To}` is the affected service period (calendar dates, From not after To).

- Repeat: the same identifier with the same content by the same human returns the view; otherwise `ErrConflict`.
- Deduplication: a request whose original and key match a correction that is not withdrawn returns that correction (`Duplicate` set on the returned view only) when target, reason, evidence and period are all equal, and is refused `key_reused` otherwise. Reprocessing the same source never creates a second correction.
- Rules: no unlinked credit; one open correction per original (`correction_pending`); `Target` below the effective amount (`no_change` when equal, `debit_unsupported` when above).
- The base is the effective amount, the number of credits and a digest of their identifiers. The record keeps the base, the signed difference, requester and time, and a digest (label `corrections_request_v1`) over the workspace and all of these.

## Approval and withdrawal

`Approve(workspace, id, digest, actor)` by a human other than the requester, on a requested correction whose digest matches (`digest_changed`), whose base still equals the issuance state (`base_changed`, for example after a direct credit) and with no unlinked credit. `Withdraw(workspace, id, actor)` by a human ends a requested or approved correction and frees its key; a requested correction can always be withdrawn, while an approved one is refused only while an unlinked credit exists (`credit_issued`), because that credit is probably this correction's and must be applied or explained first; once every outside credit is explained, the approved correction can be withdrawn.

`Explain(workspace, original, credit, Explanation{Reason, Evidence}, actor)` by a human records an unlinked credit (one issued directly in issuance or recorded from an ERP) as an explained ledger entry with a reason from the same closed set and evidence, together with the credit's issuance approver and issuer; the explainer must not be the credit's issuer (`self_explanation`). It then counts as linked. The same explanation by the same human returns it; a different explanation of the same credit is `ErrConflict`; a credit linked by a correction is `document_linked`; a credit that is not an unlinked credit of this original is `not_unlinked`. Any sequence can therefore reach a state where a new correction can be requested.

Repeats: the same approver approving the same digest again returns the view; applying the same document again by the same actor returns the view; any other repeat is `ErrConflict` or a state refusal (`not_requested`, `not_approved`, `closed`).

## Application

`Apply(workspace, id, document, actor)` by a human or system on an approved correction. The current credits must be exactly the base credits plus `document` (`base_changed` otherwise), so the document is new since the base and no other credit slipped in. The document must be a credit of this original, of the same class, region, tax, statutory flag, `External` flag, contract, customer, entity, currency and scale (`document_mismatch`), and its total must equal the absolute difference numerically at the original's scale (`amount_mismatch`). For ERP-recorded credits the ERP issue time is not compared with the local approval time; the base check alone proves the credit is new. The effective amount then equals the target. Because targets are non-negative and below the effective amount, the remaining credit room in issuance (original total minus issued credits) always covers the requested credit.

## Due dates and receivables

Credits carry no due date of their own (`not_applicable` in issuance); the original's issue date, original and effective due date are unchanged by a correction, and the correction date is never a new start date. Reducing the original's outstanding amount in `modules/receivables` by the applied credit is a residual (PT-03, PT-07).

## Effective view and ERP finals

`EffectiveOf(workspace, original)` returns the original total, the effective amount from issuance, the applied corrections, unlinked credits, any open correction and the ERP records. `RecordFinal(workspace, original, Final{Reference, Amount, Observed}, actor)` by a human or system records an ERP-reported final amount: a canonical non-negative decimal at most the original's scale in the original's currency, an observation time not in the future. The same reference with the same amount and time by the same actor returns the record; the same reference with anything else is `ErrConflict`. Each record is compared with the current effective amount when read (`matched` or `diverged`); nothing overwrites billing. Records are bounded per workspace.

# Residuals

Debit documents, line, tax and allocation breakdowns and revalidation of corrections (AR-07), RecognitionAdjustment and the ERP recognition cadence, receivables effects (PT-03, PT-07), shared-pool propagation, automatic difference drafts in issuance, delivery (next increment), persistence and screens.

# Verification plan

Literal tests for every refusal and repeat; a property test of random request, approve, withdraw and apply sequences against an independent model of effective amounts (never negative, equal to the last applied target, difference equal to target − base); duplicate reprocessing; `debit_unsupported`; direct and ERP credits issued at any step, with explanation always restoring progress; credit matching with real `modules/issuance` documents in an integration test; ERP divergence; `-race -count=10`, a 386 build, gofmt, vet, the whole repository with `-race`, 100% coverage and mutation testing.

# Review history

- Design review round 1: FAIL (3 blockers, 3 major, 3 minor). Folded in: decreases only, with debits a residual (blockers 1 and 3); effective amount and base computed from issuance credits through a new `CreditsOf`, unlinked credits refused and shown, `base_changed` on any credit change, withdrawal refused while an unexplained credit exists (blocker 2); full snapshot matching including workspace (4); due-date policy (5); ERP final repeat, form and actors (6); ERP issue time not compared, base check instead (7); `Period` (8); residual list and strict deduplication (9).
- Design review round 2: FAIL (1 major, 3 minor). Folded in: requested corrections can always be withdrawn and `Explain` records outside credits, so no original gets stuck (1); stale debit wording removed (2); repeat rules (3); recording order (4).
- Design review round 3: FAIL (1 major, 1 minor). Folded in: an approved correction is withdrawable once no unlinked credit remains (1); explanations keep the credit's approver and issuer and refuse its issuer as explainer (2).
- Design review round 4: PASS. Noted: explaining a correction's own credit and then withdrawing the correction records the decrease as an explained entry; the effective amount stays right because it always comes from issuance.

# Change review resolution

Change review round 1: PASS with low findings, all applied: `Apply` and `Withdraw` record the closer's actor kind and repeats compare it; credit totals must be canonical in the effective computation (a signed or malformed total from another `Documents` implementation is `documents_unavailable`); tests add reprocessing a key after application, a same-identifier other-kind repeat, a signed credit total, and property-test assertions of base and difference against an independent cents formatter. Accepted as stated: `Approve` repeats only while approved, and `RecordFinal` needs issuance to be reachable even for an exact repeat because it returns a live judgement.
