---
type: PrototypeDesign
title: Scalable multilingual console redesign
description: Interactive full-inventory design draft with bounded large-data navigation and multilingual review states.
status: draft
---

# Scalable console design

HB-63; root owns shell, state/data adapter, styling, checks and integration. The isolated screen_catalog implementer owns catalog.js only. Independent interface_audit reviews architecture/changes. This module is an interactive design prototype, not a production React application or backend service. It preserves stable v10 screen identities and maps the 49-screen registry; existing accepted v10 remains active until an accepted replacement is explicitly fixed. The new draft does not supersede unreplaced module requirements.

## Design contract

- English/Arabic priority with Korean first-class; locale copy is a review candidate, not certified practitioner translation. HB-61 terminology rules at commit 2694eab apply to this draft even though PR57 remains separate.
- TNB for domains, searchable secondary screen navigation and an on-demand context drawer. UAE-informed 16px base typography, 24px desktop gaps, breakpoints 640/768/1024/1280/1536 and logical RTL layout. Brand navy/teal is a deliberate product variation from federal identity colors. Local fallback fonts remain possible; no compliance claim without rendered evidence.
- Logical synthetic populations of 100,000 accounts and 1,000,000 file records; generate bounded pages, not a million DOM rows. Production requires server-side indexed search and cursor pagination; this prototype supplies an isolated asynchronous adapter, not a performance benchmark of a backend.
- Cursor and selection bind scope, screen, query, provider, status, sort and snapshot. Page size/sort invalidate cursors; filter/scope changes clear selection/preview. Exact totals only when known; filtered totals can be unknown. No fabricated last page.
- Page selection and all-filtered selection are distinct. All-filtered stores a query descriptor and exclusion IDs, never the entire match set. Preview displays scope and count certainty; expired snapshots block submission. No money calculation, authorization or issuance is performed.
- Account picker is a bounded searchable modal list with keyboard movement, explicit selection and focus return. Search and detail views use the same demo scope.
- Large-file intake inspects metadata only; no content read or upload. Retained queue capped at 100 items, displayed page capped at 25. Simulated processing has bounded concurrency; cancel and retry operate on specific eligible states.
- Product actions are simulated, with explicit persistent draft labeling. Jobs are in memory and disappear on reload; no persistent scheduler or production success is implied. Double submit must not duplicate the same demo operation.
- Native tables, sticky headers, constrained horizontal table scrolling, labels/focus and bounded detail panels. Customer screens use a separate safe synthetic projection; hiding columns does not establish authorization.

## Source and acceptance basis

Frozen v10: docs/design-baseline-v10/registry.json, 89 page IDs. Additional baseline: docs/ui/screen-registry.json, 49 IDs. Onboarding r2 retains provider-specific requirements. NAVER Cloud and KT Cloud appear as required profiles with live capability unverified, not connected providers. HB-62 source audit at commit 837bf50 establishes the missing production API boundary.

Official design references: [UAE layout](https://designsystem.gov.ae/guidelines/layout), [UAE typography](https://designsystem.gov.ae/guidelines/typography), [W3C combobox](https://www.w3.org/WAI/ARIA/apg/patterns/combobox/), [W3C listbox](https://www.w3.org/WAI/ARIA/apg/patterns/listbox/). These references guide the prototype; their existence does not certify accessibility or local terminology.

Independent design review: PASS with required scope/snapshot invalidation, query-descriptor selection, stale-response suppression, bounded metadata queues, honest synthetic jobs, safe customer projections and keyboard focus handling. Actual-change review and rendered QA will be recorded in verification.md.

## Run

Serve this directory locally with `python3 -m http.server 8763 --bind 127.0.0.1`, then open http://127.0.0.1:8763. No build or external service is required. Source modules are dependency-free design code so reviewers can inspect interactions independently of the pending production React/API integration.

## Review and implementation boundary

Open `?lang=en#operations`, `?lang=ar#operations` or `?lang=ko#operations`; `#contents` exposes the entire catalog. Six TNB domains organize the eleven catalog groups, with searchable secondary navigation and bounded detail dialogs. Browser refresh resets drafts, file descriptors and jobs. Local large-file examples are metadata scenarios only.

[Verification](verification.md) records independent findings, fixes, browser evidence and remaining acceptance. The 111-route coverage is navigation and common-interaction coverage. It does not establish complete business behavior for every form/detail/workflow. The existing v10 baseline is preserved until the owner accepts a replacement scope.
