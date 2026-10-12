---
type: Verification Report
title: HB-19 high-level architecture review and verification
description: Records provenance, independent review, documentation checks and unresolved implementation evidence.
status: draft
---

# HB-19 verification

Tracking: [HB-19](https://linear.app/hybrid-billing/issue/HB-19/document-high-level-architecture-and-component-data-flows-before). Accountable owner: Seung Woo Park. Root is the sole writer and PM/planner integrator; a separate read-only reviewer assesses architecture, billing and security. Execution machine: `machine:mac_mini`. Branch: `docs/HB-19-high-level-architecture`.

## Scope and provenance

Baseline main: `e436a49424db7491140308f8c3895ce75cf176c3`. Proposed SaaS input: [PR14](https://github.com/myblueton-sw/hybrid_billing/pull/14), OPEN at publication preparation, head `6acb38c68117ff1f1865776aca00f87c61956196`. Five permitted files: README, architecture readiness, new high-level architecture, its static SVG and this report. No product code, runtime configuration or detailed schema is included. The original 43 requirement groups and 49 screen identities remain unchanged.

A local untracked draft preceded restored browser access. Formal publication began only after verifying the actual Hybrid Billing workspace/team/project, creating HB-19, assigning Seung Woo Park, setting In Progress and claiming `machine:mac_mini`. The isolated branch starts from main; PR14 content is referenced as a proposal, not silently merged.

AGENTS.md and the available workflow rule were read. Companion rules/skills, `.linear.json`, planning gate, detailed planning inputs and artifact/Git policies are absent locally. The owner's explicit instruction to proceed despite other-local inputs applies to this bounded document task; no missing policy is invented. Metadata follows existing repository documents; full OKF v0.2 compliance cannot be certified without its missing local specification. Planning completion remains undeclared. Static SVG is intentionally versioned design artwork; local PNG, ZIP and rendering helper are excluded.

## Review and verification

The earlier independent draft review passed after a correction omission was resolved: issued/confirmed history is immutable; linked adjustments re-enter applicable gates; closed-period corrections use approved open-period adjustments; invoice cancellation does not imply recognition cancellation, receipt reversal or refund. The reviewer read the full draft and inspected its matching PNG overview. F05/F08 and the visual caption were re-reviewed.

Final repository integration review: PASS. Separate read-only reviewer inspected all five actual files, including untracked content, and reported no blocking findings. Financial and tenant gates, correction immutability, recognition/receipt/refund separation, recovery controls and PR14 proposal provenance remain consistent. Reviewer independently checked relative links and SVG XML and confirmed that only the SVG description metadata changed from the previously visually reviewed artwork. Tracking/machine verification is root-observed browser evidence, not independently exercised by the reviewer.

Root documentation QA: PASS for local Markdown links, file-size bounds, ordered F01–F12 contracts, two fenced Mermaid source views, passive SVG XML without scripts/external references, unchanged diagram geometry, credential-pattern scan, unchanged canonical reconciliation JSON and whitespace. Staged-diff review is performed immediately before commit; scope is limited to the five listed documents/artwork.

The SVG is a summarized static overview, not a node-for-node rendering of both Mermaid views. Its geometry is unchanged from the visually inspected draft; only tracking metadata changed. Mermaid renderer: NOT_RUN.

Runtime, monetary execution, tenant/security enforcement, provider/ERP compatibility, deployment, load/recovery and user-task/accessibility tests: NOT_RUN. Design review does not prove those behaviors. Financial/tenant invariants are review criteria, not executable test results.

## Handoff and next decisions

Review module/data ownership and permitted flows first; then transaction and correction boundaries, command/event/query envelopes, provider capability profiles and workload evidence; only then compare storage, jobs/broker, read projections, worker and deployment configurations. No technology, node count or SLA is approved. Full trace reconciliation and explicit owner planning completion are prerequisites to product development.

This package is prepared for PR review. Merge authorization, merge read-back and ticket completion remain pending. The machine claim will be released at publication handoff; the ticket remains open pending authorized integration.


## 2026-10-12 logical architecture revision

The owner explicitly requested a restart from backend high-level **logical** architecture, at concept level. Root owns integration and all edits; a separate read-only architect supplied design advice and reviewed the actual revised document against `83f270a`. The existing ticket/PR is reused. Original workspace rules, skills, binding and planning inputs are available in this authorized environment; earlier absent-input statements above are historical, not current assertions.

An isolated `docs/HB-19-logical-architecture` working branch extends the published PR head and merges main `b5c65cf66992f703247b8e4713f86cef7835b767` without conflict. Publication advances the existing PR branch without rewriting history. Four revised files: high-level architecture, README/readiness entry points and this report. The original SVG stays a supplemental data-flow sketch with unchanged bytes.

The document now starts with logical concerns, a responsibility/ownership map, six principal interactions and three consistency boundaries. It preserves the merged HB-16 contracts and conditional PR14 SaaS proposal. Latest HB-20 input at local branch commit `62f7df1` supplies the >100M source records/day premise and recorded object-storage/Parquet foundation; unmerged engine/catalog proposals are not adopted. This is no new physical schema, topology, technology comparison or deployment configuration.

Independent actual-document review: PASS, no blocking financial/security contradictions. Reviewed fixed-release inputs, policy revisions, exact-result approval, economic Claim ownership, external unknown outcomes, current scope and recovery. Two clarity suggestions were applied: direct monthly evidence/calculation paths to recognition in the overview, and explicit deferral of HawkEye/TokenMeter in the supplemental source context. Entry-point changes summarize the revised scope. Final independent integrated four-file re-review: PASS; both clarity findings resolved, no blocking architecture/financial/security contradictions.

Root QA: PASS for OKF type frontmatter, balanced source fences, local Markdown link targets, F01-F12 retention, original SVG bytes, current-main reconciliation JSON, scoped whitespace and added-line common credential patterns. An initial JSON preservation check compared against pre-HB-16 `83f270a` and failed because the merged HB-16 changes differ; the corrected current-main baseline check passed. This was a validation-baseline correction, not an artifact defect. No exhaustive new original-source audit is claimed. Mermaid rendering, runtime/security enforcement, load/recovery, compatibility and final policy/planning acceptance remain NOT_RUN/UNVERIFIED.

PR review/publication does not authorize merge or product development. The machine claim will be released at handoff and recorded on the ticket; ticket remains In Progress until authorized integration/read-back.


## AI/RAG integration and HA extension

2026-10-12 owner extension: reserve shared AI analysis/RAG integration for both backend and collection, then explain HA on failure. Root owns all document/artwork edits on the existing ticket branch; separate read-only architecture/security review supplies design advice and inspects actual revisions. Scope: high-level architecture and intelligence notes, this record, two new passive SVG solution/HA views. Local browser HTML and rendered PNG remain preview artifacts under output, not deployed products or a new application.

The solution view now separates processing from object storage, ClickHouse OLAP DBMS and PostgreSQL transactional authority. Parquet remains a file format and Iceberg/Catalog table metadata, not another independent storage service. Shared AI includes permitted collector evidence-reference triggers, backend AI requests, governed retrieval/read-only data tools, embedding/index worker, derived vector/keyword index and isolated model gateway/runtime. Findings/proposals retain ordinary deterministic eligibility and financial approval. Retrieval applies current permissions before search and context/output release, including index/cache lineage and latest deletion/revocation controls. External endpoints and training are not implicitly approved.

HA is a candidate three-independent-failure-domain profile, not three total servers or measured availability. Dedicated Kafka controller roles, durable receipt/job semantics, strict synchronous PostgreSQL candidate promotion/fencing and CH publication/read-release boundaries are specified. The two-replica CH insert acknowledgement hypothesis explicitly pauses new projection publication after replica loss. Object/catalog/index HA products and exact quorum profiles, N-1 capacity, RPO/RTO and backup/DR location remain unresolved. Two AI service instances do not certify redundant model/GPU resources.

Independent design advice and actual three-Markdown/two-SVG design review: PASS with guardrails incorporated; no blocking financial, authorization or failure-policy contradiction. A nonblocking diagram suggestion was applied: explicit cited-result/proposal return paths to backend and collection through ordinary gates. Root verified the added passive SVG paths and visually inspected the rendered desktop PNG. Two attempts at independent arrow-only re-review failed because the inherited model was at capacity; that targeted re-review is UNVERIFIED, not falsely recorded as PASS. No financial/authorization/HA decision changed in that clarity fix.

Root QA: PASS for Markdown type frontmatter, fences and local links, both SVG XML structures and absence of scripts/foreign objects/external resources, common credential patterns, original reconciliation/artwork preservation, F01-F12 retention and scoped whitespace. Rendered solution and HA PNGs were visually inspected for readable labels and routing; preview anchors are present in the same local HTML. Browser runtime, inference, embedding, customer-data use, failover, restore, N-1 capacity and performance tests remain NOT_RUN. Original planning gate and pending merge status remain unchanged.
