---
type: IntegrationDesign
title: Collected FOCUS overview and account-derived organization sample
description: Replaces generated overview figures with the existing authorized collected FOCUS path and preserves exact costs, account lineage and unavailable operational states.
status: draft
---

# Goal and source choice

The owner requires U01 to run on collected FOCUS data and requires organizations/samples to be grounded in that data. Root owns implementation and verification. Reuse the existing `focus_import`/`hb_focus` collection path, not `focus_local`'s independent synthetic single-account service. Existing six-provider sample lineage distinguishes original public AWS/Microsoft/Google Cloud/Oracle records from explicitly synthetic VMware/Alibaba fixtures. It remains declared FOCUS1.0 and exploratory-query-only, not certified billing input.

At discovery, the original collected stack was stopped while the separate HB141 integration stack owned its old host ports. Preserve both stacks and every retained volume/journal. Bring the original stack up on alternative loopback ports using its existing project/volumes and unchanged image digests. Add trusted startup-only port configuration with original defaults; never accept endpoints/actors from browser input. Do not initialize/reset the collected namespace or bypass integrity failures.

# Owned change surface

Root may edit new overview projection/HTTP/CLI modules, the five local endpoint helpers plus their shared configuration resolver, a portable Compose port override, U01 React data/API/components, scoped browser/backend tests and HB85 documentation. Existing collection authorization, sealed-revision selection, journal/warehouse integrity and native-hierarchy algorithms are reused. No writer delegation; reviewers are read-only. Independent financial/tenant-contract design and change review is required before completion.

# Backend contract

A read-only loopback service binds a verified reader actor at trusted startup. Browser parameters only narrow authorized provider/account/period/basis selections. Host/Origin/Sec-Fetch and custom-client-header protections remain; unsupported methods, identities and unknown parameters fail closed. Vite may proxy only the dedicated overview prefix to the fixed local service, retaining the guarded API boundary.

Use `collection.query(..., _consumer=...)` so authority, sealed-window revision replacement, row/aggregate bounds, graph verification and server-side complete-selection aggregation remain in one protected read context. Do not sum browser pages or combine superseded refreshes. No role, workspace or arbitrary company membership is chosen by browser input.

The projection returns a pinned data/selection identity, explicit charge-start-in-tenant-timezone month attribution, source delivery-window coverage/provenance, distinct billed/effective cost, authorized provider/currency/period options, bounded account search and bounded normalized row/detail data. Paginated reads must pin the same selection; changed source/hierarchy/filter context rejects stale continuation. Protected data is cleared after errors, scope changes or obsolete asynchronous responses.

Money retains Decimal(38,18) coefficients as strings across JSON. No binary-number money arithmetic, implicit cents/KRW scale, FX or sums across financial meanings/currencies. Null/all-missing/known-zero remain distinct. BilledCost is the explicit default source-cost basis; EffectiveCost is a separate option, never customer sales or organizational allocation. Charts must tolerate signed collected values without changing exact totals.

# Organization and sample meaning

Internal company/tenant membership remains server-owned synthetic sample authorization, not a FOCUS-derived legal entity. Display this distinction. FOCUS-derived organization options come from observed provider plus BillingAccountId/SubAccountId and scoped persisted account context. Identity includes provider and tenant scope; names are not keys and equal labels do not merge accounts. Missing subaccount or native ancestry remains absent/unknown, not fabricated OU/project/directory structure. Account results and rendered rows are bounded and searchable.

The existing public sample is reused through its actual collected/published path. No new anonymous numbers are substituted if storage or authority is unavailable. Original source identifiers and secrets are not committed. Sales, allocation, closing/approval/ERP/signed-off receipt operations remain unconnected unless their own authoritative contracts exist. The default U01 route must not silently fall back to the old generated fixture screen.

# Verification and acceptance

Verify retained-stack metadata and actual authorized query before claiming connection. Use isolated tests/mocks or a separately isolated namespace; `tests/focus_import/conftest.py` resets hb_focus and must not run on the retained sample store. Cover wrong-company/actor access, old snapshot/filter continuation, revision replacement through existing selection, precision/null/zero/credits, bounded accounts/rows, field provenance, unavailable upstreams and late-response invalidation. Run the actual collected sample through the browser in EN/KO/AR desktop/mobile, verify exact aggregate read-back and review controls without owner confirmation. Record observed row/provider/account/period counts only after successful reads.

The preceding finance/billing role findings remain open. This integration changes data provenance and organization grounding; it does not by itself establish commercial design acceptance, production identity, source conformance, owner FIX, deployment or merge.

# Execution status

See [verification and runbook](focus-verification.md) for actual retained read-back, separate financial/security change-review passes, frontend corrections and the local storage outage blocking final runtime acceptance. Implementation does not imply completed U01 business fitness or owner FIX.
