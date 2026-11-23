---
type: ContractDesign
title: Public protected setup writes
description: Public commands for protected rules, verification channels and protected changes that consume retained attestations and named references.
status: draft
---

# Outcome and ownership

HYB-341, machine:mac_mini. This is item 3 of [the handoff](handoff.md). An operator who has confirmed a reusable configuration can establish the protected rules, register and admit verification channels, and request, verify and decide protected reference changes through public commands, without typing an identifier, a fingerprint or an evidence value. The commands wrap the private service functions that already exist and are tested. They add no new protected-module behavior.

Root owns the design and integration. An independent reviewer covers the new mutations, the permission mapping, the effect and recovery semantics and the consumption of attestations. The [discovery contract](issuance-protected-setup-discovery-contract.md) and the [evidence contract](issuance-setup-evidence-contract.md) are reused and not repeated. Requirement basis is QA-01 and AUD-QA-01 with HB-23, HB-28 and HB-32, as quoted in the evidence contract.

# Verified starting facts

Read from main at ee2a9bf; later merges to main changed only the console client and one test literal.

- The private functions are `proposeProtectedRules`, `confirmProtectedRules`, `registerChannel`, `admitChannel`, `requestProtectedChange`, `verifyProtectedChange` and `decideProtectedChange`. Each begins with its own caller check and has before-module, after-module and after-publication fault points.
- On a first attempt each of them writes its retained record before it calls the module: the rules record, the channel record with state `registration_unknown` or the pinned admitter, the change record with phase `request_unknown`, the pinned verification, or the appended decision. On the retry of an owned unknown record the module is called again with no retained write in that call, and a validation of the module's answer can then fail with the retained record unchanged. With the current module those validations are defensive and no trigger was found, but they exist for exactly the case where service and module disagree.
- The module reads `Of` and `Trail`, which the private functions use, settle a lapsed change in module state and may advance the module clock. That is time-determined and idempotent, and the service's comparison tolerates it. The retained module view is not rewritten when it happens.
- After a module effect the private functions return a named DependencyUnavailable failure from their fault points, and they return module errors unwrapped. Neither is fit to be a public outcome: the first would present an effect as a clean refusal.
- The public configuration mutations already solve this with an effect tracker: any error after a retained change becomes the uncertain error, which the command framework stores under the idempotency key with an operation identifier while the server revision advances.
- `channelDigest` and `changeDigest` hash the JSON of the whole record. A field that is omitted when empty leaves existing digests unchanged.
- `registerChannel` takes a closed selector and derives the subject. `requestProtectedChange` requires the verification channel to be admitted, of the same subject, old enough at the request instant, and not in the custody of the requester; it refuses an instruction that names the verification channel or its fingerprint, and one whose sender matches a registered channel while naming none.
- A request whose sender matches no registered channel is accepted as `requested`. It never applies without verification through a different admitted channel and a separate approval, and it lapses otherwise.
- Attestations, their consumption rules and the synthetic reference catalog exist as defined in the evidence contract. `setupReference` resolves a code and use to a fixed identifier and fingerprint. Nothing consumes an attestation yet.
- The four step actions are direct-human actions and the gate refuses to create a delegation for them, which a test asserts. The approve action is mapped the same way in the authority domain; its delegation refusal is not yet asserted by a test.
- The framework passes a handler failure through unchanged only for its handler codes, and for EvidenceStale and SelfApprovalDenied only when the command declares them. Any other failure of a mutating command becomes an internal error with an operation identifier.
- The service limits of 100 channels and 100 changes equal the module limits and are checked first, so a module limit refusal after a reservation is not reachable.
- Rule configuration is governed by installation governors through `protectedRuleCaller`, separately from the step capabilities.

# Commands

All seven are mutating, target the selected contract, and take `configuration` and `digest`. They are refused during hold and are subject to step-up like every mutating command.

