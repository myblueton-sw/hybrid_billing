---
type: Verification
title: HYB-326 OKF status values and resolved residuals
description: Every tracked Markdown front matter now uses the OKF status values draft, stable or deprecated, a unit test rejects any other value, and residual lines later resolved by other tickets carry dated pointers.
status: stable
tags: [HYB-326, OKF, documentation]
sources:
  - resource: ../../../.agents/rules/knowledge-format.md
  - resource: ../HYB-295/verification.md
---

# HYB-326 verification (root, 2026-11-18, main 12a4163)

## Rule

`.agents/rules/knowledge-format.md` line 8: `status` uses draft, stable or deprecated.

## Change

- Front matter only (between the opening `---` and the next `---`): 117 files `status: final` and 7 files `status: reviewed` (HB-160–HB-164, HB-167, HB-168 plans) set to `status: stable`. Body text was not touched.
- `tests/docs_format/test_okf_status.py`: scans every tracked `*.md` front matter and fails on any other status value; a second test pins that only front matter is read.
- Dated pointers appended to residual lines resolved later, original text unchanged:
  - `HB-260/plan.md:142`, `HB-260/verification.md:93`: administrator-issued resets built by HB-263/HB-264 (PRs #183/#184; `identity.issue_reset_link`, `apps/api/server/admin.go:484`).
  - `HB-264/verification.md:168-169`: bulk revocation and role validity periods built by HYB-299 (PR #218).
  - `HB-265/verification.md:188`: trust workflow built by HYB-297 (PR #215).
  - `HB-266/verification.md:126`, `HB-267/verification.md:61`: grant approval request built by HYB-298 (PR #212).
  - After the review (round 1 FAIL: the same resolved residuals were left stale in sibling records), also: `HB-263/verification.md:202-203, 215-216`; `HB-264/verification.md:181`; `HB-266/verification.md:129, 133, 137-138`; `HB-267/verification.md:64, 68, 72`; after round 2 also `HB-263/plan.md:224`, `HB-259/plan.md:172`, `HB-259/verification.md:90`. Caveats added where the later record carries one: role validity covered by interpretation (HYB-299 Q2); trust activation on a non-sample installation blocked until enrolment is decided (HYB-297 R4/Q3); federated checks still open.

## Checks

- `python3 -m unittest discover -s tests/docs_format -v`: 2 tests OK.
- The same test against the unchanged documents (stash): FAILED, listing every `final` and `reviewed` file — the check is shown to fail before the fix.
- `git diff --stat`: 124 status lines plus 22 annotated residual lines; no other line changed.
- Independent review: round 1 FAIL (incomplete annotation, caveats) and round 2 FAIL (three more stale plan/verification residual lines); fixed as above. Screen-design descriptions, copy intent, scope statements and historical review reports were deliberately not annotated. The test reads every tracked `.md`, which is wider than the rule's `docs/` scope; tool files without front matter are skipped, and the wider scope is kept on purpose.

## Residuals

- The repository has no CI workflow in the tracked tree, so the test runs only when invoked; it is not wired into a pipeline.
