---
type: Verification
title: HYB-309 P02 plugin license check before activation — verification record
description: Commands run and results observed for the plugin license field, the permitted-license installation setting, the activation refusal, the integrations read, the P02 console display, mutation testing on the changed lines and the browser QA and P02 regression. Run on mac_mini on 2026-11-18.
status: stable
tags: [HYB-309, P02, HB-40, verification]
sources:
  - resource: ./plan.md
---

# HYB-309 verification

**Scope.** This record covers the implementation of [the plan](plan.md) in `/private/tmp/hb309`:
- branch `feat/HYB-309-p02-plugin-licence` on main `8d673f1`;
- uncommitted;
- machine `mac_mini`, 2026-11-18;
- Go `go1.25.7 darwin/arm64`, Node `v26.7.0`.

Every number below was observed in this session. Nothing listed as not run was run. No independent change review has happened yet.

**Go build cache.**
- Go commands used `GOCACHE` directories in the session scratchpad: `gocache309` for mutation, cleaned at the start and end of each batch, and `gocache309build` for builds and repeat runs.
- Exceptions: the first `go test ./...` run, the `-race` run and the module-level checks during development used the default cache.

## Results

Command → observed result.

### Go checks

| Check | Result |
| --- | --- |
| `gofmt -l modules apps` | No output |
| `go vet ./...`, run three times | Exit 0 each time |
| `go test -count=1 ./...` | 73 packages `ok`, 0 failed |
| `go test -race -count=1 ./modules/plugins ./apps/api/server ./apps/api/cmd/hb-api` | All three `ok`: 7.5 s, 543.1 s and 14.3 s. Exit 0. |
| `go test -count=40 ./modules/plugins` | `ok` (repeat run before mutation) |
| `go test -count=20 -run 'PluginLicense\|TestIntegrationsReadShape\|TestIntegrationSeed' ./apps/api/server` | `ok` |
| `go test -count=20 -run PluginLicenseFlags ./apps/api/cmd/hb-api` | `ok` |

**Coverage.** `go test -cover`, base `8d673f1` (scratchpad worktree `hb309base`) against this change:

| Package | Base | This change |
| --- | --- | --- |
| `modules/plugins` | 100.0% | 100.0% |
| `apps/api/server` | 100.0% | 100.0% |
| `apps/api/cmd/hb-api` | 92.9% | 93.6% |

`modules/plugins` measured 99.9% until `TestValidLicensesWrapper` was added. The exported `ValidLicenses` was reached only from the server package before that.

### Console

| Check | Result |
| --- | --- |
| `npm test` | 755 tests, 755 pass, 0 fail. Base: 749 tests, 749 pass. |
| `npm run build` (runs `tsc --noEmit`, then the vite build) | `✓ built`; no type errors |

Console line coverage, with `node --test --experimental-test-coverage` on the two integrations test files:

| File | Base | This change |
| --- | --- | --- |
| `service/integrations.ts` | 98.63% | 98.69% |
| `live-model.ts` | 98.95% | 99.50% |
| `live-copy.ts` | 69.44% | 77.47% |
| `live-codes.ts` | 100% | 100% |
| `LiveModules.tsx` | not loaded | 75.73% (now loaded by the new render test) |
| `LiveProposals.tsx` | not loaded | 74.42% (now loaded by the new render test) |

### Source rules

- No source comments were added: a search of the added lines finds none, and the `modules/plugins` source test passes.
- No Go test file contains `flo`+`at`.
- Every changed file is under 1,500 lines. The largest is `apps/api/server/integrations.go` at 736 lines.

## Tests added or changed

### `modules/plugins`

