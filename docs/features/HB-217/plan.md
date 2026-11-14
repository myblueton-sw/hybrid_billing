---
type: ImplementationPlan
title: Billing findings and billing cases with separate review states, public answers and internal notes
description: Defines the second increment of work item W-22, a register of billing findings that are recorded by hand or by a named detector and reviewed through in review, confirmed normal, adjustment needed and resolved without changing any discount, approval or issued document, and a register of billing cases for inquiries and disputes on issued documents and lines with an owner, a response due time with its history, public answers that the customer sees and internal notes that it never sees.
status: draft
---

# Scope and ownership

[HB-217](https://linear.app/hybrid-billing/issue/HB-217) is the follow-up of HB-216 for the rest of work item W-22. This increment covers the detail contract AR-08 (BillingFinding) and the BillingCase of the implementation deliverable HB-22-I with the inquiry permissions of HB-22-V. Payment terms (PT-01 to PT-08), AR-10, DB-10, refunds and external authority locks stay in later increments of HB-217 or in follow-up tickets. It is built in delivery order under the owner's instructions of 2026-11-12 and 2026-11-14. Base: main `bdd5915`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the case register decides which customer sees which answer (tenant and permission boundary). Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-14:

- AR-08: a BillingFinding records a cost spike, a new charge, a suspected missing credit or discount, an unattributed amount, a reconciliation difference, a rate conflict or missing metering. It holds the target, the period, the meaning of the amount, the currency, the evidence, the severity and the owner, and separates in review, confirmed normal, adjustment needed and resolved. Manual registration and the review flow are required; automatic detection thresholds and scope are not decided. A finding alone changes no discount, approval or issue state.
- AUD-AR-08 (NOT_RUN): a new charge or benefit suspicion can be registered even when the reconciliation totals agree; confirmed normal and adjustment needed are distinct; the actual correction or resolution evidence is traceable.
- HB-22-D: a dispute does not cancel an issued document or the obligation to pay. The dispute SLA is open.
- HB-22-I: a BillingCase separates document and line, owner, due time, public answer and internal note.
- HB-22-V: inquiry permissions are verified.
- payment-terms.md: a dispute or a collection hold does not remove the obligation to pay or the overdue fact and is shown as a separate axis (PT-07).
- Existing code: `modules/balances` (HB-216) keeps balances; nothing records findings or cases.

# Decision

One Go package, `modules/cases`, standard library only. A registry with an injected clock (UTC, never earlier than its highest reading) and one mutex keeps per workspace the findings and the cases. Nothing calls it and no entry point is wired. The package imports no other module, so a finding or a case cannot change a discount, an approval, a balance or a document; a source test checks the imports.

Every string is an identifier under the project's rule (letters, digits, `. _ : -`, at most 128 bytes), a currency code of three capital letters, a date `YYYY-MM-DD` from 2000-01-01 to 2099-12-31 that exists, a decimal number text of at most 64 bytes (an optional leading `-`, digits without a leading zero unless the integer part is `0`, an optional fraction of 1 to 18 digits; read as text and never as floating point), a digest of 64 lowercase hexadecimal characters, or a value of a closed list. There is no free text: a reason is a code from a closed list and anything a person writes is kept elsewhere and referenced by its digest. Customer and actor identifiers must be pseudonymous (stated obligation).

Errors: `ErrInvalid` (form or size), `ErrNotFound`, `ErrConflict` (another fact under the same identifier), `ErrRejected` (state or rule), `ErrLimit` (room). Checks run in this order: form, lookup, identical repeat or conflict, rule, room.

## Actors

Every call names an `Actor{ID, Side, Customer}`. `Side` is `operator` or `customer`; a customer actor names its customer, an operator actor names none. The registry trusts the actor as stated: authenticating it is the caller's duty (stated obligation). Findings are operator-only; a customer actor gets `ErrRejected` for any finding call.

## Findings

`RecordFinding(ws, Finding, actor)` records, written once:

- `ID`;
- `Kind`: `cost_spike`, `new_charge`, `missing_credit_or_discount`, `unattributed`, `reconciliation_difference`, `rate_conflict`, `metering_gap`;
- `Target{Customer, Contract, Document, Line, Resource}`: the customer is required, the others optional identifiers, the line only with a document;
- `PeriodStart`, `PeriodEnd`: dates with start ≤ end;
- `Meaning`: `cost`, `sale`, `difference`, `none`; with `none` the amount and currency are empty, otherwise both are required; only `difference` allows a negative amount;
- `Amount`, `Currency`;
- `Evidence`: a digest, required;
- `Severity`: `low`, `medium`, `high`, `critical`;
- `Owner`: an operator identifier;
- `Source`: `manual` or `detector`, and `Detector`, an identifier required exactly with `detector`.

The finding starts `in_review`. An identical repeat by the same actor returns it; any other content under the same identifier is a conflict. A finding never needs a reconciliation difference (AUD-AR-08).

Transitions, each with a reason code and an evidence digest, each kept in the finding's history with actor and time:

- `ConfirmNormal(ws, id, actor, reason, evidence)`: `in_review` → `confirmed_normal`; reasons `expected_change`, `already_corrected`, `duplicate`, `out_of_scope`.
- `RequireAdjustment(ws, id, actor, reason, evidence)`: `in_review` → `adjustment_needed`; reasons `rate_error`, `missing_credit`, `missing_discount`, `attribution_error`, `metering_error`, `source_error`.
- `Resolve(ws, id, actor, Resolution{Kind, Reference, Evidence})`: `adjustment_needed` → `resolved`; kinds `adjustment_draft`, `correction_document`, `credit_document`, `source_correction`; the reference is the identifier of that record elsewhere and is required.
- `Reopen(ws, id, actor, reason, evidence)`: `confirmed_normal` or `resolved` → `in_review`; reasons `new_evidence`, `resolution_failed`.
- `Assign(ws, id, actor, owner)`: any state; the owner changes and the change is kept in the history.

Separation of duties: the actor of `ConfirmNormal` and `RequireAdjustment` must not be the actor who recorded the finding (`ErrRejected`). `Resolve` and `Reopen` may be any operator. A transition from the wrong state is `ErrRejected`. Each transition carries a client step identifier; repeating the same step with the same content returns the current view, the same step identifier with other content is a conflict. History per finding is bounded (`Limits.Steps`).

Reads: `Finding(ws, id, actor)` and `Findings(ws, actor, Filter{State, Customer, Severity})` sorted by identifier, bounded by `Limits.Page` with a cursor (the last identifier).

## Cases

`OpenCase(ws, Case, actor)` records, written once:

- `ID`;
- `Kind`: `inquiry` or `dispute`;
- `Customer`;
- `Subjects`: 1 to `Limits.Subjects` pairs `{Document, Line}` with the line optional, no pair repeated; a dispute needs at least one subject;
- `Topic`: `amount`, `quantity`, `rate`, `tax`, `credit`, `delivery`, `access`, `other`;
- `Disputed`: for a dispute, the disputed amount text and currency, optional but together;
- `Body`: a digest of the opening message kept elsewhere.

A customer actor may open a case only for its own customer. An operator may open a case for any customer of the workspace, for example on the customer's behalf. Either way the case keeps who opened it and from which side. The case starts `open` with no owner and no due time.

Operator actions:

- `Assign(ws, id, actor, owner)`: sets the owner.
- `SetDue(ws, id, actor, due, reason)`: sets the response due time, a time strictly after now and at most 366 days later; the first value is the original due time, every later value keeps the original and adds a history entry with reason code `initial`, `customer_request`, `information_pending`, `escalation`, `owner_change`.
- `Note(ws, id, actor, Entry{ID, Body})`: an internal note.
- `Answer(ws, id, actor, Entry{ID, Body})`: a public answer; the case becomes `answered`.
- `Close(ws, id, actor, Outcome)`: from `open` or `answered`; outcomes `answered` (inquiry only), `upheld` (dispute only, with a remedy kind `adjustment_draft`, `correction_document`, `credit_document` and a remedy reference), `not_upheld` (dispute only). The registry records the remedy reference and creates nothing.

Customer actions on its own cases:

- `Reply(ws, id, actor, Entry{ID, Body})`: a public message; a closed case becomes `open` again (reopened) only through `Reopen`.
- `Reopen(ws, id, actor)`: a closed case becomes `open` again; allowed for its customer and for operators.
- `Withdraw(ws, id, actor)`: from `open` or `answered`; the case is `withdrawn` (terminal).

Operators may also reply publicly with `Answer`; customers can never write a note. Entries are identified per case; the same entry identifier with the same content is an identical repeat; another content is a conflict. Entries per case are bounded (`Limits.Entries`).

Permissions and visibility:

- A customer actor sees only the cases of its customer; any other case is `ErrNotFound`, so it cannot learn that a case exists.
- A customer view contains the subjects, topic, state, kind, the public messages and answers, the current due time and the outcome. It never contains internal notes, the owner or the due history reasons.
- An operator view contains everything.
- `Cases(ws, actor, Filter{State, Customer, Kind, Document})` lists under the same visibility, sorted by identifier, with a page bound and cursor.

`Disputes(ws, document)` (operator only) returns the open or answered disputes naming that document, so a later receivables increment can show the dispute axis next to the obligation. A dispute never changes the document or the obligation; there is no call into another module.

Overdue response: a view carries `ResponseOverdue`, true when the case is `open` (not answered since the last customer message), a due time is set and now is not before it. The SLA itself is not decided; the operator sets the time.

## Digests and limits

Each finding, transition, case and entry has a SHA-256 digest over length-prefixed fields with a version label. `Limits{Findings, Cases, Steps, Entries, Subjects, Page}` are per workspace or per record, all positive; exceeding one is `ErrLimit`, except a page, which is clamped.

# Provisional choices awaiting the owner

- Reason, kind, topic and outcome lists above.
- Separation of duties only for ConfirmNormal and RequireAdjustment.
- Customers may reopen closed cases without a time limit.
- The response SLA is set by hand; no default time.
- Customer views hide the owner identifier.

# Verification

- Literal tests for every transition, refusal, visibility rule and the tenant boundary (workspace and customer).
- A property test against an independent model of the states, history and visibility over random call sequences.
- Race detector, count=10, a 386 build, gofmt, vet, the whole repository test run, a mutation run.
- Source scan: no floating point, no comments, no import of another module.
- Not covered: persistence, screens, notifications, the receivables dispute axis display, system acceptance AUD-AR-08 end to end.

# Design review resolution

The first design review returned FAIL with findings 1 to 12. Where this section and the text above disagree, this section governs.

1. Finding target: the customer is optional. At least one of customer, contract, document and resource is required. `unattributed` forbids a customer. `FindingFilter.Unattributed` selects findings with no customer, and it cannot be combined with a customer filter.
2. Disputes: `Disputes(ws, customer, document, actor)` is operator-only. It returns only the open or answered disputes of that customer that name that document. Stated caller obligation: the subject documents of a case belong to the case's customer.
3. Case identifiers are scoped per (workspace, customer). Every case call names the customer and the identifier. A customer actor naming another customer gets `ErrNotFound` before any lookup. Opening a case with another customer's identifier is therefore impossible to observe.
4. Entry identifiers are scoped by visibility. Public entries (customer messages and operator answers) share one space. Internal notes have their own. A customer can never collide with a note. Same content means the same kind, body, actor and side.
5. Check order for operator-only calls (every finding call and `Disputes`): actor form, then side (`ErrRejected` for a customer), then the remaining form, lookup, repeat, rule and room. A customer learns nothing about finding identifiers.
6. Customer case view, exactly:
   - the case as opened;
   - the state;
   - the opened side, the opener only when a customer opened it, and the opened time;
   - the current due time and ResponseOverdue;
   - public entries, with the actor and digest blanked on operator entries;
   - the outcome;
   - the remedy only when its kind is `correction_document` or `credit_document`.

   Never shown: internal notes, the owner, the closer, the original due time, the due history and the case digest. A property test checks that no customer view contains a note identifier or an operator identifier.
7. The case state table and step identifiers. Every case transition (`AssignCase`, `SetDue`, `Close`, `ReopenCase`, `Withdraw`) carries a client step identifier. Step spaces are separated by side. Repeating a step with the same content by the same actor returns the current view; the same step with other content is `ErrConflict`. Results per state (✓ allowed, ✗ `ErrRejected`):

   | call | open | answered | closed | withdrawn |
   | --- | --- | --- | --- | --- |
   | Reply (customer) | ✓ open, awaiting | ✓ open, awaiting | ✗ | ✗ |
   | Answer (operator) | ✓ answered | ✓ answered | ✗ | ✗ |
   | Note (operator) | ✓ | ✓ | ✓ | ✓ |
   | AssignCase, SetDue (operator) | ✓ | ✓ | ✗ | ✗ |
   | Close (operator) | ✓ closed | ✓ closed | ✗ | ✗ |
   | ReopenCase (customer of the case or operator) | ✗ | ✗ | ✓ open, awaiting | ✗ |
   | Withdraw (customer) | ✓ withdrawn | ✓ withdrawn | ✗ | ✗ |

   The closed state is `closed`. Reopen clears the outcome, remedy, closer and closed time from the current state and keeps each closing in the case's closing history. A reopened dispute is listed by `Disputes` again.
8. Due time after a reopen or a customer message after an answer: the current due time is cleared, so ResponseOverdue is false until an operator sets a new one with reason `follow_up`. The original due time and the due history are kept. `initial` is allowed only while no original due time exists.
9. Limits: `Limits{Workspaces, Findings, Cases, CustomerCases, Steps, Messages, Entries, Subjects, Page}`, each with an upper cap.
   - `CustomerCases` bounds the cases of one customer.
   - `Messages` bounds customer messages per case. `Entries` bounds operator answers and notes per case. A customer can therefore never use up the operator's room.
   - Close, Reopen and Withdraw count against no room.
   - Finding `Steps` counts only assign and reopen steps. Each decision step needs a reopen before another, so the total stays bounded. A finding can always reach a decision.
   - Due changes count against `Steps`.
10. `ReopenFinding` is allowed from `confirmed_normal`, `adjustment_needed` and `resolved`. Reasons: `new_evidence`, `resolution_failed`, `decision_revised`.
11. Separation of duties: ConfirmNormal and RequireAdjustment refuse the recorder and the actor of the latest reopen. An identical RecordFinding by another actor is `ErrConflict`.
12. An inquiry may have zero subjects; a dispute needs at least one. A disputed amount is positive.

The reviewer's hypothesis is recorded as provisional: customers may reopen without a time limit, so a dispute marker can be kept on a document indefinitely. It is listed for the owner's decision.

## Re-review resolution

The re-review returned FAIL on findings 13 and 14 (medium) and 15 to 18 (low).

13. `ReopenCase` counts against a per-case `Steps` room, and so do `AssignCase` and `SetDue`. `Close` and `Withdraw` count against no room, so a case can always reach `closed` or `withdrawn`. Each close is preceded by a counted reopen, except the first, so the closing history stays bounded.
14. Only the record digest of operator entries is blanked in a customer view. The `Body` digest of a public operator answer stays visible, and a literal test checks that a customer reads an answer's body.
15. An identical repeat of any step or entry returns the current view and causes no transition: no state change and no due-time clearing. A literal test retries a customer reply after an answer.
16. The operator `Cases` list sorts by (customer, identifier), and the cursor is that pair (`CaseFilter.AfterCustomer`, `CaseFilter.After`).
17. Reopen reasons by source state:
    - from `confirmed_normal`: `new_evidence`, `decision_revised`;
    - from `adjustment_needed`: `new_evidence`, `decision_revised`;
    - from `resolved`: `new_evidence`, `resolution_failed`.

    Any other pair is `ErrRejected`.
18. A note on a `withdrawn` or `closed` case is intended, so an operator can record follow-up. It uses the operator `Entries` room.

# Change review resolution

The independent change review returned FAIL on exactness (findings 1 to 3). It found no confirmed defect in the customer boundary, the state table, the finding transitions, the rooms or the concurrency. Its hypotheses and test gaps are resolved as follows.

1. A customer actor that opens a case for another customer now gets `ErrNotFound`, as item 3 says. The literal test and the property model expect `not_found`.
2. Subjects beyond `Limits.Subjects` are now `ErrLimit`. The check runs after the form checks and the repeat check.
3. Case transitions now carry digests:
   - every due change, including clearing, has its own digest;
   - every closing has its own digest, and its reopen has a separate one (`ClosingView.Reopen`);
   - assignments are kept as an owner history (`CaseView.Owners`: step, owner, actor, time, digest) that only operators see.
4. Accepted as provisional: a customer whose open hits the workspace case room gets `ErrLimit`, which reveals only that the workspace is full.
5. `Disputes(ws, customer, document, after, actor)` takes a cursor (the last case identifier) and returns at most `Limits.Page` cases.
6. Intent confirmed: after a reopen, any due reason except `initial` is allowed. `follow_up` is the expected reason, not the only one.
7. The customer `Note` refusal before the repeat check is intended (item 4).

Test gaps closed:

- T1: dispute filtering by kind, state, document, cursor and page.
- T2: the finding property model now uses its own transition and reopen tables. A literal test refuses `decision_revised` from `resolved`.
- T3: in every state, a set of refused calls (including operator reopen and withdrawal) leaves the operator view deeply equal to before.
- T4: closing a case with `source_correction` is `ErrInvalid`.
- T5: mixed concurrent replies, answers, assignments and lists keep each room exact.

Mutation survivors of the first run were closed by literal tests:

- identifier, currency and digest character bounds;
- an amount with no meaning;
- assignment in every finding state;
- malformed reads and moves;
- exact page sizes;
- a conflict that differs only in the currency;
- a pinned finding digest computed independently in the test.

These fixes were not reviewed again.
