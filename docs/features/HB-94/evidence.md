---
type: ImplementationDesign
title: HB-94 safe source evidence inspection
description: Connects a normalized source row to an authorized non-executing original CSV extract and preserves its return context.
status: draft
---

# Scope and ownership

Root owns implementation and QA. Independent reviewers inspect authority/source disclosure and canonical planning/interaction boundaries. HB-93 PR72 is an explicit pending dependency, incorporated into this ticket branch without merging main. This is a fixed local synthetic approver workflow, not production BL/AU/CU authorization.

# Evidence contract

A read-only endpoint accepts only canonical release UUID, expected manifest hash and positive release-relative ordinal (1–10,000). Every request enters the existing inspection transaction, rechecks current capability before returning, and validates pinned registry, manifest hash, scope, source revision, policy and input readiness. No caller-supplied storage path is accepted.

The server derives the expected original key from its authorized scope and release, verifies stored references, reads at most the existing 8 MiB admission limit and verifies original bytes/hash. Parsing under the pinned registry/current capability must reproduce the manifest row count, normalized row digest and controls. Only the selected record's allowlisted source fields and recomputed normalized values are returned. Source strings retain their admitted lexical values, including null markers; normalized null is distinct from zero. Arbitrary JSON/tags/unclassified fields and storage paths are omitted.

The response is a safe text extract of received source input, not a downloadable original, published document, OCR result, immutable warehouse snapshot or independent billing validation. Recomputed normalization does not certify current warehouse contents. Failed or unavailable objects produce a generic evidence error without guessing whether the cause is deletion, archival or key loss. Retention and public-document state remain unknown/unconnected.

# Screen and navigation

Keep the canonical full-width shell and per-screen owner review. Show the selected input/row and verified-source status first; compare source text and recomputed normalized values in a bounded table/cards; make integrity hashes and connection limits available separately. Consulted [Cloudscape details-page guidance](https://cloudscape.design/patterns/resource-management/details/details-page/) for information hierarchy and contextual return links; no code or styling imported.

| Item ID | Behavior |
| --- | --- |
| S04:action:evidence | Open exact selected row evidence; no amount or storage key in URL |
| S05:scope:source-row | Release, manifest, ordinal and server authority bind the extract |
| S05:preview:safe-text | Plain text source/recomputed fields only; no HTML, formula or URL execution |
| S05:integrity:source | Verify bytes/hash and pinned parse; distinguish from billing approval |
| S05:state:unavailable | Clear prior protected fields on denial, stale or failed request |
| S05:action:return | Restore S04 page/snapshot/row and focus; reauthorize on return |
| S05:state:retention | Unconnected retention, restore, export and public documents stated explicitly |

The route carries release, manifest, row and validated page cursor. No arbitrary return URL is accepted. S05 opened without a selected row provides a working S04 entry link. Back/Forward and Escape preserve the selection and do not interfere with a modal. React text nodes render all source content; no iframe, HTML renderer, URL autolinking or spreadsheet engine is used.

# Verification and remaining work

Planned checks: real isolated S3 source read, tampered bytes/manifest/source key/row rejection, cross-scope and revoked access, bounded non-executing payloads, exact decimal strings, no writes or original path exposure, S04→S05→S04 focus, stale/error/retry/empty contexts, late response discard, multilingual desktop/mobile renders and independent change/visual review. No tests or review verdicts are claimed by this design.

PDF/XLSX, OCR, public customer views, original export, archival/restore, production permission contracts and large-volume evidence jobs remain outside this bounded local slice. No owner FIX, main merge or cross-OS execution is implied.

# Actual implementation and verification

Implemented the scoped row-extract API and S04 evidence link. Refactored the existing S04 release checks into `inspected_release()` without removing checks. The response includes source-column presence separately from original text and normalized null. Original record ordinals use the same CSV parsing semantics as normalization.

Independent design review: conditional PASS by planning_boundary_review and gap_review. Backend source review: PASS. Frontend review found that S04 ignored `focus=evidence` on return; the focus branch was corrected and independently re-reviewed PASS. These verdicts are scoped code/design reviews, not owner acceptance.

Actual verification on macOS arm64, Python 3.11.7, Node 26.7.0 and Chromium headless shell 1243:

- Evidence backend tests: 7 PASS, including actual isolated S3 reads, read-only persistence counts, cross-scope/loader denial, stale/invalid ordinals, original-key/hash/manifest failures, revoked authority, HTTP restrictions, quoted multiline records, absent/empty/null markers, inert strings and excluded fields. Adversarial original fixtures are patched in memory; production objects are not modified.
- Existing S04 backend regression: 8 PASS after extracting shared checks.
- Console tests: 66 PASS, including evidence identity/page bounds and malformed-response validation.
- Typecheck/build: PASS after correcting a test-fixture typing error. An initial browser run used the previous build and failed return focus; after a successful rebuild and server restart, the current artifact passed. The existing bundle-size warning remains.
- Build manifest: 242 artifact hashes verified. New CSS passed the scoped prohibited-style scan; canonical asset verification also runs in the console build.
- S05 browser QA: six EN/KO/AR desktop/mobile renders PASS. Verified actual source extract, explicit return, Back/Forward, Escape, topmost review-dialog cancellation without acceptance, detail close-to-row focus, inert HTML/formula text with zero external requests, denied/stale/malformed clearing, entry/malformed context and late-response cancellation.
- S04 browser regression: six EN/KO/AR desktop/mobile renders PASS after adding the evidence action.
- Claude CLI visual review: provisional PASS on desktop/mobile full-page renders; Arabic reduced-scale uncertainties were resolved with full-size element captures and a final scoped PASS. The record ordinal/total was isolated as one LTR group. This is not native-language or owner acceptance.

Generated reports and screenshots remain under `output/validation/HB-94` and are not staged. Production S05 parity, PDF/XLSX hostile-file execution tests, native-language acceptance, screen-reader coverage, large-volume performance and Windows/Linux execution remain unverified. No real owner FIX or main merge has occurred.

The S04 change adds the evidence action and return-focus behavior; its existing acceptance is not carried forward. S05 inherits the canonical review control with a new source revision. App routing and banners are scoped to the new live screen; no shared logo, palette or navigation IA is replaced.
