---
type: Verification
title: HYB-318 verification for the O01 create change plan handoff and the O02 base revision
description: Commands, results, browser QA, regressions, mutation and the planning cross-check for HYB-318 (HYB-300 part B), recorded by the implementer before independent review.
status: stable
tags: [HYB-318, HYB-300, O01, O02, verification, relationship.propose, base]
sources:
  - resource: ../HYB-300/plan.md
  - resource: visual-spec.md
  - resource: references.md
  - resource: ../../ui/screens/O01/spec.md
  - resource: ../../ui/screens/O02/spec.md
---

# HYB-318 verification (implementer record)

Branch `feat/HYB-318-o01-change-plan-handoff` on base `df2f36c`, worktree `/private/tmp/hb318`, machine mac_mini. The implementer recorded everything below; it is not an independent review. Root's first review round (change PASS with lows, render FAIL, TW FAIL) is folded in under "Review fixes"; the re-review is pending. Nothing has been committed.

## What changed

**Server (`apps/api/server/relplans.go`)**
- `relationship.propose` takes an optional field `base` (`commands.Integer`; the OpenAPI body schema has `x-type: integer`, and `base` is not required).
- The handler checks presence with `ctx.Fields["base"]` and parses with `strconv.ParseUint(raw, 10, 64)`. The value must round-trip through `FormatUint`, otherwise the command refuses `invalid_request` `base`.
- Order of checks: `validChange`, past date, `base` form, active revision (`dependency_unavailable`), then `base` against the active revision (`revision_conflict`, no field), then `checkIDs` and the rest as before.
- Without `base` the behaviour is unchanged.

**O01**
- "Create change plan" is an anchor styled as a secondary button. It sits in the detail action row before "View history", with a hint below.
  - Kind carried per view: `hierarchy` in the tree, `rk` in the accounts view, the group kind for a member outside groups, and `rk` in the history view (after the end note).
  - The link carries `kind`, `member`, `from` (only when the as-of date is today or later), `base` (the pinned relationship revision) and `lang`.
  - The action is absent, never disabled, in these cases: a legal entity or a node of unknown kind in the tree (R8), a node not yet loaded, a selection not on the loaded pages, the group panel (R13), history without a timeline, any state other than ready, and denied.

**O02**
- The link values are read once and removed from the URL; `lang` is kept.
- What is prefilled: the kind, the member and the effective date (today or later). The change and the target are never prefilled.
- The member field uses the list when the member is among the choices, and "Enter an ID" mode otherwise. This is decided once, after the first choices read settles, and never after a user edit.
- The handoff block (`#o02-handoff`, `live-note-rule`, `tabIndex=-1`, no live role per R7) appears between the heading and the form. It shows, as applicable:
  - the dropped-value line (R14);
  - the prefill sentence;
  - the passed-date sentence;
  - the "Base changed." notice with "Base on revision {a}".
- A read-only fact "Based on relationship revision" is the first child of the fields column.
- The base is compared with `page.active_revision` on every render, so every plans read that changes the active revision re-evaluates it (R6).
- Propose with a stale base is blocked on the client with the `eBase` summary item, which links to the button (R4).
- "Base on revision {a}" does three things:
  - it sets the base, clears the propose attempt and removes `eBase`;
  - it announces the change and moves focus to the Current heading;
  - it does not count as an edit.
- After any `revision_conflict`, O02 reloads and compares:
  - when the carried base differs from the reloaded active revision, the block shows `baseRefused` and takes focus, and no `#o02-notice` appears;
  - otherwise O02 shows the existing conflict sentence.
- Unsaved input: `work.tsx` passes the shell's `onDirty` to O02 (`changeProps`).
  - O02 reports `true` only after a user edit.
  - It reports `false` after a successful propose, synchronously before navigating (see "Defect found in QA").
  - Leaving the targets view resets an edited form after the shell guard has confirmed.
- Copy:
  - O01 `createPlan`, `planHint`, `planHintNoDate`;
  - O02 `prefilled`, `prefilledNoBase`, `baseChanged`, `baseRefused`, `useBase`, `baseAdopted`, `baseFact`, `baseActive`, `baseNotActive`, `eBase`, `fromPassed`, `linkDropped`;
  - KO uses 리비전 throughout (R11).
