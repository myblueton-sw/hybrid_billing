---
type: ContractDesign
title: Public payment terms and applicability bindings for a selected contract
description: A bounded read and four public commands for calendar payment terms and their contract bindings under a confirmed issuance configuration.
status: draft
---

# Outcome and ownership

HYB-341, machine:mac_mini. This is item 4 of [the handoff](handoff.md). An operator who has confirmed a reusable issuance configuration can record explicit calendar net days for it, have a second person approve them, bind the approved term to the contract for an explicit date range, and have that binding independently approved. Later invoices reuse the approved binding. The design adds one read and four commands that wrap the private term functions already implemented and tested. It adds no default payment days, no business-day calendar, no due-date calculation and no claim that an invoice can be issued.

Root owns the design and integration. An independent reviewer covers the new disclosure, the mutations, the permission mapping and the effect semantics, because payment terms set invoice due dates. The [discovery contract](issuance-protected-setup-discovery-contract.md) and the [protected writes contract](issuance-protected-writes-contract.md) are reused for the patterns they already settled and are not repeated.

# Requirement basis

`docs/planning/v1.16/payment-terms.md` requires per-contract payment conditions of N days from the invoice issue date, with a term identifier and revision, net days as a whole number of zero or more, calendar or business days, the contract time zone, an effective start and end, and a change reason and approval history. It states that 30 and 45 days are examples and not a default for every contract, that calendar days from the issue date are the proposed first mode, and that business-day and holiday adjustment must not be applied until a calendar and calculation rule are agreed. It also requires that a contract change does not recalculate documents already issued. The linked requirements it names are HB-15, HB-22, HB-24 and HB-27 to HB-31.

This contract covers recording, approving and binding a term. What it does not cover is stated here so that nothing is narrowed silently:

- Planning also lists for a term a display name, the basis field `invoice_issue_date`, and a holiday adjustment policy with a calendar version. None is recorded by this slice. The basis is the issue date by construction of the existing coverage calculation; a display name is left to the O03 view and its terminology review; holiday adjustment is not offered at all.
- Planning leaves the maximum net days undecided. The range 0 to 365 is the bound the existing service and module already enforce, a provisional value and not a planning decision.
- The handoff asks for an exact term revision. The retained term carries revision 1 and the read returns it; a different net-days value is a new term under a new configuration, not a second revision of the same term.
- Due-date snapshots on issued documents, extensions, overdue status and collection are outside this contract.

# Verified starting facts

Read from main at 44f9ee3; later merges to main changed only a console client and the handoff document.

- The private functions are `recordTerm`, `approveTerm`, `bindTerm` and `approveTermBinding`. Each begins with `termCaller` for its own capability and has before-module, after-module and after-publication fault points.
- The four capabilities map to four closed direct-human gate actions on the terms resource kind, each needing an independently governed console ceiling and a human grant. A separate Inspect capability with its own evidence field exists for reads, through `termReadCaller`, which skips only the hold refusal. None of these is the protected Inspect right.
- `recordTerm` accepts only the calendar mode, net days from 0 to 365, a revision from 1, an identifier and a reason. It derives a service-owned module term identifier, so an unowned module record cannot be adopted.
- A term is owned by a scope. For a configured term the scope is the configuration itself, and `recordTerm` refuses a second term for an occupied scope. One configuration therefore has at most one term record, with one revision and one net-days value.
- `bindTerm` requires an approved owned term, dates in ISO form with an optional end not before the start, one of the closed reasons `initial`, `renegotiation` and `correction`, and a zone equal to the configuration's zone. It derives the contract from the configuration. It refuses a new binding whose date range overlaps any retained binding of the same contract, whatever that binding's state or origin; an open-ended binding counts as reaching 9999-12-31.
- The module's `Bind` additionally refuses a binding whose zone differs from the zone of any non-withdrawn binding of the same contract. `bindTerm` does not check that before it reserves. Two configurations of one contract can name different zones, so this refusal is reachable after a reservation.
- Terms and bindings created for a source profile, without a configuration, exist in the same retained maps and use the same contract identifiers. Their bindings take part in the overlap rule.
- `approveTerm` and `approveTermBinding` refuse the author with SelfApprovalDenied and pin the approver before the module call.
- Retained states are `registration_unknown`, `pending`, `approval_unknown` and `approved` for both records.
- The module reads `Term` and `BindingOf` return copies and change no record. They may advance the module clock high-water mark like every module call.
- The module supports withdrawing and ending a binding. The service calls neither.
- On a first attempt each private function writes its retained record before it calls the module; on the retry of an owned unknown record it calls the module again with no retained change before that call. After a module effect the fault points return a named DependencyUnavailable failure, and module errors are returned unwrapped.
- No forward deadline exists in these functions or in the module functions they call. The standing checks are time dependent, in that the configuration must already be active and sessions, grants and governor standing must be current, but all of them run before the reservation and so refuse cleanly.
- Later operations on a term or binding recheck the current standing of its recorded author and pinned approver, and of the configuration's proposer and confirmer.
- The setup boundary today requires the protected Inspect right for every setup read and for the configuration mutations. Term authority is a different domain.
- The draft context read already evaluates coverage for a given instant through the existing consumption path. No public command records or approves a term today.