| Command | Static permission | Further fields | Declared refusals |
| --- | --- | --- | --- |
| `issuance_setup.propose_protected_rules` | administer installation | `rules`, `approvals`, `validity_seconds`, `max_lead_seconds`, `channel_age_seconds`; rationale required | EvidenceStale |
| `issuance_setup.confirm_protected_rules` | administer installation | `rules`, `rules_digest`; rationale required | EvidenceStale, SelfApprovalDenied |
| `issuance_setup.register_channel` | register channel | `channel`, `subject_kind`, `kind`, `reference`, `attestation`; rationale required | EvidenceStale |
| `issuance_setup.admit_channel` | admit channel | `channel`, `channel_digest`, `attestation` | EvidenceStale, SelfApprovalDenied |
| `issuance_setup.request_change` | request protected change | `change`, `field`, `reference`, `channel`, `channel_digest`, `provenance`, `instruction`, optional `instruction_channel`, `effective`, `attestation`; rationale required | EvidenceStale, SelfApprovalDenied |
| `issuance_setup.verify_change` | verify protected change | `change`, `change_digest`, `outcome`, `attestation` | EvidenceStale, SelfApprovalDenied |
| `issuance_setup.decide_change` | approve protected change | `change`, `change_digest`, `verdict`, optional `reason` | EvidenceStale, SelfApprovalDenied |

Any failure the framework would not pass through for the command, for example an unauthenticated result from a private caller check or a declarable code the command does not declare, is returned as PermissionDenied when nothing changed, so that it is never shown as an uncertain operation.

A rationale is required exactly where the private function retains a reason: the rules proposal, the channel registration and the change request. The rules confirmation takes one because the existing governor mutations do; it is not retained. Admission, verification and decision take none; their attestation or closed verdict is the content.

New identifiers (`rules`, `channel`, `change`) are generated by the console, as the attestation identifier is. They are request identifiers of at most 128 bytes.

## Authority

- Rules commands have their own boundary branch: console client, sample installation, protected Inspect and `protectedRuleCaller`, the current governor check, which the handler repeats. They do not use the configuration mutations' identity check, which treats the command target as a configuration. A step capability does not authorize them.
- The five step commands share the boundary branch of the attestation commands, extended by a step table that includes approval: console client, sample installation, then `protectedCaller` with the command's capability. Inspect is not required and installation administration does not substitute.
- Rules are installation-wide. The rules commands still target a contract with a confirmed configuration, because the console reaches them from that contract's documents view; the configuration is context only and the rules record does not depend on which one is named.
- Every private function repeats its own caller check. No public path reaches a private function with a caller the private function would refuse.

## Guard order

1. Boundary and handler authority.
2. Request shape, including every closed value below.
3. The configuration checks of the discovery contract without its reader check for the step commands, and with it for the rules commands.
4. Target binding. A channel named by `admit_channel`, and the `channel` and `instruction_channel` of `request_change`, must be a channel row of the named configuration under the discovery contract's provenance rules. A change named by `verify_change` or `decide_change` must be a change of the named configuration. A target that is missing or belongs elsewhere is EvidenceStale, with no distinction. The identifier of `register_channel` or `request_change` is either unused, or names a retained record that is a row of the named configuration; in the second case the private function decides ownership and exact equality of intent, which is how an owned unknown record is recovered and an identical request repeated. Rules need no binding.
5. Timing margin, defined under effects.
6. Resolution of references and of the attestation, as defined below. Any attestation failure is EvidenceStale with no distinction between its causes. The evidence contract's phrase "before any module lookup" means before any module write or reservation; the configuration observation of step 3 precedes it.
7. The private function.

A public request requires its verification and instruction channels to be public records. A channel created by the private functions in fixtures is refused with EvidenceStale, like any other target that does not satisfy the binding.

## Closed values and resolution

