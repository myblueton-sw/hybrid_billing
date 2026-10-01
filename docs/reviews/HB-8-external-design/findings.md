---
type: Design Review
title: HB-8 external-source design review
description: Evaluates the current design against primary publications and defines closure conditions for six open contract refinements.
status: draft
---

# External-source design review

Ticket: [HB-8](https://linear.app/hybrid-billing/issue/HB-8/review-the-current-design-against-external-standards-and-expert). Reviewed Git baseline: `585d9e1776d7cfb291f9e76c80a981912a91f5d9`. Sources checked on 2026-10-01. Accountable owner: Seung Woo Park. Root owns writing/integration; separate read-only architecture, DBA and QA/security agents supplied independent analyses. Worked machine: `mac_mini`.

Scope: the seven [HB-7 architecture documents](../../architecture/architecture-readiness.md), relevant local v1.16 contracts/screens and [HB-6 findings](../HB-6-content/findings.md). Local companion inputs are not all published in Git; their paths/lines identify local evidence, not fresh-clone availability. This is content/threat analysis, not an implementation review.

**Verdict: conceptual direction PASS; affected capability contracts incomplete (FAIL); runtime UNVERIFIED.** Six OPEN refinements comprise two HIGH and four MEDIUM planning priorities. HIGH indicates potentially severe cross-customer disclosure or identity substitution if a missing boundary is implemented incorrectly. No running exploit was reproduced. Close or explicitly defer these contracts before adopting the affected capability.

Public expert publications were consulted. External human expert consultation was **NOT_PERFORMED**. Independent AI reviews are not external human sign-off, security certification or financial/legal approval. No customer information or credentials were used.

## Conceptual assessment

| Concept | Assessment | Evidence and limitation |
| --- | --- | --- |
| FinOps foundation and allocation | PASS direction: explicit source coverage and versioned attribution; provider cost, selling price and allocation remain distinct | [FinOps Data Ingestion](https://www.finops.org/framework/capabilities/data-ingestion/) considers granularity, completeness and access; [Allocation](https://framework.finops.org/framework/capabilities/allocation/) recognizes different shared-cost methods. Neither selects this product's pricing/accounting policy |
| Modular core before service proliferation | PASS direction: coarse business authority, workers and conditional analytics/transport expansion | Martin Fowler's [Monolith First](https://martinfowler.com/bliki/MonolithFirst.html) discusses operating overhead and uncertain early service boundaries. Expert experience is not a capacity benchmark or a prohibition on later extraction |
| Kubernetes cost layering | PASS direction: infrastructure cost and workload allocation form lineage, not additive duplicate charges | [OpenCost specification](https://opencost.io/docs/specification/) separates asset/workload costs. It is an allocation concept, not selling-price authority or mandatory VMware/OpenCost adoption |
| Authority versus analytics/transport/telemetry | PASS direction: coordinated transactional authority, immutable eligible evidence and derived projections | [Technology assessment](../../architecture/technology-assessment.md) bounds ClickHouse/Kafka/OTel guarantees. ED-01/04 refine analytics; they do not replace financial authority |
| Tenant/customer/org and financial identity | PASS direction: Workspace, Party, servicing relationship, org attribution, receipt and Claim are distinct | [Billing model](../../architecture/data-ownership/billing-data-model.md) preserves temporal snapshots, explicit sharing, credit coordination and separate recognition/issuance/settlement. No double-entry accounting requirement is inferred |
| Intelligence as assistance | PASS direction: deterministic eligibility and ordinary approval retain authority; forecasts are distinct from actual charges | [Intelligence contract](../../architecture/intelligence-layer.md) covers evidence/scope and temporal evaluation; ED-06 refines conditionally permitted training |

## Open findings

Line references identify the reviewed baseline before entry-point edits. All six findings remain **OPEN**; publishing this report does not resolve them.

### ED-01 — HIGH: analytical query enforcement boundary

**Evidence:** `docs/architecture/security-boundaries/identity-session-permissions.md:39`, `docs/architecture/admin-design.md:52`, `docs/architecture/technology-assessment.md:23`. Scoped analytics/current authority are required, but query identity/credentials, enforcement point, allowed columns and stale-permission behavior in the analytical engine are unspecified.

**Failure scenario (inference):** a reporting/export/model worker uses a broad credential and omits tenant filtering or selects restricted margin fields. PostgreSQL RLS cannot protect the analytical copy. ClickHouse describes row policies as read filters and warns that table-modification privileges can defeat them. [ClickHouse row policies](https://clickhouse.com/docs/reference/statements/create/row-policy).

**Closure owner and contract:** architect/security/DBA define a mandatory validated query gateway or equivalently verified database-policy mechanism; read/write credential separation, field/customer restrictions, current-policy binding, pooled-context cleanup and prohibited direct access. No mechanism is adopted here.

**Acceptance/dependency:** negative cases cover missing/tampered scope, joins, counts/aggregates, pooled connections, exports/model retrieval and revoked grants. Stale/unsupported authority cannot disclose restricted rows/fields. Depends on first tenant/read-model profile and revocation contract.

### ED-02 — HIGH: authentication-root change authority

**Evidence:** `docs/architecture/admin-design.md:18,40`, `docs/architecture/security-boundaries/identity-session-permissions.md:14,16,18`; local `docs/ui/screens/M02/spec.md:55–63,85`. M02 already requires policy-approval authority to activate a revision. The gap is who may replace trusted issuer/key/subject/MFA policy, beyond bounded membership administration and ordinary verified key rotation.

**Failure scenario (inference):** a bounded administrator replaces trusted signing-key configuration for an existing issuer and presents an assertion for an existing approver's subject. Membership grants remain unchanged; a grant ceiling alone does not prevent impersonation. A new issuer must not bypass the existing prohibition on email auto-linking. OIDC relies on trusted issuer keys and stable issuer/subject pairs. [OIDC validation](https://openid.net/specs/openid-connect-core-1_0.html#IDTokenValidation), [identity](https://openid.net/specs/openid-connect-core-1_0.html#ClaimStability), [rotation](https://openid.net/specs/openid-connect-core-1_0.html#RotateSigKeys).

**Closure owner and contract:** security/finance distinguish authentication-root replacement, require separately authorized activation and identity-migration proof, and specify affected-session/token invalidation and recovery. Verified provider key rotation differs from changing who vouches for privileged identities. Self-hosted OS/DB root remains a documented infrastructure trust boundary.

**Acceptance/dependency:** bounded access/customer administrators cannot substitute issuer/key/subject linkage or MFA evidence to impersonate approvers. Legitimate rotation/migration preserves verified identity and applies approved session invalidation. Depends on first authentication profile and authority/dual-control matrix.

### ED-03 — MEDIUM: collection fetch-policy coverage

**Evidence:** `docs/architecture/data-collection.md:35–41,47–49`; local `docs/planning/v1.16/detailed-requirements.md:33–41,411–419` (AR-02/QA-05). Existing contracts protect connector/webhook destinations; the expanded contract does not explicitly propagate them to export manifests, pagination URLs, artifact references, OCR resources and diagnostic retrieval with destination-bound credentials.

**Failure scenario (inference):** a valid upstream response references a metadata/internal endpoint, or a redirect forwards an upstream credential to another origin. Initial connection validation does not validate subsequent requests. OWASP recommends application/network restrictions and avoiding redirect bypasses. [OWASP SSRF guidance](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html).

**Closure owner and contract:** connector/security define one policy across retrieval methods: schemes, allowed destinations/origins, credential binding, actual resolved addresses, redirects, bounded retrieval and explicit approved private endpoints. Self-hosted on-prem retrieval requires deliberate policy rather than an indiscriminate private-address ban.

**Acceptance/dependency:** malicious manifest/pagination/artifact references, DNS changes and redirects cause no forbidden request or credential forwarding; approved private endpoints remain scoped/testable. Depends on initial source/network profiles. Reconcile AR-02/QA-05, not a conflicting policy.

### ED-04 — MEDIUM: analytical monetary arithmetic

**Evidence:** `docs/architecture/technology-assessment.md:23,42,46`. ClickHouse may hold monetary/detail projections; adoption checks name decimal round trips without arithmetic/aggregate-result compatibility.

**Counterexample (documented behavior, not locally executed):** ClickHouse documents `Decimal(2.0000) / 3` yielding `0.6666` by truncation. Under an illustrative, separately approved four-decimal nearest-rounding policy the result would be `0.6667`; no such policy is selected here. Storage round trips cannot detect the difference. Documentation also identifies unchecked Decimal128/256 overflow and Float64 results for some statistical functions. [ClickHouse Decimal](https://clickhouse.com/docs/reference/data-types/decimal).

**Closure owner and contract:** DBA/finance/analytics define allowed expressions, intermediate bounds/scales, rounding points, aggregate result types and unsupported-operation handling; alternatively constrain financial projections to precomputed authoritative results with approved exact aggregation. Precision selection is an existing decision; missing arithmetic verification is a separate category.

**Acceptance/dependency:** independent oracles cover division, multiplication, cancellation, near-bound sums, scale conversion and Decimal-to-Float expressions. Mismatches/unsupported expressions cannot feed financial approval/reconciliation. Depends on amount policy and monetary processing in the selected analytical profile.

### ED-05 — MEDIUM: delivery mode and replacement scope

**Evidence:** `docs/architecture/data-collection.md:29,36,52,63`; local `docs/planning/v1.16/architecture.md:82` already defers adapter duplicate/replacement rules. This refines that acknowledged decision with complete-snapshot, omitted-row and append-correction invariants.

**Counterexample (reasoned):** snapshots `[A=10]` then `[A=10,B=5]` should yield 15; appending both yields 25. An append correction −2 should then yield 13; replacing the dataset with that delta loses history. Per-artifact counts and request deduplication do not specify the operation. [FOCUS 1.4 Delivery Handling](https://focus.finops.org/docs/specification/v1-4/attributes/delivery-handling/) distinguishes overwrite snapshots, append delivery and hybrid mechanisms.

**Closure owner and contract:** source architect/finance specify per-profile mode/scope, revision order, omitted-record removal, reversal meaning and hybrid transitions; preserve approved immutable inputs while publishing a new eligible revision. FOCUS is conceptual evidence, not mandatory format/version adoption or provider support certification.

**Acceptance/dependency:** repeated snapshots, disappearing rows, append corrections, cross-transport overlap and out-of-order delivery yield expected eligible datasets without duplicate economic coverage or historic mutation. Depends on first source profiles and authority/precedence decisions.

### ED-06 — MEDIUM, conditional: trained-artifact withdrawal

**Evidence:** `docs/architecture/intelligence-layer.md:29,37`, `docs/architecture/architecture-readiness.md:74`. Conditional pooled training requires compatible policy; retention inventory lists feature stores, training snapshots and inference caches, but not learned weights/checkpoints/adapters and eligibility after withdrawal.

**Failure scenario (inference):** source snapshots/embeddings are removed, but a fine-tuned model containing memorized customer content remains served. Retrieval filtering cannot remove learned information. Carlini and co-authors demonstrated extraction of GPT-2 training examples; this supports the risk, not proof that this product's unselected models leak. [USENIX Security 2021 research](https://www.usenix.org/conference/usenixsecurity21/presentation/carlini-extracting). [OWASP sensitive-information disclosure](https://genai.owasp.org/llmrisk/llm022025-sensitive-information-disclosure/) also identifies training-data disclosure.

**Closure owner and contract:** intelligence/security/privacy either prohibit sensitive tenant-data training/fine-tuning initially, or define trained-artifact lineage, eligible audiences, affected-version suspension/retirement/replacement and backup lifecycle. Do not promise that file deletion erases learned information or that an unlearning technique guarantees erasure.

**Acceptance/dependency:** source withdrawal identifies affected trained versions; ineligible versions cannot remain served or revive from backup. Depends on actual use-case/training decision and retention policy. Scoped retrieval/inference does not automatically require tenant-data training.

## Closure order and ownership

Proposed roles below are not accepted external-specialist assignments. Seung Woo Park obtains named finance/security/operations owners. Packages are local plan references, not invented tickets; executable follow-ups require actual Linear tickets/machine claims first.

| Order | Proposed work/owner | Dependency and acceptance |
| --- | --- | --- |
| 1 | Root PM: first source/auth/tenancy/analytics/intelligence scope and named decision owners | Extend PP-07; values/budgets/deadlines need evidence. Explicitly defer unsupported capabilities |
| 2 | Security/architect/DBA: ED-01/02; connector/security: ED-03 | First profiles; reconcile revocation and AR-02/QA-05. Independent authority/threat review before dependent implementation |
| 3 | Source architect/finance: ED-05; DBA/finance/analytics: ED-04 | Approved source/amount policies; independently expected datasets/amounts. ED-01 remains necessary with precomputed financial results |
| 4 | Intelligence/security/privacy: ED-06 adoption or explicit deferral | Named use cases/training decision; no sensitive shared training before lifecycle closure |
| 5 | Root tech-writer: requirements/contracts/work/screens/acceptance and English reconciliation | Integrate PP-08; one writer per shared trace file, one canonical policy |
| 6 | Separate domain reviewers and root QA integration: final planning re-review | Extend PP-09; cite closure or reviewed scoped deferral for each finding. Owner acceptance is separate from PR merge |

The seven [HB-6 PC findings](../HB-6-content/findings.md) remain OPEN. Existing unresolved versions, Claim keys, numeric values, session limits, revocation bounds, sizing, support combinations, staffing and budget are decisions, not six additional defects. Required initial capability boundaries must be resolved or explicitly excluded before owner planning acceptance; later implementation experiments validate approved candidates. Development remains prohibited until the owner declares planning complete.

## Source register and verification limits

Primary publications consulted: FinOps Data Ingestion/Allocation; FOCUS 1.4 Delivery Handling; OpenCost specification; ClickHouse row-policy/Decimal documentation; OIDC Core 1.0 with errata set 2; OWASP SSRF and GenAI guidance; Martin Fowler's 2015 essay; Carlini et al.'s 2021 USENIX research. Direct links accompany claims. Versionless pages are dated observations, not pinned compatibility evidence. No complete standards-conformance claim is made.

Independent architecture, DBA and QA/security baseline analyses were performed. Root consolidated duplicate training findings and framed delivery semantics as refinement of an existing adapter decision. M02 activation approval is acknowledged; ED-02 does not claim approval is absent. Historical HB-7 PASS remains a scoped preparation review, not exhaustive certification; this deeper review adds closure obligations.

Product/security/integration/performance/recovery/model experiments are **NOT_RUN**. No real provider, DB schema, model or ERP behavior was exercised. [Document verification](verification.md) records checks on the review deliverable. HB-8 completion means review publication; finding closure and planning acceptance remain outstanding.
