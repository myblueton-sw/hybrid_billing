---
type: VerificationReport
title: HB-85 recomposed overview verification
description: Separates actual font, layout and interaction checks from independent visual critique and pending owner acceptance of the rejected overview replacement.
status: draft
---

# Candidate and rejection history

The owner rejected the first PR77 candidate on 2026-11-03 for typography, global layout and usability. Earlier technical passes did not establish commercial design quality. This report covers the complete U01 viewport replacement, not just the old body. Final source revision is `008bd182b66a`. Owner acceptance remains pending.

Host: macOS arm64, Node 26.7.0, npm 11.19.0, Chromium headless shell 153.0.8010.12. The Node/npm/browser implementation remains portable by design; Windows/Linux and other browser engines are unverified. No production services or customer data were used.

# Actual root verification

From `apps/console`:

| Command/check | Result |
| --- | --- |
| `npm run build` | PASS: TypeScript and Vite build; existing bundle-size advisory remains |
| `node tools/verify-build.mjs` | PASS: 438 artifact hashes, HTML and seven license notices |
| `npm test` | PASS: 79 tests, zero failures |
| `npm run qa:overview -- --url <loopback-preview> --playwright-module <installed-module> --chromium <installed-browser>` | PASS: 12 small/enterprise EN/KO/AR desktop/mobile renders and scoped journeys |
| New scoped CSS guard | PASS: no literal colors, radius, shadows or gradients; canonical asset hashes verified by build |
| Staged/source diff checks | PASS before commit |

Rendered checks cover 1440px and 390px, exact currency-separated costs, partial/all-missing/verified-zero/empty/unconnected states, source/organization/month/currency filters, 2,400 departments and 4,800 closing units, bounded pages, hierarchy selection, URL scope reset, list-only filter invariance and exact detail focus entry/return. The chart-currency control retains its ID in the new chart-header location.

At 1440×900, all three locales show the full plot and at least four work rows, with title-to-toolbar height under 200px. Compact view renders work before amounts and trend in the DOM. Money strings remain within the compact viewport. Computed headings/body/table/control/metadata roles match the current design. Actual Sans KR/Arabic 600 faces load; font-family, 600-face loaded status and font readiness were checked, rather than assuming a CSS weight means a real face exists.

Axis label centers and gridlines align within the scoped tolerance at both widths. Nice ticks/headroom use bigint scaling and percentage-only Number conversion; exact amounts are unchanged. A standalone axis test covers normal, zero/sub-unit and huge bigint inputs. The scale is for the current nonnegative fixtures; signed-cost plotting is not established.

Review ID, revision, status and confirmation remain visible at both widths. Isolated QA opens the review dialog, verifies the U01 scope, cancels with Escape and checks focus/status restoration. It never acknowledges or saves owner acceptance. Final captures and local request/font evidence are generated under `output/validation/HB-85-revision`; no external runtime requests were observed in the six capture sessions. Full 12-view captures remain under `apps/console/qa-output/overview`. These generated files and dependencies are excluded from Git.

# Independent findings and corrections

A separate source/financial reviewer passed the recomposed scope, then independently ran all three chart-scale tests and passed the axis/Paging follow-up. It checked whole-unit versus missing-record identity, overlap, exact stage/record inspection, scope invalidation, currency meanings, unavailable states and preserved review controls. This is not commercial design acceptance.

Claude CLI ui-designer used the installed official frontend-design skill and rejected-render evidence to propose a complete viewport composition. Its first revised visual critique found floating chart-axis labels, absent gridlines and an overemphasized repeated headline. Root introduced aligned readable ticks/gridlines and headroom, reduced headline emphasis, aligned actions and restricted the selected underline to the month label. Final formal critique of revision `0a16f75fa54a` visually passed the first viewport and chart, while leaving minor lower-screen defects and owner acceptance separate. It directly inspected all three desktop first screens, KO/AR compact charts, KO lifecycle/compositions/detail and AR compact compositions. EN compact and full AR lifecycle/detail were not independently visually reviewed.

Root subsequently corrected the minor incomplete bottom borders, first lifecycle-header fill, completed/not-applicable link density, detail amount alignment and inherited 12px field labels (now checked at 14px), retaining keyboard focus and actual actions. The final root captures and browser rerun cover these changes; the independent visual verdict is not represented as having inspected that later polish.

The visual review questioned organization C's KRW zero. The unchanged model explicitly creates a recorded zero for that on-premises fixture; existing numeric/browser checks verify it. It is distinct from source/currency combinations without records. No missing value was converted to zero or not applicable.

# Limits

This remains a synthetic screen candidate. Production authentication/permission projection, overview aggregation, sales/allocation/receipt ledgers, authoritative ERP, deadlines, owners and financial commands are unconnected. Native-language and assistive-technology acceptance, signed costs and non-Chromium table semantics remain unverified. Shared real font faces can affect other consumers; their screen acceptance is not inferred. No owner FIX, main merge or deployment occurred. The owner decides whether this candidate meets the intended commercial standard.

# Subsequent finance and billing review

Two separate AI specialist-role reviews of this same source/build returned FAIL for the default enterprise home. They assessed role decisions and workflow rather than repeating technical checks. [The report](finance-billing-review.md) confirms non-FOCUS fixture provenance, records required design changes and separates human acceptance from role-based AI critique. Its billing findings and provenance were independently read back for report accuracy. No application change or new runtime test was performed in that review step.
