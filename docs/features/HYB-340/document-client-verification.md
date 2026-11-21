---
type: VerificationRecord
title: B09 document client parsing and submission foundations
description: Bounded reads, explicit packets and real service interoperability before UI integration.
status: draft
---

# Implemented scope

[HYB-340](https://linear.app/myblueton/issue/HYB-340), machine:mac_mini. Root implemented the client foundations in [the reviewed screen plan](document-screen-plan.md). New modules under approval-inbox/documents define bounded contracts, duplicate-key-rejecting JSON parsing, context/history pagination, explicit packet validation and session-bound transport. Existing B09 financial decision source is unchanged. The new modules are not yet mounted in the screen.

Context pages are accepted only as one complete immutable set with identical request/digest/revision/header/permission, ordered unique rows, valid cursors, bounded pages and exact total. No partial form data is returned. History independently preserves document state and unresolved preparation receipts. Foreign workspace context, changed headers, stale revisions, duplicate or missing rows and malformed projections refuse. The strict parser rejects duplicate and escaped-duplicate keys, invalid Unicode, malformed JSON and excessive depth/nodes/bytes. Parsed objects have no inherited prototype.

Packet construction keeps quantity, tax, unit, evidence, declared totals and rationale explicit. Canonical decimal strings and BigInt addition preserve exact amounts, scale and declared totals without rounding. Packet text is capped at 4096 UTF-8 bytes; per-line quantity/tax inputs retain the server profile's twelve-integer-digit bound. Client validity is not a guarantee of server retained-record admission or current authorization.

Transport validates and defensively clones/freezes action, target, fields, rationale, revision and attempt before session verification. Read fields are also cloned. A replaced user/workspace/CSRF prevents dispatch. Command response streams stop at the byte ceiling and decode UTF-8 strictly. Malformed successful mutation responses, post-dispatch timeouts and ambiguous failures remain unknown. A clean refusal requires the established request/status/code/retry/next-action envelope without an operation. Matching operation diagnostics are retained; unrelated request diagnostics are not adopted. Successful receipt state, target and applicable document digest must match the frozen intent. This transport alone does not retain attempts after navigation or recover unknown effects.

# Review and corrections

Independent refusal_review initially identified mutable intent across asynchronous dispatch and absent runtime intent validation as blockers. Root added pre-await validation and deep copying, plus mutation-during-session/response tests. Review also requested total validation at context acceptance and strict duplicate-key handling; both were implemented. The initial implementation bounded response text after buffering. The final implementation instead bounds stream consumption before accumulation and cancels oversized streams. Corrective and final source/runner reviews passed. The reviewer did not execute tests.

An initial file-writing command used repository-relative paths from the console directory and failed before creating packet/transport files. Root corrected the working directory; the early typecheck covered only the files then present and is not final evidence. The first client unit run passed 49 tests in 253.207ms. Added timeout/stream/UTF-8 coverage passed, then two malformed refusal metadata cases completed the final 55-test foundation scope. Final npm test passed all 1000 console tests in 4480.748ms. Final build/typecheck passed with the existing large-chunk advisory. Logs: /private/tmp/hyb340-document-client-first.log, /private/tmp/hyb340-document-client-expanded.log, /private/tmp/hyb340-document-client-console-final.log and /private/tmp/hyb340-document-client-build-final.log.

# Actual service interoperability

The opt-in TestDocumentActualTypeScriptClient launches the real TypeScript client against ordinary sample Server runtime with separately provisioned actors and an ephemeral loopback listener. Cookies are transferred only through transient process environment and never emitted in artifacts. The initial runner failed before evidence output because a root-level .ts entry was transformed as CommonJS despite top-level await. The diagnostic run established that cause; renaming the entry to .mts made module intent explicit. Initial failure log: /private/tmp/hyb340-document-client-runtime.log. The corrected integration passed in 0.809s.

Integration plus source rules passed in 1.218s after the streaming/refusal corrections. Evidence: /private/tmp/hyb340-document-client-runtime-verified.json; log: /private/tmp/hyb340-document-client-runtime-verified.log. It verifies actual context and proposal permission, strict packet proposal, discovery by a different actor, independent confirmation, real internal ERP preparation, identical-key replay, owned receipt discovery and revocation preserving preparation history. Server read-back requires net 15.00, tax 1.00, quantity 5.00, one owned preparation, revoked document state and no external ERP commands. Advice and inspection are read through the generic envelope parser here; dedicated domain parsers are still pending. This is Node client interoperability, not browser verification or a lost-response recovery test. No production Go code changed; the full Go suite was not repeated for this client increment.

Before commit, the fixture replaced the POSIX executable shim with Node and the installed package's declared tsx/dist/cli.mjs entry. Independent source review found no blocker. The final actual integration/source run passed in 0.982s; evidence and log are /private/tmp/hyb340-document-client-runtime-portable.json and /private/tmp/hyb340-document-client-runtime-portable.log. This removes a shim dependency but only macOS execution is verified. Independent platform_refusals also passed the source/document/runtime evidence review without executing tests.

# Remaining work

Dedicated retained-basis, action-advice and preparation-observation parsers, one session-owned controller, structured form/history/actions and exact unknown recovery remain to implement before mounting B09. Then run actual authenticated browser journeys, EN/KO/AR layouts and independent Claude/TW/source re-review. No new screen, owner acceptance, native-language sign-off, external financial action, persistence, merge or deployment is claimed. The complete owner goal and ticket remain active.
