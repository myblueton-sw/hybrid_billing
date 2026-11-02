---
type: Design
title: Enterprise organization overview candidate
description: Defines bounded organization navigation and leaf-level synthetic aggregation for large organization screen validation.
---

# Scope and ownership

Owner requested large-organization design and implementation. Root is the sole writer on HB-140; an independent reviewer owns scope/financial review and Claude CLI owns visual composition/review. Changed overview items are organization hierarchy, scope profile, paginated breakdown, closing results and evidence details. Existing acceptance is not carried forward; no owner FIX or merge is claimed.

The enterprise example contains twelve companies, ten divisions per company and twenty departments per division (2,400 departments). The existing three-organization profile remains available. This is synthetic source-cost data, not a production organization or permission model. Default all means all departments in this example workspace/profile, not globally authorized data.

# Scope and amount contract

Compact selection tokens select all, none, a company/division subtree or explicit departments. Parent and child overlap resolves to a union of unique department IDs before totals. Unknown tokens contribute no department; empty or entirely unknown selections never broaden to all. Amounts exist only at leaf level. Source and organization scope intersect. Currency totals remain separate; this does not implement group consolidation or intercompany elimination.

Searching or paging changes visible results only. It never selects all search results implicitly and never changes global scope totals. Explicit selection actions commit immediately. Workspace/profile/month/source/organization changes invalidate details and result pages. Browser history restores committed URL scope; transient picker search/page is not a financial selection.

All potentially large result surfaces render at most twenty entries per page, with exact fixture totals and ranges. The frontend fixture population is loaded locally for demonstration only. It is not server-side paging or evidence of production throughput.

# Production boundary

Production requires authenticated server-side scope resolution and aggregation, permission/disclosure revision, snapshot identity, explicit reporting basis and currency, stable cursor pagination, hierarchy search, cancellation and rejection of stale requests. Client URL IDs cannot grant access. Server records and aggregate queries must enforce the same scope; no full production population should be sent to the browser. These endpoints, authorization and real organization migration are deferred, not simulated as successful connections.

# Validation

Implementation review and runtime verification completed for the synthetic frontend; visual re-review status is recorded below. Independent design review passed with conditions: unique leaf union; fail-closed invalid scopes; search/page independent of totals; all large surfaces bounded; reset across scope boundaries; independent controls for 2,400 leaves and overlapping/disjoint parents. No server throughput, Windows/Linux, native-Arabic or screen-reader acceptance is claimed.

# Design evidence and implementation choices

External references checked 2026-11-02: [Microsoft Cost Management scope hierarchy](https://learn.microsoft.com/en-us/azure/cost-management-billing/costs/understand-work-scopes) supports navigating nested cost scopes; [AWS Organizations hierarchy navigation](https://docs.aws.amazon.com/organizations/latest/userguide/navigate_tree.html) distinguishes hierarchy and list views. Adoption is searchable hierarchy navigation and explicit scope, not their permission or financial semantics.

Claude CLI inspected the preceding KO render and proposed an inline hierarchy picker, full-width organization breakdown, a separate example-size selector and bounded result panels. Root retained the independently reviewed 12×10×20 fixture instead of the CLI's alternative 3×8×100 proposal. Compact selection is a versioned 600-hex-character bitmap tied to the fixed synthetic department ordering, with all/none shortcuts. It is not a production identifier/authorization contract. Search is explicitly by code within the current path; arbitrary organization names and real directory search require the future service. Parent checkboxes include descendants and show partial selection. Source-cost details preserve the current selected-source intersection. Department × internal/external path is an explicit synthetic closing-unit assumption.

Independent actual change review: PASS after correcting stale-page restoration and picker context invalidation. Reviewer independently ran 46 passing unit tests. Initial browser QA found mobile exception-button overflow; root corrected wrapping. Controlled hash-driven checkbox tests now wait for visible closing counts after clicks instead of asserting synchronous checked state.


# Verification results

PASS: TypeScript/Vite build, 46 unit tests, 242 artifact hashes. The overview browser suite covers twelve small/enterprise EN/KO/AR × desktop/mobile renders, 2,400 departments / 4,800 closing units, at-most-twenty rendered rows, searched department removal, partial parent selection, compact URL, parent detail, profile reset, month round-trip reset and all previous small-profile scenarios. Supplemental fresh-context browser checks cover open picker EN desktop and KO/AR mobile overflow, 20-row search results and picker page reset after changing month and returning. Screenshots remain generated local QA artifacts. No owner acceptance records were created.

Commands: `npm --prefix apps/console run build`, `npm --prefix apps/console test`, `node tools/overview-qa.mjs --url <candidate> --playwright-module <installed-module> --chromium <installed-browser>` and `node tools/verify-build.mjs` (the last two from apps/console). Host: macOS arm64, Node26.7.0, existing Chromium headless shell. Existing build chunk-size warning remains. No dependencies or OS-specific runtime requirements added.

The first visual review rejected the missing mobile open-picker grid span. Root added full-grid span, aligned monetary cells, unified row-link/pager styles and clarified included-department wording. Root regenerated open-state captures. A second visual review identified inherited column direction in picker labels; root set explicit row direction, natural checkbox sizing and start-aligned search, then verified computed styles and fresh EN/KO/AR crops. Final targeted Claude CLI re-review PASS; prior corrected visual findings remained PASS. Native Arabic and screen-reader acceptance remain unverified.

Candidate: http://127.0.0.1:8790/#overview?lang=ko&ov-scale=enterprise. Draft PR66; no merge or owner FIX.

CLI-only telemetry for four calls (design, two FAIL visual reviews, final PASS): {"input_tokens": 44, "cache_creation_input_tokens": 74867, "cache_read_input_tokens": 230409, "output_tokens": 20980}. No root-turn total or no-skill control measurement is available; this is not a savings claim.