# Read: issuance_setup.terms

A nonmutating command targeting the selected contract with `configuration`, `digest` and optional `after` and `snapshot`. Its permission is ViewCost on the terms resource kind with the terms evidence field. The setup boundary gives it its own branch: console client, sample installation, then `termReadCaller`. The protected Inspect right is neither required nor sufficient. It is allowed during hold. The configuration identifier and digest it needs come from `issuance_setup.configurations`, which requires the protected Inspect right; in practice a terms reader or writer therefore also needs that right to find a configuration, although this command's own gate does not ask for it. A terms reader can also tell an existing configuration from an unknown one by the wrong-digest refusal, which is accepted because terms authority is workspace-wide.

Guard order: reader authority; request shape; the configuration checks of the discovery contract without its protected reader check (unknown or foreign configuration PermissionDenied, wrong digest EvidenceStale, unconfirmed PrerequisiteFailed); integrity and observation; bounds. Corruption or a retained record that contradicts its module observation is EvidenceStale with no partial rows. A lookup fault is DependencyUnavailable and is never reported as absence; a new fault point `issuance-setup-terms-lookup` precedes every module read this command makes.

As in the discovery contract, the retained state, the module observation and current eligibility are separate. A completed act is shown from the observation, a pinned actor the module does not yet show appears only as a pending field, and `current_eligibility` is always `unchecked`. The read evaluates no coverage for any date and reports nothing as ready, due or applicable. A recorded author or approver who has since lost a right does not hide a row.

## Term block

The configuration's term, validated by its retained digest and scope ownership and observed with `Term`.

| Retained state | Accepted observation |
| --- | --- |
| none | `absent` |
| `registration_unknown` | `absent` only with an empty retained module view; otherwise `pending` |
| `pending` | `pending` |
| `approval_unknown` | `pending`, or `approved` by the pinned approver |
| `approved` | `approved` |

When the module holds the term it must equal the retained module view. Fields: `term_state`, `term_observation`, and when a term is retained: `term`, `term_digest`, `term_revision`, `net_days`, `mode`, `term_author`, `term_approver`, `term_pending_approver`, `term_written_at`, `term_approved_at`. The reason is not returned.

## Binding rows

Scope is every retained binding whose contract is the configuration's contract, because every one of them decides whether a new date range overlaps and which zone the contract is held to. Each row carries `origin`: `configuration` with the identifier of the configuration it was made under, or `profile` with an empty configuration for a binding made for a source profile. Bindings of other contracts are never rows.

Each binding is validated by its retained digest, and its embedded term must deep-equal the retained term it names and be approved, as the draft context already requires. It is observed with `BindingOf` under the same state table as the term. A module binding that has been withdrawn, has a pending end or has recorded ends is EvidenceStale: the service has no such operations and no rule for accepting those effects.

