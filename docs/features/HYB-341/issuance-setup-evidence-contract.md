---
type: ContractDesign
title: Attested evidence and reference source for protected issuance setup
description: Retained human attestations and a named synthetic reference source that protected setup writes must use instead of typed identifiers.
status: draft
---

# Outcome and ownership

HYB-341, machine:mac_mini. This is item 2 of [the handoff](handoff.md) and a prerequisite of item 3. Before a protected rule, channel or change write is exposed, each evidence identifier it takes must name a retained record that a specific human owns, and each protected reference must come from a named source. This document defines both. It adds no protected write, no upload, no vault and no claim that a customer instruction, an account or a contact was verified.

Root owns the design and integration. An independent reviewer covers the new mutations and disclosure, the permission mapping and the consumption rules that item 3 must implement. The [discovery contract](issuance-protected-setup-discovery-contract.md) and the earlier setup reviews are reused and not repeated.

# Requirement basis

`docs/planning/v1.16/detailed-requirements.md`, QA-01, requires for payment account, document recipient, tax identifier and template changes a separate verification, approval and validity period, and requires that the protected references before and after, the verifier and approver, and the evidence of confirmation through an existing registered channel are kept. A new email instruction alone never applies an account and an LLM never generates or replaces payment information. Acceptance criterion AUD-QA-01 refuses an email-only change and an LLM-generated account, and applies a change only from its effective date after existing-channel confirmation and separate approval. Linked requirements are HB-23, HB-28 and HB-32.

The protected module already enforces the refusals, the separation of actors, channel age, verification validity and the effective date, and it already keeps the references before and after and the verifier and approver with their instants. What is missing is the kept evidence: today the four evidence values are opaque strings.

# Verified starting facts

Read from main at defcf91.

- The protected writes take four caller-supplied evidence identifiers: the channel registration evidence, the admission evidence, the change instruction evidence and the verification evidence. Each is validated only as an identifier of at most 128 bytes (`financialissuancechannel.go`, `financialissuancechange.go`, `financialissuancechangewrites.go`).
- The module binds the instruction evidence into its request digest and the verification evidence into its confirmation digest (`modules/protected/changes.go`). The service channel digest covers the registration evidence. The admission evidence and the pinned verification are outside the retained channel and change digests and are pinned by equality.
- The retained channel and change digests hash the JSON of the whole record, so a new field changes the digest of every record unless it is omitted when empty, as the configuration fields on the selector and subject already are.
- A channel and a change also take a protected reference of identifier and fingerprint. A non-in-person instruction additionally takes a sender fingerprint. No product source supplies any of these values. Fixtures use synthetic constants.
- Customer and entity channels carry no configuration reference and are shared by every origin that froze the same organization facts. Only a document-kind channel and every configured change belong to one configuration.
- The API has no byte upload path. Command fields are text, at most 4096 bytes each, with at most 64 fields. The command journal keeps a keyed digest of the payload and no field values; server log calls carry the user and command only.
- The JSON decoder replaces invalid UTF-8 with U+FFFD before a handler sees a field.
- The command framework evaluates one static permission per command before the handler runs. Every mutating command is subject to step-up. A setup mutation that fails after an effect returns an uncertain error, which the framework stores under the idempotency key with an operation identifier while the server revision advances.
- `modules/evidence` is imported by no server code. Its artifact kinds are a closed set for billing reproduction and it records a caller-supplied digest without ever holding bytes.
- No general record exists in which a named human asserts a fact separately from an approval.
- The protected writes have no production caller. They are reached only from tests.

# Decision

Evidence for protected setup is a retained human attestation. A named human, entitled to perform the step, states in their own words what they received or did, and may add where the source material is kept outside the product. The server fixes the record, computes its digest and keeps it. A protected write then names that record instead of a typed identifier.

The attestation is an accountable claim. It is not proof. Nothing in a response, a field name or screen copy may call it verified, confirmed or proven.

Four steps are attested: channel registration, channel admission, change instruction and change verification. The admitter's and the verifier's attestations are independent of the registrar and the requester, because the module refuses those actors for those steps. The registrar's and the requester's attestations are statements about their own act; they add accountability, not independence. The approver attests nothing: the approver is kept as the module's decision actor, verdict and instant.

