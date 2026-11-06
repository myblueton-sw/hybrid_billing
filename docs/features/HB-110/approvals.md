---
type: ScreenDesign
title: Read-only approval inbox and result evidence cards
description: Designs B09 request inspection without inventing financial approvals or command authority while the producer is unconnected.
status: draft
---

# Scope, ownership and evidence

HB-110 implements canonical B09 `/approvals`, pending/decisions/delegation/history/recognition views from the authorized English B09 spec. Root owns PM, UI/integration and QA; separate finance/TW and security/contract reviewers inspect design and actual changes; Claude CLI ui-designer leads composition and final rendered comparison. An isolated writer may own the read model/client/tests. The branch stacks on open B08 PR85 at `8f9a902e4d262d48244e654f25b2e545a60e3913`; main was separately read as `36394edf179d544d7ec2756f5329eba60bcc2d4e`. Marketplace remains held for CSP-specific differences, not FIX. Prior leaves remain unchanged.

Previously verified capability discovery found no financial billing-result/approval/claim/ERP-recognition producer. Source-import confirmations and customer-usage export approvals belong to other authorities and must not populate B09. FOCUS costs are not approval targets. No real requests, authors, decisions, ceilings, delegation grants, result digests or counts can be inferred. Replace the generic DEMO approval ledger; the normal service view is honestly unconnected, not an empty verified inbox or zero pending count.

# Frontend read contract and integration boundary

This increment proposes a consumer-only read projection `GET /api/approval-review`; no deployed producer is claimed, no backend endpoint is implemented, and no financial command URL exists. The existing local reader excludes this proposed route; an actual 404 is to be verified during runtime QA. Only 404/501 means unconnected; denial, malformed data, stale scope and server failures remain distinct errors and remove old records. Use same-origin GET, existing trusted local client header, no-store, cancellation, bounded response and strict validation. A proxy forwarding this read path is development wiring, not proof of an approval service. Future producer/auth/disclosure compatibility requires its own independent implementation review.

Response contract draft: schema1, numeric_format `decimal-strings-read-v1`, read_only true, commands `unconnected`, source_mode `live` or `synthetic`, server-owned actor and opaque authorization-scope digest, snapshot and permission_revision digests, exact view/filters/cursor, observed_at, nullable exact total, next_cursor and at most25 request records. Source_mode synthetic is always prominently labelled as synthetic review examples, never actual approval targets; it is used only by isolated intercepted QA. Because no live producer is bound, live responses fail closed as producer_unbound until a later independently reviewed binding change; source_mode alone is never proof of that review. The normal unconnected view loads no fixtures. There is no URL option, hidden window hook or persistent fixture provider that creates approvals.

Requests carry canonical IDs, action kind (billing result, final ERP amount, adjustment, policy or monthly expense recognition), source status, target reference/label with nullable legal entity/customer/period, explicit author, updated/deadline times, and at most16 unique currency+amount-basis observations. Each amount has base/candidate/declared difference/ERP values as bounded finite decimal strings or null. Different currencies and net/tax/gross/expense bases remain separate. If both base/candidate and a declared arithmetic difference exist, require exact consistency without floating arithmetic; no missing amount or counterpart becomes zero. Decimal transport bounds (at most80 characters,76 digits and38 fractional digits) reject unsupported values; they do not activate a numeric profile, rounding point or currency quantum. ERP amounts are external observations, never inferred. Explicit ERP applicability (applicable/not_applicable/unknown), external revision and as-of stay separate; null cannot mean not applicable. IDs/digests are not permission grants.

A bounded evidence object supplies nullable input/result digests and revision, declared dependency/completeness/validation state and at most16 plain-text blockers. These are source declarations, not a client certification of billing readiness. Missing authority/limits/actual approver remain explicitly unknown. An approval record may be inspected without authorizing any financial action. No client approves, rejects, hands off, delegates or posts, even when source evidence is declared passed. No batch selection, approval simulation, synthetic decision save or disabled pretend action bar. Only an actual future separately reviewed command producer can enable those workflows.

Delegation has no verified grant producer and remains an unconnected service view; request rows cannot become delegation grants. Recognition request inspection uses only monthly_recognition kind and clearly separates monthly expense recognition from customer issuance, receivables, posting and collection. No fake monthly batches or reconciled zeros.

# Screen behavior

Retain existing commercial shell, title/review control and five stable tabs. Full-width scope toolbar: action kind and bounded text search, explicit Search and New query. These filters are request evidence only, not current authority or amount eligibility. Do not offer legal entity/customer/amount options without a producer. Pending/decisions/history/recognition show a bounded request table when read data exists and an honest unconnected empty state otherwise. Distinguish loaded empty (exact0), unknown total, partial evidence, blocked/stale/expired/superseded records, failed/denied/loading and unconnected. Source synthetic mode carries its visible label on list and selected detail.

Select only a canonical current-page request to open a result card, retaining scope and focus. Changing view/filter/cursor/permission context clears selection immediately; abort and monotonically gated request identity rejects late results. History back/forward restores filters but never a previous record without a new authorized read. Cursor requires the exact snapshot and permission revision; stale/denied/error states remove old request, amount and evidence panels. No persistent cache, localStorage, draft, download or external message.

Result card: action/target/period/source status first; explicit source author, actual decision actor if provided, deadline and update time; separate per-currency/basis base/candidate/difference/ERP amount cards; blockers and source-evidence disclosure. Unknown amounts and digest/revision/authority fields stay unknown. Self-approval and delegation/limit policy are not guessed from names or role-preview AP. Clearly state that this screen reads requests and cannot execute approval; do not render an approval button that silently simulates success. Row inspection, close/Escape/focus return, filters, paging, retry/new query, tabs and related B08 navigation are actual interactions.