- `fill()` in `changes/live-model.ts` renders copy with revisions in `<bdi dir="ltr">` and dates in `<time>`. Each value is kept on one line with the word before it and the text directly after it (for example "리비전이 0에서", "1(으)로", "from 0", "to 1.", "시작일(date)이"). The plain-text `eBase` summary uses a no-break space before each number and, in KO, a word joiner before the particle.

## Commands and results

| Check | Command | Result |
| --- | --- | --- |
| gofmt | `gofmt -l apps modules` | no output |
| vet | `go vet ./...`, `GOOS=linux go vet ./...`, `GOOS=windows go vet ./...` | PASS |
| linux/386 build | `GOOS=linux GOARCH=386 go build ./...` | PASS |
| Go tests | `go test ./...` | first run: 71 packages ok, `apps/api/server` FAIL in `TestTrustConcurrentActivation` ("activations 0 0", HYB-297 code, not touched here) while browser QA was running; the next full run of the package passed (see the race row) |
| Coverage | `go test -count=1 -coverprofile ./apps/api/server` | ok, 100.0% of statements |
| Race | `-race` full package and `-race -count=40` on the base tests | PASS (see "Race and flake check") |
| Typecheck | `npx tsc --noEmit` (apps/console) | PASS |
| Console tests | `npm test` | 690 tests, 690 pass, 0 fail (after the review fixes and the N2/N3 recheck) |
| Build | `npm run build` | built |
| Comment and size rules | added lines grepped for `//` and `/*`; the largest changed file is 921 lines | only the existing `eslint-disable-line react-hooks/exhaustive-deps` directive pattern; all files are under 1,500 lines |
| Test word rule | `grep -c float apps/api/server/relplans_base_test.go` | 0 |

New Go tests (`apps/api/server/relplans_base_test.go`):

| Test | What it checks |
| --- | --- |
| `TestProposeWithBaseEqualToActive` | `base` equal to the active revision seals the plan and records base 1; a propose without `base` still seals |
| `TestProposeWithStaleBaseCreatesNothing` | Another plan is activated with injected impact evidence (revision 2), then a propose with base 1 returns 409 `revision_conflict` with no fields. The plan store and `open_plans` are unchanged, a future base (3) is also refused, and base 2 is accepted |
| `TestProposeBaseValues` | `-1`, `01`, `""`, `18446744073709551616`, `1.0` and `x` give `invalid_request` `base`; no plan is created |
| `TestProposeBaseCheckOrder` | The order of the checks: a past date before `base`; `validChange` before `base`; `base` form before the active read; `dependency_unavailable` for an unreadable active revision; `revision_conflict` before `checkIDs`; with the current base, `checkIDs` still refuses `member` |
| `TestProposeBaseAtTheHandler` | Calls the handler directly: `01`, `001` and `-1` give `invalid_request [base]`; a stale `9` or `0` gives `revision_conflict` with no fields. Over the API, `4294967296` and `18446744073709551615` give `revision_conflict` |
| `TestProposeBaseInTheCommandSchema` | OpenAPI: `base` is `x-type: integer`; the required fields stay `[action from kind member]` |

New console tests:
- `changes/live-model.test.ts`: `handoffHref` order and the omitted `from`; `readHandoff` with an unknown kind, a malformed ID, a passed date, an invalid date and bad bases (`""`, `-1`, `01`, `1.0`, `x`, 2^64, 2^53+1, `+1`), plus `0` and `MAX_SAFE_INTEGER`; `prefilled`; `baseStale`; `conflictShown` together with `react("propose","conflict")`; `fill`, including the KO gluing.
- `changes/live-render.test.tsx` (EN/KO/AR):
  - the handoff block markup with no role;
  - the prefill sentences with and without a base;
  - the changed-base notice and its only button;
  - the refused wording;
  - the dropped and passed lines;
  - the base fact, active and not active;
  - the `eBase` text.
- `organizations/live-render.test.tsx` (EN/KO/AR):
  - the link `href` and its order before View history;
  - the hint with a date, and the hint without a date for a past as-of date;
  - absence without a plan target;
  - `planNode` and `planMember` (a legal entity and a node of unknown kind are excluded).
- `app/screen-groups/work.test.tsx`: `changeProps` passes the shell's `onDirty` (registered in `package.json`).

## Browser QA