Protected references in the supported synthetic boundary come from a named synthetic catalog, declared as sample facts, in the same way the configuration preset is. A production source for real protected values does not exist and is not designed here.

# Owner-visible residuals

Two choices in this design set the assurance level of a QA-01 control. They do not block implementation in the synthetic boundary, and they are recorded so that the owner can overrule them.

- "Independent human attestation" is read as each separated actor attesting their own step. No second human countersigns an attestation and the approver writes none.
- The evidence form is retained free text written by a person. Outside the synthetic boundary such text may hold personal or contact detail, so production use needs a data-classification and retention decision first.

# Attestation record

| Field | Content |
| --- | --- |
| `id` | Generated by the console; a request identifier of at most 128 bytes, because the module carries it as its evidence value |
| `workspace` | The boot workspace |
| `context` | The identifier and digest of the confirmed configuration under which it was written |
| `purpose` | `channel_registration`, `channel_admission`, `change_instruction` or `change_verification` |
| `target` | The identifier of the channel or change the attestation is for |
| `target_digest` | For admission and verification, the retained digest of the existing target; empty for registration and instruction |
| `actor` | The attesting human |
| `statement` | Required text, at most 2000 bytes |
| `locator` | Optional text of at most 512 bytes naming where the source material is kept outside the product |
| `recorded_at` | The server clock at retention |
| `digest` | Server-computed over every field above |
| `consumed_by` | Empty, or the operation that used it: purpose, target identifier and the retained digest of the target record. Not part of the digest |
| `withdrawn_at` | Empty, or the instant its author withdrew it. Not part of the digest |

The four purposes are closed because they are the four steps the module already has. A closed vocabulary of evidence bases, such as "signed contract" or "callback", is not introduced: planning leaves the responsible roles open, and a vocabulary would present a guessed policy as a rule.

## Text admission

The statement and locator are free text. The product cannot check that they are true or that they hold no contact or account detail. Screen copy must tell the writer to describe the act and to keep protected values out of the text.

The server refuses with InvalidRequest a statement that is blank, and a statement or locator that exceeds its byte limit, contains U+FFFD, or contains a control character other than tab and line feed. U+FFFD is refused because the decoder substitutes it for invalid input, and retaining it would keep text the writer did not write.

Rows are serialized as JSON inside a JSON envelope, and escaping can multiply the size of a text. Before retention the server builds the row this attestation would produce with every server-set field at its maximum length and the longest state value, and refuses with InvalidRequest unless the serialized row is at most 4096 bytes and its form inside the page envelope is at most 6000 bytes. Ten such rows and the envelope fields then stay inside the 65536-byte response bound. A text that passes the raw limits can therefore still be refused when it consists largely of characters that JSON escapes. This follows the receipt admission that configuration proposals already use. The row's field set is fixed by this contract: adding a row field later would invalidate the admission of records already retained and needs a new schema. A carriage return is a refused control character, so the console must normalize pasted line endings to line feeds before sending.

## Capacity

- At most 1000 attestations are retained in total.
- An actor may hold at most 10 attestations that are neither consumed nor withdrawn.

Consumed attestations are bounded by the existing limits of 100 channels and 100 changes at two each. The per-actor bound stops one capability holder from filling the store with unusable records; withdrawal lets an actor clear their own mistakes. The residual risk is stated: withdrawn records are retained and count toward the total, so repeated attest and withdraw by capability holders can still exhaust the total. Only the command rate limit slows that. In this in-memory boundary the state is lost on restart; a production design must bound it differently.

# Writes

Each purpose has its own command, so that the framework's single static permission is exactly the capability of the step.

| Command | Permission action on the protected resource kind | Target at attest time |
| --- | --- | --- |
| `issuance_setup.attest_channel_registration` | register channel | No retained channel has this identifier |
| `issuance_setup.attest_channel_admission` | admit channel | A retained channel in state `registered` or `admission_unknown` whose digest equals `target_digest` |
| `issuance_setup.attest_change_instruction` | request protected change | No retained change has this identifier |
| `issuance_setup.attest_change_verification` | verify protected change | A retained change in phase `recorded` with retained module state `requested`, whose digest equals `target_digest` |

