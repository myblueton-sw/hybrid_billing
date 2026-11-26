---
type: ValidationRecord
title: HYB-304 issuance draft recovery across authentication refresh
description: Records the reproduced issuance draft loss, lease-scoped fix and actual authentication, lifecycle and mutation verification.
status: draft
sources:
  - resource: ./m04-consumer.md
  - resource: ./issue-draft-evidence.json
---

# Reproduction and correction

On HYB-304, machine mac_mini, root reproduced an actual mounted issuance form losing its untouched one-day duration and rationale after successful password confirmation and canonical reload: the duration became the configured30-day default. The fail-before authentication runner stopped at the first EN1440 assertion, after the preceding thirteen interaction scenarios and sixty read-state variants had passed. That baseline is not a complete authentication-matrix pass.

Root added one bounded, memory-only draft per authenticated recovery lease. It stores exactly a frozen target, days and reason, never passwords, secrets or pickup references. Same client/user/workspace/CSRF locale/view remounts and canonical refresh retain these values. Replacement scope, authentication loss and disposal fence old access and clear them. Accidental unload is guarded while a draft exists; explicit sign-out clears it. No browser storage is used.

The draft cannot silently replace another target. The operator can return to the canonical retained target or explicitly discard that exact draft. Clearing requires both object identity and target, so an older completion cannot erase a newer same-target edit or another target. Inputs are admitted against effective UTF8/value/rationale/serialized-body bounds. A new submission rechecks current canonical eligibility and limits; a previously frozen unknown submission retains its original envelope and replay window. A known refusal without an issue entry retains editable values. These last refusal semantics were source-reviewed; no separate mounted known-first-refusal assertion is claimed.

# Actual checks and independent review

Current Console tests passed1003 cases with all thirteen actual HTTP wire samples enabled. Product typecheck/build and recovery-fixture typecheck passed. The restored five draft-model tests passed after three semantic mutations each failed the intended assertion: cross-target overwrite, stale completion clearing a replacement, and scope-fence bypass. TSX caching was disabled; original working source bytes/SHA were restored.

The current authenticated browser runner passed fourteen named scenarios, sixty read-state captures and sixty-four authentication captures with no reported errors. EN/KO/AR at1440/1280/1024/390 exercise distinct injected unsupported-MFA/unavailable/reauthentication-required metadata, actual password-confirmation requests (twelve), canonical refresh retaining one-day/rationale values, and locale/view remounts without automatic submission. Additional390-width states cover another-target keep/discard in all three languages and actual sign-out/new sign-in with an empty new-lease draft. Metadata overrides are synthetic prerequisites; password confirmation and session replacement use the actual app/backend. The strengthened unknown issuance scenario retains an unrelated unsent draft through byte-identical original replay after actual suspension. Request bodies are compared only in memory.

Eighteen isolated actual React/Chromium recovery scenarios passed, including same-lease remount retention and user/workspace/CSRF/client replacement plus actual client authentication-loss notification. Their unload checks dispatch a synthetic cancelable beforeunload event; this is not native browser-navigation/BFCache certification.

Independent read-only /root/hyb310_review passed the design before implementation and the actual source afterward, including bounded immutable fields, identity-safe clearing, scope cleanup, current admission and frozen unknown retry. It did not execute tests. /root/hyb310_copy_render passed candidate terminology and inspected actual EN1440 refreshed values, KO390/AR390 other-target notices, AR390 empty-password dialog and EN390 new-session empty draft. That static review is scoped to those captures, not all language/state combinations.

The build-integrity observation binds678 source inputs and439 artifact files to revision878afadede2346608a96c558e863d40f3cfda68a5e611d0af283f248396301ec; current source fingerprint and every built-file hash matched. The manifest binds raw temporary artifacts; they remain local and may be removed by reboot. No generated screenshot, password, activation data or command body is committed.

# Remaining integration gates

This is an incremental bugfix checkpoint after b338d1147ba18d51ef217af1f8d90e00fc1e03ec, not feature completion. ActionForm's separate local unsubmitted state/revocation rationale remains an explicit open remount risk, requiring its own actual reproduction and correction. Remaining full M04 journey/form/error-copy parity and final integration reviews remain open. Go source did not change and was not rerun: the latest server short result remains98.8%, below the required100% package target without waiver. Draft PR251 is not marked ready. Claude visual leadership, native/practitioner language, assistive technology, native BFCache and owner FIX are not certified. Parent HYB-304 remains open for M03/M02 after M04. Existing owner authorization covers eventual reviewed merge and remote/main read-back.
