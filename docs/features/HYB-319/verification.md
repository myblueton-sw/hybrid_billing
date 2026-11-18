---
type: Verification
title: HYB-319 verification for O01 view group drafts and the O02 group draft handoff
description: Commands, results, browser QA, regressions, mutation and the planning cross-check for HYB-319 (HYB-300 part C), recorded by the implementer before independent review.
status: final
tags: [HYB-319, HYB-300, O01, O02, verification, view_group.draft, view-group-drafts]
sources:
  - resource: ../HYB-300/plan.md
  - resource: visual-spec.md
  - resource: references.md
  - resource: ../HYB-318/verification.md
  - resource: ../../ui/screens/O01/spec.md
  - resource: ../../ui/screens/O02/spec.md
---

# HYB-319 verification (implementer record)

Branch `feat/HYB-319-o01-view-group-drafts` on base `1cf7f3f`, worktree `/private/tmp/hb319`, machine mac_mini. The implementer recorded everything below. It is not an independent review, a render PASS, a TW review or owner acceptance. Nothing has been committed.

## What changed

**Server (`apps/api/server`)**
- New `groupdrafts.go`: the in-memory draft store (`vgd-` + 16 hex IDs, `group-` + 16 hex group IDs from the server, never from the name), derived state, the workspace-scoped label map, pruning and the two reads.
- New `groupdraftcmds.go`: `view_group.draft`, `view_group.edit`, `view_group.discard_draft` and `view_group.propose`. All four use the installation administration permission, are mutating with a rationale, resolve only in the boot workspace and take the API revision as the expected revision.
- `relplans.go`:
  - `planMeta.draft`;
  - `planRow.draft` (`{id, name, group, new_group, member_count}`, `null` for a single-member plan);
  - `s.label` for member and target names;
  - the blocked reason moved into `blockedOf` (shared with the drafts read);
  - draft, chunk and seal moved into `sealPlan`, which chunks by `TimelinesPerChunk` (used by both proposes);
  - a successful activation calls `draftActivated`.
- `organizations.go` and `orgtargets.go`: label lookup only. The targets read also matches `q` against the label map.
- `orgsettings.go`: four settings, per-field defaults for values not given, the cross-setting checks and `orgSettingRows` (reusing `settingRow`).
- `server.go`: the draft store and a `relLimits` field (the registry is built from it; tests inject small limits).
- `boundary.go`: two GET routes. `admin.go`: the definition list.
- `trusthold.go` and `trust_hold_test.go` (added to the allowed files by root): `view_group.discard_draft` is `holdAllow`/`reasonPending`, because it withdraws a pending proposal. `view_group.draft`, `edit` and `propose` are `holdRefuse`/`reasonChange`, because they change organization state. The test's expected count is now 77.
- `apps/api/cmd/hb-api/main.go`: `settingNumber` accepts six digits, because `view_group_activated_kept` allows up to 100000. The flags themselves come from `OrgSettingList()`.

**Reads**
- `GET /api/v1/organizations/view-group-drafts?after=&limit=` (1–50, default 50) returns: `revision`, `active_revision`, `settings` (every `OrgSettings` row with its source), `open`, `open_limit`, `drafts`, `next`.
  - Each draft row has `id`, `name`, `group`, `group_label`, `new_group`, `member_count`, `members` (first 50), `author`, `author_login`, `created`, `updated`, `discarded`, `state`, `activated_revision`, `pruned` and `plan`.
  - `plan` is `{id, state, closed, blocked, rule}` or `null`.
  - Order: open and proposed drafts first, then the rest, newest first.
  - An unknown `after` is `invalid_request`.
- `GET /api/v1/organizations/view-group-drafts/{id}/members?after=&limit=` (1–99, default 50): `{revision, draft, member_count, pruned, members, next}`. An unknown draft, or one in another workspace, answers `unsupported_command`.

**Console**
- `admin.ts`:
  - the four command IDs;
  - `viewGroupDrafts()`, `draftMembers()` (`"missing"` on `unsupported_command`) and `groupDraftOf()`;
  - `PlanRow.draft`.
- O01, new `drafts.tsx` and `drafts-model.ts`:
  - `useDraftList`, `useDraftEditor`, `DraftsSection`, `SelectionBar`, `RowCheck`, `HeaderCheck` and `DraftPanel`;
  - the model covers allowed actions per state, state blocks, 31-member chunks, edit-call ordering, pending adds, ID parsing, the name rule, the dirty test, bounded selection, links and refusal mapping.
- O01 `Live.tsx`:
  - page selection in the accounts table and the not-in-a-view-group list;
  - the selection bar in the list-head end slot;
  - the drafts table in the groups view (view group kind only);
  - the draft panel overriding the detail panel, with `data-mode="draft"` widening it at 1280px and wider;
  - "Return to the draft" after step-in Back;
  - selection cleared with an announcement when the list key changes;
  - `onDirty`.
