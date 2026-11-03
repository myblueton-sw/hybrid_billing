---
type: Review
title: Planning-to-screen review and source workflow corrections
description: Records independent planning parity findings, implemented reception and routing fixes, and remaining service integration gaps.
---

# Scope and ownership

The owner requested continued implementation with reviewers comparing planning, designs and actual screens. Root owns PM/planning, implementation and integrated QA. Separate read-only reviewers assessed S03/S06 service boundaries and whole-console/source-journey planning parity. Claude CLI independently inspected actual reception-review renders. HB-142 starts from main 36394ed and explicitly incorporates pending HB95 d05763f, whose dependencies are HB141 and HB74. No main merge, production operation or owner FIX is authorized by this review.

The canonical route registry contains 49 distinct screen IDs with planning-aligned menu families. This is inventory parity only. Synthetic candidates, local-service modes and explanatory panels are not completed production services. The reviewed planning evidence is the authorized local S01–S06 and U01/U02 screen specifications and planning action contracts; a fresh clone may not contain every local planning input.

# Findings and disposition

| ID | Finding | Disposition |
| --- | --- | --- |
| HB142-F01 | S03 detail failure retained account/workspace and file intent after scope denial. | Fixed: all protected state and native file input clear; fresh authorization is required. |
| HB142-F02 | S03/S06 live object routes ignored canonical object IDs and displayed generic lists. | Fixed: exact scoped detail endpoints; unknown/foreign/mismatched results fail closed. No automatic revalidation. |
| HB142-F03 | S03 directly persisted uploads without reviewing target/period/bytes/revision/duplicate policy. | Fixed: explicit pinned reception review, cancellation, fresh profile comparison and one POST intent. |
| HB142-F04 | Version/revision changes could bypass dirty navigation protection after upload. | Fixed: successful reception removes the file; new file-bound edits invalidate review and mark dirty. |
| HB142-F05 | U01/U02 used private period/currency query keys that the shell dropped during navigation. | Fixed for supported period/currency continuity: canonical period/ccy take precedence; owning-screen aliases migrate on write/navigation. |
| HB142-F06 | S04→S05 shows reference strings without real evidence traversal/preview. | Open: next read-only integration slice after typed backend contract and authority review. |
| HB142-F07 | S01/S02 candidate steps lack configuration-revision-bound readiness/handoff. | Open: requires shared staged configuration and real diagnostic/readiness evidence. |
| HB142-F08 | Generic candidate tab remounts discard local filter/selection state. | Open: define retained per-screen/snapshot state without preserving stale authorization. |
| HB142-F09 | U02 exposes only list/calendar; some planned recognition/detail/history workflows are summaries or unavailable. | Open: exact cycle-bound routes and evidence navigation remain necessary. |

Source/organization subset projection and snapshot continuity across overview/closing remain open. The period/currency fix does not invent matching organization models, arbitrary currency support or currency-based closing calculations. Canonical input confirmation/publication/rejection, independent billing validation, completeness evidence, OCR, durable diagnostic jobs and additional S06 reconciliation modes remain deferred backend work rather than newly fixed regressions.

# Implemented behavior

The S03 reception dialog pins one immutable File reference, version, positive signed-64-bit revision and explicit server profile fields. Confirmation re-reads the profile and compares profile type, workspace, connection, account, period, supported versions and byte limit. Changed context rejects reception before POST and clears protected state. Existing server validation remains authoritative; this frontend comparison is not authentication or billing confirmation.

Cancellation/Escape performs no POST and returns focus to the trigger after closing the native dialog. Once POST starts, closing/navigation cannot promise cancellation of server processing; the dialog explains unknown-outcome inspection before retry. A successful POST clears both React file state and the native file input. A subsequent receipt refresh cannot automatically resubmit. Stable review-item IDs include `S03:service:review-reception` and `S03:service:exact-detail`. Exact S06 details remain within its existing source/evidence review unit.

