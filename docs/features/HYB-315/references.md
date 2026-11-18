---
type: ReferenceRecord
title: HYB-315 external references record
description: External references for the P05 screen that root opened on 2026-11-18, with what was observed, adopted and rejected.
status: stable
tags: [HYB-315, P05, references]
sources:
  - resource: plan.md
---

HYB-315 external references, opened by root on 2026-11-18 (WebFetch summaries; what was observed, what is adopted, what is rejected)

1. GOV.UK Design System, Summary list — https://design-system.service.gov.uk/components/summary-list/ — OPENED.
   Observed: rows of key (dt), value (dd) and optional actions; for key facts and metadata, not tabular data; borders on by default, a no-border variant exists for lists without actions, with a caution that borders help users who zoom or use magnification; missing values link to where they are entered. Narrow-screen behaviour not described on the page.
   Adopt: dl key/value shape for facts (p05-sum); no row actions. Reject: "Change" links. Note: keep row separation readable under zoom (spacing or the 1px row border), per the page's caution.

2. GOV.UK Design System, Error summary — https://design-system.service.gov.uk/components/error-summary/ — OPENED.
   Observed: at the top of main (below a back link, above the h1); focus moves to it; each link goes to the field and matches the inline message wording. The page gives no rule for hint placement for single fields (the checkbox example puts the hint after the legend, before the error).
   Adopt: summary placement, focus, link wording. The hint-between-label-and-control order is the project's existing service-field pattern, not a claim from this page. Reject: any coloured accent edge.

3. Material Design 3 list-detail canonical layout — https://m3.material.io/foundations/layout/canonical-layouts/list-detail — NOT AVAILABLE (HTTP 404 on 2026-11-18). Replaced by:
   Android Developers, List-detail layout — https://developer.android.com/develop/ui/compose/layouts/adaptive/list-detail — OPENED.
   Observed: with enough space the detail pane is shown beside the list; with limited space the selected item's detail takes the whole space and only one pane is visible at a time; back in single-pane returns to the list; the selection is kept across configuration changes.
   Adopt: split vs single-pane step by available width, back link to the list, selection kept across the layout change. Reject: animations, elevation, rounded surfaces.

4. Azure DevOps, Requirements traceability — https://learn.microsoft.com/en-us/azure/devops/pipelines/test/requirements-traceability — OPENED.
   Observed: requirements linked to test results; the Requirements quality widget lists requirements in scope with a pass rate and a failed-test count, links the failed count to the tests, and tracks requirements without any associated test.
   Adopt: per-requirement test counts with failures linking to detail, and requirements without tests shown explicitly. Reject: pass-rate percentages (P05 shows counts and "no result" states, never a rate).

5. GitLab, Release evidence — https://docs.gitlab.com/ee/user/project/releases/release_evidence.html — OPENED.
   Observed: a JSON snapshot separate from the release, taken at collection time; a release can have several snapshots; identified by collection timestamp (the page does not mention a digest).
   Adopt: evidence is separate from the release and is a dated snapshot. Correction: the "digest as identity" idea is ours (from the API), not from GitLab. Reject: any release affordance.

6. Grafana k6, Thresholds — https://grafana.com/docs/k6/latest/using-k6/thresholds/ — OPENED.
   Observed: per-metric pass/fail criteria such as p(95) < 200; each threshold reported with a tick or cross; a run passes only if all thresholds pass.
   Adopt: metrics listed with units per run; verdict as words from the server. Reject: tick/cross output and any client-side threshold evaluation (budget approval is not connected).
