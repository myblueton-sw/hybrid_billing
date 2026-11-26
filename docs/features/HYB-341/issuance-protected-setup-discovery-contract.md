---
type: ContractDesign
title: Protected setup discovery for a selected issuance configuration
description: Bounded nonmutating reads of protected rules, verification channels and protected changes for one confirmed configuration of the selected contract.
status: draft
---

# Outcome and ownership

HYB-341, machine:mac_mini. This is item 1 of [the handoff](handoff.md). An operator who has confirmed a reusable issuance configuration in O03 documents must be able to see which protected rules, verification channels and protected reference changes already exist for it, including interrupted and uncertain work, before any protected write is exposed. The design adds three reads and two module observation functions. It adds no write, no grant, no evidence capture and no claim that issuance is ready.

Root owns the design and integration. An independent reviewer covers the new disclosure, the module observation policy and the tenant and permission boundary. Passed reviews of context, configuration history, options, action and the Inspect capability are reused, not repeated. Evidence capture (handoff item 2) and protected writes (item 3) stay separate; this contract must not be read as accepting them.

# Verified starting facts

These were read from main at c8aebd6.

- `financialIssuance` retains a single `protectedRules` record that carries its workspace, at most 100 `channels`, `changes` and the ordered `changeOrders` index keyed by subject and field (`financialissuance.go`).
- `issuanceSetupBoundary` refuses a non-console client. `protectedReadCaller` enforces the current human session, the protected domain Inspect capability with its Cost evidence field evaluated for the console client, no delegation and an owned applied human grant under a console ceiling. It skips only the hold refusal (`financialissuancesetupcontext.go`, `financialissuanceprotectedaccess.go`).
- `observeConfigurationHistory` validates a retained configuration from its frozen scope and a nonmutating kind lookup, without current governor standing (`financialissuanceconfigurationhistory.go`).
- `configuredChangeSubject` derives the three subjects from the configuration: `customer:<Scope.Contract.CustomerID>` for document_recipient, `entity:<Scope.Document.IssuerLegalEntityID>` for payment_account and `kind:<Kind.ID>` for document_template. `configuredReferences` consumes exactly these three fields. `configuredChangeSubject` reads current organization facts, so discovery derives the same strings from the frozen scope and does not call it (`financialissuancesetuporigin.go`, `financialissuanceconfiguredconsumption.go`).
- `currentChannel` and `changeStanding` require current actor rights, current organization facts and governor standing; `channelRules` requires current governor standing; `changeTrail` settles module state through `Trail` and `Of` and refuses any unowned module entry. They are write and consumption guards and are unsuitable for retained history.
- `protected.Registry.Of` and `Trail` settle open changes to lapsed in stored state. `RulesOf`, `ChannelOf` and `ChannelsOf` do not change records. Every module entry point may advance the registry's monotonic clock high-water mark (`modules/protected/registry.go`, `changes.go`).
- The module accepts a verification channel when `Admitted + Rules.ChannelAge` is not after the change's request instant, using the rules fixed into the change at request time (`changes.go`, the `ReasonChannelTooNew` check).

# Reads

All three commands are nonmutating, target the selected contract ID and require body fields `configuration` and `digest`. They use the permission already declared for `issuance_setup.context`: ViewCost on the protected resource kind with the protected evidence field. They are added to the setup boundary check, to the hold read allowance and to the synthetic sample restriction exactly like the four existing setup reads.

| Command | Additional fields | Returns |
| --- | --- | --- |
| `issuance_setup.protected` | none | Rules facts and one summary per protected field |
| `issuance_setup.protected_channels` | optional `after`, `snapshot` | Up to ten channel rows per page |
| `issuance_setup.protected_changes` | required `field`; optional `after`, `snapshot` | Up to ten change rows per page for one field |

Three commands are chosen over one command with a section selector so that each definition declares exactly its own fields and irrelevant fields are rejected by the command framework.

## Guard order

