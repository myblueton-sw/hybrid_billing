---
type: ImplementationPlan
title: ERP common submission ledger with capability-checked connections, existing-path guards, idempotent submissions, row-level document links and revision-ordered status sync
description: Defines the first increment of work item W-24, a register of ERP connections that never write the ERP database, master mappings, anchors for existing ERP orders and payables fixed before submission, billing submissions with single-use keys, unknown outcomes resolved by lookup before any retry, external document rows linked many-to-many with platform-attributed totals kept apart from document totals, and webhook and polling observations applied in revision order without ever creating a submission.
status: draft
---

# Scope and ownership

[HB-226](https://linear.app/hybrid-billing/issue/HB-226) starts work item W-24 (requirement HB-24; detail contracts DB-01, DB-03, DB-04, DB-11 and DB-12; acceptance AUD-DB-01, -03, -04, -11, -12 and the HB-24-V criterion) after W-23. It is built in delivery order under the owner's instructions of 2026-11-12, 2026-11-14 and 2026-11-15. Base: main `e094a6c`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews the plan and the change, because submissions decide what is billed in the ERP. Machine: mac_mini.

Not in this increment: provider connectors (Odoo, SAP; W-25), the standard XLSX files themselves (W-26), payment terms (already in `modules/receivables`), the recognition-purpose submissions of `modules/recognition` (to be joined in the integration layer), persistence, screens and real ERP calls.

# Observed baseline

- HB-24-D: in ERP issuing mode the ERP owns final numbers, tax and accounting documents; API and file are transports over the same submission ledger; the ERP's external references, draft lock and post protection and tax and master authority are kept.
- HB-24-I: ERPConnection, MasterMapping, BillingSubmission, ExternalDocumentLink and SyncEvent; the commands connect, validate_mapping, prepare, submit_draft, read_draft, post, get_status, correct and get_balance with capabilities; many-to-many document and row reconciliation.
- DB-01: the standard settlement and customer files, result registration and the shared ledger are common release features; an unverified ERP upload template is never shown as verified (AUD-DB-01).
- DB-03: existing ERP sales orders, contracts and prepaid references and the direct or order-based path are fixed before submission; one scope is never billed by two paths; payables are reconciled against existing AP documents and only unreflected scope is sent; an unknown external ledger holds the submission (AUD-DB-03).
- DB-04: only rows and allocations linked to platform submissions are reconciled; unrelated ERP products on the same invoice are excluded; document totals and platform-attributed totals are shown separately (AUD-DB-04).
- DB-11: REST, OData, SOAP or API, or approved files; never direct ERP database writes; unsupported commands are not completed by a bypass (AUD-DB-11).
- DB-12: webhooks are complemented by periodic status checks that record the target, last check, next check, source revision, retries and owner; missing or reordered webhooks are recovered by polling and never recreate an invoice (AUD-DB-12).
- HB-24-V: external success with local save failure, manual ERP edits, reordered webhooks, split and merged documents and a switch from API to file create no duplicate documents; timeouts are `unknown` and only confirmed unreflected items are retried.
- `modules/connectors` registers source connectors, not ERP targets. `modules/recognition` keeps its own recognition submissions. No module holds an ERP submission ledger.

# Decision

A new package `modules/erpledger` holds one registry per process with an injected clock (highest reading, UTC, rounded to zero), one mutex, workspace isolation, the errors `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`) and `ErrLimit`, and the check order form, lookup, repeat or conflict, rule, room. An identical repeat of a write by the same human returns the current view. Writes need a human (`ai_not_allowed`, `not_human`), except `Observe`, `Checked` and `Document`, which a system connector may call because they register what the ERP reports; an assistant is always refused, including on repeats. Links are allocation decisions and need a human. Amounts are decimal strings on the connection's amount scale, held as integer units and bounded by the decimal profile.

## Connections

`Connect(workspace, Connection, actor)` records an immutable connection: identifier, entity, ERP system label, transport (`api` or `file`), capabilities (a non-empty subset of `connect`, `validate_mapping`, `prepare`, `submit_draft`, `read_draft`, `post`, `get_status`, `correct`, `get_balance`), currency, amount scale (0 to 8), polling interval (1 minute to 1 day), owner and, for `file`, the upload template identifier. The transport `database` is refused (`direct_database_forbidden`) and so is any capability string outside the list (`ErrInvalid`). A command that is not in the capabilities is refused with `unsupported` and changes nothing; no command has a fallback.

