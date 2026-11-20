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

# Next increment: immutable rating candidate provenance

Independent producer review found no assembled server-owned charge-run store. `rating.Result` supplies actual calculated charges and exact totals, but has no persisted run identity/digest. `reconcile.Readiness` alone does not establish complete source eligibility, and its run spec contains caller-supplied facts. ERP external document rows are not yet bound to a charge snapshot. The next increment therefore builds the internal candidate store required before a safe approval bridge; it does not expose an approval command or turn synthetic inputs into eligible financial evidence.

Owned files extend to `apps/api/server/approvalproducer.go`, `approvalproducer_test.go` and this ticket's records. The store is internal and initially exercised by synthetic module-driven tests; live Server assembly is deferred until the producer eligibility adapter has a reviewed contract. No existing server seed or HTTP boundary changes in this increment.

An immutable candidate input contains workspace, run ID, customer, entity, period dates, source revision/digest references, complete decimal ProfileSpec, rating limits, Policy and input Lines. Source references are provenance labels only, never eligibility proof. Identifiers use the existing bounded ASCII identifier grammar; periods are exact calendar dates with start no later than end; source digests are 64 lower-case hex digits. Bound candidates to 1,000 records, 1,000 lines and 1 MiB encoded input/output per record in this preliminary internal store. These are development bounds, not production capacity policy.

The store serializes a private copy of the input, reconstructs the decimal profile, calls rating.Compile and Rate itself, and stores the actual result. It never accepts a caller-supplied result or caller-supplied digest for the computed result. Versioned fixed-order JSON structs (no maps), preserving array order and exact decimal text, are SHA-256 hashed with the domain label `hb_charge_candidate_v1`. The digest covers all input/provenance/profile/policy/limit fields and the complete result. Different array orders or decimal spellings may conservatively produce different digests; no semantic-equivalence claim is made. Unknown fields cannot enter the typed internal input. This encoding is not a public or durable wire contract.

Workspace/run ID identifies an immutable record. Identical canonical input replays its recorded result; changed input under that key conflicts. Another workspace is isolated. Record capacity does not reject an existing exact replay. Keep canonical bytes private and decode fresh values on every return so mutation of caller inputs/returned nested slices cannot alter stored evidence. No cached eligibility/approved flag is stored, and no current approval can be derived solely from this candidate. Rating exclusions and holds remain present rather than being erased or treated as eligible.

The standalone store owns a mutex for atomic lookup/insert. Future installation assembly must use server domain lock before store/module locks; the store never calls back into Server. Tests must verify literal independent charge totals and profile provenance, deterministic repeat, change binding across identity/provenance/policy/inputs, collision conflict, workspace isolation, input/output alias isolation, capacity/replay, concurrent single-key conflict, malformed/oversized inputs and preserved held/excluded result. Follow-on approval creation remains refused/unimplemented until authenticated current source eligibility, reconciliation linkage and policy authority are bound from their owners. This candidate store does not complete that dependency.

Additional resource bounds before arithmetic: each numeric ProfileSpec dimension must be positive and at most 80, with the existing decimal.NewProfile relation checks retained. Rating Rules/Tiers/Discounts/Lines limits are positive and at most 1,000; Intermediate is positive and at most 80. Require 1–1,000 unique source IDs and valid source revision/digest references. These hard development bounds supplement encoded size limits. The captured rating total is the rating result's own amount basis, not proof of tax-inclusive invoice total or final ERP amount.

Independent pre-code review returned scoped PASS after adding invalid UTF-8 rejection before JSON encoding and an approvalproducer.go-only decimal import exception in source_test.go (now also owned). The input and complete envelope each have a separate 1 MiB cap. JSON errors propagate. Concurrent caller mutation of supplied slices during capture is unsupported; mutation after return and mutation of returned copies cannot affect records.

Implementation review requested complete-envelope digest, held-rule and output-overflow assertions. It also identified serialization allocation preceding the encoded-size check. Preflight now limits aggregate raw string bytes to 1 MiB and eligible classes to four before JSON allocation; JSON escaping/structural overhead can still exceed 1 MiB transiently, and the subsequent encoded limit bounds retained records. This is bounded internal storage, not a streaming memory guarantee.
