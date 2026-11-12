---
type: VerificationRecord
title: Still-working line verification
description: Records what was executed and reviewed for the shared still-working line on the reader-backed screens, with results, failed regression suites, limits and accepted residuals.
status: draft
---

# Delivered behavior

[HB-164](https://linear.app/hybrid-billing/issue/HB-164/reader-backed-screens-say-when-a-read-is-still-waiting) changes the console from main `6374496` as the owner's second chosen follow-up to HB-162. Root (Claude Code) owns planning, implementation, terminology writing, QA and integration; a separate read-only reviewer reviewed the plan before implementation and the change afterwards; the ui-designer role gave design input and compared renders. Machine: mac_mini. See the [plan](plan.md).

After four seconds in the loading state, U01, S04, B01, B03, B04, B05, B06, B07, B08, retained connections, import review and import review's collection summary add one line under their loading sentence: "Still working on this query. Other queries may be running first, so this one may be waiting its turn." (Korean and Arabic in the plan). The line is a second line of the loading sentence's paragraph, 14px in the muted color, and is gone as soon as the screen loads or fails. Requests, errors, clearing, selection and focus are unchanged.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| Console unit suite (`npm test --prefix apps/console`) | PASS 309 (306 before): the schedule shows nothing one millisecond before the delay, shows once at the delay and never after cancelling; the first render is empty; the wordings contain no digit, duration or error word |
| TypeScript check, Vite build, `tools/verify-build.mjs` | PASS; 438 artifact hashes and licenses |
| `apps/console/tools/slow-read-qa.mjs` on the retained sample | PASS three times, the last on the final build; 0 page errors; see below |
| Existing browser suites of the screens, one after another | PASS on the first attempt: U01, S04, B01, B03, B04, B05, B06, B07, retained connections, collection status. FAIL: B08 and receipt reader; both also fail on main's build, see below |
| Retained source, control and prior-result files | PASS 93 file hashes equal to the HB-150 preservation manifest after all reads |

The tool never waits a fixed time for the line. In the held case it holds the screen's reader requests until the line has been observed and then lets them through, and an observer inside the page records when the loading state and the line appear. Results of the last run:

- Held case, 26 runs (eleven routes in English at 1440; U01, B07 and import review also in Korean and Arabic at 1440 and 390 and in English at 390): the line appeared 4,007 to 4,046 milliseconds after the loading state, exactly once, visible, with the exact text, in a paragraph inside an element with `role="status"` inside the loading screen; 14px, weight 400, block, 4px above, the muted token's color (also equal to the loading sentence's color on every screen), no border, background or animation; the sentence above it 16px; reading direction as expected; no page overflow. After release every screen reached its loaded state and the line was gone.
- Fast case, eleven routes: the recorded responses were replayed at once and the screen reached its loaded state; 4.6 seconds later no line existed in any main loading paragraph. On ten routes no line existed anywhere. Import review needed one live request for its collection summary, so the page-wide assertion is not made there; in an earlier run that summary read took longer than four seconds and showed its own line, which is the intended behavior.
- Failed case, eleven routes: after a `503` answer every screen reached its error state and no line existed 4.6 seconds later.
- Collection summary: with only its request held, the line appeared inside the summary's status paragraph, was the only line on the page, and was gone after release.

The component's effect and its cancelling on unmount are exercised only by this tool; the unit tests cover the schedule function and the wording.

Regression suites that failed, and why they are not attributed to this change:

- B08 `billing-runs-qa.mjs` failed twice on this change at the same assertion (line 485: no difference cell after a routed lower total and "New query"). Run against main's build it passed once and then failed twice at the same assertion; against this change it failed four times in all. The case clicks "New query" and then waits for a loaded state that is already present, so it can read the screen before the new answer arrives. Recorded as an existing unstable case, not corrected here.
- `receipt-reader-qa.mjs` failed twice at line 78 (four receipt controls where two are expected) and fails identically on main's build. Existing failure, not caused or corrected here.

The suites ran against the readers restarted from main (`6374496`) while the machine was under unrelated load (load average 8 to 32). The ten passing suites ran on the build before two behavior-neutral corrections from the change review (an unused parameter removed, an unintended formatting hunk reverted); the unit suite, the build and the slow-read tool were rerun on the final build. No reset, ingestion or benchmark was run.

Host: macOS arm64, Node 26.7.0, Playwright 1.56.1 with Chromium headless shell build 1243. Other operating systems, browsers and assistive technology are unverified. Captures stay local under `output/validation/HB-164` and are excluded from Git.

# Limits

- How assistive technology announces the line is not verified. A status element is announced as a whole, and five screens mark an ancestor busy while loading, which may defer the announcement until loading ends.
- The screen cannot tell a waiting query from a running one; the line says "may".
- The delay counts from when the loading element appeared, so after a selection change during loading the line can show sooner than four seconds into the new request.
- No space is reserved: on import review the line moves the receipt search field down by one line while it is usable.
- In the collection summary the loading sentence is itself 14px and muted, so the line is not quieter than its sentence there.
- The gap between sentence and line differs by two to three pixels between screens because the line inherits each screen's line height.
- Languages other than English were rendered on U01, B07 and import review only; contrast was not measured.
- AI wording review only. On retained connections and import review the screens' own controls say Refresh rather than query; the line keeps the shared term.

# Independent review and design input

Reviewer in each round: the repository's read-only `independent-reviewer` agent role (AI); designer: the `ui-designer` agent role (AI). 2026-11-12. Neither executed anything.

| Round | Result |
| --- | --- |
| Design review of the plan, before implementation | FAIL without blocker: 3 HIGH, 5 MEDIUM, 5 LOW. Resolved as in the plan's resolution table: the line only in the loading phase of persistent status paragraphs, no stated duration, the collection summary added, deterministic checks |
| ui-designer design input, before implementation | Adopted: the line inside the sentence's paragraph, 14px muted, no reserved space, no decoration. Not adopted: a wording with "up to about 30 seconds" |
| Change review of the implemented diff | PASS for the code without blocker or HIGH: one MEDIUM, this record was not yet written; LOW corrected: the tool asserts the reached state in every case, bounds the delay above, checks the token's literal color and scopes the fast case to main paragraphs; the unused delay and timer parameters were removed; a formatting-only hunk in the shared stylesheet was reverted; naming aligned. No second round ran on these corrections |
| ui-designer render comparison | PASS with four LOW: the summary's equal size, the gap drift, the layout shift on import review and the wording next to Refresh, all recorded as limits. Its request to compare the line's color with the sentence's was answered by the tool: equal on every screen |

# Accepted residuals

- The B08 and receipt-reader suite failures above remain; each needs its own ticket.
- B03's candidate awaiting the owner's renewed confirmation now also contains this line; FIX is not carried forward.
- No owner FIX was created by this change.