# Composition and external evidence

Use full available viewport, canonical IA/logo/local Plex, 16px body/controls/table,20px h2,14px meta and Arabic1.8 line-height. Neutral square separators, no decorative colored borders/shadows. Desktop list/result split and inline mobile details; separate full-column table disclosure. Exact money/IDs/times/counts use complete LTR isolates; no-wrap count groups and Korean money words. No full-page inherited-bold selection block. Review controls use complete request/permission/snapshot/view/selection/evidence/error identity without assistant owner acknowledgment.

[Microsoft approvals documentation](https://learn.microsoft.com/en-us/power-automate/approve-reject-requests) informs inbox-to-details and separate response/history concepts. [GitHub deployment review documentation](https://docs.github.com/en/actions/how-tos/managing-workflow-runs-and-deployments/managing-deployments/reviewing-deployments) informs explicit target/actor/effect context and policy-dependent self-review. Reject their domain commands, licenses, permission defaults, colors and deployment/payment effects; no account/tool setup is imported. Exact pages are consulted during this ticket before composition. Existing Carbon table patterns inform bounded disclosure, not copied assets or styles.

# Stable items and validation

Items: B09:scope:request-filter; B09:data:inbox; B09:action:inspect-request; B09:data:result-card; B09:data:evidence-boundary; B09:data:decision-history; B09:data:delegation-boundary; B09:data:recognition-boundary; B09:state:async-unconnected.

Independent finance/permission/TW design and actual-code/render reviews. Unit tests cover strict source-mode/readonly/command denial, signed exact/null/currency+basis distinctions, snapshot/permission/cursor and response identities, bounded canonical page selection and safe source states. Verify actual default404→unconnected; isolated synthetic intercepted responses exercise pending/history/recognition result cards, empty/unknown counts, source-null, denial/stale/error/late response, current-page detail/focus and no state/financial writes. Capture all30 view/locale/layout combinations (five × KO/EN/AR ×1440/390), with unconnected service evidence separate from synthetic cards. No test creates owner FIX. Full financial approval/SoD/limits/delegation/history/ERP integration, native/human finance, other OS/engines and production capacity remain pending.

# Independent design conditions

Finance/TW review conditionally passed explicit approval-stage separation, bounded decimal-string display with no universal precision, ERP applicability/external evidence, source decision versus issuance and source-owned actor context for My decisions. Security review conditionally passed only an explicitly unbound consumer prototype: fail closed on live data until a separately reviewed producer binding; preserve synthetic labeling with no URL/window/storage bypass. Cursor binds snapshot and permission revision, response echoes exact view/filter/cursor and a server-owned authorization-scope reference. Denied/stale/malformed/query transitions remove old data. Source amounts, role preview and result digests never create authority. No original financial/SoD/ERP approval acceptance is claimed.

# Designer reconciliation and item mapping

Claude CLI ui-designer supplied composition after reading the actual shell capture, B08 precedents, canonical foundations and frontend-design skill. Adopted full-width split inbox/card, inline mobile detail, hidden unconnected toolbar, closed full-table/evidence disclosures and raw source amounts. Its initial suggestion to collapse billing_result into another stage was rejected against the authorized B09 spec and independent finance review. The numeric contract has no universal scale18; exact bounded decimal strings replace that assumption. Future live cards are not enabled by this design: live payloads fail closed pending a separately reviewed producer.

| Stable item | Location and behavior | Relevant states / verification |
| --- | --- | --- |
| B09:scope:request-filter | Kind and request-reference/target search; reset cursor pins on change | Loading, loaded, error; query/API and browser checks |
| B09:data:inbox | Bounded 25-row page with exact/unknown counts and labelled synthetic origin | Empty, unknown total, partial, unconnected; strict validation and captures |
| B09:action:inspect-request | Current-page ID opens card; close/Escape returns visible trigger focus | Selection, query/page/view transition, unmount; browser and independent source review |
| B09:data:result-card | Separate stage, target, actors, timestamps and currency+basis source values | Missing/zero/exact decimals/blocked; API tests and finance review |
| B09:data:evidence-boundary | Closed source checks/digests; no authority inference or command | Unknown/stale/partial evidence, producer-unbound; independent security review |
| B09:data:decision-history | Historical decisions separated from current authority; My decisions binds actual actor | Approved/rejected/source actor; API and browser checks |
| B09:data:delegation-boundary | Honest unconnected grant service, no inferred delegation | Unconnected, no grant query or retry; captures |
| B09:data:recognition-boundary | Monthly recognition request kind only; no posting/reconciliation claim | Monthly request and missing ERP; API and browser checks |
| B09:state:async-unconnected | Explicit loading/unconnected/denied/stale/failure/retry; discard old evidence | Late success/error, permissions, snapshot and malformed response; browser checks |

All items remain candidates for owner review, with financial producer/commands, actual grant authority and native/human acceptance deferred. The common review control includes the exact view/query/actor/scope/snapshot/permission/record/error identity without creating owner confirmation.

# Final visual correction scope

The actual-render designer review required visible active mobile tabs and direction isolation for source English text on Arabic pages. B09 alone now uses a wrapping two-column mobile tab grid (last view spans both columns), replacing its inherited horizontal strip; no shared navigation stylesheet changed. Source titles/blockers and table actor names use auto-direction isolates. Normal-size mobile crops and selected-tab bounds verify the revised candidate. Detailed results and remaining producer/native/owner limits are in [verification](verification.md).
