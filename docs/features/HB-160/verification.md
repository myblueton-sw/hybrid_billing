---
type: VerificationRecord
title: B07 source-row evidence verification
description: Records what was executed and reviewed for the B07 source-row terms, value states and scope identity on the retained sample, with results, limits and accepted residuals.
status: draft
---

# Delivered behavior

[HB-160](https://linear.app/hybrid-billing/issue/HB-160/align-b07-source-row-evidence-with-the-s04-source-field-terms-and) refines B07 from main `554fccb` using the retained public and synthetic sample verified in HB-149. Root (Claude Code) owns planning, implementation, terminology writing, QA and integration; a separate read-only reviewer reviewed the plan before implementation and the change afterwards; the ui-designer role compared renders. Machine: mac_mini. See the [plan](plan.md).

A candidate row shows its two costs through the shared components: Missing without currency, dash or digit, and Exact zero as a note in the row detail only. A null service reads Missing in the table and "Service: Missing" on the mobile card, whose description now carries B07's field state. The table's second column is headed with the billing account ID and charge period start and shows the instant in UTC. The row detail keeps the row ID always visible as a labeled field with the sentence that it is not a transaction ID, and its "Period, account and delivery references" disclosure lists one field per line: charge and billing period start and end in UTC, FOCUS version, billing account ID, subaccount ID, account scope ID, tenant reference, delivery ID, row ordinal and sample provenance. The candidate and insufficient-classification aggregates read "No values present" when no value is present and use the shared amount format. The account scope uses the shared labels in a label-over-value grid and B06's account-scope wording. Korean copy says 공급사. B07's own field states, `/api/focus-marketplace`, `contract.ts`, `api.ts`, classification, paging and permissions are unchanged.

# Retained sample facts used

Read on 2026-11-12 through the console proxy by root. The unfiltered selection has 1,121 authorized rows and one candidate row: billed cost `USD 0.342`, effective cost exactly zero, UTC instants, annotated as a public sample. The sample therefore has no missing or negative candidate cost, no null service or subaccount on the candidate row and no non-UTC instant; those states are covered by routed responses.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| Console unit suite (`npm test --prefix apps/console`) | PASS 302, unchanged count; no new logic |
| TypeScript check, Vite build, `tools/verify-build.mjs` | PASS; 438 artifact hashes and licenses; existing large-chunk advisory remains |
| B07 browser suite `apps/console/tools/marketplace-qa.mjs` on the retained sample | PASS; 0 page errors; see below. The unchanged suite also passed on main `554fccb` before any edit |
| S04 browser suite | PASS as regression, rerun on the final build |
| Isolated backend regressions (`tests/focus_marketplace`, `tests/focus_overview` projection and HTTP) | PASS 64; unchanged backend |
| Retained source, control and prior-result files | PASS 93 file hashes equal to the HB-150 preservation manifest after all reads |

Each layout or state case starts from a fresh page load at its width; the account scope, the insufficient-classification disclosure, the row detail's disclosure and the full-column table are opened after the row is selected and before assertions. On the real row at 1440 in English: the four period instants as `YYYY-MM-DD HH:MM:SS UTC`; billing account ID, account scope ID, tenant reference, delivery ID and ordinal equal to the response; the full row ID, visible, with its sentence; the provenance label for the row's class; Exact zero under the effective cost in the detail and no note in any table cell; the two-part column header; the instant in the table cell; the shared labels in the account list. Selecting an account reads "One account scope selected" and shows a 64-hex account scope ID.

Routed responses are patched from the live response of the same request, identically for the row and the selected row, and only when the response has the sample's single candidate row. In EN, KO and AR at 1440: a null billed cost, service and subaccount read Missing in the table and the detail; the effective cost keeps its zero state without a note in the table; a `Z` charge start is shown as collected with its note in the table and the detail; the candidate aggregate reads "No values present" with 0 / 1 and the insufficient-classification aggregate reads "No values present" and is visible; the first account reads Missing; the screen text has no superseded phrase, dash or typographic minus; no state node sits inside a left-to-right isolate. The same response in Arabic at 1251, 1024 and 768, and in Korean and Arabic at 390, where the visible card reads "Service: Missing", the source-null description state and Missing without a note. A synthesized second row reads Missing beside a present sum with 1 / 1 and no state on the aggregate. A negative billed cost reads `USD -2.00` in the row, the detail and the aggregate.

Layout with the detail and disclosures open in EN, KO and AR at 1440, 1251, 1250, 1024, 768, 767 and 390: no page overflow; no detail field, table cell, header, account field, card or total wider than its box or with a child outside it; no account value, raw instant in a cell, row ID half or detail instant broken across lines; at least one value node; in Arabic the row ID ends on its label's edge. A denied read leaves no row, source value, field, account or sum node. The existing view, detail, paging, account, stale and failure cases and the 24 view, language and width variants still pass.

Not asserted: the real row's subaccount value; cell heights; computed weights of the state nodes; the stacked account list's colon placement; widths below 390.

Host: macOS arm64, Node 26.7.0, Python 3.11.7 from the prepared environment, Playwright 1.56.1 with Chromium headless shell build 1243. Other operating systems, browsers and assistive technology are unverified. Captures stay local under `output/validation/HB-160` and are excluded from Git.

# Findings corrected during verification

The first suite run opened the account scope before the row selection reloaded the screen, so the list was closed when asserted; disclosures are now opened after selection. This was a suite error, not a product result.

# Independent review

Reviewer in each round: the repository's read-only `independent-reviewer` agent role (AI), 2026-11-12. It read files and executed nothing.

| Round | Result |
| --- | --- |
| Design review of the plan, before implementation | FAIL without blocker: 2 HIGH, 5 MEDIUM and LOW items. All resolved in the plan's "Design review resolution" table before code was written; the file split was dropped |
| Change review of the implemented diff | PASS with 1 MEDIUM and 7 LOW. MEDIUM: glossary section missing, added. LOW corrected: "unclassified" replaced by "counted as insufficient classification evidence" in English and Arabic; provenance asserted by its exact label; the 390 card assertions scoped to the visible card; patches guarded to the single-row response; plan verification text aligned with the suite. LOW left: see residuals |
| ui-designer render comparison | FAIL on two items, corrected and re-run: account buttons broke IDs and labels mid-token at 768 to 1024 and had uneven columns at 1440, now a fixed label-over-value grid (four tracks, two by two at 1250 and below, stacked at 767 and below) with a no-break assertion; a `Z` instant broke inside its date in the table cell, now kept on one line with the first two columns at 26%. Also corrected: "No values present" has one weight in both aggregates; the second header is two lines without a slash. The corrections were verified by the suite and by root reading the captures, not by a second designer round |

# Accepted residuals

- AI wording review only. Arabic and Korean practitioner review remains open. The shared provider label reads المزود while B07's toolbar and coverage note read المزوّد.
- B07's field states (Source null, Field absent, Blank value and the others) appear beside Missing in one detail. They are distinct by design: the eight marketplace fields carry a server state, the plain nullable row values read Missing.
- "No declaration" in the query and coverage evidence is rendered inside a left-to-right isolate. It predates this change and is not a row value.
- In the table the time zone mark can wrap below the time at narrow column widths.
- The detail below the table at 768 to 1250 is one full-width column of short values; unchanged structure.
- "Inspect source / 원천 근거 보기" shares its Korean with the B04 and B05 evidence toggle.
- No owner FIX was created or carried forward by this change.