Each attest command has a matching withdraw command under the same permission: `issuance_setup.withdraw_channel_registration`, `withdraw_channel_admission`, `withdraw_change_instruction` and `withdraw_change_verification`. Each withdraws the caller's own attestation of that purpose that is neither consumed nor withdrawn.

All eight are mutating, target the selected contract and take no rationale; the statement is the content and a rationale would duplicate it. The four attest commands take `configuration`, `digest`, `attestation`, `target`, `statement` and optional `locator`; the two that bind an existing target also take `target_digest`. A withdraw command takes `configuration`, `digest` and `attestation`.

## Authority

- The setup boundary gives these eight commands their own branch. It admits them for the console client in the sample installation and calls `protectedCaller` with the command's capability instead of the `protectedReadCaller` call that every other setup command takes: current human session, the capability's gate action for the console client, no delegation, and an owned applied human grant under its console ceiling. The handler repeats that check. Hold refuses, with explicit entries in the hold table.
- Inspect is not required to attest. A verifier needs the verify capability and nothing else. In practice the screen reads discovery rows to choose a target, which does need Inspect; that is a property of the screen, not of this command.
- Installation administration is not required and does not substitute for the capability. The governor check used for rules and configuration does not apply.
- A withdraw command requires the capability of its purpose, because the framework admits no command without a concrete permission and no session-only command exists. It refuses with PermissionDenied any attestation that is missing, of another purpose or written by another actor, without distinguishing those cases. A person who has lost the capability cannot withdraw; at most ten of their unused attestations then stay retained and unusable, since consumption also requires the capability.
- Step-up applies as it does to every mutating command.

## Guard order of an attest command

1. Boundary and handler authority as above.
2. Request shape and text admission.
3. The configuration checks of the discovery contract, without its reader check, which would require Inspect: unknown or foreign configuration PermissionDenied, wrong digest EvidenceStale, unconfirmed PrerequisiteFailed.
4. Identity. An existing attestation with this identifier is returned unchanged when its workspace, actor, purpose and every caller-supplied value are identical, and is EvidenceStale otherwise with no detail about the existing record.
5. Target state from the table. A target that does not satisfy its row is EvidenceStale, with no distinction between missing, changed and wrong state. For admission the caller must not be the channel's registrar and for verification not the change's requester; either is SelfApprovalDenied.
6. Capacity: PrerequisiteFailed at the total or at the caller's bound.
7. Retention.

Binding the existing target's digest at attest time means an admission or verification statement cannot be recorded before the thing it speaks about exists, and cannot be carried over to a different registration or request under the same identifier.

## Outcome and faults

The record is a service fact with no module effect, so it has no unknown phase in service state. The client-visible outcome follows the existing setup mutations:

- A fault before retention returns DependencyUnavailable. Nothing is retained and the server revision does not move.
- A fault after retention returns the framework's uncertain error with its operation identifier. The record is kept and the server revision has advanced. A retry with the same idempotency key replays that stored error.
- A new request with the same attestation identifier and identical inputs finds the retained record in step 4 and returns it. If that record has since been consumed or withdrawn, the outcome still carries business state `recorded`, and the `state` field says `consumed` or `withdrawn`; a client must read `state`.
- Every successful outcome, including that return of an existing record, carries business state `recorded` and a new result revision, as existing setup mutations do.

The response returns `schema` (`hb_issuance_setup_attestation_v1`), `contract`, `configuration`, `attestation`, `purpose`, `target`, `actor`, `recorded_at`, `digest`, `state` and `assurance` with the fixed value `human_attestation`. It does not echo the statement or locator.

A withdraw command runs the same authority, shape and configuration checks, then sets `withdrawn_at` and returns the same fields with state `withdrawn` and business state `withdrawn`. Repeating it returns the record unchanged with a new result revision, as every successful mutation outcome carries one. The configuration named by a withdraw command must be a confirmed configuration of the selected contract; it need not equal the attestation's recorded context, because actor and purpose are the binding. The receipt names the contract and configuration of the withdraw command, not the recorded context. A consumed attestation cannot be withdrawn and returns PrerequisiteFailed. Its fault points and outcomes are those of attest: DependencyUnavailable with no change before the mark is set, the uncertain error with an advanced revision after it, and a new identical request returning the withdrawn record. Withdrawn records are kept: a deleted claim would be a hole in the record of what people asserted.

