---
type: ImplementationDesign
title: HB-91 local onboarding evidence navigator
description: Connects the five canonical onboarding stages to actual local metadata without implying persisted handoff or activation approval.
status: draft
---

# Scope and ownership

Root owns PM, implementation and QA. A separate authority reviewer conditionally passed this design. HB-90 PR74 is the pending dependency; the branch starts from current origin/main and incorporates that dependency explicitly. This increment owns the onboarding module page, its tests and minimal App route wiring. No shared frame redesign or backend mutation is included.

# Behavior and boundaries

The five stages are path, permissions, first data, first reconciliation and activation. Selected or visited stages never imply completed stages. The existing validated metadata endpoint reauthorizes every refresh; old content and destination links clear on failure. Exact connection identity is preserved, and destinations independently reauthorize.

The configured manual synthetic CSV path is read-only. Loader/approver access is not evidence of setup-owner, export-writer or collector-reader assignment. These roles remain separate unknowns. A registered candidate is not evidence of first-ever receipt, completeness or independently verified billing. Quarantine and unresolved operations remain separate holds. Reconciliation has not been checked by this endpoint; S06 performs source/storage diagnostics only. Activation, ERP/tax, cutover, recovery and business gates remain unverified.

Handoff and activation controls are unavailable with explicit missing-contract reasons. No owners, expiry defaults, costs, metric denominators, percentage completion or authorization grants are invented. No owner FIX is created. Production operations and Windows/Linux execution remain unverified.

# Composition evidence

Official [Cloudscape Wizard](https://cloudscape.design/components/wizard/) and [multipage create](https://cloudscape.design/patterns/resource-management/create/multi-page-create/) were consulted on 2026-11-03. Adopt a separate step navigation, content heading, focused stage body and navigation actions. Reject automatic completion semantics, copied theme and persistent colored borders. Keep full viewport width, canonical shell and neutral separators. Mobile collapses to one column; stage selection is explicit and keyboard reachable. Claude CLI composition and final screenshot review are recorded below when actually returned.

# Implementation items

| Item | Behavior |
| --- | --- |
| S02:scope:connection | Fixed server connection, policy revision and observation |
| S02:navigation:stage | Five navigable stages; URL/history preserve selection, never completion |
| S02:data:path | Configured synthetic manual input and installed contracts |
| S02:data:permissions | Separate unknown handoff roles and current metadata route access |
| S02:data:first-input | Latest exact candidate and separate hold reasons |
| S02:data:reconciliation | No inferred reconciliation; current-role exact S06 route |
| S02:state:activation | Explicit unverified gates and unavailable command |
| S02:action:refresh | Abort stale request; clear data and links on denial or invalid response |

# Validation plan

Unit checks cover stage resolution and role-specific destinations. Browser checks cover all five stages, history, exact navigation, refresh, denial/malformed/stale responses, EN/KO/AR desktop/mobile, neutral borders and keyboard focus. Reuse existing metadata security tests; no new backend authority is added. Independent implementation review and Claude CLI render comparison are required before handoff.

# Running the increment

Build with the existing console command, then run the local service with the isolated synthetic configuration and an explicit loader or approver process. Open /#/onboarding?lang=ko&service=local; stage selection uses view=path|permissions|first-data|reconciliation|activation. Optional connection is an exact UUID; empty or malformed explicit context is rejected before fetching.

Run npm test and npm run build in apps/console. The browser runner is tools/onboarding-qa.mjs with --url, --loader-url, --playwright-module and --chromium arguments. Generated captures and reports belong under output/validation/HB-91; they are not Git deliverables.

# Review and verification

The authority design review conditionally passed the read-only scope. Implementation review passed its permission and readiness semantics, with a request-ID display correction applied. Independent frontend review found that explicit empty connection context could fall back to default scope. The new context guard rejects empty/malformed identifiers before fetching. Targeted re-review PASS; three model tests independently executed.

Claude CLI returned a composition proposal using the existing shell. Its installed frontend-design skill was unavailable in the worktree, so no skill-assisted review is claimed. Root independently consulted the official references above. The implementation uses stage-specific facts instead of repeated six-field templates, plain mobile navigation instead of a disclosure, natural-width text navigation and no decorative glyphs or sticky panels. Single-connection filters and unsupported command inputs are intentionally absent.

Initial visual review failed a stretched preview-mode control, monospace status text and unstable next-link alignment. Scoped CSS removes the inherited wizard grid from S02, statuses retain the normal 16px locale font, and next links stay at inline-end. The Arabic counter is directionally isolated. These changes do not alter other screens.

Actual checks on macOS arm64, Node 26.7.0 and Chromium headless shell 1243:

- 72 console tests PASS; typecheck/build PASS with the existing bundle-size warning.
- 30 rendered combinations PASS: five stages in EN/KO/AR at 1440 and 390 px, including no overflow, neutral borders, loaded logo and 16px body.
- Browser history, keyboard refresh, exact S03/S04/S06 navigation, role-specific destination absence, denial/malformed clearing and retry, empty candidate, foreign/empty/invalid identity and late-response discard PASS.
- Arabic readiness text measured at 16px with IBM Plex Sans Arabic, not the monospace code style.
- Final Claude CLI visual re-review PASS on five representative updated captures; no remaining blockers. Independent source re-review of scoped layout changes also PASS. Screenshot review does not certify focus, native-language or screen-reader acceptance.

No backend code or authority changed; prior metadata endpoint verification is dependency evidence rather than a newly executed backend suite. Production roles, full onboarding mutations, native-language/screen-reader acceptance, scale, Windows/Linux operation and owner FIX remain unverified. No main merge or deployment is claimed.
