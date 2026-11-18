---
type: VerificationRecord
title: HYB-299 M01 role periods and bulk revocation, implementation verification
description: What was built against the approved plan (design rounds 1-5 and the root amendment), the review fixes of 2026-11-18 (security, render and technical writing), the checks run with their observed results, browser QA, regressions, mutation, deviations, plan-text changes for root and residuals. Written by the implementer; the reviews have not re-reviewed this build.
status: stable
tags: [HYB-299, verification, M01, role-period, bulk-revocation]
sources:
  - resource: plan.md
  - resource: visual-spec.md
  - resource: references.md
  - resource: ../HYB-297/verification.md
  - resource: ../HYB-298/verification.md
---

# Scope of this record

- Branch `feat/HYB-299-role-validity-bulk-revocation`, worktree `/private/tmp/hb299`, first built on main `8d65a5f`, rebased by root onto main `df2f36c` (HYB-300 Part A) before the review fixes. Nothing is committed; root owns commit, review and merge.
- Every result below was observed in this worktree on 2026-11-18 on the Mac mini (darwin/arm64). Commands ran with the session `GOCACHE`. Where a check was not run, it says so.
- The three independent reviews (security change review, `ui-designer` render review, technical-writing review) ran on the first build and failed; every listed fix is applied below ("Review fixes"). They have not re-reviewed this build.

# Rebase deltas (plan written on `07458be` plus in-flight worktrees)

None changes a behaviour or a contract the plan relies on, except item 1, which needed an addition (see Deviations D6).

1. **The sample seed member is invited, not a member, at every start** (`seed.go` `seedMember` only invites; the membership is created when the person first sets a password or signs in). Plan assumption A1 ("seed user IDs and their memberships are recreated identically at every start") is false for `sample-member`. A replayed membership record for a person whose membership does not exist yet is therefore applied at acceptance (D6).
2. `gate.InstallationAction` / `auth.InstallationAction` (HYB-297) gained `revoke_access_in_bulk`; the M03 single-action guards pick it up unchanged.
3. Command count: 73 on the base, **78** with the five new IDs (plan text says 77).
4. HYB-292 limits: burst 60, refill 1 s. A 5,000-person draft is 27 staging batches of at most 4,096 bytes, plus prepare and execute: 29 commands, inside the burst. The staging transport does not depend on more than that rate; a `rate_limited` answer waits for the server's hint and resumes.
5. HYB-297 `admin3` holds `administer_installation`, not the bulk right (the plan grants the bulk right to `sample-admin` and `sample-admin-2` only).
6. HYB-298 replay iterated every decision as a grant or delegation revocation; it now skips the identity kinds (`decisionledger.IdentityKind`).

# What was built (plan items)

| Plan item | Status | Where |
| --- | --- | --- |
| Role period on every membership; time-aware `effective` predicate in every membership check; `MemberInfo.Effective`; source test on `MembershipActive` | done | `modules/identity/registry.go`, `workspace.go`, `boundary.go`, `periods.go`, `source_test.go` |
| `SetRolePeriod`, `ChangeRoleUntil`, `ExpireRoles` (blocked defence), `ApplyRecorded`, `Check*` dry-runs, batch suspensions, operator `Members` read | done (`CheckSuspend*` not added, D2) | `modules/identity/periods.go`, `checks.go`, `bulk.go` |
| `revoke_access_in_bulk` installation action | done | `modules/auth/types.go`, `modules/gate/types.go` |
| Drafts, manifest, digest `bulk_revocation_v1`, equal-manifest reuse, execute, outcomes, job state mapping, pruning, replay views | done | new `modules/bulkrevoke` |
| Ledger: nine new kinds, per-kind validation, format v2, `ErrFormatNewer`, `CommitBatch`, raw pre-scan `Scan`, `Stop` | done | `modules/decisionledger/identity.go`, `batch.go`, `prescan.go`, `record.go`, `ledger.go` |
| Server clock: base plus monotonic elapsed time, forward-only catch-up, wall lead beyond tolerance refuses increases, `-clock-base` with pre-scan and verified-choice rule, `clock_base` record | done | `apps/api/server/clock.go`, `clockbase.go`, `clockrows.go` |
| `identity.set_role_period`; `role_until` on create, invite, change role; ceiling, bounds, omitted means no end; `periodIncrease`; anchor rule | done | `periods.go`, `admin.go` |
| Increases: dry-run, intent, apply (same instant), abort, confirm, `durable`; defect stops the ledger | done | `increases.go` |
| Sweep at Start, top of `s.execute`, `signIn`, `setPassword` (redeem), 30 s ticker | done | `periods.go`, `commands.go`, `signin.go`, `bulkcleanup.go` |
| Bulk commands `add_targets`, `prepare`, `execute`, `discard`; ownership in `Resolve`; impact preview with eight proposal stores | done | `bulkstage.go`, `bulk.go`, `bulkimpact.go` |
| Phase A deny, phase B header then 256-record chunks, phase C cleanup on `modules/jobs` (100 items per task, lease 30 s, life 1 h, 3 attempts, 1-10 s) | done | `bulk.go`, `bulkcleanup.go` |
| Reads: users read additions; `/identity/bulk-revocations`, `/{id}`, `/{id}/items`, `/{id}/targets` (root amendment) | done | `users.go`, `bulkread.go` |
| Identity replay: decreases always, increases only with their confirm, decreases only in degraded replay, role guard, anchor-keeping skip, operator-recovery state, job rebuild from headers, counters | done | `identityledger.go` |
| Trust hold rows for the five commands; direction from the stored end | done | `trusthold.go`, `periods.go` |
| Seven installation settings and `-clock-base` | done | `settings.go`, `cmd/hb-api/main.go` |
| Console: selection, bulk flow (effect, staging, preview, confirm dialog), Revocation jobs tab, role period cell/fact/form, Role ends on create/invite/change role, conditions, settings disclosure | done | `apps/console/src/modules/identity/pages/users/*`, `service/identity-bulk.ts` |
| Staging failure keeps the draft; resume (same idempotency keys) or discard (root amendment, R-5) | done | `BulkPrepare.tsx`, `identity-bulk.ts` |
| M01 AR copy corrections and `(open)` instead of `(selected)` | done | `live-copy.ts` |
| Glossary rows | done (10 rows) | `docs/ui/terminology/product-glossary.md` |

# Review fixes, round 4 (final render recheck)

- Medium: in the narrow "Role and period" cell of the Users tab (a row open), the revocation sub-line ("Suspended by bulk revocation n") and "Installation administrator" now appear as on the wide layout and the Membership tab. QA A and L assert the sub-line for the bulk-revoked person (admin2 in A, admin3 in L) in users-open at 1440 and 1280 in EN/KO/AR.
- Low: fact lists (`.m01-sum`) capped at 68ch so labels sit near values; 24 px after the people-to-revoke section; the expired notice's first sentence in 600 (`Emph`).
- Checks: tsc clean, npm 710/710, build ok; QA A 334/0, L 352/0, E 53/0.