Row fields: `binding`, `digest`, `origin`, `configuration`, `term`, `term_revision`, `net_days`, `mode`, `zone`, `from`, `to`, `reason`, `state`, `observation`, `author`, `approver`, `pending_approver`, `written_at`, `approved_at`. Rows are sorted by binding identifier, ten per page, with the snapshot, cursor and size rules of the discovery contract. The snapshot binds the reader, the scope, the term block and every row.

Field sources follow one rule. Identifiers, digests, net days, mode, revision, zone, dates, reason and the author come from the retained record. The approver and both instants come from the observed module view and are empty while the observation is `absent`. The pending approver is the retained pinned approver while the module shows no approval. The handler derives the module identifiers for its observations exactly as the private functions do.

The response returns `schema` (`hb_issuance_setup_terms_v1`), `contract`, `configuration`, `digest`, `zone` (the configuration's document zone, which a binding must use), `current_eligibility`, the term block, `snapshot`, `count`, `next` and the rows.

# Commands

All four are mutating, target the selected contract and take `configuration` and `digest`. They are refused during hold and are subject to step-up like every mutating command.

| Command | Static permission on the terms resource kind | Further fields | Declared refusals |
| --- | --- | --- | --- |
| `issuance_setup.record_term` | record issuance term | `term`, `net_days`; rationale required | EvidenceStale |
| `issuance_setup.approve_term` | approve issuance term | `term`, `term_digest` | EvidenceStale, SelfApprovalDenied |
| `issuance_setup.bind_term` | bind issuance term | `binding`, `term`, `term_digest`, `from`, optional `to`, `reason` | EvidenceStale |
| `issuance_setup.approve_term_binding` | approve issuance term binding | `binding`, `binding_digest` | EvidenceStale, SelfApprovalDenied |

- `term` and `binding` are identifiers generated by the console, at most 128 bytes.
- `net_days` is a whole number from 0 to 365 in canonical decimal form. No default is supplied and no value is suggested by the server.
- The mode is always calendar days and is not a field. Business-day terms are not offered, because planning requires an agreed calendar first.
- The revision is always 1, because a configuration holds exactly one term.
- `from` and `to` are ISO calendar dates. An empty `to` is an open-ended binding.
- `reason` is `initial`, `renegotiation` or `correction`. It is the binding's closed reason and takes the place of a rationale; `bind_term` takes no free-text rationale.
- The zone, the contract and the configuration reference are derived from the confirmed configuration. The caller supplies none of them.
- A contract is held to one zone across all its bindings, because the module requires it. The read's rows show the zone each existing binding uses.

## Authority

The setup boundary's command list is extended with the four commands and the read, each with its own branch; a command missing from that list would have no boundary at all. The four commands' branch is: console client, sample installation, then `termCaller` with the command's capability, which the handler repeats and the private function repeats again. Terms Inspect is not required to write, protected rights do not authorize these commands, and installation administration does not substitute. The four gate actions are direct-human actions; a test asserts that the gate refuses to delegate each.

## Guard order

1. Boundary and handler authority.
2. Request shape and closed values.
3. The configuration checks, without any reader check.
4. Target binding. A retained term belongs to the named configuration when its retained configuration snapshot is present and deep-equals the confirmed configuration; a retained binding belongs to it when its embedded term does. A term or binding of another configuration, and one made for a source profile, belongs elsewhere. A term named by any command must be unused or belong to the named configuration; so must a binding. A target that belongs elsewhere is EvidenceStale. For an unused identifier the private function applies its own rules: `approve_term` and `approve_term_binding` then return EvidenceStale, and `bind_term` with an unused term returns PrerequisiteFailed, which differs from the EvidenceStale of a term that belongs elsewhere; that small distinction is accepted. For a retained target of the configuration the private function decides ownership and exact equality of intent, which is how an owned unknown record is recovered and an identical request repeated.
5. Zone pre-check, for `bind_term` with an unused binding identifier only. If any retained binding of the contract, of any origin and state, has a zone different from the configuration's zone, or the configuration's zone is longer than the 64 bytes the module accepts, the command returns PrerequisiteFailed before anything is reserved. A retry of an existing binding is never subject to it.
6. The private function.

There is no timing margin, because no forward deadline exists that could pass between the service check and the module call.

# Effects, outcomes and recovery

The handler follows the protected writes contract. Before and after the private function it records the retained record with its scope index, and a nonsettling module observation of the same record (`Term` or `BindingOf`).

- No error: the receipt, a business state equal to the retained state, and a new result revision. A receipt that cannot be built after anything changed is the uncertain error.
- An error and nothing changed: a clean refusal. A handler or declared failure passes through; any failure the framework would not pass through is returned as PermissionDenied; a module refusal, limit or not-found maps to PrerequisiteFailed, a module conflict or invalid result to EvidenceStale, anything else to DependencyUnavailable.
- An error and anything changed: the uncertain error, with an operation identifier, an advanced revision and same-key replay.

Recovery is a new request by the same actor with the same identity and inputs; the private functions reconcile an owned unknown record and refuse another actor. A retry that fails again at the same point without changing anything is a clean refusal for that call.

Receipts return `schema`, `contract`, `configuration`, `current_eligibility` `unchecked`, and for the term commands (`hb_issuance_term_receipt_v1`) `term`, `term_digest`, `state`; for the binding commands (`hb_issuance_term_binding_receipt_v1`) `binding`, `binding_digest`, `state`.

# What the commands do not do

- They do not change, replace, withdraw or end a term or a binding. A term's net days cannot be corrected once recorded, and a retained binding, pending or approved, occupies its date range for the contract. The module can withdraw and end a binding; the service has no governed operation for it.
- They do not calculate a due date, evaluate coverage for a date, or state that an invoice is payable or overdue.
- They do not offer business-day terms, holiday adjustment, month-end terms, instalments or early-payment discounts.
- They do not add a user's pending term work to the bulk revocation impact of M01. That gap is tracked with the other impact gaps.

# Owner-visible residuals

Accepted for the synthetic boundary by this design; they can be overruled.

- One configuration has one term with one net-days value. A mistaken value can only be left unapproved; a different value needs a new configuration, which today means a new contract revision or a new billing profile version.
- A mistaken binding cannot be withdrawn. Its date range stays occupied for the contract until restart of the in-memory boundary.
- A binding reserved by the service and then refused by the module would stay in `registration_unknown`, with its date range occupied for the whole contract and every retry a clean refusal. The one known cause, a zone that differs from the contract's existing bindings, is removed by the zone pre-check. A module binding the service does not own could still cause it; the service cannot create one. No release operation exists. Release, withdrawal and ending are tracked in HYB-345.
- An open-ended binding, pending or approved, occupies the contract through 9999-12-31. No later binding can follow it, so the reasons `renegotiation` and `correction` are usable only for date ranges that do not overlap an existing binding.
- If the recorded author or pinned approver of a term or binding loses their terms right, or the configuration's proposer or confirmer loses administrator standing, every later operation on that record is refused while its scope or date range stays occupied.
- A contract is limited to one zone across configurations.
- The maximum of 365 net days is the existing provisional bound.
- Finding a configuration requires the protected Inspect right in addition to terms authority.
- Terms carry a reason but no attested evidence. Planning asks for a change reason and approval history, which the retained reason, author, approver and instants provide; it does not ask for evidence of the commercial agreement.

# Alternatives not chosen

- Reusing the protected Inspect right for the read. Terms authority is a separate domain that was reviewed as separate; a protected-evidence reader is not thereby entitled to payment terms.
- Offering a list of common net-days values. Planning says 30 and 45 are examples and not a default; a list would be read as a recommendation.
- A caller-supplied revision or mode. With one term per configuration and one supported mode they would be fields with a single legal value.
- Showing only the configuration's own bindings. A binding of another configuration of the same contract is what makes an overlap refusal understandable.
- Evaluating coverage in the read. That is a point-in-time fact for a given invoice instant and already belongs to the draft context.

# Required verification

Command HTTP tests with independently approved human and console grants created by the existing package-internal helpers.

- the sequence through public commands: a term recorded and approved by two people, bound and the binding approved by two people, then visible through the read with the expected states, and consumed by the existing coverage path for an instant inside the range;
- read authority: terms Inspect, protected Inspect only, a write capability only, administrator without Inspect, delegation, revoked grant and ceiling, ended session, non-sample installation, held read allowed;
- read content: no term, each retained state of term and binding at every interruption boundary, an unknown state never shown as completed, pending approver fields, a binding made under a second configuration of the same contract listed with its configuration, a binding of another contract absent, corrupt digest and scope index, a module binding withdrawn by a direct module call refused, lookup fault distinct from absence, paging and snapshot rules, unchanged state and server revision;
- write authority per command: its own capability, every other terms capability, terms Inspect only, protected capabilities, administrator, revoked grant and ceiling, ended session, hold, non-sample installation; the gate refusing delegation of each action;
- separation: the author approving their own term and their own binding;
- closed values: net days negative, above 365, non-canonical and missing; an invalid date, an end before the start, an unknown reason; no default applied when a value is absent;
- preconditions: a second term for the configuration, binding an unapproved term, an overlapping range including one that overlaps a pending binding of another configuration of the same contract, any range after an open-ended binding, and a binding under a second configuration of the same contract whose zone differs, each a clean refusal with unchanged state and nothing reserved; the same second configuration with the same zone and a disjoint range succeeding;
- standing: an approval refused after the term's author lost the record capability, with the term still listed by the read;
- target binding: a term or binding of another configuration, one made for a source profile, and a missing one, each with the code stated in the guard order; a profile-origin binding of the same contract listed by the read with origin `profile`. A confirmed configuration of a second contract is not built by an existing fixture; the cross-contract case is exercised as a foreign contract target and that limit is recorded;
- effects: every fault point of the four private functions through its public command with the fault then cleared: clean refusal with unchanged state and revision before the reservation; uncertain outcome with an operation identifier, an advanced revision and same-key replay after it; recovery by the same actor; refusal for another actor;
- compatibility: every existing private term test unchanged and passing;
- hold-table completeness with the four commands and the read allowed during hold.

# Limits

Synthetic in-memory boundary only. The O03 payment-terms view needs its own scoped design and terminology review. No implementation or test run is claimed by this document.

# Design review

The independent-reviewer agent (read-only, model reported as claude-fable-5-2) reviewed the first draft against the code at 44f9ee3 and the planning text and returned FAIL with three blocking findings. First, the claim that no cause is known for a stuck unknown record was false: the module refuses a binding whose zone differs from the contract's other bindings, the service reserves before that check, and two configurations of one contract can name different zones. Second, the requirement basis narrowed planning without saying so: display name, the issue-date basis field and holiday policy were dropped silently, and 0 to 365 was presented as decided. Third, bindings made for a source profile were undefined in the read and in the target rule, and residuals were missing for open-ended bindings and for loss of standing. The authority model, closed and fixed values, the effect comparison set and the fallback mapping passed.

Root corrected all three in one pass, adding the zone pre-check, the list of what the slice does not cover, the `origin` of a row, the exact predicate for belonging to a configuration and the further residuals. The bounded corrective re-review found the three resolved, confirmed from the module code that no other refusal after a reservation is reachable through the service, judged the embedded-term rule safe because an approved term record is never rewritten, found no rule that blocks recovery or an identical repeat, and returned an overall design PASS.

The reviewer executed no tests. Inferred and not traced end to end: that the configuration snapshot a term retains deep-equals the configuration the handler confirms, which the first public sequence test will show. Not verified: step-up handling, and fixture support for a confirmed configuration of a second contract. The reviewer listed as decisions that belong to the owner: one irrevocable net-days value per configuration and non-withdrawable bindings until HYB-345; whether the closed binding reason satisfies the planned change reason; the provisional maximum of 365; deferring the display name to the O03 view; terms operators also needing protected Inspect; and one zone per contract. This is a design review, not a source review of an implementation.