1. Sample installation and console client at the boundary, then `protectedReadCaller` at the boundary and again in Apply.
2. Request shape: valid contract and configuration IDs, a valid digest, a closed `field` value, and `after` and `snapshot` present together or both absent.
3. `configurationHistoryIndex`.
4. Configuration identity. An unknown ID, another workspace or another contract returns PermissionDenied with no existence distinction. For a configuration of the selected contract, a digest that differs from the retained digest returns EvidenceStale. A confirmed configuration recorded without setup context is readable, as it already is in `issuance_setup.configurations`; whether protected writes support such a record is decided with those writes.
5. `observeConfigurationHistory`. Any state other than retained `confirmed` with observation `active` returns PrerequisiteFailed. Protected subjects are derivable only from a confirmed configuration.
6. Section integrity and observation as defined below. Corruption, ambiguous ownership or a retained record that contradicts its module observation returns EvidenceStale with no partial rows. A lookup fault returns DependencyUnavailable and is never reported as absence. The module has no workspace record until rules are first proposed; `ChannelsOf` and `PeekTrail` then return not-found, which is an empty observation (no module channels, an empty trail) and not a fault.
7. Output bounds.

Current contract metadata is not required; retained history stays readable when today's metadata or organization facts have changed. The reader's own current authority is always required.

## Three separated layers

Every row and the rules block keep these apart.

- Retained state is what the service recorded: the rules or channel state, or the change phase together with the last retained module state.
- Observation is what the module holds now, read without changing it. Every completed-act field (confirmer, admitter, verifier, decisions and their instants) is taken from the observation. An actor the service pinned before a module effect that the module does not yet show appears only in a separate pending field. The first-step actors (`rules_proposer`, channel `registrar`, change `requester`) and the values fixed by the retained request (`effective`, `provenance`, `instruction_kind`, `channel`) come from the retained record, because they identify who may retry an unknown first step; `requested_at` and `approvals_required` are empty while the observation is `absent`. When the module holds a record, it must equal the retained module view, apart from the lapse projection of an open change and the retirement of a channel.
- Current action eligibility is not evaluated. Every response carries `current_eligibility` with value `unchecked`. No field is named ready, eligible, mature or usable.

An observation never promotes a retained unknown state. A recorded historical actor who has since lost a right, a session or governor standing does not hide or alter the row. Whether the calling operator may act is the subject of a later action projection and of the write itself.

# Module observation policy

Two functions are added to `modules/protected`.

- `Peek(workspace, id)` returns the view of one change.
- `PeekTrail(workspace, subject, field)` returns the views of one subject and field in insertion order.

Both return detached copies. An open change whose deadline has passed is returned as lapsed with the same `Ended` instant that settlement would record, computed on the copy. Stored changes, indexes, channels and rules are not modified. Input validation and missing-workspace behavior match `Of` and `Trail`. Like the already reviewed `ChannelsOf`, they take the registry lock and may advance the monotonic clock high-water mark; they have no settlement or record effect. With a non-decreasing clock this has no effect on later writes; if the clock regresses, a read pins later module timestamps exactly as any existing module read does. `Of`, `Trail` and every write keep their current behavior. Once the high-water mark passes a deadline, a stored open change and a stored lapsed change behave identically, so the absence of settlement is proven by in-package module tests and not at the server level.

The projected view is chosen over the raw stored view because an open change past its deadline can never proceed, and showing it as open would misstate history. It is chosen over calling `Of` or `Trail` because a discovery read must not settle state that only a write or consumption path is entitled to settle.

# Rules block

`issuance_setup.protected` returns the workspace rules that the service owns.

| Retained state | Accepted observation | Condition |
| --- | --- | --- |
| none | `absent` | Module has no workspace record |
| none | `unowned` | The module lookup succeeds while the service retains no rules; no rule values are returned |
| `registration_unknown` | `absent` or `pending` | `absent` requires an empty retained module view; `pending` requires the single module proposal to equal the retained module view |
| `proposed` | `pending` | Exact retained module view, no confirmer |
| `confirmation_unknown` | `pending` or `active` | Exact retained module view in both cases; `active` requires the pinned confirmer |
| `confirmed` | `active` | Exact retained module view and confirmer |

