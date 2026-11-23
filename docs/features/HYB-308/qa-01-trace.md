---
type: TraceReport
title: AUD-QA-01 on O03 against what is built
description: Clause-by-clause trace of the payment and recipient change control requirement against the issuance setup work merged under HYB-341, to establish what remains for HYB-308.
status: draft
---

# Scope and method

HYB-308, machine:mac_mini. The ticket asks for the O03 control of AUD-QA-01. The trace classification of 2026-11-17 (`docs/reviews/HYB-301-trace-classification/report.md`, O03 AUD-QA-01) found nothing of it built and the Documents tab marked as not connected. Since then HYB-341 has merged a protected settings module binding, public commands and the O03 documents view. This report traces each clause of the requirement against main at afe69fa on 2026-11-24, by reading code and tests. Nothing new was executed for it; where a test is cited, it is one that already exists and passes in the suite run recorded under HYB-341. No product code is changed.

Requirement text, translated from `docs/planning/v1.16/detailed-requirements.md`, QA-01 and AUD-QA-01, and repeated in the QA-01 section of `docs/ui/screens/O03/spec.md` (that specification and the planning files are in the authorized local workspace and not yet in the published repository; its protected reference row names payment information, recipient and template and omits the tax identifier): changes to a payment account, document recipient, tax identifier or template require separate verification, approval and a validity period; the references before and after, the verifier and approver, and the evidence of confirmation through an existing registered channel are preserved; an account is not applied on a new email instruction alone, and an LLM does not generate or replace payment information; the control is mandatory even though the responsible people and the specific periods are undetermined. Acceptance: an account change with only a new email, and an LLM-generated account, are refused, and a change is reflected in documents only on its effective date after confirmation through the existing channel and separate approval.

# Trace

| Clause | State | Evidence | What remains |
| --- | --- | --- | --- |
| Payment account change | built as a candidate with synthetic references | `protected.FieldPaymentAccount`; request, verify, decide commands in `financialissuanceprotectedwrites.go`; O03 changes section; real-server run in the HYB-341 handoff | real references |
| Document recipient change | built as a candidate with synthetic references | `protected.FieldDocumentRecipient`; same commands and section | real references |
| Template change | built as a candidate with synthetic references | `protected.FieldDocumentTemplate`; same commands and section | real references |
| Tax identifier change | not built in the service | the module defines `FieldTaxIdentifier` (`modules/protected/types.go:40`), but the service's list of protected setup fields has three entries and omits it (`financialissuancesetupprotected.go:131`); no command, read or screen offers it | the field in the service for its two subjects (issuing entity and customer), its sample references, the reads and a screen placement; the issuance module needs it only for statutory document kinds, which the service refuses today, so it has no consumer on main; four existing tests assert that it is refused |
| Separate verification | built | the verifier must differ from the requester and from the channel's registrar and admitter (`modules/protected/changes.go:199`, `:218`); the requester may not be a custodian of that channel either, and the verification channel must differ from the channel the instruction came through; suspension and real-server tests | none identified |
| Separate approval | built | the approver must differ from the requester, the verifier and the channel's custodians (`modules/protected/changes.go`, approval); the number of approvals comes from the rules | none identified |
| Validity period | built | protected rules carry a validity, a maximum lead and a channel age; an unverified change lapses at its effective time and a verified one at the validity period (`modules/protected/changes.go:52`) | the periods are provisional values; see HYB-349 on the fixed 24-hour values. What is built is a window for approval after verification, ending no later than the effective time; whether that is the requirement's validity period is a reading for the owner |
| References before and after preserved | built | the change record keeps `Before` and `After` (`financialissuancechange.go:31`) | kept in process memory within the synthetic boundary; nothing survives a restart; durable retention is not built. The reads withhold them from setup readers by contract; whether a reviewer of the change may see them is not designed |
| Verifier and approver preserved | built | the change view keeps the verification with its verifier and each decision with its approver | kept in process memory within the synthetic boundary; nothing survives a restart; durable retention is not built |
| Evidence of confirmation through an existing registered channel preserved | built as a human statement | verification names an accepted channel that is at least the rules' channel age old and records a statement of the verifier as its evidence (`issuance-setup-evidence-contract.md`) | kept in process memory within the synthetic boundary; durable retention is not built. The statement is an accountable claim and not proof that the channel was used; no message is sent or received by the service, as the classification report anticipated |
| No account on a new email instruction alone | built and tested | `TestFinancialProtectedWritesEmailOnlyChangeNeverAppliesAlone`: a request whose instruction came from an unregistered sender is recorded as requested, cannot be approved without verification, never resolves and lapses | the acceptance says "refused"; the service records the request and withholds its effect until verified. Such a request can then be verified through a different accepted channel, approved and applied, which matches "after confirmation through the existing channel"; the test does not exercise that path. Whether recording the request meets "refused" is a reading for the owner |
| No LLM-generated payment information | built and tested | the same test: a request with assistant-generated provenance is refused and nothing is retained; the module refuses it as well (`modules/protected/changes.go:118`) and refuses non-human actors | the provenance is declared by the requester; nothing detects an undeclared generated value |
| Reflected in documents only on the effective date after confirmation and approval | partly built | a draft review resolves each reference at the review time, so only a scheduled change whose effective time has passed is used (`financialissuanceconfiguredconsumption.go:46`); `TestIssuanceJourneyReachesApprovedDraftsForTwoInvoices` | no document is issued by the service yet, so "documents" is shown for the draft review only |
| The control on O03 | built as a candidate | ten slices of the O03 documents and payment terms views (HYB-341 screens plan); real-server run in three languages | owner screen review; native-language review |
| The control on I01 and B09 | not built | QA-01 names screens O03, I01 and B09; no protected change appears on the invoice screen or in the approval inbox | design and build |

