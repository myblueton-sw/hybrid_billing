---
type: Verification
title: HYB-310 P02 integration rights verification
description: Observed results of the HYB-310 implementation in the ticket worktree — Go and console tests, mutation testing on changed lines, coverage against main, the race detector and rendered browser QA — with deviations, residuals and the copy that still needs technical-writing review.
status: draft
tags: [HYB-310, P02, permissions, verification]
sources:
  - resource: plan.md
  - resource: visual-spec.md
---

# HYB-310 verification

## Scope and base

- **Worktree and base.** The worktree is `/private/tmp/hb310` on branch `feat/HYB-310-p02-permission-split`, rebased onto main `8a0c8d7` (HYB-307 merged). Nothing is committed.
- **Plan.** Status is stable, with design review round 4 recorded as APPROVE (notes 1–3 folded in).
- **Visual spec.** `visual-spec.md` is root's file and was not edited; its §13 root decisions were followed.
- **What these results are not.** Results recorded here are from a single machine run on 2026-11-19. They are not owner acceptance, a security review of the diff or a technical-writing approval.

## What changed

- **Rights.** Eight installation-kind rights were added to `modules/auth` and re-exported by `modules/gate`:
  - `administer_api_clients`;
  - `administer_modules`, `approve_modules` and `stop_modules`;
  - `administer_channels`, `approve_channels` and `stop_channels`;
  - `reprocess_events`.
  `InstallationAction` is true for exactly 14 actions.
- **Author-only commands.**
  - `modules/auth` gained `CheckAuthor` and `RecheckAuthor`, wrapped by `modules/gate`.
  - `modules/commands` gained the `Author` definition flag. Registration accepts it only on mutating, non-approval definitions whose permission is an installation action.
  - The executor calls `CheckAuthor` before the idempotency lookup and `RecheckAuthor` at commit.
  - OpenAPI marks these commands `author_only` and `right_not_required`.
  - `authority_unavailable` is now accepted as a handler code.
- **Module operations.**
  - `modules/plugins` gained `Close`, with the reason `requester_lost_right`.
  - `modules/webhooks` gained:
    - withdraw and close for proposed subscriptions, resume requests and letter requests;
    - `HoldRedelivery`, `CloseRedelivery` and `ReapproveRedelivery`;
    - `Dispatch` no longer leases a held delivery;
    - deliveries carry their requester and approver.
  - `modules/identity` gained `Installation().Member`, a lookup of one member.
- **Server.**
  - A single right table (`integrationRight`) drives the 20 `integrations.*` definitions, the P02 read, the availability flags and OpenAPI.
  - New commands: `subscription_withdraw`, `subscription_withdraw_resume`, `letter_withdraw_redeliver`, `letter_withdraw_dismiss` and `redelivery_reapprove`.
  - **Starter recheck.** Each approval checks state first, then whether the starter still qualifies, giving qualifies, lost or unknown. A starter who lost the right gets business state `closed` with `reason: requester_lost_right`. An unknown result returns 503 `authority_unavailable` or `dependency_unavailable` and closes nothing.
  - **Request records.** Requests get IDs with tombstones, capped at 10,000 per kind with fail-closed eviction.
  - **Dispatch.** `dispatchDeliveries` is the only caller of `Dispatch`.
  - **Grants.** Seed and bootstrap grants are all-or-nothing. In the sample seed, admin3 holds stop and reprocess only.
  - **Bulk impact.** `integration_proposals` is now rechecked and also counts resume requests.
- **Read.**
  - All nine admission decisions are taken together; one unavailable decision fails the whole read with 503.
  - The read returns `rights` and `own_pending`.
  - A withheld area returns only `state: not_permitted`.
  - `no_right` and `requester_lost_right` blockers are reported before state reasons.
  - Approval admission (`s.admits`) is skipped for readers without the approval right.
- **Console P02.** The changes follow the visual spec:
  - the rights line, including the unknown variant (D2);
  - withheld panels, with "Your pending items" inside them;
  - per-row `no_right` and starter-lost lines, filtered to the actions the row's state makes applicable;
  - the closed-on-approval notice (D3);
  - the no-longer-held note, shown in the row note;
  - the denied block with "Your last action was not carried out" (D4);
  - held redeliveries with "Approve again";
  - Withdraw on the caller's own rows;
  - no command offered while a failed reload shows (D6);
  - the D1 starter-lost line without "will be closed".