- O02 `Live.tsx`:
  - `draft` in the link (`readHandoff`);
  - the `GroupHandoff` block and a form with only the effective date, the reason and Propose;
  - `view_group.propose` with `{from, base}`;
  - refusal mapping;
  - the group plan lead with "Draft name:";
  - a four-column member table for group plans;
  - group plan titles in the open and closed lists.
- `work.tsx` passes `{...changeProps(p, service)}` (including `onDirty`) to O01, as it already did for O02.

## Batching and edit ordering (R1, R4)

- **New draft.** Call 1 is `view_group.draft` with up to 31 members and the name or group. The remaining members follow as `view_group.edit` `add` calls of up to 31.
- **Edits.** `editCalls` puts up to 31 adds and up to 31 removes in each call.
  - Before each call it lowers the removes so that the count stays at least 1, then lowers the adds so that the count stays at most the cap.
  - The name goes in the first call.
  - If no order can keep every intermediate count between 1 and the cap, the client refuses before sending. The model test covers 40→1, 1→61, 50/40/40 at cap 50, and 1+40−1.
  - This matches the server rule, which checks the result of each `edit` against 1..`view_group_draft_members`.
- **Expected revision.** Each call reads `viewGroupDrafts("", 1)` first for the current API revision, because a command outcome does not carry the result revision to the client.
- **Progress** is "{done} of {total}" with the exact client-held denominator.
- **Partial failure.** The draft exists, the unsaved members stay pending, the alert states the saved and unsaved counts, and "Try the remaining {rest} again" retries them.

## Commands and results

| Check | Command | Result |
| --- | --- | --- |
| Format | `gofmt -l apps modules` | 0 files |
| Vet | `go vet ./...`; `GOOS=linux go vet`; `GOOS=windows go vet` (apps, modules) | clean |
| 386 build | `GOOS=linux GOARCH=386 go build ./...` | ok |
| Go tests | `go test -count=1 ./...` | every package ok |
| Coverage | `go test -cover ./apps/api/server` | 100.0% of statements |
| Race | `go test -race -count=1 ./apps/api/server ./modules/relationships` | ok (337 s and 13 s) |
| Coverage | `go test -race -cover ./modules/relationships` | 100.0%, race clean |
| Console | `tsc --noEmit`, `npm test`, `npm run build` | clean; 715 tests, 715 pass; build ok |
| Comments | added `//` lines | only the existing `eslint-disable-line react-hooks/exhaustive-deps` directive pattern |
| Line counts | largest changed files | `changes/Live.tsx` 1045, `admin.ts` 1025, `organizations/Live.tsx` 989, `drafts.tsx` ≈740; all under 1,500 |

**Go tests added**
- `groupdraftcmds_test.go`:
  - the lifecycle (create, edit, propose, held, activation with injected impact evidence, `Union`, labels before and after activation, a second draft never changing the label);
  - a draft on an existing group;
  - the refusal table (name, group, members: party, group ID, pool ID, legal entity, unknown, repeated, blank line, 32 per call, the group itself), edit refusals, propose refusals (date, base forms including "01", "-1", 2^64 and empty, a stale base);
  - limits (member cap, open limit; proposed counts, discarded does not, a discarded plan reopens the draft);
  - entries per timeline (a full member outside the group is refused on draft, edit and propose; a full member already in the group is accepted);
  - `no_change` and `plan_limit` rejections with the draft staying open;
  - two-chunk sealing with injected limits;
  - faults (parties, roles, active revision, plan draft, plan chunk, plan state);
  - new group collisions (`invalid_stage`);
  - an existing group gone (`invalid_request`, no field);
  - the open-plan limit and expired plans discarded first;
  - roles beyond the sample shape (a party with a timeline; a contract pool target with its own timeline);
  - resolve and direct `Apply` (long rationale, a non-canonical base reaching the handler, plan record pruning after a draft command).
- `groupdrafts_test.go`:
  - read order, paging, limits and settings rows with sources;
  - the members read paging and limits;
  - unknown and other-workspace drafts;
  - labels and pruning (activated bodies pruned, labels kept, discarded drafts pruned to the setting);
  - state from the registry after the plan record is pruned (more than 200 closed plans);
  - **grants nothing**: the access read's grants and delegations, an access preview decision for the member, and the member's organizations and drafts reads (403) are unchanged after draft, edit, propose and an activation with impact evidence;
  - helpers;
  - targets matched by label while paging on a pinned revision.
- `orgsettings_test.go`: bounds for each new setting, the cross checks (ordered before the bounds), a given zero kept and refused, partial defaults.
- `main_test.go`: the new flags, their bounds and usage text, and six-digit numbers.

