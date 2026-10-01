---
type: Planning Review
title: Remaining content gaps in the v1.16 planning baseline
description: Source-backed findings and independent domain reviews identify seven remaining planning gaps and their proposed closure.
status: draft
---

# Content review of the current planning baseline

Ticket: https://linear.app/hybrid-billing/issue/HB-6/record-the-v116-content-review-and-establish-the-planning-closure-plan

Verdict: **FAIL for planning completeness**. Seven source-backed omissions remain. This is a content review, not an implementation review or release approval. All findings remain open; proposed remedies below are not approved product policy.

## Evidence and scope

The original DOCX SHA-256 matches `docs/planning/v1.16/source-inventory.json`. Fresh body XML extraction exactly matches the existing 2,075 extracted lines, including table rows. B identifiers refer to zero-based body XML children, not pages.

Three read-only agents independently compared chapters 1–43, 44–86, and 87–123 with current planning contracts, requirements, work records and relevant screen specifications. Root reconciled their findings and checked the cited source passages. Existing audit findings were considered so resolved controls and explicitly open policy values were not counted as new omissions.

## Findings

| ID | Severity | Missing or weakened content | Evidence in current repository | Required closure |
| --- | --- | --- | --- | --- |
| PC-01 | MEDIUM | Full default calculation sequence and price-change treatment of sequence changes | Source B0110; `docs/planning/v1.16/requirements.md:289`; `detailed-requirements.md:95` | Preserve quantity → base price → discount → fees → minimum/maximum → FX → tax/rounding. Define approved revisions for order changes and prohibit same-stage priority conflicts. |
| PC-02 | MEDIUM | Preapproved warning acceptance policy and identified acknowledger | Source B0502; `requirements.md:417`; `ui-design.md:127` | Define eligible warnings, policy revision, actor/time and evidence. Keep unwaivable blockers distinct from acceptable warnings. |
| PC-03 | MEDIUM | Conditional restrictions on direct access to an adopted billing engine | Source B0418; `architecture.md:49`; `requirements.md:595` and `:629` | For an adopted external engine, prohibit browser tenant credentials and restrict direct engine APIs using network/service identities. Test attempts to bypass platform approval. Shared platform authorization already exists. |
| PC-04 | MEDIUM | Source-dated Odoo/SAP adapter constraints | Source B0836–B0842; `requirements.md:449`–`:457` | Restore ERP profile distinctions, actual API/permission/plan checks, per-call transaction boundaries, and separate SAP request, billing and FI states. Preserve source-era vendor conditions as historical inputs pending current verification. |
| PC-05 | MEDIUM | Detailed request/release commands for distinct dispute holds | Source B1488; `ui-design.md:217`; `docs/ui/screens/B11/spec.md:65`–`:68` and `:136` | Define unissued-billing hold, delivery hold and external ERP collections-hold requests separately, with scope, authority, reason and external pending/confirmed/unknown results. Preserve document and payment obligations. |
| PC-06 | LOW | Distinct onboarding effectiveness metrics and waiting-time categories | Source B1110; `docs/ui/screens/S02/spec.md:56`; `requirements.md:77` | Define connection rate versus first verified billing completion rate, cohort/denominator and timestamps for user work, external approval and source generation waits. |
| PC-07 | LOW | Customer favorites | Source B1359; `docs/ui/screens/U01/spec.md:55`; `requirements.md:541` | Define personal canonical customer references, add/remove/filter actions and permission-revocation behavior. Favorites confer no access rights. |

PC-01 is financially material: with base 100, discount 10%, fee 20 and minimum 120, the source order yields 120; applying the minimum before discount/fees yields 128. This is an independently calculated planning example, not a product test result.

For PC-04, the source describes Odoo 19 JSON-2, database `/doc` and permissions, online Custom-plan conditions and independently scoped API calls. It also distinguishes SAP EBDR → billing document → FI, prohibits treating EBDR or Journal Entry success as final invoice issuance, and separates Business One Service Layer and ECC interfaces. These are source preservation requirements; their current vendor validity was not verified in this review.

## Preserved content and known decisions

Payment Terms retain issue-date snapshots, overdue uncertainty when authoritative settlement data is stale, partial settlement and reversals. Monthly ERP recognition retains separate recognition/invoice purposes, adjustment deduplication, reconciliation and double-recognition blocking. Their PT-01–08 and ERPC-01–08 references exist.

Previously documented detailed controls remain represented. Initial supported combinations, monetary precision and rounding values, identity providers, retention periods, operational SLOs and final licensing terms remain declared decisions. Their unresolved values are not counted among the seven new omissions.

## Verification and limitations

| Check | Result | Boundary |
| --- | --- | --- |
| Original source hash and fresh extraction comparison | PASS | Text and tables; no rendered-page review |
| Existing planning structure validator, output redirected locally | PASS | 123 chapters, 43 requirements, 43 work packages, dependency and document links |
| Existing detailed trace validator, output redirected locally | PASS | 35 detail groups and 49 screen groups; structural checks do not establish semantic completeness |
| Independent architecture review | FAIL | PC-01–03; no runtime security or performance verification |
| Independent billing/ERP review | FAIL | PC-04; no vendor or accounting certification |
| Independent product/QA review | FAIL | PC-05–07; no interaction, accessibility or rendering tests |
| Product acceptance execution | NOT_RUN | 119 original plus 91 additional test records, including PT/ERPC |

Historical audit summaries mention 75 additional tests and nine wireframes. Current artifacts contain 91 additional test records and 49 screen specifications/wireframes. Historical results must remain distinguishable from the current baseline. Updating summary freshness is documentation maintenance, not an eighth product-content omission.

The latest user rule requires English Git-bound documents. Existing Korean documents require a separately tracked translation/synchronization task; this review does not silently replace the source or approve a bulk translation.

The content-review phase performed no product development, provider connection, ERP mutation or external delivery. Subsequent README/workflow commits and PR #1 integration are tracked separately under HB-5. The planning-development gate remains closed.
