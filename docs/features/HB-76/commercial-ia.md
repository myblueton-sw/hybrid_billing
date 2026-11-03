---
type: NavigationDesign
title: Task-based commercial menu draft
description: Renders the owner's revised menu draft with direct overview and reports, business workbenches, lower collection/administration and contextual routes.
status: draft
---

# Current revision and scope

The owner requested the latest menu draft on screen after the menu-purpose audit. This replaces the earlier eight-peer-group presentation for this navigation scope. Root owns implementation and QA. Canonical IDs, screen contracts, module ownership, server authority and current owner-review rules remain unchanged. service-ia.ts is the active presentation mapping. It describes route placement, not capability completion.

# Visible menu

| Area | Entries |
| --- | --- |
| Overview, direct | U01 |
| Costs and allocation | S04 cost/usage, B01 cost pools, B03 allocation policies |
| Settlement | B08 runs, U02 closing management, B09 approvals, S06 reconciliation/differences, I02 external submissions/results |
| Billing | I01 documents, B11 balances/credits and inquiries through existing balances/cases views |
| Organizations and contracts | O01 organizations, O03 contracts, B04 sales pricing, B05 exchange-rate policies |
| Reports, direct | R01 |
| Lower Data collection | S03 received inputs, S01 connections |
| Lower Administration | M01 users/access, P02 integrations, P01 platform operations |

My work is a global menu containing U03 work/notifications and execution-job access. A01 remains an authorized global Assistant entry. M05 stays self-only. C01–C03 and L01–L03 retain their separate audience surfaces.

# Contextual access and permission fallback

Contextual ownership: U04 from U01; S02 from S01; B02/B06/B07/S05 from S04; B10 from B08; I04 from I01; I03 from I02; O02 from O01; R02 from R01; M02–M04 from M01; P03–P07 from P01. This accounts for all 49 canonical IDs exactly once without requiring 49 visible menu items.

Related-work disclosure provides guarded, permission-filtered canonical links for contextually placed workbenches. I01 additionally links to I03 for non-ERP file purposes. Generic links deliberately discard unrelated object IDs, snapshots and filters rather than imply the destination has inherited valid object context. Existing exact row/document evidence links remain unchanged.

Users allowed a contextual child but denied its parent receive an authorized fallback menu entry. S05 is an explicit exception: its live viewer needs exact object context, so no generic working evidence-search/viewer is invented. Independent evidence discovery for an S05-only user remains unimplemented; required object evidence and its existing exact links are preserved. Do not claim this menu draft completes that separate capability.

B11's two entries share one canonical route with actual balances/cases tab keys. Current selection follows the active tab. An active contextual page highlights its owning visible workbench; fallback children can highlight themselves. Lower operational placement does not turn intake into configuration-only work. Monthly recognition opens directly through I02 without requiring an invoice.

# Review and actual validation

The same previously reviewed neutral full-width shell is used. No external assets, dependencies or OS-specific runtime interface is added. Source-derived review revision changes with the navigation; no previous FIX is transferred.

Independent design review required child-only entry and non-ERP file discoverability. Actual diff review found generic S05 fallback and inconsistent Arabic meanings; both were corrected. Final targeted source re-review PASS. 76 console tests and typecheck/build PASS, including complete canonical classification and child-only fallback checks. Nine EN/KO/AR desktop/tablet/mobile renders PASS, including direct entries, lower sections, hidden contextual LNB entries, Enter-key mobile settlement expansion, guarded live S01-to-S02 navigation, non-ERP I03 link from I01, B11 tab aliases with exactly one active entry and global My work/job access. Claude CLI visual review found no blocking menu issue in three representative captures; requested report-label and mobile Assistant alignment polish was applied. The opened Chrome deliverable uses the standard static preview on port 8804 with labelled synthetic data; the guarded local-service test remains on port 8803. Static preview does not provide the local-service API. Existing bundle warning remains.

This is a rendered navigation draft. It does not implement new calculation, approval, accounting, evidence-search, collection or authentication capabilities. No owner FIX, main merge, deployment or untested OS support is claimed.
