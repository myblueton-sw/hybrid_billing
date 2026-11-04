---
type: ScreenDesign
title: Infrastructure cost composition candidate
description: Reviews collected source rows as infrastructure cost components without inventing approved on-premises ledgers.
status: draft
---

# Scope and evidence

HB-101 implements B01 against the owner's next-screen request and continuous technical-writing review requirement. Root owns model, storage, integration and terminology writes; the assigned B01 implementer owns only its screen, copy, component and CSS files. The authorized companion `docs/ui/screens/B01/spec.md` and v1.16 ui-design119/505 govern asset/lease/depreciation/license/power/facility cost composition, included-cost duplication, owner approval and product relationships. B03 allocation policy and customer sales pricing remain separate. The branch explicitly depends on unmerged HB-143/PR78 and retains B01, `/cost-pools` and internal CostPool identities.

Independent TW first recommends user-facing Korean `인프라 원가 구성`, English `Infrastructure cost composition`, Arabic `مكوّنات تكلفة البنية التحتية`. Internal identifiers are unchanged. Root will maintain one terminology rule/glossary and review each new/modified screen's menu, title, tabs, fields, actions and status copy before and after rendering. No native-language/human professional acceptance is inferred from AI specialist review.

# Data and implemented boundary

Reuse the existing protected collected FOCUS GET and exact18-place coefficients. No API, authority, DB or financial-approval changes. Default source selector is the actually collected VMware sample, visibly labelled as sample source evidence; provider name never establishes native on-premises ownership or complete cost composition. Other actual authorized providers/months remain selectable. Current VMware authorized source contains three explicitly synthetic rows; exact read-back is recorded after verification.

This frontend increment is an editable **review draft**, not an approved cost policy or server-persisted ledger. Legal owner, asset/lease/depreciation evidence, facility/power/license contracts, product inclusion and approval history are not present in FOCUS and remain unverified. Do not fabricate them. The user can name a draft, choose one actual charge month, select at most25currently loaded source rows, assign an explicit cost component category and declare a reference-backed included-in relationship among chosen rows. Unknown evidence stays a warning; collection finalization never means cost approval.

The summary and component editor are real forms with exact selected-source previews. Each row identity is unique. Preserve BilledCost versus EffectiveCost and signed/null values. Do not sum currencies. Save requires a name, specific charge month and at least one selected valid row; cross-period/cross-currency/duplicate/cyclic/foreign relationships block save. Unclassified cost types and missing policy owner/contract evidence remain review warnings and block any implication of approval. Included-in declarations cover whole selected rows only, require a reference and matching charge intervals, and remain user-declared and unverified. The preview separately labels selected-source subtotal, declared-included reference subtotal and review subtotal, excluding each descendant exactly once. Missing source amounts remain missing or partial; signed credits remain signed. Charge-start month is attribution only, without proration or infrastructure-cost eligibility. No authoritative duplicate-cost certification or approved denominator is created.

# Temporary save and authorization

An explicit `Save review draft` action stores only bounded user-entered draft metadata, canonical row IDs/categories/inclusion references and the pinned source/selection references and a SHA-256 fingerprint of the query (never its raw search text) in browser storage. No source amounts, raw source JSON, billing account IDs or credentials are persisted. Browser save/read-back failure is visible. This is labelled `This browser only`; no authenticated author, server revision, financial approval or durable project acceptance is invented.

Restore only after a fresh successful authorized source query and validate the stored schema, company scope, source/selection snapshot, page and all row references against the current bounded source page. Reject stale/malformed/foreign saved drafts. Failed or obsolete reads hide and clear the previous source/draft content; never restore local draft data as fallback. Browser storage is not an authenticated per-user store and provides no cross-user confidentiality guarantee. It is limited to this explicit public/synthetic sample reader. No actual customer data is introduced; current reader is the existing explicit public/synthetic sample path.

