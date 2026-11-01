---
type: DesignWorkflow
title: Claude CLI visual design leadership
description: Defines the owner-requested Claude CLI design role, evidence handoff and actual execution record.
---

# Ownership and workflow

The owner rejected the prior entry composition and requested that future UI visual design be led through Claude CLI. Root remains accountable for scope, frontend implementation, integration and actual browser checks. The repository role is `.claude/agents/ui-designer.md`; mandatory style constraints remain in `.agents/rules/service-screen-style.md`.

The design lead receives current rendered desktop/mobile images, existing source, canonical assets, actual external service reference captures and the owner's constraints. It diagnoses visual hierarchy and proportion, proposes one coherent composition, critiques the proposal and supplies implementation-ready specifications. Root implements that specification, then returns new renders to the same CLI role for visual comparison. A CLI response, model name or visual PASS is not human-designer certification, owner acceptance or production readiness.

The official installed `frontend-design` skill provides visual-design process guidance. Owner rules override skill defaults, including full-viewport service layout, neutral borders, canonical logo and local IBM Plex, square geometry, no gradients/shadows/decorative accent strokes, preserved IA, EN/KO/AR and RTL. Full-viewport layout does not mean controls must be stretched to the viewport width. No invented authentication, customer claims or service capability is allowed.

# Invocation boundary

Use the locally installed Claude CLI with the inherited CLI model. Explicitly load the ui-designer agent definition and allow only Read/Glob/Grep. The initial invocation uses restricted mode, empty strict MCP configuration, no session persistence and an explicit task brief. No shell execution, writes, nested agents/CLI, form submission, deployment or external messages are granted to the design agent. Root owns repository writes.

CLI is development tooling; it is not a dependency of product installation or operation. The agent definition contains no OS-specific skill path. Each invocation supplies the verified local skill location and only the required additional read directory. Do not copy third-party skill source or reference images into maintained product code. If authentication, the CLI, images or skill are unavailable, report that boundary instead of claiming successful design work.

# Initial execution inputs

Claude Code 2.1.316 was found and authenticated. Authentication account metadata is deliberately omitted. The official local frontend-design skill was read; SHA-256: `d91970639e9f5c37682ac7ab60094d35f1c7c1f38d731bd56396563aee10c1d3`. Its source and license were not copied into this repository.

The design request supplied four current start/login KO/AR desktop/mobile captures, seven relevant source files and existing tokens. Previously captured official Atlassian/Microsoft references were supplied as local visual evidence; the CLI must not claim to have browsed them itself. GitLab was excluded as unavailable sign-in evidence. The owner's reported problems are disproportionate controls, excessive empty space and incoherent overall scale.

# Returned design assessment

The actual invocation completed successfully (`is_error=false`, subtype `success`, 19 turns, 236048 ms, no permission denials). Reported model: `claude-opus-5-5`, inherited from the CLI environment. The reviewer reported reading the assigned skill, constraints, source and supplied images. It did not browse, run commands, render pages or execute tests.

Observed defects: approximately 748px-wide desktop controls, disconnected title/form regions, large empty areas below the 40/60 panes, inconsistent alignment edges and a mobile header approximately 165–170px high. These are the reviewer's image estimates, not browser measurements.

The proposed replacement is a full-viewport white grid with aligned title/primary/alternatives/context regions; no centered page container. Desktop fields are capped at 400px, primary actions use content width, and secondary routes become neutral separated rows capped at 640px. Shared gutters scale from 24px to 64px. Below 900px the layout becomes a single column. Mobile controls use the available column width, and the language selector becomes compact while retaining its accessible name. Existing IA, four entry paths, unconnected notices, preview separation, logo and IBM Plex remain constraints.

The reviewer adopted narrow readable form columns and explicit alternate paths from the supplied Atlassian/Microsoft captures. It rejected floating cards, shadows, illustrations and copied provider options. Reference sources: https://id.atlassian.com/login and https://login.microsoftonline.com/. GitLab was blocked and supplied no usable sign-in evidence.

This is a proposal, not an implemented or accepted replacement. The reviewer explicitly identified remaining risks: a two-column design may still resemble the rejected split, wide-screen whitespace remains unproven, secondary rows may appear generic, and shared CSS/component internals require verification. Root must resolve these concerns during implementation and return actual EN/KO/AR desktop/mobile renders to Claude CLI. No prior owner FIX is fabricated or overwritten by this report.

# Validation and measured usage

PASS: actual read-only CLI invocation returned a substantive screenshot-based design assessment; role and mandatory ownership rules are recorded. PASS: scoped document/role content manually checked against the invocation and owner constraints. No application source changed in this workflow revision; no new browser/build tests were run for it. The proposed screen implementation and follow-up rendered review remain pending.

CLI-reported token counts for this invocation only: uncached input 12; cache creation input 40050; cache read input 91902; output 24966, including reported thinking 16210. These categories reflect cumulative CLI telemetry, not unique source size or root-turn usage. Restricted mode ignored settings/hooks; JEV use inside this invocation is not claimed. Total root-turn token/cache accounting and a no-skill control run are unavailable. No measured token savings or causal optimization comparison can be stated.

# Professional console visual revision

