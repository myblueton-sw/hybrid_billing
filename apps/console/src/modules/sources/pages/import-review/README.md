---
type: ScreenImplementation
title: S03 retained collection inspection
description: Describes current authorized stored-row observations and separate whole-file receipt metadata with their failure boundaries.
status: draft
---

# Runtime and scope

The default `/imports` screen reads retained public/synthetic sample evidence. `/api/focus-receipts` supplies unchanged, wholly attributable metadata for the four existing tabs. `/api/focus-collection-status` separately reads and verifies authorized stored slices, then returns only bounded counts, provider/source-window breakdowns and provenance. No intake/replay/publication or financial action is implemented here.

Receipt-prefix search affects receipt results only. Current source counts cover all permitted contributing rows, including permitted slices of mixed originals whose whole-file metadata cannot be displayed. Empty authorized rows are exact zero; upstream completeness and original-file availability remain unverified. Source windows are not charge-start months. The S04 link passes only language and lets the destination reauthorize.

Each async status request is bound to its receipt observation identity and exact scope/permission pins. Ordinary stored-read failures remove source values while retaining separately valid metadata. HTTP403, authority mismatch or revocation clears metadata, selection and source together. Obsolete requests are cancelled and late results rejected. Both observations display their own timestamp/snapshot.

# Presentation and verification

Use full-width neutral sections, canonical assets/local fonts, 16px data/controls, 44px targets and responsive receipt cards below720px. Full-precision receipt timestamps remain in detail/time attributes; list timestamps may show seconds and midnight source boundaries as dates. Stable S03 items include the prior receipt fields plus stored rows, provenance, source windows, summary async state and source navigation. Shared screen review remains revision-bound and owner-only.

See [HB-150 plan](../../../../../../../docs/features/HB-150/plan.md) and the ticket's verification record for actual tests, rendered review and limits. Product dependencies and supported runtime remain in the console manifest; no global formatter/browser dependency is required by the product.