Setup:
- The API binary `hb-api318` was built from this worktree.
- HYB-318 runs used a scratch copy of the built console with its preview proxy pointed at `127.0.0.1:8831` (console 4471/4472).
- Regressions used the static proxy `qa298proxy.mjs` over `/private/tmp/hb318/apps/console/dist` to API 8833 (console 4473).
- Every run used the sample seed, except the runs marked `none`.
- Runner files are in the session scratchpad (`hb318work/qa318.mjs`, `run318.sh`, `reg318.sh`, `regall318.sh`).

| Run | Result |
| --- | --- |
| `qa318` mode A (functional, EN 1440), after the review fixes | 84 PASS, 0 FAIL, no page errors |
| `qa318` mode L (EN/KO/AR × 1440/1280/1024/390, 17 states), after the review fixes | 852 PASS, 0 FAIL, no page errors |

Earlier rounds (before the review fixes): mode A 74 PASS, mode L (six states, three widths) 207 PASS.

Mode A covers:
- **Action per view:**
  - tree: absent on a legal entity, present on a unit and an account;
  - accounts: absent without a selection, `hierarchy` by default, `payer` with `rk=payer`, absent for a selection that is not loaded, no `from` for a past as-of date;
  - groups: absent on the group panel, `view_group` for a member outside groups;
  - history: present after the end note, absent without a timeline;
  - denied user: absent.
- **The action itself:** neutral computed border, zero radius, at least 44px high, not primary, ordered before View history, hint wired with `aria-describedby`.
- **Arrival:**
  - the URL loses kind, member, from and base and keeps `lang`;
  - kind, member (list mode) and date are prefilled; change and target are not;
  - the prefill sentence is shown, with no base notice when the base equals the active revision;
  - the block has no role and receives focus;
  - the base fact reads "1 · active" and is first in the fields column;
  - the block has neutral borders and sits between the heading and the form.
- **Back:** restores the O01 URL with `sel=sample-acct-01`, the panel and the action, with no prompt.
- **Unsaved input:**
  - after a user edit, Back prompts once ("Discard unsaved input?"); dismissing keeps the form.
  - The next Back leaves the app and gets the browser's own `beforeunload` prompt (one prompt per attempt). This is the existing `shared/navigation.ts` behaviour: after a dismissed Back, the history entry is replaced, so the next Back goes past O01.
- **Base equal:** the propose body carries `"base":"1"`; impact view; no prompt after success.
- **Base changed (link `base=0`):**
  - notice text, the button "Base on revision 1", the fact "0 · not active; revision 1 is active", neutral borders;
  - Propose is blocked by `eBase` with no request sent; the summary links to `#o02-use-base`;
  - after "Base on revision 1": the notice is removed, the fact is active, focus is on Current, the change is announced, `eBase` is cleared, there is no prompt, and the propose sends `"base":"1"`.
- **Refused propose:**
  - the first plans read was intercepted to report `active_revision` 0, so the client saw no stale base;
  - the server refused `base` 0 with `revision_conflict`;
  - after the reload the block shows `baseRefused`, with no `#o02-notice` and no summary;
  - focus is on the block and the entries are kept;
  - "Base on revision 1" followed by Propose succeeds.
- **Invalid link values:**
  - `kind=bogus`, `member=bad id`, `base=01` and a passed `from`: one dropped line, the passed-date sentence, an empty date, no base fact, no prefill sentence, no kind, and the values are removed from the URL;
  - `base=` (empty) with `member=sample-acct-99`: the dropped line, the prefill sentence without a revision, Enter-an-ID mode, payer checked.

Mode A also checks, after the review fixes: no computed outline on `#o02-handoff` after arrival and on `#o02-current-title` after "Base on revision 1" (M1), and the new EN copy. Screenshots: arrival, adopted, Back to O01, denied.

Mode L states and checks:
- O01 states: the accounts, history and tree actions; a legal-entity tree node (action absent, R8); the group panel (absent, R13); a member outside groups (present); loading (organizations reads held open: absent); error (organizations reads answered 500: absent).
- O02 states: the arrival with a stale base; the `eBase` summary; a bad link; the stale base again; after "Base on revision 1" (notice removed, focus on Current, no outline); the one-option member select before the choices load (reads held open: two options, the prefilled ID selected, the loading line shown).
- Below 1280 O01 shows the detail as a step-in view, so the runner opens it by selecting the row at 1024 and 390.
- Each O02 state with the block focused: computed outline none (M1).
- Each state: no digit at a line end followed by a Hangul particle on the next line, and no number on a different line from the word before it (M2, L4), checked per character inside the handoff block, the base fact, the summary, the O01 hint, the panel facts and the button.
- Each state: `dir`, no horizontal scroll, and the layout checker (overlap, clipping, split dates, broken words, cells cut at their wrapper, text escaping its box).
- Neutral computed borders on the action, the block and the button.
- The base button keeps its natural width at 768px and above.