Validation uses the retained digest, single-entry module history, revision 1 and no discard, as `protectedRulesModule` does, but without governor standing. Any other combination is EvidenceStale.

Fields: `rules_state`, `rules_observation`, and when the service owns rules: `rules_id`, `rules_digest`, `rules_approvals`, `rules_validity_seconds`, `rules_max_lead_seconds`, `rules_channel_age_seconds`, `rules_proposer`, `rules_confirmer`, `rules_pending_confirmer`, `rules_proposed_at`, `rules_confirmed_at`. `rules_confirmer` and `rules_confirmed_at` come from the observed module view. `rules_pending_confirmer` is the pinned confirmer while the module still shows the proposal as pending. Absent values are empty strings. The proposal reason is not returned.

# Field summaries

The same response carries one group for each of `document_recipient`, `document_template` and `payment_account`, with field name prefixes `recipient_`, `template_` and `account_`.

- `subject_kind`: `customer`, `document_kind` or `entity`. The subject identifier itself is not returned; it is derived, not operator input.
- `changes`: number of owned changes for this configuration in retained order.
- `other_provenance`: number of retained changes under the same subject and field that belong to another origin.
- `unowned_module`: number of module trail entries that no retained change owns.
- `latest_scheduled` and `latest_scheduled_effective`: the last owned change whose phase is `recorded` and whose retained state and observation are both `scheduled`, and its effective instant. A change in an unknown phase is never named here. The value ignores other-provenance and unowned entries, which the counts beside it report. It is a historical fact, not the result of `Resolve`, and does not state that a draft can consume it.

`tax_identifier` is not reported. The supported synthetic kind is nonstatutory and nothing consumes that field.

The response also carries `schema` (`hb_issuance_protected_setup_v1`), `contract`, `configuration`, `digest`, `channels` (the number of channel rows in scope), `channels_other_provenance`, `channels_unowned_module` and `current_eligibility`. The total is 39 fields: 8 in the envelope, 13 for rules and 18 for the three summaries, inside the established 64-field minimum.

# Channel rows

Scope is the set of retained channels whose retained facts equal the configuration's frozen facts. For every kind the record's workspace equals the configuration's workspace, and the retained workspace snapshot and its version equal the frozen ones. In addition:

- customer: the retained contract revision deep-equals the frozen contract and the subject equals the frozen customer subject;
- entity: the retained billing profile and version equal the frozen profile and version, and the subject equals the frozen issuer legal entity subject;
- document kind: the selector's configuration reference equals this configuration and the retained configuration snapshot deep-equals it.

Customer and entity channels carry no configuration reference. They are shared by every origin that froze identical facts, including another configuration of the same contract revision or billing profile version, which is exactly the set a protected change of this configuration may use. Only a document-kind channel belongs to one configuration. The reader's Inspect right is workspace-wide, so this sharing is not a wider disclosure.

A retained channel with the same subject string and different provenance is counted in `channels_other_provenance` and is not a row. Equal subject strings do not prove the same provenance. Module channels returned by `ChannelsOf` for the three subjects that no retained record owns are counted in `channels_unowned_module`; their contents are never returned and they are never adopted.

Every retained channel on one of the three subjects, including one counted as other provenance, is validated by its retained digest, selector and subject correspondence and spec shape, then observed with `ChannelOf`. A corrupt retained channel of another origin on the same subject therefore refuses the read; it is not silently counted.

| Retained state | Accepted observation |
| --- | --- |
| `registration_unknown` | `absent` only with an empty retained module view; otherwise `registered` |
| `registered` | `registered` |
| `admission_unknown` | `registered`, or `admitted` by the pinned admitter with the pinned admission evidence; the module view equals the retained view in both cases |
| `admitted` | `admitted` |
| any state with a retained module view | `retired` when the module view equals the retained view except for retirement |