**Console tests added**
- `drafts-model.test.ts` (10 tests).
- `drafts-render.test.tsx` (7 tests, EN/KO/AR):
  - the drafts section, the selection bar, the checkboxes, the new draft form, the panel per state;
  - a draft name never in the Group column;
  - no page-level `beforeunload` in `organizations/Live.tsx`, `drafts.tsx` or `changes/Live.tsx`;
  - `work.tsx` passes `changeProps` to both pages.
- `changes/live-model.test.ts`: draft links, plan titles, draft load states, refusal mapping.
- `changes/live-render.test.tsx`: the group handoff states, and the group plan lead with the draft name only as "Draft name".
- `admin.test.ts`: the two reads and `PlanRow.draft`.
- `work.test.tsx`: O01 receives `onDirty`.

## Browser QA

Runner: `scratchpad/hb319work/qa319.mjs`. It is assembled from the HYB-318 head, which carries the overlap, clipping, outside, time-split, broken-word, cut-cell and horizontal-scroll checks. The body adds number/particle splits, ID and date splits, neutral borders and the no-outline check on programmatic focus targets.
- Setup: `hb-api319` with the sample seed, behind `qa298proxy.mjs`, serving the built `dist`.
- Ports: console 4471–4474, API 8831–8834.
- Screenshots: `scratchpad/hb319work/shots*`.

**Functional flow (MODE F, EN 1440): 46 PASS, 0 FAIL**
- Checkbox names carry the ID.
- The bar reads "None selected", then "2 selected of 6 shown". The header checkbox shows the indeterminate state. A checked row has no tint. The bar is neutral.
- A search clears the selection, shown in the bar and announced.
- Start moves the members into the form, announces it and focuses the title. The panel widens. The note comes first. The programmatic focus target has no outline.
- Validation errors appear in a summary.
- A tab change while dirty raises the app guard, and dismissing it keeps the input.
- Selecting a row while dirty shows the inline notice, focused. Keep editing returns focus.
- Saving gives the draft panel, focused and announced, after exactly one `view_group.draft`.
- Add 1 to the draft, then Remove (marked, Undo) gives the unsaved summary and hides Create change plan. Save changes is announced.
- The discard confirmation takes focus, and Keep draft returns it.
- The Create change plan href carries `draft`, `from`, `base` and `lang`.
- In O02:
  - the link values are cleared;
  - the handoff shows the title, the draft name, the note, the effect sentence and "Members (first 2 of 2)", and is focused without an outline;
  - the form has no kind or member fields.
- Propose gives the impact view with "Join view group group-…: 2 members from …" and "Draft name: Ops reviewers".
- Back in O01, the draft is held and offers no plan or discard action. The drafts table shows "Change plan open". The group table never shows the draft name.
- Discarding the plan in O02 reopens the draft with "Last plan discarded". Proposing again works.
- The inline notice's Discard switches drafts.
- The page registers no unload handler.

**Layout matrix (MODE L, EN/KO/AR × 1440/1280/1024/390, 31 states)**
- States:
  - accounts none, selected, cap (patched setting, unchecked boxes disabled), open limit (Start absent);
  - new form, form errors, saving progress (delayed command), draft open, draft dirty, leave notice, discard confirmation;
  - stage notice, dependency notice, partial save (mocked: 31 of 41 saved);
  - groups table, outside list selected, return link (narrow);
  - O02 group handoff, stale base, refused (mocked 422 without fields), missing draft, group plan detail;
  - draft held, rejected (`plan_limit`, patched), activated and pruned (patched), discarded (patched), gone (mocked 404), drafts read failed (no checkboxes), drafts read loading.
- Patched and mocked states are UI-state captures over the real server, with the read or command response replaced in the browser. They are not server behaviour; the server paths are covered by the Go tests.
- First run: 2374 PASS, 224 FAIL. The defects found and fixed:
  - the drafts table was wider than the list in draft mode (cut cells and horizontal scroll). It now switches to stacked rows through a container query below 760px of its own width;
  - the rejected sentence's nowrap span swallowed the rule sentence (text escape and horizontal scroll). The rule sentence is now inserted as plain text;
  - Created and Last changed stamps were split. Stamps are nowrap, and "· stamp" is kept together;
  - revision numbers in mono were reported split from their word. They are plain `bdi` now, as in Part B;
  - member rows put the ID and the "Will be removed" label in one text block. Rows are now a grid with the text in its own paragraph;
  - O02 group plans: the five-column member table broke group IDs, and the lead split "구성원 2개" and the AR date. Group plans now use a four-column table with nowrap IDs, and the lead is built with `fill`;
  - the stacked drafts table cell padding.