# Review fixes, round 3 (rechecks of 2026-11-18)

Root rebased the tree onto main `1cf7f3f` (HYB-318) and amended `plan.md` with D6 and F1-F4. The rechecks failed on one render high, one security medium and one TW truthfulness item; each is fixed here.

| Item | Change | Evidence |
| --- | --- | --- |
| Render high: headers did not match cells | The fold rules hid or showed `td.m01-l`/`td.m01-m` only; the same classes sit on the `th` cells (`Live.tsx` `folded`). Every fold selector in `bulk.css` now covers `th` too, inside the container query and the 767 px block | New QA check: in every rendered table, the visible header count equals the visible cell count of each row, and each header overlaps its cell. It ran first on the unfixed build: 24 failures (Users and Membership, EN/KO/AR, 1440 and 1280, with and without a row open: "7 headers / 6 cells", "7 headers / 4 cells") |
| M4 option hints | The shared `.live-roles label { gap: 12px }` made a 12 px row gap inside the grid label; `gap: 0` on the effect labels, hint `margin-top: 2px`, 16 px between options | New QA check: each hint sits at most 8 px below its option and closer to it than to the next option. Unfixed build: 9 failures ("14px below its option, 12px above the next") |
| Staging failure | While a draft is incomplete only "Resume sending" and "Discard the draft" are offered; "Preview impact" is hidden | QA E "no Preview impact primary while the draft is incomplete" |
| Not-in-loaded "Remove" | Flex row with a 12 px gap | QA N screenshots |
| Expired preview (spec 5.1) | "Continue" re-reads the list first; if it has expired, the preview shows the expired alert and "Prepare again" instead of the confirmation | QA E "an expired preview offers Prepare again", "Prepare again returns to the effect step" |
| N1 (security medium) | `change_role` with `role_until` omitted keeps the stored end only within the setter's own end: if the setter has an end and the kept end is none or later, the answer is `invalid_request` on `role_until` | `TestAnOmittedEndStillHonoursTheSettersEnd` fails without the change (checked) and kills both operator mutants on the new line |
| Duplicate anchor check | The `periodAnchors` call in `changeRole` repeated `protect` (same anchor set minus the target); removed. `TestChangeRoleKeepsAnAnchorAndAllHeldListsDenyNobody` still refuses through `protect` | Go tests |
| `?kind=` | Values other than empty or `excluded` answer `invalid_request` | `TestTargetsReadRefusesAnUnknownKind` (fails with the check removed) |
| R1 (TW truthfulness) | Non-owners no longer fetch the targets; the preview shows "Only the person who prepared this list can see the people on it." (KO, AR as proposed) instead of "People to revoke (0)" | `preview-render.test.tsx` (EN/KO/AR, owner and non-owner); QA E "R1 another administrator sees no false count of people" |
| TW minors | AR `blocked` uses المستهدفين; glossary row for the AR target word; AR excluded reads "مستثنى (غير محسوب)" in the label, heading, list count and item outcome; unused `b.selected` and `b.executedL` deleted; `retry_budget_spent` is a real server reason (`bulkcleanup.go`, final cleanup failure), so its mapping stays | `bulk-copy.ts`, glossary |

Checks after round 3: gofmt clean; vet ok (linux, windows, darwin); cross builds ok; `go test ./...` ok; coverage `apps/api/server`, `modules/bulkrevoke`, `modules/decisionledger` 100.0%; `-race` on `apps/api/server` ok; tsc clean; `npm test` 710/710 (new `preview-render.test.tsx`, registered); build ok. Mutation on the newly changed Go lines: the two operators on the N1 line (`&&`, `||`) killed by `TestAnOmittedEndStillHonoursTheSettersEnd`; removing the `kind` check is killed by `TestTargetsReadRefusesAnUnknownKind`; removing the duplicate anchor call leaves every test passing through `protect`.

Browser QA after round 3 (final console build): A 328/0 (now with the header/cell check on Users and Membership with and without a row open at 1440 and 1280, and the hint check at every width), C 3/0, B 5/0, D 5/0, N 31/0, E 53/0 (rerun after the user column was widened from 36% to 42% in the folded layout so logins no longer break mid-word with a row open, and after the R1 check was narrowed to the heading "People to revoke (n)" — the counts block keeps a "People to revoke" label), L 346/0 on the same build. Regressions after round 3 (`reground299.sh`): M01 52/0; M03 A 67, B 21, C 19, D 30, E 20 (0 failed); M02 all groups 0 failed; HYB-293 B 32, C 27, D 54, A 21 (0 failed); P01 46/0; P02, P03, P04, P05 every group 0 failed (same counts as before). The console build changed once during the chain (user column width only). Opened and checked: `users-open-*`, `membership-open-*` (EN/KO/AR 1280/1440), `not-owner-en-1440`, `preview-resumed-en-1440`, `jobs-list-*`, `dialog-*`, `isolated-job-en-1440` (reason now "The ledger is isolated"), `expired-preview-en-1440` ("This preview expired. Prepare it again." with "Prepare again").


Three independent reviews ran on the first build (render: FAIL with 3 high, 10 medium and 6 low findings; technical writing: FAIL; security: FAIL on one medium). Root rebased the work onto main `df2f36c` (HYB-300 Part A) and listed every fix. Each item below was applied in this worktree; the tests and QA checks named are the evidence.

## Security

| Item | Change | Evidence |
| --- | --- | --- |
| F1 `-clock-base` stayed usable after one use | The flag is accepted without a lead only provisionally while the latest `clock_base` choice is unverified; `settleClockBase` refuses with `ErrConfig` (and writes nothing) when that choice is verified. The D9 exception therefore covers only an unverified choice. Help text in `hb-api/main.go` aligned | `clock_test.go` (inverted: a flag left on after a verified choice is refused, no record written); the unverified-choice path keeps its existing test |
| F2 shortening classified as an increase | `change_role` is an increase only for a promotion to administrator or a later/removed end; the same role with a shorter end is a decrease (`role_period_set`), allowed during a trust hold and a degraded ledger; the hold rule reads the same direction; the anchor rule applies when an administrator's end is shortened through `change_role`; the live-link prerequisite applies to promotions only | `TestChangeRoleClassifiesByDirection` (no intent record, shortening with the ledger stopped, lengthening refused, degraded replay keeps the shorter end), `TestHoldDecidesThePeriodDirection` (hold: demotion keeping the end and a shorter end allowed, a later end, a removed end and a promotion refused), `TestChangeRoleKeepsAnAnchorAndAllHeldListsDenyNobody` |
| F3 anyone with the rights could run or discard another person's prepared list | `execute` and `discard` of a prepared list are owner-only; another holder gets the same answer as an unknown ID (`permission_denied`); counts stay readable, targets and exclusions do not. The console shows neither action to non-owners and says that the list expires otherwise. This is the default answer to owner question Q3 | `TestPreparedListsAreTheOwnersToRunOrDiscard`, `TestBulkSuspendUserAndHeldTargets` (another holder cannot run someone else's list); QA E "F3 another administrator sees no run or discard" |
| F4 omitted `role_until` on `change_role` | An omitted `role_until` keeps the stored end; `end=none` (what the console sends) removes it; `end=at` needs a time and `end=none` refuses one. Create and invite keep "omitted means no end, within the setter's cap" | `TestChangeRoleKeepsAnOmittedEnd`, `TestRolePeriodCeilingAndOmissions` |