A template starts unverified. `VerifyTemplate(workspace, connection, evidence, actor)` marks it verified with an evidence reference, by a human other than the connection's author. The view always shows whether it is verified.

## Mappings

`Map(workspace, Mapping{Connection, Kind, Platform, External}, actor)` records a mapping of kind `customer`, `product` or `tax` with a revision per connection, kind and platform key; the latest revision is in force. It needs the `validate_mapping` capability.

## Existing ERP paths

`Anchor(workspace, Anchor{ID, Connection, Kind, External, Customer, Sources}, actor)` records an existing ERP document that already covers whole source lines: `sales_order`, `contract`, `prepaid` or `ap_document`. Anchors are immutable. Anchor blocking and source uniqueness are workspace-wide, across connections and transports, so a switch from API to file never re-bills a source. An anchor is an ERP fact and is always recorded; when it covers a source of a live submission whose purpose it blocks, that submission is marked `diverged`, because the same scope is then billed by two paths in the ERP.

`Survey(workspace, Survey{Connection, Customer, Period, Outcome, Evidence}, actor)` records that the existing ERP ledger of a customer and period (`YYYY-MM`) was checked through that connection, with outcome `complete` or `unknown`. Surveys are per connection; the latest counts.

## Submissions

Keys are unique per workspace. `Prepare(workspace, Submission, actor)` records a submission: connection, purpose (`customer_invoice` or `payable`), customer, period, path (`direct`, or `order_based` with the anchor it follows, for invoices only) and lines (identifier, source line, product, tax code, quantity, net and tax). A repeated line identifier or source line inside one submission is `ErrInvalid`. A repeat of the key with identical content by the same human returns the view (the preparation time is not compared); anything else under the key is `ErrConflict`. Refusals, in order:

- actor kind;
- `unknown_connection`; `unsupported` without `prepare`;
- `not_surveyed` when the customer and period have no survey on this connection; `ledger_unknown` when the latest survey is `unknown`;
- `mapping_missing` when the customer, a product or a tax code has no mapping on this connection;
- `unknown_anchor` when an order-based path names no `sales_order`, `contract` or `prepaid` anchor of this connection and customer;
- `already_reflected` when a line's source is covered by any anchor that blocks the purpose (`sales_order`, `contract` or `prepaid` for an invoice, except the anchor an order-based path follows; `ap_document` for a payable);
- `already_submitted` when a line's source is in another live submission of the same purpose on any connection. A submission is live unless it is `rejected`, `absent`, `superseded` or `withdrawn` and not diverged, so a new key never re-bills a scope whose outcome is unknown or unconfirmed;
- `empty` without lines; `ErrLimit` for too many lines or out-of-bound totals.

The digest (label `erp_submission_v1`) covers every field, the mapping revisions used, the totals, the author and the preparation time.

`Command(workspace, key, name, actor)` requests `submit_draft`, `read_draft`, `post`, `get_status`, `correct` or `get_balance`. It needs the capability (`unsupported`) and the state (`wrong_state`): `submit_draft` (for `file`, the file handoff) only from `prepared`, and for `file` only over a verified template (`template_unverified`); it moves the submission to `sent` and starts the watch. `post` only from `draft`; `correct` only from `posted`; the others from any state after `prepared`. `submit_draft`, `post` and `correct` are refused while the submission is diverged (`diverged`). Commands are recorded in order.

## Observations

`Observe(workspace, Observation{ID, Connection, Key, External, Revision, State, Source}, actor)` records what the ERP reported, from a `webhook`, a `poll` or a `result_file`. The identifier is single-use per connection; an identical repeat returns the view and different content is `ErrConflict`. The target is the submission named by the key, else the one whose external reference on this connection matches. A key on another connection, or a key and an external reference that name different submissions, is `ErrConflict`. An observation that matches nothing is kept as an orphan and never creates a submission. An observation of a `prepared` submission is `wrong_state` and records nothing.

- States: `draft`, `posted`, `rejected` (the ERP refused it, no document exists), `absent` (a lookup confirmed that the ERP holds nothing) and `unknown`. `absent` is accepted only from a `poll` or a `result_file`, never from a webhook, and applies only to an `unknown` submission, because a lookup right after sending may miss a document the ERP is still processing; an `absent` report for a `sent` or `draft` submission is recorded as premature: it moves the submission to `unknown` and advances the revision but frees nothing, and a later `absent` at a higher revision confirms the absence. Only `rejected` and `absent` free the sources through observations.
- Revisions are positive integers per submission. A revision at or below the last one is recorded as stale and changes nothing. A higher revision always advances the last revision: it applies the state from `sent`, `draft` or `unknown`; it repeats a terminal state without change; and when it reports a different state for a `posted`, `rejected` or `absent` submission it marks the submission `diverged` for manual review and keeps the state. This covers manual ERP edits and an earlier absence that turns out wrong.
- When a `rejected` or `absent` submission diverges, every later live submission that shares a source with it is marked `diverged` as well, so a replacement is never left unnoticed.
- `draft` and `posted` need an external reference, unique per connection; once set, a different reference is `ErrConflict`.
- A poll also updates the watch entry.

