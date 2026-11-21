---
type: VerificationRecord
title: Separate ERP result authority and rendered grant labels
description: Fixed-domain permission provisioning with current independent governance and truthful existing-screen labels.
status: draft
---

# Implemented scope

[HYB-341](https://linear.app/myblueton/issue/HYB-341), machine:mac_mini. Root implemented the independently reviewed [result authority plan](result-authority-plan.md). Domain 6 uses the shared grantauthority engine with its own registry/maps, digest version, grant/fault prefixes, resource kind and protected Cost field. Domain 5 document rights remain unchanged. Five explicit capabilities map to propose, confirm, revoke, internal apply and inspect; the four mutation actions are direct-human and nondelegable. Inspection is Cost-classified even when configured metadata attempts to downgrade it.

Six result_authority provisioning commands use current installation governance, independent approval, exact owned console ceilings, actual human recipients, step-up, revision, rationale, CSRF and retained request identity. Current governor checks precede cached receipts. Final grant construction separately enforces exact workspace/action/kind/fields/scope/validity. Unknown grant effects reconcile existing revoked or expired grants as historical receipts without recreating them. Generic grant additions reject result actions/kind/field while existing explicit revocation remains possible. Sample initialization creates the empty registry only; SeedNone has none. No result grants or result evidence are defaulted.

Fallible bulk-impact enumeration includes pending/unresolved result permission proposals, excluding completed grants. New localized M01 labels distinguish them from document permission proposals. Existing M03 and P02 grant rows display the new action and synthetic-result kind; M03 also displays the Cost evidence field. P02 retains its existing action/kind presentation. These are labels for actual issued rights, not new result-provisioning controls or result-consumer UI. The product glossary records seven terms and the internal-apply boundary.

# Independent review

refusal_review passed design before implementation and subsequently reviewed fixed-domain source, current-governor/replay boundaries, exact grant shape, generic bypass, forced Cost metadata and unknown-effect recovery. It also passed the independent expected mappings, cross-domain actual document-ceiling refusal, revised exact Cost/hold assertions and authenticated browser fixture. It inspected logs/JSON but did not execute tests or inspect images.

platform_refusals passed source terminology and representative M01 captures, then eight actual M03/P02 context/apply/inspect captures. It confirmed internally/내부/داخلياً, distinct synthetic result terminology and the P02 field-display limitation. It did not run tests or certify native-language/owner approval.

Actual Claude CLI ui-designer first inspected source and prior M01 renders and recommended preserving the existing composition and shared grant-approval recheck note. It then inspected eight new captures covering M01, M03 and P02, EN/KO/AR including mobile, and reported a bounded PASS with no blocker. Reports: /private/tmp/hyb341-impact-design.txt and /private/tmp/hyb341-rights-render-review.txt. Static rendered review is distinct from browser assertions and server tests. No owner FIX was created. Claude noted the existing P02 action/kind wording difference and inherited unused M03 width as nonblocking; this increment does not certify or redesign that whole-screen layout. The shared recheck note is separately supported by actual proposer-revocation tests and independent source review, not by injected display cases.

# Actual verification

| Check | Result | Evidence |
| --- | --- | --- |
| Initial scoped authority tests | Passed: resultauthority 0.259s, auth 0.271s, server 1.908s | /private/tmp/hyb341-authority-first.log |
| Full auth/gate/shared-authority/document/result module race suites | Passed: 2.137s / 2.854s / 1.290s / 1.297s / 1.299s | /private/tmp/hyb341-authority-modules.log |
| Corrected Cost/hold and cross-domain cases | Passed, 0.832s | /private/tmp/hyb341-authority-corrections.log |
| Scoped server result/document/grant/hold race | Passed, 15.878s | /private/tmp/hyb341-authority-server-race.log |
| Server/auth/gate/resultauthority vet | Passed | /private/tmp/hyb341-authority-vet.log |
| Final console tests | 1057 passed, 43271.823ms | /private/tmp/hyb341-authority-console-final.log |
| Final console typecheck/build | Passed; build 622ms, existing chunk/native-config-loader advisories | /private/tmp/hyb341-authority-build-final.log |
| Authenticated grant/impact browser matrix | Passed, 119.869s; 108 cases, zero runtime errors or bulk execution requests | /private/tmp/hyb341-result-rights-final/evidence.json and corresponding .log |
| Full Go short suite | Passed all 83 packages; server 197.789s | /private/tmp/hyb341-authority-full.log |

The first full server run failed in 149.833s because Cost-preview expectations still named four fields and the explicit hold inventory expected 130 entries with new entries inserted out of sorted order. Root updated the tests to the actual fifth protected field, five added mutations and sorted 135-entry inventory; the denial assertions remain intact. No production permission or hold behavior was relaxed. The initial Vite launch used the repository root without a package script and failed; the corrected apps/console launch succeeded. These failed invocations are not passes.

The opt-in browser fixture creates two pending result-permission proposals, one completed result grant excluded from the pending count, two pending document-permission proposals, and all five distinct console/human capabilities through actual independent-governor HTTP commands. It prepares a removal preview without executing it. Credentials stay in transient environment/session memory; output has fixed error categories, not raw diagnostics. Grant IDs are synthetic non-secret references.

Eighteen M01 cases cover three locales and widths with actual rechecked=true and nine explicitly injected rechecked=false display conditions. Ninety M03/P02 cases cover five capabilities, three locales and widths, and two screens. Checks assert localized labels, direction and no page overflow; M01 also checks local row containment and note weight. The final server check preserves pending count, grant count and zero preparation owners, not exhaustive equality of every grant field. The browser blocks bulk execution and records zero attempts. Actual review inspects selected images, not all 144 image artifacts. Executed environment: macOS and installed Chromium only.

# Remaining original goal

Rights provisioning is a prerequisite, not ERP-result execution. Independently supplied packet provenance, independent evidence confirmation, actual result-consumer checks, ownership-first interrupted Document/Link ingestion, complete coherent reconciliation and StageERPFinal remain required under the original W-23 plan. Dedicated provisioning/result controls and their complete user journeys are also outstanding. No external ERP call, posting, issuance, delivery, production persistence/policy, merge, deployment, native-language certification or owner acceptance occurred.
