---
type: VerificationRecord
title: HB-272 verification of the O01 organization and relationship read API
description: Module and apps/api tests, coverage, race, cross builds, mutation results and review outcomes for the party, billing-profile and timeline listings, the synthetic organizations seed and the organizations read. Run on mac_mini on 2026-11-16.
status: final
---

# Scope

This record verifies [HB-272](https://linear.app/hybrid-billing/issue/HB-272) against its [plan](plan.md). Every check was run on mac_mini (macOS) with Go 1.25.7, using synthetic data only.

# Results

| Check | Result |
| --- | --- |
| `gofmt -l` and `go vet` (the changed packages) | clean |
| `go test -race ./server` (apps/api) | pass, 100.0% statement coverage |
| modules/organizations and modules/relationships tests with `-race` | pass. `listing.go` is 100% covered in both modules. The package totals (94.0% and 98.9%) include statements that were already uncovered on main. |
| `-race -count=40` on the new tests in all three packages | pass |
| `GOOS=linux GOARCH=386` and `GOOS=windows` builds | pass |
| Mutation testing of the four changed source files (96 mutants) | 88 killed and 8 survived. One survivor (`n < 1` changed to `n <= 1`) was then killed by an added limit=1 test, confirmed by re-running that mutant. The other 7 are equivalent. |

**Why the remaining 7 survivors are equivalent**
- **Parse errors (3 mutants).** Removing the error check on date, limit or revision parsing changes nothing. A failed parse leaves a zero value whose round-trip text cannot equal the raw input, so the request is still refused.
- **Sort comparisons (3 mutants).** Changing `<` to `<=` in the ID sorts changes nothing, because IDs are unique within a workspace.
- **Error check on limit parsing (1 mutant).** A failed parse yields 0, which the `n < 1` check already refuses.

**The tests cover**
- **Listings:**
  - order;
  - copies, including entries not shared with the registry;
  - workspace isolation with two populated workspaces;
  - unknown workspaces;
  - the empty workspace at revision 0;
  - paging bounds and cursors;
  - unknown kinds and revisions;
  - ISO dates.
- **The read:**
  - the response shape;
  - the move boundary (the day before and the day of the move) and the day before the start;
  - both view groups for the shared account;
  - every kind;
  - the default day;
  - a pinned revision 0;
  - paging through every member, an exact page, limit 1 and limit 99;
  - invalid day, kind, limit, cursor and revision values;
  - the fault path;
  - a member refused, and no session;
  - an empty install;
  - another workspace;
  - an unknown or invalid workspace, including with a pinned revision;
  - the 500-row cap.
- **The seed:**
  - each unit's hierarchy matches `Party.ParentID`;
  - each external profile's payer has paying accounts.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only, tenant-boundary and data-accuracy focus):
  - Round 1: FAIL, with four Medium findings: view-group targets, the exclusive end date, the empty install, and the workspace source.
  - Round 2: PASS.
- **Change review:** PASS, with six Low findings. L2–L4 were fixed; L1, L5 and L6 are recorded in the plan.

No review is owner acceptance.

# Residuals

These are listed in the plan:
- the read is admin-only in pre-launch;
- names come from a seed label map;
- there is no lazy tree, only a flat hierarchy listing;
- each page scans and sorts that kind's timelines;
- a member appears only under kinds where it has a timeline;
- the sample data is deliberately cross-customer;
- the workspace-isolation test calls the read directly.