- **`license_test.go`.**
  - The two-form grammar: accepted and refused tables, including `LicenseRef-`, `licenseref-x`, `LICENSEREF-x`, `-`, `.`, `MIT-`, `0BSD`, `Zlib`, expressions, `+` and length bounds.
  - A manifest with a refused licence is invalid at `Propose`.
  - The permitted list: sorted, unique, at most 64 entries. The `Licenses()` read returns a copy.
  - The known-answer hash is built by hand, independently of `encode()`, and pinned. The pinned value equals `encode()`.
  - The signature covers the licence.
  - Install, upgrade, rollback and enable are refused at `Propose` with `license_not_permitted`. The counter, the pending proposals and the installation are unchanged, including in a verification workspace.
  - White-box backup checks (`r.config.Licenses` changed between calls):
    - `Recheck` and `Approve` agree on `license_not_permitted` for each change kind;
    - `IssueToken` and `CheckEgress` refuse; `CheckToken` forgets the token;
    - the egress recheck after the destination call returns `ErrStale`;
    - the mark sweep stops all three installations with reason `license_not_permitted`.
  - Reason order: revoked, then licence, then vulnerable.
- **`property_test.go`.**
  - Random licences per version and a random permitted subset per seed.
  - The model's `runnable` includes the licence, and `VersionView.Permitted` must agree with it.
  - Every install, enable, upgrade or rollback whose only obstacle is the licence must fail with `license_not_permitted`.
  - Removing the licence case from `runnable` makes the property test fail (checked by hand, then restored).
- **Fixtures.** `form_test.go` gets the new known answer and a `license` row in the distinct-encoding table. In `world_test.go`, manifests carry `MIT` and the test registry permits `Apache-2.0` and `MIT`.

### `apps/api/server`

- **`pluginsettings_test.go`.**
  - The default list.
  - Sorting.
  - `-plugin-licenses=` gives the empty list.
  - Each of these refuses at `licenses()` and at `New` with an error ending in `plugin_licenses`: whitespace, an empty element, a leading or trailing comma, a malformed entry, a bare or lower-case reference, a duplicate, 65 entries, and a repeated flag.
  - With the seed, a list without `Apache-2.0` refuses with the message "the sample seed needs Apache-2.0 in plugin_licenses". Without the seed it starts.
  - The read reports the empty list as `start_flag`.
  - With `Apache-2.0,LicenseRef-sample-proprietary`, all three versions are permitted, and the upgrade to 1.2.0 is proposed (proposal 11), approved and installed.
  - An upgrade to 1.2.0 under the default list returns 422 `invalid_stage`. A resend with the same idempotency key also returns 422 `invalid_stage`. No proposal is recorded and the revision does not move.
- **Existing tests.**
  - Proposal numbers move by one (10 to 11, 11 to 12) because the seed now registers 1.2.0 as proposal 10.
  - The seed has 10 entries and versions `1.0.0,1.1.0,1.2.0`.
  - The read-shape test checks `license`, `license_state` and `modules.licenses`.
  - The limit test's own registry gets the default list.

### `apps/api/cmd/hb-api`

- **`main_test.go` (`TestPluginLicenseFlags`).**
  - The flag keeps the raw value, `Given` and the use count; a repeat gives `Uses` 2.
  - Start through `run` is refused with `ErrConfig` ending in `plugin_licenses` for each of: a space, a repeat, `MIT` with the seed, empty with the seed, a trailing comma, and a malformed member.
  - The usage text lists the default.

### Console

- **`live-model.test.ts`.**
  - Not-permitted and unknown states are not runnable and are withheld from the offer.
  - The `license` blocker, and the mixed case that keeps `untrusted`.
  - Withheld versions in the stopped and enabled states.
  - `dryRunBlocked` and the blocker sentence with the blocked sentence.
  - An `approve_blocked = license_not_permitted` fixture.
  - `refusedLicense` and `refusedValues` across `settle`.
  - Copy in EN, KO and AR.
  - A render test of `ModulesView` (server-side markup):
    - the list and its source;
    - the licence lines;
    - the not-offered note;
    - the target licence on proposals;
    - a licence-blocked approval with its reason and blocked sentence, and no Approve button;
    - the empty and unknown lists.