### Defects found in QA and fixed

1. **A prompt after a successful propose.** After a successful propose of an edited form, the shell guard prompted "Discard unsaved input?": the guard closure still held `dirty=true` when O02 navigated to the impact view. Fixed by `flushSync(() => { setTouched(false); onDirty(false); })` before `onView`. Retested in mode A ("no prompt after a successful propose").
2. **A gap inside the button label.** "Base on revision 1" rendered the `bdi` as a separate flex item with a gap. The label is now one `span`.
3. **A split date in the KO hint.** At KO 1280 the hint's date split across lines. `time` and `bdi` in the hint and the block no longer wrap.
4. **A number apart from its particle.** In KO at 390, the revision number wrapped away from its particle ("0 / 에서"). `fill` now keeps the following text on the same line as the value.
5. **A number apart from its word in the summary** (found by the new M2/L4 check). KO `eBase` at 390 wrapped "활성 리비전은 / 1입니다". The `eBase` strings now use a no-break space before each number in all three languages.

## Review fixes (2026-11-18, root's `hb318-fixes.md`)

| Item | Change | Evidence |
| --- | --- | --- |
| Render M1, persistent outline | `#o02-handoff:focus` and `#o02-current-title:focus` have `outline: none` (programmatic, non-tabbable targets only); interactive controls keep the global focus token | mode A and L outline checks, 0 FAIL |
| Render M2, KO number and particle | `fill` keeps the word before a value, the value and the text directly after it in one `nowrap` span (`리비전이 0에서`, `1(으)로`, `시작일(date)이`); the KO `eBase` string binds particles with U+2060 and numbers with a no-break space | unit tests for `fill`; mode L check, 0 FAIL |
| Render L4, EN "from 0 to / 1." | the same rule keeps "from 0" and "to 1." together | mode L check |
| Render captures | group panel, outside member, legal-entity node, loading, error, denied, one-option member select, adopted base, Back to O01, 1024 layout | mode A and L screenshots in `hb318work/shots` |
| Change F1/F2 | `formStep` (edit, adopt, proposed, left), `adopted`, `baseBlocks`, `baseField`, `memberMode` and `afterConflict` are pure functions in `changes/live-model.ts`, used by `Live.tsx` and unit-tested, including both conflict branches | `live-model.test.ts`, four new tests |
| Change F5 | O02 reports `onDirty(false)` on unmount | code |
| Change F3/F4 | no change. The client accepts a carried revision only up to `Number.MAX_SAFE_INTEGER` (16 digits, safe integer); a larger link value is dropped as not valid, so no base is carried and the server applies its usual check | `readHandoff` tests (2^53+1 dropped) |
| Recheck N3, the Current aside | The aside and the member choices come from the same `readChoices` result (`kindMembers = choices.byKind[kind]`). Before that read settled, the aside said "Not in the loaded listing for this relationship". It now shows the loading sentence until the read settles (`currentView` in `live-model.ts`) | model test; layout run on the pending-choices state at EN/KO/AR × 1440/390: 42 PASS, 0 FAIL |
| Recheck N2, the loading line | `.o02-choices-loading` has a 24px gap below it, measured as 24px before "Relationship" | same run |
| TW | every proposed EN/KO/AR string adopted (`planHint`, `planHintNoDate` with the lower-case kind in EN only, `prefilled`, `prefilledNoBase`, `baseChanged`, `baseRefused`, `useBase` AR, `baseAdopted`, `baseFact` AR, `baseNotActive` KO, `eBase`, `fromPassed`, `linkDropped` KO); glossary rows updated and rows added for the prefill without a revision and the passed link date | render and model tests updated; mode A quotes the new EN strings |

## Regressions (this build, sample seed unless noted)

