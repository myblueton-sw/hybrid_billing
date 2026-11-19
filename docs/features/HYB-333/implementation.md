---
type: ImplementationStatus
title: HYB-333 implementation candidate before combined verification
description: Records the resumed credential-pickup and unknown-outcome implementation with the actual build results and the checks deferred by the owner.
status: draft
sources:
  - resource: ./plan.md
  - resource: ./visual-spec.md
---

# Implementation candidate

Root resumed the existing `fix/HYB-333-m04-unknown-issue` worktree on machine `mac_mini`. The owner directed that HYB-310, HYB-333 and HYB-306 implementations be completed before one combined verification pass; no new implementation ticket is authorized.

The candidate includes server pickup state and bounded collection bookkeeping, read-only identity admission for pickup estimates, additive services-read metadata and replay window, the user/service issue-attempt store, locked held forms, issue and never-collected reminders, deliberate-leave handling, and revoke focus return. Create requests keep unknown outcomes and reuse their frozen attempt. The screen register, field map and candidate glossary entries are synchronized.

# Build evidence

- `go build -o /private/tmp/hyb333-api ./apps/api/cmd/hb-api`: exit 0, macOS arm64.
- `npm run build` in `apps/console`: exit 0, including TypeScript. Existing large-chunk warning remains.
- These are compilation results only. No resumed test, mutation, race, browser, accessibility, native-language or independent change review is claimed.

# Pending combined verification

Run the plan's scoped tests, error/retry and secret-exposure cases, mutation and race checks, affected-screen regressions, EN/KO/AR at the four widths, technical-writing and independent security/change review. Fix findings before PR integration. Keep the plan residuals and owner-only screen acceptance separate. No PR, merge or ticket completion has occurred.


# Combined verification update

The owner resumed verification after the three implementations. See [verification](verification.md) for actual passing checks, reproduced defects and repairs, independent review, coverage and remaining limits. This earlier implementation-status record is preserved as handoff evidence; its pending-check statements describe the earlier candidate, not the current verification result. No owner acceptance or merged integration is inferred.