- `subject_kind` is `customer`, `entity` or `document_kind`. The service builds the selector from the confirmed configuration: the contract and its revision, the billing profile, or the configuration reference. The caller supplies no subject.
- `kind` is `email`, `phone`, `portal` or `postal`.
- `reference` is a catalog code. For a channel its use must be the channel use of the given subject kind; for a change it must be the use of the given field. The service resolves the identifier and fingerprint. An unknown code or a code of another use is InvalidRequest.
- `field` is `document_recipient`, `document_template` or `payment_account`.
- `provenance` is `staff_entry`, `customer_document` or `customer_portal`. The assistant-generated value is not accepted by the public command, and the module refuses it as well.
- `instruction` is one of three forms:
  - `in_person`: no channel and no sender;
  - `registered_channel`: `instruction_channel` names an admitted retained channel of the same subject; the service takes the channel kind and sender fingerprint from it;
  - `unregistered_sender`: an email instruction whose sender is the catalog's unregistered sender. `instruction_channel` must be empty.
- `effective` is an RFC 3339 instant. The private function requires it to be after now and within the rules' maximum lead.
- `outcome` is `confirmed` or `denied`. `verdict` is `approve` with an empty `reason`, or `reject` with one of the module's four closed reasons.
- The four rule values are whole numbers inside the module's bounds: approvals 1 to 3, validity 1 hour to 30 days, maximum lead 1 hour to 366 days, channel age 24 hours to 90 days, each given in seconds. No default is supplied.

# Attestation consumption

The evidence contract's consumption rules are implemented as follows.

| Command | Attestation purpose | Target | Where its identifier goes | Where its digest is retained |
| --- | --- | --- | --- | --- |
| `register_channel` | `channel_registration` | the new channel | the channel's registration evidence | a new field of the channel spec, inside the channel digest |
| `admit_channel` | `channel_admission` | the channel | the admission evidence | a new field beside the pinned admitter, outside the channel digest |
| `request_change` | `change_instruction` | the new change | the instruction evidence | a new field of the change spec, inside the change digest |
| `verify_change` | `change_verification` | the change | the verification evidence | a new field of the pinned verification, outside the change digest |
| `decide_change` | none | | | |

- Every new field is omitted when empty, and the admission attestation digest is additionally cleared when the channel digest is computed, as the admitter and admission evidence are. Records created by the private functions without an attestation keep their digests, and their fixtures stay valid. A record is a public record exactly when its registration or instruction attestation digest is present.
- Before reserving anything the service checks the attestation as the evidence contract lists: workspace, purpose, target, actor equal to the caller, recomputed digest, not withdrawn, the bound target digest for admission and verification, and `consumed_by` empty or naming this operation. For admission and verification the target digest is that of the retained target. For registration and instruction an unconsumed attestation needs no target digest; a consumed one is accepted only when a retained record with this identifier exists and `consumed_by` names its digest, which is the retry of an owned operation. The final equality with the digest of the record being created is checked inside the private function at the reservation.
- At the existing reservation point of each private function, in the same step that stores the intent, the service marks the attestation consumed by purpose, target identifier and the retained digest of the target record. The existing before-write fault point precedes both. For registration and instruction the target digest is the digest of the record being created, which already includes the attestation digest; there is no cycle because the attestation digest excludes its consumption mark.
- A retry of the same uncertain operation by the same actor finds `consumed_by` naming it and proceeds through the private function's own recovery.
- The existing standing checks of a retained channel and a retained change (`currentChannel`, `changeStanding`) additionally require, for every attestation digest a record holds, whether or not the record is otherwise public, that the attestation still exists with that digest and is consumed by this record. A missing or changed attestation is EvidenceStale in every later write on that record. The discovery reads are unchanged.
- A decision consumes nothing. The approver is kept by the module as decision actor, verdict and instant.
- A registrar cannot hold an admission attestation for their own channel and a requester cannot hold a verification attestation for their own change, because the attest commands refuse them. Such a caller is therefore refused here with EvidenceStale at the attestation check, before the private function's SelfApprovalDenied. A custodian requesting or verifying and every separation case of a decision reach SelfApprovalDenied.

