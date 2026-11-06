---
type: ImplementationEvidence
title: Organization, work, import and assistant frontend candidates
description: Records the seven dedicated HB-144 leaf implementations, their planning trace, local interactions and unverified integration boundaries.
status: draft
---

# Scope and ownership

[HB-144](https://linear.app/hybrid-billing/issue/HB-144/implement-the-32-remaining-commercial-frontend-screen-candidates) owns O01, O02, O03, U03, U04, S03 and A01 in this batch. This writer owns only `apps/console/src/modules/organizations/remaining/` and this record. Root owns App routing, canonical tabs, review controls, integration, independent review and rendered QA. Other writers own separate domains. No existing leaf, shared source, backend or package manifest was edited here.

The isolated staging branch is `work/HB-144-work`, based on batch plan commit `0aee256`. This is not a fresh-main branch or a separate machine claim. Root retains the batch machine claim. No merge, deployment, real-account, financial operation, import, public message or owner FIX was performed.

# Planning and contracts

The seven original screen specifications in the authorized companion workspace supplied field, action and layout requirements. `screens.json` retains their exact hashes and linked screen tickets. `contracts.ts` freezes the per-screen and per-view field inventory, primitive reference/date validation and the explicit unconnected evidence state. It is an implementation inventory, not a deployed response or command contract. No new HTTP endpoint or producer was added.

All source evidence is unavailable in these leaves: authenticated scope, as-of time, exact total, financial amount, receipt, permission and job state remain unknown. The implementation deliberately performs no source read, fake source success, synthetic mode switch, generated record row or financial calculation. A later producer requires independently reviewed scoped schemas, authority, current evidence and asynchronous boundary handling. Root's proposed shared read contract is not silently activated by these views.

| Screen | Views | Dedicated composition and local behavior |
| --- | --- | --- |
| O01 | tree, accounts, groups, history | As-of/entity/relationship/provider scope review; unconnected tree and account/history columns; group-name and canonical-member draft, duplicate and format rejection; no ownership or access grant. |
| O02 | targets, impact, approval, result | Staged target-kind/manifest/revision/effective-time draft; before/after relationship, financial and ERP/disclosure impact panels; approval prerequisites; separate unknown success/held/failed/unprocessed results. Input-format review cannot establish actual impact or approval. |
| O03 | parties, schedule, products, documents, termination, payment-terms, recognition | Separate buyer/payer/issuer references; direct/resale draft; distinct service and billing intervals, exclusive-end validation; component/allowance references; protected payment evidence without bank details; separate contract and final-settlement state; nonnegative integer days; separate billing/recognition cadence. No due-date, proration, amount or ERP calculation. |
| U03 | tasks, notifications, detail, delivery, payment-terms | Job/incident or public document/inquiry reference and month/filter review; distinct job and incident status inventories; delivery versus user reading; payment authority/freshness unknown. Explicit admin audience is required for internal job fields. Missing audience gets restricted customer presentation and public-only review field IDs. |
| U04 | findings, review, evidence | Findings list versus source/contract/comparison evidence, certainty unknown; manual review-note draft with kind/scope/meaning/reference validation; empty or bare completion rationale rejected; evidence/correction/access branch separate. No judgment, assignment, financial adjustment or resolution is persisted. |
| S03 | imports, quarantine, completeness, extraction | Receipt/source-period/correction-reference draft; quarantine metadata-only boundary; expected versus received bytes/rows/partitions/watermark; original versus extraction candidate; evidence-reference review. No file picker/upload, extraction confirmation, retry or publication; worker confirmation remains S06. |
| A01 | conversation, plan, results, retention | Scope/month/currency and request composer with character count; context changes clear request; input review only, no transmitted question, transcript echo, generated answer or tools; fixed unknown server-plan fields; separate job result and retention states. Conversation deletion does not imply job/approval/audit deletion. |

Drafts remain in component memory and clear on owning view/context remount or explicit clear. Root must key the dispatcher by screen/locale/context; the dispatcher also keys its subtree by screen, locale, view, audience and parameter identity. `onDirty` reports ephemeral edits. Review serialization contains only screen, view, audience, phase, boolean draft presence and connection boundary; entered references, rationale, request content and credentials are excluded. No browser storage, network, clipboard or content logging is used.

Stable item IDs are screen-prefixed and exposed through the review callback. Actual domain boundaries and forms have `data-item-id`; normal users are not shown design bookkeeping. Root supplies the owner-only review button. No local input-format review means owner acceptance or a completed server command.

# Visual composition and references

Actual Claude CLI invocation completed with exit 0 using the installed `ui-designer` role, restricted Read/Glob/Grep, and the official local frontend-design skill. The prompt supplied all seven original specifications, mandatory owner constraints and the existing S04 render directory. The designer reported reading all seven specs and viewing `en-1440-final.png`, `en-1440-detail.png` and `ar-390-final.png` from HB-143. It did not inspect the absent HB-112 directory, current new leaves, native Arabic quality or live producers. The CLI's statement that it did not recursively invoke itself is not an invocation failure; the parent process invoked it.

External official references opened during this implementation:

- [Cloudscape tree view](https://cloudscape.design/components/tree-view/): hierarchy as a distinct region; no synthetic organization nodes or hidden child counts.
- [Cloudscape wizard](https://cloudscape.design/components/wizard/): staged target, impact, approval and result reading order.
- [Cloudscape split panel](https://cloudscape.design/components/split-panel/): keep list context and evidence detail separate.

The fetched pages expose component-guide navigation; interactive examples were not executed. Concrete layout decisions therefore also rely on the actual CLI composition and project planning. No claim is made that those pages certified the chosen dimensions. Adopted: distinct domain reading order, staged review, list/evidence separation and container-width adaptation. Rejected: upstream colors, components, assets, shadows, rounded geometry, default small typography and unbound command success. Nothing was copied or installed.

The resulting CSS is scoped to `.work-domain`, uses the complete available width, 16px body/control/table text, 20px h2, 14px metadata, Arabic 1.8 line height, local inherited font tokens, square neutral separators and no shadows. Body columns adapt at leaf-container widths 760 and 1080px. Narrow tables retain all columns inside a labelled scroll region. All drafts have labelled controls, explicit local boundary, clear and input-review actions; failed validation produces linked error summary and first-invalid focus.

# Verification and remaining boundaries

- PASS: `npm run typecheck` after correcting a TypeScript case-only resolver collision by naming the contract leaf `ContractProfile.tsx`.
- PASS: `npm run build` (TypeScript and Vite). Existing Vite chunk-size warning remains. Root has not mounted these leaves yet in this isolated tree, so bundle success is not route/render evidence.
- PASS: formatting executed with the installed Prettier binary on the owned source directory; all maintained files remain below 1,500 physical lines.
- UNVERIFIED here: actual App integration, all rendered locale/viewport variants, keyboard round trips, independent current-code review, browser interactions and final Claude render comparison. Root performs those in the integrated batch.
- UNVERIFIED: live authentication, field/tenant enforcement, receipts, billing/ERP/recognition, imports, assistant, jobs, durable drafts or customer authority. UI presentation is not server authorization.
- UNVERIFIED: native Arabic acceptance, human finance/billing practitioner acceptance, Windows/Linux executions and universal OS compatibility. The leaves use portable React, browser controls and CSS with no OS-specific product dependency.
