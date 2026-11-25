---
type: VerificationRecord
title: Dedicated result discovery and read-only action readiness
description: Historical preparation references and current action advice without document permission substitution or ledger writes.
status: draft
---

# Scope and contract

HYB-341, machine:mac_mini. Root implements the protected prerequisite reads in the independently reviewed [screen plan](result-screen-plan.md): result_evidence.preparations, context and advice. This increment does not implement the B09 result screen or provisioning controls.

Discovery requires dedicated result Cost rights before lookup and the exact current workspace target. It validates retained service-owned preparation/document/charge/candidate association and exposes sorted bounded rows without unrelated document Cost permission. Context supplies reference line/product/unit only, never reported amounts. Proposal permission and readiness are distinct; hold blocks proposal, missing rights leave readiness unchecked, and unavailable ledger observations do not remove valid retained context. Paging reuses the exact service-revision cursor contract and bounded output admission.

Advice reports confirm/revoke/apply/recover separately. Permission checks precede output-admission checks, then applicability/current-fact checks. Confirmation permits well-formed mismatches and stale financial producers; revoke remains available under hold/stale producers. Fresh apply requires no existing owned application. Recovery requires the original actor and receipt-empty ownership, current evidence/rights/watermark and exact read-only ledger observations. Recorded receipts make apply/recover inapplicable. Inspection/advice never reserve, apply or settle an operation.

# Independent corrections and actual evidence

refusal_review passed discovery/context source and requested more historical/cursor coverage. It identified missing confirm/revoke output admission and a missing prospective application-size check in advice. Root added output admission for every allowed action, extracted shared pure resultApplicationAdmission for actual writes and read-only readiness, removed a duplicate late admission call and documented output-error precedence. The reviewer passed the substantive corrections; latest log references are supplied separately from the earlier failed fixture attempt. No reviewer test execution is claimed. platform_refusals independently passed the evidence/terminology boundary and requested the plan clarify context owns propose permission while advice owns confirm/revoke/apply/recover; root corrected that wording.

| Check | Result | Evidence |
| --- | --- | --- |
| Initial discovery/context race | Passed, 2.853s | /private/tmp/hyb341-result-context-race.log |
| Current read-only readiness/ownership/context race | Passed, 5.400s | /private/tmp/hyb341-result-readiness-final.log |
| Output admission and authorized foreign-actor recovery race | Passed, 2.390s | /private/tmp/hyb341-result-advice-corrections.log |
| Reviewed focused race after duplicate-call removal | Passed, 6.171s | /private/tmp/hyb341-result-readiness-reviewed-race.log |
| Final vet | Passed | /private/tmp/hyb341-result-readiness-vet-final.log |
| Full Go short regression | Passed all 83 packages; server 148.120s | /private/tmp/hyb341-result-readiness-full.log |

Actual tests remove all document-evidence grants while retaining result Cost rights; discovery/context still work while document history refuses. They cover exact workspace targets, corrupted submission association, Cost-before-lookup, held proposal, revoked proposer permission, end-cursor/header consistency, stale cursor, revoked document with unavailable ledger retaining historical context, mismatch confirmation, held revocation with absent financial producer, unknown/recorded applicability, newer watermark, unavailable recovery observation and a different fully authorized actor refusing owned recovery without state changes.

An attempted eleven-preparation fixture reused the same economic source. The actual ledger correctly refused already_submitted, both with reused and new financial request IDs. Root preserved production duplicate protection and replaced that invalid fixture with an honestly scoped end-cursor/header/stale-cursor test. Multi-page discovery over distinct economic sources and historical context after financial-approval withdrawal are still additional coverage, not claimed passes. Existing generic paging tests are not substituted for that integration evidence.

# Remaining work

Actual B09 composition guidance has been requested from Claude CLI against predecessor renders and existing source. Invocation is not completed design review. UI implementation, session-owned frozen attempts, result-only entry independent of approval-list access, M03/P02 provisioning workflow and actual EN/KO/AR rendered journeys remain required. Replacement protocol, StageERPFinal and issuance/delivery remain unfinished under the original goal. No external ERP effect, production persistence, owner FIX, native-language certification, merge or deployment occurred.