The private functions gain an attestation parameter or spec field for this. Callers that pass none, which is every existing test and private caller, behave exactly as today.

# Effects, outcomes and recovery

Before calling the private function each handler records three things for the one record the command can affect: the retained record (the rules record, the channel record or the change record with its order entry), the attestation, and a nonsettling observation of the module's copy (`RulesOf`, `ChannelOf` or `Peek`). After the call it compares all three.

- No error: the outcome carries the receipt, a business state and a new result revision. If building or bounding the receipt fails after anything changed, the outcome is the uncertain error.
- An error and nothing changed: a clean refusal. A handler or declared failure passes through. A module refusal or limit maps to PrerequisiteFailed, a module conflict or invalid result to EvidenceStale, a module not-found to PrerequisiteFailed as the draft commands map it, and anything else to DependencyUnavailable.
- An error and anything changed: the uncertain error, whatever the private function returned. The framework stores it with an operation identifier and the server revision advances. A retry with the same idempotency key replays it.

The module observation is part of the comparison because of the retry paths described in the starting facts: an effect applied by a retry and then rejected by the service's validation leaves the retained record unchanged but changes the module's copy, and must not be reported as a clean refusal. The observation projects a lapse without settling; if a deadline passes between the two observations the call is reported as uncertain, which errs on the safe side. The observations may advance the module clock high-water mark, as every module call does; with a non-decreasing clock that is not a change.

A retry that fails again at the same point without changing anything, for example a second failure after the same module write, is a clean refusal for that call: nothing happened in it. The record stays in its unknown state and the discovery reads show it.

## Recovery and its limits

Recovery of an uncertain operation is a new request by the same actor with the same business identity and inputs. The private functions reconcile an owned unknown record: registration and request retry only an exact owned intent, admission and verification require the pinned actor and evidence, a decision requires the actor of the pending decision. Another actor is refused. The discovery reads show the unknown state and the pinned pending actor in the meantime.

Recovery does not always converge. A record can be left in an unknown state that no request can complete, when the module refuses after the service reserved. The reachable cause is time passing between the service's own check and the module call: a requested effective instant that was just ahead of the clock at the check and no longer is at the module, or a change that lapses between the settled read and the verification or decision write. The consequences are wider than an occupied identifier:

- a change left in `request_unknown`, `verification_unknown` or `decision_unknown` blocks every later request for the same subject and field, because a request requires every earlier entry to be recorded. For `payment_account` the subject is the issuer entity, shared by every configuration of that billing profile;
- the consumed attestation stays consumed;
- the first protected rules proposal cannot be discarded or replaced, so a proposal nobody confirms, or one left unknown, blocks rules for the installation.

Nothing short of restarting the in-memory boundary clears such a record. The service has no release, discard or withdrawal operation and this contract adds none; that is tracked as a separate issue.

## Timing margin

To keep the time-driven case out of reach, the public commands apply a fixed margin of one minute, as a clean PrerequisiteFailed, only to a call that would make a new reservation:

- `request_change`, when no retained record has the change identifier: `effective` must be at least the margin after the current instant;
- `verify_change`, when the retained record is `recorded`, holds no pinned verification and is observed as `requested`: its deadline, the effective instant, must be at least the margin after the current instant;
- `decide_change`, when the retained record is `recorded`, holds no decision by the caller and is observed as `confirmed`: its deadline, the earlier of the effective instant and the verification instant plus the validity fixed in the change, must be at least the margin after the current instant.

In every other case the margin is skipped and the private function reconciles or refuses. A recovery request and an identical repeat reserve nothing, so they are never refused by the margin, however late they arrive: refusing them would leave the record in its unknown state, which is the outcome the margin exists to prevent.