- **`service/integrations.test.ts`.** `licenseStateOf` and `licensesOf` fail closed, and `integrationsOf` carries the fields.

## Mutation testing

**Tool and method.**
- Tool: `scratchpad/hb309work/mutate.py`.
- Operator mutants run only on the lines changed against `8d673f1`, plus every line of new files.
- The operators are:
  - comparison and boolean swaps;
  - `!` removal;
  - `true`/`false`;
  - `++`/`--` and `+ 1`;
  - `> 1` to `> 0`;
  - `return nil` to `return ErrConfig`.
- Each package runs in its own rsync copy with `GOCACHE=scratchpad/gocache309`, cleaned at the start and end of the batch.
- Each mutant runs `go vet`, then the package tests: all tests for `modules/plugins`, and `-run` filters for the server and the command.
- Restores: after each file, the file's hash is compared with the original. After each batch, the copy is compared with the worktree (`diff -r`). All restores matched (`restore-diff-rc=0`).

| Package and files | Killed | Survived | Not compilable |
| --- | --- | --- | --- |
| `modules/plugins`: `manifest.go`, `changes.go`, `registry.go`, `operations.go` (first run) | 59 | 2 | 1 |
| `apps/api/cmd/hb-api`: `main.go` | 3 | 0 | 1 |
| `apps/api/server`: `pluginsettings.go`, `integrations.go`, `integrationseed.go`, `server.go` | 11 | 0 | 4 |

**Survivors.**
- `modules/plugins`, first run: `manifest.go:78` `c <= 'Z'` to `c < 'Z'`, and `c <= 'z'` to `c < 'z'`. No accepted licence contained `Z` or `z`.
  - `Zlib`, `zlib`, `Z`, `z`, `9`, `A0` and `LicenseRef-z9Z` were added to the accepted table.
  - Re-run of the plugins batch: 61 killed, 0 survived, 1 not compilable (`||` to `&&` on the same line, refused by `go vet`).
- `apps/api/server`: none. All four non-compilable mutants are `return nil` to `return ErrConfig` in two-value returns.

## Browser QA

**Runners and servers.**
- Runner `scratchpad/qa309.mjs`, driven by `scratchpad/qa309reg.sh`.
- Headless Chromium through Playwright 1.56.1.
- The console is served from each build's `apps/console/dist` through `qa298proxy.mjs`.
- The API binary is built from each tree:
  - `hb-api309` from this change;
  - `hb-api309base` from `8d673f1`.
- Ports: 4511–4514 for the console and 8851–8855 for the API.
- Load average was 9.5–12 during the runs.

**Group L (default list).**
- In EN, KO and AR, at 1440 and 390 wide:
  - three versions;
  - 1.0.0 and 1.1.0 read `Apache-2.0` and permitted;
  - 1.2.0 reads `LicenseRef-sample-proprietary` and not permitted;
  - the list reads "installation default", contains `Apache-2.0` and `Zlib`, and has no `CC-BY`;
  - no horizontal scroll;
  - the identifier renders left to right;
  - no coloured border on the licence line or the list.
- Proposal 8 shows "License of 1.1.0: Apache-2.0 · Permitted on this installation".
- The not-offered note appears once and names 1.2.0 and its licence. It comes before the change choices in reading order.
- The propose form lists 1.2.0 but offers no upgrade to it, and Remove is still offered.
- The note also appears in AR at 390 and in KO at 1440, without horizontal scroll.

**Group S (`--plugin-licenses=Apache-2.0,LicenseRef-sample-proprietary`).**
- The list reads "installation setting", and 1.2.0 is permitted.
- There is no not-offered note.
- admin1 proposes the upgrade to 1.2.0, and the proposal shows the target licence as permitted.
- The confirmation dialog for admin2 shows the licence. The approval reads Applied, and the installation reads 1.2.0.

