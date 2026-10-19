---
type: Billing Contract
title: Source-derived ERP issuance handoff and cutover obligations
description: Restores omitted or weakened v1.16 source obligations for issuer changes, automatic posting defaults, post-posting read-back, file handoff, route switching, tax readiness and cutover.
status: draft
sources:
  - resource: ../../../통합_빌링_플랫폼_구현_기획안_v1.16.docx
  - resource: ../../reviews/HB-22-source-omission/findings.md
---

# Source ERP and issuance obligations

Part of Linear ticket [HB-22](https://linear.app/hybrid-billing/issue/HB-22); requirement group HB-22 (prepaid credits and disputes) is a different identifier. This document restores source clauses that the [source omission review](../../reviews/HB-22-source-omission/findings.md) found missing or weakened. Locators follow the [source inventory](../../planning/v1.16/source-inventory.json). It is a planning contract under the [planning gate](../../planning/v1.16/planning-gate.md); it certifies no ERP capability and issues no ERP command.

It supplements PC-04 in [inputs, pricing and ERP](inputs-pricing-and-erp.md), the [planning actions](../commands/planning-actions.md) and the [payment and recognition supplement](payment-and-recognition.md). Submission identity, unknown-outcome lookup and outbox rules there continue to apply. FA-12 (issuance mode and artifact relationships) remains a separate open finding.

## EI-01: issuer and issuance-mode changes (P1-01)

Source: B0155, B0873.R009.

- Self-issuance applies only to contracts where numbering, PDF, tax and correction capabilities are implemented and verified.
- When only accounting connects to the ERP, the platform references the existing invoice and never issues a second customer document.
- **A change of issuer or issuance mode applies only to unissued scope, from its effective date.** Issued documents keep their original issuer and mode.
- **An ERP outage never switches a contract to self-issuance automatically.** Connection failures and mode changes resume after recovery; they are never bypassed through self-issuance.

## EI-02: automatic posting default (P1-02)

Source: B0219, B0850.

**Automatic posting is disabled by default at initial go-live**, even when the ERP profile supports guarded posting. HB-22 proposal, not source text: enabling it later is a separate approved profile change; the enabling authority and timing are open decisions. Without a protection mechanism (pre-posting amount lookup or draft lock, or a verified alternative such as an SAP billing block, Odoo permission control, server extension or ERP-internal manual approval), automatic posting is unsupported.

This supersedes only the default-enablement sentence in `docs/planning/v1.16/architecture.md` (the external-execution paragraph, around line 112), which reads "do not enable automatic posting without draft-version protection": protection is a precondition, not a reason to enable by default. The transaction, Claim reservation and outbox rules in that paragraph are unchanged. Amending that local companion is an open follow-up recorded in the [HB-22 verification](../../reviews/HB-22-source-omission/verification.md). A file path is never shown as ERP-applied before the external document number and amount are confirmed. Contracts without an ERP are verified with self-issuance and customer delivery files.

## EI-03: post-posting read-back (P2-07)

Source: B0850, B0873.R003, B0873.R005.

After posting, the adapter **reads the final ERP document again and compares it with the pre-approval snapshot** (amount, tax, lines, currency, customer, product). A difference before posting invalidates approval and requires re-review. A difference after posting is flagged and enters correction review; the approved snapshot is never overwritten to match. A manual ERP edit that changes the draft version requires reapproval.

## EI-04: ERP master data, period locks and replacement (P2-08, P2-06)

Source: B0846.R003, B0870, B0871, B0873.R006.

- Customer, company and product masters use the master designated at onboarding. The platform reads and maps against it; **bidirectional overwrite is prohibited**.
- A locked accounting period or inactive master returns an error and holds the submission. **The platform never substitutes an arbitrary posting date or product.**
- Business Central's `salesInvoices` creation returns a draft object, so creation and subsequent posting are separate adapter steps. ERPNext links Frappe REST, document methods and Sales Invoice submission. These are dated official-capability observations, not implementation evidence.
- **After an ERP replacement, historical documents remain traceable through their original connection and identifiers.** A new ERP connection never rewrites past document references.
- ERPs without an approved API use the standard XLSX and verified product-specific upload files; generation, download and external application are distinct states with registered result evidence.

## EI-05: standard settlement workbook (P3-03)

Source: B1032.R001–R007, B1035, B1040.

The standard MSP settlement workbook contains:

| Sheet | Content | Rule |
| --- | --- | --- |
| Guide and handoff summary | File ID, revision, period, entity, target ERP, owner, purpose | Marked as manual-entry or upload |
| Invoice header | Customer code, document reference, currency, invoice date, payment terms | Values approved by the settlement owner |
| Invoice detail | Line reference, service, quantity, unit, selling price, discount, net | Fixed calculation result linked to source |
| Tax and adjustment | Tax code, finalization flag, rounding, credit, total | **Values the ERP will determine are marked not final** |
| Customer and accounting mapping | Customer, product, account, cost center, project codes | Only fields the ERP type needs |
| Reconciliation result | Totals by currency and document, differences, missing, block reasons | Unexplained differences never adjusted arbitrarily |
| Application result form | Source line ID, ERP number, status, failure reason | Separate from approved-amount edits |

Each template version carries ERP product, version, file format, required columns, codes, character encoding, tax responsibility, rounding rules and test history. **Unmapped codes block generation of the upload file.** Large outputs split on customer, period and document boundaries, with a file list, row counts and totals; the full source stays as separate CSV evidence.

A customer-facing XLSX contains only permitted data (invoice, own usage, sales detail). Internal sheets are never hidden or password-protected inside a customer file. **The customer file is generated as a new file** so hidden rows or sheets, comments, defined names, external links, caches and document properties carry no sensitive cost or other-customer data.

## EI-06: API-to-manual route switching and recovery (P3-04)

Source: B1044, B1047, B1048.

- File generation fixes submission ID, billing revision, customer, entity, target ERP and line IDs. The operator registers either the ERP upload result file or manually entered document number, date, amount and status. The system reconciles them with the original statement and **records operator confirmation separately from automatic ERP confirmation**.
- Before switching from API to manual: **lock automatic dispatch**, then look up in-flight and unknown-result requests. Only transactions confirmed as not applied are assigned to the manual route, **by approval of a designated owner**. If confirmation is impossible, provide a review-only file and hold re-entry submissions that risk duplication. API and file routes share one submission ledger.
- After connection recovery: **synchronize manually applied ERP numbers and statuses first**, and never recreate completed transactions through the API. Where the ERP does not enforce reference uniqueness, operator re-entry cannot be fully prevented technically, so submission history, **duplicate-candidate checks** and result reconciliation compensate. Manual handoff is never an unconditional retry path for automatic submission.

Proposed commands. They are **not yet registered** in the [planning actions](../commands/planning-actions.md) registry; registration is a follow-up. Each uses that registry's common request, idempotency, revision and error envelope.

| Command | Required body | Permission and guard | Outcome |
| --- | --- | --- | --- |
| `erp_route.lock_dispatch` | ERP connection and scope, expected route revision, rationale | Scoped `erp_route.lock` | New automatic dispatch stops. Outbox rows already queued are **held, not dropped**, and remain in the shared submission ledger |
| `erp_route.assign_manual` | Submission IDs, each with confirmed not-applied evidence, expected revisions, rationale | Requested by an operator; executed only after approval by a designated owner holding `erp_route.assign_manual.approve`. **The requester cannot approve their own assignment** (`self_approval_denied`) | Assigned submissions move to the manual route; unknown-outcome items get a review-only file |
| `erp_route.register_manual_result` | Submission ID, ERP document number, date, amount, status, or an ERP result file reference | Scoped `erp_route.register_result` | Reconciled against the original statement; recorded as operator-confirmed, distinct from ERP-confirmed |
| `erp_route.sync_after_recovery` | ERP connection, expected route revision | Scoped `erp_route.lock` | Manual ERP numbers and statuses synchronized first; completed transactions never recreated; duplicate candidates flagged |
| `erp_route.unlock_dispatch` | ERP connection, expected route revision, rationale | Scoped `erp_route.lock`. Allowed only after `sync_after_recovery` has finished and no unknown-outcome item remains unassigned | Held outbox rows for not-applied submissions resume; others stay held for review |

Permission names and states are proposals, not deployed APIs or role grants.

## EI-07: readiness states, tax readiness and cutover (P2-04, P2-05)

Source: B0714, B0902, B0950, B0219.

- Reconciliation completion does not certify supplier price legitimacy or tax compliance. Document receipt, scope reconciliation, customer-sale verification and **tax-integration verification are separate states**. For customers that need a tax integration, it is part of the go-live completion condition.
- Installation complete, source readable, billing verifiable, ERP issuance ready and operational go-live are separate states. Account authentication alone is never called billing-connected. Go-live is allowed only for contracts that completed an approved issuance route and recovery test.
- Cutover runs **two periods of parallel reconciliation against the existing billing, while issuing from exactly one designated system**. The platform never issues customer documents for a period the legacy system also issues.
- Before go-live, test MSP source download, customer disclosure policy, recipients and CSV totals. Go live per contract as each passes.
- The operational handover package includes owners, close calendar, permission renewal, backup, correction procedure, file delivery, incident contact path and configuration version.

## EI-08: provenance labels for exports (P2-12)

Source: B0977.

The preserved provider CSV is released unchanged only to a principal authorized for its full customer scope, after the current-authority recheck of the [access and trust contract](../security/access-and-trust.md); others receive a labelled source-based extract. For API-only sources, response evidence and the converted CSV are distinct; **the converted CSV is never called the provider's original file**. A partial-row extraction is labelled a source-based extract.

## Synthetic acceptance cases

All cases are NOT_RUN.

| Case | Finding | Scenario and expected result |
| --- | --- | --- |
| H22-E01 | P1-01 | ERP outage during close: no automatic switch to self-issuance; submissions remain held or unknown until recovery |
| H22-E02 | P1-01 | Issuer change effective mid-period: issued documents keep the old issuer; only unissued scope from the effective date uses the new issuer |
| H22-E03 | P1-02 | New contract with a guarded-posting-capable ERP: automatic posting is off at initial go-live |
| H22-E04 | P2-07 | ERP reprices or changes tax after posting: read-back detects the difference and opens correction review; approved snapshot unchanged |
| H22-E05 | P2-08 | Locked period or inactive product master: error and hold; no substitute date or product. ERP replaced: historical document links resolve through the old connection |
| H22-E06 | P3-03 | Upload file with an unmapped code: generation blocked. ERP-determined tax: marked not final. Customer file: no defined names, external links, hidden sheets or document properties carrying internal data |
| H22-E07 | P3-04 | Switch to manual with three in-flight requests (one applied, one not applied, one unknown): dispatch locked; only the not-applied one goes to manual with designated approval; unknown gets a review-only file |
| H22-E08 | P3-04 | Recovery after manual entry: manual ERP numbers synced first; no API recreation; a duplicate candidate is flagged |
| H22-E09 | P2-04 | Reconciliation complete but tax integration unverified for a customer that requires it: go-live not complete |
| H22-E10 | P2-05 | Parallel-run period: exactly one system issues; the other produces comparison only |
| H22-E11 | P2-12 | API-only source export: labelled converted CSV or source-based extract, never provider original |
| H22-E12 | P3-04 | Operator requests manual assignment and attempts to approve it: rejected with `self_approval_denied`. Unlock requested while an unknown-outcome item is unassigned: rejected; queued outbox rows remain held |
