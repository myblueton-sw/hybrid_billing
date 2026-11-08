---
type: Implementation
title: U02 read-only closing list and calendar candidate
description: Records the scoped closing-screen body, external design evidence, synthetic contract and integration boundaries.
---

# Scope and dependencies

Owner requested the next screen after HB140 merged. HB86 owns U02, implemented by root in the operations module. Accountable owner: Seung Woo Park. Existing `core-index-close` route hosts this read-only candidate. Full `/closing`, common permission registry, navigation/frame/data components and body migration remain dependent on HB75–79. This does not certify full U02 completion or replace those dependencies. Catalog title/description are corrected only for this existing screen, and package test entry plus existing work QA are updated to cover its changed body; no shell or navigation structure is changed.

Source contract: authorized `docs/ui/screens/U02/spec.md`, screen binding register and HB72 registry. Local English U02 spec governs over older JSON omissions. D9 ordering uses list/calendar first, with recognition context after history. D7 mutation workflow remains deferred; no guessed action sequence is executed. Root is sole writer; independent reviewer checks domain semantics and Claude CLI owns composition/render review.

# External design evidence

Checked 2026-11-03: [Oracle task views](https://docs.oracle.com/en/cloud/saas/financial-consolidation-cloud/usfcc/about_views.html), [Oracle task summary and detail](https://docs.oracle.com/en/cloud/saas/financial-consolidation-cloud/usfcc/cm_tasks_viewing_home_page.html), and [SAP closing tasks](https://learning.sap.com/courses/close-orchestration-with-sap-advanced-financial-closing/running-closing-tasks_ae8840c3-ae93-4502-96fa-43a60403e5c3). Observed: alternate list/calendar presentation, period/status filtering, task-specific detail, blocker logs and audit history. Adopt those navigation/information patterns. Reject vendor visual branding and execution/approval controls whose local contracts are not integrated. No assets copied.

Claude CLI inspected the prior generic WorkQueue render and the supplied frontend-design skill. Proposed compact filters, separate state counts, original/current deadlines side by side, date grid/mobile agenda, and detail with conditions, evidence, history then recognition. Root uses native buttons for list/calendar mode (not a custom tab widget), a native modal detail with Escape/focus return, and a horizontally scrollable complete table on mobile. Mobile card rendering, interactive changes and production calendar policies are deferred rather than silently approximated.

# Synthetic data contract

1,200 unique closing targets for either September or October 2026 settlement cycles, with current stage deadlines in the following month. Each target has exactly one displayed operational stage schedule, so calendar counts are unique target counts for that schedule, not all stage events. Original/current timestamps and revision are explicit fixture data. Display timezone is UTC, including local/UTC equality in this fixture; business-day/holiday/DST policies are unconfigured and never computed. As-of is fixed at 2026-11-03T12:00:00Z; overdue means an unfinished, non-exempt target has passed its current deadline. Original deadline remains visible after postponement.

Separate states are incomplete, blocked, held, exempt and complete (240 of each per cycle before filtering). ERP unknown blocks applicable internal and external contracts. Internal issuance/delivery conditions are inapplicable. Complete rows have no unknown required ERP outcome. Exempt is an evidenced exception, never completed; no amounts are invented or removed. All references/owners/contracts are synthetic. Monthly ERP outcome is distinct from quarterly reconciliation, which is not connected.

Filters intersect cycle/workspace, organization-contract-entity code, owner, stage, state, path and overdue flag. List and calendar share the full filtered population and stable target IDs. Date selection narrows the same list. At most twenty rows render. Filter/scope/state changes invalidate detail and pages; presentation changes preserve filters. Detail displays original/current schedule history and fixture evidence IDs without implying a live evidence viewer or download. Actions requiring approval, assignment, exemption, reopening, schedule or notification authority are explicitly deferred.

# States, review and scale limits

Unconnected mode constructs no fixture targets. Local empty/error/revoked switches are labeled simulations; they do not implement authentication, permission revocation or a service retry. Governed mutations, role preview differences, server cursor/snapshot expiry, loading/partial async state, notification rules and full monthly/quarterly reconciliation remain pending foundations/services. The 1,200-row fixture is interaction evidence, not the specification's 100,000+ target/server-scale verification.

Legacy screen review remains available in the shell. New U02 item IDs cover scope/filters, view, counts, targets, calendar, page, conditions, history, recognition and service state. Existing generic WorkQueue item semantics are not silently carried forward. No owner FIX, merge or deployment is authorized by implementing this candidate.

# Review and verification

Independent design review PASS with conditions: stage-specific identity, full-filter calendar counts, original deadline retention, required ERP checks on internal paths, no inferred calendar policy and no production-scale claim. Runtime and actual change reviews are recorded at handoff.


Independent actual-change review initially found two medium issues: confirmed ERP evidence could display as pending in the condition table, and an unsupported path filter fell back to external targets. Both were corrected; independent re-review passed and independently ran all 50 unit tests. The browser regression also checks confirmed accounting evidence directly.

Claude CLI render review required detail table row-header separators. The first CSS border fix did not resolve visible overlap; computed styles revealed global sticky `th` positioning. Scoped static positioning for detail row headers and removal of the list row headers' vertical sticky offset resolve the cause. The closing body explicitly retains the active 16px baseline. Cycle and deadline month are separately labeled in the calendar heading. Final visual re-review is recorded below.

Actual checks on macOS arm64, Node 26.7.0/npm 11.19.0 and Chromium headless shell build 1243:

- `npm --prefix apps/console test`: 50 tests passed.
- `npm --prefix apps/console run build`: TypeScript and Vite passed; existing chunk-size advisory remains.
- `node tools/verify-build.mjs` from the console directory: 242 artifact hashes verified.
- Existing browser/work/entry/setup suites: 37/32/50/21 checks passed; overview suite passed including its 12 locale/scale renders.
- `node tools/closing-qa.mjs --url http://127.0.0.1:8791/` with configured Playwright module and Chromium paths: 18 list/calendar/detail renders passed across EN/KO/AR and 1440/390 widths, plus shared filters, full filtered calendar counts, date identity, original/current history, ERP condition evidence, focus/Escape return, scope invalidation, empty/error/restore/revoked/unconnected states and neutral border checks. Rerun after final style fixes passed.

Screenshots and raw logs are generated local evidence, excluded from Git. Browser tests use fresh isolated contexts and do not record owner FIX. Candidate preview is `http://127.0.0.1:8791/#core-index-close?lang=ko`. Windows/Linux, other browser engines, native-language practitioner acceptance, server-scale behavior and production permissions remain UNVERIFIED. Product code uses browser/Node interfaces; host-specific browser paths are optional QA arguments, not installation requirements.

Final Claude CLI visual re-review: PASS, no required blockers, based on updated English detail, Korean mobile detail/desktop list and Arabic mobile calendar screenshots plus scoped CSS. At screenshot resolution the neutral row-header borders remain low contrast; shared Arabic top-tab clipping is outside this body scope. Native language and full assistive-technology acceptance remain pending.

Telemetry from four actual Claude CLI design/review calls (reported model `claude-opus-5-5`): input 22, cache creation 62,528, cache reads 73,595, output 13,156 tokens. These are CLI-only usage categories, not total root-agent usage. No no-skill/no-cache control run was executed, so no measured token savings or JEV effect is claimed.

Handoff: read-only candidate ready for owner inspection and draft PR review; no owner FIX, merge, live service integration or full U02 completion is recorded. HB75–79 integration dependencies remain open.