| Run | Result | Console errors |
| --- | --- | --- |
| Group L on this change | 64 PASS, 0 FAIL | none |
| Group S on this change | 8 PASS, 0 FAIL | none |
| Group L on base `8d673f1` (the checks must fail first) | 0 PASS, 64 FAIL | not counted |
| Group S on base | Not started: the base API refuses with "flag provided but not defined: -plugin-licenses", so the runner found no activation file | not applicable |

On base, a first version of the runner passed 10 checks: the no-scroll checks, "no upgrade to 1.2.0 offered" and "Remove offered". These only guarded against regressions. Each was tied to the new behaviour it is meant to show (the list shown, 1.2.0 listed, the note present), and the base run was repeated (0 PASS).

**Command line (this change, with the seed).**

| Arguments | Result |
| --- | --- |
| `--plugin-licenses=MIT` | `hb-api: configuration is invalid: the sample seed needs Apache-2.0 in plugin_licenses` |
| `--plugin-licenses=` | Same message |
| `'--plugin-licenses=Apache-2.0, MIT'` | `configuration is invalid: plugin_licenses` |
| `--plugin-licenses=Apache-2.0,` | `configuration is invalid: plugin_licenses` |
| `--plugin-licenses=Apache-2.0 --plugin-licenses=Apache-2.0` | `configuration is invalid: plugin_licenses` |

The base binary with `--plugin-licenses=MIT`: "flag provided but not defined".

**P02 regression.** `qa285fullon297.mjs`, groups A–D:

| Build | A | B | C | D | Console errors |
| --- | --- | --- | --- | --- | --- |
| This change | 162 PASS, 0 FAIL | 32 PASS, 0 FAIL | 10 PASS, 0 FAIL | 16 PASS, 0 FAIL | none; D had the 2 expected aborts |
| Base `8d673f1` | 162 PASS | 32 PASS | 10 PASS | 16 PASS | (same count) |

Proposals 8 and 9, the upgrade to 1.1.0, stop, disable, enable, the rollback to 1.0.0, the subscription and dead-letter flows, and the fail-closed unknowns all passed unchanged.

Screenshots are in the scratchpad (`qa309shots/`, `reg309shots/p02/`), not in the repository.

## Deviations from the plan

- **Exported helper.** `plugins.ValidLicenses` was exported so that the server validates the start setting with the module's grammar instead of a copy of it. This is a new exported function in `manifest.go`.
- **Server settings file.** The plan named `settings.go`, or a new `pluginsettings.go`. The new file was used, and `settings.go` is unchanged.
- **Render test.** A render test of `ModulesView` was added inside `live-model.test.ts`, the allowed test file. It loads `LiveModules.tsx` and `LiveProposals.tsx` with a CSS stub hook.
- **Seed numbering.** The seed's 1.2.0 registration is proposal 10, as planned. Existing server tests that create proposals moved by one number.
- **Source values (root amendment after review round 2).** `modules.licenses.source` uses `default` and `start_flag`, like the existing settings read (`settings.go:11-14`), not the plan's original `installation_default` and `installation_setting`. The plan, server, console parser and word map, tests and field map were updated. The display words are unchanged.

## Not run or not verified

- The independent change review, the re-review and the TW review of the rendered copy.
- Native-language review of KO and AR.
- Legal review of the default list.
- Linear updates: the Linear MCP reported 401 at session start, and Linear was not touched.
- No commit and no PR.

## Re-run after the source-value amendment (2026-11-18)

| Check | Result |
| --- | --- |
| `gofmt -l apps modules` | No output |
| `go vet ./...` | Exit 0 |
| `go test -count=1 ./apps/api/server ./apps/api/cmd/hb-api ./modules/plugins` | All three `ok` |
| `tsc --noEmit` | Exit 0 |
| `npm test` | 755 tests, 755 pass, 0 fail |
| `npm run build` | `✓ built` |
| `qa309.mjs` group L, on a rebuilt `hb-api309` and `dist` | 64 PASS, 0 FAIL, no console errors |
| `qa309.mjs` group S, same build | 8 PASS, 0 FAIL, no console errors |