- **Documents.**
  - Glossary rows were added.
  - P02 entries were updated in the binding register and the field map.
  - `HB-284/plan.md:376` now points to this ticket.

## Defect found by the new tests

- **Defect.** `newRequest` could evict the request it had just created when every older request was live and the cap was reached. The author could then not withdraw that request.
- **Fix.** The new record is excluded from eviction.
- **Test.** The defect was found by `TestRequestEvictionKeepsLiveRequestsAndStopsAtTheLimit`, which failed before the fix and passes after it.

## Fixes after the change reviews

Root's lists `scratchpad/hb310-fixes.md` and `hb310-render.md` were applied.

**Security change review:**
- **Finding 1, redelivery screening.**
  - The filter is now the pure function `screenable`, unit-tested with open, dead, running and held deliveries (`TestTheFilterScreensOnlyQueuedUnheldRedeliveries`).
  - `TestOneUnknownStarterSkipsTheWholeRound` covers a requester unknown while the approver qualifies, and the reverse. A per-user fault `starter-identity-<user>` makes one side unknown. Both cases fail the round: `Dispatch` is never called (counted through `webhookCall("dispatch_round")`), and nothing is held or closed.
  - `TestAHeldRedeliveryIsNotScreenedAgain` shows that a held redelivery is not re-screened even when its requester then loses the right.
  - The `integrationheld.go` filter and unknown-standing mutants are now killed (see "Mutation testing").
- **Finding 2, paging.** Delivery lists are paged with `allDeliveries`; the fixed 200-item window is gone. `TestDeliveryListsArePagedPastOnePage` sets the page size to 1, so the held redelivery lies beyond the first page; it is still screened and re-approved.
- **Finding 3, who closed it.**
  - The plugin closure now stores the person whose approval attempt found the loss as `closure_trigger` and `closure_trigger_login`, and leaves `closed_by` empty.
  - The console builds the found-when sentence from the trigger, with the person and the request ID set in mono.
  - Tests: server (`TestApprovalsCloseWorkWhoseStarterLostTheRight`) and console (`a closure is credited to nobody…`).
- **Finding 4, user change.** `setScope` now clears the lost-right marks whenever the workspace, user or session changes (console test `lost-right marks are cleared when the signed-in user changes`).
- **Plan wording.** The plan now says that a replay is protected by `CheckAuthor`, which runs before the journal lookup, and that `RecheckAuthor` covers new calls.

**Render and copy review:**
- **Reload failure (503).**
  - The heading reads "The integrations could not be loaded." (KO and AR as given).
  - The read-only note reads "The last reload failed, so this screen is read-only until a reload succeeds."
  - The empty "Propose a subscription" section is not rendered.
  - The cause line also used the sign-in text "The sign-in service is not responding…". It now reads "The integrations service is not responding. Try again shortly." [TW, new string not in the review].
- **Arabic and history.**
  - Arabic "Held" is now "محتجَز", and the AR held status begins with it.
  - The subscription history "By" cell for a `requester_lost_right` closure reads "Closed automatically" / "자동 종료" / "أُغلق تلقائيًا". The approver appears only inside the found-when sentence.
- **Copy items 2–8.**
  - The AR dispatcher term is "المُرسِل غير متصل".
  - EN right labels are lower-cased inside sentences, except "API" (D5).
  - Lists use fixed separators: EN "; ", KO ", ", AR "، ".
  - KO no-right line: "… 현재 이 권한이 없습니다."
  - KO pending tag: "승인 대기".
  - AR target label: "الهدف".
  - D8 is kept as "Back to edit".
- **Minor fixes.** The "Propose a subscription" no-right line no longer repeats its heading as a bold label. Request IDs in closure lines are set in mono.

## Go tests

| Command | Result |
| --- | --- |
| `go test` on `modules/auth`, `gate`, `commands`, `plugins`, `webhooks`, `identity` | ok (each package) |
| Final `go test -count=1 -cover` on the eight changed packages, after every change | ok, each package (see "Coverage") |

