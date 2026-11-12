---
type: ContractDesign
title: Closing-target read contract for U02
description: Proposes the read-only closing-target query for the closing list and calendar, separates what planning defines from what it leaves open and from what no producer supplies today, and lists the owner decisions and implementation sequence.
status: draft
---

# Purpose and status

[HB-156](https://linear.app/hybrid-billing/issue/HB-156/design-the-closing-target-query-contract-for-u02) follows the owner's answer of 2026-11-12 to a question from root: for U02 (`/closing`), design the closing-target backend first. Root (Claude Code) wrote this design. The independent reviews that have run are recorded at the end with their results. Machine: mac_mini. This document proposes a contract. It deploys nothing, changes no code, schema, data or permission, and is not an implemented API.

Evidence was read on 2026-11-12. Statements with a file reference are observations; statements without one are proposals of this document. The planning files `requirements.md`, `detailed-requirements.md` and `ui-design.md` under `docs/planning/v1.16/` are local authorized-workspace inputs that are not yet in Git; their line numbers refer to that local copy. The planning contracts cited here are themselves proposals whose acceptance cases are NOT_RUN.

# What planning defines

- Scope is the "permitted per-contract ClosingCycle"; internal chargeback and external-issuance contracts have different completion conditions; each cycle is shown as completed, incomplete or exempt, and a calendar cell opens the same ClosingCycle detail (`docs/ui/screens/U02/spec.md:29-31`).
- A ClosingCycle holds `original_due_at`, `current_due_at`, deadline-change history and a fixed target manifest. Exemption is a state separate from completion and needs reason, authority and evidence. Denominator changes from new targets, exemptions and holds, the first deadline and delay history stay queryable after additions, postponement and reopening (`detailed-requirements.md:389-391`, QA-03).
- Each `StageSchedule` is keyed `(cycle_id, schedule_id, stage)`, several named schedules may exist within one stage, and it carries original and current deadline, timezone and calendar, completion predicate and history (`docs/contracts/commands/planning-actions.md:68`, `docs/architecture/data-ownership/logical-data-schema.md:189`). The schedule-change command identifies its target as "the ClosingCycle plus explicit operational stage and schedule ID, never a bare cycle deadline" (`U02/spec.md:128`).
- The original deadline is immutable; the current deadline needs a calendar revision, timezone, reason and impact review; a missing calendar policy blocks and no date is substituted (`U02/spec.md:63-65`, `:130-132`). Local and UTC timestamps are both bound (`U02/spec.md:128`).
- Monthly-recognition closing and quarterly-billing closing keep separate cadence and target identity; the payment schedule is governed by a separate finance authority (`U02/spec.md:130`, `planning-actions.md:72`).
- Reads are server-filtered, stably sorted and paged by snapshot cursor; totals come from the same snapshot; an expired cursor needs a clear error and resync path; unauthorized objects must not be disclosed through existence, counts, totals or autocomplete; owner is personal data and exemption evidence is restricted; more than 100,000 targets are expected (`U02/spec.md:31`, `:68`, `:82`, `:155-171`; HB-30 `requirements.md:543-545`; QA-05 `detailed-requirements.md:417`).
- Unknown is never folded into another value (`U02/spec.md:165`). Accepted or unknown job outcomes are not completion (HB-34, `requirements.md:607-623`). DST or policy changes force review of a schedule (`U02/spec.md:134`).

# What planning leaves open

These are gaps or conflicts in the planning sources. Each maps to an owner decision below; this document does not settle them.

1. **Row grain.** The list, the three states, exemption and denominators are described per ClosingCycle, and QA-03 puts both deadlines on the cycle; the command contract puts deadlines on each StageSchedule. Whether a list row and a count unit is a cycle or a stage schedule is not stated.
2. **Cycle key.** No document gives the ClosingCycle key or its relation to `SettlementPeriod` and `BillingCycle` (`docs/architecture/data-ownership/monthly-settlement-schema.md:77-82`). Installation, legal entity, currency (`ui-design.md:102`), issuance path and the recognition-versus-billing cadence are all candidate dimensions.
3. **Stage list.** At least five lists disagree: nine status axes (`ui-design.md:86`); the completion checklist source, allocation, calculation, approval, issuance, delivery (`U02/spec.md:66`); the schedule enum collection, calculation, approval, issuance, delivery, payment (`planning-actions.md:68`); HB-31's collection, verification, approval, issuance, delivery, payment (`requirements.md:557`); and the wireframe's five columns (`ui-design.md:238`). The frontend uses another list that adds accounting.
4. **State model.** Planning has no single closing-state enum. It offers completed, incomplete, exempt (`U02/spec.md:29`); a wireframe partition of complete, blocked, in progress, pre-activation (`ui-design.md:236`); proposed revision states open, provisional, closing, blocked, reconciled (`monthly-settlement-schema.md:78`); and an "Exempt or held" column with a blocked user state (`U02/spec.md:90`, `:117`). "Held" does not say which typed hold it means (`docs/contracts/operations/holds-onboarding-and-favorites.md:18-30`). The frontend's five-way partition matches none of these exactly.
5. **Calendar policy.** Holiday calendar, business-day and DST values, and the roles approving deadline changes and exemptions, are unresolved (`U02/spec.md:213-215`).
6. **Read permission and its source.** U02 names no read permission; `closing.read` appears only in the U01 specification.
7. **Flow order.** F02 and the end-to-end monthly order differ (D7, `docs/features/HB-72/screen-structure.md:241`); whether D7 has since been resolved was not verified.

# What exists today

| Input a closing target needs | Producer in this repository |
| --- | --- |
| Contract and billing profile, contract cycle and timezone | None; planning assigns them to the contract domain, O03 (`ui-design.md:107`) |
| Organization and legal entity as billing parties | None; `hb_focus.tenant` and audiences authorize FOCUS reads only |
| Stage schedule, calendar revision | None |
| Scope and permission source for closing objects | None; the local reader runs as one process-configured actor and resolves FOCUS audience membership only (`modules/focus_import/overview_http.py:20-23`, `receipt_projection.py:40-52`). Planning lists identity and permissions (HB-03) among the prerequisites of HB-30 (`requirements.md:547`), and HB-31 depends on HB-30 (`requirements.md:565`). Whether the customer-access grant types in `modules/customeraccess` could serve was not examined |
| Collection outcome | Partial: operator-registered expectation windows per connection and delivery declarations (`modules/focus_import/connections.py:158-191`, `receipt_metadata.py:316-351`) |
| Allocation, calculation, approval, issuance, delivery, ERP outcome | None (`docs/features/HB-110/approvals.md:12`, `docs/features/HB-111/claims.md:12`) |
| Business holds, exemptions | None |

Migrations contain no table for periods, cycles, closing, calendars, stage schedules, contracts or organizations (grep of `migrations/`). No route contains "closing" (`modules/focus_import/overview_http.py:70`).

The collection evidence cannot stand in for closing targets. It is keyed by connection, delivery window and FOCUS tenant, not by contract or cycle; its completeness is operator-asserted and marked unverified upstream (`connections.py:185`, `overview.py:138`); the due time computed from window end plus lag (`connections.py:173-182`) is dropped from the receipts projection; and nothing maps a connection or FOCUS tenant to a contract. Those windows already have an honest home in the S03 completeness view.

**Consequence.** No closing target can be produced from current data without inventing contracts, cycles and deadlines. A producer needs two things that do not exist: a registry of contracts and cycles, and a scope and permission source for closing objects.

# Alternatives

| Option | What it gives | Assessment |
| --- | --- | --- |
| A. Derive targets from collection windows | Rows today | Rejected: wrong identity, deadlines nobody registered, collection completeness presented as closing state |
| B. Fix the read contract and bind U02 as a consumer with no producer | An honest screen and a stable target for the producer | Possible only after the grain, key, stage and state decisions, because a strict consumer encodes them; it also changes the U02 normal view, which the owner has not chosen |
| C. Closing module with operator-registered, sample-labeled cycles | A producer for identity, deadlines and counts without waiting for the contract domain | Departs from planning ownership (O03 owns contract cycle and timezone; synthetic contracts are not copied into operations, `ui-design.md:171`). Acceptable only as explicitly non-authoritative sample data with the limits stated under decision 7 |
| D. Wait for the contract domain | No interim work | Leaves U02 showing example targets |
| E. Minimal contract-and-cycle registry in the contract domain first, then the closing module on top | The ownership planning describes | Largest scope; needs its own design and the scope and permission source first |

Root's recommendation is E as the direction and B as the first visible step once the decisions are made, with C only if the owner wants a producer before the contract domain exists. This is a recommendation, not a decision.

# Proposed read contract

`GET /api/closing-targets`, read-only, with the same local-client header, no-store and response-size bound as the existing readers. Most of this contract depends on open decisions and is provisional. Nothing in this section may be frozen into a strict parser before its decision is made.

| Part | Depends on decision |
| --- | --- |
| Record grain, count unit, sort key, cycle-level flags, calendar bucket unit | 1 |
| Key dimensions, `cadence` and its values, `currency`, `path` and its values | 2 |
| `stage` values, `current_stage`, which schedules exist | 3 |
| `completion` values, `stage_state`, flags instead of a partition, `holds[]` and what a hold attaches to | 4 |
| Scope list, `permission_revision` coverage, which fields are restricted and to whom | 5 |
| `source_mode` values | 7 |

Not dependent on those decisions: read-only reply with no commands, server-side scope resolution, snapshot and cursor mechanics, non-disclosure rules, error codes, and the 25-record page.

**Grain (provisional).** Proposed: one record per ClosingCycle, because planning states completion, exemption and denominators per cycle, with the cycle's stage schedules as a nested list. The maximum number of schedules per cycle is set in the producer ticket and enforced at registration, so a page of 25 cycles stays inside the response bound and the list is never truncated silently. `total` and `counts` then count cycles. The alternative, one record per stage schedule with a separate `cycle_total`, is decision 1.

**Identity.** `cycle_id` and, for each nested schedule, `schedule_id`, `stage` and a schedule label. Identifiers are opaque, embed no label, and stay stable when labels change. Display names and sort positions are never keys.

**Cycle fields.**

| Field | Null | Meaning |
| --- | --- | --- |
| `contract_ref`, `organization_ref`, `legal_entity_ref` | legal entity may be absent | Each is an object stating `present`, `absent` (legal entity only) or `redacted`, with the reference only when present. `redacted` is returned whenever the caller lacks permission for that field or object, whatever the underlying value, so it reveals nothing about existence; no reader resolves references to labels today |
| `period_start`, `period_end`, `timezone` | no | Usage period, start inclusive and end exclusive, in the contract timezone |
| `path` | no | `internal` or `external` |
| `cadence` | no | Recognition or billing closing, kept as separate identities |
| `currency` | yes | A dimension only; the read carries no amount |
| `manifest_revision` | no | Fixed target manifest the cycle belongs to |
| `completion` (provisional) | no | `complete`, `incomplete`, `exempt` or `unknown`. `unknown` is returned whenever any input of the completion predicate has no bound producer, even if a bound input already shows unfinished work; that is a deliberate conservative choice, and the known part is carried by `blockers[]` and the schedules' `stage_state` |
| `current_stage`, `next_due` (provisional) | yes | The stage and current due value of the earliest applicable schedule whose `stage_state` is not complete, ties broken by `schedule_id`; both null when no such schedule exists and for complete and exempt cycles. They supply the list's stage and remaining-time columns and the sort key |
| `exemption` | null unless exempt | Null exactly when `completion` is not `exempt`. When exempt it is an object that is either the reason, authority and evidence references or, for a caller without the field permission, `redacted` |
| `blockers[]` | may be empty; maximum set in the producer ticket | Codes such as `calendar_policy_missing`, `erp_outcome_unknown`, `approval_stale`, `outcome_unconnected` |
| `holds[]` (provisional) | may be empty | Typed holds as the holds contract names them (`billing`, `delivery`, `collections`); releasing one clears no other. Holds attach to a manifest or partial scope in that contract, so how a hold maps to a cycle is part of decision 4 |
| `owner` | see meaning | Personal data: an object stating `assigned`, `unassigned` or `redacted`, with the reference only when assigned. `redacted` is returned whenever the caller lacks the owner field permission, whether or not an owner is assigned |

**Schedule fields (nested).** `schedule_id`, `stage` (provisional, versioned enum), `label`, `applicable` (false when the stage does not apply to the cycle's path), `stage_state` (`complete`, `incomplete` or `unknown`, from that schedule's completion predicate), `original_due` and `current_due`, `schedule_revision`, `calendar_revision` (null when the policy is missing), `change_pending`, and `due_state`.

A due value carries the UTC instant together with the stored local date-time and IANA zone it was registered with; local values are stored, not recomputed from the instant, so later timezone-data changes do not move a deadline. If timezone rules later change so that the stored instant and local value disagree, the cycle gets a blocker such as `timezone_rule_changed` naming the schedule, and the schedule is reviewed; neither value is rewritten silently. This document proposes that a schedule cannot be registered without an explicitly entered deadline, and that a missing calendar policy is then reported as a blocker beside the entered deadline and never produces a computed or default date. That reads the planning rule "blocks rather than substituting a date" as blocking completion, not registration, and is itself a proposal.

`due_state` is evaluated in this order: `not_applicable` for an inapplicable schedule, a complete stage, or a complete or exempt cycle; then `unknown` when `stage_state` is unknown; then `overdue` when `stage_state` is `incomplete` and the current due instant is before `as_of`; otherwise `not_due`. A cycle is overdue when at least one of its schedules is overdue. Overdue therefore means known overdue. Because unknown must never be replaced by zero, the counts also carry `past_due_unknown`, the cycles that have an applicable schedule past `as_of` whose state is unknown, and a consumer shows the two together. Until stage outcomes have producers every `stage_state` is unknown, so the overdue count is returned as null, not zero, and `past_due_unknown` carries the cycles past their deadlines.

Blocked, held and overdue are flags derived from blockers, holds and due state. They are not members of a partition with completion.

**Query.** `period` is required and bounded to a maximum span set by the producer ticket. Optional: `cadence`, `stage`, `completion`, `path`, `owner`, `overdue`, `blocked`, `held`, `contract`, `organization`, `legal_entity`, `q`, `due_month`, `due_date`, `cursor`, `snapshot`, `permission_revision`. Unknown parameter names and enum values are rejected. `stage` selects cycles that have an applicable schedule in that stage, which need not be the cycle's `current_stage`; it narrows the cycles but does not trim the nested list, change the sort key, or restrict which schedules of a selected cycle feed the calendar buckets. `overdue`, `blocked` and `held` select only cycles that are neither complete nor exempt, the same population as their counts. `due_month` only switches on `due_dates[]` and does not narrow the list or the counts; `due_date` is rejected unless it falls inside the given `due_month`. For reference filters an unauthorized value and a nonexistent value give the identical reply. Filters and `q` never match a field the caller cannot read; an `owner` filter from a caller without the owner field permission behaves as a nonexistent value. `q` is matched only after the permission filter, has a minimum and maximum length and returns no suggestions. A submitted scope parameter confers nothing and is rejected.

**Snapshot, time and paging.** The snapshot pins both `as_of` and the data revision it reads, and is bound to the actor, the `permission_revision` and the selection without `due_month`, `due_date` and `cursor`, so one snapshot serves the calendar and every date chosen from it. Every page, `counts` and `due_dates[]` of that snapshot are evaluated at the pinned instant over the pinned revision, so neither a passing deadline nor a concurrent schedule change makes pages disagree with the counts. The cursor is an opaque keyset cursor over the stable sort key: `next_due` instant ascending with null last, then `cycle_id`. It is not a numeric offset, and the producer must materialize `next_due` to serve a keyset over 100,000 cycles. The snapshot is a server-side reference with a stated lifetime, not a digest of the whole result. An expired snapshot or cursor returns a distinct code and the client resyncs with a new query. A cursor requires both `snapshot` and `permission_revision`. Current authority is re-evaluated on every page: a changed `permission_revision` or selection returns `409 query_snapshot_changed`, so counts computed before a scope change are never shown beside later pages, and full revocation returns 403.

**Reply.** `schema`, `read_only: true`, `commands: "unconnected"`, `source_mode`, `auth_scope`, `permission_revision`, `snapshot`, `as_of`, an exact echo of the selection, `total` (nullable when not computed), `counts`, `due_dates[]`, at most 25 records, `next_cursor`, and a `request_id` on failures. `source_mode` is `synthetic` for sample-labeled registered cycles and `live` only for cycles from an authoritative contract registry behind a separately reviewed binding.

`total` is the number of cycles matching every filter including `due_date`, and is the size of the list being paged. `counts` are computed once per snapshot over the cycles matching every filter except `due_date`, so choosing a date does not change them: `counts.cycles` is that number, the four completion counts sum to it, and blocked, held, overdue and `past_due_unknown` are overlapping flag counts over cycles that are neither complete nor exempt. When `total` or `counts` is not computed it is null and no invariant is claimed.

`due_dates[]` is returned only when `due_month` is given and has at most 31 entries, also ignoring the `due_date` filter. A bucket is keyed by the stored local date of a schedule's current due value and covers applicable schedules of the cycles in the `counts` population whose local date falls in the month. Each bucket gives `cycle_count`, the number of distinct cycles with such a schedule on that date, and `schedule_count`. The `due_date` filter selects cycles having an applicable schedule due on that local date, so the list opened from a cell has exactly `cycle_count` rows. There is no viewer-timezone meaning: one cell can hold instants many hours apart, and list order stays by UTC instant.

**Errors.** The existing reader codes are reused: `400 request_invalid`, `403 scope_denied` and `origin_denied`, `405 method_denied`, `409 snapshot_required` and `query_snapshot_changed`, `503 source_busy`, plus the existing bounds codes for page and response size. New: a snapshot-expired code, and `429` when rate limited. `404` or `501` means no producer is bound. Full revocation returns 403, not 409. Error bodies echo no identifiers. A denied or failed read removes previously shown targets, counts and calendar.

**Authorization (provisional on decision 5).** The server resolves installation, workspace, customer, legal entity, contract and period scope from the authenticated context. `permission_revision` is proposed to cover grants, membership, scope assignment, field permissions and the authority epoch; the actual coverage follows the permission source chosen in step 0. Counts, calendar buckets and search include permitted cycles only. The read permission is proposed as `closing.read`, with owner and exemption evidence behind field-level permissions.

**Not in this read.** Amounts; a per-condition status surface for allocation, reconciliation and ERP outcome; and deadline and manifest history. Planning requires those axes not to be hidden, so a detail read is a required later contract, not an omission by choice.

# Constraints on the later schema

Stated now so that a schema ticket is not built against them; no table is designed here.

- The original deadline is write-once. Current-deadline changes are append-only revisions that keep the instant, the local date-time, the zone and the calendar revision.
- Manifest revisions are immutable. Additions, exemptions and holds each create a new revision and record the revision they applied to, so earlier denominators stay queryable.
- Exemption is a separate evidenced record, never an overwrite of completion. Reopening creates new revisions and edits nothing in place.
- Registration is idempotent on a natural key and conflicts on a changed payload.
- Every row carries the installation or workspace boundary. Nothing is keyed to FOCUS tenant keys.
- Recognition and billing cadence are separable identities.

# Screen consequences

Whether U02 changes before a producer exists is decision 6. If it does, the normal view would show that the closing service is not connected, and the example targets would be reachable by no URL parameter, window state or storage value, only by isolated QA interception; today the example mode is the default and a URL parameter selects scenarios (`apps/console/src/app/App.tsx:80-83`, `core-index-close/Screen.tsx:30`, `:42`). Against this contract the screen would also change its state presentation, paging, timezone display and filters. Those are screen design changes for a later U02 ticket with the ui-designer role.

# Owner decisions needed

| # | Decision | Root's recommendation |
| --- | --- | --- |
| 1 | Row grain and count unit: cycle, or stage schedule with a separate cycle total | Cycle, with nested schedules |
| 2 | ClosingCycle key and its relation to SettlementPeriod: which of installation, workspace, contract, usage period, cadence, legal entity, currency and issuance path are key parts | Workspace, contract, usage period and cadence as key; the rest as attributes. An assumption until confirmed |
| 3 | Canonical stage list for schedules; whether payment schedules appear in this read | The command contract's enum, declared versioned; allocation, reconciliation and ERP outcome as completion conditions shown in the detail read; payment excluded until its finance authority is defined |
| 4 | State model | Completion in four values including unknown, with blocked, held and overdue as flags. This departs from the wireframe's partition |
| 5 | Read permission and the scope and permission source for closing objects | `closing.read`; the source must exist before any producer and is a dependency, not something this module should invent |
| 6 | Whether U02 may change to "not connected" and lose its example targets before a backend exists | Yes, but it is the owner's choice; the owner chose backend design first when asked about U02 |
| 7 | Direction: contract-and-cycle registry first (E), or sample-labeled operator-registered cycles in a closing module (C) | E. If C is chosen: cycles are sample-labeled and non-authoritative, always `source_mode: synthetic`, never feed commands, notifications or jobs, are bound to a workspace and never to a FOCUS tenant, need a registration permission, and are purged or re-keyed when the contract domain exists |

Calendar values, approval roles and the D7 flow order stay open in planning. They are not needed for the shape of this read, but registering real schedules depends on the calendar decisions.

# Implementation sequence

Each step is its own ticket with design, independent review, verification and PR. Each step names the decisions it needs.

0. **Scope and permission source.** Decide and, if necessary, build the source of workspace scope and grants that closing reads will use. Needs decision 5 only. Dependency of steps 2 to 4.
1. **Contract, then consumer.** 1a: publish the decided contract under `docs/contracts/operations/`; needs decisions 1 to 5 and 7. 1b, only if decision 6 allows it: bind U02 as a consumer that treats 404 and 501 as not connected, rejects live payloads as producer-unbound and discards shown data on denial or error. The not-connected view follows the existing pattern of the approval inbox and claim history screens; any further layout change goes through the ui-designer role and screen review. Allowed files for 1b: the U02 page folder `apps/console/src/modules/operations/pages/core-index-close/`, the example-mode selection in `apps/console/src/app/App.tsx`, `apps/console/tools/closing-qa.mjs`. Acceptance for 1b: no example target through any URL, window or storage path; an actual 404 from the local reader verified; strict parsing unit-tested; three languages rendered.
2. **Registry, then closing schema.** Under direction E: 2a, a contract-and-cycle registry designed and owned in the contract domain; 2b, the closing schema on top of it. Under direction C: the closing schema with sample-labeled registered cycles under the limits of decision 7. Either way under the constraints above, with data and tenant review and no screen change.
3. **Read projection and endpoint.** Requires independent security review and these negative cases: cross-workspace targets and filter values; unauthorized filter value indistinguishable from nonexistent; `total`, `counts` and `due_dates[]` equal to counts over independently filtered permitted records; prefix search never matching unauthorized references; a cursor or snapshot replayed by another actor, after revocation and after scope narrowing; a submitted scope parameter rejected; redacted fields, including filters and search never matching a redacted field; error bodies echoing no identifier; sample cycles isolated per workspace. Expected results: a snapshot or cursor used by another actor, or with a changed selection or `permission_revision`, returns `409 query_snapshot_changed`; after full revocation 403; after the lifetime the expiry code. Count cost, snapshot lifetime and the maximum period span are set and measured here.
4. **Bind U02 to the producer.** Reviewed binding change, screen redesign, rendered verification. Completion stays `unknown` for every cycle whose predicate inputs have no producer.

# Independent review

Reviewer in both rounds: the repository's read-only `independent-reviewer` agent role (AI), run on 2026-11-12. It read files only and executed nothing. Root has not verified the reviewer's individual citations beyond applying the corrections.

| Review | Result |
| --- | --- |
| Architecture and data review of the first draft | FAIL, revise and re-review, 17 findings. The reviewer upheld the central conclusion and the rejection of option A. Findings included: the stage-schedule row grain presented as planning fact; a review claimed before it was recorded; a first step that froze undecided enums and preceded the backend; operator-registered cycles conflicting with planning ownership and lacking a tenant binding; no authorization source for closing objects; unspecified or unscalable time, snapshot and cursor semantics; error codes differing from the existing readers; unknown encoded as incomplete; non-null deadlines against the missing-calendar rule; incomplete calendar buckets and non-disclosure rules; incomplete planning citations; no constraints for the later schema; omissions in the key and stage decisions |
| Re-review of the second draft | FAIL, narrow revise, no blocker, two high findings. Of the 17 findings, 12 resolved and 5 partly resolved. New findings: overdue and due state not derivable at cycle grain; sort key and stage filter ambiguous; count units and invariants contradictory; the snapshot pinning time but not data; provisional parts not all marked; redaction inconsistent with the exemption and reference rules; the nested list unbounded; steps 1 and 2 needing a split; wording of this section; three smaller items |
| Narrow check of the third draft | PASS: no high or blocking finding, fit to merge as a draft design awaiting owner decisions. Three medium and five low one-sentence items: overdue reported as zero while states are unknown; the date filter against the snapshot binding; stage filter wording against the bucket population; precedence of due states; `next_due` on exempt cycles and ties; flag filter population and `due_month` behavior; `redacted` implying existence; small gaps in the provisional table and blockers |

The author then applied those eight items. That last set of corrections was checked by root only, not by a further review.

# Limits

No runtime behavior, performance, compatibility or permission enforcement is verified by this document. The evidence is a read of the current tree and the local planning inputs; whether a closing producer exists outside this repository is unknown. The recommendations are root's proposals and are not owner decisions.