# Consumption rules for item 3

These bind the public protected writes. They are requirements on that later contract, not implemented behavior.

- The write names an attestation identifier where the private function takes an evidence string. The identifier is passed to the module unchanged, so the module digests continue to bind it.
- After caller authority and request shape and before any module lookup or reservation, the service requires: the attestation exists in this workspace; its purpose matches the step; its target equals the channel or change identifier of the write; its actor is the caller; its retained digest recomputes; it is not withdrawn; for admission and verification its `target_digest` equals the retained digest of the target now; and `consumed_by` is empty or names this same operation. Any failure is EvidenceStale, with no distinction between the cases.
- The attestation's `context` is not compared with the write's configuration. Customer and entity channels are shared across configurations, so an attestation written while working in one configuration legitimately supports a channel used by another. Workspace, target, actor and purpose are the binding.
- An operation is identified by purpose, target identifier and the retained digest of the target record. `consumed_by` holds exactly that. A retry of the same uncertain operation by the same actor matches it and reuses the attestation.
- The service retains the attestation digest in the intent it reserves and marks the attestation consumed in the same step as the reservation. The before-write fault precedes both. A later failure keeps both, so an uncertain operation keeps its evidence and no other operation can take it.
- The registration and instruction attestation digests are new fields of the retained channel and change records and so enter their digests. The fields are omitted when empty, which keeps the digests of privately created records and their fixtures unchanged. Admission and verification attestation digests are pinned beside the pinned actor, outside the retained digest, which is the assurance those pins have today and not a stronger one.
- One attestation supports one operation.
- A retained public channel or change whose attestation is missing, changed or consumed by a different operation is EvidenceStale in every later write on that record and in `issuance_setup.attestations`. The merged discovery reads do not validate attestations and are not changed.

# Read: issuance_setup.attestations

A nonmutating command targeting the selected contract with `configuration`, `digest` and optional `after` and `snapshot`. It uses the discovery contract's reader authority, configuration guards, hold allowance, ten-row pages, snapshot rule and bounds without change.

Scope for a configuration is the union of:

- attestations consumed by a channel or change that is a row of this configuration under the discovery contract's provenance rules, whatever context they were written under;
- attestations that are not consumed, written under this configuration as context.

Rows are sorted by identifier: `attestation`, `digest`, `purpose`, `target`, `actor`, `recorded_at`, `statement`, `locator`, `state` (`unconsumed`, `consumed` or `withdrawn`) and `assurance`. The page returns `schema` (`hb_issuance_setup_attestation_page_v1`), `contract`, `configuration`, `digest`, `current_eligibility`, `snapshot`, `count`, `next` and the rows.

Any capability holder can write text against any target identifier that satisfies the target rule. An unconsumed or withdrawn attestation is therefore one person's unused claim and must never be presented as supporting its target. The client joins a row to a channel or change row only when its state is `consumed`, and the screen design must show the others separately.

Each retained record is validated by its recomputed digest before projection, and each consumed record against the target it names. A corrupt record refuses the page with EvidenceStale and no partial rows.

# Reference source: issuance_setup.references

A nonmutating command targeting the selected contract with `configuration` and `digest`, under the discovery contract's reader authority and configuration guards. It names the synthetic references the adapter supports, as the options read does for the configuration preset.

- Two references for each of `document_recipient`, `document_template` and `payment_account`, so that an initial reference and one replacement can be exercised.
- Two channel references for each of the three subject kinds, so that an incoming instruction channel and a different verification channel can be exercised.
- One sender that is registered as no channel, for the AUD-QA-01 case of an instruction that arrives only by a new email. The module accepts such a request as `requested`; it does not refuse it at request time. The control is that the change never applies on its own: it needs verification through a different admitted channel and a separate approval, and lapses otherwise. The entry lets that be requested and shown not to apply. The module's request-time refusal is the other case, a sender that matches a registered channel while no channel is named.