Row fields: `channel`, `digest`, `subject_kind`, `kind` (email, phone, portal or postal), `state`, `observation`, `registrar`, `admitter` (from the observed view), `pending_admitter` (the pinned admitter while the module shows no admission), `registered_at`, `admitted_at`, `retired_by`, `retired_at`, `rules` (`current` when the retained rules digest equals the service's retained rules digest, otherwise `other`) and `request_not_before`.

`retired_by` is an actor value written by a module effect the service does not own; the service has no retirement operation. It is shown because channel writes already treat retirement as a distinct named refusal and the operator needs to see why a channel is unusable.

`request_not_before` is `Admitted + ChannelAge` and is filled only when the retained state is `admitted`, the observation is `admitted`, the row's `rules` value is `current` and the rules block is `confirmed` with observation `active`; otherwise it is empty. It states the earliest module request instant at which this channel satisfies the age rule under the current rules; the module compares against its own request instant, not wall time. It is kept although it is derivable from two returned values, so that clients do not reimplement the rule. Waiting after an earlier request does not repair that request, and the value does not say that the calling operator may request or verify through this channel. Custodian, self-verification and same-channel exclusions are not evaluated here.

Not returned: the protected reference ID and fingerprint, registration and admission evidence IDs, the rationale, and any contact or account content. Handoff item 2 decides which evidence labels a person may see; until then no evidence identifier is disclosed.

Rows are sorted by channel ID. `count` is the exact number of rows in scope.

# Change rows

`field` selects one of the three fields. Scope is the ordered `changeOrders` list for the configuration's subject and that field. The index is checked in both directions as `changeTrail` does: no duplicate entry, every listed ID retained with matching workspace, subject, field and digest, and no retained change for that scope missing from the list.

A retained change whose configuration snapshot deep-equals this configuration is a row. Others are counted in `other_provenance`. Each retained change in the list, in either group, is observed against `PeekTrail` in order:

- `validChangeProgress` must hold for the retained phase;
- a retained `request_unknown` with no retained module view must be the last entry, and `Peek` by its ID must return not-found; it reports observation `absent`. If the module holds that ID anywhere, the read is EvidenceStale;
- every other retained entry is matched positionally: the n-th retained entry with a module view must be the n-th module trail entry, with the same change ID. It must match by the existing rule: identical, or the exact requested or confirmed to lapsed projection, with unchanged change, requester and prior reference;
- an unowned entry is a module trail entry whose ID no retained change in this scope holds. Unowned entries are accepted only after every retained entry; they are counted in `unowned_module` and returned in no row. An unowned entry before or between retained entries is EvidenceStale. That position is unreachable through service writes, because `changeTrail` refuses every further write once an unowned entry exists;
- a retained change that the module shows as withdrawn is EvidenceStale. The service has no withdrawal operation, and accepting that foreign effect would need an equality rule beyond the reviewed lapse projection.

Counting trailing unowned module entries, where `changeTrail` refuses, is deliberate: the operator must be able to see that an unowned record exists and blocks later writes, without the service adopting it or disclosing its content.

Row fields: `change`, `digest`, `field`, `phase` (`recorded`, `request_unknown`, `verification_unknown` or `decision_unknown`), `state` (the retained module state, empty when none is retained), `observation` (`absent` or the projected module state), `provenance` (the closed provenance value), `instruction_kind`, `channel` (the verification channel ID), `rules` (`current` or `other`, as for channels), `requester`, `requested_at`, `effective`, `approvals_required`, `verifier`, `verification_outcome`, `verified_at` and `decisions` (at most three entries of approver, verdict and instant), all taken from the observed view, `pending_verifier` (the pinned verifier while the observed view has no verification), `pending_decision_actor` (the pinned approver when the retained decisions exceed the observed ones), `ended_by` and `ended_at`.

Not returned: the prior and new reference IDs and fingerprints, the instruction sender fingerprint and evidence, verification evidence, digests of module requests and verifications, rejection reasons and rationales.

Rows keep retained order, which is the order in which requests were reserved. `count` is the exact number of rows in scope.

# Paging, snapshots and bounds

The summary response prefixes the channel counts (`channels_other_provenance`, `channels_unowned_module`) because it also carries per-field counts; a page names its own counts `other_provenance` and `unowned_module`.

Each paged response returns `schema` (`hb_issuance_protected_channel_page_v1` or `hb_issuance_protected_change_page_v1`), `contract`, `configuration`, `digest`, `current_eligibility`, `snapshot`, `count`, `next`, `other_provenance`, `unowned_module` and `row_00` to `row_09`. Change pages also return `field`.

The snapshot digest binds workspace, reader, contract, configuration ID and digest, the section, the field for changes, the retained rules record and its observation, every retained record in scope with its observation, the serialized rows and the `other_provenance` and `unowned_module` counts. A first page carries neither `after` nor `snapshot`. A continuation requires a matching snapshot and an `after` that is a member of the row order. A changed snapshot or an unknown cursor returns EvidenceStale. Because a lapse projection depends on the module clock, a deadline passing between pages changes the snapshot; the client starts again from the first page.

Each row serializes to at most 4096 bytes and the field map to at most 65536 bytes through `boundedDraftFields`. A transport below 64 fields or 4096 bytes per value returns UnsupportedCapability. More than 100 retained channels or changes, or an oversized row or response, returns PrerequisiteFailed. No truncated count or partial page is returned.

# Effects

The reads create no identifier, reserve nothing, reconcile no unknown phase, change no retained record, order entry, module record, grant or server revision, and are permitted during installation hold. The only tolerated module effect is the monotonic clock high-water mark described above.

# Alternatives not chosen

- Reusing `currentChannel`, `changeStanding` and `changeTrail` for discovery would hide history whenever an original actor, governor or organization fact changed, and `changeTrail` settles module state.
- Returning one aggregate response for everything cannot stay inside the field and size bounds once a configuration has many changes, and would force partial data.
- Evaluating maturity, actor separation or resolvability in these reads would duplicate write-time rules and present a point-in-time guess as readiness.
- Disclosing reference fingerprints or evidence IDs now would fix a disclosure contract before the evidence-capture contract exists.

# Required verification

Actual command HTTP tests with independently approved human and console Inspect grants. Retained states come from real private writes and their fault points. Module-only rules, channels and trail entries, a retired channel, a withdrawn change and corrupt index entries cannot be produced by service writes; package-internal tests create them by direct module calls or retained-map edits. Lookup faults use a new fault point, `issuance-setup-protected-lookup`, in front of every module read these commands make.


- reader: missing Inspect, revoked human grant, revoked console ceiling, ended session, delegated inspection, administrator without Inspect, non-sample boundary, held read allowed;
- configuration: unknown ID, another contract's configuration, wrong and malformed digest, an unconfirmed configuration, a confirmed record after profile suspension and after original governor revocation;
- rules: none with absent module, none with unowned module rules, each retained state before and after its module effect, lookup fault distinct from absence, original proposer and confirmer revoked;
- channels: all three subject kinds, each retained state at all six interruption boundaries, retired channel, same subject with other provenance, module-only channel, registrar and admitter revoked after capture, a reader who is the channel's registrar receiving the same row with `current_eligibility` `unchecked`, `request_not_before` exactly equal to admission plus channel age and empty in every non-admitted case. The service retains one immutable rules record, so a row with `rules` `other` or an admitted channel under unconfirmed rules cannot be produced by a fixture and is not tested;
- changes: each field, all nine interruption boundaries, unknown phases left unchanged when the module has progressed and never named as `latest_scheduled`, requested and confirmed changes past their deadlines shown as lapsed with unchanged retained records, scheduled history, a module-only trail before any owned change and after an owned change, a module-withdrawn retained change refused, orphan and duplicate order entries, changed retained state, requester, verifier and approver revoked after capture;
- paging: eleven or more rows, a second reader's cursor, a snapshot changed by a write and by a lapse, an unknown cursor, reduced transport budget. Row and response overflow are guarded but cannot be reached by a fixture: identifiers are at most 128 bytes, so a row stays near 2 KB and ten rows stay under the response bound;
- effects: unchanged retained maps, order index, module rules and server revision after every read, including held reads.

Actor separation (self, custodian and same-channel exclusions) and the module's acceptance at exactly `request_not_before` and refusal one nanosecond earlier are write-time behavior. Existing change tests cover them (`TestFinancialProtectedChangeTooYoungNoEffect`, the configured setup fixture's exact 24-hour request) and the public write contract of handoff item 3 must cover them again; this read only reports the instant.