A submission marked `diverged` by spreading or by an anchor keeps receiving observations normally; only the flag stays until resolved.

`Resolve(workspace, key, Resolution{Outcome, Evidence}, actor)` clears `diverged`, by a human other than the author: `accept` applies the last observed state and reference (refused as `wrong_state` when nothing was observed), `keep` leaves the state, and `superseded` retires the submission with evidence of an ERP-side correction or reversal, which frees its sources. `keep` and `accept` are refused while a blocking anchor covers one of the submission's sources (`already_reflected`), except the anchor its order-based path follows; only `superseded`, or `Withdraw` while `prepared`, remains. `keep` is also refused when the last observation contradicts a `rejected`, `absent`, `superseded` or `withdrawn` state (`wrong_state`), so a retired submission the ERP reports as existing can only be accepted or superseded. A resolution that would leave two live submissions sharing a source is refused (`already_submitted`); the way out of a wrong absence is to supersede the replacement and then accept the original, or the reverse.

`Withdraw(workspace, key, actor)` retires a `prepared` submission that was never sent (`wrong_state` otherwise), by any human, clears `diverged` because nothing reached the ERP, and frees its sources. Status, draft and balance commands are refused on a withdrawn submission.

## Status watch

Watched submissions are those in `sent`, `draft` or `unknown`, every diverged one, and `posted` ones for 30 days after posting, because a posted document can still be edited. A watch entry has the last check, the next check (the last check, or the submission time, plus the connection's interval), the last revision, the retry count and the connection's owner. `Due(workspace, at)` lists the keys whose next check is at or before `at`. `Checked(workspace, key, outcome, actor)` records a status check without an answer (`timeout`, `error`): it increments the retries, moves the next check and turns a `sent` submission into `unknown`. A poll observation resets the retries.

## Documents and links

`Document(workspace, Document{Connection, External, Revision, Rows}, actor)` records an ERP document with rows (identifier, product label, net, tax, quantity); a higher revision replaces the rows and a lower one is stale (`ErrConflict` unless identical). `Link(workspace, Link{Connection, External, Row, Key, Line, Net, Tax, Quantity}, actor)` allocates part of a row to a line of a live submission on the same connection (`connection_mismatch`). Each allocated component is zero or has the sign of the row component and of the line component; a zero row or line component takes only zero. The allocations of a row never exceed the row and those of a line never exceed the line in absolute value (`over_allocated`). A document revision that removes a row, flips a sign or shrinks a row below its allocations marks those links `stale`; a stale link can be replaced by a new link for the same row and line. A link records its author, and only that human's identical repeat returns the view.

Documents are keyed per connection. The same ERP document recorded under two connections is two records, but a line's allocations are bounded across all links, so attributed totals never count twice.

`Reconcile(workspace, key)` returns, per line, the sum of its non-stale links and whether it equals the line, and for every linked document both its full totals and the platform-attributed totals (non-stale links of live submissions only). Rows without links never count in attributed totals.

## Reads and limits

`ConnectionOf`, `SubmissionOf`, `DocumentOf`, `Orphans` and `Due` return copies. Limits per workspace: connections, mappings, anchors, surveys, submissions, lines per submission, documents, rows and links per document (stale links included, so a document relinked very often reaches the limit) and observations, each with a fixed maximum; commands are capped at 1,000 per submission.

# Tests

- AUD-DB-01: a file connection (capabilities `prepare`, `validate_mapping`, `submit_draft`) with result registration completes a first reconciliation without any API command; the upload template stays unverified until verified with evidence by a second human.
- AUD-DB-03: a source covered by an existing sales order, contract or prepaid anchor is refused for a direct invoice and allowed for the order-based path that follows it; payables skip sources in AP documents; an unknown or missing survey holds the submission.
- AUD-DB-04: a merged ERP invoice with platform and unrelated rows, and one submission split over two documents, reconcile only the attributed amounts.
- AUD-DB-11: `database` transport is refused; unsupported commands are refused and change nothing.
- AUD-DB-12: missing and reordered webhooks are recovered by polls; stale revisions change nothing; orphan observations create nothing.
- HB-24-V: external success with local save failure (a repeat with the same key and a later observation), manual ERP edits (`diverged`), an absence that turns out wrong (both submissions `diverged`), split and merged documents, a switch from API to file (a new connection never re-bills live sources), and timeouts kept `unknown` with no new key accepted until a poll confirms `absent`.
- A property test against an independent model of submission states, revisions and allocation sums; race, 386 build, gofmt, vet, coverage and mutation testing.

# Provisional choices

- The kinds, states, transitions, capability names, interval range and limits.
- Purposes limited to customer invoices and payables in this increment.
- `diverged` instead of following manual ERP edits automatically.
- Corrections recorded as commands only; correction documents arrive as new document revisions.
- Anchors cover whole source lines; partially reflected AP or prepaid amounts are out of scope.
- An unverified file template blocks the file handoff.
- A webhook `rejected` for a draft frees the sources while the ERP still knows the draft reference.
- Posted submissions are watched for 30 days.
- `absent` applies only to `unknown` submissions; a connection-specific delay or double poll is not modelled.

Residual, not built: the AUD-DB-01 release gate that blocks omitting the common file features, and the HB-24-V scale controls (queues, concurrency and call-rate limits per entity and ERP connection).

# Review history

- Design review 1 (independent reviewer, read-only): CONDITIONAL PASS with 6 high, 6 medium and 4 low findings. Folded in: diverged submissions stay live and divergence spreads to later submissions sharing a source; repeated sources inside a submission are `ErrInvalid`; workspace-wide anchors and source uniqueness, surveys per connection; anchors refused over live submissions; observations bound to their connection and conflicting key and reference refused; confirmed absence (`absent`, poll or result file only) as the only freeing outcome besides `rejected`, and `Checked(timeout)` turning `sent` into `unknown`; revision rules for every case; posted submissions watched for 30 days; `Resolve` for diverged submissions; link connection, sign, removal and exclusion rules; unverified templates block the file handoff; whole-line anchor coverage and invoice-only order paths; observation repeat conflicts; repeat and digest wording; fixture capabilities; residuals recorded.
- Design review 2 (same reviewer): CONDITIONAL PASS; all 16 findings resolved, 4 medium and 3 low new findings, folded in: `superseded` resolution to leave a wrong-absence pair, `absent` only from `unknown` with premature reports recorded, anchors always recorded and marking overlapping live submissions `diverged`, `Withdraw` for prepared submissions, observation handling of spread-diverged submissions, repeat comparison without preparation time, documents keyed per connection with line-bounded attribution.
- Design review 3 (same reviewer): CONDITIONAL PASS with 2 medium and 1 low findings, folded in: diverged submissions refuse `submit_draft`, `post` and `correct`, and `keep`/`accept` are refused while a blocking anchor covers a source; a premature `absent` moves `sent` or `draft` to `unknown`; `keep` cannot retire a submission the ERP reports as existing.

# Change review resolution

Change review 1 (a separate independent reviewer, read-only): FAIL with 5 medium and 3 low findings and 3 hypotheses; no double-billing path was found. All were addressed:

1. `Withdraw` clears `diverged`, so an anchor-diverged prepared submission frees its sources for the order-based path; tested.
2. Returned views copy the resolution.
3. Stale links are skipped when looking for an existing link, so a row can be relinked after a revision; tested.
4. `Link` needs a human; `Document` stays open to system connectors, now stated in the plan.
5. The property test adds documents, revisions and links with independent row, line and attributed-total invariants, and `Resolve` keep and accept with the shared-source invariant after every step. `live` and the observation rules are still mirrored in the model; the invariants are independent of them.
6. A diverged submission records the reported reference, so later reference-only observations reach it.
7. Assistants are refused before repeats in `Observe` and `Document`; links record their author.
8. Amounts are checked against the connection scale before the rule refusals; status commands are refused on withdrawn submissions; surveys, links and commands have limits.
- H1: a premature absence is recorded as `unknown` in the last observation, so `accept` cannot apply an unconfirmed absence.
- H2: recorded as a provisional choice.
- H3: `accept` into a watched state starts a watch entry.

Change review 2 (same reviewer): PASS; all findings and hypotheses resolved. Low notes kept: the per-document link limit counts stale links (now stated above), the property model copies production results after `Resolve` instead of predicting refusals, and system connectors' identical observation and document repeats return the view for any non-assistant actor.
