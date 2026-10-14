---
type: Source Based Review
title: Existing document omission check of the planning reviews
description: Rechecks original planning clauses against current documents and prior reports and distinguishes newly localized gaps from source loss unresolved policies and editorial advice.
status: draft
---

# Existing documents as the review baseline

Tracking: [HB-15](https://linear.app/hybrid-billing/issue/HB-15). Accountable owner: Seung Woo Park. Root owns PM/planning and report integration; separate read-only architect, DBA and QA reviewers perform source-first comparisons. Machine: `mac_mini`. Date: 2026-10-14. Only this review report is written; original planning, schema and screen files remain unchanged.

Owner clarification: check omissions against **existing documents**, not merely agreement with the writer's report or preferred new features. The reviewed reports are [re-audit.md](re-audit.md) and [purpose-review.md](purpose-review.md). Their report-accuracy PASS statements do not establish source completeness. Earlier fourteen findings were actionable review groups, not an exhaustive clause count or proof of no omissions.

**Source-to-report completeness: FAIL.** This source-first pass identifies three additional concrete relationships or screen-contract details not explicitly localized by the earlier reports. Existing source requirements are preserved, but current detailed schema/screen integration and the report's specific closure guidance are incomplete. No prior financial finding requires retraction. PA-04 is editorial presentation advice, not a new mandatory product feature. No universal zero-omissions guarantee or product/planning acceptance follows.

## Actual reading and revision basis

Fresh collective reread includes every nonempty paragraph and table row/cell in the three verified chapter packs: DBA chapters 1–41, 795 lines in four nontruncated reads; architect 42–82, 776 lines in eight reads; QA 83–123, 747 lines in six reads. Pack headings add lines, so their sum is not a new original-document line count. Root compared the underlying 2,075-line extraction byte-for-byte to the authorized existing extraction: unchanged. Original DOCX provenance and body-block/table-row locators remain as recorded in [re-audit verification](re-audit-verification.md).

| Pack | SHA-256 |
| --- | --- |
| Chapters 1–41 | `b29f6b62006b47c68253f078778affd0bac48c510d181e80f7ff7013e7f5c5eb` |
| Chapters 42–82 | `21b127792f00e8ac208d5592222249bb189180c9a815e39760cb30f0002f0e62` |
| Chapters 83–123 | `8c2cfb15775de45a922dd8b45ae76405ff3836cc8c0a6fea32827b53429203f5` |

Reviewers also inspect relevant existing requirements/details/contracts and current schema or screen relationships. Root freshly reads the required-contract clauses and original locators of all 35 detailed requirement bundles, the recent owner register and the earlier coverage/report dispositions. Prior 43-group/49-screen coverage remains supporting evidence; this pass does not freshly inspect every complete screen body, rendered layout, external source link or runtime.

Fixed bases: merged main `b5c65cf66992f703247b8e4713f86cef7835b767`; prior HB-15 `49db6ee2321f28e87caba53a8b76c66b80fdbd1c`; HB-21 `129c525ba3e942a086a4cbb71b18dc7f5795cf4c`. Remote heads were read back and matched. Proposed HB-18–20 revisions remain as recorded in preceding reports. Authorized local planning companions remain necessary; fresh-clone completeness is not claimed.

## Newly localized source-derived gaps

### SR-01 — MEDIUM: policy-owning Party is missing from the detailed relationship

Original B0293–B0299 distinguishes the platform customer from the billed customer, and separates infrastructure cost, allocation and selling-policy owners. B0299 explicitly requires `owner_party_id`, policy type, scope/version/effectivity, proposer, approver and delegation. Existing requirements `requirements.md:57,221` and chapter 23 retain the policy-ownership obligation.

HB-21 logical catalog:99 records `PolicyRevision / owning domain`, effective scope, approver and activation evidence but no explicit owning Party/delegation binding. A software domain owner is not the legal/business policy owner. The previous purpose matrix corrects the principle but does not localize this missing relationship in its closure instructions.

Required completion: owning Party and policy type, proposer/approver identities, effective scope/version, applicable current delegation and owner-authority check. Unknown future personnel are a policy decision, not a reason to omit the relationship. This is a source-derived schema/linkage gap, not observed unauthorized access or a requirement for new roles invented by the reviewer.

### SR-02 — MEDIUM: supplier invoice identity and cost-document bridge need explicit binding

Original B0096.R004 requires SupplierInvoice issuer/account/document/currency and many-to-many cost-document linkage. B0163 requires reconciliation by issuer, billing account, invoice, period and currency, with explained tax/credit/commitment/late-item differences. Original revision/correction preservation supplies the document-revision obligation. Existing requirements `requirements.md:167`, collection design:26,35 and REC01/REC02/COM02 retain document-scoped reconciliation.

HB-21 generic SourceArtifact/SourceRevision/CostComponentFact and logical:112 ReconciliationCase do not explicitly model the typed supplier document identity, exact linked component scope and bridge amounts. FA-12 concerns ERP documents and owned ERP lines; that is a different relationship and does not close this supplier-side requirement.

Required completion: exact supplier issuer/account/document/revision, compatible billing and usage periods, currency/basis, cost-component links and explained reconciliation bridge. Supplier document originals remain immutable. Independent-tariff/internal operation does not acquire a fictitious invoice prerequisite from this correction. This is a specific detailed-schema and report-localization gap, not loss of the original supplier-cost requirement.

### SR-03 — MEDIUM: demo entry profile and purpose choices are compressed in P06

Original B1286/B1293/B1317 distinguishes public sample viewing, synthetic settlement and failure reproduction; B1317 explicitly selects MSP, standalone enterprise or corporate group. B1318 lists representative scenarios with inputs, event sequence, expected amount/blocker and completion criteria. Existing W38 and `ui-design.md:160` preserve the broad demo scope.

Authorized local P06 `spec.md:3–17` narrows the purpose to synthetic scenarios. Fields :49–55 omit explicit operating-profile and purpose/mode selection; actions :71–74 contain generic run/compare/failure/reset without those fixed choices. Its source/acceptance section :117–133 covers DB-09 and QA-10 but does not explicitly bind SIM01–10/OPSUI06. Prior reports cover isolation and general canonical integration, but do not name this compressed original entry contract.

Required completion: bind profile/purpose and exact scenario/fixture revision to the existing screen/action/result contract, preserving public-sample viewing versus independent synthetic expected amounts and the original scenario acceptance references. Retain mock-only custody and sender isolation; do not treat viewing public samples as validated invoice answers. Undecided fixture versions/hardware are legitimate unknowns. This is a screen/trace omission within existing demo scope, not a new demo product or loss of all HB-38 requirements.

## Rechecked prior findings and classification

| Original clause / existing obligation | Current document comparison | Report disposition |
| --- | --- | --- |
| B0010/B0026/B0029, B0148 | Required common calculation/fact/approval selling dependencies versus internal/recognition purposes | FA-01 / PA-01 confirmed contradiction; original internal route remains preserved |
| B1484; B0477.R002 | Credit issuer identity; positive/correct/approved FX versus component-specific signed money | FA-02/03/05 confirmed key/validation gaps; not missing customer tariff values |
| B0486.R007/B0495 | Monthly adjustment lifecycle is supplied; typed commands/screens/full trace unfinished | FA-04 / PA-03 confirmed integration gap; do not claim the schema adjustment mechanism is absent |
| B0058/B0068/B0073–76 | Native costs, evidence classes and separate customer pricing | PA-02 confirmed role/stage linkage gap; no observed double charging or universal repricing |
| B0935–36/B1059/B1061 | Onboarding diagnostics/configuration, temporary custody and validity versus metrics-only catalog | FA-06 confirmed operational-schema gap; existing onboarding requirements retained |
| B1433/B1439/B1441/B1526–28 | Plugin manifest/activation/compatibility/storage ownership versus run reference | FA-07 confirmed lifecycle binding gap. Citation correction: current merged domain-contracts Extension/release row is :30, not :29 |
| B0082/B0085–92/B1424/B1427–29 | Source-result return versus external event/subscription/delivery/replay | FA-08 confirmed distinct relationship gap; no invention of an extra messaging product |
| B0919/B1414–15/B1515; B1203/B1506/B1513 | Enrollment/auth policy and scoped customer transfer versus generic access/recovery | FA-09/10 confirmed integration gaps; restore is not authorized customer transfer |
| B0533/B0695/B1475/B1481/B1485/B1487 | Commitment, Marketplace, final-work state, opening balance and inquiry versus compressed families | FA-11 confirmed source-specific relationships missing; conditional support stays conditional |
| B0858/B0866/B1493/B1497 | ERP-final/owned-line/issuance/artifact/delivery/result records | FA-12 confirmed integration gap; standard XLSX requirement is already retained |
| B1381/B1388/B1391 | Editable operational notification templates, escalation and alternate owner | FA-13 confirmed screen/action gap; personal preferences do not substitute |
| Full existing source, 35 detailed bundles and owner additions | Full English clause/action/task/case mapping and authorized companion publication unfinished | FA-14 remains open; SR-01–03 localize additional detail under that broad obligation, not three new requirement groups |
| Original five profiles and three operating types | Business responsibilities retained; overview gives greater prominence to selling/technology | PA-04 is editorial presentation advice supporting the purpose; not an original mandatory diagram layout or new product feature |

No assigned architecture finding requires a new original-scope omission claim beyond existing localized gaps. Financial and QA source-to-report completeness FAIL because SR-01–03 were not explicitly localized. This bounded conclusion does not certify every clause-to-implementation relationship or turn untested behavior into a document pass.

## Review responsibility and acceptance boundary

The reviewer starts with each existing obligation and asks where its data/state, actor authority, action, output and acceptance evidence appear in the current design. Missing detail, contradictory semantics and missing action/trace are reported with original locators and current document references. A missing customer-specific tax/rate/calendar/retention value or actual vendor capability remains an unresolved decision/evidence gate; it is not automatically a lost product requirement. Desired diagram presentation and candidate technical solutions are explicitly recommendations.

The current report supplies three additional localized gaps, corrects one citation and classifies earlier findings; it does not remediate original schemas/screens. Follow-up must bind these source clauses through the same independently reviewed planning work and canonical trace, preserving original source IDs and existing scope. All runtime/provider/ERP/DB/load/recovery/user-task tests remain NOT_RUN. Planning completion, implementation and merge remain unauthorized.

## Verification and handoff

Architect, DBA and QA independently reread the actual integrated report and returned PASS for reporting accuracy within their respective scopes. This does not alter source-to-current-design/report completeness FAIL. Root checked frontmatter, English-only prose, balanced fences, local links and credential patterns: PASS. All nine preceding report artifacts were byte-compared with `49db6ee`: unchanged. Actual full staged diff inspection and cached whitespace check PASS: exactly this one English report, no unrelated/source/generated files or credentials detected. Commit/push/read-back remain pending for this new report. The original reports remain historical byte-preserved. Publication uses existing [PR11](https://github.com/myblueton-sw/hybrid_billing/pull/11), not main. HB-15 stays In Progress until authorized integration and merge read-back; release the machine at handoff.
