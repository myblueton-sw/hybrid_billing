---
type: ImplementationPlan
title: B09 governed final approval request preview and decision flow
description: Bind reported gross preview and immutable request attempts to selected ERP result evidence.
status: draft
---

# Scope and ownership

HYB-341, machine:mac_mini. Root implements and integrates; refusal_review independently reviews financial/public boundaries; platform_refusals reviews terminology and rendered evidence. B09 final decision controls reuse the existing workbench and support EN/KO/AR at 1440/1024/390. New scoped item B09:results:final-request adds a request region to selected reported-result detail. M01 gains the final-request pending impact row. No owner acceptance or external ERP effect is authorized by this work.

# Actual design input

The required local Claude CLI was invoked with --print --agent ui-designer --allowedTools Read and completed with exit 0. Its transcript is /private/tmp/hyb341-final-request-design.txt. Its self-description saying no CLI process ran is not the launcher evidence; the root observed the actual subprocess completion. The reviewer inspected current final decision crops, earlier result-list captures and Detail.tsx. Earlier recovered.png paths labelled mobile/Arabic were actually English desktop list views without selected detail; they are not evidence for those claimed viewport/locale/detail states. Current decision crops also do not alone prove full-width layout.

The established external reference is the official [GOV.UK check answers pattern](https://design-system.service.gov.uk/patterns/check-answers/), already inspected for the replacement screen plan. Adopt grouped readable effect/target summaries and edit-before-send; reject fixed content width, foreign branding and mutable-history edit links. No dependency or third-party asset is added. The CLI used the recorded reference notes and did not independently browse the source or read the frontend-design skill. Root retains responsibility for the installed skill and actual full-page verification.

# Request region contract

Place the bounded region after the selected result's existing action form/review, using a neutral top separator and full available detail width. Keep local Plex, square controls, visible focus, logical CSS and isolated IDs/money. Entry asks for a request reference and rationale, then explicitly checks the server preview. Server permissions and current evidence determine readiness; recorded result/application state alone is insufficient.

The frozen preview leads with exact reported gross and its visible basis, then server-provided net/tax/currency, source/result author/requester/contributors/policy and the user's request/rationale. Technical digests, checked time and global revision remain in collapsed details. No browser arithmetic or inferred currency is allowed. Preview is advisory and reserves nothing. Enable sending only for evidence_ready; show distinct hold, capacity and target/dependency refusal states. Root resolves the designer's open readiness question this way to match the fail-closed service contract.

The new region and existing action editor must not mix competing drafts. Scope changes invalidate preview and preserve immutable pending/unknown attempts. One actual send freezes family, target, result digest, expected revision, rationale and attempt identifiers. Duplicate submission is suppressed. Unknown outcome permits only explicit retry of the identical attempt, retaining uncertainty after a later permission refusal. Reauthentication never automatically sends. A known no-effect refusal may permit a fresh check; never claim nothing was sent for an unknown effect. Recorded receipt is a request, not an approval, and may link to the exact final approval inbox item. Do not call response as_of a recorded-event timestamp; omit that unsupported designer field.

# Current decision verification

The expanded server projection/final/charge/source race run passed in 52.928s. Actual final decision browser coverage passed 54 scenarios in 14.473s; rejection passed 38 in 11.099s, with no page runtime errors. These cover locales/widths, exact USD16 basis, validation/focus, actual approve/reject, lost response after actual effect, immutable replay, explicit reauthentication, refusal retaining uncertainty and success independent of list refresh. Some denied/hold/reauth responses are explicitly injected for rendered-state coverage; actual backend hold behavior is separately tested.

Rendered work found and corrected two scoped defects: outcome panels exposed basis only as an accessible money label, so the basis is now also visible; widening the shared decision predicate enabled the charge-only document panel for final records, so document identity now explicitly requires the charge family. The rejection runner verifies absence of that panel. It also verifies actual unbound seeded final rows and injected legacy capability cannot expose final controls. Initial legacy-capability navigation retained the prior document; the runner now navigates through about:blank before the normal-capability flow instead of treating stale rendered state as a new read.

Root inspected the actual mobile checked surface and failure captures. Independent rendered terminology and final Claude visual acceptance are pending. The request-preview client/UI is not yet implemented; this plan is not completion of B09 or the full product goal.

# Rendered terminology correction

Independent review of five named current captures found that the shared decision boundary saying it does not confirm an ERP amount conflicts with recording StageERPFinal approval. The EN/KO/AR boundary now excludes changing amounts in an external ERP instead; this accurately separates the recorded financial decision from external effects for both charge and final stages. Actual final approval rendering is being rerun after that text correction. The earlier browser passes do not certify this changed wording.

The request UI must also avoid promising that an identical retry resolves uncertainty. Say that it resends the identical request and may still return an unknown outcome. After reauthentication, do not claim an uncertain dispatched attempt was never sent; retain its uncertainty. These are required corrections to the initial designer draft, not completed request UI acceptance.

# Request client foundation

The strict preview parser, fixed-family transport and session-scoped attempt store are implemented under final-request. Preview binds selected result identity/digest/author/currency and authenticated requester, validates bounded sorted contributors and canonical server gross, and preserves reported net/tax strings without arithmetic. Submission uses only erp_final.request and frozen expected revision/result identity/rationale/attempt identifiers. Exact final-stage pending receipts are required; malformed or lost dispatched responses remain unknown. Known no-effect refusal classification is conservative. Late checks cannot cross selection changes; duplicate sends are suppressed synchronously and unknown retries retain the same immutable intent and prior uncertainty. Successful request receipt clears its own draft while retaining the historical entry.

Independent client review required freezing the exported preview input before asynchronous session verification, even though the store already passes frozen scope. This is corrected and tested by mutating every original scope identity field during a deferred session response. Canonical gross also rejects 16.00 when the service contract emits 16. Seven focused tests passed in 285.484ms and the console build passed with existing bundle warnings. The new test is included in the normal console test script. Scoped re-review is pending; the component/session lifecycle and request UI have not yet been wired or rendered.

The final-decision browser rerun after the external-ERP wording correction passed in 15.303s. Rejection captures still need regeneration for that revised wording; earlier rejection captures are not current copy acceptance.

# Mounted request UI checkpoint

The request region is now mounted in the selected result detail, with session-scoped attempt outcomes retained at screen level. Scope and session changes invalidate preview; existing result actions and final requests interlock. Independent source review found three integration defects: result-history retry bypassed the final-request lock, review fingerprints omitted final draft transitions, and historical outcomes stole focus on mount. Root corrected all three; refusal_review returned scoped corrective source PASS, without claiming rendered acceptance. The review fingerprint includes scope, reference, rationale, preview, errors and attempt phases. Completion focuses only inside the same outcome area; explicit Send moves focus to the new pending outcome.

Technical-writing review required the blocked message to distinguish an editable evidence-action draft from an unresolved effect. EN/KO/AR now instruct users to clear only drafts and inspect unresolved outcomes. Local retry-detail wording also avoids implying that reloading deletes durable server records. Native-language acceptance remains unverified.

The new opt-in TestFinancialFinalRequestScreenBrowser uses an actual synthetic requester session and server, not fixture-only command replies. Its first run reached preview in five locale/viewport combinations before the frozen test clock exhausted the server command bucket. Instrumented reproduction confirmed HTTP 429 rate_limited on history/preparation reads, with no page errors. The request-only fixture now advances its synthetic clock one second per HTTP request; production command limits remain unchanged. This is explicit test-time simulation, not a production rate-limit fix. Final rerun and rendered independent acceptance remain pending at this checkpoint.

# First actual request journey pass

TestFinancialFinalRequestScreenBrowser passed in 11.401s after the synthetic-clock correction and a runner-only destination locator correction (desktop/mobile copies require selecting the visible row). The runner recorded 30 cases: empty, invalid and server-preview states for EN/KO/AR at 1440/1024/390, plus Arabic-mobile pending, deliberately lost-response unknown and recorded outcomes. Actual erp_final.request completed on the synthetic server before its response was dropped; retry submitted byte-identical request JSON and returned a recorded receipt. The request reference cleared on success and the result link reached the visible approval-history row. No browser page errors or unexpected HTTP refusals were observed. Evidence: /private/tmp/hyb341-final-request-render/evidence.json and corresponding PNG files; execution log: /private/tmp/hyb341-final-request-render.log.

Root inspected the English mobile preview and Arabic mobile recorded crops. This pass does not yet cover every required asynchronous interlock, late response, session replacement, computed neutral-border check or independent Claude visual acceptance. It proves this synthetic request journey, not production ERP connectivity, full-product completion or owner acceptance.

Independent platform_refusals corrective terminology review returned scoped PASS after actually inspecting preview-en-390, preview-ko-390, preview-ar-1440, unknown-ar-390 and recorded-ar-390. It confirmed the request/approval and gross/net/tax distinctions, advisory preview, uncertain retry and local retry-detail limits. A minor Arabic desktop UTF-8 label wrap remains readable. This review did not execute tests or certify full-page, native-language or owner acceptance.

# Expanded integrated verification

The complete console script passed 1,138 tests, skipped two and failed none in 5.587s. The current final-rejection browser rerun passed 12.100s after the external-ERP boundary wording correction. The expanded final-request runner now checks computed neutral borders in every captured state and captures all nine full-page previews. It detected inherited primary-button colored borders; the scoped final-action CSS now overrides normal, hover and disabled borders with the neutral control token. A delayed preview released after New read cannot restore the discarded preview or draft. Pending result inspection cannot replace the selected detail. Final request browser verification passed 16.885s, with 31 evidence cases and no page errors or unexpected HTTP refusals. Go read-back additionally confirms exactly one additional owned pending request, reported gross16, the actual requester and no approval decisions. These checks do not yet establish all session-replacement and keyboard/assistive-technology scenarios.

The full server suite and actual Claude CLI visual re-review were launched against the current integrated work. Their completion remains pending here. The linear HYB-341 checkpoint was updated while retaining In Progress and machine:mac_mini; no commit, push or merge is implied by that update.

# Visual corrections and full regression

The actual Claude CLI ui-designer invocation exited0 and inspected eight named images, including full-page EN/AR desktop and KO/AR mobile plus outcome crops. It returned a scoped visual FAIL: Arabic rationale was aligned by the user text direction rather than the RTL value column; the summary reused required/byte-limit instructions; the inherited result stylesheet removed the last row divider. Root corrected the value to an auto-direction bdi inside an inherited-direction dd, shortened only the summary key to Rationale / 사유 / المبرر, and overrode the final-row separator within this summary. The optional contributor-list border cleanup and gross emphasis were also applied. Independent TW returned corrective source PASS for the short labels and preserved input instructions. No native/owner acceptance is claimed.

The expanded actual-server browser rerun after these corrections passed13.858s with 31 cases. Claude corrective visual re-review is running separately. The integrated full server package passed200.834s; its run began before the additive browser-only post-readback assertions (which passed in the later opt-in runs). Production source was unchanged during that server run. Final console build before these last presentation changes passed; a post-correction typecheck/build remains necessary.

The actual corrective Claude CLI invocation completed with exit0 and returned scoped visual PASS for all three blockers in five named captures (preview-ar-1440, preview-ko-390, unknown-ar-390, recorded-ar-390, page-preview-en-1440). No new scoped blocker was found. It explicitly did not certify keyboard focus, font loading, 1024 layouts, native-language or owner acceptance. Post-correction console build and server vet passed. Consolidated current evidence and remaining checks are in [ERP-final workflow verification](erp-final-workflow-verification.md).

Final keyboard/session follow-up passed14.974s, 32 cases: keyboard validation/retry, error/preview focus, visible send focus after actual Tab navigation in nine locale/width variants, actual logout removal and protected return. Expected post-logout401 responses are distinguished from unexpected refusals. The unresolved existing-result interlock now uses the reviewed locked text, while only its draft branch offers clearing; independent TW source review passed. See the consolidated report for remaining in-place session replacement, mixed retry and assistive-technology limits.