## Render

| Item | Change | Evidence |
| --- | --- | --- |
| H1 wide and narrow cells both shown at 390 | Hiding rules raised above `live.css` (`.live-table td.m01-l`, `td.m01-m`, `td .live-sub.m01-fold`), also inside the 767 px block | QA check (a) on every users screen |
| H2 AR dates scrambled | One `Stamp` everywhere (job facts, preview, dialog, limits, period cell, notices): date and time each in their own `bdi`, zone in an LTR `bdi`, Arabic comma in AR; date-bearing sentences use `{at}` templates filled with the stamp; "Loaded at" and the time-zone hint isolate their time and zone too. Root cause of the scrambling: the shared `bdi { direction: ltr }` rule (`shared/styles.css:187`) forces every isolate left-to-right, and an Arabic `Intl` date carries right-to-left marks, so an isolated date still rendered as "182026/11/". Date and time parts (`bdi.m01-part` without `dir="ltr"`) now get `direction: rtl` in right-to-left documents; the "Records" label is kept outside the LTR sequence range | QA check (b) on every AR screen, which also requires the rendered direction of each date and time isolate to be right-to-left; AR screenshots at 1440/1280/390 |
| H3 job name ran into the next word | Effect and origin are block lines under the opener; the "(open)" tag has 8 px on both sides | QA check (c) on the jobs list and detail |
| M1 select-loaded visible at ≥768 | `.m01-selbar label.m01-select-loaded` beats `label.live-check`; shown only below 768 | QA users screens at 1440/1280 (no repeated block) and 390 |
| M2 "Password set" twice | `.live-table td .live-sub.m01-fold` hidden at wide widths | QA check (a) |
| M3 disclosure markers | Summaries are `display: list-item` with a 44 px target from padding; no glyph | screenshots |
| M4 option hints | `margin: 2px 0 0` on hints, 12 px between options | screenshots |
| M5 `::first-line` | Removed; the first sentence is wrapped in `strong` (`Emph`), and the scope sentence, "Access ends at once…" and the blocked text use `--hb-ink` | screenshots |
| M6 empty groups | Each group has its own empty sentence; "No revocation jobs." only when every group is empty (TW C7 wording) | QA jobs screens |
| M7 raw reason codes | `ReasonCode` maps the ledger codes to the HYB-298 wording plus `record_not_confirmed` and the cleanup retry budget; anything else is "Reason code: {code}" in mono | `bulk-copy.ts` `reasonCode` |
| M8 pending-draft failure | When the first staging batch is refused, the console reads the jobs page and shows only the applying cause (own open draft, or the workspace limit) with an "Open Revocation jobs" link; the general failure paragraph is not shown then | QA E "M8 pending-draft failure shows only the applying reason", "M8 the link opens Revocation jobs" |
| M9 dialog facts | `dt` 14 px muted; "Prepared at" added | QA dialog screenshots |
| M10 period form pre-fill | `initialEnd` (moved to `period-model.ts`) starts from the stored end, "No end date" when there is none; QA reads the radios before any click | `period-model.test.ts` "the period form starts from the stored end"; QA "M10 the form opens on the stored end before any click" |
| L1 From/Ended | Label and stamp on one line (`m01-dated`, the stamp is inline) | screenshots |
| L2 | Job detail and preview show "Prepared at"; waiting/unconfirmed states are 600 in the list (`m01-strong`) | QA jobs screenshots |
| L3 | Blocked list: name and role, the role period line, then "Remove from selection" with 8/16 px gaps | QA D blocked screenshot |
| L4 | Plurals via `counted()` and the TW strings | `bulk-copy.ts` |
| L5 | With the pointer moved off the table, checked rows and the caller's own row have no background | QA "L5 pointer off" at every locale and width |
| L6 | Draft rows say "Not counted until prepared" and, when the API has it (`staged` on the job row), "Sent so far: {n}" | `TestBulkRevocationFlow` (`staged` on the draft row), `identity-bulk.test.ts` |

## Technical writing

Every proposed string (C1-C13 and the extras) is adopted as listed, with these notes: `openDraft` is split in two because the console can tell the causes apart (the jobs page says whether the caller has an open draft); the general sentence is not needed. "Show more excluded people" has a key and a control backed by a new paged read (`/identity/bulk-revocations/{id}/targets?kind=excluded`, owner only, frozen lists only). The jobs list loads 100 rows and now offers "Show more lists" with "Loaded so far: {n}". The draft discard uses the person's reason or the localized "Discarded on the revocation screen". The glossary gained "administrator without an end date", and the ends-soon AR cell lists the plural forms. `revocationTitle`/`revocationBody` were deleted. The command layer keeps only `invalid_request` fields, so the server cannot name the refused cause in the response; the console reads it from the jobs page. The executor does answer `revision_conflict` for `add_targets` (for example when another administrator signs in during staging), so the optional line "Another change happened on the server. Sending the same people again." (`b.resent`) is shown next to the progress line while the same batch is resent with the same key (`identity-bulk.test.ts` checks the `resent` progress event).

## New QA checks, proved on the build before the fixes

`qa299.mjs` gained three checks run on every captured screen: (a) no two visible cells in a row with the same `data-label` and no repeated text block in a row, (b) in AR, every date and time text node is the sole content of its own `bdi` (a zone in an LTR `bdi`), (c) no visual word join between adjacent inline runs (same line, gap under 3 px; Korean particles after a stamp or a number are allowed). They ran first against a build of the index as it was before these fixes (`git checkout-index` copy, `hb-api299pre`, phase L): **43 failures** — (a) "Password set", "No end date" and "Membership suspended" repeated per row in EN/KO/AR at 1440, 1280 and 390; (b) "Prepared at … · valid until …", "Computed at", "Loaded at", the jobs "not kept" line and the zone hint in AR at every width; (c) "Bulk revocation 1|Open|Suspend membership" in the jobs list and detail in EN and AR. The same phase on the fixed build: 0 failures for (a), (b) and (c). A first version of check (b) only checked the isolate structure and passed while the AR screenshots still showed "182026/11/"; (b) was strengthened to require a right-to-left rendered direction for each date and time isolate, failed 18 times on that build (the AR users, period form, preview, dialog, jobs and jobs list screens at 1440, 1280 and 390), and passed after the `direction` fix. The clipping check now measures each element's own text nodes, so a visually hidden label inside a button is no longer reported as outside its container.