On 2026-11-01 the owner reported that the whole console still did not read as a professional solution and asked Claude CLI to finish HB-67. Claude CLI acted as root and visual design lead in this revision. IA, routes, six-domain order, header controls, review-before-content placement, canonical item IDs and screen functions are unchanged. No owner FIX is created or carried forward; every implemented screen's shared-shell fingerprint changes and returns to review.

## Diagnosis from actual renders

Before-state captures at 1440 and 390 pixels showed: about 190px of stacked chrome (preview strip, 96px header, tabs); 44px controls and 16px text applied uniformly, making work screens resemble a document; teal edge strips on section headings, metric cards and the active rail item, which also violated the neutral-border rule; underlined identifiers in every table row; full panels for related links; a 3px focus outline drawn around the programmatically focused main region on every route change; and entry screens whose title and form were split into disconnected panes.

## External evidence

Fetched in this session:

- IBM Carbon data table style (https://carbondesignsystem.com/components/data-table/style/): 14px regular body cells, 14px semibold column headers, row sizes 24/32/40/48/64px.
- AWS Cloudscape typography (https://cloudscape.design/foundation/visual-foundation/typography/): 14px/20px default body, 24px page heading, 20/18/16px lower headings.

Adopted: 14px work density, compact 36px controls, 40px header row and 44px body rows, restrained heading scale. Rejected: rounded corners, shadows, colored row/selection accents and provider-specific visual identity. The earlier Atlassian/Microsoft sign-in evidence above remains the basis for a narrow entry form column with explicit alternate paths.

## Adopted specification

| Area | Specification |
|---|---|
| Chrome | 28px neutral preview strip, 64px white header, 44px domain tabs. Active tab uses ink-colored 2px underline and semibold weight; no accent color. |
| Logo | Canonical color lockup at 32px height (155px width), unchanged geometry, LTR at inline-start. The mark spans 36 of 48 viewBox units, so it renders at 24px with 16px vertical clear space (minimum 12px). Mobile uses 28px height: 21px mark, above the 20px minimum. |
| Type | Work area 14px body (Arabic 15px), 13px secondary text and table headers, 12px captions, 24px page title, 16–18px section titles. Entry fields use 15px input text. |
| Controls | 36px on fine pointers, 44px on coarse pointers through `pointer: coarse`, including review, bulk, pager, related-link and row-link controls; table checkboxes enlarge to 24px. Primary navy, secondary white with neutral border. Disabled uses neutral fill. |
| Tables | Bulk bar attached to table, sticky 40px header, 44px rows, tabular numerals, monospace record IDs without permanent underline, neutral hover. Row height matches the planning default in `docs/planning/v1.16/ui-design.md`. |
| Navigation | Baseline 246px rail (210px below 1280px, horizontal below 1024px) with domain heading and count; active item uses neutral fill and weight. |
| Surfaces | Neutral page background, white panels with hairline neutral borders, no colored edge strips, shadows or radius. Focus ring 2px; main region no longer shows a focus outline after route changes. |
| Entry | Title, principal action and alternatives in one aligned 440px column; alternate paths are separated rows; installation status or help in a 400px neutral context column at inline-end. Single column below 1024px. Secondary entry screens use a 560px form column with a 360px help column. |

The 16px console body override in the design-system guide remains the documented baseline elsewhere. This revision deliberately applies 14px density to HB-67 work surfaces based on the evidence above; Arabic stays at 15px with the 1.8 line height, with 14px secondary text and 13px captions. The owner approved the 14px work-area density on 2026-11-01 in the Claude CLI session. `packages/design-system/styles/console.css` now sets body and table size to 14px and the console consumes `--type-body-size`. The untracked `docs/design-system.md` override paragraph and logo-size row still state 16px and the 48px header logo; they must be updated when that guide enters the tracked tree. This approval covers type density only; it is not screen FIX.

## Verification

Root executed in worktree `/private/tmp/hb67-react-console` on macOS arm64, Node 26.7.0, Playwright 1.56.1 with Chromium headless shell 1243:

- `npm run build`: PASS (TypeScript strict and Vite bundle).
- `npm test`: PASS, 24 tests.
- `npm run qa`: PASS, 37 checks, after updating the header-control height assertion from 44px to 36px for fine pointers.
- `npm run qa:work`: PASS, 32 checks.
- `npm run qa:entry`: PASS, 50 checks, including logo load, full-width entry main, neutral selected-workspace border and language listbox bounds.
- `node tools/verify-build.mjs`: PASS, 242 final artifact hashes including HTML and licenses.
- Rendered KO/EN/AR at 1440 and 390 pixels inspected for operations, overview, customer detail, issuance review, start, login, scope and bootstrap.

Independent read-only review (separate Claude subagent, no edits): initial verdict FAIL-fixable on constraints and PASS WITH FIXES on professional quality. Blocking items were the undocumented 14px change (resolved by owner approval), a mobile logo mark below 20px (fixed at 28px height) and fixed 32px targets on coarse pointers (fixed). Re-review: logo PASS, touch targets PASS with a record-card checkbox gap (fixed), no new blocking issue. A facts-column ID wrap found in re-review was fixed by stacking label and value; confirmed on the regenerated customer-detail capture by root, not re-reviewed.

Not verified: coarse-pointer devices, Windows/Linux, other browsers, screen readers, native-language terminology and owner acceptance.
