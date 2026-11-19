---
type: ImplementationStatus
title: HYB-310 implementation candidate before combined verification
description: Records the resumed permission-split fixes and deferred combined verification without replacing the original implementation evidence.
status: draft
sources:
  - resource: ./plan.md
  - resource: ./verification.md
  - resource: ./visual-spec.md
---

# Resumed implementation

Root (Codex) resumed the existing worktree on machine `mac_mini`. The owner directed completion of HYB-310, HYB-333 and HYB-306 before one combined verification pass; no new implementation ticket is authorized.

Previous fixes for redelivery screening, paging, automatic closure attribution, failed reload copy and lost-right cleanup remain in the candidate. The resumed independent code review found that author-only withdrawal accepted another active client with the same direct grant, contrary to the console-only plan. A new regression case demonstrated the failure before the fix. Independent design review approved an immutable configured author-client boundary. `auth.Catalog.AuthorClient` is now copied privately at construction; empty disables author-only calls, invalid nonempty IDs refuse construction, and the API sets `ConsoleClient`. Both initial and commit-time checks retain the active-client and direct-grant requirement. Ordinary permission decisions remain separate. Change re-review and full regression checks of this fix are pending.

The Arabic Held glossary entry now matches `محتجَز`, distinct from Pending. The active visual specification now uses the accepted locale separators. Separate-agent QA/TW inspected inherited screenshots and found those render fixes sound, but this does not certify a fresh render, native-language approval or owner acceptance.

# Evidence and pending work

- Before the author-client fix, resumed console tests: 800 passed; console build: exit 0. These do not verify the later authorization fix.
- New `TestAuthorRefusesAnotherClientWithTheSameGrant`: failed before the fix because another granted client was allowed.
- `go build -o /private/tmp/hyb310-api ./apps/api/cmd/hb-api`: exit 0 after the author-client fix. Its new test cases remain unexecuted pending the combined pass.
- Original detailed test and browser evidence remains in `verification.md`; it is historical, not proof for the latest diff.
- Claude CLI independent review was attempted and returned a usage-limit response; no successful Claude review is claimed. The independent agent reviewed source and the proposed repair separately.
- The owner deferred remaining tests, mutation/race checks, fresh render QA, copy checks and independent change re-review to the combined pass after all three implementations. No PR, merge or completion is claimed.


# Combined verification update

The owner resumed verification after the three implementations. See [verification](verification.md) for actual passing checks, reproduced defects and repairs, independent review, coverage and remaining limits. This earlier implementation-status record is preserved as handoff evidence; its pending-check statements describe the earlier candidate, not the current verification result. No owner acceptance or merged integration is inferred.