Tests added, by plan item:

| Plan item | Tests |
| --- | --- |
| T1 | `TestInstallationActionsAreExactlyTheFourteen`; `TestIntegrationRightsAreSeparateFromEveryOtherRight` (every pair of the 14 rights); service-subject, delegation and wrong-kind refusals |
| T2 | `TestIntegrationDefinitionsCarryTheirRights`: a literal table of 20 rights and the author flag; no non-integration command accepts an integration right |
| T3 | `TestEveryIntegrationCommandNeedsExactlyItsRight`: 14 commands × 12 profiles = 168 cases, including "every other right"; `TestTheRightNeedsBothThePersonAndTheConsoleGrant` |
| T4 | `TestTheReadShowsEachAreaOnlyToItsRights` (15 profiles); `TestAnUnavailableDecisionFailsTheWholeRead`; `TestReadersWhoCannotApproveTriggerNoAdmissionChecks`; `TestTheReadRefusesBadSessions`; `TestAnEndedSessionCannotReadP02` |
| T5 | `TestUnknownTargetsAndMissingRightsLookAlike`: byte-identical bodies apart from the request ID, including an expired grant |
| T6 | `TestRealRevocationPathsTakeEffectPerArea`: demotion and re-promotion, membership suspension, user suspension, role period, bulk revocation, grant revoked and console grant removed, for one right per area |
| T7 | Separation is covered inside T12 and T14, and by the existing self-approval tests |
| T8 | `TestSeedAndBootstrapGrantAllOrNothing` (k = 0..7, rollback failure, bootstrap, failed bootstrap grant); `TestGrantFailuresAreReportedExactly` |
| T9 | `TestM03NeverOffersIntegrationRights` |
| T10 | `TestTheIntegrationsReadIsDocumented` |
| T11 | `TestHoldTableIsExplicitAndComplete` (88 rows) |
| T12 | `TestApprovalsCloseWorkWhoseStarterLostTheRight` (6 loss paths × 5 approvals); `TestClosingFreesTheSlotAndStateChecksComeFirst`; `TestARevocationRacingAnApprovalIsOneOrTheOther`; `TestStarterChecksFailClosedOnEveryApproval`; `TestTheStartingRightAloneKeepsWorkApprovable`; `TestClosingsLeaveHistory` |
| T13 | `TestQueuedRedeliveriesAreHeldOrClosed`; `TestDispatchHasOneCaller`; `TestScreeningSkipsAndFailsTruthfully`; `TestScreeningUsesTheReprocessRight`; module tests `TestHeldRedeliveriesAreNotLeased` and `TestAHeldOpenRedeliveryReturnsToTheQueue` |
| T14 | `TestAuthorsWithdrawTheirOwnWork`; `TestLetterRequestWithdrawalIsPerKindAndPerRequest` (N1 cross-kind; R3-3 sequence; root note 1); `TestResumeRequestsAreWithdrawnByTheirAuthor`; `TestRequestTombstonesAreCapped` (root note 2); `TestOwnPendingListsOnlyTheCallersLiveWork` |
| T15 | `modules/commands/author_test.go`: registration, executor, recheck at commit, replay and a 300-round property test; `TestAHandlerMayAnswerAuthorityUnavailable`; gate and auth `CheckAuthor` rule tests |
| T16 | `TestAdmissionDoesNotChangeTheAreas` |
| Property tests | `TestCommandsAreRefusedExactlyWhenTheRightIsMissing` (25 seeds × 12 steps with random re-grants); `TestTheReadAgreesWithRandomRights` (25 seeds) |

Existing tests that were updated because their expectations changed:
- the seed step count went from 20 to 21;
- `TestIntegrationCommandSet` and `TestIntegrationSeed`;
- the hold-table lists;
- the allowed grant-change call sites;
- `TestBulkImpactCountsOnlyThePersonsOpenProposals` (`rechecked` is now true);
- read-shape strings, which now include the history `reason` and `request`, and the request `id` and `target`;
- withdrawal by a non-proposer is now 403 `permission_denied`;
- the approval faults now set up pending requests first.

## Mutation testing (changed lines)

