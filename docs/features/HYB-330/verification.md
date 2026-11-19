---
type: Verification
title: HYB-330 response diagnostics verification
description: Records implementation, independent review and synthetic browser evidence for P01 through P05 command diagnostics.
status: stable
sources:
  - resource: ./plan.md
  - resource: ./verification-evidence.json
---

# Result and scope

Ticket: https://linear.app/myblueton/issue/HYB-330. Base 653adfe; machine mac_mini. Root integrated P01/P03/P04/P05 changes from the isolated platform implementer and owns shared parsing and P02. Only frontend feedback and its evidence changed. Backend permissions, command outcome interpretation and retry keys remain unchanged.

AdminError preserves bounded response code and request_id. Retry-After retains both, step-up uses its final command response, and local cancellation/network failure invent neither. Boundary-generated IDs and command-result echoes are both supported by using response data. Retained notes select the same command/target as their message; permission-loss and terminal feedback retain their own response identity. New attempts clear earlier details and rerender sibling feedback. Monotonic scope tokens reject late results after session changes.

# Review

Independent design review: PASS with constraints. The first actual-change review requested repairs for parent terminal feedback, late-session results and missing next steps. Those repairs were implemented. Independent integrated actual-change review: PASS, no remaining blocker. The reviewer separately executed 184 diagnostic and affected-screen tests, all passing. Independent visual QA/TW spot review also passed after inspecting six representative screenshots across all five screens. It found no clipping, overlap or new layout blocker; it does not establish native-language approval. Root remains responsible for browser execution.

# Executed verification

| Check | Result |
| --- | --- |
| npm test, complete console suite | PASS, 870 tests, zero failures |
| npm run typecheck | PASS |
| npm run build | PASS; existing Vite configuration and large-bundle warnings remain |
| Independent shared diagnostics and five screen model suites | PASS, 184 tests |
| P02 EN/KO/AR at 1440 and 390 | PASS, 48 assertions; injected refusal after actual form submission, matching receipt, focus, LTR values, wrapping, neutral border and new-attempt network failure clearing |
| P01/P03/P04/P05 EN/KO/AR at 1440 and 390 | PASS, 72 assertions; injected retained refusal entries on actual seeded service pages, correct rendering and geometry/LTR checks |
| Delayed previous-session response | PASS; the actual loaded model generation changes before the response and receives no old-session receipt |
| Terminal permission denial | PASS; command identity remains visible rather than being replaced with the follow-up read identity |
| Documentation status tests | PASS, two tests |
| git diff --check and maintained source/test line limits | PASS |

All browser data is synthetic. P02 command errors were intercepted; other platform retained notes were injected through their actual models, so those renders do not claim every backend refusal was reproduced. Unit tests cover response parsing, throttling, step-up, malformed metadata, network failures, cleanup and grouped-command precedence. Thirty screenshots were captured. Root visually inspected representative desktop/mobile screenshots from every changed screen, including mixed Arabic/Latin identifiers. Screenshot hashes and runtime versions are in verification-evidence.json.

# Failed checks and repairs

The first P02 browser run exposed an old refusal remaining visible after a new attempt failed on the network. Store deletion alone did not rerender its sibling note; the onAttempt callback now rerenders the parent and clears previous notice details. The rerun passed.

A preliminary platform rendering harness imported a separate hot-reload model instance and timed out. A cold development-server start fixed the harness, and the delayed-session check explicitly rejects an uninitialized imported instance. Only the subsequent passing runs are counted. Early login setup used the wrong synthetic workspace/login and was corrected before counted scenarios.

# Limits and integration boundary

Tested on macOS arm64 with Node v25.9.0 and Chromium 153.0.8010.12. Linux/Windows execution, native-language approval and assistive-technology review were not performed. No owner FIX, real data, deployment, financial operation or production readiness is claimed. Temporary screenshots and browser scripts are not durable artifacts; their hashes and summarized results are retained here.

The full local P02 specification and some companion rules were absent; the owner explicitly authorized continuing without restoration. Scope follows the verified ticket requirement, existing source and available rules. PR merge and ticket completion remain separate, pending authorized integration; this record does not pre-certify either.
