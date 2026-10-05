---
type: Planning Review
title: Full-scope planning reconciliation and confirmation gate
description: Traces the owner scope clarification to existing requirements and records proposed accuracy gates, independent review and remaining decisions.
status: draft
---

# HB-13 planning reconciliation

Tracking: [HB-13](https://linear.app/hybrid-billing/issue/HB-13/reconcile-full-provider-planning-scope-and-accuracy-first-confirmation). Baseline: `cade708`, 2026-10-06. Branch: `docs/HB-13-full-scope-planning`. Accountable owner: Seung Woo Park. Root owns PM/planning, writing and scoped document QA. A separate read-only reviewer performs design and actual change review. Worked machine: `mac_mini`.

This is a bounded clarification of the full existing planning scope, not a new reduced baseline or completion of all planning. Only documentation changed. Product development remains gated on the owner's explicit planning-complete declaration.

## Evidence and provenance

The owner requests the entire planning scope, accuracy first, mandatory cross-validation, operator confirmation and explicit attribution of JEV recommendations. The phrase describing an optional capability is truncated and remains unresolved; no exception or feature is inferred.

Local authorized v1.16 `requirements.md` contains 43 top-level requirement groups. Requirement IDs such as HB-05 and HB-13 in that file are source-local requirement references, not Linear issue IDs. This Linear HB-13 is not the VMware requirement of the same name. The local source and `provider-profiles.md` remain unpublished inputs; this ticket preserves them and does not certify a standalone clone or publish their full English baseline. PP-08 remains responsible for canonical source/work/screen/acceptance reconciliation.

| Existing source-local reference | Evidence preserved / current English contract |
| --- | --- |
| HB-05; provider profiles | AWS, Azure, GCP, OCI, Alibaba, VMware, HawkEye and TokenMeter all remain collection design targets; [full scope](../../architecture/data-collection.md#full-planning-scope) |
| HB-06 | Immutable source revisions, retry versus conflict, no amount-only deduplication; [publication semantics](../../contracts/billing/inputs-pricing-and-erp.md#ed-05-eligible-dataset-publication) |
| HB-07 | Mapping revisions, full-scope duplicate/period/total checks and distinct monetary bases; [collection lifecycle](../../architecture/data-collection.md#common-lifecycle) |
| HB-08 | Retained evidence, lineage and reproducibility; [usage lifecycle](../../architecture/usage-detail-lifecycle.md) |
| HB-09 | Matching scope/currency/basis, full reconciliation and zero unexplained difference; no fabricated supplier invoice for internal or independent-price bases |
| HB-10/11/12/13 | Existing HawkEye/TokenMeter, on-prem cost-pool and VMware measurement/history responsibilities remain scoped; VMware is not a platform installation prerequisite |
| HB-23/29/38 | Approval versus execution, explicit corrections and independent expected results remain separate; input confirmation cannot replace later financial gates |
| Other baseline groups and RA-01–14 | Preserved unchanged in scope; no claim that this ticket audited or completed their entire detailed trace |

Source-profile evidence must retain the documented branches: AWS payer/member/transfer and source authority; Azure EA/MCA/CSP billing roles; GCP billing/project/reseller and export roles; OCI tenancy/subscription/region; Alibaba site/direct/reseller relationships; VMware historical measurement and approved cost policy; HawkEye/TokenMeter immutable eligible evidence and complete exports. These are source-dated design inputs, not newly verified vendor facts or certified capabilities. Exact versions/methods/permissions and evidence remain to be collected for each applicable profile.

Earlier first/initial profile wording is superseded by the [owner clarification](../../planning/recent-owner-requirements.md#full-scope-clarification-after-hb-11): evidence and activation can be sequenced without removing design targets. Existing historical review reports retain their baseline-specific verdicts. No supplier capability, numeric tolerance, technology choice or named domain specialist is invented.

JEV recommendation provenance: the preceding turn suggested architect/standard; this execution turn selected no skill and returned uncertain review depth. Root selected PM/planner/technical-writing and scoped QA based on the actual task, with separate contract review because billing eligibility and scoped authority are affected. These recommendations are advisory; neither JEV nor an AI review supplies owner approval.

## Proposed acceptance scenarios

All cases below are design obligations and **NOT_RUN** runtime tests. H13 references are local acceptance IDs, not new baseline features or Linear tickets. Apply relevant cases across the full provider/profile matrix; a single passing adapter cannot certify the rest.

| Case | Required outcome |
| --- | --- |
| H13-01 full scope | All eight collection families remain represented; unavailable capability stays unverified with a reason, without disappearing from design scope |
| H13-02 independent comparison | Reused transformed totals, missing pages/files, unresolved conflicts or sampling-only evidence cannot satisfy the full-scope comparison gate; original evidence is retained |
| H13-03 basis correctness | Required complete metering plus approved independent-price policy can proceed with supplier cost explicitly unknown; incompatible currency/basis comparisons and unexplained differences cannot pass; no fake invoice |
| H13-04 actor scope | Wrong-workspace, wrong-source-scope or revoked confirmation authority blocks publication; customer explanation never reveals protected shared-payer evidence |
| H13-05 stale decision | Affected input, mapping, attribution, profile or comparison revision change after confirmation requires renewed checks and confirmation; concurrent stale publication loses the expected-revision check |
| H13-06 retries and corrections | Identical API/upload evidence is one economic dataset; retried publication creates no extra Claim; late correction creates a new linked revision without overwriting history |
| H13-07 separate approvals | Confirmed input alone cannot authorize pricing policy, final amount, issuance or ERP effects; an operator cannot waive an unresolved blocker |
| H13-08 optional unknown | No automatic confirmation or optional-model exception is enabled from the interrupted phrase; optional recommendations cannot replace mandatory checks |

## Verification and limits

| Check | Outcome |
| --- | --- |
| Ticket, owner, workspace and machine | PASS: authenticated Hybrid Billing UI showed HB-13 In Progress, Seung Woo Park, project selection and machine:mac_mini; no API connection success asserted |
| Baseline and branch | PASS: fetched origin/main and created the ticket branch at cade708; pre-existing untracked local inputs preserved |
| Independent proposed design review | PASS with conditions incorporated: preserve all eight families and basis-specific evidence, link ED-05 to the confirmation gate, bind revisions/current scoped authority, distinguish computational independence from staffing |
| Independent actual change review | Initial FAIL: ambiguous existing “VMware is optional” wording could contradict full collection scope. Clarified installation/workload-path optionality and preserved financial separation of duties. Independent targeted re-review PASS; no remaining blocking findings |
| Scoped document validation and root QA | PASS: Python checked six English documents for required frontmatter/status, local file links, balanced fences and common credential patterns; confirmed eight H13 cases, eight collection families and 43 existing source requirement headings. Root inspected scope and basis-specific scenarios. No runtime test executed |
| Staged diff and whitespace | PASS: root inspected the actual six-file staged diff for unrelated content, generated assets and sensitive material; git diff --cached --check passed. Independent reviewer also ran git diff --check successfully |
| Runtime / UI / provider / ERP / performance | NOT_RUN; document checks cannot certify implementation or external compatibility |
| PR / merge / ticket completion | Pending; no merge or planning acceptance asserted |

Remaining decisions: clarify the interrupted optional capability; confirm the proposed exact input-confirmation interaction and scoped role matrix; obtain missing profile, numeric, retention, identity and operational evidence without reducing baseline scope. Existing D-01–07 and PP-07/08/09 remain open. No human finance/security approval is claimed. PR preparation and independent review are separate from an authorized merge and final ticket completion.