| Screen runner | Groups | Result |
| --- | --- | --- |
| P01 `qa283full` | X | 46 of 46 pass |
| P02 `qa285fullon297` | A B C D | 162, 32, 10, 16 pass; 0 fail |
| P03 `qa287fullon297` | R A B C | 288, 23, 12, 24 pass; 0 fail |
| P04 `qa289fullon297` | R A B C D | 559, 28, 20, 46, 11; 0 fail |
| P05 `qa315bodyon297` | R F A B C1 C2 C3 P, and E with seed none | 481, 132, 17, 9, 2, 3, 30, 12, 142 pass; 0 fail |
| O02 `qa296` | X | 209 pass, 0 fail |
| O01 `qa300` | A, P, C (C with seed none) | 423, 72, 55 pass; 0 fail |
| O01 `qa300` mode B | harness: this tree plus 130 synthetic accounts, `org_filter_scan` 100, the listen address changed to 8834 in a scratch copy | 182 pass, 0 fail |

- The table is the run after the review fixes.
- In the first round P04 C failed once ("success focuses the notice") and passed 46/46 in two reruns; it passed in this run.

## Mutation (changed Go lines)

Method:
- 16 mutants were applied to `relplans.go` in an rsync copy (`hb318work/mut`), each run with `go test -run 'TestPropos|TestRejected|TestStale|TestPlan|TestOpenPlan|TestCommand|TestOpenAPI|TestCatalog'`.
- The copy's GOCACHE was removed afterwards.

First round:
- 12 of 16 mutants were killed.
- Three survivors were killed by new tests:
  - "drop canonical check" and "conflict with field": by the direct handler test;
  - "parse base 32": by the wide bases;
  - "field as text": by the schema test.

Second round: 15 of 16 killed.

| Mutant | Result |
| --- | --- |
| drop `pinned` in the parse check; invert or drop the canonical check; parse check never; refusal without the `base` field | killed |
| drop `pinned` in the conflict check; invert the conflict; conflict never; conflict code changed; conflict with a field | killed |
| `ParseUint` with 32 bits | killed |
| field removed; field as Text; field required | killed |
| conflict check moved after `checkIDs` | killed |
| `err != nil` dropped | survives; equivalent |

Why the survivor is equivalent:
- On a syntax error `strconv.ParseUint` returns 0, and on a range error it returns `MaxUint64`. Neither formats back to the raw text that failed (`"0"` and `"18446744073709551615"` parse without error), so the `FormatUint` round-trip refuses every input the error check refuses.
- The only reader of `err` here is this condition.
- The check is kept for readability; root may decide to drop it.

## Race and flake check

| Command | Result |
| --- | --- |
| `go test -race -count=1 ./apps/api/server` | ok (399.7s). An earlier attempt was ended by an external SIGTERM at 219s and is not counted |
| `go test -race -count=40 -run 'Base\|Propos' ./apps/api/server` | ok (101.0s) |
| `go test -count=100 -run TestTrustConcurrentActivation ./apps/api/server` | this branch: 3 of 100 iterations fail. Base `df2f36c` (from `git archive` into the scratchpad): 19 of 100 fail, with the same "activations 0 0" and two 401 list failures |

`TestTrustConcurrentActivation` is a pre-existing HYB-297 flake, and HYB-318 does not change it. It also caused the first `go test ./...` failure above. Root may want a separate ticket for it.

## R8 finding (legal entity as a hierarchy member)

Fact: in the organizations domain a legal entity cannot have a parent. `PutParty` refuses a legal entity with a `ParentID` (`modules/organizations/registry.go:125-128`), and an internal unit must have a legal entity parent (`:130-140`).

Fact: the relationships registry and `checkIDs` (`apps/api/server/relplans.go:199-245`) do not refuse a hierarchy timeline whose member is a legal entity. A probe test on this branch (removed after the run) proposed `hierarchy` for the member `sample-le-north` with the targets `sample-le-south` and `sample-unit-south-web`. Both returned 200 `sealed`.

Decision taken here (visual spec R8): the action is absent on legal-entity nodes; units and accounts get it.

Server gap, outside Part B's allowed lines (`relplans.go`: base field and check only), recorded for root:
- `relationship.propose` accepts a legal entity as a hierarchy member, and the hierarchy validation only checks cycles and depth.
- A follow-up should refuse it (`invalid_request` `member`) or add a hierarchy rule in `modules/relationships`.

## Planning cross-check

