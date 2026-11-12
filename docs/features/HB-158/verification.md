---
type: VerificationRecord
title: B05 source evidence verification
description: Records what was executed and reviewed for the B05 source-field terms, value states and evidence layout on the retained sample, with results, limits and accepted residuals.
status: draft
---

# Delivered behavior

[HB-158](https://linear.app/hybrid-billing/issue/HB-158/align-b05-source-evidence-with-the-s04-source-field-terms-and-value) refines B05 from main `b834de8` using the retained public and synthetic sample verified in HB-149. Root (Claude Code) owns planning, implementation, terminology writing, QA and integration; a separate read-only reviewer reviewed the plan before implementation and the change afterwards; the ui-designer role compared renders. Machine: mac_mini. See the [plan](plan.md).

B05's source row picker, evidence, source summary, rule editor source line and account scope present collected row values with the S04 source-field terms: Missing and Exact zero, the charge period in UTC with start and end labels, billing account ID and subaccount ID in the picker, and row ordinal, sample provenance and the full row ID in the evidence, which opens in a full-width row under the record through a toggle in the availability cell. The separate full-columns table for mobile is removed. The source-value components first written for B04 now live in `sources/collected-focus/SourceValues.tsx` and serve B04 and B05. `/api/focus-overview`, the conversion and rounding arithmetic, validation and the browser draft format (`model.ts`, `values.ts`, `store.ts`) are unchanged, and B05's converted, rounded and compared amounts keep their format.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| Shared component unit test (`SourceValues.test.tsx`) | PASS 6: Missing alone without currency, dash or digit in three languages; sign, currency and exact digits; the zero note only on request; unit present, null, empty and blank; quantity states; UTC and raw instants and the labeled period |
| Console unit suite (`npm test --prefix apps/console`) | PASS 302, previously 296 |
| TypeScript check, Vite build, `tools/verify-build.mjs` | PASS; 438 artifact hashes and licenses; existing large-chunk advisory remains |
| B05 browser suite `apps/console/tools/fx-rounding-qa.mjs` on the retained sample | PASS; 0 page errors; see below. The unchanged suite also passed on main `b834de8` before any edit |
| B04 browser suite and render comparison | PASS; screenshots of the B04 screen body with an evidence row open and a row selected are pixel-identical between the unchanged build and this build in EN 1440, KO 390 and AR 1440. The markup differs by the renamed classes. Real rows carry `+00:00` instants, so the changed raw-instant rule is exercised on B05 only |
| S04 browser suite | PASS as regression |
| Isolated backend regression (`tests/focus_overview`) | PASS 36; unchanged backend |
| Retained source, control and prior-result files | PASS 93 file hashes equal to the HB-150 preservation manifest after all reads |

Browser cases on real rows, English at 1440 unless stated. VMware: billing account and subaccounts in the picker; no full-columns table and no disclosure in the picker; the evidence shows ordinal `1000`, Synthetic test data, the full row ID, both period instants with UTC and `USD 0.75`; opening a second row closes the first; selecting a row, changing the cost basis and paging on AWS each close the evidence; selecting an account in the account scope closes it too, the account list shows labeled billing account ID, subaccount ID and provider, and the selected account's summary shows billing account and subaccount; after selecting a row the rule editor shows `USD 0.25` with the basis label on the line below and the summary shows labeled start and end lines. Oracle: `USD 0.24` with `USD 0.00` and Exact zero in the evidence, in Korean and Arabic too; on the effective basis the rule editor and the summary show `USD 0.00` with Exact zero. AWS at offset 450: `USD -2.6137` in the picker and the rule editor. Alibaba Cloud: `CNY 8.80`. A 403 on a new query leaves no source value, evidence field or picker row in the error state.

A routed response derived from the real VMware response, valid for the client validator, in EN, KO and AR at 1440: a null amount on the selected basis reads Missing without currency, dash or digit in the picker, with the availability text; a null service and a null subaccount read Missing; the evidence shows Missing for the billed cost and subaccount, the Unclassified provenance label and the `Z` instant as collected; another row shows Missing for the effective cost; after selecting the degraded row the summary shows Missing for service and cost and the rule editor shows Missing with the basis label below, each carrying the missing state attribute. In English the summary period shows the raw instant, the not-converted note on its own line and the UTC end.

Layout with an evidence row open on Oracle rows in EN, KO and AR at 1440, 1280, 1001, 1000, 800, 641, 640 and 390: no page overflow; no direct child of a picker cell outside its cell; above 800 no horizontal scroll of the picker region; at 800 and below the picker is in card layout; at 390 the select and evidence buttons are at least 44px high. At 390 the degraded response in Korean and Arabic with evidence open passes the same cell check, and after selecting the degraded row the rule editor line is visible and reads Missing. The cell check runs in the picker with evidence open; the other views keep the earlier mobile cell-height check of the policies view. The existing arithmetic results (`KRW 7.40`, `KRW 8.25`), the check that the stored draft holds no source value, and the rule, preview, baseline, save, restore, guard, denial and late-response cases still pass, with their 24 view, language and width variants.

Host: macOS arm64, Node 26.7.0, Python 3.11.7 from the prepared environment, Playwright 1.56.1 with Chromium headless shell build 1243. Other operating systems, browsers and assistive technology are unverified. Captures stay local under `output/validation/HB-158` and are excluded from Git.

# Findings corrected during verification

- A measured 641px width made the table 31px wider than its region; the floor was lowered from 640px to 600px.
- A new attribute name collided with the account-scope buttons' existing attribute and was renamed. A routed response needed the provenance counts adjusted to stay valid. After a denied read the screen recovers through its "Reset row and retry" button, which the suite now uses. These three were suite or naming errors found by the suite, not product results.
- In the summary a raw instant ran together with its not-converted note on one line; the note now sits on its own line. B04 shared the rule and received the same correction.

# Independent review

Reviewer in each round: the repository's read-only `independent-reviewer` agent role (AI), 2026-11-12. It read files and renders and executed nothing.

| Review | Result |
| --- | --- |
| Design review of the plan, before implementation | FAIL without blockers: the rule editor's source-cost line missing from the plan; several planned assertions unobservable at their widths; rename hazards and style consequences of the shared move; B05-specific layout rules; the full-columns table unjustified once the evidence row exists; further wording sites; verification gaps. The plan was revised before implementation |
| Review of the actual change | PASS with conditions, no high or blocking finding: money semantics and the shared move confirmed correct by reading. Medium: the two unfinished rows of this record; the source-row column narrowest between 641px and 800px; the rule editor line splitting for values but not for Missing; assertion gaps (the editor line on mobile, closure on account change, an overstated "every view"). Low: the wording of the B04 comparison, the account summary without subaccount, an unclear residual. All were applied; the corrections were verified by root's rerun and renders, not by a further review |

# Design and wording review

The ui-designer role compared baseline and changed renders: conditional PASS. Blocking: between 641px and about 800px the fixed columns left about 100px for the source-row column. Should fix: the rule editor line breaking with a dangling separator, an amount splitting from its currency in the cost column, bold Missing looking like the selected-row bold, and the toggle height. Polish: spacing of the period lines and the segmented evidence borders. Applied: the picker turns into cards at 800px and below with capped button widths and a two-column evidence grid; the basis label sits under the amount in every state; a 192px cost column; selection is shown on the pressed button and no longer by bolding the service name; a 44px toggle; period spacing; one rule closing the evidence row. Not applied: a hover fill that tints the record but not its evidence row, and the tab strip cut at mid widths, which belongs to the shell. The designer judged the removal of the mobile full-columns table acceptable and noted that only one evidence row can be open, so rows cannot be compared side by side. The corrected renders were checked by root and the suite, not by a second designer pass; the designer could not read the mobile renders at text level and did not see the account list open.

Wording was reviewed by the change reviewer from source strings (AI). Open for a native or practitioner reviewer: Arabic spelling differences between B05 strings and the shared labels, the Arabic headings for sample provenance and row ordinal, and English "supplier conversion history" beside "Provider".

# Accepted residuals

- B05's computed amounts keep a leading minus and an explicit plus on differences while source amounts read `USD -2.61`; the rule editor's source cost and the rounded result appear in one view.
- Horizontal scroll of the picker region is asserted only above 800px, where the table always fits; at 800px and below the picker is cards.
- Only one evidence row can be open at a time. The Korean label 원천 근거 보기 is now used on B04 and B05 for this toggle and on B08 and B07 for a row-detail button with a different English label.
- The Escape key closes disclosures but not an open evidence row.
- A `Z` instant is UTC but is shown as collected with the not-converted note, because the shared parser reads only `+00:00`.
- After a denied read the screen recovers through its reset button and not through a later scope change; this predates the change.
- U01, B06 and B07 keep older wording until their own tickets; HB-152 is unchanged.

# Acceptance

Parent HB-105 remains awaiting explicit owner screen acceptance. This change alters the B05 candidate revision; no acceptance exists for B05 and none is created. Moving the shared components changes B04's source fingerprint without changing its rendered behavior. No ingestion, reset, reimport, schema or permission change, live account, customer data, deployment, payment or message occurred. Merge, read-back, ticket completion and machine release are recorded in the ticket only after they occur.
