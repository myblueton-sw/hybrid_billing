---
type: Implementation
title: Complete planning-screen preview candidate
description: Records the 49-screen preview coverage, domain compositions, evidence, review decisions and unconnected boundaries.
---

# Owner request and delivery scope

The owner requested applying all screen candidates against the completed planning baseline. HB74 owns one integrated preview branch from main 36394ed, with root as the sole writer and Seung Woo Park as accountable owner. Existing HB75–79 and screen tickets remain requirement/dependency references; this candidate does not silently mark their production contracts or acceptance complete. HB86's reviewed closing body is incorporated from candidate f517ff3 without merging PR67 or changing main.

The preview covers the 49 canonical screen IDs in HB72, their planning paths, grouped admin navigation, separate customer projection, pre-login entry and individual review controls. Module-owned `screens.json` and `candidates/<ID>.json` files hold screen-specific columns, tab views, field evidence and authored examples. Shared components implement actual table, detail, draft, wizard, tree, inbox, permission-matrix, safe-document, disclosure-preview, chart and conversation interactions. A screen directory is navigation tooling, not a substitute for these bodies.

Every new example is visibly synthetic. Most new list candidates contain three authored records; this is not a production-scale dataset. U01 retains HB140's enterprise/monthly overview; U02 retains the HB86 list/calendar/detail. Entry components retain existing identity-preview behavior. No real credential, customer data, financial calculation policy, authentication, approval, issuance, delivery, payment or deletion service is connected.

# Planning decisions and trace

Read-only domain extraction inspected all 49 authorized local `docs/ui/screens/<ID>/spec.md` files and the screen binding register. Business extraction covered O01–O03, S01–S06, B01–B11 and I01–I04. Access/operations extraction covered L01–L03, U01–U04, R01–R02, A01, M01–M05, P01–P07 and C01–C03. Neither extraction is runtime verification.

- D1: this planning-first candidate presents S02's five planning steps. The 112-entry legacy redirect table retains the eight historical onboarding targets as compatibility references. This is not a claim that their previous mutable workflow has migrated unchanged.
- D3: S05 remains internal; customer document/evidence presentation is confined to C02. No customer permission is inferred from historical CU wording.
- D7/D8/D9: specifications govern over older design JSON; primary work appears before appended payment/recognition tabs. Commands whose policy or ordering is unresolved are unavailable.
- D11: M05 appears under My account, following HB72's navigation decision.
- D13: the integrated shell uses HB72's recorded owner-approved 14px work density, with 16px mobile text inputs and existing hosted bodies retaining their own scoped type. This explicitly identifies the conflict with the older local 16px guide; no archive token is altered.
- Binding-register additions are surfaced as context where unambiguous, but conflicting financial policies are not executed. B04 adjustment approval is not policy approval; B05 has no inferred FX/markup/fee defaults; B08 has no executable stale-dependency override. Operational notification rules remain separate from M05 preferences. Portal Assistant is excluded.

U01 hosts the newer HB140 overview rather than reverting to HB72's older generic operations body. Its original item prefixes remain emitted by the hosted component. U02 and entry bodies also retain their own item prefixes. Newly composed bodies use canonical planning IDs. The legacy-to-planning map records historical screen relationships, not an assertion that every previous implementation item is visible or accepted in the new body. Existing source implementations remain available in Git; no prior local FIX is copied to changed fingerprints.

# Visual design evidence

Root consulted official sources on 2026-11-03:

- [Cloudscape resource views](https://cloudscape.design/patterns/resource-management/view/): collection tables and contextual details. Adopt bounded collection/detail interaction; reject vendor branding and unnecessary split panes.
- [PatternFly dashboard](https://www.patternfly.org/patterns/dashboard/): dashboard composition reference. Retain scoped facts and equivalent tabular values; no borrowed assets or invented production metrics.
- [Carbon form styling](https://v10.carbondesignsystem.com/components/form/style): field widths depend on content within a responsive layout. Adopt grouped fields; reject a centered fixed-width page container.

The actual restricted Claude CLI ui-designer inspected prior desktop Korean closing and mobile Arabic calendar renders, HB72 structure and the authorized frontend-design skill. Its proposal specified grouped 248px navigation, full-width frame, neutral borders, 14px work density, compact scope controls, separate portal navigation, vertical wizard steps and distinct domain compositions. Root implements the proposal with native buttons/disclosures/dialogs; responsive tabs wrap rather than clip. Modal detail is used on mobile instead of introducing an unverified object-navigation history contract. Screenshots and subsequent findings are local generated evidence, excluded from Git.

# State, scope and review contracts

The resolver receives audience and preview visibility separately from the URL. Customer U03 has a separate data configuration; M05 is self-scoped. Unknown or forbidden objects return no object data. Query parameters confer no authority. The synthetic persona selector is explicitly review tooling, not authentication or a server permission grant. Real permission catalog/action binding remains incomplete and non-executable.

Changing workspace, organization, period, currency, object, persona or mode invalidates candidate detail and drafts. Explicit navigation and browser-history navigation ask before discarding dirty input. The review context includes those normalized values, audience, tab, scenario and emitted item IDs. The parent does not erase initial child review state. Local acceptance remains owner-only and browser-scoped, with no assistant-created project FIX.

Unconnected/loading/empty/error/revoked/partial examples expose honest state without fixture tables. A public-document view never falls back to a record outside the filtered population; charts and disclosure previews are bound to filtered records. Unknown policy values remain unconfigured. Disabled domain command controls explain why execution is unavailable. Draft review stores only volatile local input and does not produce a server job or success claim.

# Verification and remaining boundaries

Independent routing design review passed conditionally; complete action-policy mapping is unverified. Actual routing review found prototype-property legacy lookup failure; own-property lookup and regressions fixed it, and re-review passed with 53 tests. Runtime review then found out-of-scope document/chart presentation, missing object selection, incomplete reset/review scope, erased initial item IDs and draft/history discard gaps. These findings were reproduced and corrected; final re-review and browser results are recorded at handoff.

The first complete matrix passed 294 renders (49 screens × KO/EN/AR × 1440/390) and 164 tab transitions before the additional semantic findings. This was route/layout coverage, not full contract acceptance. Unit tests including the incorporated closing model passed 56 cases. A first browser-history guard failed the added regression; it was replaced with a central navigation guard before React subscriptions. Do not treat the earlier matrix alone as completion.

OS neutrality is a design constraint: product code uses browser/Node interfaces and local assets. Actual execution is macOS arm64, Node 26.7.0/npm 11.19.0 and Chromium headless shell build 1243. Windows/Linux/other browser engines, native-language practitioner approval, full accessibility, complete action contracts, real integrations and large-scale performance remain UNVERIFIED. Full child-ticket completion, owner FIX and merge are not implied by this candidate.

Visual comparison of seven representative renders identified five required composition fixes: oversized mobile header, wrapping mobile review identity, open mobile scope controls, gallery-like approval inbox, and a flat organization tree. Root moved mobile header controls into the menu, kept a compact review ID/status row with revision inside confirmation, collapsed scope on mobile, introduced queue-plus-selected-detail with item-scoped disabled commands, and implemented indented keyboard-navigable tree items. Canonical logo geometry is unchanged; RTL changes placement, never mirrors the asset. Remaining visual re-review is recorded below.

# Final scoped handoff

Final integrated build and TypeScript: PASS. Unit suite: 56 PASS. The complete rendered matrix passed 294 views and 164 tab transitions; the subsequent focused semantic run passed after the final inbox changes. The focused run intentionally records zero matrix renders/tabs, not another full-matrix execution. It checks exact selected Identifier, a single item-bound approval command group, filter-empty boundaries, unknown objects, review context, unsaved drafts and browser back, tree keyboard interaction and responsive controls. A mobile assertion initially raced the responsive render; it now waits for the closed-scope selector. The fresh capture run passed. Build verification checked 242 final artifact hashes including HTML and seven license notices. Vite retains a non-blocking bundle-size warning (626.36 kB main JS).

Independent read-only source review passed the corrected routing, filtered-view, state/review and navigation boundaries, then specifically passed canonical Inbox identity and the removal of duplicate page commands. Claude CLI inspected nine representative refreshed images; four original findings passed, while B09 duplicate page commands failed. Root removed that duplicate and corrected its Identifier. A final actual B09 English desktop PNG review passed the remaining finding. These are scoped source and visual reviews, not full 49-screen specification, financial-policy, localization or production-security certification. Minor visual observations remain: redundant boundary wording, some duplicated scenario summaries, and density/alignment refinements.

The portable entry points are `npm ci --prefix apps/console`, `npm --prefix apps/console run build`, and `npm --prefix apps/console start`. Browser QA uses `npm --prefix apps/console run qa:planning -- --url <preview-url> --playwright-module <installed-module> --chromium <executable>` from the repository root. `--quick yes` runs only semantic checks. The local review URL is `http://127.0.0.1:8792/#/design/screens?lang=ko`; it depends on the active local preview process. Existing legacy-shell browser suites were not represented as passing against this replacement.

No owner acceptance record was created. This candidate awaits owner review and authorized merge; no child ticket or PR67 is completed by this handoff.

# Measured CLI usage

Four actual Claude CLI design/review calls reported input_tokens: 20, cache_creation_input_tokens: 91,561, cache_read_input_tokens: 67,599, output_tokens: 20,058. These categories are provider telemetry, not total Codex/team usage. No equivalent no-skill/no-cache control run exists, so token savings or efficiency percentages cannot be computed. JEV invocation and whole-session token usage are not measured here.