Not covered by a fixture: a retained change of another origin under the same subject and field. No existing fixture creates two origins on one subject and field, so the `other_provenance` change count is verified only as zero.

In-package module tests for `Peek` and `PeekTrail`: detached copies, identical output to `Of` and `Trail` for settled and non-open changes, projected lapse at the deadline boundary and one nanosecond after it for both deadline kinds, stored state unchanged after observation with a following `Of` performing the first stored settlement, and input and missing-workspace refusals.

# Limits

This contract discloses retained protected setup for the synthetic in-memory boundary only. It does not expose protected writes, evidence capture, payment terms, access provisioning or current action guidance, and it does not establish the unseeded operator journey. The O03 presentation of these reads needs its own scoped design review.

# Design review

The independent-reviewer agent (read-only, model reported as claude-fable-5-2) reviewed the first draft against the code at c8aebd6 and returned FAIL with six blocking findings: ambiguous trail matching with interleaved unowned entries; `latest_scheduled` able to name a change in an unknown phase; unspecified source layer for completed-act fields in unknown states; unspecified handling of a missing module workspace record; a withdrawn-history test contradicting the match rule together with a wrong field count; and incomplete channel provenance rules. Reader authority, non-disclosure, the module observation policy and hold behavior passed in that first review.

Root corrected all six in one pass and adopted the reviewer's non-blocking corrections to the starting facts, the clock assumption, the `request_not_before` conditions, the snapshot binding, the named fault point and the fixture description. The bounded corrective re-review returned RESOLVED for all six findings, judged the removal of the legacy refusal safe, found no new blocking defect and gave an overall design PASS. Its three minor clarifications (retained first-step actors, the reduced test wording and the count naming) are included above and were not sent for another review.

