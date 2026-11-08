---
type: Implementation
title: Integrated monthly billing overview candidate
description: Records the multi-source and multi-organization synthetic overview, exact amount semantics, independent reviews and browser evidence.
---

# Candidate scope

HB-140 / PR66. Owner requested implementation after the researched information architecture and emphasized the value of an integrated view across many sources and organizations. Root owns code and integration. Claude CLI owns read-only visual composition/render review. A separate DBA reviews monetary semantics and actual changes. This replaces the rejected work-counter hierarchy in the current candidate, without claiming owner FIX, full U01 completion, production authorization or backend readiness.

The overview defaults to all four sources and all three example organizations within one selected example workspace. Source and organization checkboxes support zero, one or several selections. Empty selection never broadens to all. A full-scope reset is explicit. The overview-specific header uses a styled native example-workspace select, leaving other screens' existing account picker unchanged. The legacy directory stays in review tools; /home is still a temporary alias pending HB-75 migration.

# Information and behavior

- Summary shows USD and KRW totals separately, plus known/expected source-by-organization monthly records and distinct closing units. It is not a completion rate.
- Six-month cost trend uses the chosen chart currency only; changing it does not hide other-currency source or organization totals. No FX conversion. A selected-month exact value appears above the bars; the expandable table preserves every exact amount and coverage count.
- Partial bars use a distinct existing neutral fill and explicit text. All-missing data has no numeric amount or zero-cost bar. Verified zero and absence of matching currency data remain different.
- Source and organization breakdowns show amounts in both original currencies. Clicking an item opens the same-scope detail and moves focus into view; closing restores trigger focus. Missing-data exception actions use the exact record ID.
- Monthly lifecycle uses independent collection, reconciliation, allocation, calculation, approval, issuance, accounting, delivery and receipt states. Rows are unique contract paths. Each row displays its full source set. Filtering to one provider includes intersecting whole closing units, never a misleading provider-specific completion count.
- Internal-only fixture paths mark issuance/delivery/receipt not applicable. ERP unknown and receipt pending are independent of issuance/delivery. No payment or accounting actions execute.
- Detail resets on workspace, month, chart currency, source, organization or mode change. Unconnected mode constructs no fixture amounts or closings.

# Synthetic data contract

Fixtures cover May–October 2026, AWS/Azure/GCP/on-prem, organizations A/B/C and two separately scoped sample workspaces. All identities are deliberately synthetic. Source-cost amounts exclude tax; this is a reporting-month example, not production supplier invoices. USD has two minor-unit decimals and KRW has zero. Values are stored and summed as bigint; Number conversion is confined to a bounded chart percentage. Formatting supports positive, zero and negative minor units, although current fixture amounts are nonnegative.

October GCP/organization B is deliberately missing. On-prem/organization C is confirmed zero. Independent controls: May USD 16,470.00 / KRW 6,100,000; October known USD 17,840.00 / KRW 6,900,000, with 11/12 records known. These are fixture test controls, not financial claims. Source counts are not closing-target denominators.

Reporting and settlement months are explicitly equal only in this fixture; real linked-period mapping remains unconnected. Demo snapshot is DEMO-2026-11-02, not a live refresh timestamp. Known amounts are not labelled final invoices. Sales/allocation amounts, forecasts, margin and receivable balances remain unavailable; the screen states this. The richer researched design remains the target for later contract-backed increments.

# Design and independent reviews

External research/adoption boundaries remain in [Unified overview research](unified-overview-research.md). Claude CLI read current rejected captures, the installed frontend-design skill and the research before proposing the integrated composition. Its fixture-data prerequisite was satisfied by the explicit model above and separate financial design review.

The first implementation visual review identified reversed Arabic coverage ratios and focus that did not scroll details into view. Root fixed all three ratio locations with explicit LTR isolation and removed preventScroll; exact-record exception links, selected-month numeric amount and muted partial bars were also added. An intermediate re-review correctly rejected stale screenshots after the first mobile run stopped on a header overflow. Root replaced misuse of the language-specific component with a unique native workspace selector, fixed the overflow and regenerated all six captures. No failed review is recorded as a pass.

DBA final review: PASS for the scoped synthetic implementation. Reviewer independently ran four model tests including separate currency totals, missing/zero/empty distinctions, closing-unit deduplication, large integer exactness and negative formatting. Live financial correctness, permission enforcement and real ERP/receipt outcomes remain unverified.

# Verification

PASS: TypeScript/Vite build; 44 unit tests; 242 artifact hashes. Existing suites: browser37, work32, entry50, setup21 passed during this revision. Final overview suite passed six EN/KO/AR × 1440/390 renders, independent totals, partial/all-missing/zero, zero/one/multiple selections, unique closing units, monthly navigation/back, scoped visible detail with focus return, filter invalidation, exact exception record, workspace scope change, neutral computed panel borders, unconnected no-data and directory separation. It uses a fresh isolated browser context, not the owner's acceptance records. Hash/controlled-input QA assertions were changed to wait for actual observable results; early failures are not counted as passes.

Root inspected desktop KO and mobile AR captures. Final Claude CLI visual re-review: PASS on the fresh KO desktop and AR mobile renders; correct 11/12 ratio, workspace selector, snapshot, selected amount, partial bar and neutral unknown-state borders confirmed. Screenshots do not verify interactions; root browser tests supply that evidence. Native Arabic, screen-reader and Windows/Linux execution remain unverified. Actual host is macOS arm64, Node26.7.0/npm11.19.0, Chromium headless shell via existing Playwright. Build retains the existing large-chunk warning. No runtime dependency was added; the portable npm entry points remain.

Candidate preview: http://127.0.0.1:8790/#overview?lang=ko. The separately delegated design preview on port8787 was not replaced. No owner FIX, merge or deployment is performed by this work.

Stable review mapping: overview:region:layout is retained. The prior settlement-period item is retired in favor of overview:field:reporting-month because this fixture uses an explicitly distinct reporting basis. New source/organization selectors, chart currency, monthly-cost, breakdown, closing-matrix and detail items use new scoped IDs. No acceptance record is carried forward.

CLI telemetry for this implementation turn (four successful process returns, including two review FAIL verdicts before final PASS): {"input_tokens": 30, "cache_creation_input_tokens": 65044, "cache_read_input_tokens": 144733, "output_tokens": 12623}. These are CLI-only cumulative token categories, not root-turn totals or savings. No no-skill control was run. Restricted CLI ignores JEV/settings hooks; no JEV use inside it is claimed.


# Enterprise follow-up

The owner's large-organization requirement is implemented as a selectable 2,400-department synthetic profile with hierarchy search and bounded results. See [enterprise scope and verification](enterprise-scope.md). This supersedes the prior flat-only organization interaction for the enterprise profile, without claiming production service scalability.