# Checks run

After the review fixes (rebased on `df2f36c`): gofmt clean; `go vet ./...` ok for linux, windows and darwin; cross builds ok; `go test ./...` all packages ok; coverage `apps/api/server` 100.0%, `modules/bulkrevoke` 100.0%, `modules/decisionledger` 100.0%, `modules/identity` 98.9% (unchanged gaps); `go test -race` ok on server, bulkrevoke, decisionledger, identity, auth, gate and `cmd/hb-api`; `tsc` clean; `npm test` 689/689; `npm run build` ok. New test files: `apps/api/server/review_fixes_test.go`, `modules/decisionledger/bounds_test.go`; new console file `users/Stamp.tsx`.

Before the review fixes:

| Check | Command | Observed |
| --- | --- | --- |
| gofmt | `gofmt -l apps modules` | no output |
| vet | `GOOS={linux,windows,darwin} GOARCH=amd64 go vet ./...` | ok on all three |
| cross build | `GOOS={linux,windows,darwin} GOARCH=amd64 go build ./...` | ok on all three |
| Go tests | `go test ./...` | all packages ok (see the flake note below) |
| coverage | `go test -cover` | `apps/api/server` 100.0%; `modules/bulkrevoke` 100.0%; `modules/decisionledger` 100.0%; `modules/identity` 98.9% (base 98.5%; every new or changed statement covered; the remaining gaps are pre-existing `bump` overflow and session branches); `modules/auth` 97.4% and `modules/gate` 97.2% (unchanged from base; only constants changed); `apps/api/cmd/hb-api` 91.7% (base 91.1%; gaps are `main()` and serve errors) |
| race | `go test -race` on identity, decisionledger, bulkrevoke, auth, gate, commands, cmd/hb-api and apps/api/server; `-race -count=40 -run TestBulkConcurrency` (four concurrent executes of one manifest with concurrent reads, cleanup and sweep) | all ok; exactly one execution accepted in each of the 40 runs |
| tsc | `npx tsc --noEmit` | clean |
| console tests | `npm test` | 675/675 (new: `period-model.test.ts`, `bulk-model.test.ts`, `identity-bulk.test.ts`, registered in `package.json`) |
| console build | `npm run build` | ok |
| source rules | server `TestSourceRules`, `TestOneServerClock`; `modules/bulkrevoke` `TestSourceRules`; identity `TestActiveMembershipIsComparedOnlyInThePredicateAndTheTransitions` | pass. No Go test file contains the forbidden type word. No new file above 1,500 lines |

