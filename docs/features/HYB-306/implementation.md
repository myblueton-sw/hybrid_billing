---
type: ImplementationStatus
title: HYB-306 implementation candidate before combined verification
description: Records the resumed persistent own-access history implementation and the build evidence while the owner defers verification until all three in-flight implementations are complete.
status: draft
sources:
  - resource: ./plan.md
  - resource: ./visual-spec.md
---

# Implementation candidate

Root resumed `feat/HYB-306-m05-access-history` on machine `mac_mini`. The owner directed completion of HYB-310, HYB-333 and HYB-306 before one combined verification pass, without starting a new implementation ticket.

The candidate includes the access-history storage module, asynchronous recording, durable user-incarnation markers, bounded own-workspace queries, session-end recording hooks, retention settings and one-time operator flags, legal-hold markers, clock and size safeguards, OpenAPI, the client parser, and the EN/KO/AR M05 history region with filtering and Show more. The region remounts across user/workspace/session changes, invalidates late reads on cleanup, and clears history on an unauthenticated answer. The screen register, field map and candidate glossary are synchronized.

# Build evidence

- `go build -o /private/tmp/hyb306-api ./apps/api/cmd/hb-api`: exit 0 on macOS arm64 before the final console scope-cleanup change.
- `npm run build` in `apps/console`: exit 0 after the final scope-cleanup and history-model test additions, including TypeScript. Existing large-chunk warning remains.
- These are compilation results only; tests, fault injection, mutation, race, rendered layout, native-language and independent implementation review remain pending.

# Pending combined verification

Use the stable plan's scenarios for restart, failure and crash recovery, shortened retention, time jumps, holds, concurrent incarnation/compaction, scope isolation, non-disclosing refused login, paging and all session-end sources. Verify EN/KO/AR at the four widths, keyboard behavior, async stale-response rejection, and affected M05/P04/sign-in regressions. Reconcile OpenAPI goldens and execute the required independent security/data and change reviews before integration. Keep all plan residuals visible. No PR, merge, owner FIX or ticket completion has occurred.


# Combined verification update

The owner resumed verification after the three implementations. See [verification](verification.md) for actual passing checks, reproduced defects and repairs, independent review, coverage and remaining limits. This earlier implementation-status record is preserved as handoff evidence; its pending-check statements describe the earlier candidate, not the current verification result. No owner acceptance or merged integration is inferred.