- **Tool.** `scratchpad/mutate310.py`, run against copies of the worktree. Changed lines are measured against `8a0c8d7`, and new files are mutated in full. Server mutants run the integration tests filtered with `-run`; module mutants run their whole package. `GOCACHE` was one directory per worker under the scratchpad, deleted after each batch.
- **Restore check.** After every run, `diff -rq` of the copies against the worktree showed no source difference. The only differences were test files edited after the copies were made.

| Run | Mutants | Killed | Did not compile | Survived |
| --- | --- | --- | --- | --- |
| Round 1 | 656 | 436 | 95 | 125 |
| Final full run (after the new tests) | 659 | 526 | 103 | 30 |
| Rerun of the 30 survivors against later tests (matched by mutated text; 42 mutants) | 42 | 23 | 0 | 19 |
| After the review fixes: `integrationheld.go` and `integrationrights.go` in full, with the new tests | 237 | 211 | 25 | 1 (`integrationrights.go:369`) |

All five mutants of the screening filter (`integrationheld.go:31`) and every mutant of the unknown-standing branch (`:70`) are killed in the last run. Of the earlier 19 survivors, 15 remain; the 4 on the old `integrationheld.go:46` and `:52` lines are gone. They were each traced to a reader:

| Survivor | Classification |
| --- | --- |
| `auth/author.go:18` if-false | Equivalent. The subject comparison still refuses, and a principal's subject is never empty. |
| `auth/author.go:38` `||`→`&&` | Equivalent. Stamp equality already separates approval and author stamps. |
| `integrationcmds.go:204` (2), `:384`, `:418` (2), `:437` (2) | Defence in depth. `CheckAuthor` (author and kind) or the module refusal already gives the same refusal at the boundary. Kept on purpose. |
| `integrationcmds.go:529` `&&`→`||` | Equivalent while each held redelivery has a unique ID. |
| `integrationcmds.go:615` destination `Required` | Equivalent. The handler refuses an empty destination with `invalid_request` on the `destination` field. |
| `integrations.go:643` `&&`→`||` | Unreachable state: a module resumer without a server resume record. |
| `integrations.go:705` | Equivalent. When the reader is the requester and holds the right, the starter cannot be lost. |
| `integrationrights.go:369` | Equivalent in tests: every fresh grant has version 1. |
| `signin.go:490` | Changes only which log line is written. |

## Coverage

Statement coverage, `go test -count=1 -cover`, main `8a0c8d7` against the change:

| Package | Main | Change |
| --- | --- | --- |
| `modules/auth` | 97.4% | 97.5% |
| `modules/gate` | 97.2% | 97.3% |
| `modules/commands` | 99.5% | 99.5% |
| `modules/plugins` | 100.0% | 100.0% |
| `modules/webhooks` | 100.0% | 100.0% |
| `modules/identity` | 99.0% | 99.1% |
| `apps/api/server` | 99.9% (785.3 s) | 100.0% (778.2 s); after the review fixes 99.9% (953.3 s) |
| `apps/api/cmd/hb-api` | 93.6% | 93.6% |

The module figures for the change were the same in the run after the review fixes. No package is below main; the server returned to main's 99.9% after the review fixes, and which statements are uncovered was not profiled. That run is the final full test run of these packages, and every package reported `ok`. The other 72 Go packages (`go test` over `./...` without `apps/api/server`) also passed.

## Race and vet

| Command | Result |
| --- | --- |
| `go vet ./...` | exit 0 |
| `go test -count=1 -race -timeout 120m` on the eight changed packages | every package `ok` (`apps/api/server` 4844.2 s); 0 `DATA RACE` reports; exit 0 |

## Console

| Command | Result |
| --- | --- |
| `npx tsc --noEmit` | ok |
| `npm test` (all console suites, after the review fixes) | 800 tests, 800 pass, 0 fail |
| `npm run build` | built |

New console tests:
- `service/integrations.test.ts`: rights, own pending items, withheld areas, held redeliveries and request IDs, all failing closed;
- `live-model.test.ts`:
  - `deniedOutcome` returns `no_right` or `gone`;
  - withheld areas and own pending items per view;
  - applicability;
  - pruning of the lost-right note;
  - rights copy in EN, KO and AR, including the fixed list separators and lower-case labels inside sentences;
  - closure credit (trigger only inside the found-when line, request ID in mono, "Closed automatically" in history);
  - clearing lost-right marks on a user change;
  - the hidden empty "Propose a subscription" section;
  - a rendered operator view (no Approve button, only the no-right lines, no own-proposal sentence).