Mutation testing, coverage and the P02 regression were not repeated. This change touches only constant values and map keys, plus one deleted constant pair.

## Review round 3 fixes (2026-11-19)

Change review: PASS with 2 Low. Render and copy review: FAIL with 2 Medium and 4 Low. Every item in `scratchpad/hb309-fixes.md` was applied. The plan's "Review round 3" section lists them.

### Fail first

The checks were run against the build before the fixes: the previous console `dist` and the previous `hb-api309`, copied to `scratchpad/hb309prefix` and `hb-api309prefix`.

| Run before the fixes | Result |
| --- | --- |
| `qa309.mjs` group L | 63 PASS (the earlier checks), 5 FAIL |
| Group R | 0 PASS, 12 FAIL |
| Group S | 7 PASS, 1 FAIL |

The failing checks in group L are:
- note identifiers in mono;
- the not-permitted version on two lines;
- only the state sentence at weight 600;
- the enable proposal shows the installed version's licence;
- the open-form note is the last item of the choices and describes them.

Group R fails in every language. The old message was the "refused this proposal … too many pending proposals" text.

Group S fails only on the new start-flag source label.

Two checks that passed before the fixes were folded into fail-first checks:
- the open-form "not repeated" guard;
- the stale-read precondition of group R.

`TestIntegrationsReadFaults` with the `plugins-licenses` step was run with the read line temporarily reverted. It failed ("got 200"), and the line was restored.

### After the fixes

| Check | Result |
| --- | --- |
| `gofmt -l apps modules` | No output |
| `go vet ./...`, three times | Exit 0 each time |
| `go test -count=1 ./...` | 73 packages `ok`, 0 failed |
| Go coverage | `modules/plugins` 100.0%, `apps/api/server` 100.0%, `apps/api/cmd/hb-api` 93.6% (no drop) |
| `-race` on `./modules/plugins` and `./apps/api/server` | Both `ok` (4.1 s and 469.8 s) |
| Mutation on `apps/api/server/integrations.go`, all changed lines including the new `pluginCall("licenses", …)` | 1 mutant, 1 killed, 0 survived; restore verified (`restore-diff-rc=0`) |
| `tsc --noEmit`, `npm test`, `npm run build` | Exit 0; 755 tests, 755 pass; `✓ built` |
| Console coverage of the touched files | See below |
| `qa309.mjs` group L | 68 PASS, 0 FAIL |
| Group R: refusal triggered by a stale read, captured in EN 1440, KO 1440 and AR 390 | 12 PASS, 0 FAIL |
| Group S | 8 PASS, 0 FAIL |
| Group C2: 2× device-scale captures at 390 of the licence list, the 1.2.0 row and the note, in EN, KO and AR | 9 PASS |
| Console errors in groups L, R, S and C2 | none |
| P02 regression `qa285fullon297.mjs` | A 162, B 32, C 10, D 16 PASS, 0 FAIL; no errors apart from D's 2 expected aborts |

Console line coverage of the touched files:
- `live-codes.ts` 100%;
- `live-copy.ts` 79.22%;
- `live-model.ts` 99.18%. The uncovered lines are 113, 288, 360 and 395-396, which predate HYB-309;
- `LiveModules.tsx` 80.38%;
- `LiveProposals.tsx` 79.08%;
- `integrations.ts` 98.69%.

**How group R works.**
- A Playwright route marks 1.2.0 as `permitted` in the integrations read, so the console offers the upgrade.
- The route is removed before the confirmation.
- The server refuses with `invalid_stage`, and the console reloads the real read.
- The message is then checked:
  - it starts with "The plugin registry refused this change. Nothing changed.";
  - it names `1.2.0` and `LicenseRef-sample-proprietary` in mono;
  - it says that the list changes only when the server is restarted with a different list;
  - it has no pending-limit wording.