The margin is an adapter safety constant, not a business rule. It does not change what the module accepts, and the private functions are unchanged. With it, the stuck state needs the handler to stall for more than a minute between its check and the module call while holding the domain lock. That residual, and the defensive validation failures, are accepted for the synthetic boundary and listed for the owner below.

## Receipts

Every success returns `schema`, `contract`, `configuration`, `current_eligibility` with value `unchecked`, and:

- rules commands (`hb_issuance_protected_rules_receipt_v1`): `rules`, `rules_digest`, `state`;
- channel commands (`hb_issuance_protected_channel_receipt_v1`): `channel`, `channel_digest`, `state`;
- change commands (`hb_issuance_protected_change_receipt_v1`): `change`, `change_digest`, `phase`, `state` (the module state the private function returned, which can be `lapsed` while the retained copy still says `requested` or `confirmed`).

Business state is the rules state, the channel state, or the module state of the change. A receipt carries no reference identifier, fingerprint, evidence value or reason. The client reads the discovery and attestation reads for everything else.

# What the commands do not do

- They do not evaluate or report readiness. A scheduled change is consumed by a draft only through the existing draft commands and their own checks.
- They do not replace or discard rules, retire a channel, withdraw a change or release a record left in an unknown state. The service has no such operations.
- They do not let the verifier choose the verification channel. The requester fixes it in the request.
- They do not refuse two channels of one subject that name the same reference. Using such a pair as instruction and verification channel is refused as an invalid request.
- They do not send a message or make a call. Verification is the verifier's attested act.
- They do not accept a tax identifier change. The supported synthetic kind is nonstatutory.
- They do not provide caller-specific action guidance. A refusal code is the guidance until a separate action projection is designed.
- They do not add the pending protected work of a user to the bulk revocation impact of M01. That gap is tracked separately with the other impact gaps.

# Owner-visible residuals

These follow from wrapping functions that have no release operation. They are accepted for the synthetic boundary by this design and can be overruled.

- A change left in an unknown state blocks its subject and field until restart. The timing margin makes the known cause impractical to reach; it does not remove the state.
- The first protected rules proposal is irrevocable for the installation.
- The one-minute timing margin is a fixed adapter constant, not an installation default.
- Installation-wide rules are reachable only through a contract that has a confirmed configuration.
- Unchanged from the evidence contract: step actors act without Inspect, and evidence is self-attested by the acting person.

# Alternatives not chosen

- Comparing only the retained record before and after. It misses an effect applied by a retry and then rejected by the service's validation.
- Threading a "module attempted" marker through all seven private functions. The nonsettling observation gives the same answer without editing reviewed code paths.
- Adding release, discard or withdrawal operations here. Each is a new governed write with its own separation rules and belongs in its own contract.
- Passing the attestation digest only at the public layer and marking consumption after the private function returns. A failure between the module effect and the mark would leave an effect whose evidence another operation could still take.
- One command per subject kind or per instruction form. The static permission does not differ, so a closed field is sufficient.
- Letting the caller supply the instruction sender fingerprint or a reference identifier. The evidence contract excludes both.

# Required verification

Command HTTP tests for every public command, with real attestations written through the attest commands. Capability grants and console ceilings are created with the existing package-internal grant helpers, because no public provisioning command exists; that is handoff item 5. "Unchanged state" means retained service state and server revision; module settlement of a lapse is not a change.

