---
type: VerificationRecord
title: B09 document inspection and session controller
description: Strict historical projections, retained attempts and verified response-loss recovery before screen mounting.
status: draft
---

# Scope and boundaries

[HYB-340](https://linear.app/myblueton/issue/HYB-340), machine:mac_mini. Root implemented the next increment of the [reviewed document screen plan](document-screen-plan.md): retained-basis/action-advice/preparation-observation parsers, a gateway requiring matching inspection/advice revisions, and a session-owned controller. The [client foundation record](document-client-verification.md) describes the earlier checkpoint. No React screen consumes this controller yet. The full product goal remains active; this is not a completed B09 journey or owner acceptance.

Inspection validates exact fields, request/workspace/document identity, basis profile, organization/currency, connection, survey and required mappings. Quantities and taxes remain canonical exact decimal values. Lifecycle events require complete metadata, a different confirmer and full nanosecond ordering. A newer inspection can advance an older history row's recorded state without claiming current eligibility. Advice separately validates permissions and readiness. Preparation observation validates ownership, frozen submission, exact totals and receipt consistency. A matching ledger observation with no recorded receipt remains unresolved. Opaque bounded mapping references are historical evidence, not authorization inputs.

The controller retains immutable attempt envelopes and receipts across selection changes while discarding obsolete query results through independent generation counters. All-pending state is synchronous before dispatch. Mutation completion invalidates same-request advisory reads started while the command was in flight. Recorded receipts survive subsequent refresh failure. Session identity includes user, workspace and CSRF; transport verifies the live session before every command. The session registry is bounded and uses no persistent browser storage.

An unsent review freezes explicit packet facts, rationale, revision and HTTP identity. Editing discards only the unsent review. Unknown proposals prevent replacement proposals for the same request/digest after data reloads and same-session navigation. Reloading the browser document discards this in-memory attempt state. Identical retries retain the original envelope and operation reference; a later clean refusal cannot erase an earlier unknown effect. Both submit and retry reject overlapping pending work on the same request.

Fresh-HTTP ERP recovery is explicit and limited to the inspected owned preparation key, original actor, document and digest. Preparation history, inspection and advice must share a revision; conflicting observations or already recorded receipts cannot enter this recovery path. A successful matching recovery retains the original unknown attempt as historical evidence but suppresses obsolete retries and unlocks later governed actions. A new login may discover server history; this does not reconstruct a lost original request envelope or rationale.

# Review and corrections

Independent refusal_review identified two parser blockers: ambiguous mapping-kind/platform concatenation and millisecond-only timestamp ordering. Root added a closed mapping-kind allowlist and exact BigInt nanosecond comparisons, with collision and reversed-nanosecond tests. Corrective parser review passed. The reviewer inspected source/tests without executing tests.

The first store review identified an unknown-proposal replacement path. Root blocked new proposals for an unresolved proposal on the same request/digest and added reload/navigation coverage. The recommended per-request retry lock was also implemented and tested. Corrective source review passed for gateway/controller, all 21 store cases and the actual client runner/Go fixture. The reviewer verified proposal replacement and per-request pending guards, owned recovery and preservation of unknown evidence; that source reviewer did not execute tests or inspect rendered screens.

An initial test-file write used a repository-relative path from the console directory and failed before creating the file. The corrected command created the intended file. Initial controller typecheck found a TypeScript union inference mismatch for action fields; annotating the intent builder's return contract resolved it. Neither failure is counted as a passing check.

# Verification

The parser increment passed 35 focused cases in 218.039ms, all 1035 console tests in 5197.770ms, build/typecheck and actual client interoperability in 1.637s. Logs: /private/tmp/hyb340-document-inspection-tests.log, /private/tmp/hyb340-document-inspection-build.log and /private/tmp/hyb340-document-inspection-runtime-final.log.

Controller tests cover stale context/history/inspection, unsent review immutability, duplicate clicks, navigation during submit, same-request reads during mutation, recorded results after failed refresh, unknown plus later refusal, clean refusal, permissions/readiness, revision mismatches and exact owned recovery. Fresh-HTTP recovery is verified with controlled gateway fixtures, not an actual server uncertainty injection.

The actual Node client runner now uses DocumentStore for ERP preparation. Its fetch wrapper consumes and discards a successful real server response, producing an unknown client result after the effect. After a selection change, the same original intent is retried and the server returns its recorded receipt. The fixture checks exactly one workspace-owned preparation with net 15.00, tax 1.00 and quantity 5.00, no external ERP commands and retained history after document revocation. This run passed in 0.970s. Evidence: /private/tmp/hyb340-document-store-runtime.json; log: /private/tmp/hyb340-document-store-runtime.log. This is real service interoperability after response loss, not a browser test or a fresh-HTTP recovery test.

Final console verification passed all 1056 tests in 4409.834ms; build/typecheck passed with the existing large-chunk advisory. Logs: /private/tmp/hyb340-document-controller-final-tests.log and /private/tmp/hyb340-document-controller-final-build.log. The scoped Go source-rule check passed in 0.488s; log: /private/tmp/hyb340-document-controller-source.log. Independent platform_refusals checked the final test/build/runtime logs and requested the explicit browser-reload limitation now included above; no tests were executed by that reviewer.

# Remaining acceptance

Mount the single controller at B09's scope owner and share it across responsive detail copies. Implement structured form/review, historical facts, current permission/readiness, explicit actions, unresolved attempts and recovery UI. Then run authenticated browser journeys, EN/KO/AR viewport/RTL/keyboard checks, technical-writing review and actual Claude rendered review. Production Go behavior is unchanged; only the existing opt-in integration fixture changed. No full Go regression, additional OS, external ERP effect, durable persistence, final issuance, deployment, merge or owner FIX is claimed for this increment.
