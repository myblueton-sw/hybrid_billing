---
type: Verification
title: Result authority discovery and provisioning context checkpoint
description: Server-only evidence for exact governor-protected proposal and option reads before UI integration.
status: draft
---

# Scope

HYB-341, machine:mac_mini. Root added result_authority.history and result_authority.context, backed by the actual result-domain registry and current identity/grant facts. This server prerequisite does not complete the M03/P02 user flow. No external rights, deployment, merge or owner acceptance occurred.

# Checks completed

Actual HTTP tests verify active ordinary human recipients, proposer exclusion, matching owned console ceilings, cross-capability cursor refusal, complete ten-row paging, time-driven proposal expiry without service-revision change, invalid fields/cursors, unauthorized ordinary users, unavailable effect lookup and orphan ceiling refusal. A population of 1,001 isolated synthetic humans verifies complete recipient bounds. Unknown post-write effects remain unresolved after history/context inspection, and held-state reads remain available. Tests also verify fractional-second ceiling boundaries can be reused exactly by a human grant proposal and missing applied-console reverse associations fail closed.

Initial targeted HTTP run passed in 0.845 seconds. The final result authority/permission race run passed in 4.828 seconds. Final server vet passed. Full Go short regression passed all 83 packages, including server 147.650 seconds. Independent source reviewer required exact timestamp precision and stronger reverse-association checks; root corrected both and received scoped corrective source PASS. The reviewer inspected source/tests and did not execute them.

# Failures that changed the work

Initial tests used an inactive sample member and null request maps; the fixture now activates its synthetic recipient and sends the proper empty object. The new command definitions initially omitted EvidenceStale from declared refusals, causing generic internal errors instead of the intended stale response; the declaration is now explicit. The expiry test originally allowed the synthetic login to time out; it now renews authentication after advancing time, verifies the service revision is unchanged and then exercises snapshot expiry. No authorization or expiry check was removed.

# Open acceptance

Frontend strict parsing/session and attempt state, M03/P02/B09 navigation, actual multi-user provisioning-to-use browser journeys, localization and rendered independent review remain open. Existing B09 result review acceptance does not certify these new provisioning controls. Current decision checks are deliberately deferred to submission; historical rows are not ready/authorized flags. Read-only inspection never settles unknown grants.


# Client foundation checkpoint

The isolated M03 result-authority client now parses closed history/context/inspection responses, preserves UTC nanoseconds, enforces projection-bound pagination and binds protected requests to the current actor/workspace/CSRF session. The JSON parser was mechanically extracted to a service module; existing document callers retain their domain-specific error wrapper. An opt-in Go fixture exports only synthetic protected response bodies, without session credentials, for actual producer/client interoperability.

The fixture generation test passed in 0.500 seconds. Seven initial client contract/read tests passed against the generated wire data. Read transport and affected document/result transport regressions passed 133 tests with no failures or skips. Focused client tests subsequently cover proposal/decision receipt identity, interrupted effects, exact refusal classification and historical revoked/expired receipts. Typecheck passed after correcting test-only heterogeneous object inference. Deterministic generated nanosecond boundary cases use the existing node:test toolchain; no random property-testing or shrinking claim is made.

Independent source review identified two mutation-client defects: the rationale validator did not match the registry's 1,024 UTF-8-byte/control-character bounds, and ordinary invalid_stage refusals were incorrectly classified as uncertain effects. Root corrected both and added regression cases. Corrective source review passed; the reviewer executed no tests. Historical action review controls now distinguish proposer withdrawal, independent approval/rejection and original-decision-actor recovery with the identical rationale. These controls do not certify current authority; the server revalidates it on submission.

At this foundation checkpoint, attempt retention, asynchronous screen state, forms/navigation, actual mutation interoperability, full provisioning-to-use browser QA and rendered design review were pending; the following checkpoint updates that boundary. This checkpoint is client foundation evidence, not completion of the M03/P02 screen or the full project.