That is 13 entries, one bounded field each, with `schema` (`hb_issuance_setup_references_v1`), `contract`, `configuration`, `digest` and `current_eligibility` `unchecked`: 18 fields. Each entry carries a stable code, its use and `applicability` `synthetic_only`. Reference identifiers and fingerprints are not returned; the discovery contract withholds them and the caller never needs them. Inside the adapter each code maps to a fixed identifier whose fingerprint is the SHA-256 of the identifier text. Distinct identifiers give distinct fingerprints, which the module's sender, same-channel and no-change checks rely on. They stand for no real account, address or template. Localization maps the codes to sample labels.

Item 3 must require that a public channel registration or change request names an advertised reference by code and that the service resolves the identifier and fingerprint. A public change request supports an in-person instruction, an instruction through a named admitted channel whose sender the service resolves from that channel, and the named unregistered sender. The caller never supplies an identifier or a fingerprint. Private functions and records they created are unchanged.

Channel age of at least 24 hours and the five distinct humans the module requires are properties of the workflow, not of this catalog. Whether a sample installation can supply five people without fixtures is not established here.

# Retention and access

Attestations are immutable apart from consumption and withdrawal marks, and are never deleted or purged in this boundary. They live in the same in-memory synthetic state as the records they support and are lost with it. No retention period is set: `docs/planning/v1.16/retention-access.md` declines to fix statutory periods and requires a governed retention policy revision, which does not exist yet for this evidence. Production persistence, retention, legal hold and deletion for attestations are unresolved and must not be inferred from this design.

Reading requires the protected Inspect right with its evidence field. Writing requires the capability of the supported step. The statement and locator appear only in `issuance_setup.attestations`. They are not written to logs or the command journal, and must not be copied into tickets, commit text or the discovery rows.

# What this does not establish

- That a person, an account or a contact is who or what an attestation says. AUD-QA-01's confirmation through the existing channel is recorded as the verifier's attested act; the product sends no message and makes no call. Earlier verification records already defer that part because messaging is not connected.
- Stored source documents. The locator points outside the product.
- A production source for protected values, or approval to retain free-text statements outside the synthetic boundary.
- Tax identifier references. The supported synthetic kind is nonstatutory.

# Alternatives not chosen

- One attest command with a purpose field. The framework evaluates one static permission before the handler, so the capability could not depend on a body field.
- A withdraw command that needs only a session, so that a person who lost a capability could still withdraw. The framework cannot register a command without a concrete permission.
- Byte upload with a server-computed hash. There is no upload path, and a file or its hash proves neither an instruction nor control of a contact.
- `modules/evidence` artifacts. The kind set and bundle model describe billing reproduction, and the module holds only a caller-supplied digest.
- Free entry of evidence identifiers and fingerprints. The operator plan already rejects unexplained identifier boxes.
- A closed evidence-basis vocabulary or a mandatory countersignature. Both would invent policy that planning leaves open.
- An expiry period for unused attestations in place of withdrawal. It would introduce a period nobody has decided.
- Deriving a fingerprint in the browser from a typed protected value. Low-entropy values make such a fingerprint guessable, and it would move real protected values into the client.

# Required verification

Actual command HTTP tests with independently approved human and console grants. A corrupt retained record and the consumption fixtures need package-internal map edits or the item 3 writes. Invalid UTF-8 cannot be sent through real HTTP because the decoder substitutes it; the U+FFFD refusal is the test.

- authority: each attest command with its own capability and refused with every other capability, with Inspect only, as administrator without the capability, through delegation, after grant and ceiling revocation, with an ended session, during hold and in a non-sample installation; a writer without Inspect succeeds;
- shape and text: blank and oversized statement, oversized locator, U+FFFD, control characters, escape-heavy text at the row bound and at the envelope bound, malformed identifiers, unknown and foreign configuration, wrong digest, unconfirmed configuration;
- target: registration and instruction against an existing identifier; admission and verification against a missing target, a wrong digest and a wrong state; the registrar attesting admission and the requester attesting verification;
- identity: an identical new request returns the same record and instant; the same identifier with a changed statement, target or actor is EvidenceStale without disclosure;
- capacity: the eleventh unconsumed attestation of one actor refused, freed by withdrawal, and the total bound;
- faults: nothing retained and no revision change before retention; after retention the record kept, the revision advanced, the same-key retry replaying the stored error and a new identical request returning the record;
- withdrawal: own unconsumed record, repeat, another actor's, another purpose's and a missing record indistinguishable, a consumed record refused, refusal after loss of the capability, and its fault points;
- read: both scope rules including a shared customer channel attested under one configuration and listed under another, state values, digest recomputation, a corrupt record refusing the page, paging and snapshot rules, reader authority, held read, unchanged state and server revision;
- references: the 13 fixed codes, no identifier or fingerprint in the response, authority, stale digest, no state change;
- consumption, verified with item 3: wrong purpose, wrong target, another actor's attestation, a withdrawn one, a consumed one, a missing one, a changed target digest, reuse by the same uncertain operation, and attestation loss or change after consumption.

