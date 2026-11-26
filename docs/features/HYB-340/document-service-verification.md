---
type: VerificationRecord
title: Actual document governance and internal ERP preparation commands
description: Source and HTTP evidence for bounded synthetic runtime document service integration.
status: draft
---

# Implemented scope

HYB-340, machine:mac_mini. Root implemented the six commands in [the plan](document-service-plan.md), actual sample-only registry startup and complete bounded evidence projections. Tests use authenticated HTTP command calls, separately provisioned document rights and ordinary Server startup ERP facts. The existing explicit-document fixture provides caller evidence and financial prerequisites; its separate private adapter is not substituted for the runtime adapter. Runtime starts with zero documents, ERP ownership or document grants.

Actual proposal, separate Cost inspection, independent confirmation, exact ledger preparation, historical inspection and hold-safe revocation are exercised. Prepared totals are net 15.00, tax 1.00 and document quantity 5.00, with no external command or financial approval execution. Unknown-after-effect responses retain operation references. Cached HTTP keys remain historical; fresh keys reconcile exact owned intent under current rights. Missing, unavailable and conflicting observations do not update ownership. Tests cover revoked rights before unresolved recovery and before cached replay, console authority loss, ended sessions, self-confirmation, CSRF, revision, step-up, hold, all six unseeded commands, output bounds, maximum future reasons and fallible pending impact.

# Review findings and corrections

Independent refusal_review required plain uncertain-effect errors after potentially committing Prepare, rather than typed clean dependency refusal. The implementation now preserves uncertain ownership after invocation, post-effect receipt failure and owned conflicting/unavailable observations. Historical inspection never invokes reconcile.

Source review found that pre-command UnsupportedCapability lacked an HTTP boundary mapping. Added the established 501/NextNone rule and authenticated tests for all six SeedNone commands. Corrective review passed source and tests without executing them. It recommended valid read envelopes for the two inspections; the final supplemental test uses them.

The first HTTP test run failed 0.738s because assertions expected 409 instead of the existing 422 prerequisite status and used incorrect operation/next field names. Those assertions were corrected to the actual public contract without changing production behavior. Corrected HTTP tests passed 0.758s; additional boundary scenarios passed 2.096s.

The first full short suite failed at server 141.934s. Its failures were expected seed-count and hold-inventory assertions not yet updated for the distinct startup step and four new mutations. Original replay state/row assertions remain intact; only the proven one-step increment and explicit command identities changed. Focused startup/replay/hold/unseeded regression passed 2.538s. The failure log is retained at /private/tmp/hyb340-document-service-full.log.

The second full run failed at server 134.551s in the pre-existing current-financial-approval policy test. Thirty repeated policy cases reproduced three failures; a further diagnostic run observed ten live construction sessions before a replacement policy opened another session. The failure was identity.refusal before the current-charge callback. Frozen fixture timestamps let the existing oldest-session eviction choose the retained consumer at the ten-session limit. approvedCharge now ends its temporary construction sessions before opening the returned consumer. Production session limits, eviction and authority checks are unchanged. Independent refusal_review passed this fixture-only correction and exact seed/hold inventory adjustments. Diagnostic logs are /private/tmp/hyb340-document-service-policy-diagnostic.log and /private/tmp/hyb340-document-service-policy-session-diagnostic.log.

The corrected current-financial-approval race suite passed ten repetitions in 43.379s. Root also reproduced a projection-error mismatch: an inspection returned fields together with a typed size refusal, which the command engine correctly treated as an invalid outcome and returned 500. Both inspection handlers now return an empty outcome on projection failure. The regression injects a lower configured limit after admission and requires 422; it does not establish a public runtime configuration-change feature. Independent refusal_review passed the correction. Initial diagnostic failure 0.602s is retained at /private/tmp/hyb340-document-service-output-diagnostic.log.

# Validation evidence

Financial document/ERP/source-rule scoped race passed 19.385s; server vet passed with no diagnostics. These checks include the HTTP unknown recovery, output bounds and authority scenarios. The race selector did not include TestHoldTableIsExplicitAndComplete; that inventory was verified by the focused regression above. Logs are /private/tmp/hyb340-document-service-race.log, /private/tmp/hyb340-document-service-vet.log and /private/tmp/hyb340-document-service-regression.log. After the session fixture correction, the full short suite passed all 82 packages (server 141.901s) before the final inspection-error correction. On the final production and test code, the full short suite passed all 82 packages (server 133.761s), the document/ERP/source/hold inventory race passed 20.692s, and server vet passed without diagnostics. Final logs are /private/tmp/hyb340-document-service-full-verified.log, /private/tmp/hyb340-document-service-race-final.log and /private/tmp/hyb340-document-service-vet-final.log. The valid-envelope SeedNone supplement passed 0.546s and is also included in the final full run. Independent platform_refusals passed source/document/diagnostic comparison, then read back the final full, race and fixture-regression logs and passed the final evidence review. It did not execute tests.

# Remaining owner goal

This checkpoint connects the public API and ordinary synthetic runtime. The M01 pending-document label and rendered EN/KO/AR review, document/ERP service-screen interaction, further W-23 scope and owner acceptance remain incomplete. No browser run, production connector, external submission/posting, deployment, merge or goal completion is claimed. Only macOS execution is verified. Work continues under the same ticket and machine.