# Mounted read and decision checkpoint

M03 now mounts a separate result permission area before any generic access hooks. B09 links to that area, and P02 offers its link only for the actual console client. A foreign client route is rejected before mounting the protected result component. History and exact inspection drive historical actor-based decision review; current authorization remains a server submission check. New proposal creation is not yet connected.

Session-owned immutable attempts retain uncertainty across remounts and after clean retry refusals. Original-actor fresh-key recovery preserves decision identity and rationale. Independent review required disposed controllers to stop dispatching and mutation refresh to respect newer history/context/selection generations. Those corrections and focused race tests pass. Review also required actual state/draft fingerprints and explicit discard guards on local navigation; both were corrected and received scoped source PASS.

Final focused authority tests passed 34 cases with no failures or skips, in 214.423958 ms. Typecheck and diff whitespace validation passed. An actual Chromium app run intercepted protected reads using the synthetic Go-produced wire fixture: EN/KO/AR at 1440/1024/390 each rendered history/detail with exact fractional UTC values, correct direction, no horizontal page overflow and no clipped controls. The runner observed no page errors or unexpected API calls. A populated decision rationale survived declined local refresh and back navigation. This is browser/client evidence using intercepted reads, not live server mutation interoperability.

Independent TW corrected original-actor/new-request recovery wording, the later-recovery historical note, 업무 공간 terminology and the exact ERP result evidence apply capability label. Corrective TW source PASS covers these changes. The reviewer viewed EN desktop, KO tablet and AR mobile read/detail captures; recovery/settled-history wording remains source-only. No native-language or owner acceptance is claimed. Claude rendered review, new proposal forms, keyboard/focus completion and real multi-governor provisioning-to-use acceptance remain open.


# Proposal and actual provisioning checkpoint

The result area now supports deliberate console-ceiling and person-permission proposals. Capability and policy come from server context; person choices bind exact current recipient and ceiling pages. Independent cursors are retained, mixed policy/revision responses fail closed, and invalid validity or a one-nanosecond ceiling escape cannot enter review. Review freezes the target, fields, rationale, revision and HTTP attempt. Changing scope or paging requires explicit discard before clearing selections. The model filename is proposal-model.ts to avoid case-insensitive resolution against Proposal.tsx on macOS.

Browser evidence covers all 18 console/person × EN/KO/AR × 1440/1024/390 form/review combinations using intercepted synthetic contexts. It verifies no overflow/clipped controls, review-heading focus, scope selectors hidden during review and lossless submitted UTC values. A separate synthetic paging scenario verifies declined transitions preserve inputs, accepted transitions clear stale selections and the two cursors remain independent. These rendered fixtures are not the actual-server acceptance below.

The actual-server fixture uses the ordinary sample runtime assembly, two distinct governors and an ordinary active human recipient. It creates a console proposal, approves it independently, creates a person proposal against that ceiling and approves it independently. The second real grant write deliberately fails after writing. The same decision actor inspects and recovers the exact recorded decision/rationale through a fresh HTTP attempt. The old uncertain attempt remains historical and its retry controls disappear after recorded recovery. Server read-back verifies exactly two writes/two additional grants, effective ApplyERPResult capability for the recipient and denial of the separately tested financial CheckApproval request. This is not exhaustive financial-authority coverage.

Initial actual EN happy flow passed in 4.750 seconds; the interrupted/recovery flow passed in 5.167 seconds. The full actual provisioning/recovery flow then passed all nine EN/KO/AR × 1440/1024/390 combinations in 36.000 seconds. The command allowlist and exactly one expected injected internal_error per scenario are asserted. No credentials are written to arguments, captures or evidence files. These tests do not send external ERP commands or use customer accounts.