# Limits

Synthetic in-memory boundary only. No implementation or test run is claimed by this document. The screen that collects a statement needs its own scoped design and terminology review.

# Design review

The independent-reviewer agent (read-only, model reported as claude-fable-5-2) reviewed the first draft against the code at defcf91 and the QA-01 planning text and returned FAIL with six blocking findings: an authority contract that one static command permission could not express; a capacity that one actor could exhaust; fault and replay semantics that contradicted the command framework; four undefined points in the consumption rules, including the configuration binding of shared channels; row and page bounds that ignored JSON escaping; and no source for an instruction sender. The core decision, a retained human attestation with stated non-claims and a named synthetic catalog, passed, as did requirement coverage, the logging check and the retention section.

Root rewrote the document in one pass. The first bounded re-review found the six findings resolved and raised two new points introduced by the rewrite: a withdraw command with no concrete permission cannot be registered in this framework, and the description of the unregistered-sender entry misstated the module, which accepts such a request and relies on non-application without independent verification and approval. Root corrected both. The second bounded re-review returned an overall design PASS with no blocking finding; its two clarifications on withdrawal are included above.

The reviewer executed no tests. Not verified by the reviewer: that the gate describes the four capability actions on the protected resource kind for command registration, which the first build will show; whether a sample installation can supply five distinct people; and the ERP result and upload-path statements, which rest on root's separate read-only survey. The reviewer recorded the two items under "Owner-visible residuals" as decisions that belong to the owner. This is a design review, not a source review of an implementation.

# Implementation and verification

Root implemented the attest and withdraw commands, the attestations read and the references read in `apps/api/server/financialissuanceattestation.go` and `financialissuancesetupreferences.go`, with registration in `financialissuancesetupmutation.go`, `financialissuancesetupcontext.go` and `trusthold.go`. The configuration check of the discovery reads was split from its reader check so that writers without Inspect reuse it. The consumption rules are not implemented; no protected write consumes an attestation yet.

The independent-reviewer agent (read-only, model reported as claude-fable-5-2) reviewed the working tree. Conformance, authority, disclosure, text admission and bounds, and correctness passed with no code defect; the gate accepted the four capability actions at registration. Tests failed the review: the authority matrix, foreign and unconfirmed configurations, bounds exactly at the limits, a changed actor on one identifier, the total capacity, the fault after withdrawal, withdrawal for three of the four purposes, paging and a claim consumed by a channel shared with a second configuration had no assertion. Root added those cases in one pass and adopted the suggestions: the serialized-size admission now precedes the configuration checks, a receipt that cannot be bounded after a write returns the uncertain outcome, and only the two commands that can refuse self-attestation declare that refusal.

Not covered, in addition to the exclusions stated under required verification:

- A delegated step capability reaching these commands. It cannot exist: the four step actions are direct-human actions and the gate refuses to create a delegation for them, which a test asserts for each purpose. The caller check's own delegation refusal is therefore exercised only for Inspect, by the existing discovery tests.
- Consumed states exist only through package-internal map edits until the item 3 writes exist.
- The references read's cursor refusal and the refusal of a target digest on a new target are enforced by the framework's undeclared-field check before the handler, so the handler's own clauses are not exercised.

The bounded corrective re-review returned PASS for merging with no blocking finding. Executed on the reviewed tree: `go vet ./apps/api/... ./modules/...` clean and the full `go test ./apps/api/... ./modules/...` passing 81 packages (`apps/api/server` 367.701 seconds). Three test-only edits the re-review suggested were made afterwards and checked by rerunning the attestation, hold-table and source-rule tests, not the full suite. The reviewer executed no tests.