- Re-run on 1440/1280/390 × EN/KO/AR: 1980 PASS, 0 FAIL.
- Final full matrix (4 widths × 3 languages × 31 states, final build): **2598 PASS, 0 FAIL**. Final functional flow: **46 PASS, 0 FAIL**.
- **Excluded from the number check, with a reason:** the shared Reason hint ("Up to 256 bytes (about 256 Latin letters)") from `identity/pages/users/Live.tsx` splits "(about" from "256" at 1280 in EN. That component is shared with O02 and other screens and is outside this ticket's files. Recorded under Residuals.

**Proof that the checks fail on broken builds**
- Injected CSS (MODE PROOF), against a passing build: neutral border (coloured selection bar), no-outline (outline on the draft title), number split and number/particle split (KO), text escapes, outside the root, horizontal scroll and broken word each produced FAIL lines.
- On real defects: time split, ID and date split, cut cell and horizontal scroll were each shown failing by the first matrix run on the defective build (above), and pass after the fixes.

## Regressions (this build)

Runner `scratchpad/hb319work/regall319.sh` (proxy 4473, API 8833, `hb-api319`), using the HYB-297-updated runners and the HYB-318 P05 and qa318 runners.

| Suite | Result |
| --- | --- |
| P01 (`qa283full`) | 46 checks, "FAILS 0 OF 46" (the wrapper's fail counter matches that summary line) |
| P02 (`qa285fullon297`) A/B/C/D | 162/32/10/16 PASS, 0 FAIL |
| P03 (`qa287fullon297`) R/A/B/C | 288/23/12/24 PASS, 0 FAIL |
| P04 (`qa289fullon297`) R/A/B/C/D | 559/28/20/46/11 PASS, 0 FAIL |
| P05 (`qa315bodyon297-full318`) R/F/A/B/C1/C2/C3/P, E (no seed) | 481/132/17/9/2/3/30/12, 142 PASS, 0 FAIL |
| O02 `qa296` | 209 PASS, 0 FAIL |
| O01 `qa300` A / P / C (final build) | 423 / 72 / 55 PASS, 0 FAIL |
| O01 `qa318` A / L (final build) | 84 / 876 PASS, 0 FAIL |

- The first `qa300` run on this build found real regressions: the new checkbox column cut "Organization since" at 1280 with qa300's data (A: 4, P: 2), and a KO conflict label broke inside a word at 1280.
- Fixed by:
  - 3px inline cell padding on the checkable accounts table between 768px and 1439px;
  - keeping the KO conflict label on one line.
  The checkbox column keeps 44px.
- `qa300` also asserted the exact header list. The new checkbox header has no text, so the runner copy `hb319work/qa300on319.mjs` filters empty headers. That is the only runner change. P01–P05 and O02 ran on a build that differs from the final one only in the O01 accounts-table class and CSS.

## Mutation (changed Go lines)

Method: an rsync copy with a session GOCACHE removed after each batch. `go test -failfast -skip TestTimingIndistinguishable ./apps/api/server` (the timing test is skipped so that machine load cannot fake a kill).
- Batch 1: 96 mutants. 69 killed, 7 invalid (the mutation did not compile because of unused variables), 13 survived.
- Batch 2: the 7 invalid mutants rewritten to compile, and 7 survivors re-run after new tests. All 14 killed.
- New tests written to kill survivors:
  - 32 well-formed lines refused;
  - a party with a timeline that is not a target;
  - a contract pool target with its own timeline;
  - direct `Apply` with a long rationale on discard;
  - a non-canonical base at the handler;
  - plan record pruning after a draft command;
  - expired plans discarded before a group propose.
- Six survivors in round 1. After the review (L1), `base required` and `members required` are killed by the new refusal cases (see Review round 1), which leaves 4 argued:
  - `prune skip pruned`: re-pruning an already pruned activated draft writes the same empty values. Pruned drafts are always the oldest by close order, so the newest kept set does not change. No reader can tell the difference.
  - `members ws` (the members read using the sample workspace instead of the answer's workspace): the server boots one workspace, and every session's answer names it. A second-workspace session cannot be made in this build, so the mutant matches the original on every reachable path. A direct test proves that a draft stored in another workspace is not served. The same argument was recorded for the Part A reads.
  - `cross members` and `cross scan` (dropping the cross-setting checks): the per-setting bounds (members ≤ 10000 = `TimelinesPerPlan`; scan ≥ 100 > 99) refuse the same values with the same setting name. The checks are kept because the plan requires them and they protect the relation if the bounds change.
- Result: 90 of 96 killed; 6 argued above.

## Planning cross-check

| Source | Requirement | Covered by | Status |
| --- | --- | --- | --- |
| O01 spec line 50 | Organization ID and display name: canonical ID references, the name is a draft ("명칭 초안"), never a key | Group IDs generated by the server; draft names live only in drafts and plans, and become the label only on activation (M1, Q8); every handoff and link carries IDs only | covered |
| O01 spec line 73 | View group draft: group management permission, allowed members only, no automatic grant, not-ownership notice | Four commands; member rules (source accounts only, not targets or parties); the no-grant Go test; the persistent note in the form, the panel and the O02 handoff | covered; permission is installation administrator (Q2 default, residual) |
| O01 spec line 77 | Tabs: tree → accounts → groups → history | Unchanged tabs. Drafts sit in the groups view, selection in the accounts and groups views | covered |
| O01 spec line 104 | Progress with a fixed denominator and success/held/failed/unprocessed, no invented percentage | "Saving: {done} of {total}" with the client-held total; the partial alert gives saved and unsaved counts with a retry; no percentage | covered for batched saves (a client chain, not a server job) |
| HYB-300 plan Part C | Store, four commands, two reads, settings, labels, pruning, the O01 and O02 UI, tests | This record | covered; deviations below |
| Q3 | Drafts add members only | `edit` `remove` removes from the draft only; no leave path; "Remove from draft" wording | covered |
| Q5 | Settings defaults and bounds | 16 (1–200), 1000 (1–10000), 200 (50–10000), 1000 (100–100000), with start flags and sources shown in the drafts read | covered |
| Q8 | The draft name becomes the label only on activation; no rename of existing groups | Label map written once on activation for new groups only; `name` refused on existing-group drafts | covered |

## Deviations and notes for root

1. **Propose refusals for `members` and `group` carry no field.** The command registry copies refusal fields only when they are declared fields of the command (`modules/commands/execute.go:401-411`). `view_group.propose` declares only `from` and `base`, so `invalid_request` `members` or `group` reaches the client field-less. O02 shows "This draft can no longer become a change plan as it is…" for a field-less `invalid_request`. The Go tests assert the field-less code. The plan text ("`invalid_request` `members`" and "`invalid_request` `group`" at propose) needs this note.
2. **An unknown draft ID on a command** answers `permission_denied`, not `unsupported_command`, because the registry does not distinguish a target that is not found. After the review (L3), the O01 client reloads the drafts read on `permission_denied`, `invalid_stage` or `revision_conflict` from an edit, discard or propose. When the draft is no longer listed it shows "This draft no longer exists". `refusalOf` also maps `unsupported_command` to "missing" (members read) and shows the same sentence. Browser test MODE R "L3".
3. **The reason is cleared after each successful save or discard**, so every command gets its own rationale. Discard uses the reason field rendered in the open draft panel.
4. **The drafts table switches to stacked rows by container width (below 760px)**, so the R2 wide panel can stay at 1280. IDs never wrap.
4a. **Accounts table width at 1280 (render B2, root decision 2026-11-18, option (a)).**
    - The 3px padding rule is removed and 8px cell padding is restored.
    - The checkbox column had rendered at 60px because a more specific padding rule overrode it. It is now 44px with no padding at 768px and wider.
    - The wide draft panel is used only in the groups view (R2 fallback). The accounts view keeps the standard panel.
    - With a sixth column (relationship kind), the EN table was still 11px wider than the 568px list at 1280. Root approved option (a): in the accounts view at 1280–1439, while checkboxes are shown, the panel is `clamp(340px, 34%, 440px)` (`data-mode="pick"`).
    - The KO conflict label is now "귀속 충돌 (대상 n개)" with a space before the parenthesis, so it wraps at the space and never inside a word.
    - Verified with `qa300` A (6-column kind variant, EN/KO/AR at 1280, 423/0) and the new cell-gap check (≥12px). The 340px panel content is checked for clipping by the layout runner.
5. **Group plans in O02 use a four-column member table** (no "Dated change" column, which repeats the lead).
6. **`OrgSettings.withDefaults` now defaults each zero field that was not given**, instead of all-or-nothing. A given zero (a flag) is kept and refused by `validate()`.
7. **`settingNumber` in `main.go` accepts six digits**, so `view_group_activated_kept` can reach its upper bound of 100000.
8. **"Add {n} to the draft"** de-duplicates against the loaded member pages of an existing draft only. An account on an unloaded page is refused by the server as `add` (shown with the members sentence).
9. **The drafts read pages from a position in the derived order.** A draft that changes group (for example open → activated) between page reads can repeat or be skipped. An unknown cursor is `invalid_request`.

## Review round 1 fixes (root `hb319-fixes.md`, 2026-11-18)

Root rebased the work onto main `e360bf7`. That covered `admin.go` definitions, the hold-test count of 82, and the `bulk_cover_test` exemption for `view_group`. These items were then applied.

**Change review**
- **SoD (owner-level, fail closed).**
  - A draft records every administrator who created or edited it (`editors`).
  - `view_group.propose` copies them into the plan (`planMeta.editors`).
  - `relationship.activate` refuses `prerequisite_failed` when the activator is the proposer or any editor (`planMeta.authoredBy`). The plans read reports `blocked: own_proposal` to those readers.
  - Tests: B edits A's draft, A proposes, B's activation is refused with impact injected; the draft's author cannot activate another administrator's proposal; an administrator who never touched the draft can activate.
- **M1.** `loadMembers` has a sequence guard, so a late page for another draft is ignored (browser test R "M1", delayed members read).
- **M2.**
  - The created draft ID is kept in the partial state (`draftId`) independently of `list.find`.
  - "Try the remaining N again" and Save both continue that draft with `view_group.edit` `add`.
  - Browser test R "M2" (mocked create, a failing first edit, the drafts read omitting the new draft): exactly one `view_group.draft`, and every edit targets the created ID.
- **M3.**
  - Before saving adds, the client pages the members read (99 per page) and drops IDs already in the draft.
  - It then looks up the remaining IDs as source accounts (`organizations?subject=account&member=…`, 50 per call) and drops the others.
  - The notice states both, for example "1 were already in this draft and were not added. Not a source account, so not added: sample-le-north.", and it stays after the save.
  - Browser test R "M3": the edit sends only `sample-acct-05`.
  - Refused batches name their cause with `eMembersServer`.
- **L1.** New refusal cases kill the two `Required` mutants: `view_group.draft` with no fields gives `invalid_request [members]`; `view_group.propose` without `base` on a proposed draft gives `invalid_request [base]`.
- **L2.** `planMeta.snapshot` holds `{id, name, group, new_group, member_count}` from the propose, so the plans read keeps the draft block after the draft is pruned or deleted. Tested after pruning.
- **L3.** See deviation 2.
- **L4.**
  - The first call of a save uses the loaded drafts-read revision; later calls read a fresh one.
  - On `revision_conflict` the client reloads, keeps the pending changes and shows "This draft changed before your save…".
  - If the reloaded draft is no longer open, it states how many unsaved changes were dropped (`draftChangedClosed`).
  - Browser test R "L4".
- **L5.**
  - Discard has an in-flight guard (one POST for a double click, browser test R) and shows "Saving…" while it runs.
  - `finish` does not reset the panel when the user moved to another draft or form while saving (generation guard, browser test R).
- **L6.** Added to Residuals.

**Render review**
- **B1.** Below 768px the checkbox cell is auto width and starts at the inline start, and "Select" never wraps.
- **B2.** See deviation 4a.
- **B3.** O02 group plan ranges render each date in its own nowrap `<time>`; only "From" may break.
- **QA checks.** The runner now checks every visible text node:
  - words broken across lines (stacked labels and table cells at every width);
  - dates split across lines in EN, KO and AR formats, inside or outside `<time>`;
  - at 1280 and 1440, a gap of at least 12px between the text of adjacent table cells.

  Each check was run first on the unfixed build and failed there: "Select" broken at 390 in EN and AR, "Dec 20, 2026" split in the O02 group plan at 1280, and cell gaps of 6–12px at 1280.
- **M1.** Drafts render as compact two-line rows when the section is narrower than 760px (ResizeObserver):
  - line 1: name link, state, "{n} members";
  - line 2, 14px muted: `vgd-…` · New/Existing group `group-…` · plan link or "None".

  The real table stays where it fits. This replaces the container-query stacked table.
- **M2 / TW 1.** The panel shows "{n} saved in this draft"; the add area shows "{n} after your changes · up to {max}" for an existing draft and "{n} to add · up to {max}" in the new form.
- **M3.** 24px above the Reason block and each h3 in the panel, and 8px below the add meta. The O02 members h3 is spaced as before.
- **m1.** Link buttons have a 44px minimum height.
- **m2.** Created shows the author and the time on two lines, without a dot.
- **m3.** The outside-list count sits in the heading slot with the h3.
- **m5.** At 390 the count is on line 1 and the buttons on line 2.
- **m6.** The drafts section's bottom hairline is removed.
- **m7.** KO held sentence per TW 2.
- **m8.** A member without a name shows its ID once.
- **m9.** The runner moves the pointer off before each capture and asserts that checked rows keep the default background.

**TW review.** Rows 1–23 are adopted as proposed (EN final where marked; KO and AR as drafts).
- Root's decision on row 20: state "Open" / "작성 중" / "مفتوحة", and `stProposed` KO "변경 계획 열림".
- Row 24: the server never names the entries-full case. `view_group.draft` and `edit` refuse `invalid_request members` or `add`, and `propose` refuses field-less (deviation 1), so `eMembersServer` covers it. `entriesFull` is not added. The new client-side causes (already in the draft; not a source account) have their own sentences.
- Hypotheses:
  - `refusalOf` now returns "missing" for `unsupported_command` (unit tested);
  - `cap` no longer subtracts the editor count for a closed draft (browser test R "cap");
  - `member_and_target` wording for view group plans is reported only (out of scope).
- Glossary: the duplicate provider and filtered-empty rows are deleted. The rows for draft state, draft name, group handoff and group plan lead are updated, and rows are added for member counts, partial save, the held wording and the members-not-in-group heading.

**Plan-text changes for root**
1. **SoD editors rule.** A view group plan carries its draft's editors. The activator must be neither the proposer nor any editor (`prerequisite_failed`).
2. **`planMeta` snapshot.** The plans read's `draft` block is the propose-time snapshot, not the live draft.
3. **M3 pre-check.** The client drops members already in the draft and IDs that are not source accounts before an add, and states both. The server remains authoritative.
4. **B2 option (a).** The accounts-view panel is `clamp(340px, 34%, 440px)` at 1280–1439 while checkboxes are shown, and the wide draft panel is used only in the groups view (visual spec §16).
5. **Compact draft rows** replace the stacked table below 760px of the section's own width (render M1).

## Review round 2 fixes (root recheck, 2026-11-18)

1. **Durable tests for the console fixes.** The decisions are now pure functions in `drafts-model.ts`, used by the editor hook:
   - `partitionPrecheck` (already in the draft / not a source account / kept);
   - `savePlan` (continue the created draft via edit / new / edit / none);
   - `isLatest` (member-page sequence);
   - `sameView` (save generation);
   - `callRevision` (the loaded revision only for the first call of a fresh save);
   - `afterRefusal` and `readOutcome` (gone only when the drafts read succeeds without the draft).

   Each has unit tests in `drafts-model.test.ts`. `drafts-render.test.tsx` checks that `drafts.tsx` calls each of them at its decision point, so reverting a fix fails a test.
2. **`own_draft`.**
   - The plans read reports `own_draft` to an editor of the plan's draft who is not the proposer; `own_proposal` stays for the proposer. Activation still refuses both.
   - O02 shows "You created or edited the draft for this plan. Another administrator must activate it." (KO/AR drafts), and the step note reads "Draft created or edited by you".
   - Go test (editor gets `own_draft`, proposer gets `own_proposal`), O02 model test and O02 render test.
3. **Neutral first-call conflict copy.** A `revision_conflict` now shows "The data changed before your save. The draft was reloaded; your unsaved changes are kept below." (`dataChanged`, KO/AR drafts). An `invalid_stage` keeps `draftChanged`. Browser test R "L4".
4. **Revoked role.**
   - A refused command is followed by a drafts read. Only a successful read without the draft shows "This draft no longer exists".
   - A denied read shows the denied sentence, and an unauthenticated one shows "Sign in again".
   - Unit test (`afterRefusal`) and browser test R "item 4" (drafts read and edit both 403).
5. **Render notes.** n1: the compact row reads "No change plan" (KO/AR drafts). n2: "Open" is the TW decision; root records it in the spec. n3 and editors not being exposed are now residuals.
6. **Decision 14 captures.** `accounts-six-columns` (relationship kind view group, checkboxes, a selected account) in EN/KO/AR at 1440/1280/390, in `scratchpad/hb319work/shots-r3-L` and `shots-r3-six` (`HYB319_accounts-six-columns_{lang}_{width}.png`). At 1280 the runner asserts the panel is exactly 340px with six header cells.

**Runs (round 2)**
- tsc clean; `npm test` 746/746; build ok.
- gofmt 0; vet ok; `go test ./apps/api/server` ok twice; coverage 100.0%.
  - One run during concurrent browser QA failed once, and I did not capture which test failed. Two full runs without load and a run that skipped the timing test all passed. This matches the earlier timing-test flake under load (`TestTimingIndistinguishable` passed three times on its own), but that is an assumption.
- Browser:
  - layout EN/KO/AR × 1440/1280/390: 2295 PASS, 3 FAIL. All three were the new six-column state at 390, where the runner opened the panel before ticking rows (a runner order bug). After fixing the order, that state is 42/0 at 390 and 1280.
  - F 47/0; R 10/0; `qa300` A 423/0.

## Review round 3 fixes (root copy and change recheck, 2026-11-18)

- **N1.** A conflict on the first call of a new draft, when no draft exists yet, now shows `dataChangedNew`: "The data changed before your save. Nothing was saved and your input is kept. Save again." (KO/AR drafts). The choice is the pure function `conflictCopy`, which is unit tested and whose use in `drafts.tsx` is asserted.
- **N2.** `notAccount` is hedged: "Not found as a source account, so not added: {ids}." (KO/AR drafts).
- **AR conflict label.** The parenthesised count is wrapped in a nowrap span (`conflictLabel`) everywhere the label appears.
- **Revert checks.** `drafts-render.test.tsx` now asserts the full guard statements:
  - `if (!isLatest(n, memberSeq.current)) return;` (M1, success path) and the error-path guard;
  - `if (!draft || discarding) return;` (L5).

  Each guard was deleted locally: the test failed both times (9 pass / 1 fail), and the file was restored byte-identical.
- **Runs.** tsc clean; `npm test` 747/747; build ok. Affected qa319 states (accounts none / selected / six-columns, form new / errors / saving, draft open / dirty, partial save, groups table) EN/KO/AR × 1280/390: 492/0. `qa300` A: 423/0.

## Residuals

- Admin-only permission (Q2). The spec's group management permission does not exist in the gate.
- Drafts, names, labels and plans are in memory and reset on restart.
- Activation is held (HYB-296), so groups created by drafts cannot become effective in pre-launch. The activated and pruned panel states were captured with patched reads.
- The shared Reason hint splits "(about" from "256" at 1280 in EN. That component is outside this ticket.
- KO and AR copy are drafts for the technical-writing review. The glossary rows are added for that review.
- Render note n3 (root recheck): carried as a residual.
- Draft editors are recorded on the server and enforced at activation, but the drafts read and the UI do not list them; the only visible effect is the `own_draft` reason.
- L6 (unreachable pre-launch): pruning bounds are global, not per workspace; activated drafts are hollowed, not deleted; per-command plan lookups scan the drafts store.
- Batches beyond 31 members were exercised in the browser only with mocked command responses, because the sample seed has six accounts. The Go tests cover the 31-per-call limit and two-chunk sealing.

## Final runs (after review round 1, base e360bf7)

- **Go**
  - gofmt: 0 files.
  - vet: darwin, linux and windows clean.
  - `go test -count=1 ./...`: every package ok.
  - Coverage: `apps/api/server` 100.0%, `modules/relationships` 100.0%.
  - `-race` on both: ok (341 s, 13 s).
- **Mutation**
  - Round 1: 96 mutants. 92 killed (the two `Required` mutants now included); 4 argued (prune of already pruned drafts, members read under one boot workspace, two cross-setting checks).
  - Round 2: 11 mutants on the SoD and snapshot lines, all killed. One first failed to compile and was rewritten.
- **Console**: tsc clean; `npm test` 738/738; build ok.
- **Browser**
  - F: 47/0.
  - R (robustness, mocked where stated): 9/0.
  - Full matrix, EN/KO/AR × 1440/1280/1024/390 × 31 states with the repaired checks and the pointer moved off: 2956 PASS, 2 FAIL. Both failures were "나머지 10개" splitting in the KO partial-save sentence. They are fixed with no-break spaces between each word and its number, and the affected states were re-run on all 12 combinations: 600 PASS, 0 FAIL. Each state was prepared in sequence (accounts-none, form-new, form-saving, draft-open, partial-save, groups-table).
- **Regressions** (each run 0 FAIL)

| Suite | PASS |
| --- | --- |
| P01 | 46 ("FAILS 0 OF 46") |
| P02 A/B/C/D | 162/32/10/16 |
| P03 R/A/B/C | 288/23/12/24 |
| P04 R/A/B/C/D | 559/28/20/46/11 |
| P05 R/F/A/B/C1/C2/C3/P and E | 481/132/17/9/2/3/30/12 and 142 |
| O02 `qa296` | 209 |
| `qa300` A/P/C | 423/72/55 |
| `qa318` A/L | 84/876 |

- The O02 copy fixes found by the new checks are KO only: "변경 전 (전체 기간)", "변경 후 (전체 기간)" and "해당 날짜 UTC 00:00부터 적용됩니다.". The O01 dates in list metas and the outside-list heading are now wrapped in nowrap `<time>`.

## Root final verification (2026-11-18, main e360bf7)

- Root rebased from 1cf7f3f onto e360bf7 (HYB-299): conflicts in `admin.go` (both bulk and draft definitions registered), `trust_hold_test.go` (82 commands = HYB-299's five rows plus these four), `main_test.go` (merged by function) and `package.json`; root added the `view_group` exemption in `bulk_cover_test.go` because `view_group.propose` creates relationship plans that the relationship store already counts with the proposer recorded.
- Root re-ran: `gofmt` clean; `go vet ./...` ok; `go test ./...` failed once only on `TestTrustConcurrentActivation` (known flake on main, HYB-321) and the server package passed on the immediate rerun; coverage 100.0% on `apps/api/server` and `modules/relationships`; `tsc` clean; `npm test` 747/747; build ok; no changed file over 1,500 lines.
- Reviews: change review FAIL → FAIL (durable console tests) → closed by the round-3 source assertions, each confirmed failing on revert; render review FAIL → PASS → decision 14 confirmed on the six-column captures; technical-writing review FAIL → FAIL (one string) → fixed in round 3.
- Owner-level decision recorded: separation of duties for drafts (author and editors cannot activate the resulting plan), plan amendment 1 for Part C.