Full console regression passed 1,124 tests with one existing opt-in skip, no failures, in 5,187.974666 ms. Typecheck passed. Build passed in 459 ms with existing bundle-size warnings. Independent source review passed the proposal lifecycle and actual fixture/runner scope; the reviewer did not execute tests. TW corrected page-scoped active-ceiling wording, proposal uncertainty versus decision recovery, exact 업무 공간 terminology and RTL UTC isolation. Representative corrective Arabic renders passed TW review; native-language/owner approval remains separate.

Actual Claude visual review found radio-column inheritance, editable scope controls remaining above review, cramped heading/actions, missing review facts and hover contrast. Root fixed the shared-label flex-direction inheritance with a scoped row rule, hid scope controls during review, added recipient login/ceiling window/lifetime/observation facts, separated RTL timestamps and added scoped neutral dark primary hover styling. All 18 corrected form/review cases passed; four EN/KO desktop/tablet rest-and-hover cases assert actual background/foreground colors and pass. Narrow hover corrective Claude review passed after inspecting actual EN desktop and KO tablet rest/hover renders. At that earlier hover-only checkpoint, the runner asserted both rest and hover colors but its JSON recorded only hover colors; rest colors were supported by the executed assertion and captures. The final proposal runner below explicitly records both color readings. The reviewer did not certify keyboard focus. History/error-state rendered review, full keyboard flow and remaining acceptance checks are still open; this is not owner FIX, a merge or completion of the full product goal.


# Keyboard follow-up

Detail entry now focuses its heading, decision entry focuses the rationale, invalid rationale returns focus to that input, and frozen decision review focuses its heading. The actual-server EN/AR 390 journeys passed again in 8.448 seconds. For proposal, approval and interrupted-recovery review stages, the runner waits for heading focus, presses Tab and verifies a button has focus-visible with a nonzero outline. This checks the primary review-to-send transition, not every keyboard navigation path or history/back focus restoration. Final typecheck and diff validation passed after these changes. Remaining history/error render acceptance and broader focus traversal still require scoped follow-up before the consolidated PR checkpoint.


# Final read and validation surfaces

The repository now includes portable argument-driven read and proposal browser runners. The read runner covers nine localized history/detail combinations, mobile return focus to the selected row, declined local navigation with preserved rationale, proposal-close focus restoration, and 24 empty/denied/unavailable/malformed refresh combinations across EN/KO/AR desktop/mobile. Old rows and detail are removed, no unexpected API requests/page errors occur, and page overflow checks pass. These use intercepted synthetic responses; context headers are deliberately adapted from the fixture and are not claimed as untouched producer output.

The proposal runner covers all 18 form/review combinations, focuses invalid-input summaries, follows their links to fields, verifies review-heading focus, and explicitly records both rest and hover colors. Corrected final renders pass with no page overflow or clipped review controls. The earlier draft navigation runner initially refused its own about:blank transition while the dirty-state effect was settling; it now accepts explicit harness beforeunload transitions while retaining dismissal for the local discard-confirm cases. No product guard was weakened.

Final source review passed the post-commit-render focus restoration and runner boundaries. Final Go server vet passed. Final build passed in 410 ms with the existing bundle-size warning. Actual Claude read/error/validation surface review completed with scoped PASS after inspecting six named EN/KO/AR desktop/mobile captures. These checks remain scoped development evidence, not native-platform, owner or merge acceptance.


# Consolidated review boundary

Source, scoped TW and actual Claude visual corrective reviews have no unresolved blocking finding for this increment. The final Claude review retains four non-blocking suggestions: stronger field-local validation cues, a focus outline on the error summary, a more compact long grant-ID display and removal of the empty-list selection prompt. These are not hidden acceptance passes. Full native-language, other-OS and owner FIX remain unverified. The final browser runner proves focus returns to the selected history row (or its heading fallback), not to the back button; root retains the actual assertion as the evidence if review prose differs.

This increment implements M03 result permission provisioning and its P02/B09 entry points. The overall product goal still includes other planned billing flows and integration/persistence work. No deployment, merge, external ERP command, customer-data use or ticket completion is implied by this checkpoint.