**Captures.** `scratchpad/qa309shots`:
- `license-refusal-{en-1440,ko-1440,ar-390}.png`;
- `crop2x-{list,row-1.2.0,note}-{en,ko,ar}-390.png`.

### Deviations from the plan in this round

- `RowNote.kept` in `LiveCommand.tsx` now accepts a React node, so that the refusal can render identifiers in mono. `LiveCommand.tsx` was not in the plan's allowed files. Every other caller still passes a string.
- The installation-default glossary row is written by the writer. The reviewer's wording was not available in the fixes file.
- Plan section 3.4 ("Propose form"): with the form open, the note now sits inside the choices (L3).

### Not repeated in this round

- Mutation testing outside `integrations.go`. The other Go change is a test, `property_test.go`.

## Review round 4 fixes (2026-11-19)

Change review and render/copy review both passed, with low findings. Fixes:
- **L5 + N1:** the refusal's next step now depends on the context: propose, enable or approve.
- **N2:** the not-offered note is read once. It stays inside the fieldset, and `aria-describedby` is removed.
- **Deviation recorded:** `LiveCommand.tsx` (`RowNote.kept` widened to `ReactNode`). It is also listed in the plan's "Review round 4" section.

### Fail first

`qa309.mjs` group L was run against the console `dist` from round 3: 67 PASS, 2 FAIL.
- "open form: note is the last item of the choices and not repeated" failed.
- "accessibility tree: the note is read once" failed: the fieldset still described the note.

The context unit tests could not exist before the fixes, because `keptLicenseText` took no context.

### Results

| Check | Result |
| --- | --- |
| `tsc --noEmit` | Exit 0 |
| `npm test` | 755 tests, 755 pass, 0 fail. The new assertions are in existing tests: each of the three contexts in EN, KO and AR, including the unknown variants. |
| `npm run build` | `✓ built` |
| `qa309.mjs` group L | 69 PASS, 0 FAIL |
| Group R | 12 PASS, 0 FAIL |
| Group S | 8 PASS, 0 FAIL |
| Group C2 | 9 PASS, 0 FAIL |
| Console errors in groups L, R, S and C2 | none |
| P02 regression `qa285fullon297.mjs` | A 162, B 32, C 10, D 16 PASS, 0 FAIL; no errors apart from D's 2 expected aborts |

No Go file changed in this round, so the Go checks were not repeated.

### Not covered by the browser

- The approve-context and enable-context refusals cannot be produced through the service in this build. The approve case needs a pending proposal for a version that is not permitted. The enable case needs an installed version that is not permitted. Both are covered by unit tests only.

### Status

The plan and this record are marked stable because every listed re-run passed.

### Still open

- The TW review of the final wording.
- Native-language review of KO and AR.
- No commit.

## Root final verification (2026-11-19, rebased on main 3bdb750)

- Reviews: design review round 1 CHANGES REQUIRED → round 2 APPROVE (L1–L3 folded in) → root amendment (source values default/start_flag); change review round 1 PASS (2 Low) → round 2 PASS; render and copy review round 1 FAIL (2 Medium, 4 Low) → round 2 PASS (lows fixed in round 4).
- Rebase conflict in `apps/api/cmd/hb-api/main_test.go` (HYB-294 `TestUserFilterScanFlag` and this ticket's `TestPluginLicenseFlags`): both functions kept.
- Root re-ran: gofmt clean; `go vet ./...` clean; `go test -short ./...` all packages ok; coverage plugins 100.0%, server 100.0%, hb-api 93.6%; console tsc clean, npm test 771/771, build ok; OKF status test ok; P02 browser regression `qa285fullon297` A/B/C/D 162/32/10/16 pass, 0 fail.
- Follow-ups: HYB-328 (P07 notices), HYB-329 (AR installation default in M03), HYB-330 (P02 refusal code and request ID).