| Planning source | Requirement | Covered by | Status |
| --- | --- | --- | --- |
| `O01/spec.md:72` | "변경 계획 만들기 · 관계 변경 권한 AM · 대상 revision 확보 · O02 변경 초안" | O01 action in every view except the group panel; the pinned relationship revision is carried as `base` and held by the server check (`revision_conflict`); O02 opens a draft form | covered. The permission is installation administrator, as in HB-272 (D4, Q2, residual). The O02 form is a browser draft until Propose |
| `O01/spec.md:73` | 조회 그룹 초안 | Part C (HYB-319) | not in HYB-318 |
| `O01/spec.md:81` | O02로 이동하며 canonical 대상 ID·기간·revision만 전달; 수신 화면에서 권한 재검사 | The link carries only kind, canonical member ID, date and revision (names, target and action never). O02 reads them once, validates them (R14), runs its own reads and commands with their own permission checks, and shows the existing denied block on refusal | covered |
| `O01/spec.md:79` | 미저장 초안은 이탈 영향을 안내 | the shell guard through `onDirty`; one prompt per attempt; no page-level `beforeunload` added | covered for O02 |
| `O01/spec.md:102` | revision 고정, 대상 변경을 조용히 갱신하지 않음 | the base never moves without "Base on revision {a}"; it is compared on every plans read | covered |
| `O02/spec.md:49` | 변경 종류 enum, 연결/해제/이동/그룹 | the prefilled kind is editable; the change is never prefilled | unchanged by Part B; group is Part C |
| `O02/spec.md:50` | 고정 대상 manifest | the single-member plan as before; the group manifest is Part C | unchanged |
| `O02/spec.md:51` | 기대 revision, 읽기 전용, 동시 변경 시 충돌 | `base` is read-only (fact) and is sent as `relationship.propose` `base`; the server refuses `revision_conflict`; O02 shows the base notice or the plain conflict sentence, never both. The API expected revision is unchanged | covered |
| `O02/spec.md:52` | 적용일 | prefilled from the as-of date only when today or later; otherwise `fromPassed` | covered |
| `O02/spec.md:82` | 수신 화면에서 권한 재검사 | O02's own reads and commands | covered |
| HYB-300 plan, Part B server | optional Integer `base`, presence, canonical `ParseUint`, check order, `revision_conflict` with no field | `relplans.go`; Go tests and mutation above | covered |
| HYB-300 plan, Part B console | the O01 button per view; the O02 prefill; the base notice; refusal handling; `onDirty` | as above. The plan's `role=status` prefill note is superseded by R7 (focus, no role) | covered with R-decisions |
| HYB-300 plan, Part B tests | the Go and console test list | all listed tests exist; "an activation between O02's load and propose" is `TestProposeWithStaleBaseCreatesNothing` | covered |

## Deviations and notes for root

- **Copy.** The TW proposals are adopted; KO now names the destination 조직 변경 계획. The visual spec §10 is root's to update.
- **Older pinned revision (R9).** It is not exercised in the browser: the sample seed has only revision 1 active, and revision 0 is empty, so no member exists to select. The O01 code carries `base.revision` (the pinned revision). The base-changed path was exercised with a hand-edited `base=0` and with an intercepted plans read.
- **Leaving the targets view.** When the view leaves the targets view, an edited form is reset (the shell has already confirmed the discard), and the handoff block is cleared.
- **Remaining race.** The `onDirty` report is an effect. A Back pressed within the same frame as a keystroke could beat it. QA waits 400ms after typing.

## Residuals

- Admin-only permission (D4).
- The server accepting a legal entity as a hierarchy member is tracked as HYB-322.
- Native-language KO/AR review of the new copy.
- Re-review of the review fixes (render, change, TW) by root's reviewers.

## Root final verification (2026-11-18, main df2f36c)

- Root re-ran `gofmt`, `go vet ./apps/api/...`, `go test -cover ./apps/api/server` (100.0%), `tsc`, `npm test`, build; no changed file over 1,500 lines.
- Reviews: change review PASS (lows F1, F2, F5 fixed; F3, F4 noted); render review FAIL then PASS (M1 focus outline, M2 number and particle, L1–L4, N2 gap, N3 premature "not in listing"); technical-writing review FAIL then PASS.
- Residuals: the server accepts a legal entity as a hierarchy member (HYB-322); R9 shown only with a hand-edited or intercepted base because the seed has one revision; KO and AR copy remain drafts for native review; `TestTrustConcurrentActivation` flakiness is tracked in HYB-321.