The reviewer executed no tests and did not open the sibling HYB-341 contract documents. Its unverified hypotheses remain open and are not addressed here: unkeyed retained and snapshot digests may let a reader confirm a guessed low-entropy hidden input, which is the same exposure as the existing configuration digest; and whether the test clock can regress. This record is a design review. It is not a source review of the implementation.

# Implementation and verification

Root implemented the contract in `modules/protected/observe.go` (`Peek`, `PeekTrail`), `apps/api/server/financialissuancesetupprotected.go` (guards, observations, paging and the three projections) and the registration points in `financialissuancesetupcontext.go` and `trusthold.go`. A subject longer than the module's 128-byte identifier limit can hold no channel or change, so it is treated as an empty observation without a module call.

The independent-reviewer agent (read-only, model reported as claude-fable-5-2) reviewed the working tree. Conformance, authority and disclosure, nonmutation, correctness and registration passed with no code defect. Tests failed the review: the three `absent` observations, every populated pending-actor field, an unowned trail entry before an owned one, the revoked console ceiling, delegated inspection and the registrar-as-reader row had no assertion. Root added those cases in one pass and adopted the non-blocking suggestions: request shape is checked before configuration identity, a module-withdrawn change is refused explicitly, the lookup fault point precedes `Peek`, and the boundary matches the full command name. The bounded corrective re-review returned PASS for the added tests, the fixture fidelity of the retained-map edits, the code edits and the document's consistency with the code, with no blocking finding. It noted that the delegation test proves that delegation alone cannot read; it cannot isolate the delegation check from the owned-grant check.

Executed on the final working tree: `go vet ./apps/api/... ./modules/...` clean; `go test -race ./modules/protected` PASS; all `TestFinancialSetupProtected` server tests PASS; the full `go test ./apps/api/... ./modules/...` run passed 80 packages and failed `apps/api/server` (449.694 seconds) on exactly two tests, `TestAccessPreview` and `TestProposalStoresCoverEveryProposingNamespace`. Both fail identically on unmodified main at ffcf4bc and are tracked separately; no full-suite PASS is claimed. The reviewer executed no tests.

