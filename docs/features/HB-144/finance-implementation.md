---
type: ImplementationEvidence
title: HB-144 finance seven-screen frontend implementation
description: Records dedicated domain surfaces, local interactions, source boundaries and scoped verification for I01–I04, R01–R02 and B02.
status: draft
---

# Scope and evidence

Tracking: [HB-144](https://linear.app/hybrid-billing/issue/HB-144/implement-the-32-remaining-commercial-frontend-screen-candidates). Root owns integration and independent review; this writer owns only `apps/console/src/modules/issuance/remaining/` and this record. The isolated branch is based on the HB-144 batch plan commit, stacked on B11 PR88, not fresh main. Other writers' files and the prior 17 screen candidates remain untouched. No owner FIX, merge, deployment, financial operation or public delivery occurred.

All seven original authorized specifications were read from the companion workspace. Exact source hashes and linked screen tickets are recorded in [screens.json](screens.json). The frozen client presentation field/view map is `contracts.ts`; it is not a deployed response schema or backend policy. The parent explicitly confirmed execution may use dedicated unconnected surfaces and local input validation without inventing a synthetic evidence protocol. Therefore these leaves accept no server data and make no API requests. There is no browser URL, storage or window switch producing sample financial evidence.

# Implemented screen candidates

| ID | Dedicated composition | Views | Working local interactions |
| --- | --- | --- | --- |
| I01 | Invoice document sheet, responsibility list, original/correction panels, separate approval/delivery and payment/recognition evidence | 6 | Scope, issue responsibility, currency and document-reference input validation; clear; requirements disclosure |
| I02 | ERP comparison workbench, attributed versus whole-document inspection, row linkage, events and capabilities | 7 | Scope/product/outcome criteria; comparison scope and measure selection; validation; clear |
| I03 | Handoff ledger, workbook manifest, generation versus application workflow and per-row reconciliation | 5 | Scope/template/submission criteria; review/upload/monthly file-purpose selection; manifest disclosure; validation; clear |
| I04 | Explicit customer-data composition beside restricted projection, artifact validation and type-specific hold interpretation | 4 | Recipient-reference validation; three category toggles with default exclusion; dynamic projection; clear; hold-effect selection |
| R01 | Analysis criteria, monthly missing-data trend with equivalent table, comparison, temporary view configuration and export evidence | 7 | Single amount meaning/currency/basis/dimension selection; date validation; browser-only temporary configuration; clear |
| R02 | Occurrence/receipt audit table, actual-principal detail, before/after revision panels and retained evidence | 4 | Time-basis, date, actor kind and fixed-reference criteria; validation; clear |
| B02 | Historical asset table, separate event lanes, intervals, gap/reset evidence and model/meter semantics | 4 | Period/asset/measurement-grade criteria; measurement-kind explanation; validation; clear |

The 37 views are dedicated domain bodies within the existing IA and review shell. Root supplies canonical tabs, navigation, review controls and screen context. The export accepts screen ID, locale, query parameters, view and state/dirty callbacks. Scope, locale or view changes remount the leaf and clear ephemeral inputs. Review state omits user-entered references and names; revisions and nonsecret control state distinguish edits. No credentials or payment-account values are requested.

Read-only evidence remains explicitly unknown until a reviewed producer is connected. No fake rows, counts, amounts, payment state, successful ERP application, artifact generation or hold effect are displayed. Unknown receipt authority never becomes paid or overdue. Monthly expense recognition remains distinct from actual posting and quarterly billing. There is no arithmetic, implicit unit conversion, original-document overwrite or fake downstream command. I04's local public composition has only invoice, own usage and permitted sales categories; it never accepts internal values and is not authorization or a complete policy/inference checker.

# Visual design and external evidence

Actual Claude CLI `ui-designer` composition invocation succeeded using the installed official frontend-design skill, the seven original specs, the batch plan, and existing S04 EN 1440 / AR 390 screenshots. Response: `/private/tmp/hb144-finance-design.txt`. It explicitly reported its inspected sources and limits. It did not fetch the external pages itself; the writer separately consulted these official sources:

- [Cloudscape view resources](https://cloudscape.design/patterns/resource-management/view/): adopt tables for numeric/text comparison, rather than identical cards for every domain.
- [Cloudscape split panel](https://cloudscape.design/components/split-panel/): adopt contextual secondary detail, stacked below content at smaller widths.
- [Stripe invoicing overview](https://docs.stripe.com/invoicing/overview): adopt separate invoice lifecycle and payment concepts only; do not import Stripe authority, finalization, collection policies, assets, dependencies or setup instructions.

Implementation uses the existing local IBM Plex fonts and semantic neutral tokens, full available width, 16px body/controls/table, 20px section headings, 14px metadata, square corners and no shadows or decorative colored edges. Arabic mirrors layout while IDs/date inputs remain LTR. The writer rejected invented unknown-result HTTP codes, decorative glyph icons, fake trend bars, client finance/date eligibility calculations and automatic allowance of unresolved disclosure fields, even where the designer suggested them. Layout remains product-specific; shared code is limited to criteria/evidence primitives.

# Verification and limits

PASS: Final TypeScript `npm run typecheck --prefix apps/console` after all source adjustments. All maintained source files are below 1,500 physical lines.

PASS: An isolated Vite/Chromium body harness rendered 37 views × KO/EN/AR × 1440/390 = 222 variants, with 84 full-page screenshots, no page errors or viewport overflow, 20px section headings, 16px controls and square controls. Seven primary local form journeys validated missing inputs, valid criteria, local-only feedback and clear behavior; I04 also verified category projection and clearing. Input values did not enter the reported review state. The first interaction runner used an immediate DOM assertion and reported an I03 validation failure; a focused reproduction showed valid state. The corrected runner waits for the rendered status and passed. This is isolated body evidence, not canonical App, backend or accessibility certification. Report and captures: `output/validation/HB-144-finance/` (generated, excluded from Git). A few final explicit field slots and review-state additions require the parent's integrated pass.

UNVERIFIED: Canonical App routing/shell and review-control integration, final rendered Claude comparison, final independent financial/security/TW source review, native Arabic and human billing-practitioner acceptance, screen-reader operation, every browser/OS, real producer authorization, pagination/snapshot contracts, live data, downloads, ERP, full-file sanitization, full-row validation and all financial/public side effects. No source claims those exist. Product portability uses standard React/browser APIs and locked tooling; execution evidence is macOS/Chromium only.
