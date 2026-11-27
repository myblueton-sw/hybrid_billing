---
type: ImplementationPlan
title: HYB-304 command failure meaning and original uncertainty
description: Corrects M04 command failure wording and safe references while preserving the unknown outcome of an original request after a refused retry.
status: stable
sources:
  - resource: ./m04-consumer.md
  - resource: ./m04-action-draft.md
---

# Final status

Checkpoint record. The work it names as remaining or not yet built was completed later in HYB-304, which closed on 2026-11-27 ([closing audit](closing-audit.md)).

# Reproduction and owned scope

HYB-304 remains In Progress, machine mac_mini, after pushed checkpoint3cf95a02c4bc7253d373d32c24741aba2aadd1bd. Root owns M04 command presentation, scoped retained-intent metadata, tests/tools and affected documentation. Existing separate read-only source reviewer /root/hyb310_review and terminology/render reviewer /root/hyb310_copy_render provide independent reviews. No new agent, server contract, authorization decision or retry behavior is introduced.

The actual mounted first409 state-refusal check failed: its nested error heading still described an unrelated save/user action instead of an M04 command refusal. The current IntentNotice also omits its safe error code/request reference, while IssueControl exposes only generic blocked text. The fail-before runner /private/tmp/hb304-command-error-before.log records the failure before later scenarios.

Independent design review identified a second correctness issue: commandIntents converts a definitive refusal of an unknown retry into phase refused, which alone cannot prove the original failed. New model tests fail on missing uncertainty history. Add a presentation-only hadUnknown flag initialized false and set when a request becomes unknown/lapsed. It stays sticky through later refused outcomes until discard/dispose; a validated completed response may display completion. Preserve actual phases, envelopes, retry eligibility, replay windows and authentication behavior.

# Proposed presentation and verification

Use one M04 CommandFailure presentation with explicit unknown/refused/confirmation outcome, existing neutral FailureBox and existing sanitized ErrorDiagnostics. IntentNotice must preserve original uncertainty even when the latest retry was refused. Issuance uses its existing refusal error/retry metadata: retry refusals cannot prove original failure. Confirmation failure concerns password confirmation, not saving users. Raw server messages, unsafe diagnostic tokens, stack traces, passwords and secret pickup references must not appear.

Unknown guidance says the original may have completed and a failed retry does not prove its failure; current target metadata is not a command-completion receipt. Only a first definitive revision_conflict says expected revision no longer matches and suggests reviewing a new request after reload. Denial stays neutral about the specific target/current context, without claiming global access loss. Distinguish actual local cancellation from unverified confirmation transport; avoid promising unavailable retries. The TW candidate direction and revised sticky-history design independently passed before implementation. KO/AR candidates still require source/render review, and no native/practitioner or owner acceptance is inferred.

Pin actual first-refusal and unknown-then-refused behavior, safe diagnostics and malicious-token suppression, first-versus-retry history, unchanged request identity, disposal and replay admission. Run scoped Console tests/build plus actual rendered EN/KO/AR/width evidence and source/TW re-review. Keep existing frozen evidence unchanged. Full M04 parity, package100% coverage and final feature reviews remain open before the already authorized PR251 merge/read-back; this plan claims no completed implementation or merge.

# Implemented checkpoint and actual verification

Root implemented sticky hadUnknown metadata on retained generic intents and a scoped CommandFailure presentation. The flag is set when an outcome becomes unknown/lapsed, survives a later refused replay and remains until disposal/discard; validated done remains completion. Actual command phases, envelopes, replay admission and retry eligibility are unchanged. IntentNotice uses one phase heading with inline explanation and safe diagnostics; standalone issuance/confirmation failures retain their own heading. Issuance reads the actual refusal.error/retry shape. No user-save, invitation, installation-wide authority-loss or raw-response message is inferred.

