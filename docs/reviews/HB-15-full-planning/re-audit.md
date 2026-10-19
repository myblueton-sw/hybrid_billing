---
type: Comprehensive Planning Review
title: Full source to current planning omission re-audit
description: Reports current financial schema conflicts and source preserved but incomplete screen command operational and canonical planning integration after the HB-21 redesign.
status: draft
---

# Full planning omission re-audit

Tracking: [HB-15](https://linear.app/hybrid-billing/issue/HB-15). Owner: Seung Woo Park. Root owns PM/planning and report writing; separate read-only architecture/security, DBA and QA reviewers inspect their assigned evidence. Execution machine: `mac_mini`. Review date: 2026-10-14. Documentation/design only; no baseline product/source/screen edits, customer policies, product tests, planning completion or merge authorization.

**Overall completeness: FAIL.** No whole original requirement group was found deleted from the authorized local baseline. However, fourteen actionable documentary findings remain: one current approval-subject contradiction, financial key/validation gaps, explicit source-derived schema omissions and screen/command/acceptance integration gaps. Preserving 43 IDs is not proof of clause completeness, and earlier bounded HB-21 PASS does not certify this comprehensive scope. The findings describe implementation risk if the documents are followed prematurely, not observed production failures.

## Actual basis and review method

Original v1.16 DOCX SHA-256: `e7772927bc521ee035d50a7762b2f75611b80b2162e385878dc4c36c6c0db84d`. Root freshly extracted body text and every table row and compared all 2,075 lines byte-for-byte with the authorized extraction: identical. Structural inventory: 1,443 body paragraphs, 102 tables, 1,546 direct body blocks, 123 chapters. B identifiers are zero-based body child positions; R identifiers are zero-based table rows, not rendered pages. Empty paragraphs carry no substantive text. No notes/comments parts were present; page rendering and the 128 original external links were not freshly validated.

Collective original reread: DBA chapters 1–41, architect 42–82, QA 83–123. Each explicitly confirmed reading its whole nonempty text/table extract in bounded nontruncated chunks, with additional cross-domain chapters as needed. Root verifies extraction/provenance and integrates findings; root does not claim independently rereading every source sentence a second time. Every original requirement group has a primary reviewer. QA inspected all 49 screen specifications' field/action/state/acceptance semantics and cross-screen links, preferring ten merged specs and using 39 authorized local specs. This is not a verbatim audit of every screen paragraph or visual JSON/SVG/page rendering.

Merged-main basis: `b5c65cf66992f703247b8e4713f86cef7835b767`, including HB-16 remediation and HB-17 competitor review. Existing HB-15 branch/report at `08d801f` is retained and main merged history-preserving before this re-audit. Earlier [findings](findings.md) and [verification](verification.md) remain historical, not silently rewritten as current.

Proposed inputs remain separate: HB-18 SaaS `6acb38c68117ff1f1865776aca00f87c61956196`; HB-19 gateway/AI/HA `75c4c188e5d42d682d286cebe9900b2bc0bc660c`; HB-20 object/Parquet/capacity `62f7df1`; HB-21 redesigned schema `129c525ba3e942a086a4cbb71b18dc7f5795cf4c`. Reading them does not merge/approve their choices. The latest premise is >100M source records/day, not a measured guarantee or capacity ceiling. Engine/node/profile choices remain open.

See [coverage](re-audit-coverage.json), [input byte inventory](re-audit-inputs.json) and [verification](re-audit-verification.md). The inventory records 161 evidence files/revisions, not a claim each file was read word-for-word. Original/publication gaps are disclosed rather than replaced by guessed policies.

## Findings

### FA-01 — MEDIUM: a selling settlement subject is required for independent approval purposes

**Current contradiction.** HB-21 `logical-data-schema.md:51,113` makes catalog fields required unless conditional, then keys the common ApprovalRequest/Decision/ApprovedSnapshot by a final selling settlement snapshot and monthly-close/service/rate/adjustment dependencies. The same document `:141` and merged `payment-and-recognition.md:44,48` require independently approved cost RecognitionBatch without waiting for customer invoice cadence. Original B0148.R002/R005 and B0849–B0850 distinguish financial approvals and ERP-final authority.

Impact: a designer may force independent cost recognition through a customer selling subject or reuse an approval for a different purpose. **Required:** typed `ApprovalSubject(purpose,snapshot_type,exact_snapshot_ref,digest)` and separate FinalSettlement, RecognitionBatch and ERPFinalDraft dependency sets, authority and effect predicates. This is not an unknown customer rate or an actual runtime failure. Independently confirmed by DBA after root identified the candidate.

### FA-02 — MEDIUM: credit balance identity omits issuing legal entity

Original B1484 requires authority, customer and issuing legal entity. HB-21 `logical-data-schema.md:128` keys CreditAccount by customer/currency/type/authority without issuer. Workspace isolation alone cannot distinguish multiple issuers serving the same customer in one workspace.

**Required:** issuer/legal entity and applicable authority-account in balance/reservation/application identities; explicit approved cross-issuer transfer rules if supported. Original requirement remains preserved; this is a current schema-key omission.

### FA-03 — MEDIUM: positive FX validation is not explicit in the active detailed schema/cases

Original B0477.R002 rejects zero, negative, wrong-pair and unapproved exchange rates. HB-21 `logical-data-schema.md:100`, monthly schema `:54,120` and financial contract `:37` preserve provenance/selection without explicitly carrying the source-positive-rate guard. AR-06's focused acceptance also omits this source case.

**Required:** `rate > 0`, positive quotation denominator, correct pair/direction, approval and effective-time validation, plus source-linked negative cases. Negative monetary adjustments remain independently allowed by field/component policy; exact decimal arithmetic alone rejects neither a plausible zero charge nor a negative FX result.

### FA-04 — HIGH: adjustment application and month-to-quarter invalidation stop at the schema

Original AR-07 requires AdjustmentDraft, explicit target-current delta, full impact, tax/allocation revalidation and fresh approval. HB-21 `monthly-settlement-schema.md:86,92–100,124` supplies the redesigned adjustment and dependent-quarter guards. But local B04 `spec.md:51–52,69–70` offers target/quantity preview followed by **policy** approval; B08 `:65–68,128–145` offers calculation/compare/retry/final approval without adjustment-apply or superseded-month flow; merged B09 `:71,74–76,98–103` and U02 `:71–72,117` do not connect those new identities/states. Shared action registry does not contain the complete redesigned adjustment lifecycle. Acceptance trace has no MS cases.

Impact: a UI/command implementer cannot follow draft → compute → approve adjustment → apply once → new result → independent final approval, nor recover a stale quarterly approval after a late month replacement. **Required:** bind existing B04/B08/B09/U02 fields/actions to typed commands and exact revisions, net/tax/gross/currency, replay/current-result and issued/unknown-dispatch guards; map MS-05/MS-09 to preserved source/work/case identities. New editorial MS IDs are not executed tests or new requirement groups.

### FA-05 — MEDIUM: the FX screen applies a raw-rate sign rule to markup and fees

Local B05 `spec.md:50` combines published FX, sales markup and fees with a blanket rejection of zero/negative. Original B0463/B0464/B0467/B0474 and AR-06 separate those values; merged Claim/numeric `:49` requires component-specific signed semantics.

**Required:** distinct raw-rate, selling markup and fee fields/validation; positive FX while permitting zero markup/fees and separately approved signed components as applicable. This screen conflict is distinct from FA-03's missing source-rate guard.

### FA-06 — MEDIUM: onboarding outcome metrics omit operational setup state

Original B0935–B0936, B1059/B1061 and preserved DB-08/QA-07 bind diagnostic validity to configuration/template/version and narrowly scoped temporary handoff. HB-21 logical `:172` models cohorts/attempts/milestones; evidence `:18–24` models source connections/coverage without OnboardingProject, SetupStep, DiagnosticRun or ConfigurationRevision relationships.

**Required:** exact settings/version/expiry evidence, affected-step invalidation/revalidation, assigned support actor/resource/expiry and distinct setup/export-writer/collector-reader custody. Connection/verified metrics cannot serve as readiness authority. The original onboarding requirement is present, not a new feature request.

### FA-07 — MEDIUM: plugin lifecycle is referenced but not modelled

Original B1433.R001–R003, B1439/B1441 and B1526–B1528 require manifest/digest/API/license/scopes/budgets, compatibility, customer activation and removal/history/storage responsibility. HB-21 logical `:110` pins a plugin revision to calculation but lacks PluginManifest/Revision/Activation/CompatibilityEvidence/storage-owner relationships; merged domain `:29` provides a family summary.

**Required:** trusted immutable manifest/revision, compatibility/license decision, scoped configuration/activation, pinned job digest and owned storage/import/deletion inventory. Preserve initial contract/test/SDK/example obligations independently of conditional external-plugin execution. Unsafe old code cannot be reactivated merely to read historical output.

### FA-08 — MEDIUM: source-result return and external event delivery lack distinct schema bindings

Original B0082.R003/R005/R006, B0085–B0092, B1424.R003–R006, B1427–B1429 and B1524 preserve source envelope, approved invoice/status/download return and external event replay contracts. HB-21 evidence `:20–21` and logical `:145,169` contain generic receipt/inbox/outbox/job fields without the full typed source/result-return and external subscription/delivery/cursor relationships.

**Required:** separate source profile/envelope with request/attempt/workload/unit/quality/correction lineage and permission-safe approved-result return; separate external event/subscription/delivery cursor with source+id, object revision, key version, retention/expired-cursor resync, failure/replay and subscription/send-time current recipient checks. RT-01/QA-05 remain the source contracts; do not turn both into one unnamed event payload.

### FA-09 — MEDIUM: enrollment and authentication-policy lifecycle are absent from the entity catalog

Original B0919, B1414–B1415, B1515 and added HB-42 retention/access enrollment contracts distinguish bootstrap/invitation/purpose token, local realm/login identity and provider/auth policy. HB-21 logical `:92–95` has users/memberships/sessions/trust but lacks those typed relationships.

**Required:** one-use verifier consumption/expiry/role ceiling, local login/realm uniqueness, ordinary IdentityProviderConnection/AuthenticationPolicyRevision and real support/client custody. Trust replacement is not ordinary enrollment; secrets remain protected references. Existing identity documents preserve scope, so this is schema integration rather than a lost authentication requirement.

### FA-10 — MEDIUM: customer transfer is compressed into disaster recovery

Original B1506.R005, B1513 and B1203 require scoped customer transfer, historical import and unfinished final-work responsibility. HB-21 evidence `:66` has RecoveryManifest/WriterGeneration but no transfer package/import identity/recipient acknowledgement or remaining settlement handoff.

**Required:** authorized disclosure/export transfer manifest, read-only historical import/legacy links, deduplicated import identity, new installation binding and open-final-work custody. A coherent restore does not prove an authorized customer migration; imported issued documents cannot replay as new charges and secret material cannot migrate through the package.

### FA-11 — MEDIUM: commitment, opening-balance, Marketplace and inquiry schemas remain compressed

Original B0533.R001–R007, B1475, B1481.R006/B1485/B1487 and B0695.R001–R007 are preserved in local requirements/detail/work/cases. HB-21 evidence `:46,48` reduces commitment/Marketplace state to component flags, and logical `:128,170` omits complete opening-balance migration and BillingCase records. Contract ended versus remaining `pending_final/settled` work, final evidence acquisition and gap-owner records also need explicit relationships; ending the service cannot silently complete its final settlement.

**Required:** typed CommitmentVersion/PeriodCost/BenefitAttribution with provider-supplied residual/savings basis; per-currency/issuer opening-balance import and old-ledger overlap; offer/agreement/participant/eligibility/settlement states; inquiry public/private scope, evidence and resolution distinct from financial effects. Conditional adoption remains conditional. Do not invent a CSP amortization engine or promote product Marketplace selling from later scope.

### FA-12 — MEDIUM: ERP/issuance/export/delivery families lack explicit mode and artifact relationships

Original B0858.R001/R002/R005, B0866, B1493.R001–R006 and B1497 plus preserved DB-03/04/05/06/07/12 require issuer-mode authority, ERP-final draft, owned-line reconciliation, existing order/AP disposition, periodic lookup and stable verified artifacts/packages. HB-21 logical `:113–114,143–145,174` has generic submissions/documents/export records without explicit ERPFinalDraft/MasterMapping/ExternalDocumentLineLink, IssuanceProfile/DocumentSnapshot/ArtifactGeneration, serialization policy or fixed DeliveryPackage/Attempt relationships.

**Required:** mode-specific issuer/number/tax/final-amount approval and posting guards, no reuse of failed/cancelled numbers, foreign-owned-line exclusion, same-document artifact recovery and immutable delivery/result-registration mapping. Standard XLSX is a common required release path, now retained in release-gates `:198–200`; the historical optional-XLSX loss is not a current defect. This is localized FR-06/PP-08 schema integration, not removal of ERP or XLSX scope.

### FA-13 — MEDIUM: editable operational notification rules have no screen/action closure

Original B1381, B1388.R002 and B1391 require operator-editable D-7/D-3/D-1/day templates, stage escalation and alternate owner. U02 `:58–97` covers schedules/owner/reopen, P02 `:49–76` channels/extension/quota, M05 `:52–55,74` personal preferences; none binds operational notification-rule/template revision, escalation targets, impact preview and approved application.

**Required:** configure these rules through existing screens and typed commands with current target/recipient/hold/receipt checks, separate from personal quiet hours. Actual timing/threshold values remain profile choices; the configurable workflow is an existing requirement, not an optional unknown value.

### FA-14 — MEDIUM: later requirements and detailed source contracts are not one portable canonical baseline

Root mechanically verified 43 groups, 129 tasks, 35 detailed additions, 119 original tests with 127 preserved occurrences, 91 existing added tests and 49 screen identities. These counts do not finish the clause/action matrix. HB-18 RA-15/SB-A01–16 remain a separate proposal; HB-19 API Gateway/Client Layer, shared AI/RAG and HA; HB-20 >100M/storage premise and HB-21 redesigned commercial/MS cases are not integrated across original requirement/work/API/screens/acceptance. Main and supplementary entry points still carry historical >1M examples, with supersession explained but full synchronization explicitly pending. Required planning/rules/bindings and 39 screen specs remain authorized-local companion inputs rather than a standalone-clone package.

**Required:** complete English clause → entity/state/command/permission → screen/action → D/I/V task → acceptance mapping with original locators and input revision provenance. Preserve historical evidence; qualify source-group IDs versus identically numbered Linear tickets. Publish authorized companions through scoped reviewed work, reconcile registers and test fresh-clone portability before final PP-09. No count-only check or current review closes this known FR-06/PP-08 obligation.

## Historical disposition and preserved scope

| Earlier finding | Current re-audit disposition |
| --- | --- |
| FR-01 fetch deny | HB-16 supplies metadata/loopback denial and approved private distinction; no recurrence of old exception found. Runtime and final ED-03 closure not certified. |
| FR-02 input confirmation | Concrete candidate/digest/authority/publication command and S03/S06/B09 integration supplied. Remaining full action/trace publication OPEN. |
| FR-03 trust/metrics/holds/preferences | M02 independent trust/epoch, S02 denominator, U01 current rights and B11/I04 independent holds supplied. FA-06 concerns operational onboarding records, not old metric arithmetic. |
| FR-04 Claim/numeric | Concrete partial coverage/pinning/reversal/replace and exact numeric/DSL/full-pool mechanisms supplied by HB-16; do not repeat historical claim they are absent. Actual policy activation/physical enforcement OPEN. |
| FR-05 deadlines | Stage/schedule identity and separate issued-payment extension supplied. FA-13 concerns operational notification rules, not the fixed deadline identity. |
| FR-06 / PP-08 | Still OPEN; newly localized current-schema/action findings above do not erase originals. |

PC-01–07/ED-01–06 have supplemental semantics; final finding/profile/canonical closure remains OPEN under the recorded decision process. Conditional engine/training, deferred HawkEye/TokenMeter, product Marketplace, self-hosted VM/Kubernetes alternatives and all provider families remain preserved. Customer-specific numeric/tax/FX/calendar/retention values, exact supported vendor versions, SLO/RPO/RTO, named domain decision owners and all NOT_RUN evidence are legitimate unresolved gates, not automatically omitted requirements. Product licensing/offline entitlement and third-party rights remain separate from SaaS feature entitlement.

## Required follow-up order

1. Correct FA-01–03 and mode-specific financial keys/approval subjects; independently review before any implementation.
2. Integrate adjustment and dependent-period lifecycle through existing commands/screens/work/cases (FA-04/05/13), retaining current financial/security gates.
3. Complete source-derived operational/identity/plugin/event/migration and financial/artifact family relationships (FA-06–12) through independently scoped work; no wholesale guessed table design.
4. Reconcile full English source/addition trace and portability (FA-14), obtain only real product/profile decisions, independently re-review closure or explicitly approved scoped deferral, then seek the owner's separate planning-complete declaration.

This review does not execute those remediations or approve a reduced feature baseline. The authorized task is review: original planning, screen specifications and proposed schemas are preserved for subsequent targeted corrections. JEV's uncertain standard-depth recommendation is advisory; root chose source-wide domain review from the owner's request and monetary/security risk. No implementation, live provider/ERP, DB/concurrency, load/failover, rendered UI/accessibility or legal/vendor revalidation is claimed.
