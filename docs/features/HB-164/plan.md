---
type: ImplementationPlan
title: Still-working line on reader-backed screens
description: Defines one shared line that the reader-backed screens add to their loading state after a few seconds, now that a read can wait for the collected reader before it is served.
status: reviewed
---

# Scope and ownership

[HB-164](https://linear.app/hybrid-billing/issue/HB-164/reader-backed-screens-say-when-a-read-is-still-waiting) is the second follow-up to HB-162 chosen by the owner on 2026-11-12. Base: main `6374496`. Root (Claude Code) owns planning, implementation, terminology writing, QA and integration. A separate read-only reviewer reviews this plan before implementation and the actual change afterwards; the ui-designer role gives design input on this plan and compares the final renders. Machine: mac_mini. No owner FIX is created or carried forward.

# Observed baseline

Facts read on 2026-11-12 from the code at `6374496`:

- Since HB-162 and HB-163 the collected reader runs two queries at a time and lets up to six further reads wait up to 20 seconds. A screen cannot tell whether its read is running or waiting; it only knows that no answer has arrived.
- Eleven screens read from the collected reader and show one loading element until the answer arrives:

| Screen | Route | Loading element | State attribute |
| --- | --- | --- | --- |
| U01 overview | `/home` | `section[role=status]` with a heading and a sentence | `data-focus-state` |
| S04 collected source data | `/source-data` | `section[role=status]` with a heading and a sentence | `data-source-state` |
| B01 infrastructure cost | `/cost-pools` | `p[role=status]` | `data-infra-state` |
| B03 allocation policies | `/allocations` | `p[role=status]` | `data-allocation-state` |
| B04 sales pricing | `/rates` | `p[role=status]` | `data-pricing-state` |
| B05 FX and rounding | `/exchange-rates` | `p[role=status]` | `data-fx-state` |
| B06 commitment benefits | `/commitments` | `p[role=status]` | `data-b06-state` |
| B07 marketplace | `/marketplace` | `p[role=status]` | `data-b07-state` |
| B08 billing runs | `/billing-runs` | `p[role=status]` | `data-b08-state` |
| Retained connections | `/connections` | status paragraph with `words.loading` | `data-read-phase` |
| Import review | `/imports` | status paragraph with `words.loading` | `data-read-phase` |

- On nine screens the loading element is rendered only while loading and already has `role="status"`. On retained connections and import review the status paragraph is always mounted, switches its text by phase, takes programmatic focus outside the loading phase, and on retained connections turns into `role="alert"` on an error.
- Import review issues a second reader request for its collection summary (`CollectionSummary.tsx`), with its own persistent status paragraph, after the main read has loaded.
- Paging, scope changes and row or group selection on the nine screens clear the data and show the same loading element; no path keeps old data during a reader read. No reader request has a client-side timeout.
- Five screens set `aria-busy` on an ancestor while loading (S04, B07, B08, retained connections, import review).
- There is no shared component for loading states; each screen has its own sentence and its own stylesheet. `apps/console/src/shared/` holds shared components and `styles.css`. Every screen's body text is 16px and its secondary text 14px in a muted color.
- B03 carries an owner FIX for its Korean policy-editor design with a wording change pending renewed confirmation (`docs/ui/acceptance/index.md`); its loading state was never part of the accepted scope.

# Decision

| Area | Decision |
| --- | --- |
| Component | `apps/console/src/shared/SlowReadHint.tsx`: renders nothing at first and, once it has been mounted for 4000 milliseconds, one `span` with the line below. The timing is a small function so that it can be tested with mocked timers without a browser; the component is a thin wrapper that cancels on unmount. The component's own effect and unmount wiring are exercised only by the browser tool. The delay counts from when the component mounted, so after a selection change during loading the line can appear sooner than four seconds into the new request |
| Wording | EN "Still working on this query. Other queries may be running first, so this one may be waiting its turn." KO "아직 조회 중입니다. 먼저 진행 중인 조회가 있으면 차례를 기다린 뒤 조회합니다." AR "ما زال الاستعلام جارياً. قد تكون استعلامات أخرى قيد التنفيذ قبله، فينتظر دوره." It uses the screens' term query / 조회 / استعلام, states no duration and names no error |
| Placement | As the last child of the paragraph that carries the loading sentence, so that it is a second line of that sentence inside the status element. On retained connections, import review and the import-review collection summary it is rendered only in the loading phase of the persistent status paragraph, never with error, unconnected, empty or loaded text and never inside an alert |
| Style | One rule in `shared/styles.css` for `.slow-read-hint`: block, 4px above, 14px, weight 400, the muted token, start-aligned, a 90-character measure; line height inherited. No border, background, icon, color accent, italics or animation. No space is reserved for it; no per-screen rule |
| Behavior | No change to requests, timeouts, errors, clearing, selection or focus. The line does not move focus |
| B03 | B03 gets the same line. Its acceptance record names the affected item, the reason and the impact and states that the pending candidate now also contains this line; the FIX is not carried forward |

Stable item IDs: none added. The line is part of each screen's existing asynchronous-state item.

External references opened on 2026-11-12: Jakob Nielsen, "Response Times: The 3 Important Limits" (nngroup.com), for feedback on delays beyond ten seconds; W3C, "Understanding Success Criterion 4.1.3: Status Messages", for progress text exposed through a status role without moving focus. Adopted: a plain text escalation in the existing status element. Not adopted: a percent-done indicator and an interrupt control, which the first source recommends for such delays and which this ticket defers; a stated duration.

# Allowed files

`apps/console/src/shared/{SlowReadHint.tsx,slow-read-hint.test.tsx,styles.css}`; the eleven screens' `Screen.tsx` listed above and `sources/pages/import-review/CollectionSummary.tsx` (one import and one element each); `apps/console/package.json` (test list); `apps/console/tools/slow-read-qa.mjs` (new); `apps/console/README.md`; `docs/features/HB-164/*`; `docs/ui/terminology/product-glossary.md`; `docs/ui/acceptance/{index.md,B03.md}`.

# Verification

- Unit tests with mocked timers: the line is scheduled once, not shown one millisecond before the delay, shown at the delay, never shown after cancelling; the first render is empty; the three wordings contain no digit and no error word.
- New browser tool `slow-read-qa.mjs` on the retained sample. It never waits a fixed time for the line: it holds the screen's reader requests until the line has been observed and then lets them through.
  - Held case, all eleven routes in English at 1440: an in-page observer records when the loading state and the line appear, and the difference must be at least 3.9 and under 6 seconds; exactly one line, visible, with the exact text, whose parent is a paragraph inside an element with `role="status"` inside the loading screen; computed 14px, weight 400, block, 4px above, the muted token's color, no border, background or animation; the sentence above it 16px; no page overflow. After release the screen leaves the loading state and the line is gone.
  - Fast case, every route: the responses recorded in the held case are replayed at once, and 4.6 seconds after the screen has settled no line may exist. The assertion applies only when every request was replayed; a route that needed a live request is reported as not asserted.
  - Failed case, every route: a `503` answer, and 4.6 seconds after the screen has settled no line may exist.
  - U01, B07 and import review also in Korean and Arabic at 1440 and 390 and in English at 390, with text, direction and overflow.
  - Import review's collection summary: only its request is held; the line appears inside the summary's status paragraph and is gone after release.
- Console unit suite, type check, build and build verifier.
- Existing browser suites of the eleven screens as regression, run one after another. A suite that fails on `source_busy` or a timeout is repeated and both results are recorded.
- Retained file hashes against the HB-150 manifest. No reset, ingestion or benchmark.
- Independent review of the actual change; ui-designer render comparison; technical-writing review of the three strings. AI review is not native-language, practitioner or owner acceptance.

Not verified: how assistive technology announces the added line. A status element is announced as a whole, and five screens mark an ancestor busy while loading, which may defer the announcement until loading ends.

# Deferred

A progress indicator or a cancel control; telling a waiting read from a running one, which needs a reader signal; the same line on screens that read from other local services.

# Design input and review resolution

Both were given on 2026-11-12 on the first version of this plan; neither executed anything.

| Source | Point | Resolution |
| --- | --- | --- |
| Design review, FAIL without blocker (3 HIGH, 5 MEDIUM, 5 LOW) | The status paragraph is persistent on two screens | Line rendered only in the loading phase there; baseline corrected; fast and failed cases assert absence |
| | "About 30 seconds" unsupported; "reads" is not the screens' term | No duration; query / 조회 / استعلام |
| | "A fast read never shows the line" cannot be asserted on a live reader | Recorded responses are replayed |
| | The collection summary's read was uncovered | Added to scope and to the tool |
| | Announcement by assistive technology not established | Recorded as not verified |
| | Wall-clock checks | Hold until observed; in-page timing |
| | A deterministic unit test is possible | Timing extracted and tested with mocked timers |
| | B03 record needs item, reason, impact | Written that way |
| | External reference | Two sources opened and recorded |
| ui-designer input | Line inside the sentence's paragraph, not after it | Adopted for all screens |
| | Explicit muted color and weight, 14px, 90-character measure, no reserved space | Adopted |
| | Split wording with "up to about 30 seconds" | Not adopted: the review's objection to a duration prevails |
| | Per-screen rendered checks | Adopted in the tool, apart from equal gaps across screens (asserted as the 4px margin) and contrast, which is not measured |
