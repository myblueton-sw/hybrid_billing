---
type: ScreenDesign
title: Commercial U01 overview redesign
description: Applies the planned separation of amount meanings and action-led overview hierarchy to the existing scoped cost and closing prototype.
status: draft
---

# Planning evidence and ownership

Root owns PM, implementation and QA. Inputs are the authorized root U01 specification, v1.16 UI success criteria and HB-01/23/30/31/39 requirements, plus payment-terms.md and erp-recognition-cadence.md. HB-85 starts from current origin/main with pending HB-76 PR76 incorporated explicitly. This scope owns operations/overview components and scoped QA, not the shared menu/shell. U01 is the first screen in the owner-requested commercial redesign; no all-screen completion is claimed.

U01 must support understanding amount meanings, closing denominator and blockers, then opening the next permitted work. Desktop prioritizes scope, separate cost/sales/allocation, closing/work and blockers; mobile places issues/actions ahead of charts. Currency, amount meaning, freshness and scope remain explicit. Recognition, billing, delivery and receipt are independent; monthly ERP recognition is not gated by customer invoicing.

# Observed implementation and boundaries

Existing source costs use bigint minor units and scoped fixture rows. The closing model has independent nine-stage fixture states. The model does not provide sales/allocated amounts, authentic permission projection, responsible users, deadlines, receivable balances or receipt freshness. This change cannot fabricate those fields or claim a live overview aggregate. Synthetic fixtures stay labelled; unconnected mode shows no fixture amounts or stage counts.

Do not invent owner/date/currency defaults or a functioning refresh endpoint. Show missing integrations as unavailable/unknown, never zero, approved, not applicable or permission denied. A partial known cost remains partial. Sample source coverage is not production completeness and cannot authorize input or billing use. Favorites remain unimplemented pending their policy/identity contract.

# Composition references

Consulted official Cloudscape service-dashboard guidance (https://cloudscape.design/patterns/general/service-dashboard/) and CloudZero navigation guidance (https://docs.cloudzero.com/docs/navigating-cloudzero) on 2026-11-03. Adopt the monitor/investigate/action hierarchy, predefined priority layout and scoped cost drill-through. Do not import external assets, optimization/product scope, rounded controls or decorative colored borders. Our U01 planning and independent billing meanings take precedence.

Claude CLI composition and render comparison are recorded only when actually returned. The existing canonical shell, local fonts, logo, full available width and EN/KO/AR remain.

# Candidate screen behavior

Present scope and fixture snapshot; distinguish source cost from unknown sales/allocation; expose coverage/units as nonfinancial operational facts. Actionable missing-input and unresolved ERP items precede charts. Preserve monthly trend, source and organization breakdown and all nine closing states with bounded rows/details. Receipt/deadline sections state their actual missing data, with no invented overdue total. Development-scale selection belongs to an explicit example configuration disclosure, not the ordinary business toolbar.

Each action opens the exact existing scoped prototype detail or canonical workbench without claiming external resolution. Scope changes clear selection and old detail; a list-only filter does not alter aggregate totals. Generic cross-screen navigation cannot masquerade as an exact bound backend object.

# Stable design scope

Retain existing overview:* IDs and add U01:summary:amount-meanings, U01:work:attention, U01:state:receivables and U01:scope:provenance for the revised regions. Existing owner FIX is not inherited. Per-screen review remains available through the unchanged shell.

# Verification plan

Reuse existing numeric/scope/filter tests. Run typecheck/build, existing overview browser regression for source/month/org/currency filtering, 2,400-department bounded rendering and focus/detail invalidation. Add scoped layout/meaning checks for EN/KO/AR desktop/mobile, unknown versus zero, action-before-chart ordering, no overflow and neutral borders. Record independent source/financial review and Claude CLI render findings separately. Runtime targets beyond the actual host/browser remain unverified.

# Implemented screen composition

The existing HB-76 shell, menu, tabs and review control remain unchanged. The body now uses three separate amount-meaning columns, with source cost split by currency and unavailable sales/allocation stated visibly. Settlement attention and data exceptions precede analytical charts. Compact view renders settlement attention before amount summaries in the DOM, not only visually. Example size and data-set switching sit in an explicitly named example configuration disclosure.

The attention counts cover blocked whole closing units, approval-pending units and unknown ERP outcomes. These overlapping categories are not added together. Each control inspects the first matching scoped fixture unit and restores focus on close. It does not approve or resolve a unit. Independent issuance, accounting and receipt counts are available in a disclosure. Source-record coverage remains a separate denominator.

The source and organization tables become stacked rows on compact containers, preserving complete currency amounts rather than initially clipped fragments. The nine-stage closing matrix remains bounded, horizontally scrollable and keyboard focusable. Unknown money cells retain explicit currency. Partial totals are qualified both in amount cards and the selected chart headline. The example snapshot is not live freshness.

| Stable item | Location and behavior | Integration state |
| --- | --- | --- |
| U01:scope:provenance | Scope toolbar followed by fixture, tax and FX basis | Synthetic snapshot; no authoritative as-of |
| U01:summary:amount-meanings | Separate cost, sales and allocation sections | Costs are fixtures; sales/allocation unconnected |
| U01:work:attention | Overlapping counts and exact scoped fixture inspection before charts | No assignee, deadline or command authority |
| U01:state:receivables | Missing receipt-allocation and payment-term explanation | Unknown balances and overdue amounts |
| U01:scope:example-data-set | Collapsed example configuration, scoped A/B fixture switching | Development fixture switch only |
| Existing overview:* items | Trend, source/org breakdown, nine-stage matrix, filters, paging and details retained | Existing scoped synthetic behavior retained |

# Review disposition

Independent planning review returned conditional PASS: retain distinct money meanings, fixed whole-unit denominator, overlapping attention groups, exact scoped inspection and honest unknowns. Independent implementation review initially failed stylesheet ordering; root corrected imports and subsequently found the shared shell's equal-specificity heading/control styles through computed browser inspection. Scoped selectors now preserve body/table/control typography. Source re-review passed the import, DOM-order and scope fixes, then the final partial-label, currency and compact-table changes.

Claude CLI ui-designer returned a composition proposal and inspected six first-pass EN/KO/AR renders. The first composition invocation could not access the official frontend-design skill; the render invocation successfully read the installed official skill. Its material finding was a partial chart headline lacking an inline qualifier; this is corrected. Root also corrected table/control sizes, named attention controls, single-page disabled appearance and compact money clipping. Final Claude CLI render re-review returned scoped PASS with root-computed typography evidence and nonblocking refinements retained in the verification report. No review result is owner acceptance or full U01/business-service completion.
