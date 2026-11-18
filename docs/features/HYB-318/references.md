---
type: Reference
title: HYB-318 external references
description: External professional references for the O01-to-O02 change plan handoff and unsaved-input handling, opened by root on 2026-11-18, with what was observed, adopted and rejected.
status: final
tags: [HYB-318, O01, O02, references]
sources:
  - resource: visual-spec.md
  - resource: ../HYB-300/references.md
---

# HYB-318 external references (opened by root, 2026-11-18)

## 1. AWS Cloudscape, Unsaved changes

https://cloudscape.design/patterns/general/unsaved-changes/ (opened)

**Observed:**
- Warn only when there are unsaved changes and the user tries an action that would discard them; not when nothing changed or the action opens another tab.
- In-page links, Cancel and breadcrumbs open an in-page "Leave page" modal with "Cancel" (stay) and a primary "Leave".
- Closing, reloading, history navigation and URL edits use the browser's own prompt through `beforeunload`, whose text cannot be customised.
- The example keeps the two mechanisms separate; no "skip in the future" option.

**Adopted:**
- Warn only after a user edit (prefill and base adoption are not edits).
- One app-level guard for in-app navigation and one `beforeunload` for browser exits, never a second page-level prompt.

**Rejected:**
- A modal dialog; the console's existing guard wording ("Discard unsaved input?") is kept.

## 2. AWS Cloudscape, Create resource

https://cloudscape.design/patterns/resource-management/create/ (opened)

**Observed:**
- Resources can be created from multiple entry points, through a single page, a multipage wizard or a modal.
- The page does not describe prefilling from a selected resource or how cancel and return work.

**Adopted:**
- A single-page create flow (O02's targets form) reached from a second entry point (O01's detail panel).

**Rejected:**
- A modal create flow; the handoff navigates to the full O02 screen so its own permission checks and current relationships apply.

Earlier HYB-300 references (MOJ "Filter a list", Cloudscape filter patterns and table view) remain in `../HYB-300/references.md`.
