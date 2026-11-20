---
type: VerificationRecord
title: Charge approval policy governance verification
description: Actual HYB-340 policy-history, current-authority, impact and rendered verification evidence.
status: draft
---

# Scope and result

The first HYB-340 increment connects synthetic StageCharge policy governance to actual session authorization using the existing approvals registry. It fixes historical B09 reads after policy replacement and adds the separate M01 pending-policy category. The full candidate-backed financial decision workflow remains active and incomplete. No main merge, deployment, real financial approval, ERP effect, bulk revocation execution or owner FIX occurred.

# Executed checks

Host execution was macOS arm64, Go and Node from the existing local environment. Windows/Linux operation and native-language practitioner acceptance were not executed. Temporary evidence paths below are local records, not committed runtime data or portable test fixtures.

| Check | Actual result |
| --- | --- |
| History-focused race after correcting obsolete expected refusals | PASS approvals1.116s, server2.823s; /private/tmp/hyb340-history-race-corrected.log |
| Financial/history/input-authority/guard focused race | PASS server8.035s, approvals1.119s; /private/tmp/hyb340-financial-focused2.log |
| go test -short ./... | PASS all79 packages, exact comparison to go list; server137.150s; /private/tmp/hyb340-all-short.log |
| Supplemental unseeded HTTP and step-up race | PASS1.811s; /private/tmp/hyb340-financial-boundary.log |
| go vet ./... | PASS; /private/tmp/hyb340-vet.log |
| Locked console install | PASS,33 audited packages, zero reported vulnerabilities |
| Console tests | PASS928, no failures/skips; /private/tmp/hyb340-console-test.log |
| Console build | PASS; existing large-bundle/native-loader advisories remain; /private/tmp/hyb340-console-build.log |
| Actual browser M01/B09 | PASS18 cases, no captured page errors; /private/tmp/hyb340-financial-render/evidence.json |

Full-short and vet ran on unchanged runtime implementation. The supplemental boundary test file was added afterward and executed separately. The subsequent Server field grouping change is whitespace/placement only, preserving the same fields and behavior. Documentation edits followed validation.

Focused cases establish exact recorded historical policy, bounded defensive reads, independent actual governors, revoked proposer/console/confirmer, suspended/removed memberships, ended session, immutable reasons, malformed/stale fields, competing same-revision confirmations, output-capacity refusal before effect, sidecar bound, impact failure, hold, no automatic grants and fresh-key replay under current authority. General command checks cover CSRF, revision, idempotency and step-up. History cannot authorize CheckCurrent or Execute. B09 unsupported multi-approval requests still refuse.

Initial financial tests failed on fixture assumptions, not weakened runtime requirements: an absent then unsupported view=all query was corrected to the actual view=history contract; malformed business fields use established422 rather than400. Revoked-session tests originally failed in f.run's prerequisite catalog read; f.runWith sends the command with an explicit revision so the intended refusal is actually exercised. Exact source-guard negatives were corrected to vary one dimension from the valid pointer receiver. Earlier failures remain FAIL evidence, not successful runs.

# Actual rendered journey and independent reviews

The repository runner tools/financial-policy-qa.mjs used isolated API8897 and Vite8896 with explicit browser request routing, synthetic activation and two distinct administrator sessions. It proposed two policies, prepared but did not execute membership suspension, verified M01 count2, independently confirmed one policy, and read B09 history after replacement. Each M01 and B09 path covered EN/KO/AR ×1440/1024/390. Assertions cover count, direction, overflow, exact demonstration amount text and B09 detail keyboard focus. Credentials were ephemeral and omitted from output/artifacts. The owner’s original local API/frontend were preserved; Vite8896 alone does not establish its isolated API binding.

Independent refusal_review passed history/projection and then governance source/security re-review. It inspected logs but executed no tests. It confirmed current-authority revalidation, immutable reasons, pre-effect capacity checks, exact guarded mutation site and no financial grants. Initial missing-coverage and guard-fixture findings were corrected and re-reviewed. Exact refusal assertions in several negative cases remain a nonblocking strengthening opportunity.

Independent platform_refusals performed TW/domain source and actual six-image context review: M01 full and enlarged rows at EN1440, KO1024 and AR390. PASS, no blockers. The preferred labels are Charge approval policy proposals / 청구 승인 정책 제안 / مقترحات سياسة اعتماد الرسوم. This denotes pending policy governance, not charge decisions, pricing policies, grants, payments or cancellation.

Actual Claude CLI ui-designer completed a read-only nine-image visual review and returned scoped PASS with no blocking findings (/private/tmp/hyb340-visual-review.txt). It inspected those three M01 full views, their row captures and three matching B09 history views. It did not review the other six locale/viewport combinations, amount sections, computed borders or human language acceptance. Nonblocking notes concern desktop full-page capture offsets and pre-existing weak secondary status styling. The row captures establish the changed M01 wording; no claim is made that full-page images prove every nested scroll region. The B09 browser fixture supersedes pending demonstration requests. Formerly-approved historical actor/time preservation is established by server tests, not these renders.

# Remaining integration

HYB-340 retains active ownership for separately governed financial Approve permission, stage/currency/amount authority, v7 original-author subjects, fresh input/pricing/target/source revalidation and actual financial decisions. HYB-225 retains durable storage, later stages, protected changes, recognition and unreconciled W-23 screens. The feature stack remains HYB-225 → HYB-337 → HYB-340. A draft PR and review are not authorized merge or ticket completion.

Independent final refusal_review audit passed the contract, glossary, runner and verification claims without executing tests or viewing images. The read-only audit accepted this scoped checkpoint/draft PR only.