## Browser QA

- **Setup.**
  - The server is a fresh `hb-api` built from this worktree. The console is the production `dist` served through `qa298proxy.mjs`.
  - Ports 4551–4559 and 8880–8889. Each group used a new mode-0700 data directory, deleted afterwards.
  - Runner `scratchpad/qa310p02.mjs`, driven by `qa310run.sh`. Screenshots are in `scratchpad/q310shots/` (39 files).
- **Failing first.**
  - **First round.** Every check not marked `[guard]` was first run against main `8a0c8d7`, built in `/private/tmp/hb310-base`. Not one of them passed on main. `[guard]` checks are setup steps or regression guards; they pass on both builds, except 6 in group E that also fail on main.
  - **Review-fix checks** were first run against the previous HYB-310 build (before these fixes):
    - S: 94 PASS, 28 FAIL, including the 503 heading, read-only note and empty-heading checks, the EN list and AR separators;
    - E: 74 PASS, 18 FAIL, including the lower-case no-right lines, the repeated label, the list separators in EN, KO and AR, and the KO no-right wording;
    - F-en, F-ko, F-ar: 2 FAIL each, the closure "By" cell and the request ID in mono;
    - the added 503-cause check failed in EN, KO and AR (`q310-old-S-cause.out`).
  - **Not proved failing.** The AR "no sign-in wording" check was corrected after the old build was replaced, so it was not itself proved failing. On the old build, the AR 503 heading check failed with the sign-in text.

| Group | Main `8a0c8d7` | This change | Page or console errors (change) |
| --- | --- | --- | --- |
| S (admin: rights line, old line absent, EN/KO/AR at 1440 and 390, 1280 and 1024, style, 503 state in EN/KO/AR) | 26 PASS, 82 FAIL (first-round checks) | 125 PASS, 0 FAIL | none |
| E (admin3 operator: withheld Clients, computed styles, no_right lines, precedence, RTL) | 40 PASS, 46 FAIL (first-round checks) | 92 PASS, 0 FAIL | none |
| F-en (lost right: closed-on-approval, history credit, starter-lost line, `no_access`, re-promotion, withdrawal) | 7 PASS, 9 FAIL (first-round checks) | 18 PASS, 0 FAIL | none |
| F-ko | 7 PASS, 4 FAIL | 13 PASS, 0 FAIL | none |
| F-ar | 7 PASS, 4 FAIL | 13 PASS, 0 FAIL | none |

- **Demotion path.** The demotion and re-promotion in group F use the same `identity.change_role` command that M01 posts, sent from the admin's signed-in browser session with its CSRF token. The M01 screen flow itself was not driven; HYB-293 covers it.
- **Stale page.** The stale-page approval first meets the revision conflict that the demotion causes, as designed. The second confirm then gets the closure. Group F now also closes subscription Y from a stale page and checks the history "By" cell.
- **Screenshots.** They were re-rendered on the final build, including `S-503-{en,ko,ar}-1440` and `F-history-{en,ko,ar}-1440`.

### Regressions on this change

| Runner | Result |
| --- | --- |
| `qa285fullon297.mjs` A / B / C / D (final build) | 162 / 32 / 10 / 16 PASS, 0 FAIL. Group D reports 2 expected aborts, as before. |
| `qa309.mjs` L / S (after the review fixes, before the 503-cause string) | 69 / 8 PASS, 0 FAIL |
| `qa298on297.mjs` A–E (its phased driver, `qa298on310.sh`; same build as `qa309`) | 67 / 21 / 19 / 30 / 20 PASS, 0 FAIL, 0 errors |
| `qa267.mjs` | Stops at its line 74, on main `8a0c8d7` as well. It waits for the removed direct `access.grant` notice; this is documented in `HYB-298/verification.md:156`. |

### Not browser-verified

