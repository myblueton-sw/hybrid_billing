---
type: ImplementationPlan
title: Current approval prerequisites and W-23 integration
description: Resumes the existing W-23 integration scope through the pre-launch plan, beginning with non-consuming current approval verification.
status: draft
---

# Original objective and ownership

[HYB-225](https://linear.app/myblueton/issue/HYB-225) retains the full existing W-23 follow-up: stage prerequisites; adjustments, protected changes and recognition integration; authenticated grants; persistence; O03/U02/I01/I02/B08/B09 bindings; independent DBA and QA. The [pre-launch plan](../HB-257/plan.md) authorizes local synthetic in-memory assembly, not adoption of unresolved production policy/database decisions or external effects. Completing the first dependency below does not complete the ticket or the overall owner goal.

Root owns PM, planning, implementation and integration. Accountable owner: Seung Woo Park. Independent financial/security reviewer is read-only. Machine: mac_mini; branch feat/HYB-225-approval-integration from origin/main 0f9d9a2 in /private/tmp/hyb225. HYB-337 remains separately active on the same machine while its full regression runs; its staged files and runtime are not changed by this work. No merge, deployment, real accounts, ERP calls, payment or messages are authorized.

Required companion skills/rules absent from the authorized workspace remain absent under the owner's no-restoration instruction. Existing module source, tracked contracts and independent review remain the evidence; no policy is reconstructed from missing inputs.

# Evidence and dependency

The [HB-221 contract](../HB-221/plan.md#execution) requires current policy, digest, exact amount, currency, authority and item eligibility checks. Its register implements those checks in the private check method. RequestOf returns stored state, not current validity. Execute performs checks but records/consumes an execution; after a reported success its idempotent replay intentionally returns historical success without rechecking current approval. Neither call can safely serve as a non-consuming prerequisite proof.

The existing HYB-225 ticket requires charge approval before ERP final approval, issuance and delivery. [Financial gates](../../contracts/billing/inputs-pricing-and-erp.md) separate input eligibility, fixed amount approval, separately authorized issuance/submission and external confirmation. Adjustment and recognition records have distinct targets and remaining caller obligations in HB-222/HB-224 verification. Unknown prerequisites cannot become not-applicable or be satisfied by an approval badge.

# First implementation boundary for independent review

Add Registry.CheckCurrent(workspace string, expected Execution, actor Actor) error in modules/approvals. The existing Execution value supplies exactly the same expected stage/target/item/digests/amount/currency shape as Execute. Its ID remains a validated candidate identifier for shape compatibility; CheckCurrent neither looks up/reserves that ID nor consumes its target key. It is not an execution, decision, reservation, bearer authorization or certificate of downstream readiness.

The method validates workspace and actor with existing rules, normalizes/validates the expected execution through executionForm, enters the module mutex/monotonic clock, finds the request within that workspace and calls the same private check used by Execute. It returns existing errors/refusal reasons. It performs no request, decision, authority, execution, outcome or consumed-key write; normal clock watermark advancement remains possible, as in existing reads. No duplicate monetary arithmetic or authority policy is introduced.

Current verification deliberately has no historical-success replay bypass: a revoked/expired/superseded approval fails CheckCurrent even if Execute can return a previously reported success for the identical operation. Conversely, an already consumed but still current approval may pass CheckCurrent; Execute retains its own consumed-key/idempotency rule. A caller cannot use this method to authorize a second execution. This distinction must be explicit in verification and consumer design.

The method does not authenticate actors or fetch live business evidence. The future integration caller must derive those from current authenticated service/module facts, hold the existing installation domain lock across prerequisite reads and the dependent mutation, and still invoke Execute at the final effect boundary. This first increment adds no HTTP endpoint and changes no existing execution behavior.

# Scope, acceptance and tests

Root may edit modules/approvals/check.go and focused test files, plus docs/features/HYB-225 and tools/approval-current-mutants.py for disposable verification. Existing check helpers may be factored only if review shows it necessary; unchanged Execute behavior is a regression requirement. No changes to HYB-337 files or to unrelated modules.

Before implementation, independent review must confirm public-contract semantics and the non-consuming versus execution boundary. Test current approval success without changing requests/executions/keys; pending/rejected/withdrawn/superseded; changed policy/digests/amount/currency; authority revocation/expiry and exact validity boundary; missing/foreign workspace/request; invalid actor/form; item exclusion; repeated/concurrent checks; and a completed historical execution whose approval later becomes invalid. Ensure the real Execute still consumes once and current checks do not exhaust execution limits. Use existing synthetic module governance and independent expected outcomes. Run formatting, vet, module/race checks and decision mutations/coverage; record exact results and re-review.

# Following integration work remains required

After this prerequisite method is reviewed, design the actual service-owned stage chain against immutable producer records: charge result → applicable final ERP reconciliation/amount → issuance confirmation → delivery. Derive applicability, source lines, revisions, deciding approvers, protected-change provenance and recognition attribution from their owning modules. Current auth grants and source-eligibility/cross-validation remain mandatory. Do not supply these as client booleans or fabricated hashes. The stage chain must distinguish an unissued changed result from immutable issued history and unknown external outcomes; no blind retry or new effect follows historical success.

Required independent producer/DBA review precedes that integration implementation. Persistence and unselected production policies remain unresolved; the in-memory pre-launch boundary does not adopt them. This dependency step leaves the complete ticket scope open rather than redefining W-23 as a helper-only task.
