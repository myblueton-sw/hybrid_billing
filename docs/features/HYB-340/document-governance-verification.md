---
type: VerificationRecord
title: Atomic governed document packets and current preparation authority
description: Actual session-backed proposal, independent confirmation, explicit revocation and historical inspection within the internal ERP adapter.
status: draft
---

# Implemented behavior

HYB-340, machine:mac_mini. financialERP.documents is now the single owner of immutable canonical basis bytes and document governance. A distinct proposal digest binds the basis digest, actual proposer and rationale. Pending/confirmed/revoked state and immutable confirmation/revocation actor/reason/time live in the same record. The ungoverned retainDocumentLocked entry point was removed; validation alone returns a candidate record without storing it. New proposal validates all prerequisites before one bounded insertion. IDs cannot be repurposed.

Actual dedicated user/console authority is checked before lookup. Confirmation requires another human and the original proposer's current direct subject permission, never a fabricated session. Current charge approval, candidate/organization and connection/mapping/survey checks precede confirmation and preparation. Preparation now requires its own PrepareERPRecord permission and a confirmed proposal digest; basis digest alone is insufficient. Explicit revocation blocks new preparation/recovery, including after an uncertain existing module effect, without erasing ownership or ledger history.

Revocation may close pending or confirmed packets during hold and unavailable producers. Confirmation and proposal refuse hold. Repeated identical decisions preserve events; changed actor/rationale or revoked confirmation refuses. Current document confirmer permission is required when confirming, but its later loss does not implicitly revoke an already-confirmed packet. Existing current financial approver checks still apply at preparation. Event writes enforce monotonic time; history reads do not invent eligibility. Separate Cost inspection returns decoded defensive copies and currentEligibility=unchecked.

# Actual test scope and corrections

The real ERP fixture now provisions dedicated rights through actual document_authority commands, proposes under one session and confirms under a different one before preparation. Tests exercise separate proposal/basis digests, copied nested packet fields/events, valid same-ID content conflict, confirmation/revocation replay, requester-independent document dual control, revoked proposer authority, no inherited administrator permission, denied existing/missing reads, pending/stale/held/clock/capacity refusal, explicit revocation semantics and unknown ERP history after packet or preparation-permission revocation. Existing exact net15.00/tax1.00/document quantity5.00 preparation tests still execute with real retained candidates and approvals.

Initial focused tests passed2.769s. The first race run failed14.382s in the pending-state fixture: the returned error was identity refusal before the callback, not the expected command prerequisite refusal. A diagnostic repeated run confirmed a non-command identity error. Fixture construction repeatedly opened sessions at the same fixed time and exceeded the existing10 sessions per user; oldest-session eviction could remove the retained consumer. Temporary governor sessions are now closed and three actor sessions reused. Production session limits and permission behavior were unchanged. The failed logs remain failure evidence.

Independent refusal_review passed the internal design and concrete source, then reviewed the session-fixture correction and unknown-effect permission-revocation test. It inspected corrected root race logs without executing tests. It also passed the supplemental concurrency test, which admits both valid serialized outcomes without claiming both schedules were observed. Independent platform_refusals passed document/source/log comparison, including exact test counts and execution order, without rerunning tests. Public wire schemas, runtime ERP assembly, output capacity and UI are outside those scoped verdicts.

# Executed root validation

| Check | Result | Local diagnostic evidence |
| --- | --- | --- |
| Initial full short suite | PASS82 packages; server136.584s | /private/tmp/hyb340-document-governance-full.log |
| Repeated pending-case race after fixture correction | PASS10 runs,4.911s | /private/tmp/hyb340-document-governance-session-regression.log |
| Corrected financial document/ERP/source race | PASS13.425s | /private/tmp/hyb340-document-governance-race-corrected.log |
| Server vet | PASS, no diagnostics | /private/tmp/hyb340-document-governance-vet.log |
| Concurrent confirmation/revocation and preparation/revocation race | PASS3 repetitions,3.526s | /private/tmp/hyb340-document-governance-concurrency.log |

The full suite passed on the final production implementation before the test-only session-lifetime correction and supplemental dedicated-grant/concurrency coverage. Those changed scenarios ran separately. No production code changed after that full run. Only macOS execution is verified; no new OS-specific dependency, browser change or UI acceptance is claimed. Logs are local temporary diagnostics; source tests are committed evidence.

# Remaining full integration

These are actual private service helpers under the caller-owned domain lock, exercised with real authenticated sessions. Ordinary Server startup still does not assemble this ERP adapter, and no public document packet or preparation command exists yet. The next required step is the strict bounded caller packet builder, server-derived snapshots, named synthetic runtime ledger facts, authenticated command/output/hold/revision/idempotency integration and historical module preparation readback. Explicit tax/quantity evidence remains required; no production tax policy is selected. No external send/post/issuance, financial approval consumption, deployment, merge or owner FIX occurred. W-23 and the owner goal remain active.