Object routes query only the requested object after the current profile check. They never fall back to the first candidate. Detail pages include a locale/service-preserving return link and omit generic list/upload sections. S06 exact details do not automatically run a diagnostic. Invalid scope responses remove protected information.

`canonicalScope` migrates `overview-period` and `ov-currency` only for U01, and `close-period` only for U02. Presence of canonical `period` or `ccy` wins, including empty/invalid values; aliases are not used to repair invalid canonical input. Controls write canonical keys and remove owning-screen aliases. Other route context is not interpreted as new authority.

# Design and review

The reception dialog preserves neutral borders, square controls, a bounded readable column and local fonts. It has a named native modal, clear cancellation and a primary reception action, with directionally isolated identifiers/timestamps. [Cloudscape focus guidance](https://cloudscape.design/foundation/core-principles/accessibility/focus-management-principles/) and [USWDS modal guidance](https://designsystem.digital.gov/components/modal/) informed focus/task structure, not product theme adoption. Existing owner review stays on S03 and covers this dialog as an item; no automated owner acceptance occurred.

Independent design review PASS required snapshot use, explicit profile comparison, epoch/abort guards, no automatic retry, exact object reads and focus recovery. Actual source review PASS for the implemented scope. A minor misleading ready-helper was corrected to use the same input-envelope validation. Separate period/currency design/source review PASS; that reviewer independently executed two scope-helper tests. Reviewers did not claim executing the browser suite.

Actual Claude CLI visual review PASS for KO1440 and EN/AR390 reception dialogs. It verified required fields, readable layout, neutral geometry and directionality. Optional period-label, unit-formatting and explanatory-hierarchy refinements remain; native-language, full screen-reader and owner acceptance are not certified.

# Verification

- Frontend suite: 62 PASS; TypeScript and build PASS. Existing large-bundle warning remains.
- New browser workflow suite: six modal locale/width renders, zero POSTs for all cancellation/changed-scope checks, exact and unknown S03/S06 details, denial clearing, dirty retain/discard, reception review revision binding, nondefault month/KRW roundtrip, canonical conflict precedence and Back/Forward PASS.
- Actual synthetic ingestion regression: upload after explicit reception confirmation, stored reload, identical replay and changed-payload conflict PASS; six S03 locale/viewport renders PASS.
- The first focus test failed because the trigger was focused while the native modal still made the background inert. Closing the dialog before restoring focus fixed it; the repeated browser assertion passed.
- An initial QA command used a duplicated relative path and did not run. A later same-URL test retained the existing error state because no navigation/remount occurred; the runner now explicitly reloads the same URL. Neither failed attempt is counted as a pass.

Run `npm --prefix apps/console test` and `npm --prefix apps/console run build`. Use the existing isolated local-service setup from [HB141](../HB-141/integration.md). Start separate fixed-role processes:

```sh
python tools/console_service.py --local-operator --actor loader-a --port 8795
python tools/console_service.py --local-operator --actor approver-a --port 8796
```

From `apps/console`, run:

```sh
node tools/review-flow-qa.mjs --playwright-module <module> --chromium <executable>
node tools/service-qa.mjs --url http://127.0.0.1:8795/ --playwright-module <module> --chromium <executable>
```

The review-flow runner is read-only, including intercepted failures; the service runner writes only synthetic receipts in the explicitly isolated profile. Generated evidence is under `output/validation/HB-142` and `apps/console/qa-output/service`, not Git deliverables. Stop/restart servers after frontend builds because their asset allowlist is loaded at startup. Tested host: macOS arm64, Node26.7.0, Python3.11.7 and Chromium headless shell1243. Windows/Linux and other browser execution remain unverified.

Final targeted source re-review PASS for native-dialog close-before-focus, input-envelope readiness guidance and exact inspect-result matching. S06 regression also passed 11 scenarios and six locale/viewport renders on the updated build. Final 242 artifact hashes and scoped CSS/OKF checks passed. These passes do not close the open planning gaps above.
