---
type: VerificationRecord
title: Governed amount-authority verification
description: Actual validation and independent review of HYB-340 charge approval limits after the policy-governance checkpoint.
status: draft
---

# Increment and limits

This extends07621f0/PR250 with authenticated synthetic StageCharge module amount-authority propose/confirm/inspect/revoke. Limits bind recipient, currency, exact nonnegative amount and inclusive UTC validity. It creates no gate grant, financial charge decision, payment, external ERP effect or second approvals registry. HYB-340 and the overall W-23 goal remain active; direct-human permission and v7 candidate-backed decision integration are still required.

# Executed validation

| Check | Actual evidence |
| --- | --- |
| Initial pure preparation/source-guard/hold tests | PASS approvals0.334s, server0.523s; /private/tmp/hyb340-authority-first.log |
| Initial exact HTTP/revocation tests | PASS server2.053s, approvals1.399s; /private/tmp/hyb340-authority-http.log |
| Expanded race after clock correction | Server PASS3.742s; module fixture failed, corrected separately; /private/tmp/hyb340-authority-focused2.log |
| Corrected module race | PASS1.334s; /private/tmp/hyb340-authority-module-corrected.log |
| go test -short ./... | PASS79/79 packages, exact comparison to go list; server128.976s; /private/tmp/hyb340-authority-all-short.log |
| go vet ./... | PASS; /private/tmp/hyb340-authority-vet.log |
| Console tests/build | PASS929 tests and build; existing bundle/native-loader advisories remain; /private/tmp/hyb340-authority-console-test.log and -build.log |
| Actual isolated browser | PASS9 EN/KO/AR ×1440/1024/390 cases plus confirmation/revocation/replay; /private/tmp/hyb340-authority-render/evidence.json |

The full-short run includes the final runtime changes and pure-preparation clock-call assertion. Source and tests were unchanged throughout that run. No blanket full-race claim is made. Host execution is macOS arm64; Windows/Linux operation was not executed.

Tests cover canonical exact amounts including zero, bounded derived IDs, stale/removed/suspended actors, revoked governor/console authority, ended session, recipient separation, competing governors, immutable reasons, corrupt/missing bindings, sidecar/output capacity, fallible pending impact, SeedNone, hold and step-up. Receipts preserve revocation even when an authorized caller repeats old confirmation. Amount-authority confirmation does not add a gate grant.

An initial competing-governor fixture used a not-yet-activated ordinary member; the fixture now activates that synthetic recipient before testing confirmation concurrency. A module watermark regression initially named nonexistent grant-ann; it now uses setup's actual g-ann. Those initial failures remain FAIL evidence, not successful tests.

# Clock finding and re-review

Independent refusal_review identified a P2 issue: raw server time could disagree with the approvals registry's clamped effective time after rollback. The correction adds AuthorityAt, returning authority and effective time under one registry lock. Fresh confirmation uses its module timestamp, read receipts use AuthorityAt's timestamp, and revocation preserves coherent as-of time. Receipt capacity preflight increased to23 fields. Module and service tests cover future start, inclusive end, expiry then rollback and revocation precedence. PrepareAuthority remains pure and cannot call the clock.

Independent source re-review returned PASS with no unresolved blockers, inspecting the source and corrected module log but executing no tests. Current governor/recipient checks, exact amount representation, immutable intent and absence of AddGrant remain enforced. This is not acceptance of the still-unimplemented direct-human permission bridge or financial decision transport.

# Actual screen and journey evidence

The repository runner tools/financial-authority-qa.mjs used isolated API8897, Vite8896 and explicit browser routing, with three activated synthetic administrator sessions. One governor proposed two limits for a third person; the other independently confirmed one. A governor then revoked that limit; historical confirmation replay continued to report revoked. Exact1234.56789012 limit assertions are command-result evidence, not a rendered money field claim. No bulk membership suspension or financial amount decision occurred.

M01 showed limit proposals2 separately from policy proposals0, input data permissions and pricing categories. All nine locale/width cases asserted count, direction and absence of page overflow. No captured page errors occurred. Activation credentials were ephemeral and absent from output/artifacts; original owner services were preserved. Vite8896 alone is not evidence of the isolated API binding.

Independent platform_refusals TW review directly inspected six images: context and enlarged row at EN1440, KO1024 and AR390. PASS, no blockers; no tests executed or native-language practitioner acceptance. Preferred labels: Charge approval limit proposals / 청구 승인 한도 제안 / مقترحات حدود اعتماد الرسوم. They count pending proposals and do not imply effective approval permission or cancellation.

Actual Claude CLI ui-designer reviewed nine images for the same three variants: context, enlarged row and full-page capture. Scoped PASS, no blockers (/private/tmp/hyb340-authority-visual-review.txt). It relied on context/row captures where desktop full-page nested scrolling looked offset. It did not verify the other six variants visually, keyboard focus, unseen lower-panel states or native-language quality. No new interactive control or focus handling changed. These limits are not owner FIX or global screen acceptance.

# Next integration

Use a separately reviewed immutable direct-human financial permission proposal with deterministic effect reconciliation; module amount authority stays independently required and revocable. Preserve the auth prohibition on Approve console ceilings and the command transport's human-only checks. Then derive v7 subjects with original capture author and current input/pricing/target/source evidence. Continue the original HYB-225 scope; this verified increment does not redefine completion.

Independent final documentation audit found an ambiguous remaining-work phrase in the maintained contract. It now explicitly names direct-human gate.Approve permission grants, distinguishing them from the implemented module amount authority. No test or delivery scope changed.
