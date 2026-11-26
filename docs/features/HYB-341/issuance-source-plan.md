---
type: ImplementationPlan
title: Owned economic source for the governed issuance draft
description: Retain exact current approved result and contract provenance before adding separately governed document settings.
status: draft
---

# Scope

HYB-341, machine:mac_mini. Root owns apps/api/server/financialissuancesource.go and focused tests; refusal_review independently reviews this contract and change. The next issuance draft needs exact contract, document parties and reported line identities as well as final gross. The current final subject alone intentionally exposes neither contract provenance nor all rows. Build a detached private source basis from owned producer records; do not introduce a second caller-supplied monetary input.

# Proposed behavior

A private service helper takes workspace, final request ID/digest and actual consumer ID. It first calls currentFinancialFinalLocked, preserving actual session/admission, hold, current recorded approver permission/owned amount authority, current upstream/result/policy/ledger and approval CheckCurrent. Only then read the matching service-owned final binding, retained result basis and original candidate. Require exact candidate digest and document/contract/billing revision linkage. Derive workspace, economic target, customer, issuer/buyer/payer document key, contract and billing profile revisions, period, currency/scale, exact reported rows/net/tax/gross, result digest/application receipt and original contributors. Preserve original reported amounts and complete row identity; no summing, rounding, display text or inferred legal values.

Bind the returned detached value to a versioned digest over these facts and the exact final subject/decision. Repeated reads with unchanged evidence are stable; caller mutation cannot change any retained producer. Unsupported source schema/profile, missing approved final state, changed provenance or lost current rights refuse without publishing a draft or reserving an execution. No public command, default grant or new service-owned registry is added here.

# Document-specific inputs remain separate

This output is not an issuance.Draft and cannot select an issuance mode or bypass draft approval. It includes no fabricated recipient, payment account, template, language, tax verification, external number or issuance time. Those facts must come from separately governed mode/profile, protected and receivable registries in the next adapter. Existing ExternalInvoice path alone cannot distinguish self issuance, ERP issuance and accounting reference. Public source reads and mutations require separately reviewed current permissions; this private helper establishes no new authorization domain.

# Verification

Use actual synthetic public final request/approval and the existing result/application chain. Assert exact two-line reported net/tax and total, actual contract/party provenance, stable digest, defensive copies, no service revision/registry mutation, and refusal for pending final state, digest mismatch, revoked approver authority, missing producer and ended/foreign session. Preserve current result replacement and ledger drift rejection via the already-required current helper. Scoped server/race and source checks must pass; no new UI or issued document is claimed.

# Status

Independent concrete design review passed with explicit result/receipt, owned charge/candidate, contract/profile/party revision, full-row and digest linkage conditions. Root implemented the private detached helper. Independent source review passed without finding a linkage or aliasing blocker; it requested exact refusal assertions, which were added.

Focused source race passed4.259s. The combined source/final workflow/source-rule race run passed46.598s; server vet passed. An initial exact row-ID assertion used IDs from a different fixture; corrected expectations use the actual publicResult row-0/row-1 source, preserving independent amounts10.25+4.75 and tax0.25+0.75. The final exact-refusal race suite passed4.326s. The ended-session fixture now removes the service session after the access answer is obtained, proving the helper rechecks the live session; removing it before fixture entry correctly failed at the identity boundary and did not exercise the helper. Tests verify actual public final request/approval, contract and document read-back, stable digests, defensive rows/contributors/subject/component copies and unchanged service revision/approval execution records.

The returned billing-profile recipient is an organizational party identifier, not a verified protected delivery address. No customer contact, template, mode or external issuance is inferred. Public permission governance, protected fields/terms, retained issuance draft and subsequent execution remain incomplete. This is a prerequisite, not a replacement outcome.
