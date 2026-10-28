---
type: VerificationReport
title: HB-36 licensing transition verification
description: Records provenance, scope preservation and document checks for adopting standard noncommercial terms without revoking historical grants.
status: draft
---

# Scope

[HB-36](https://linear.app/hybrid-billing/issue/HB-36/adopt-source-available-noncommercial-terms-with-separate-commercial); owner Seung Woo Park; machine mac_mini. Isolated branch `docs/HB-36-noncommercial-license` starts at main `4f990f168790ad15b10bf6359927fadeefb8e6ef`. The writer owns licensing texts and direct current summaries only. Root independently reviewed the applicability design before edits and reviews the resulting diff before commit.

The change introduces the [decision](../../licensing/source-available-adoption.md), exact standard LICENSE, preserved MIT license, copyright notice, current policy/scope/contribution text and synchronized README/notice/planning summaries. No runtime, UI, telemetry, remote lock, customer data, signed agreement or payment code is changed.

# Source provenance

Official text: `https://polyformproject.org/licenses/noncommercial/1.0.0.txt`, reached from the official license page's download link. Retrieved on 2026-10-28 with curl. Root LICENSE is an unmodified byte copy of that text. SHA-256: `ffcca38841adb694b6f380647e15f17c446a4d1656fed51a1e2041d064c94cc8`.

Historical MIT text: exact LICENSE bytes at the base commit, now `LICENSES/MIT-legacy.txt`. SHA-256: `83a62b0955e876667b851d0de97f4d88f3eb2dc1edaf22e2afe3925b956ded9c`. This retains the prior collective copyright and permission notice; it is not a newly narrowed grant.

# Verification

Completed on 2026-10-28:

- PASS: root LICENSE is byte-identical to the official downloaded text; SHA-256 matches the value above.
- PASS: `LICENSES/MIT-legacy.txt` is byte-identical to `git show HEAD:LICENSE` at the recorded base revision; SHA-256 matches the value above.
- PASS: all local Markdown links in the 14 changed files resolve, and each changed Markdown document under `docs/` has the required nonempty `type` frontmatter.
- PASS: current-reference search found no stale current MIT grant or future-additions-only scope in the changed current documents. Historical MIT statements remain explicitly marked historical. No package license field required synchronization.
- PASS: `git diff --check`; changed-file scope is limited to the 14 licensing and direct-summary paths below. No runtime files changed.
- PASS: root's independent review of the actual license, scope, notice, contribution terms, current policy, README, third-party boundaries, historical decision, new adoption decision and planning summary. This is a scoped artifact review, not legal certification or an ownership audit.

Changed files: `CONTRIBUTING.md`, `LICENSE`, `LICENSES/MIT-legacy.txt`, `LICENSING.md`, `NOTICE`, `README.md`, `THIRD_PARTY_NOTICES.md`, `docs/licensing/commercial-evaluation.md`, `docs/licensing/mit-adoption.md`, `docs/licensing/product-license-draft.md`, `docs/licensing/product-policy.md`, `docs/licensing/source-available-adoption.md`, `docs/planning/v1.16/planning-decisions.md` and this report.

Commit, PR, merge and post-merge read-back remain the integrator's responsibility and are not claimed by this report.

# Limits

The owner reports sole development and no external distribution/downloads, authorizing transition of current project-authored original material. This statement has not been independently audited. Any rights already granted for an earlier MIT copy and all third-party rights remain intact; the historical MIT file is not an alternative license for the current transition revision. Contribution policy does not invent commercial sublicensing authority. A complete ownership audit, jurisdiction-specific enforceability, signed contributor permissions and actual commercial terms remain outside this documentation change.