All scope/page/cost-basis changes invalidate the displayed composition after the standard unsaved-input navigation guard. A different page is a different bounded edit context; this local increment does not claim cross-page policy assembly. Selection is not authority. Approval/request/simulation/server-write buttons are not offered as working operations; those integrations are explicit unavailable states.

# Composition and implementation ownership

Root may edit new `modules/pricing/pages/infrastructure-cost` models/components/CSS/tests, scoped B01 route and review context in App, B01 menu/title/tab copy and English screen/verification documentation. Reuse the existing compact source/overview shell for this screen; no unrelated shared layout redesign. Add `.agents/rules/product-terminology.md`, an English glossary and a small AGENTS link for the owner's ongoing language-review rule.

Preserve existing four tab keys: summary/costs/products/approvals, with precise user-facing labels. Summary shows selected review amount and editable draft identity, followed by source candidates and components. Costs provides the full component editor. Products explains missing product catalog/contract bindings and exposes only declared review relationships, not invented product records. Approvals states no authoritative history. Actual per-screen review remains visible and owner-only FIX remains pending.

Claude CLI ui-designer must lead composition with actual current renders, planning and official references, then compare actual output. Keep full viewport, local IBM Plex16body/14metadata/20headings, neutral separators, square controls, no colored edge/shadow/gradient. AllKO/EN/AR desktop/mobile variants remain necessary; no unsupported browser/OS is certified.

# Verification plan

Independent financial design/actual-diff review covers selection/inclusion arithmetic, null/credits, duplicates/cycles, mixed currencies, snapshot-bound restore and no approval fiction. Independent security review covers fresh authorized load before restore, malformed/stale/foreign storage, async clear and fail-closed source scope. TW independently reviews terminology and the rendered copy mapping. Root runs model/storage tests, build/asset checks and actual collected browser workflows: select/assign/include/save/restore/invalid save/storage failure/scope changes/Escape/owner-review isolation, six locale/layout captures and previous S04/U01 shared-scope smoke. Do not run retained-store reset fixtures.

# External layout evidence

The official [FinOps Data Center category](https://www.finops.org/framework/technology-categories/data-center/) distinguishes asset cost and service chargeback, including facilities. Adopt the cost-evidence distinction; do not infer complete on-premises cost from VMware labels. [FinOps Allocation](https://www.finops.org/framework/capabilities/allocation/) concerns cost assignment to responsible groups; preserve that as B03. [Cloudscape single-page create](https://cloudscape.design/patterns/resource-management/create/single-page-create/) groups related inputs, exposes validation and protects unsaved edits. Adopt those interaction patterns; reject its maximum-width/layout assumptions and decorative styling in favor of the owner’s full-viewport neutral constraints. The successful Claude CLI response is an AI design recommendation; rendered comparison remains pending.

# Stable implementation items

| Item ID | Location and behavior | Verification |
| --- | --- | --- |
| B01:scope:collected-source | Source/month/basis and current bounded page, explicit sample evidence and snapshot pins | Actual protected-source browser query; source errors and obsolete response checks |
| B01:form:review-draft | Named composition, specific charge-start month, unverified cost categories | Model bounds and actual browser editing |
| B01:table:cost-components | Current-page selection up to 25, exact intervals, cost basis and metadata | Bounded source table and category edit |
| B01:calculation:declared-inclusion | Whole-row referenced acyclic inclusion; gross, declared excluded and review subtotals | Exact18 signed/null/cycle/foreign/currency/interval tests and browser cycle validation |
| B01:action:browser-save | Explicit metadata-only local save, readback, fresh-authorized restore, visible failure | Model storage boundaries and browser save/restore/failure |
| B01:state:approval-unconnected | Product catalog and authoritative approval history unavailable | Products and approvals views; no simulated action |
| B01:state:source-unavailable | Loading and errors clear source/draft, no offline fallback | Browser 403 and delayed response replay |

The summary, costs, products and approvals tabs preserve the same draft during view changes. Scope and refresh changes use the existing unsaved-input guard. Owner FIX remains pending for all candidate items and variants.
