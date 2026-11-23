---
type: VerificationRecord
title: B09 decision client and session attempt controller
description: Strict actual-service preparation and immutable recovery after uncertain decision responses.
status: draft
---

# Scope and review

HYB-340, machine:mac_mini, follows reviewed decision-screen plan 4d16d5e. Root implemented the raw financial command client and session-scoped in-memory attempt controller. The client avoids the generic administrative response coercion, matches preparation to exact selected request/digest/amount/currency/author/requester/policy, validates closed outcomes and verifies decision receipts against the frozen action. It refreshes the actual session before dispatch and does not automatically retry or reauthenticate.

Independent refusal_review reviewed the concrete source and found zero-year/clock-rollover timestamp acceptance and unrelated operation-reference retention. Root corrected full UTC component validation and retains operation diagnostics only for matching request identities; focused tests cover both. Subsequent source re-review passed. The reviewer did not execute tests or inspect screenshots. It required explicit refresh invalidation and graceful capacity handling at later UI integration; those remain wiring requirements.

# State and transport behavior

A session-keyed factory retains one controller across same-session remount/navigation. Selection changes and explicit invalidatePreparation clear eligibility/drafts but retain pending/unknown/recorded attempts. Scope generation rejects late preparation. Frozen selected records, submission payloads and attempts prevent alias changes. Submission sets pending synchronously before asynchronous work; duplicate submits/retries cannot create another request. An unknown result keeps the identical payload, expected revision, rationale, reason and attempt identity; a later retry refusal preserves the earlier uncertainty and original operation reference. Historical state alone never confirms that attempt.

The session pool is bounded to sixteen sessions and each controller to one hundred retained request entries; exhaustion is an explicit error, not silent eviction of unresolved attempts. The forthcoming UI must handle it outside render and explain recovery. All state is transient process memory; no browser persistence is added. Document reload loses client evidence and cannot establish failure of an earlier effect. Session credentials remain only in the transient authenticated client and are never written to evidence.

Session and command/body reads have a thirty-second deadline. A pre-dispatch session failure causes no command; a dispatched timeout, malformed/oversized success, mismatched receipt or uncertain error is unknown. Known no-effect refusals require matching request identity, an absent operation identifier and a closed code/status pair. The 32768-character response limit is checked after buffering and is not a streaming memory cap. Rationale input is constrained to the current synthetic console profile's 2000 UTF-8 bytes; the server independently enforces its configured contract and current authorization.

# Executed verification

Core unit tests grew from six to thirteen covering review corrections, session-check timeout with no dispatch, stalled response-body timeout and valid IDs that collide with object prototype names. Root found the latter during integration review and changed retained entries to null-prototype dictionaries; independent source re-review passed. The final thirteen passed in /private/tmp/hyb340-decision-core-final-tests.log. After that product correction, console regression passed943 tests (/private/tmp/hyb340-decision-core-console-reviewed.log). Final build/typecheck passed (/private/tmp/hyb340-decision-core-build-reviewed.log), with the existing chunk-size advisory. Earlier940-test/build evidence is kept distinct. No full server-suite rerun was needed for this client/test-helper-only increment. Independent platform_refusals evidence review required this precise separation, which is recorded here.

The existing opt-in Go browser fixture now accepts a bounded internal runner name and also runs TestFinancialDecisionCoreBrowser. It starts the actual isolated synthetic server, carries its session cookie only in transient environment, uses real Vite module imports on8896 and closes the server after verification. The actual authenticated integration first passed1.991s, then passed2.361s after parser/deadline corrections, then passed2.098s after the null-prototype dictionary correction (/private/tmp/hyb340-decision-core-browser-reviewed.log). Evidence /private/tmp/hyb340-decision-core-browser-reviewed/evidence.json records exact USD15 preparation, an actual approval whose successful response is deliberately discarded, duplicate suppression, preservation across selection changes, byte-identical same-attempt retry returning replay=true and actual approved source history with the current decision actor. No page errors were captured.

This browser verification invokes the client/controller modules directly. screenDecisionControls=false is explicit: it does not prove button wiring, visible forms, keyboard behavior, locale renders, layout, session-capacity UX or final visual acceptance. Those are the next active work, with the already reviewed Claude specification. No owner screen acceptance, main merge, deployment, production/customer data or payment occurred. HYB-340 and full W-23 remain active.
