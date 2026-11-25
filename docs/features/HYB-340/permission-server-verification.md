---
type: VerificationRecord
title: Direct-human permission server integration verification
description: Actual-session permission governance, independent revocation and scoped M01/M03 rendered review.
status: draft
---

# Scope and findings

This increment follows0d800f5 on HYB-340, machine:mac_mini. Root is the sole writer. It connects the closed financialpermission engine to actual sample-server grants and five authenticated commands. No client grant, module amount limit or financial charge decision is created by permission provisioning. RequestApproval provisioning, original-author candidate subjects, current prerequisite checks and human financial decision transport remain outstanding. The owner services were preserved; QA used a separate process/data directory on8897 and the existing isolated worktree console on8896.

Independent refusal_review found two blocking integration errors: the dedicated kind was missing from the gate catalog, and the handler-only module-governor check could be bypassed by same-key command replay. Both were corrected and independently re-reviewed. Actual HTTP tests now verify grant creation and current module-governor refusal before replay. SeedNone regression then exposed an unsupported error passed to the narrower boundary response mapper; the nil adapter now follows the existing command handler's501 path. Independent review accepted that correction.

Initial tests incorrectly expected synchronous200 revocation and assumed the third sample installation administrator was already a module governor. Corrected fixtures assert the real202 revocation job and actual revoked permission, and explicitly add the third module governor before removing another. These initial failures are not verification passes.

The first full short-suite run found one outdated domain-closure fixture: domain3 is now the intentionally registered financial permission domain. The unknown-domain test now uses0,4,255; the explicit financial shape tests retain the closed-domain assertions. The initial full run failed and is not reported as passing. Independent evidence review also caught a vacuous browser assertion using an absent active property; the final runner requires the named grant to exist with state=revoked and effective=false after replay. The corrected browser journey passed in a fresh isolated activation/data directory; the initial vacuous assertion is not used as evidence.

# Executed checks

Focused race verification passed4.906s, and the final domain-closure/financial permission race recheck passed4.056s (/private/tmp/hyb340-permission-final-race.log): TestFinancialPermission*, generic retained/final grant-add refusal, hold metadata and integration boundary checks (/private/tmp/hyb340-permission-server-race.log). Actual tests cover grant dimensions, direct-human final construction, no new client/amount grants, active human recipient checks, current proposer/confirmer/session/console checks, module-governor-set freshness, recipient exclusions, bounded preflight, same-key governance loss, unknown revoked-effect reconciliation, M03 revocation, historical replay without recreation, failed bulk impact, hold, step-up and SeedNone. Shared engine module recovery coverage remains recorded in permission-verification.md.

Console tests passed929/929 (/private/tmp/hyb340-permission-console.log). Console build passed, retaining the existing large-chunk advisory (/private/tmp/hyb340-permission-console-build.log). go vet ./... passed (/private/tmp/hyb340-permission-server-vet.log). The final go test -short ./... run passed all80 packages, compared exactly against go list ./... (server125.937s; /private/tmp/hyb340-permission-server-all-final.log). No full-suite race or cross-platform operation claim is made.

# Actual browser and independent screen review

tools/financial-permission-qa.mjs passed18 route/locale/width cases: M01 and M03 in EN/KO/AR at1440/1024/390. Synthetic independently authenticated governors proposed and approved an actual subject grant; the browser command journey requested and independently approved M03 revocation, then verified historical replay did not reactivate the grant. M01 showed two pending permission requests separately from amount-limit and policy proposals; M03 showed localized actual grant terms. No page errors or document horizontal overflow were observed. Final journey evidence: /private/tmp/hyb340-permission-ui-final/evidence.json and contextual/row screenshots. This is not a new governance form or financial decision journey.

Before the test-only browser assertion correction, the actual Claude CLI ui-designer directly read EN1440, KO1024 and AR390 context/row images from /private/tmp/hyb340-permission-ui for both changed screens and returned scoped PASS without blockers (/private/tmp/hyb340-permission-visual-review.txt). The other twelve variants have automated evidence, not individual visual sign-off. Root additionally inspected KO1024 M01 and AR390 M03 context images. No layout or style was changed; screenshot inspection showed neutral separators and readable labels. Computed border/font measurements and new keyboard interaction coverage were not claimed.

Independent platform_refusals passed source terminology review in all three languages. The existing distinction between calculation candidates and charge approval remains deliberate; no financial semantics or language-driven jurisdiction/currency change was introduced. M03 ceiling and preview descriptions now scope console requirements to the screen's supported operations, explicitly excluding charge-approval eligibility from the preview. These explanatory paragraphs received source review, not rendered review in the grant-row screenshots. Native-language acceptance, owner FIX and unchanged-screen acceptance remain unverified.

# Completion boundary

Independent code/security review passed the final corrective scope after inspecting the focused race log; it executed no tests. Full W-23 and HYB-340 remain in progress. No main merge, deployment, real-account/customer-data operation, payment or external message occurred. Record the verified checkpoint in the existing draft PR250 and ticket without closing the goal or releasing its active machine.
