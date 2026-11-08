---
type: Design
title: Onboarding setup journey React implementation
description: Plans the React onboarding module for the eight setup routes, preserving the HB-63 wizard behavior under the revised console visual system.
---

# Scope and inputs

The owner directed HB-67 to continue with the remaining screens on 2026-11-01. Following the paired order in [design.md](design.md), this package implements the existing onboarding journey: `setup-progress`, `setup-profile`, `setup-attribution`, `setup-connectors`, `setup-readtest`, `setup-schedule`, `setup-first` and `setup-readiness`. Root (Claude Code, machine mac_mini) owns design and implementation; a separate read-only reviewer checks the plan and the change.

Behavior source: HB-63 commit 4d51763458b314fd766998c05fb32a1c40c99e72, `docs/design-modules/00-scalable-console/r1/onboarding.js`, `onboarding-copy.js` and `onboarding-profiles.js`. Content contract: `docs/design-modules/03-onboarding/r2/setup-workflow.md`. Visual system: the professional console revision in [claude-ui-design.md](claude-ui-design.md#professional-console-visual-revision). Route IDs, titles, group, related links and registry IDs come from the existing catalog and remain unchanged.

# Module boundary

A new `onboarding` module owns the eight routes, matching the HB-66 ownership map. It requires no other module; `setup-readiness` links to `operations` by route only. Both installation profiles register it. Copy and provider profiles are ported verbatim into module-local data. No backend, provider API, secret, file upload, invitation or message is used.

Draft state lives in a pure, DOM-free module store with an injectable timer. Drafts are keyed by the normalized account scope and data mode: scope uses exactly the shell rule (`sample-b` when the query says so, otherwise `sample-a`), so the displayed account and the draft key always match, and synthetic and unconnected drafts never share results. Drafts survive step navigation within the tab and are cleared by reload. The store exposes subscribe/snapshot for React. A run token rejects late results from an invalidated revision. Porting the baseline `disposeOnboarding`, the screen's unmount (route, locale, scope, mode or profile change) bumps every draft's run token and returns any `running` diagnostic or receipt to `idle`, so no result arrives after leaving and no step stays stuck.

# Behavior preserved from the baseline

- Step gating: steps 1–2 are always reachable; each later step requires the previous route to be complete. A locked step shows the prerequisite state with a resume action.
- Any form input clears completion from the current step onward, increments the revision, clears plan/admin/plan acknowledgment for steps up to connectors, diagnostics up to read test and receipt up to first validation.
- Connectors: provider choice resets provider-specific fields, account, identity, route and preview/IBM route. AWS account must be 12 digits. AWS new configuration skips provider fields. Blocked profiles: unverified providers, Azure outside EA/MCA, GCP without preview confirmation or outside US/EU, Alibaba without preview confirmation, IBM non-CSV. Missing identity shows the administrator handoff draft. The plan requires acknowledgment; a new configuration additionally requires simulated administrator read-back.
- Connectors form: pressing Enter triggers plan review. Plan review with a blocked profile shows the unverified message and no plan. Toggling plan acknowledgment invalidates from step 3 and then restores the reviewed plan with the new acknowledgment value.
- Read test: three separate checks (identity/reachability, metadata/schema, bounded read). It runs only when configuration is ready and the plan is reviewed and acknowledged. Scenario normal or partial passes all three; denied fails only the bounded read; outcome-unknown reports unknown. Changing the scenario invalidates from step 4. 450 ms simulated run. Continue only after pass.
- Start/resume on setup-progress goes to the first incomplete step after step 1, otherwise readiness.
- Schedule: start month and mode; IBM and file allow one-time only. History boundary notice.
- First receipt: runs only after a diagnostic pass. Completeness, schema and independent-evidence checks; the partial scenario passes diagnostics but holds the receipt; evidence always stays blocked.
- Readiness: technical, coverage and financial eligibility shown separately; financial is always blocked; publication disabled; return to billing workspace. Collection mode is shown with its localized label rather than the baseline's raw value.

Explicit unconnected mode uses its own draft. Forms and the synthetic example remain available because they are local input only. Diagnostic and receipt simulations are disabled with a not-connected state, so steps after the read test cannot complete and no simulated result appears in unconnected mode. This mode handling is a React addition; the baseline had no mode concept.

# Visual composition

## Process and evidence

Claude Code is root for this package. Following service-screen-style.md and the precedent in [claude-ui-design.md](claude-ui-design.md), a separate restricted Claude CLI ui-designer invocation reviews the actual renders before the screens are presented as visually compliant; its result is recorded in the verification section. External evidence fetched in this session:

| Source | Observed | Adopted / rejected |
| --- | --- | --- |
| IBM Carbon progress indicator usage (https://carbondesignsystem.com/components/progress-indicator/usage/) | Vertical arrangement preferred for reading; states completed, current, not started, error, disabled; disabled steps are not focusable and cannot convey information. | Adopt vertical list and the state set. Reject the colored active line; state is shown by text, weight and neutral fill. Locked steps keep a visible text status so meaning does not depend on the disabled control. |
| AWS Cloudscape multi-page create (https://cloudscape.design/patterns/resource-management/create/multi-page-create/) | Side step navigation; one page per step; standard per-field validation; alert with recovery for server errors; review page summarizes choices in order. | Adopt side step navigation, inline validation alert and the readiness review summary. The baseline's eight steps exceed its seven-step guidance; the route set is fixed by the existing IA and is not reduced. |

## Layout

Inside the existing shell (breadcrumb, title, metadata, review strip), each setup screen uses a two-column wizard: a 248px vertical step list at inline-start and the step content at the remaining width. At 1440px with the 246px shell rail this leaves about 840px for content; below 1280px the step list becomes a horizontal scrolling row so the form keeps its width, including in RTL. The step list shows number, name and status (complete, current, locked) using weight and neutral fill only; locked steps are disabled buttons whose visible text includes the locked status, and the step list item itself carries that status for assistive technology. A one-line neutral draft notice sits above the content. Forms use the two-column form grid capped at 960px as a field-group width aligned to inline-start, not a centered page container, required markers, inline validation alert and an action row: synthetic example (secondary) and save-and-continue (primary). Check results use the bordered workflow list with status chips. Readiness uses the facts list. The diagnostic scenario selector moves to the collapsed review-scenario disclosure used by work queues, because it is review tooling rather than product input. Scenario buttons follow the existing `action:scenario-<name>` convention.

Neutral-border mapping: the step list uses neutral separators and a neutral fill plus weight for the current step; status chips use semantic background tokens without borders or edge strips; draft, profile-limit, history and financial notices use the neutral notice surface; the scenario disclosure is unbordered text. Rendered QA asserts computed borders on these elements.

# Stable implementation item IDs

Shared on all eight screens: `region:layout`, `nav:setup-steps`, `notice:draft-boundary`; locked steps add `state:prerequisite` and `action:resume`.

| Screen | Items |
| --- | --- |
| setup-progress | `section:readiness-summary`, `notice:history-boundary`, `action:start-or-resume` |
| setup-profile | `field:legal-entity`, `field:market`, `field:currency`, `field:timezone`, `field:tax-policy`, `action:fill-example`, `action:save-continue`, `state:validation-error` |
| setup-attribution | `field:customer-reference`, `field:custodian-reference`, `field:effective-from`, `field:scope-ack`, `action:fill-example`, `action:save-continue`, `state:validation-error` |
| setup-connectors | `field:provider`, `field:route`, `field:billing-scope-id`, `field:workload-identity`, per provider field `field:provider-setting:<key>` (for example `field:provider-setting:location`), conditional `field:azure-agreement`, `field:preview-eligibility`, `field:ibm-route`; `notice:profile-limit`, `state:profile-blocked`, `section:handoff`, `action:review-plan`, `section:resource-plan`, `field:plan-ack`, `state:admin-readback`, `action:admin-readback`, `action:save-continue`, `action:fill-example`, `state:validation-error` |
| setup-readtest | `data:connection-ref`, `section:diagnostic-checks`, `action:run-diagnostic`, `state:idle`, `state:running`, `state:diagnostic-pass`, `state:read-denied`, `state:outcome-unknown`, `state:not-connected`, `action:save-continue`, `action:scenario-normal`, `action:scenario-denied`, `action:scenario-partial`, `action:scenario-outcome` |
| setup-schedule | `field:start-month`, `field:collection-mode`, `notice:history-boundary`, `action:fill-example`, `action:save-continue`, `state:validation-error` |
| setup-first | `data:receipt-ref`, `section:receipt-checks`, `action:validate-receipt`, `state:running`, `state:coverage-partial`, `state:receipt-pass`, `state:evidence-blocked`, `state:not-connected`, `action:save-continue` |
| setup-readiness | `section:readiness-facts`, `notice:financial-evidence`, `action:publish-unavailable`, `action:return-workspace` |

Items are reported when present on the screen, including review tooling inside collapsed disclosures, following the work-queue convention. Fingerprints include scope, mode (through the shell review context), step, completion set, revision, plan/admin/diagnostic/receipt state and scenario, so any change returns the screen to review.

# Validation plan

Unit tests use the pure store with a fake timer and cover gating, invalidation (input, scenario, plan acknowledgment), blocked-profile rules, AWS account format, resume target, stale-run rejection, dispose cancelling a running check, and isolation between sample-a/sample-b and synthetic/unconnected drafts. The assemble test checks that both profiles register onboarding. A new `qa:setup` browser suite walks the full synthetic journey to readiness, checks a locked deep link, provider reset, blocked profile, denied and outcome-unknown diagnostics, partial receipt hold, invalidation after an upstream edit, unconnected-mode disabling and the absence of synthetic results after a mode switch, leaving mid-run without a stuck state, reported item IDs per state, readiness under `profile=evidence`, computed neutral borders on the step list, chips and notices, keyboard access to the scenario disclosure, keyboard focus on step changes, and KO/EN/AR at 1440/390 with direction and overflow checks. Existing build, unit, qa, qa:work, qa:entry and verify-build must still pass.

# Limits

Synthetic local drafts only. No provider deployment, credential registry, invitation, collection, persistence, financial approval or owner FIX. Real-account profile verification remains NOT_RUN as stated in the workflow contract.

# Verification

Root executed in worktree `/private/tmp/hb67-react-console` on macOS arm64, Node 26.7.0, Playwright 1.56.1 with Chromium headless shell 1243:

- `npm run build` and `tsc --noEmit`: PASS. `node tools/verify-build.mjs`: PASS, 242 artifact hashes.
- `npm test`: PASS, 40 tests, including 16 onboarding store tests (adding provider-specific cadence validation and dependent-result clearing on input) (scope/mode isolation, gating, resume target, invalidation, AWS format, provider reset, six blocked-profile cases, administrator read-back, acknowledgment toggling, scenario and stale-run handling, partial receipt hold, receipt precondition, dispose and profile registration).
- `npm run qa:setup` (new): PASS, 21 checks. Locked deep link with five named locked steps and exact item IDs; full synthetic journey to readiness with financial eligibility blocked; denied and outcome-unknown diagnostics; leaving mid-run with no pending, passed or non-idle result; focus returning to the run action after completion; receipt pass with evidence still blocked; unconnected mode with no synthetic results and disabled checks; upstream invalidation; separate scope drafts (sample-b locked while sample-a connectors remain reachable); provider reset under `profile=evidence` and `scope=sample-b`; unverified profile blocked at plan review; neutral wizard borders; KO/EN/AR at 1440/390 with direction, overflow, LTR identifier inputs, a single primary action and the current step visible in the step row.
- Existing suites unchanged and passing: `qa` 37, `qa:work` 32, `qa:entry` 50.

## Design-lead render review

A separate restricted Claude CLI ui-designer process (Claude Code 2.1.316, agent `.claude/agents/ui-designer.md`, tools Read/Glob/Grep only, empty strict MCP configuration, no session persistence) reviewed the actual setup renders, specification and source with the official frontend-design skill (SHA-256 `d91970639e9f5c37682ac7ab60094d35f1c7c1f38d731bd56396563aee10c1d3`). Result: success, 19 turns. It opened 9 of 18 captures and did not run the app.

Verdict: visual PASS WITH FIXES; no constraint violation. Root applied every deviation: LTR direction on identifier inputs (Arabic showed "/focus" and a clipped ARN); centring the current step in the mobile step row; acknowledgment checkboxes at body size and ink color; exactly one primary action per state (plan review, then continue; run, then continue); full-width stacked actions below 600px; scroll reset before full-page captures. Composition items applied: checks capped at the 960px field-group width with a label/result grid, 16px panel heading, compact draft notice, unfilled plan box and scroll padding. The QA suite asserts the direction, single-primary and visible-current-step fixes. The designer has not re-compared the corrected renders; root inspected KO desktop, EN desktop and AR mobile captures.

## Independent change review

A separate read-only reviewer reviewed the implementation against the plan and baseline: PASS WITH FIXES, no blocking correctness, tenant-boundary or honesty defect. Root fixed focus loss when run/read-back buttons disable themselves (focus returns to the run action or to continue), strengthened the mid-run and scope-isolation browser checks, required cadence to match the provider's allowed modes and removed a duplicate required announcement. Remaining non-blocking items: raw cadence and English "revision" in the receipt/connection reference lines (inherited), unlinked field errors, partial-receipt and chip-border checks in the browser suite, and receipt stale-run unit coverage.

Not verified: coarse-pointer devices, screen readers beyond accessible-name checks, Windows/Linux, other browsers, native-language terminology, real provider profiles and owner acceptance. No owner FIX is created.