- **Held redelivery and Approve again, including AR "محتجَز".** No dispatcher is connected (R9). Covered by server, module and console unit tests only.
- **Your pending items in a withheld area.** No sample profile can produce it at run time. Covered by API tests (T14) and console unit tests.
- **The `no_right` refusal note after a command.** The browser cannot remove a single right without a test hook. Covered by console unit tests.
- **External layout references** for P02. The spec §13 records the root decision that none is needed.

## Deviations from the plan

- **Files outside the plan's list:**
  - `modules/identity/periods.go` and `member_test.go`: `Installation().Member`, so the starter recheck reads one member instead of listing them all;
  - `apps/console/src/service/admin.ts`: the command ID union;
  - console helpers in new files `live-rights.ts`, `rights-copy.ts` and `LivePending.tsx`;
  - `docs/ui/screen-binding-register.md` and `docs/ui/screen-binding-fields/operations.md`, as root asked.
- **Engine tests file.** The engine tests are in `modules/commands/author_test.go` rather than in `commands_test.go` and `property_test.go`.
- **Withdraw confirmation.** It keeps the reason field, because every integration command requires a rationale. The spec's "no reason field" remark (§4.6) does not apply.
- **Closure trace for resume and letter requests (D7).** It is shown in the subscription's history (`resume_closed`, `redeliver_request_closed`, `dismiss_request_closed`, `redelivery_closed`) because the server returns those records; no row is invented.
- **Proposed subscriptions** that are withdrawn or closed are stored by the module as `revoked` with reason `withdrawn` or `requester_lost_right`. The console shows them as "Withdrawn" or "Closed".
- **Fault hooks.** The `dispatchDeliveries` module calls are wrapped in `webhookCall` (`close_queued`, `hold`, `dispatch_round`) so they can be fault-injected. The per-user fault `starter-identity-<user>` was added for the one-side-unknown tests.
- **Banned spelling.** Two pre-existing lines in `modules/commands/document_test.go` contain the "flo"+"at" spelling. HYB-310 changed only the count line in that file.

## Residuals

- **Plan residuals R1–R9 stand.** They include:
  - no verification right;
  - no enrolment for integration rights;
  - no non-administrator holders;
  - an end-to-end held redelivery is UNVERIFIED (no dispatcher).
- **Unreviewed parts.** KO and AR copy are AI drafts. TW review and the security review of the diff have not been run.

## Copy for technical-writing review [TW]

- 503 state: "The integrations could not be loaded.", "The last reload failed, so this screen is read-only until a reload succeeds." and the new cause line "The integrations service is not responding. Try again shortly." / "연동 서비스가 응답하지 않습니다. 잠시 후 다시 시도하세요." / "خدمة التكاملات لا تستجيب. أعد المحاولة بعد قليل."
- History credit: "Closed automatically" / "자동 종료" / "أُغلق تلقائيًا".
- Rights line: "Your integration rights: {list}." (list joined with "; " in EN, ", " in KO, "، " in AR) The unknown variant: "Your integration rights could not be read, so no integration action is offered here. Reload to try again."
- "You hold no integration rights. Integration rights are given only when the installation is set up; they can't be granted here yet."
- Withheld area: "You don't hold a right for {area}. Nothing from this area is shown."
- Reasons:
  - "Needs the {right} right. You don't hold it." ({right} lower-cased in EN except "API", per D5; KO "… 현재 이 권한이 없습니다.")
  - "The person who started this no longer holds the right, so it can't be approved." (D1)
- Outcomes:
  - "{Object}: Not approved. It was closed because the person who started it no longer holds the right." The object prefixes are "Proposal n", "Subscription id", "Dead letter id" and "Redelivery id".
  - "You no longer hold the {right} right. Nothing was changed."
  - "Your last action was not carried out. Nothing was changed."
- Closure trace: "The person who started it no longer holds the right. Found when {approver} tried to approve (request {id})."
- Your pending items:
  - "Your pending items" with the lead "You can withdraw work you started, even without a right for this area."
  - Kind labels: "Plugin change proposal", "Subscription proposal", "Resume request", "Redelivery request", "Dismissal request".
  - "For {target}", "Withdraw {kind} {id}", "Confirm withdrawal", "Withdrawing closes this request. Nothing else changes.", "{kind} {id} withdrawn."
