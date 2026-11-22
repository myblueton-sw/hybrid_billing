---
type: VerificationReport
title: Governed ERP-final request and decision integration
description: Evidence and remaining acceptance boundaries for the synthetic final approval workflow.
status: draft
---

# Scope

HYB-341, machine:mac_mini, branch feat/HYB-341-erp-result, draft PR252. Root implements and integrates. refusal_review provides independent read-only financial and client review; platform_refusals reviews terminology. The actual Claude CLI ui-designer leads visual composition and corrective rendered review. This report covers the uncommitted integration after d9f7e6a, not the complete product goal.

# Implemented behavior

Dedicated ERP-final commands expose advisory preview, governed request, review, preparation and independent approve/reject actions. The exact owned subject binds current confirmed/applied result evidence, current upstream evidence, reported net-plus-tax gross, final policy and contributors. Approval and current approved-consumer access require owned whole-gross authority. Rejection preserves independent permission and current evidence checks without requiring a sufficient amount limit. Trust hold refuses request, approval and current approved-consumer validation while retaining governed rejection. The validation helper consumes nothing. Seeded unowned records cannot become actionable.

Charge and final approvals share a bounded projection with explicit distinct amount profiles. Combined capacity is checked before insertion. M01 reports pending final requests separately; revoking requester authority does not pretend to recheck an already-pending request. B09 final decisions preserve their fixed command family and exact final receipt contract.

The request region binds selected result identity, authenticated requester, exact server amounts and rationale to an advisory preview. It creates no reservation. Send freezes the immutable request; uncertain responses retain the identical retry identity and forbid a new competing submission. Successful receipts remain requests, not approvals or external ERP effects. Current checks are performed again by the server. Navigation, existing result actions and unresolved final requests interlock. Session binding masks the previous store synchronously; recorded local retry details are not described as durable server records.

# Executed checks

- Full server package: go test ./apps/api/server -count=1, PASS200.834s. Production source did not change during that run. Additive browser read-back assertions were exercised separately afterward.
- Broad financial-final, charge, approval and source race checkpoint: PASS52.928s, documented in the implementation plan. No full-package race claim.
- Complete console test script: 1,138 PASS, two skipped, zero failed, 5.587s. This ran before the final presentation-only label/alignment changes.
- Final console TypeScript/Vite build and server go vet: PASS after presentation corrections. Existing Vite configuration and bundle-size warnings remain; no new dependency was introduced.
- Actual final decision browser checkpoint after wording correction: PASS15.303s, 54 cases, zero page errors.
- Actual final rejection browser rerun with current boundary wording: PASS12.100s, 38 cases, zero page errors.
- Expanded actual final-request browser after visual and keyboard/session corrections: PASS14.974s, 32 cases, zero page errors. Expected unauthenticated responses occur only after actual logout. Nine locale/viewport combinations cover empty, validation and advisory preview; additional states cover a delayed preview discarded after New read, pending selection lock, deliberately lost post-effect response and byte-identical retry. Go read-back verifies one additional owned pending gross16 request, the actual requester and zero decisions. The approval-history destination is visible.
- git diff --check: PASS. Source/test file lengths remain below the 1,500-line boundary.

Decision browser evidence distinguishes actual approval/replay and identity confirmation from injected denied/held preparation and selected reauthentication/permission-refusal responses. Those injected states prove rendered handling, not actual revocation/hold execution in that browser run.

Browser fixtures use loopback synthetic servers, isolated browser sessions and explicit Chromium/Playwright paths. No credentials or generated captures are included in Git. The request fixture advances its synthetic clock one second per HTTP request: its original frozen clock exhausted the real command bucket. Production rate limits were not changed. An initial test destination locator incorrectly matched hidden desktop/mobile copies and was corrected to require visibility. Both failures are test evidence, not successful runs.

# Independent review and corrective findings

Backend and strict client source checks passed within their recorded scope. UI source review required guarding result-history retries, including final state in screen review fingerprints and removing focus stealing on historical remount/late completion. Corrections received scoped independent source PASS. Terminology review required distinguishing editable evidence drafts from unresolved effects and local retry details from server records. Five actual request crops and later short rationale labels received scoped TW PASS; this is not native-language acceptance.

Claude's first actual rendered review returned FAIL for RTL rationale alignment, input instructions reused as summary labels and the missing final-row separator. Root corrected each; it also removed contributor item separators and emphasized the reported gross. Computed scoped border checks first found inherited primary-button accent borders; explicit neutral overrides corrected normal/hover/disabled states. The actual corrective Claude CLI invocation exited0 and returned scoped visual PASS for all three prior blockers, inspecting preview-ar-1440, preview-ko-390, unknown-ar-390, recorded-ar-390 and page-preview-en-1440. It reported no new blocker in those five images; keyboard focus, font loading and other viewports were explicitly not certified.

# Limits and remaining work

The API remains within the explicit synthetic in-memory boundary. This report does not establish external ERP connectivity, persistence, issuance, posting, delivery, payment, owner screen FIX, native-language acceptance or other-OS operation. Current approved-consumer validation does not itself implement downstream issuance. Actual logout and protected return, keyboard validation/retry, error/preview focus and keyboard-visible send focus were verified across all nine variants. In-place identity replacement, the mixed result-retry interlock and assistive-technology rendered scenarios remain to be verified. No full-page audit of unrelated screens or remote CI success is claimed. Commit/push and PR update remain pending; no merge is authorized by this report.

# Evidence locations

Plans: [ERP-final approval](erp-final-approval-plan.md) and [request screen](erp-final-request-screen-plan.md). Local execution artifacts: /private/tmp/hyb341-server-full-current.log, /private/tmp/hyb341-console-full.log, /private/tmp/hyb341-final-request-render, /private/tmp/hyb341-final-decision-render, /private/tmp/hyb341-final-rejection-render, /private/tmp/hyb341-final-request-visual-review.txt and /private/tmp/hyb341-final-request-visual-corrective.txt. Local artifacts are supporting evidence, not portable committed deliverables.

Independent refusal_review evidence review read the terminal server/console logs and all three browser evidence summaries. It requested the injected-state and nonconsuming-helper distinctions above; root incorporated both. The reviewer did not rerun suites, inspect screenshots or independently certify the production-source freeze during the server run.

# Keyboard and session follow-up

The runner uses Enter for invalid check and identical retry, waits for error/preview focus, and verifies a visible outline on the send control after actual Tab/Shift+Tab navigation. A first attempt incorrectly expected focus-visible after mouse/programmatic focus; corrected test input exercises keyboard modality without changing product focus CSS. Computed money font family contains Plex; this is not proof of a specific semibold font file. Actual logout removes all request/outcome evidence and a protected return displays sign-in. The existing result interlock now tells users to clear only a draft; unresolved attempts reuse the separately reviewed locked message. Scoped TW source review passed this EN/KO/AR correction. Final request run14.974s includes Go read-back and expected post-logout401 responses.
