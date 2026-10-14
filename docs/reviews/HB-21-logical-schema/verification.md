---
type: Verification Record
title: HB-21 logical schema evidence and review
description: Records the drafting baseline independent reviews document QA and remaining planning gates for the logical data schema.
status: draft
---

# Logical schema verification

Tracking: [HB-21](https://linear.app/hybrid-billing/issue/HB-21). Accountable owner Seung Woo Park; machine `mac_mini` during execution. Root owns PM/planning, schema writing, integration and document QA. Separate read-only architect and DBA reviewers inspect actual drafts. Branch `docs/HB-21-logical-data-schema` starts from fetched origin/main `b5c65cf66992f703247b8e4713f86cef7835b767` in isolated worktree `/private/tmp/hb21-schema`; unrelated original-workspace files are preserved.

## Authorized work and provenance

Owner requested a data schema draft, not product development or merge. Scope: two English logical schema documents, existing ownership-model entry link, README entry and this report. No DDL, product source, migrations, customer data or generated assets. Existing 43 requirement groups and 49 screen identities remain unchanged; no complete original-source audit or full action/API matrix is claimed.

Inputs: merged HB-16 contracts and ownership/lifecycle model; unmerged HB-19 `75c4c188e5d42d682d286cebe9900b2bc0bc660c`; separate HB-18 `6acb38c68117ff1f1865776aca00f87c61956196`; HB-20 local `62f7df1` logical storage/capacity review. They are identified in [schema provenance](../../architecture/data-ownership/logical-data-schema.md#basis-and-scope). Candidate architecture and SaaS content are not silently merged or adopted. The >100M/day premise supersedes older examples within this scoped draft; untouched historical/local baseline documents are not declared synchronized.

Tracking: Linear connector returned the different `myblueton` workspace, so it was not used for project writes. Authorized browser verified Hybrid Billing workspace, team ID matching `.linear.json`, project and Seung Woo Park; created actual HB-21, In Progress, Improvement and `machine:mac_mini`. Browser issue read-back confirmed creation. No successful correct-workspace API connection asserted.

## Checks and review

| Check | Result and limit |
| --- | --- |
| Independent architecture design and actual change review | Initial actual-draft FAIL: conditional SaaS entity catalog missing despite HB-18/F09 basis. Added catalog/subscription/change/meter/seat/payment/refund and desired/applied entitlement entities; targeted actual-file re-review PASS, no remaining blocking architectural finding. Tenant/publication/GC/control/AI boundaries reviewed at document depth. |
| Independent DBA actual change review | Initial gaps: unit revision alone could alias different units; recognition links lacked confirmed-net revision and adjustment inclusion disposition. Added explicit unit identity, complete recognition inclusion/scope/evidence fields, receipt reservations and credit/receipt reversal caps. Final actual-file review, including conditional SaaS/payment extension: PASS, no remaining blocking DBA finding. |
| Root document QA | PASS: five scoped files checked for applicable nonempty type frontmatter, balanced fences, local Markdown targets/new anchors and common credential patterns; current-main reconciliation and remediation JSON bytes unchanged. Scoped whitespace passed. An initial frontmatter-check regex falsely rejected valid metadata; corrected checker passed, with no artifact changes needed. Mermaid rendering NOT_RUN; source relationship view only. |
| Runtime / DB / provider / ERP / UI / performance / recovery | NOT_RUN; documentation only. |
| PR / merge / read-back / completion | Branch publication succeeded. PR creation FAILED with GitHub `must be a collaborator` through both API and authenticated browser. No PR/merge/completion claimed; owner collaboration/access resolution is required before PR creation. No merge authorization or planning completion. |

Physical schemas, catalog/version adoption, numeric/customer policies, final state vocabularies, full canonical trace, retention/RPO/RTO and named domain decision owners remain OPEN. Draft fields are proposed constraints, not implemented controls. JEV is advisory and supplied no authorization; root selected architecture/DBA/writing responsibilities from the actual financial/tenant scope.

Root QA applies the QA skill to LS-01–10 planned scenarios and preserved CL/NM/IC/TC/PT/ERPC/QG/UD references. This is root document/scenario verification, not an independent QA-agent run or executed application test. Independent architect/DBA agents were separate from the writer and made no file edits. Staged scope, generated-asset exclusion and added-content secret checks are inspected before commit; publication and handoff are read back separately.

## Publication handoff

Reviewed schema commit: `59c904def103508b0231e2e25cf3bb1b8a898cc9`, five files / 302 added lines. Remote branch: `docs/HB-21-logical-data-schema`. API PR creation initially hit sandbox network denial; approved rerun returned HTTP 422 `must be a collaborator`. Separately authenticated GitHub browser showed the same explicit creation failure; it did not create a PR. The prepared English PR description remains locally at `/private/tmp/hb21-pr.json`, an ephemeral helper rather than a committed artifact. Read-back verifies the remote branch and exact schema blobs, not a merge. HB-21 remains In Progress pending PR/integration; machine claim is released at handoff. Product/runtime tests remain NOT_RUN.
