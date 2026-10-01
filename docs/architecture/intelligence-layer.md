---
type: Intelligence Design
title: Evidence-based validation correction adjustment and forecasting
description: Defines a scoped intelligence layer that proposes explainable actions while preserving deterministic billing and human approval authority.
status: draft
---

# Intelligence layer

Part of [HB-7](architecture-readiness.md). Owner-requested scope is validation, correction, adjustment and prediction in analytical/module workflows. This is a new design direction, not evidence that existing source requirements already specify every model or module. Initial use cases, model products and acceptance thresholds require owner decisions.

## Distinct responsibilities

| Capability | Output | Authority and acceptance |
| --- | --- | --- |
| Deterministic validation | Schema/unit/identity/coverage/currency checks and full reconciliation findings | Rules and independent monetary oracles govern eligibility; a model cannot waive an error or establish missing coverage |
| Anomaly detection | Flag unusual usage/cost/rate/FX/allocation with baseline and contributing evidence | Anomaly is a review finding, not fraud proof or an automatic change to the charge |
| Correction assistance | Proposed mapping/source interpretation or classification revision with before/after evidence | Verify against permitted original evidence; create new mapping/input revision and rerun validation; never overwrite raw evidence |
| Adjustment assistance | Explicit quantity/amount adjustment draft with reason, original target, delta and recomputation impact | Ordinary contract/policy approval, tax/allocation revalidation and new snapshot; issued items use linked correction documents |
| Forecasting | Future usage/cost/charge estimate, horizon/scenario and uncertainty | Clearly label prediction, as-of, assumptions, amount basis/currency; never substitute for actual invoice/receipt or monthly recognition |
| Module assistance | Explain findings and recommend bounded next actions through permitted commands | Same user/client policy; no direct DB/ERP writes, privileged tokens or hidden approval authority |

Validation gates remain deterministic even when statistical methods help prioritize review. Missing source evidence is unknown; an imputed estimate remains a separately marked scenario and cannot become final billing evidence without a distinct approved contractual basis. Rules may be versioned/approved and automate explicitly authorized low-risk classification; monetary correction and gate exceptions retain the required approval controls. No automatic monetary action is adopted here.

## Data and execution contract

`AnalysisRun` references eligible input manifest/revision, workspace/customer scope, permitted feature schema, method/model/rule digest, policy revision and cutoff. `Finding` carries evidence locators, observed/expected values and review status. `CorrectionProposal` and `AdjustmentProposal` carry target revision, delta, reason and impact. `Forecast` carries horizon, scenario, unit/currency/basis, assumptions, interval or documented uncertainty method and data sufficiency. `ProposalDecision` records actual reviewer, approved/rejected state and resulting ordinary command/run; analysis completion is not approval.

Model retrieval and feature materialization use current tenant/customer/field permissions and separate operational telemetry from billing data. Recheck authority before persisting/disclosing outputs and before applying a proposal. Cross-tenant retrieval, cache reuse or pooled training is prohibited without an explicit compatible disclosure/consent policy. Rejected/revoked/deleted inputs cannot reappear through model indexes, embeddings or replay. Include feature stores, training/evaluation snapshots and inference caches in retention/deletion inventory.

Customer documents and model outputs are untrusted data. A tool allowlist, schema validation, policy checks, revision checks and execution limits apply to every proposed action; embedded text cannot authorize it. Do not capture original AI usage prompts/responses by default, invent payment details or use financial/customer bodies in operational traces. Preserve protected billing evidence and the separate Assistant conversation-retention contract.

## Self-hosted module and model choices

Compare deterministic/statistical local workers first, then local inference where value justifies resources. An external model endpoint is a separate customer policy/network/data-processing decision, never a default consequence of this design. Disconnected operation must have explicit degraded/manual behavior if required; model outage cannot corrupt billing or block a valid deterministic close solely because optional suggestions are unavailable.

Each module declares owner, version/digest, supported input/output schema, allowed data/actions, CPU/memory/time budget, endpoint allowlist, compatible core versions and licensing/provenance. Isolate plugin/model execution from ledger credentials. Keep historic evidence readable without running an unsafe old module; reproduction may require controlled isolated verification. Exact host/runtime and model resource requirements are unresolved.

## Evaluation and release gates

The [conditional training/withdrawal contract](../contracts/security/access-and-trust.md) adds learned weights/checkpoints/adapters, deployed audiences and backup withdrawal. Sensitive tenant-data training remains blocked until its separately reviewed approved contract; this is no implied product exclusion or training approval.

Define first use cases and independent labeled cases before selecting models. Planned evidence: validation false-positive/false-negative cases; correction delta correctness; unauthorized tool/proposal rejection; historical out-of-time forecast backtests with no future-data leakage; per-scope/currency errors against a simple baseline, uncertainty calibration, sparse/no-data behavior, drift and monitoring. Record data/version/cutoff and human review effort, not only model scores. Choose thresholds and forecast horizon with finance/product owners; no invented accuracy guarantee.

Adoption requires independent security/billing review, traceable datasets and measured latency/resource costs under ingestion/closing load. Proposals must remain reversible until approved business execution; issued corrections remain explicit audit history. Current model training, inference, backtests, adversarial tests and integration scenarios are **NOT_RUN**.