- Held redeliveries:
  - "Held", "Approver no longer holds the right", "Redelivery decision";
  - "Held: the approver no longer holds the Event reprocessing right. Another holder must approve it again.";
  - "Approve again", "Confirm re-approval";
  - "The redelivery is released to the queue. Nothing is sent: the dispatcher is not connected in this build.";
  - "Approve redelivery {id} again", "Redelivery {id} approved again and queued. Nothing is sent.";
  - "You requested this redelivery. Another holder must approve it." and "You approved it before. Another holder must approve it.";
  - "Requester no longer holds the right".
- Buttons and states: "Withdraw resume request", "Withdraw redelivery request", "Withdraw dismissal request", "Closed", "Withdrawn".
- History rows: "Withdrawn", "Resume request withdrawn", "Resume request closed", "Redelivery request closed", "Dismissal request closed", "Redelivery closed".
- D8: the secondary button in the withdraw confirmation stays "Back to edit".
- The eight right labels in EN, KO and AR, as in plan "Console (P02)".


## Combined verification after Codex resumed

This section supersedes the earlier pending-review statements for the resumed candidate only. Original measurements remain historical evidence. Machine: `mac_mini`; branch `feat/HYB-310-p02-permission-split` based on `8a0c8d7`.

- **Security change re-review: PASS.** An independent read-only reviewer confirmed the configured author-client boundary, immutable constructor copy/validation, empty fail-closed configuration, active-client grant requirement and replay checks. The earlier granted-second-client regression failed before the repair and passes in the current suites.
- **Go: PASS.** `go test -short ./...`; `go vet ./...`; and `go test -short -race ./modules/auth ./modules/gate ./modules/commands ./modules/plugins ./modules/webhooks ./modules/identity ./apps/api/server` all exited 0. The server race run took 396.186 s; no race report.
- **Coverage.** The short coverage run reports auth 97.5%, gate 97.3%, commands 99.5%, plugins/webhooks 100.0%, identity 99.1%, server 99.9%. The recorded `8a0c8d7` server baseline above is also 99.9%; no claim of 100% exact statement coverage is made. The uncovered held-redelivery/revoked availability branch and defensive activation-log branch remain visible in `/private/tmp/hb310-combined.cover`.
- **Console: PASS.** `npm test`: 800/800; TypeScript/Vite build exited 0. No new dependency was installed.
- **Fresh browser QA: PASS.** `qa310p02.mjs` S/E/F-en/F-ko/F-ar: 125/92/18/13/13 checks passed, zero failures (261 total), with EN/KO/AR, all four target widths, real rights-loss closures, restored access and withdrawals. No unexpected browser errors. Backend-only held-redelivery and withheld-own-pending limits in the earlier record remain.
- **Mutation: PASS.** Three additional critical author-client mutants were killed: mismatch check, constructor copy, invalid nonempty configuration. Baselines passed, copied files restored by SHA, parent source unchanged and private cache removed. Detailed local evidence: `/private/tmp/hyb310-333-mutation-result.json`.
- **QA/TW re-review: PASS in inspected scope.** A separate read-only reviewer inspected fresh P02 screenshots, current copy and the synchronized glossary/separator specification. This was separate-agent review, not Claude CLI or native-language approval.
- **Format: PASS.** `git diff --check`, directly maintained changed-file line limits and two docs-format tests passed. No test, reviewer or assistant confirmed owner FIX.

Runtime: macOS arm64, Go 1.25.7, Node 26.7.0, installed Chromium headless build 1243. Raw browser artifacts are local synthetic evidence under `/private/tmp/hb-combined-qa/310/`; their hashes and candidate source hashes are recorded in `verification-evidence.json`. Temporary artifact retention is not guaranteed. The reusable earlier runner remains in the owner's external QA-runner directory; its repository packaging is not part of this ticket.

KO/AR native-language approval, assistive-technology testing, actual Linux/Windows execution and owner acceptance remain unverified. The requested Claude CLI review was unavailable because of the reported usage limit; no successful CLI review is claimed. These are pre-launch verification results, not production readiness. PR integration and merge read-back are separate stages.
