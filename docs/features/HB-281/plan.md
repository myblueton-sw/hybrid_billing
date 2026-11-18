---
type: ImplementationPlan
title: Live O03 contract and service lifecycle screen in the console service mode, bound to the HB-280 contracts API
description: Phase 4e of the pre-launch plan (HB-257). In service mode, O03 lists contracts with their parties, path, period, contract state and final settlement state, and shows a selected contract's parties, billing schedule, components, and termination and final settlement. Administrators suspend, resume, renew, end or withdraw lifecycle events and record evidence, settle or reopen, each with a reason and a confirm step, guided by the API's action hints. Money, proration, documents, payment terms and recognition are stated as not connected or not computed.
status: stable
---

# Scope and ownership

[HB-281](https://linear.app/hybrid-billing/issue/HB-281) binds O03 to the [HB-280](../HB-280/plan.md) API. Base: main `4331d3e`.

**Ownership**
- Root (Claude Code) owns the plan, implementation, tests and browser QA.
- The `ui-designer` role (AI, read-only) writes the visual specification and compares the final renders.
- A separate read-only reviewer reviews the design and the change. Focus: lifecycle and settlement states shown separately and correctly, dates (UTC, exclusive ends), no unknown value shown as zero, commands only offered when the API hint allows them, and refusals mapped.
- Machine: mac_mini.

**Allowed files**
- `apps/console/src/service/admin.ts` and its test: `contracts()`, `contract()` and the eight command IDs; new `apps/console/src/service/contracts.ts` (types and hardened parsers) with its test.
- `apps/console/src/modules/organizations/pages/contracts/`: new `Live.tsx`, `LiveDetail.tsx`, `LiveActions.tsx`, `live-copy.ts`, `live-model.ts`, `live-model.test.ts`, `live.css`. The existing preview `Screen.tsx` stays for the unconnected mode.
- `apps/console/src/app/screen-groups/work.tsx`: the live branch for O03 when the service context is present.
- `apps/console/package.json`: the test list.
- `docs/features/HB-281/*`.

# What the API offers (merged HB-280 code)

- `GET /api/v1/contracts?on=` returns `revision`, `on`, `today`, `impact` (money, proration, publication, ERP and final amount decision, each `not_computed`) and up to 200 rows: ID, name, profile and profile name, buyer, payer, issuer, issuance path, currency, timezone, cycle, state on the day, lifecycle revision, settlement, sequence, start, end, open gaps.
- `GET /api/v1/contracts/{id}?from=&until=` returns the row, terms by revision (customer, seller, transaction path, components with kind, provider-payment party and seller-billable flag), events with `withdraw_blocked` and `cancel_required`, segments for the period (exclusive `until`, at most 366 days), reopenings, evidence with write IDs and owners, the newest 100 records with actor login and reason, `earlier_records`, and `actions` with `allowed`, `blocked` and `earliest` for `contract.suspend`, `resume`, `renew`, `end`, `withdraw`, `evidence.secured`, `evidence.missing`, `settle` and `reopen`.
- Commands (target is the contract ID, reason required, `expected_revision` from the last read): `contract.suspend|resume|renew|end` (`from`), `contract.withdraw` (`event`, optional `cancel`), `contract.evidence` (`kind` secured with `reference`, `covered_until`, optional `resolves`; or missing with `reason`, `follow_up`), `contract.settle` (`as_of`), `contract.reopen` (`reason`). Success returns `business_state` (the settlement state) and `fields.write`, `fields.sequence`, and `fields.event` for lifecycle commands.
- Refusals: `invalid_request` with a field; `invalid_stage` and `prerequisite_failed` without fields; `revision_conflict`; `permission_denied` (also for an unknown contract on commands; the read answers 404 `unsupported_command`); `internal_error` with an operation; `dependency_unavailable`; `rate_limited`.

# Design

Follows the O01 and O02 live patterns: full viewport width, neutral rules only, EN, KO and AR, the shared failure box, reason field and error summary, `useWide` at 1280px, dates through `Intl` in UTC with `<bdi>` for IDs, and a status message and a polite live region for results (commands do not change the view, so no `done=` parameter is needed).

## Layout

- **List** of contracts on every view. With no contract selected it spans the full width. At 1280px and wider with a contract selected it is a narrow rail (name, state, settlement) beside the detail. Below 1280px the detail opens as a separate step with Back; below 768px the list renders stacked rows (labelled values per contract) and the filters sit in a collapsible panel, with no horizontal scroll at 390px.
- The list columns follow the spec order: contract, buyer (legal entity), payer, issuance path, period ("From S" and "until E, not included", or "No end"), state on the `on` day, final settlement (current), open gaps.
- Filters (client side over at most 200 rows): contract state, settlement state, issuance path, buyer legal entity. A count reads "N of M contracts".
- `on` (the state date, default the server's `today`) is a date field above the list; it changes the list state column only, and says so. It is validated on the client.
- Selection is `contract=` in the URL. Filters are `state=`, `settlement=`, `path=`, `buyer=`. The detail period is `from=` and `until=`.

## Views (the registered O03 tab keys)

1. **Parties and path** (`parties`): name and ID, billing profile, buyer, payer, issuer, issuance path, currency, timezone, billing cycle; from the terms, the customer, seller and transaction path (direct or resale) of each revision. Names come from the organizations listing when available, otherwise the ID.
2. **Billing schedule** (`schedule`): the lifecycle events (kind, day, rate revision), the segments for the period with days and state, a period picker (from and until, until not included), and the lifecycle commands. "Proration is not computed in this build" is stated with the segments; no amounts.
3. **Products and rates** (`products`): components per terms revision with kind, provider-payment party and seller-billable flag. Rates and allowances: "Not connected in this build".
4. **Documents and payment** (`documents`): "Not connected in this build" with what it will hold (ERP order, prepayment, payment and recipient references, template revision). No fields.
5. **Termination and final settlement** (`termination`): contract state and final settlement state side by side, each in words (ended can coexist with pending final). End day, open gaps, reopenings, evidence list (kind, reference or reason, covered until, owner, follow-up, resolved, write ID), the write log (newest first, actor, reason, what changed, earlier records count), and the evidence, settle and reopen commands. The final amount decision reads "Not computed".
6. **Payment terms** (`payment-terms`) and **Recognition** (`recognition`): "Not connected in this build".

## Commands

- Each command button appears only where its hint applies (see M6 for the fail-closed rules). A blocked hint is shown as a sentence in place of the button (for example "Suspend is not available: the contract is not active at its last event."). Unknown blocker codes, and `allowed: false` with an empty `blocked`, read "Not available for this contract now".
- Lifecycle forms: a date field bounded by M4 (min the hinted `earliest`, max the server's `today` + 3,660 days), the reason, and a confirm step that restates contract, action and day ("from the start of D, UTC"). End adds: "Ending moves final settlement to pending final. It does not cancel provider resources or subscriptions."
- Withdraw: per event in the events table, only where `withdraw_blocked` is empty. If `cancel_required`, a required choice between "Recorded in error" and "The contract continues", with a sentence explaining why.
- Secured evidence: reference (ID characters only, at most 128), covered until (at most the server's `today`, with the M4 cover-through hint), optional gaps to resolve as checkboxes of open gaps by write ID. Missing evidence: reason (the five gap reasons in words) and follow-up reference. The owner is the signed-in administrator, stated.
- Settle: as-of date (from the end day to the server's `today`); the confirm step says that settling closes final evidence only and the final amount decision is not computed. Reopen: the reopen cause (late source, late credit, correction) plus the rationale.
- After success the detail and list reload, the result is announced in the status and the live region, and focus moves to the result status (not back to the opener).

## Attempts and refusals

- Attempts follow H1 below: kept per command and contract across remounts, reused after an unknown outcome even if a reload moved the revision, and discarded on any edit or definite answer.
- `invalid` with a field: the field error and the summary; the hints reload. `stage` and `refused`: reload, then show the current hint sentence for that command. `conflict`: reload, keep the form, ask to confirm again. `denied`: the unavailable block. `limited`: the wait. `partial`: "Stopped after a change. Operation {id}. Check the write log before trying again."

## States

| State | What the screen shows |
| --- | --- |
| First load | Loading, with no numbers |
| Reload | Rows stay, `aria-busy` |
| Empty | "No contracts in this workspace", or "No contracts match the filters" with Clear filters |
| Denied | The unavailable block |
| Unauthenticated | Sign in again |
| Failure | An alert with Try again; forms keep their values |
| Contract not found | "This contract is not available" with a link to the list |

Unknown, missing or `not_computed` values are never shown as 0. Stale responses are dropped with a request sequence.

## External references

The ui-designer recalled these from memory; neither the designer nor root opened them, so they are unverified:
- **Stripe Billing subscription schedules** (https://docs.stripe.com/billing/subscriptions/subscription-schedules). Adopted: dated phases with exact start instants; actions only where the state allows. Rejected: coloured badges, modal editing, amount and proration previews.
- **Chargebee pause and cancel subscriptions** (https://www.chargebee.com/docs/2.0/pause-subscription.html). Adopted: each action has its own effective date; scheduled changes are listed and removable (per-event withdraw). Rejected: "immediately or end of term" choices.
- **Zuora Orders** (https://knowledgecenter.zuora.com/Zuora_Billing/Manage_subscription_transactions/Orders). Adopted: lifecycle actions as recorded events with an audit trail, separate from terms revisions. Rejected: several trigger dates per action.
- **AWS account closure** (https://docs.aws.amazon.com/accounts/latest/reference/manage-acct-closing.html). Adopted: closure and the final bill are separate, as ended next to pending final.
- **GOV.UK check answers and error summary** (https://design-system.service.gov.uk/patterns/check-answers/). Adopted: the confirm step restates every value; summary links move focus to fields.

The visual specification (ui-designer, AI) defines the `o03-` classes, the state pair, the tab grid at 390px, the stacked cards and the confirm step; the shell's tab links keep their existing keyboard behaviour (no arrow keys), recorded as an existing deviation from the spec.

## Out of scope (owner residuals)

- Single-administrator lifecycle and settlement commands (ADM06 separates operator and reviewer).
- Contract creation, `change` events, rate revisions, prices, proration and every amount.
- Documents, payment terms, recognition and ERP links.
- Server-side search and paging beyond 200 contracts.
- Owner FIX of the screen; KO and AR terminology review.

# Design review round 1 (CHANGES REQUIRED) and the resulting rules

**H1. Attempts survive remounts and unknown outcomes (O02 L-C).**
- A pending attempt is kept per (command, contract) in a module-level store that survives view remounts, together with its frozen `expected_revision` and the exact values sent.
- After an unknown outcome (network failure or `busy`), the next submit with identical values reuses the same attempt and the same `expected_revision`; a reload does not renew it until that resend has an answer. Editing any value, or `stale`, `partial`, success or any definite refusal, discards it.
- Form values are kept in the same store per (command, contract) so a view switch keeps them; they are cleared on success.
- Model test: busy, reload, resend uses the same key and the same expected revision.
- While an attempt is pending, a status reads "The last {action} may have been applied. Check the write log, or send it again", and the resend is offered even if the reloaded hint is now blocked; the server replays the original answer.
- The store is keyed by workspace and user as well, and is cleared on sign-out.

**M1. Contract not found.** `contract()` maps a 404 `unsupported_command` read to a distinct `missing` result, shown as "This contract is not available" with a link to the list. A client test covers it. (Commands against an unknown contract still answer `permission_denied`.)

**M2. Fieldless errors and the two `reason` fields.**
- `invalid` with no field attaches to the rationale ("Reason for this change").
- `invalid` with field `reason` attaches to the gap-reason control on missing evidence ("Why evidence is missing") or the reopen-cause control on reopen ("Reopen cause").
- The rationale limit is checked in UTF-8 bytes (256), because the server counts bytes; the counter says bytes.

**M3. Dates and exclusive ends.**
- A period reads "From S" with "until E (not included)" when an end exists, or "Ended from E" in the termination view. Segment rows use the same wording. Times are labelled UTC.
- The end confirm step reads "The contract is ended from the start of D, UTC. D is the first day it is not running."

**M4. Bounds from the server's `today`.**
- All defaults and bounds use the response's `today`, never the browser clock.
- Lifecycle `from`: min `earliest`, max `today` + 3,660 days.
- Settle `as_of`: min the end day, max `today`.
- Covered until: max `today`; when below the end day the hint says "To allow settling, cover through {end}".

**M5. Sentinels.** The parsers in `service/contracts.ts` keep missing or non-numeric `open_gaps`, `days`, `reopenings`, `earlier_records`, revisions and sequences as `null` (never 0), and missing strings as empty. Revision 0 reads "Not applicable" (suspend, resume, renew and end events carry no rate revision; a contract that has not started has no lifecycle revision). Empty dates read "—" only where the field does not apply (covered until on missing evidence, as-of, reopen and cancel on records). A missing login falls back to the actor ID (the seed shows `seed`). A direct path shows "No seller (direct)". An empty provider-payment party reads "Not applicable (seller service)". Null numbers read "Unknown", never 0.

**M6. Every blocker has a sentence, and hints fail closed.**
- Lifecycle: `settled`, `ended`, `not_active`, `not_suspended`, `horizon`, `write_limit`. Withdraw: `nothing_withdrawable`, `not_withdrawable`, `settled`, `breaks_lifecycle`, `write_limit`. Secured evidence: `not_pending_final`, `end_not_reached`, `write_limit`. Missing evidence: `settled`, `write_limit`. Settle: `not_pending_final`, `end_not_reached`, `open_gaps`, `evidence_needed`, `write_limit`. Reopen: `not_settled`, `write_limit`. Any other code: "Not available for this contract now".
- A missing action key, `allowed: true` with a non-empty `blocked`, or a lifecycle action allowed without a valid `earliest` is treated as not available.
- Start and change events read "Cannot be withdrawn in this build".

**Lows applied.**
- `invalid` on `event` reads "This event can no longer be withdrawn"; on `resolves` "A selected gap is no longer open"; both reload the hints.
- Legal entity filter: the list adds a buyer legal entity filter built from the rows.
- The detail states the contract state "today"; the list labels its state column "State on {on}". Settlement is always current and says so.
- Below 768px the list renders stacked rows (one card per contract with labelled values), and filters sit in a collapsible panel; no horizontal scroll at 390px. Below 1280px the detail is a separate step.
- `on` and the period are validated on the client (valid dates, until after from, at most 366 days) and shown as field errors.
- `expected_revision` is the detail read's `revision` that produced the visible hints.
- The announcement names the action in words, and the settlement state is labelled as settlement.
- The withdraw confirm states the settlement effect; withdrawing an end while pending final keeps open gaps and says so.
- Next to the `access_withdrawn_emergency` gap reason: "Emergency access withdrawal is never delayed to collect evidence."
- The spec's legal entity column is the buyer's, labelled "Buyer (legal entity)".
- Empty segments read "No days in this period".

# Verification plan

- Model tests: filters (including buyer), period validation, bounds from the server's `today`, every blocker sentence and the unknown and empty-blocker fallbacks, fail-closed hints, the attempt store (busy, reload, resend keeps key and expected revision; edits discard; cleared on sign-out), drafts surviving a remount, failure reactions including fieldless `invalid` and `invalid` on `reason`, `event` and `resolves`, UTF-8 byte counting, zero and empty display rules, name fallbacks, cancel requirement.
- Service tests: the 404 `unsupported_command` read maps to `missing`; parsers keep missing numbers as `null`.
- Service tests: parsers harden every field (missing, wrong type, unknown enum values).
- `npx tsc --noEmit`, `npm test`, `npm run build`.
- Playwright QA against the real API binary with the sample seed: each view in EN, KO and AR at 1440 and 390 widths, each command's success and a refusal, cancel-required withdraw, the settle path on a pending-final contract, the stage reload, keyboard order, no horizontal scroll; plus O01, O02 and M01 regressions.
- ui-designer render comparison; independent change review; `verification.md`.
