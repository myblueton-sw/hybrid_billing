---
type: ScreenImplementation
title: U01 collected FOCUS overview implementation
description: Connects the current overview candidate to collected costs and observed account mapping while preserving deferred operational requirements.
status: draft
---

# Current candidate

The [U01 specification](spec.md) remains the planning contract. The [current composition](../../../features/HB-85/commercial-overview.md) uses collected FOCUS costs, actual provider/period options, signed monthly history, observed billing/subaccount mapping and bounded source details. Canonical route is `/home`; source is `apps/console/src/modules/operations/pages/overview`.

The [integration design](../../../features/HB-85/collected-focus.md) and [verification/runbook](../../../features/HB-85/focus-verification.md) define the protected read path, sample provenance, exact money and startup commands. No generated A/B/C organization or cost fallback is used by this screen. Existing sales, allocation, closing, approval, ERP and receivable integration gaps remain visible. This is not completion of all U01 planning requirements.

# Review and installation boundary

Per-screen review remains visible and pending. Browser-local confirmation is not authenticated canonical owner acceptance. Source/selection identity, visible page offsets and chart currency participate in review context. No owner FIX, merge or deployment is granted here.

Use portable Python/Node entry points in the runbook. Existing authorized bindings, sealed sample collection and retained storage are prerequisites; a fresh clone must obtain the documented companion inputs. macOS arm64/Chromium execution does not certify other operating systems or browsers. The canonical design/build guides remain in the authorized companion workspace; this ticket does not replace them with partial guides.