# What stands between this state and closing HYB-308

1. The tax identifier is missing from the service. This is the one clause with nothing built above the module.
2. References are synthetic samples. A real payment account, recipient or template source is not connected; `docs/features/HYB-342/digest-exposure-assessment.md` sets a condition on introducing one.
3. The reading of "refused" for an email-only change, above.
4. I01 and B09 do not show the control.
5. Durable retention of change, verification, decision and statement records; it waits for the storage scope.
6. The decisions recorded in the HYB-341 handoff bear directly on whether the control is usable: who can open the screens (HYB-348), whether earlier actors are rechecked at every later use (HYB-349) and how a record that cannot progress is released (HYB-345).
7. Owner screen review of the O03 views.

None of these is buildable today without something else first. Item 1 needs a design decision on its two subjects, on its screen placement (the O03 and I01 field rows omit it) and on its consumer, since statutory kinds are refused by the service. Item 4 needs the I01 and B09 specifications read first; I01 lists the verification owner and period as open values. Item 2 waits for a reference source and item 5 for storage. Items 3, 6 and 7 are the owner's.

# Not checked

- The source chapter B1500 behind QA-01.
- `docs/ui/screens/I01/spec.md` beyond its field row and its list of open values, and the B09 specification, for what they require of this control.
- Whether the O03 specification's rows "change draft", "billing impact review" and "approval request" for a contract revision are meant to carry protected changes, or only contract terms.
- The module's tax identifier behaviour beyond its field definition.
- The O03 specification's "separate change draft" editability and its financial and personal sensitivity class for these values.
- Whether service change records persist anywhere outside process memory; this rests on the HYB-341 contracts and handoff, not on a trace of the code.

# Review

Independent review (role-based AI reviewer separate from the writer, by reading, nothing executed), 2026-11-24: FAIL with three omissions and no false statement in the trace table — the rows on preservation did not say that records live in process memory only; the tax identifier row omitted its two subjects, its missing consumer and the tests that assert its refusal; and the claim that the tax identifier and the I01 and B09 screens were buildable without a further decision was unsupported. Corrected with the reviewer's wording and references, together with its notes on the verification rule, the meaning of the validity built and the later path of an email-only request. Corrective re-review of the three findings: PASS, with no false statement introduced.
