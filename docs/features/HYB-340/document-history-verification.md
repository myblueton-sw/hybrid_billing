---
type: VerificationRecord
title: Protected document and preparation history discovery
description: Authenticated request-scoped discovery without changing historical ownership or receipts.
status: draft
---

# Implemented scope

[HYB-340](https://linear.app/myblueton/issue/HYB-340), machine:mac_mini. Root implemented the history portion of [the document workspace plan](document-workspace-plan.md). Independent refusal_review reviewed the design, corrective contract and source. The initial design required exact context field budgets, a shared validator without fabricated packet inputs, separated permission/readiness advice and fail-closed owner associations. The corrected design passed, with clarification that every owner must be validated against its own request before filtering; valid other-request owners must not fail a selected request's read.

The new nonmutating document_evidence.history command requires actual human/nondelegated console Cost authority on document evidence before lookup. It reads owned request/digest history, with separate document and preparation kinds. Ten sorted row fields plus seven metadata fields keep a full page within seventeen fields; every value is bounded. Subsequent pages require the same global revision and an existing matching cursor. Any intervening revision requires a fresh first-page read, not a claim of retained snapshot storage.

Pending, confirmed and revoked document IDs/digests are discoverable by another authorized reader. Preparation discovery exposes exact retained document association, actor, key and recorded receipt. An empty receipt stays unresolved. History does not consult or reconcile the ledger, invoke preparation, inspect current producer eligibility, consume approval or advance revision. Dedicated inspections remain responsible for full packet and current ledger observation. History stays available under hold and stale producer state while current Cost authority remains mandatory.

Workspace-local orphan/duplicate proposal associations, invalid owner keys/actors/receipt lengths and historical request mismatches fail closed. Foreign workspace records do not appear. Valid owners of another request are validated before filtering and do not cause refusal. No general connector inventory, defaults, form context, action advice or screen interaction is implemented by this history checkpoint.

# Verification and corrections

The first focused run failed in 0.741s because new assertions incorrectly expected 422 for evidence_stale and 400 for invalid_request. The established public mappings are 409 and 422 respectively. Only test expectations changed; production error mapping was not modified. The corrected run passed in 0.818s. Expanded history/source/hold tests passed in 1.002s. Logs: /private/tmp/hyb340-document-history-first.log, /private/tmp/hyb340-document-history-corrected.log and /private/tmp/hyb340-document-history-expanded.log.

Authenticated scenarios cover another proposer's discovery, confirmation using discovered identity, lost preparation response, revocation under hold, fresh-login discovery with revoked preparation permission, preserved unresolved receipt, revoked Cost before existing/missing lookup, ordinary member with dedicated document Cost access and other direct Cost grants removed, wrong request/digest/kind, producer absence, foreign records, exact seventeen-field pagination, unknown/partial cursor, insertion between pages, empty result, output capability bounds and corrupt ownership. A second real request/document/preparation proves both exclusion from the selected request and discoverability under its own request. Successful and refused reads preserve revision and ownership. SeedNone coverage and the guarded-command inventory include history.

Independent source review first requested stronger member/isolation/integrity/paging tests. Corrective review passed the expanded tests and read the 1.002s root log without executing tests. Context/advice and B09 integration remain outside that scoped verdict.

The document/history/ERP/source/hold scoped race suite passed in 22.047s and server vet completed without diagnostics. Logs: /private/tmp/hyb340-document-history-race.log and /private/tmp/hyb340-document-history-vet.log. Independent platform_refusals compared source, plan and verification against the focused/race logs and passed the evidence review; it did not execute tests.

The final full short Go suite passed all 82 packages, including server in 143.029s. Log: /private/tmp/hyb340-document-history-full.log.

# Remaining outcome

Protected form context and separate permission/readiness advice remain required by the approved plan. B09 document proposal, independent confirmation, revocation, internal preparation and recovery journeys still require implementation, actual Claude composition and EN/KO/AR rendered verification. This API increment makes no new browser, owner acceptance, native-language, production persistence, external ERP, merge or deployment claim. The full owner goal and HYB-340 remain active.
