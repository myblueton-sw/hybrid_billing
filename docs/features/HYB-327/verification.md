---
type: VerificationRecord
title: HYB-327 console response and binding verification
description: Records parser, screen, review and scope evidence for the field-map follow-up.
status: draft
---

# Result

Implementation and independent change review passed. This is a review handoff, not a merged or completed ticket. Accountable owner: Seung Woo Park. Execution machine: `machine:mac_mini`; release at PR handoff. Base: `653adfe`. HYB-330 remains an independent unmerged change and is not included.

# Changes and scope reconciliation

| Scope | Result |
| --- | --- |
| O01 | Parties, profiles and member timelines validate required rows and nested collections instead of unchecked casts. Root-party empty parent is supported. |
| M03 | Access rows, scopes, cleanup/history/receipts and preview decisions validate shape, exact booleans and actionable states. |
| M04 | Service/owner/credential rows and limits validate producer contracts. Optional pickup metadata remains optional; malformed supplied metadata fails the read. A contractual zero credential-day limit stays zero. |
| M05 | Identity/session arrays validate rows. Invalid reloads and polling clear old page data and stop presenting its actions. |
| M01 | Selection already connected by the filter-selection follow-up. API-key and pending-job impact say not applicable only for explicit server states; absent or unfamiliar states show unknown. |
| M02 | Restored/default comparison fields are displayed separately, with explicit empty/unknown distinction and LTR isolation for unknown codes. |
| P01 | Count-only history/backup summary retained; malformed collections are unknown. Current override producer always emits an empty array. Detailed history/backup/override editing is not claimed. |
| P02 | Clients section state gates all child data, even when contradictory rows accompany a denied/unknown/unconfigured state. |
| P05 | Server failure counter replaces fixed not-connected text. Adjacent text explains that budget approval is not connected and zero proves neither approval nor a passing run. |

Malformed reads use the existing `AdminError("invalid")` path. No backend API, billing calculation or permission authority changed. Existing composition and classes were reused; no CSS or visual-layout design was added.

# Verification

- PASS: full console `npm test`, 872 tests. After the final M02 separator correction, the focused `binding-gaps.test.tsx` suite passed all 13 tests.
- PASS: `npm run build` (TypeScript and Vite). Existing native-config compatibility and large-chunk warnings remain.
- PASS: independent focused `admin.test.ts`, `strict-read.test.ts`, `binding-gaps.test.tsx`: 58 tests. Parser implementer additionally ran the original 848-test suite before integration.
- PASS: actual seeded service routes M02/P02/P05 in EN/KO/AR at 1440/390 pixels. Verify named/unknown stricter fields, P02 contradictory section states, P05 zero/nonzero/unknown counters, wrapping and geometry. Final run also checks computed neutral border colors. Exact totals are in the evidence JSON.
- PASS: M01 real preview preparation with injected unknown/absent impact fields in all six language/width combinations: 12 assertions. Synthetic preparations were discarded; no revocation was executed.
- PASS: M03/M04/M05 valid read → malformed reload → valid retry: nine browser assertions. Polling cleanup was inspected independently in source; a timed polling browser scenario was not executed.
- PASS: independent design and final code/technical-writing review. Review found and resolved M05 polling cleanup, P05 counter qualification and M02 joined-label defects. Separate reviewer inspected 18 M02/P02/P05 renders, including recaptured M02 images after the separator fix, plus six M01 renders. M01 visual review covers the visible API-key row; pending-job values were verified by browser DOM assertions because they were below the captured viewport.
- PASS: `git diff --check`. Staged changes are inspected before commit.

Host: macOS; Node v25.9.0; Chromium 153.0.8010.12. Browser runner: the existing temporary Playwright installation and explicit Chromium binary, not a new product dependency. Synthetic API seed ran locally on port 8820; Vite on 8830. Screenshots/scripts/logs are local generated evidence under `/private/tmp/hyb327-qa` and `/private/tmp/hyb327-*.log`, outside Git. No customer data or credentials are committed.

Early test failures were a missing parser integration and an over-specific bdi assertion; both were corrected. Browser harness retries corrected seed activation, duplicated text selectors and a route-handler teardown race; those failed attempts are not counted as passes.

# Limits and handoff

- Missing original companion rules/specs remain under the owner's existing no-restoration instruction. No complete original P01 detail-scope acceptance is claimed.
- English/Arabic/Korean AI technical-writing and visual review is not native-language or assistive-technology acceptance. Windows/Linux operation is unverified.
- Existing owner screen acceptance is not carried forward or synthesized. No review-control acceptance was submitted.
- No deployment, real-account action or merge occurred. Remote merge/read-back and ticket Done remain pending authorized integration. Machine claim is released at this handoff.
