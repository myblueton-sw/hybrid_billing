---
type: TraceReport
title: HYB-75 registry and routing against what is on main
description: Clause-by-clause trace of the foundation ticket for the screen registry and routing against main, to decide whether the ticket can be closed.
status: draft
---

# Scope and method

HYB-75 ([HB-72 0a] Foundation: Registry and routing), machine:mac_mini. The ticket has been In Progress since before the Linear move with no branch, pull request or `docs/features/HYB-75` folder. Its scope was delivered under other tickets: `git log` shows `apps/console/src/planning/routing.ts` and `legacy.json` arriving with HB-74 (`618987e2`, 2026-11-03) and changing with HB-76 (`28cfa3e7`). This report traces each clause of the ticket against main `65995e76` on 2026-11-26 by reading code and running the one test file named below. No product code is changed.

# Trace

| Clause of the ticket | State | Evidence |
| --- | --- | --- |
| Per-module screen registry files | present | fourteen `apps/console/src/modules/<module>/screens.json` files, 49 screens in all, read by `src/planning/registry.ts` |
| Registry files generated from the HB-72 registry | not as worded | the 14 files are committed and maintained by hand; whether they match `docs/features/HB-72/screen-registry.json` was not compared (see Not checked) |
| Tab slugs | present | 224 `key` entries under `tabs` in those files |
| Machine scope keys | present | a `scopeKeys` list on each of the 49 screens |
| Hash routing with object, tab and step segments | present | `src/planning/routing.ts` `resolveRoute`; test "object tabs and interim wizard steps resolve deterministically" |
| The 112-entry `legacyRedirects` table | present | `src/planning/legacy.json`, 112 entries; exported by `registry.ts` |
| Persisted legacy-to-planning screen map for review records | present as a committed file | `legacy.json` holds each legacy ID's planning screen; `legacyReviewMap` (`registry.ts:88-90`) is derived from it and shown in the review listing (`App.tsx:1357`); no separate stored review map exists |
| Per-module copy files | different from the wording | one module has a `copy.ts` at its root (`onboarding`); the others keep copy per page, in 40 `*copy*.ts` files under `modules/<module>/pages/<page>/`, with shared copy in `src/planning/copy.ts` and `src/shared/copy.ts`. Copy is separated from components and owned by module folders; it is not one file per module |
| Required test: all 112 legacy IDs redirect to the recorded target with lang, mode and scope kept | present and passing | `src/planning/routing.test.ts`, "49 unique planning screens and all 112 legacy redirects preserve query context" |
| Required test: unknown IDs fall back by audience (admin U01, customer C01, unauthenticated L01) | present and passing | same file: "unknown, malformed and unpermitted routes fail closed including home" (admin, U01) and "prototype property names never become legacy redirects" (unknown IDs for admin U01, customer C01, unauthenticated L01) |
| Required test: the registry has 49 screens with unique paths | present and passing | same file, first test |
| All existing suites | passing | `npm test` in `apps/console`: 1426 tests, 1423 passed, 3 skipped, as run for HYB-313 on 2026-11-25 on main `59b01919`; `routing.test.ts` is in that list. It was not rerun for this report |

Run for this report: `npx tsx --test src/planning/routing.test.ts`, 7 tests, 7 passed.

# Not checked

- `verify-build`, which the ticket lists beside the suites, was not run.
- The redirect targets were not compared with `docs/features/HB-72/screen-registry.json` entry by entry; the test asserts that each legacy ID resolves to the target recorded in `legacy.json`, not that `legacy.json` equals the HB-72 design file.
- Whether the HB-72 design meant one copy file per module strictly.
- Whether the registry files were ever generated, or still match the HB-72 registry screen by screen and tab by tab, not only in their redirect targets.
- The object, tab and step segments are each tested with one example.
- The legacy-redirect test resolves each ID under its target screen's audience only, not under every audience.
- The ticket's working rules (Codex as implementer, branch and review sequence) were followed, if at all, under HB-74 and HB-76; their review records were not reread.

# Conclusion

Every clause is present on main and the three required tests exist and pass, with two differences from the ticket's wording: copy is kept per page instead of in one file per module, and the registry files are maintained by hand instead of generated from the HB-72 registry. Neither changes behaviour. Nothing remains to build under this ticket as far as this trace reaches. Whether those two differences are accepted is the owner's to say; until then the ticket is left In Progress with this report, and not closed.

# Review

Independent review (role-based AI reviewer separate from the writer, by reading, nothing executed), 2026-11-26: approve with three corrections to the record, all applied with the reviewer's wording — the test cited for the customer and unauthenticated fallbacks, what "persisted" amounts to, and the untraced clause on generation.
