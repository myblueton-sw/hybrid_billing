---
type: VerificationRecord
title: Admin status projection verification
description: Records what was executed and reviewed for the first increment of work item W-30, a status projection that ingests stage facts from work results and answers list, detail and total queries from one encrypted as-of, applying the viewer's scope first, keeping stale, unknown and zero apart, never adding different meanings or currencies, and keeping a per-subject timeline. The record lists results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-233](https://linear.app/hybrid-billing/issue/HB-233) adds the Go package `modules/overview` from main `397083a`.

- **Facts.** System actors ingest facts per customer, contract, entity, period and currency with stage, due date, owner, next action, amounts by meaning (cost, sales, allocation, outstanding), evidence references, occurrence time and source revision; receipt time is set by the register. Repeats are idempotent; each source revision has one fact.
- **Combination.** A source's current fact is its highest revision workspace-wide, so corrections move it between subjects and late lower revisions never roll back. `blocked` wins while any current source reports it; otherwise the latest occurrence leads. At most one source reports a meaning per subject at any as-of; facts that would break this are refused only when they would become current.
- **Queries.** Encrypted, authenticated, fixed-width as-of tokens fix the version and the evaluation instant; list, detail and totals use the same as-of; no sequence numbers are exposed. The viewer's scope (customer, entity, contract, period-range selectors and cost visibility) is applied before any count, total, sort or page; hidden and missing subjects answer alike. Filters, sorts (due with missing dates last, customer, period, receipt time for recent work), cursors bound to the as-of, viewer, resolved scope and filter. Totals per currency and meaning keep current and stale amounts apart, count stale and unknown rows, and leave unknown amounts empty.
- **Detail and saved filters.** Detail returns the subject's state and its own timeline with occurrence and receipt times, revisions, evidence and as-of-relative supersession; absent subjects show an empty state. Saved filters live in per-human namespaces; shared ones cannot name identifiers and always run in the viewer's scope.

# Commands and results

All runs were made in the worktree `/private/tmp/hb233` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./modules/overview` | pass |
| `go test -race -short -count=10 ./modules/overview` | pass |
| `go test -race -count=1 ./modules/overview` (includes the 100,000-subject test) | pass |
| `GOOS=linux GOARCH=386 go test -c ./modules/overview` | builds |
| `go test -race -count=1 ./...` | 46 packages, all pass |
| Coverage | `modules/overview` 100.0% |
| Mutation testing (with `-short`) | 489 mutants of `registry.go`, `query.go` and `api.go`: 433 killed, 37 did not build, 19 survived |

The 19 survivors were read and judged equivalent: `Round(1)` on clocks and occurrences; `[][3]` pair arrays; date and period parsers with `||` (a failed parse never formats back to the refused text); token checks whose alternatives are refused by authentication or field counts; page bounds at `len(rows)` that slice the same rows; `<=` in comparators reached only for distinct keys or after equality was handled; `>=` on source revisions, which are unique per source; a nonce-length bound that authentication refuses anyway; and the sequence tie-break between a fact and itself.

The 100,000-subject test pages 100 cursors of 1,000 rows, checks that every subject appears once, and compares both currency totals with the summed rows and with an independently computed integer sum. A property test over 40 seeds predicts every ingest refusal and compares stage, due, owner, next action, receipt time, status and amounts at random earlier as-ofs with an independent model. Each query scans the workspace in memory (about 0.13 s for 100,000 subjects on this machine); persistent indexes are a residual.

# Reviews

- Design review: three rounds by a separate read-only reviewer; FAIL, FAIL, PASS ([plan](plan.md)).
- Change review: one round by a fresh read-only reviewer; PASS with low findings, all applied.

# Provisional choices for the owner

- Stage names, amount meanings, the `blocked` precedence and the one-source-per-meaning rule.
- Limits: 1,000,000 subjects and 10,000,000 facts per workspace, pages up to 1,000 rows, staleness 1 minute to 90 days.
- Shared saved filters may not name customers, contracts, entities or owners.

# Not verified and residuals

Notification incidents (QA-08), closing cycles (QA-03), bulk commands with partial-success display, the U01 payment-term view from `modules/receivables`, free-text search, void facts, the `modules/auth` scope adapter (it runs under the registry mutex and must not block or call back), adapters from module events, persistent indexes, screens. Tokens stop working when the registry restarts.