**Flakes seen, both pre-existing on main `8d65a5f`:** `apps/api/server TestTrustConcurrentActivation` (fails about 1 in 8 runs on main too: the second request's expected revision is read before the third administrator signs in, so both requests can be stale), `TestTimingIndistinguishable` (timing under load), and one run of `modules/tokenmeter TestConcurrentRepeatsExportsAndVoids` (passed 6/6 on rerun, unchanged package).

**IAM-07 engine numbers** (recorded, not thresholds; `-run Scale`, not `-short`): identity deny of 100,000 memberships held the registry lock for about 65 ms; the ledger wrote a header and 100,000 item records in about 2.5 s with 392 syncs and 393 writes, and reopened and verified them in about 1.0 s; stage, prepare and execute of a 100,000-target manifest took about 69 ms. The HTTP path was exercised at 5,000 people through real 4,096-byte batches (`TestBulkStagingLimitsAndOwnership`).

# Browser QA (`scratchpad/qa299.mjs`, `qa299g.sh`)

After the review fixes, on the final build (`hb-api299` from the clean sources, production console build, proxy on 127.0.0.1:4481 → API 127.0.0.1:8841): **A 289/0, C 3/0, B 5/0, D 5/0, E 46/0, N 31/0**. Phase A now runs every captured screen (users, period form, bulk step, preview, dialog, dialog bottom, job detail, jobs list) at EN/KO/AR × 1440/1280/390 through the overlap and clipping check, the style sweep and the new checks (a), (b) and (c), plus "L5 pointer off" and "M10 the form opens on the stored end before any click".

| Phase | What was added | Result |
| --- | --- | --- |
| E (fresh data, `--bulk-revocation-manifest-minutes 5`) | Staging progress line and the rate-limited line (the first batch answered 429 with `Retry-After: 2` by the runner); R-4 no Back while sending; R-5 Resume and Discard after a failed prepare that left a draft; resume reaches the preview; focus on the preview heading, the dialog heading and the job heading; people-to-revoke list; F3 another administrator sees counts but no run or discard; R-6 the executed person leaves the selection; M8 pending-draft failure shows only the open-draft reason and its link opens Revocation jobs; an expired preview reads "Expired" in EN, KO and AR (1440 and 390) | 46 passed, 0 failed |
| N | Not-in-loaded-rows disclosure: a selected person missing from the users read (the runner removes them from the response, as a later page would) at EN/KO/AR × 1440/390, with the overlap, clipping and text checks | 31 passed, 0 failed |

Not reachable in the browser with the four-person sample seed and therefore covered by unit tests only: more than one staging batch and the `revision_conflict` resend line (`identity-bulk.test.ts`), "Show more excluded people" (needs more than 50 exclusions; `identity-bulk.test.ts` and `TestExclusionsReadPages`), "Show more lists" (needs more than 100 jobs).

Before the review fixes:

Final run on the final build: A 105/0, C 3/0, B 5/0, D 5/0 (118 checks). Production build served by `qa298proxy.mjs` on 127.0.0.1:4481 against `hb-api299` on 127.0.0.1:8841 with the sample seed.

| Phase | What | Result |
| --- | --- | --- |
| A | checkbox column, own row without a checkbox, 44 px hit area, filter-wide note, panel on demand; role period set 1-2 minutes ahead, "Ends today at", member signed out at the end, swept row "Role period ended", no Reinstate offered, restore with no end; selection kept across Users, Membership and Invitations; preview groups, targets list, truthfulness (no revoked/complete/succeeded/% words), "Approval recheck not confirmed"; dialog one primary, 12-character digest, width at most 720, checkbox required; job finished with "2 people: 2 succeeded, 0 held, 0 failed, 0 unprocessed."; member signed out by the bulk revocation; 1 px divider; held after a concurrent role change (stale confirmation offers "Reload and close" first); layout matrix EN/KO/AR × 1440/1280/390 for users, period form, bulk step, preview, dialog (with Escape focus return) and jobs, each with the container-aware overlap and clipping check (text range inside its nearest overflow or dialog container, no ancestor overflow) and the computed-style sweep (no radius, shadow, wide or coloured borders) | 105 passed, 0 failed |
| C | restart on the phase A data: "From an earlier server run", "not kept" line | 3 passed, 0 failed |
| B | ledger damaged (isolated): execute shows "Waiting: decision not confirmed", no percentage; a reinstatement is refused | 5 passed, 0 failed |
| D | `--bulk-revocation-max-targets 1`: aria-disabled checkbox and limit sentence; settings disclosure shows "Start flag"; anchor rule blocks admin2 from revoking the last anchor | 5 passed, 0 failed |

The layout matrix found real defects that were fixed before the passing run: the settings table and the open jobs table overflowed their panes (`security.css` gives tables `min-inline-size: 430px`; M01 tables now stack or reset it), login buttons did not wrap in the 568 px list, and the stacked-row grid was applied to the visually hidden header row.

Not run in a browser: staging of more than one batch (needs more than 186 selectable people; covered by `identity-bulk.test.ts` and the Go HTTP test at 5,000), the step-up dialog stacked on the confirmation (sessions in QA were inside the 15-minute window), a real wall-clock lead (covered by Go tests with an injected wall source), operator recovery (Go tests), screen-reader behaviour beyond the live region.

# Regressions

After the review fixes (`reground299.sh`, same wrappers and ports as before; the console build changed twice during the chain, both times only M01 CSS and an M01 label):

| Runner | Result |
| --- | --- |
| M01 `qa264on299` | 52/0 on the final build. The jobs-view check now waits for the list to load before reading it (it read the heading while the page was still loading once; runner timing, not product). |
| M03 `qa298on297` | A 67/0, B 21/0, C 19/0, D 30/0, E 20/0 |
| M02 `qa297` | A 17/0, B 3/0, C 3/0, D 3/0, E 2/0, I 3/0, K 130/0, L 21/0, F1 2/0, F 2/0, G 3/0, H 2/0 |
| HYB-293 `qa293on297` | B 32/0, C 27/0, D 54/0, A 21/0 |
| P01 `qa283full` | 46/0 |
| P02 `qa285fullon297` | A 162/0, B 32/0, C 10/0, D 16/0 |
| P03 `qa287fullon297` | R 288/0, A 23/0, B 12/0, C 24/0 |
| P04 `qa289fullon297` | R 559/0, A 28/0, B 20/0, C 46/0, D 11/0 |
| P05 `qa315bodyon297` | R 481/0, F 132/0, A 17/0, B 9/0, C1 2/0, C2 3/0, C3 30/0, P 12/0, E 142/0 (`--seed none`) |

Before the review fixes:

| Runner | Result |
| --- | --- |
| M01 `qa264on297` → `qa264on299` | 52/0. The only change: the Revocation jobs tab is now a working view, so "Revocation jobs are not available in pre-launch" and "Go to Users" are replaced by a check of the jobs view (plan: "the revocation tab expectation changes"). Unchanged runner first run: stopped at that step. |
| M03 `qa298on297` (HYB-298) | A 67/0, B 21/0, C 19/0, D 30/0, E 20/0 |
| M02 `qa297` (HYB-297) | A 17/0, B 3/0, C 3/0, D 3/0, E 2/0, I 3/0, K 130/0, L 21/0, F1 2/0, F 2/0, G 3/0, H 2/0 |
| HYB-293 `qa293on297` | B 32/0, C 27/0, D 54/0, A 21/0 |
| P01 `qa283full` | 46/0 |
| P02 `qa285fullon297` | A 162/0, B 32/0, C 10/0, D 16/0 |
| P03 `qa287fullon297` | R 288/0, A 23/0, B 12/0, C 24/0 |
| P04 `qa289fullon297` | R 559/0, A 28/0, B 20/0, C 46/0, D 11/0 |
| P05 `qa315bodyon297` | R 481/0, F 132/0, A 17/0, B 9/0, C1 2/0, C2 3/0, C3 30/0, P 12/0, E 142/0 (`--seed none`) |

The regressions ran on the build before the last copy fix (limit sentence plural, held-line plural); the M01, M02 and M03 results equal the HYB-297 record. No HTTP 429 was seen.

All runners used copies of the wrappers pointed at this worktree, `hb-api299` and ports 4482-4487 / 8842-8847 (`qa299reg.sh`, `qa298on299.sh`, `qa297on299run.sh`, `qa293on299.sh`, `qa315on299.sh`, `regall299.sh`); the bodies are the merged `on297` runners unchanged.

# Mutation

After the review fixes. Same tool (`scratchpad/hb299work/mutate.py`), operator mutants on the changed lines only, each in an rsync copy with its own `GOCACHE`, `go vet` then the package tests (60 s timeout); the server run was split into three copies by file.

| Package | Mutants | Killed by the run | Survived | Rejected by vet |
| --- | --- | --- | --- | --- |
| `apps/api/server` (13 changed files: `periods.go`, `increases.go`, `bulk.go`, `bulkstage.go`, `bulkcleanup.go`, `bulkimpact.go`, `clock.go`, `clockbase.go`, `identityledger.go`, `bulkread.go`, `clockrows.go`, `admin.go`, `trusthold.go`) | 516 | 386 | 124 | 6 |
| `modules/decisionledger`, the part not reached before (`identity.go` after line 83, `batch.go`, `prescan.go`, `record.go`, `ledger.go`) | 150 | 137 | 10 | 3 |
| `modules/bulkrevoke` `Registry.Exclusions` (added in this round) | 12 | 12 | 0 | 0 |

Every survivor was then either killed by a new test (the mutant re-applied alone against the new test and confirmed failing) or argued equivalent below.

**Killed by new tests.** Server: 91 of the 124, with `review_fixes_test.go` and `review_survivors_test.go`: proposal counting per store and the per-store recheck flag (`TestEveryProposalStoreCountsOnlyThePersonsPendingProposals`, `TestBulkImpactCountsOnlyThePersonsOpenProposals`), exclusion reasons (`TestClassifyNamesEveryExclusion`), `changedSince` per field, `memberState`, cleanup bounds, retry marking, live cleanup reasons and pickups (`TestAFinalCleanupFailureMarksOnlyItsOwnTask`, `TestCleanupDropsOnlyThePersonsPickups`, `TestCleanupProgressCountsSettledItems`), replay counters, notes, anchors, kept jobs, rebuilt items of both effects, recorded ends and user reinstatement (`TestReplay*`, `TestRestart*`, `TestARecordAppliedAtAcceptanceIsNotedAsReapplied`), the wall-lead log episodes and exact tolerance, the exact ledger tolerance and a damaged latest clock choice behind a verified one, read paging edges and a single error body, the full-length rationale, the suspend-user epoch, a prepare without an effect, promotions and membership reinstatements refused after a wall lead or with a stopped ledger, `activeAdministrators`, impact counts of roles, rights, session pickups and owned services, the clock-behind threshold and the operator-recovery flag in the users read, and `set_role_period` without its `end` field. The other 33 are argued below. Decision ledger: 9 of the 10 (`bounds_test.go`: a batch exactly at the size bound and one byte under, a record exactly at `RecordLimit`, increase-intent shapes, only clock-base decisions carry a clock in the scan).

**Found by mutation and fixed in code:** after a replayed record was applied at invitation acceptance (D6), its job item kept the note "Applies when the membership returns" because `noteAll` does not overwrite an existing note with "re-applied"; `applyAwaiting` now writes "Re-applied at start" for those records (`TestARecordAppliedAtAcceptanceIsNotedAsReapplied`).

**Argued equivalent (not killed):**

- `bulk.go:31`, `bulk.go:91`, `periods.go:100` (`true`→`false` in the `leaving` sets): `strandsAnchors` reads the keys only.
- `bulk.go:47` first `||` (`!known && !member`): users and memberships come from one registry snapshot; a membership without its user does not exist.
- `bulkcleanup.go:122`, `:131` (`<`→`<=` on `len(v.Items)`): the task span's `to` is always below `len(v.Items)`.
- `bulkread.go:180` (`>`→`>=`): equal sequences are the same record.
- `bulkstage.go:25`, `:43`: a command always runs with an authenticated caller; `Visible` is false for an empty user.
- `bulkstage.go:104` and `:107` and `admin.go:482` (`Required: true`→`false`): the handlers refuse the empty value with the same `invalid_request` and the same field (`users`, `digest`, `login`/`email`).
- `clock.go:76` (`lead > 0`→`>= 0`): with a zero lead the offset is recomputed to the same value.
- `clockrows.go:34` (`clock_ahead` at exactly one minute): within the tolerance the server clock catches up to the wall at every read, so the wall is never exactly one minute ahead; beyond the tolerance the lead is hours.
- `clockbase.go:96` (`line > 0`→`>= 0`): no ledger record has sequence 0.
- `identityledger.go:65`, `:259`, `:274` second `||`: records of other kinds fall through to `continue` (switch default, non-manifest target, no job under their request).
- `identityledger.go:74`, `:85`, `:87`, `:90`, `:137` (`set` on membership and user aspects): every reader also compares `state`, which is empty for an absent aspect; an awaiting entry with an active state applies nothing.
- `identityledger.go:167`, `:290` and `bulkcleanup` sort comparators: the keys are unique.
- `identityledger.go:208` (`&&`→`||` while choosing which pending ends to skip for the anchor): it only differs for a non-administrator whose user ID sorts before the first administrator with a pending end; the sample IDs and generated `user-` IDs all sort after `sample-admin`, so no test fixture reaches it. Residual.
- `identityledger.go:268` (`Headed: true`→`false`): `Headed` is read only by `Header` and `Record`, which are never called for a rebuilt job.
- `periods.go:170`, `admin.go:380` (four positions): `periodAnchors` is a no-op for a zero end and for a target that is not an anchor.
- `decisionledger/batch.go:17` (`+ 1`→`- 1`): the sequence is overwritten in `appendLines` before encoding; `shaped` does not read it.

Before the review fixes:

Operator mutants (`== != < <= > >= && || ! true false + ++`) on the changed lines only, applied one at a time in an rsync copy (`scratchpad/hb299work/mut`, separate `GOCACHE`), with `go vet` then the package tests.

| Package | Mutants | Killed | Survived | Rejected by vet |
| --- | --- | --- | --- | --- |
| `modules/bulkrevoke` (round 1) | 249 | 209 | 29 | 11 |
| `modules/bulkrevoke` (round 2, after `boundaries_test.go`) | 249 | 232 | 6 | 11 |

| `modules/identity` | 173 | 167 | 5 | 1 |

| `modules/decisionledger` (partial: stopped after `identity.go` line 83; `batch.go`, `prescan.go`, `record.go`, `ledger.go` not reached) | 137 | 130 | 5 | 2 |
| `apps/api/server` | not run | — | — | — |

`modules/decisionledger` survivors: all five were killed by tests added afterwards (`TestAnyClockFieldOnAnotherKindIsRefused`, `TestAHeaderMayHoldEveryTarget`, `TestVersionedKindsNeedAVersion`), each confirmed by applying the mutant alone. **The server mutation run did not happen**: one server mutant compiles the package and runs the selected tests in about a minute, the six new server files give several hundred mutants, and the run was stopped to hand over. The new server code is covered 100% by statements and by the HTTP, restart, property and race tests above; root decides whether a server mutation run is required before merge.

`modules/identity` survivors: four were killed by tests added afterwards, each confirmed by applying the mutant alone (`SuspendMemberships` version bump, `ReinstateMembershipAt` doing a dry run, `ApplyRecorded` resetting the start of an already active membership, `ApplyRecorded` generation bump). One is equivalent: `ApplyRecorded` clearing `operatorBlock` on a human user, which is never set for humans. An earlier identity pass is in `mut-identity-partial.txt` (it also found the self-targeting survivor in `setRolePeriod`, killed by `TestAMemberCannotSetTheirOwnPeriod`). Hung mutants (for example a lock check inverted) are counted as killed by the 60 s test timeout.

The six `modules/bulkrevoke` survivors are equivalent: strict versus non-strict comparators in sort functions whose keys are unique (manifest numbers, user IDs, ranks compared only when they differ).

# Planning cross-check (after the build)

Rows from the plan's "Requirement cross-check" (`M01/spec.md`, `requirements.md` HB-03/HB-34/HB-42, `retention-access.md` IAM-01–IAM-08, `ui-design.md`, `domain-contracts.md`), checked against the built behaviour and the test or QA evidence. Status words follow the cross-check report. No acceptance test ran.

| ID / source | After HYB-299 | Evidence | Status |
| --- | --- | --- | --- |
| `M01/spec.md:52` role and validity period | Period on every membership; ceiling (setter's end, `-role-period-max-days`); omitted means no end; time-aware predicate; sweep | `TestRolePeriodRefusals`, `TestRolePeriodCeilingAndOmissions`, identity `TestTheTimeAwarePredicateInTheWindowBeforeTheSweep`; QA A | covered, by interpretation (Q2) |
| `M01/spec.md:37` detail shows role period | "Role period" fact with start and end, "Latest revocation" link | QA A "detail role period fact" | covered for role period |
| `M01/spec.md:59` columns | Role period column and revocation effect sub-line; login realm not built | QA A layout matrix | partially covered (R8) |
| `M01/spec.md:61`, `:63` filters, server search, selection by ID | Selection by user ID across Load more, reload and tabs; filters and search not built | QA A "selection kept across tabs"; `bulk-model.test.ts` | covered for selection; search deferred to HYB-294 |
| `M01/spec.md:41`, `:55` revocation result and durable deny | Deny time, ledger records, cleanup in three groups; `deny_generation` on items and header | `TestBulkRevocationFlow`; QA A job view | covered for bulk and role periods; single commands deferred (R2) |
| `M01/spec.md:75` selected-scope revocation | `add_targets`, `prepare`, `execute`, `discard` under `revoke_access_in_bulk` plus `administer_installation`; fixed manifest and digest; role ceiling; deny first; chunked cleanup; confirmation of targets (paged targets read), exclusions and effect scope | `TestBulkRevocationFlow`, `TestBulkAnchorRule`, `TestBulkStagingLimitsAndOwnership`; QA A | covered |
| `M01/spec.md:73`, IAM-06 last administrator | Anchor rule in `protect`, period writes, bulk prepare and execute, sweep defence, replay skip and operator recovery | `TestAnchorAdministrators`, `TestBulkAnchorRule`, `TestRestartRoleGuardAndAnchorKeeping`, `TestRestartIntoOperatorRecovery`, property test; QA D | covered for the paths added here |
| `M01/spec.md:79` Revocation jobs tab | Working view replaces the unavailable block | QA A and C; M01 regression 52/0 | covered |
| `M01/spec.md:95-98`, `:100`, `:106` states, enum, fixed denominator | partial and error copy; `running`, `waiting` with reason, `succeeded`; no retry button; `targets = succeeded + held + failed + unprocessed` | `bulkrevoke` property test; `bulk-model.test.ts`; QA A equation | covered |
| `M01/spec.md:104` fixed snapshot, held targets | Page selection; filter-wide blocked with a sentence; changed targets held, never updated silently | `TestBulkSuspendUserAndHeldTargets`; QA A held | covered for page selection; filter-wide blocked (HYB-294) |
| `M01/spec.md:112`, `:114` mobile steps, keyboard, live region, never colour alone | Steps below 1280; full-screen dialog at 390; focus moves; single polite region; words and weight | QA A matrix (390 dialog, Escape focus return), style sweep | covered; screen reader not run |
| HB-03 separate action, recheck at job start and chunks | Separate bulk action; fresh decisions at execute; per-target basis recheck; chunk recheck by deny time; generation bump | `TestBulkCommandEdges`, `TestBulkCleanupRetriesAndFencing` | partially covered (registry-wide leakage acceptance not built) |
| HB-34 jobs, partial success, checkpoint | `modules/jobs` tasks with lease, fencing, checkpoint, retries; accepted is not completion | `TestBulkCleanupNotStartedAndStaleLeases`, `TestBulkCleanupRetriesAndFencing` | partially covered (R4) |
| HB-42 role expiry, bulk, security block first | Ends enforced at the instant; deny before the answer; replay before the listener | `TestRolePeriodEndsAccessAndIsSwept`, `TestRestart*` | covered for role periods and bulk |
| IAM-05 reach sessions, keys, jobs, files | Sessions, step-up evidence, pickups; keys not applicable; jobs and files outside scope | cleanup items B1-B10; QA A "member signed out" | partially covered |
| IAM-07 100,000 users | Engine-level 100,000 run recorded; HTTP path 5,000 | scale tests; `TestBulkStagingLimitsAndOwnership` | partially covered |
| IAM-08 step-up | Every new command is mutating; HYB-293 rule unchanged | step-up registry tests pass; M02 and HYB-293 regressions | covered by HYB-293 |
| RET-05 (`retention-access.md:98`) durable commit, replay before opening, isolation | HYB-298 ledger with v2 kinds; identity replay in `Start`; degraded replay applies decreases only; `ErrFormatNewer` | `TestRestartIncreasesNeedTheirConfirmation`, decisionledger `TestAWrongVersionIsolatesAndANewerVersionRefusesToStart`; QA B | covered within HYB-298's limits |
| `retention-access.md:59`, `:61`, `:65` order, 100k, screen | Deny, ledger, cleanup; deny for every target in one step; impact preview and bulk suspension | `TestBulkRevocationFlow`, identity `TestBatchSuspensions` | covered |
| `ui-design.md:150`, `:384`, `:450` preview groups | People and roles, rights, sessions and keys, pending work per store, service users, anchors after, scope | `TestBulkImpactCountsEveryStore`, `TestProposalStoresCoverEveryProposingNamespace`; QA A | covered; pending-work recheck confirmed for M02 and M03 only (partially covered, R10) |
| `domain-contracts.md:79` fixed manifest, bounded detail | Draft staged in batches, frozen with digest, items and targets paged | `TestBulkReadsRefuseAndPage` | covered |
| UI-04, UI-07 | Explicit selection basis and counts; no false progress | QA A truthfulness checks | UI-04 deferred (HYB-294); UI-07 partially covered |

# Deviations (with reasons)

- **D1 `InviteUntil`.** `Workspace.Invite` keeps its signature and `InviteUntil` takes the role end, following the plan's own `ChangeRole`/`ChangeRoleUntil` pattern, so the many existing callers stay unchanged.
- **D2 no `CheckSuspend*`.** Decreases apply and then commit; nothing calls a suspension dry-run, so none was added.
- **D3 scale test placement.** The 100,000-target run is engine-level in three module tests (identity deny, ledger chunks and reopen, manifest), not one apps/api test, because creating 100,000 memberships through the public identity API is quadratic (each invitation counts every pending invitation). The HTTP path is tested at 5,000.
- **D4 zero targets.** `prepare` of a draft whose people are all excluded is refused `prerequisite_failed` (the plan does not say; a job with no target would record nothing).
- **D5 reads in OpenAPI.** OpenAPI lists commands only, as for every other read in this service.
- **D6 records applied at acceptance.** Because the seed member's membership only appears at acceptance (delta 1), a replayed membership suspension, role end or confirmed restore for a person without a membership at start waits in memory and is applied right after that person's invitation is accepted (`awaiting`, note `applies_when_membership_returns`). Without this, a bulk-suspended seed member would regain access by signing in after a restart.
- **D7 test clock.** `Config.Elapsed` injects the monotonic source. When only `Config.Clock` is injected (every existing test), it serves as both wall and elapsed source, so existing tests that move the clock keep their behaviour. Production injects neither.
- **D8 test worker seam.** Test servers set an unexported `manual` flag that skips the cleanup worker and the 30 s ticker; tests call `drainCleanup` and `sweep` directly.
- **D9 E2 dead end.** If the latest `clock_base` record cannot be verified, `Start` is refused as the plan says, and a `-clock-base` flag is then accepted even without a lead, so the operator can recover with the flag the error names. After the security review (F1) this exception holds only while that choice is unverified: a flag left on after a verified choice and without a lead is refused with `ErrConfig`.
- **D10 exclusions before execution.** The preview shows exclusion counts by reason and, after the TW review, a paged per-person exclusion list with "Show more excluded people" (`/{id}/targets?kind=excluded`, owner only, frozen lists only). The targets list (root amendment) is paged.
- **D17 owner-only execution (F3).** Only the preparer can execute or discard a prepared list; this answers Q3. Because the preparer is always excluded from their own list (`self`), the executor can no longer be a target, so the `self` hold branch at execute was removed; `held` now means changed since preview only.
- **D18 `change_role` end field (F4).** `identity.change_role` gained an optional `end` field (`at` or `none`, the same values as `set_role_period`). Omitted `role_until` keeps the stored end; `end=none` removes it.
- **D11 selection storage.** The M01 screen is remounted per tab, so the selection is kept in module memory per signed-in user (in memory only, R-3), and the job to open after an execution is handed over the same way.
- **D12 blocked sweep.** `ExpireRoles` treats an ended administrator as blocked when no other effective administrator remains at that instant (the ended one is already not effective, so "would leave no administrator" is judged on the remaining ones).
- **D13 `role_period_set` record** carries `requested_by` (the actor's user ID; never a login or rationale).
- **D14 identity settings** are a separate list, shown in the M01 reads, not in the M03 read.
- **D16 lazy clock read.** `Subjects.Member` reads the identity clock only for a membership with an end (`effectiveNow`). An unconditional read shifted the clock-read choreography that `modules/commands` tests use to inject faults (one test deadlocked); with the lazy read those tests are unchanged.
- **D15 test adaptations** (expected values only): seed step count 19→20 and the related replay counters (a new `bulk-grants` seed step), the full-ledger fixture now uses the test clock instead of `time.Now` (it would trip the new fail-closed clock check), the allowed grant-change sites gained `grantBulk`, the users read allowlist gained `revocation`, the hold table test counts 78 commands and the new conditional row, and the `protect` fixture sets `Effective`.

# Plan-text changes root must make

After the review fixes:

- Q3 default: execute and discard are the preparer's only (F3, D17); plan step 5 "A target who is the executor is held with `self`. The executor need not be the preparer" no longer applies; the held reason is `changed_since_preview` only.
- `change_role` direction (F2): an increase is a promotion to administrator or a later/removed end; the same role with a shorter end is a `role_period_set` decrease; the hold rule and degraded ledger follow the same direction; shortening an administrator's end through `change_role` is subject to the anchor rule; the live-link prerequisite applies to promotions only.
- `change_role` omitted `role_until` keeps the stored end; removal needs `end=none` (F4, D18). Create and invite unchanged.
- E2/D9 restated for F1: the flag is accepted without a lead only while the latest choice is unverified.
- New owner-only read `/api/v1/identity/bulk-revocations/{id}/targets?kind=excluded` (D10); job rows carry `staged` for drafts.
- Pending-draft failure: the console tells "own open draft" from "workspace limit" by reading the jobs page (`open_draft`), because the command layer returns no fields with `prerequisite_failed`.

Earlier:

- A1: the seed member is invited at start, not a member; describe D6.
- Command count 77 → 78.
- `InviteUntil` (D1); no `CheckSuspend*` (D2); IAM-07 test placement (D3); zero-target refusal (D4).
- The targets read route is `/api/v1/identity/bulk-revocations/{id}/targets`, owner only, same answer as unknown for anyone else.
- Exclusions before execution are counts only (D10).
- E2 recovery with the flag (D9).

# Residuals

After the review fixes:

- No re-review of this build yet (security, render, TW).
- Browser QA ran on the API binary built before the last server change (the replay note written at invitation acceptance); that change is covered by Go tests only.
- Not reached in a browser with the sample seed: more than one staging batch, the `revision_conflict` resend line, "Show more excluded people", "Show more lists" (unit and Go tests only).
- Mutation: `identityledger.go:208` stays unkilled (needs a non-administrator whose user ID sorts before the first administrator); the other 32 server survivors and one ledger survivor are argued equivalent above.
- The command layer returns fields only with `invalid_request`, so the two causes of a refused new draft are told apart by the console from the jobs page, not by the command answer.
- KO and AR copy remain drafts for native review (R9).

Earlier:

- No independent security change review, render review or TW review yet.
- R-items of the plan stay open (R1-R15), in particular R2 (single-person decreases not recorded), R4 (in-memory job registry), R11 (HYB-298-only binary isolates on v2), R12 (operator-recovery exit), R13 (grant lapse).
- Approval recheck of a revoked proposer is confirmed only for M02 and M03; the other six proposal stores read "Approval recheck not confirmed" (R10).
- KO and AR strings, including every [TW] string from the visual spec, are drafts for TW and native review (R9).
- Pre-existing flaky tests listed above were not changed (outside this ticket).

## Root final verification (2026-11-18, main 1cf7f3f)

- Root rebased twice (df2f36c, then 1cf7f3f); conflicts only in `apps/console/package.json` (test lists merged), `apps/api/cmd/hb-api/main_test.go` (both test sets kept, merged by function) and a type-name clash (`targetsPage` renamed `bulkTargetsPage` because HYB-300 added `targetsPage`).
- Root re-ran: `gofmt` clean, `go vet ./...` ok, `go test ./...` ok, coverage 100.0% on `apps/api/server`, `modules/bulkrevoke`, `modules/decisionledger`; `tsc` clean; `npm test` 710/710; build ok; no changed file over 1,500 lines; no forbidden word in changed Go tests.
- Reviews: security change review FAIL (F1 clock base, F2–F4) → FAIL (N1 setter limit on an omitted end) → PASS; render review FAIL (3 high) → FAIL (header/cell parity) → PASS with one medium, fixed in round 4 (revocation sub-line in the narrow Users layout) and accepted without a fourth render round because QA asserts it at 1440/1280 in EN/KO/AR; technical-writing review FAIL → FAIL (R1 non-owner false zero) → PASS.
- Public-contract change (omitted `role_until` on `change_role` keeps the stored end; `end` field) reviewed by the security change review rounds 2–3 and recorded in plan amendment 5.
- Residuals: plan residuals R1–R15; owner questions Q1–Q11 with defaults (Q3 answered by plan amendment 4); `identityledger.go:208` unkilled mutant (unreachable with fixtures); staging of more than one batch, the resend line and "Show more" controls covered by unit and Go tests only; KO and AR copy remain drafts for native review; flaky tests on main tracked in HYB-321.

