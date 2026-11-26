---
type: ValidationRecord
title: HYB-304 session query and termination boundary verification
description: Records bounded session query, closed termination-body, frozen preview and quota tests plus removal of superseded internal code.
status: draft
sources:
  - resource: ./session-ui-integration.md
  - resource: ./m04-boundaries.md
  - resource: ./session-boundary-evidence.json
---

# Scope and ownership

HYB-304 remains In Progress on machine mac_mini, branch feat/HYB-304-identity-list-paging, following pushed/read-back checkpoint2e457567cf4e7ee9368bd7bed5685b7c88707c82. Root is sole writer; /root/hyb310_review independently reviewed the design, actual cleanup and new assertions. This verifies existing M05 contracts within the remaining package gate; it does not reopen owner acceptance, certify full M04 or close M03/M02. No operational/customer data or policy was changed. All lifecycle operations use isolated synthetic fixtures.

# Behavior-preserving cleanup

Repository-wide Go search found no caller of the private selfEndViews helper. Its obsolete whole-population conversion is removed; bounded operation consumers remain. The selected termination variant's second session-key check is also removed: the preceding closed-key decoder, exact three-key length and required nonnil selection/digest/count already occupy every key slot. Earlier validators and error behavior remain intact. Independent design/source review passed both removals. Existing mixed-variant tests passed before and after cleanup. Additional extra empty/null session, generation and handles cases subsequently pin the retained closed-shape validation.

# Added contract evidence

- Explicit matching workspace and exclusive until: three actual sessions at different instants produce only the middle session for the chosen interval. Tests manually traverse bounded pages, verify cumulative lower-bound/exact count and termination, and separately exercise pure current/other matcher exclusions. The first test incorrectly assumed a one-row result meant immediate exact completion; correcting that expectation was not a product fix.
- Overlong timestamps/cursors and malformed known-session query strings refuse without rows, operations, accounting changes or peer session termination.
- Termination body parsing rejects wrong JSON types, fractional/string counts, missing or malformed selection/digest/count, orphan fields and mixed variants. HTTP checks require422, no operation and unchanged record/reference accounting; every actual peer remains usable. A failing reader and truncated/wrong-closing/trailing JSON are separate body-parser helper tests.
- Correctly authenticated malformed session cursor bodies reject unknown fields, trailing data and wrong types. These are signer/decoder robustness tests, not attacker-forgeable HTTP continuations; keys and signed tokens are not logged or committed.
- A real350-target build returns an empty/count-unknown building preview, rejects preview continuation, and rejects a genuine build token older than both current and previous tokens. Valid continuation seals exactly350 targets. Before and after acceptance, tests traverse all350 unique noncaller frozen rows through bounded100-row pages and compare the complete ordered metadata. Every accepted page preserves the operation ID, and the response explicitly accounts for350 ended, zero gone and zero not included. This is immediate blocking/manifest evidence, not proof of every asynchronous cleanup item.
- Four legitimate pending views each retain63 peers plus their record, charging exactly256 units. Budgets256/257/259 exercise legacy record allocation, frozen-target reservation and application reservation refusal. Each429 preserves all64 actual sessions, all four pending records and exactly256 charged units, with no accepted operation.
- The existing equal-time pressure test remains; a second chronological variant advances actual fixture time between six rounds. Both require oldest settled operation eviction, five remaining indexes and exactly205 units.
- Captured-session projection follows real policy epoch replacement, natural authentication age and identity role promotion under an unchanged administrator-MFA policy. It preserves handle/sign-in identity while reporting revoked/expired/blocked and disabling termination. These are helper projections; they do not claim an aged or newly blocked caller could pass complete HTTP admission. The initial missing explicit MFA-none policy value was a fixture error, not a product failure.

There are nine new top-level tests plus the retained equal-time pressure test in the targeted/race selection. Source re-review passed the complete350-row comparison, explicit outcome fields and closed mixed-variant assertions. Reviews did not execute tests.

# Actual verification and remaining scope

The restored full server short suite passed at98.8% statement coverage. Its profile retains144 uncovered weighted statement units, including62 in servicequery files. The package100% target remains unmet, without narrowing or waiver. The ten-case targeted/race selection and go vet passed. Linux amd64 and Windows amd64 server test binaries compiled successfully and their ELF/PE formats were inspected; those operating systems were not executed.

Three semantic mutants were killed: accepting extra allowed variant fields, making until inclusive, and accepting an obsolete genuine build token. Each failed its intended assertion; original working source bytes/SHA were restored and the batch's disposable private Go cache was deleted before broader validation. The restored selfend source intentionally includes only the two independently reviewed removals, not any mutation.

The [manifest](session-boundary-evidence.json) binds affected sources and temporary artifacts. Empty vet/build logs do not independently establish success; exit-zero observations belong to root's actual tool executions. Console source and its prior998-test/sixty-render checkpoint are unchanged and were not rerun for this Go-only increment. M04 full journey/form/auth-notice checks, remaining package coverage, final reviews, feature PR, authorized merge and remote/main read-back remain. Native/practitioner/AT/BFCache, Claude visual leadership and owner FIX are not certified.
