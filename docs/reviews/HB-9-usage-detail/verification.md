---
type: Design Verification
title: HB-9 usage detail design verification
description: Records independent review and scoped document validation without certifying runtime disclosure or durability.
status: draft
---

# HB-9 verification

Ticket: [HB-9](https://linear.app/hybrid-billing/issue/HB-9/define-cost-and-usage-detail-disclosure-and-durable-lifecycle). Branch: `docs/HB-9-usage-detail-lifecycle`. Accountable owner: Seung Woo Park. Worked machine: `mac_mini`. Root is sole writer/PM/planner; separate read-only DBA and QA reviewers assess financial/data and disclosure/lifecycle contracts.

Scope: one English usage-detail design, this verification, and six existing English entry/reference documents. Existing untracked local planning/source/screen assets are preserved; detailed JSON/screen/trace reconciliation remains PP-08. No product implementation or exact stack/retention/format selection.

| Check | Result | Evidence and limit |
| --- | --- | --- |
| Independent DBA design review | PASS | Source-to-measure-to-result-to-document lineage, consistent bases/totals, immutable history and safe archive/rebuild obligations |
| Independent QA design review | PASS | Current authority on fixed data, safe complete disclosure/export, honest granularity and retention/restore states |
| Changed-document independent review | PASS | DBA/QA reviewed the concrete detail/lifecycle contract; both separately re-reviewed pricing addition, UD-11/12 and linked scope updates without blockers |
| Scoped English/OKF/link/whitespace/scope validation | PASS | Eight-file English/OKF/type/status/link/fence/common-credential checks passed; root inspected complete staged document scope and whitespace; twelve acceptance designs remain NOT_RUN |
| Runtime/UI/query/export/pricing/storage/retention/restore/performance | NOT_RUN | UD-01–12 are acceptance designs, not executed product tests |
| CI/branch protection | UNVERIFIED | No automation installed or CI pass inferred |

Owner requirement is confirmed. The contract refines proposed designs; physical engines, support/retention values, exact numeric rules, export formats and SLOs remain unresolved. HB-6/HB-8 findings are not closed. Owner planning-completion declaration remains absent.

During work the owner also requested how pricing logic is determined/finalized. The live ticket scope was updated before adding separate policy and charge-finalization lifecycles, a decision/responsibility packet and UD-11/12. Actual rates, full operation order and warning policy are not adopted; PC-01/02 remain open. Both independent DBA and QA re-reviews of this addition completed PASS.

PR/authorized merge/read-back and final machine release are recorded in the live ticket after they occur. Completion of this document task does not certify implementation or whole-planning completeness.
