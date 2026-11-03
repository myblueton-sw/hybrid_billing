---
type: ScreenImplementation
title: Collected FOCUS consolidated cost and usage
description: Connects S04 to the same protected collected source and pinned account scope as the integrated overview.
status: draft
---

# Current implementation

HB-143 adds the default collected S04 screen at `/source-data`. The original authorized planning specification remains the product contract; the [scoped integration design](../../../features/HB-143/collected-source.md) describes this implemented subset. It follows HB-85/PR77 and the separate HB-93 source-inspection planning; it does not certify a merged standalone main checkout.

Normalized rows, mapped allowlisted source fields and native consumption/pricing quantities are connected. Unit prices retain source currency and pricing unit; no implicit denominator or calculated rate is added. Every financial numeric field uses nullable18-place coefficients. Full-scope currency totals and25-row pages come from the protected server query. Transformation history, original-object inspection, export and global service/model/quality/attribution filters remain deferred.

The U01 source-detail link and S04 return link preserve full selection and the loaded snapshot, resetting offsets. Stale evidence fails visibly. Account mappings use complete tenant/provider/billing/subaccount identities; missing native hierarchy is not fabricated. The older `service=local` inspector remains an explicit compatibility deep link to its own separate local source.

# Verification and review

See the [verification/runbook](../../../features/HB-143/verification.md) for actual tests and independent review. Per-screen review is present and pending. No automated owner confirmation, production financial authority or cross-OS acceptance is claimed. Source/review dependencies change the candidate revision of U01 and S04; prior browser acceptance does not carry forward.