Three new history model cases failed before implementation and passed afterward: unknown followed by five different retry refusals preserves exact original request/history; first definitive refusal differs from prior uncertainty and discard resets history; elapsed replay retains uncertainty without dispatch. Four new render cases cover localized first revision refusal, uncertain original after refused replay, suppression of malicious diagnostic tokens/raw messages, and cancellation versus unverifiable confirmation. Existing rendered read/prerequisite tests remain. The nine-case model/render selection passed after three semantic mutants failed intended assertions: forgetting history, ignoring unknown presentation and removing diagnostics. TSX caching was disabled and original working source bytes/SHA restored.

The full current Console suite passed1015 cases with all thirteen actual HTTP wire samples enabled. Current product build/typecheck and recovery-fixture typecheck passed. Twenty isolated actual React/Chromium recovery scenarios passed before the inline-only presentation correction; provider source and lifecycle behavior did not change in this checkpoint, and no provider rerun after that presentation correction is claimed. Those lifecycle/visibility/unload checks remain controlled fixtures rather than native BFCache or complete ServiceRoot password flows.

The actual complete browser runner passed seventeen named scenarios, sixty read-state variants, sixty-four issuance-authentication variants and twenty-four new command-error variants, with no reported JavaScript errors. The known revision refusal is a real API409 after another authenticated tab advances revision. For the uncertain path, the original Suspend is actually accepted by the API and its response deliberately aborted; the same byte-identical retry receives an injected409 with a safe request reference and an unrendered test message. Both the current form and retained notice keep original uncertainty without enabling a new retry. Mounted app canonical read-back separately observes Suspended; that metadata alone is not a command-completion receipt. Bodies are compared only in memory, without writing them into evidence.

The command-error matrix covers first-refused and original-uncertain states in EN/KO/AR at1440/1280/1024/390. It verifies exact localized phase heading, code visibility, absence of injected raw message, direction, document containment, IBM Plex and a scoped computed-neutral-border heuristic. Independent TW inspected representative EN1440/KO390/AR390 captures for both states, not every variant or native-language quality. Static capture review passed coherent RTL, readable identifiers and distinct refusal/uncertainty.

An initial expanded runner failed on a selector that assumed the duplicate headings produced by the first presentation version. Root changed the presentation to one phase heading plus inline body and corrected the selector rather than keeping duplicate headings to satisfy it. A subsequent partial run completed all24 captures but its direct APIRequestContext read-back lacked the normal authenticated app request context and returned403. The corrected runner performs read-back through the mounted app. These partial runs do not certify later scenarios; the complete run is separately bound. The initial fail-before assertion expected the replacement nested heading; the maintained regression instead checks one phase heading, absence of the unrelated save heading and the safe canonical code.

Independent /root/hyb310_review passed actual sticky-history/presentation source after design correction, including unknown precedence, done validation, inline heading, real issuance refusal shape and safe diagnostics; no tests were executed by that reviewer. /root/hyb310_copy_render passed all candidate locale meanings and representative actual static captures, without native/practitioner, AT, Claude or owner FIX certification. The [manifest](command-error-evidence.json) binds current source and local raw artifacts. No generated capture, activation/password, raw command body or pickup reference is committed.

# Preserved contract and remaining work

Existing issuance F2 explicitly removes an entry after some definitive retry refusals and records refusal.retry=true; its historical tests allow a later new attempt. This checkpoint changes only how that uncertainty is described. Independent producer/history review completed this scoped reconciliation and recommends preserving that explicit decision in this checkpoint: [HYB-333 F2](../HYB-333/plan.md) records the removal table and the delayed-original residual. Registry replay lookup precedes revision checking, but server step-up checking precedes registry execution, so not every refusal proves original failure. Current uncertainty copy is conservative about that limitation. A broader execution-policy revision would require its own producer-backed regression and reviewed decision; it is not silently introduced here. Generic sticky history does not itself change retry admission.

Go source is unchanged and was not rerun. Latest server coverage remains98.8% against required100%, without narrowing or waiver. Remaining full M04 journey/form/field-association parity, package coverage and final integration reviews remain open. PR251 stays draft, with existing owner authorization for eventual reviewed merge and remote/main read-back. HYB-304 stays open for M03/M02 afterward. Claude visual leadership, native/practitioner language, assistive technology, native BFCache and owner FIX remain unverified.