- the sequence through public commands only, apart from grants: rules proposed and confirmed by two governors; three channels registered and admitted by distinct custodians with their attestations; the clock advanced by the channel age; sessions reopened and capability grants renewed for every standing actor, custodians included, because grants and sessions do not outlive the advance and later writes recheck the custodians and the governors; a change requested, verified and approved for each of the three fields by three further distinct people, which needs a fifth member beyond the seeded four; every row then visible through the discovery reads with the expected states and every attestation listed as consumed;
- AUD-QA-01: a change requested from the unregistered sender is accepted as `requested`, cannot be approved without verification, and lapses unapplied at its effective instant; assistant-generated provenance is refused as an invalid request;
- authority per command: its own capability, every other capability, Inspect only, administrator without the capability for step commands, a step capability without governor standing for rules commands, revoked grant and ceiling, ended session, hold, non-sample installation; the gate refusing a delegation of the approve action;
- separation, with the expected code: registrar admitting and requester verifying refused with EvidenceStale at the attestation check; custodian requesting, custodian verifying, and requester, verifier or custodian deciding refused with SelfApprovalDenied; verification through the instruction channel refused as an invalid request;
- target binding: a channel or change of another configuration, of another contract and a missing one, each EvidenceStale with no retained change; a private-fixture channel named as verification or instruction channel refused with EvidenceStale;
- closed values: each invalid value, a reference of another use, an unknown code, an instruction channel given with another form, rule values outside the bounds, a channel younger than the channel age at the request instant;
- timing margin: an effective instant inside the margin and one exactly at it; verification and decision with the deadline inside the margin and exactly at it; each refusal clean with unchanged state; recovery of an uncertain request, verification and decision inside the margin and after the deadline succeeding;
- consumption: a missing, withdrawn, other-purpose, other-target and other-actor attestation, a changed bound target digest, an attestation consumed by another operation, each EvidenceStale with no retained change; the same attestation reused by the retry of its own uncertain operation; a record whose held attestation was removed or altered by a package-internal edit refused in its next write;
- effects: every fault point of the seven private functions, including their lookup faults, reached through its public command with the fault then cleared: a clean refusal with unchanged state and revision before the reservation; an uncertain outcome with an operation identifier, an advanced revision and same-key replay after it; recovery through a new identical request by the same actor; refusal for another actor; and a repeated failure at the same point reported as a clean refusal;
- compatibility: every existing private-function test unchanged and passing, and the digests of privately created channel and change records unchanged;
- hold-table completeness with the seven commands.

Not testable with the existing seams, and recorded as untested: a record stuck in an unknown state by a module refusal after reservation, and a retry whose module answer fails the service's validation. Neither has a fault point; the first is what the timing margin guards against.

# Limits

Synthetic in-memory boundary only. The O03 screens for these commands and for attestations need their own scoped design and terminology review. Payment terms, access provisioning and the complete unseeded operator journey to draft readiness remain handoff items 4 and 5. No implementation or test run is claimed by this document.

# Design review

The independent-reviewer agent (read-only, model reported as claude-fable-5-2) reviewed the first draft against the code at ee2a9bf and returned FAIL with four blocking findings: the claim that a module effect never exists without a changed retained record is false on the retry paths of owned unknown records; the inherited limit of a stuck unknown record was misdescribed and understated, since it blocks a whole subject and field; the declared refusals were not specified, so a clean refusal could surface as an uncertain operation; and nothing bound a named target to the named configuration. Authority, closed values and resolution, the attestation consumption design and requirement fidelity passed. Outcomes, scope precision and the required verification failed as precision items and were corrected with the four.

Root corrected all four in one pass, adding the nonsettling module observation to the effect comparison, the timing margin, the declared refusals and the target binding. The first bounded re-review found the four resolved and two new defects introduced by the revision: the margin, applied to every call, would also refuse the recovery of an uncertain operation, and the rule that a new identifier must be unused contradicted recovery. Both are corrected above: the margin applies only to a call that would reserve, and an identifier may name a retained row of the configuration. The final bounded check found both resolved and returned an overall design PASS.

The reviewer executed no tests and did not read the planning source for QA-01, relying on its quotation in the evidence contract. Unverified by the reviewer: that the gate accepts the approve action as a command permission and refuses its delegation, which the first build and a test will show; and the absence of any public grant-provisioning command. The reviewer listed the items under "Owner-visible residuals" as decisions that belong to the owner. This is a design review, not a source review of an implementation.

