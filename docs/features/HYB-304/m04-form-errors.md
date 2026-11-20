---
type: ValidationRecord
title: HYB-304 M04 field errors and keyboard recovery
description: Records field-specific validation, stable picker focus, rejected-edit preservation and correction of an Arabic credential ratio reversal.
status: draft
sources:
  - resource: ./m04-consumer.md
  - resource: ./m04-command-errors.md
  - resource: ./form-error-evidence.json
---

# Reproductions and owned change

HYB-304 remains In Progress, machine mac_mini, following pushed checkpoint ed63b97a2465a7b33a34db691c4ed450aa7290a0. Root is sole writer. The actual mounted baseline failed because an empty purpose was not marked or associated with its error: aria-invalid was absent. Existing create/declaration checks produced only generic form text; issuance did not associate duration/rationale errors. Independent source reviewer /root/hyb310_review and terminology/render reviewer /root/hyb310_copy_render provide separate scoped reviews. No server command, permission, current limit or frozen retry policy changed through this UI correction.

Local first-submission checks now identify name, owner, purpose, owning organization, declared scopes, duration and rationale as applicable. The summary uses existing localized heading/labels and moves focus on each failed submission. Keyboard Enter on a summary link preserves the app route and focuses its actual control; declared scopes has its own focusable region. Inline errors and aria-describedby match the summary. Accepted editing clears only that field's error. Rejected over-limit edits keep the previous draft, announce that rejection separately and do not mark a retained valid value invalid. Explicit discard clears old local validation. Byte checks use actual UTF8/configured limits and existing serialized-body admission; naming and integer-day rules are unchanged.

Canonical preflight still checks current reference eligibility before a new declaration command. Known ineligible references remain distinct from unavailable evidence; declaration owner remains read-only. General metadata failures remain form problems rather than invented input errors. Retained unknown retries bypass new-value validation and use their original frozen envelope. No new metadata can silently authorize a command.

Independent source review first returned FAIL because owner/organization focus targets existed only inside a successful picker result. The corrected picker keeps one focusable fieldset/error anchor mounted while pending or failed, with a separate select ID. Summary activation focuses the actual select when ready and the stable group otherwise. The source reviewer then passed this correction and subsequent scopes-focus/canonical-status refinements. No execution was performed by that reviewer.

# Actual verification

The final full Console suite passed 1,016 cases with all thirteen actual HTTP wire samples enabled. Product build/typecheck and recovery-fixture typecheck passed. Twenty isolated actual React/Chromium provider scenarios passed; their visibility/pagehide/unload controls remain simulations rather than native BFCache or complete accessibility certification.

The pending owner/organization target renderer regression fails against actual parent ed63b97 source and passes after correction in all three languages. Two semantic mutants, removing the stable target and removing its error association, fail that intended assertion with TSX caching disabled. Original source bytes/SHA were restored and all seven renderer cases passed. The renderer does not execute transport; actual pending/503/recovery is separately exercised in the mounted browser.

The complete actual-app runner passed eighteen named scenarios, with 41 new form captures alongside the existing sixty read-state, sixty-four authentication and twenty-four command-error captures. The form matrix covers create, issuance and state errors in EN/KO/AR at1440/1280/1024/390; keyboard summary/field movement; repeated failed submission; per-field error clearing; explicit discard; rejected rationale edits without focus theft; and zero command dispatch from invalid forms. One EN1440 owner request is deliberately held, then receives injected503; the stable target remains keyboard-reachable and actual Reload restores a focusable select. Those two prerequisite captures are labeled synthetic. Scopes-group focus was corrected after an initial candidate focused its first selector instead of the designated group. Partial runs are not full passes.

TW inspection of representative captures found a real numeric RTL defect: an unisolated credential used/limit string made0 / 10 appear as10 / 0. An actual Range-geometry check then failed before correction. The complete ratio now uses the existing LTR bdi helper; final geometry checks and fresh AR390 issue/rejected-edit captures show0 / 10 in the intended order. The adjacent recorded-account/account-limit ratio uses the same isolation. All twelve empty-result states require a nonempty ratio observation and correct visual order; fresh AR1440/390 captures show6 / 200 separately from zero filtered matches. No count, limit or eligibility value changed. TW re-review passed both ratio corrections after its earlier FAIL. Native/practitioner, assistive-technology, Claude and owner FIX approval are not inferred from AI/static review.

The separate current actual authenticated consumer journey also passed: complete create payload/read-back, owner-preserving versioned declaration edit, authoritative expiry replacing a deliberately skewed browser estimate, original-session one-time Reveal/acknowledgement and canonical revocation. Twelve known-selection captures cover EN/KO/AR at four widths with no reported JavaScript errors. Those captures do not expand the form matrix into every translated normal-form state.

The scoped computed-border heuristic, direction, horizontal containment and IBM Plex checks passed for these forms. Source-bound final build integrity checks match all source inputs and emitted-file hashes. Generated captures and temporary evidence stay local; no secret, password, activation, pickup reference or command body is committed. The final manifest binds current sources and raw artifacts, including prior failed observations without overwriting them.

# Reference and completion boundary

Root consulted official [GOV.UK error-summary guidance](https://design-system.service.gov.uk/components/error-summary/), [error-message guidance](https://design-system.service.gov.uk/components/error-message/) and [W3C forms notifications](https://www.w3.org/WAI/tutorials/forms/notifications/) while refining and verifying the candidate. Adopt summary focus, links back to answers and associated inline errors; retain project localized copy and neutral presentation. No GOV.UK library, asset, literal heading mandate or decorative red border was imported. The initial candidate preceded this consultation; this record does not fabricate prior consultation or required Claude visual leadership.

This verifies local errors in the named create/issuance/state cases, with shared revocation/declaration logic source-reviewed. It is not complete M04 form/error/journey parity or feature merge readiness. Remaining full M04 parity, package100% coverage and final integration review remain. [Invariant cleanup](m04-invariant-cleanup.md) separately records the current server run; UI checks are not evidence for its coverage. Draft PR251 remains open with existing user authorization for eventual reviewed merge and remote/main read-back. Parent M03/M02 remain afterward.
