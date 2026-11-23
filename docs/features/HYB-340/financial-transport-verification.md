---
type: VerificationRecord
title: Candidate-backed charge request and decision transport verification
description: Authenticated synthetic request and human decision effects with current evidence and separate historical review.
status: draft
---

# Scope and independent review

HYB-340 continues after 9ee6c0f on machine:mac_mini. Root implemented financial_charge.request/review/approve/reject, immutable candidate bindings, the closed actual-human approval client exception, pending M01 impact and explicit historical receipts. Independent refusal_review accepted the concrete design after identifying that rating.Result.Total does not establish gross basis. New requests are therefore not added to the existing B09 index. The separately permissioned review command reports rated_candidate_total and current_eligibility=unchecked.

Source review found that initial subject errors were all mapped to stale evidence. The correction preserves dependency/authority unavailable and unsupported/prerequisite failures, while known evidence mismatch stays stale. Binding reads now defensively copy subject maps/entries. Independent corrective source review passed without executing tests. Additional requested tests cover insufficient amount, wrong currency, expired/future limits, concurrent approve/reject and bulk prepare failure without a snapshot. Independent terminology review supplied explicit actual-request labels and requester-permission recheck wording. Independent final source/test review and separate terminology/evidence review passed with no blockers; reviewers did not execute tests. Actual Claude CLI ui-designer also passed its scoped visual review.

# Executed verification

Focused transport/hold/step-up/source/namespace checks passed (1.849s). Initial focused race passed 5.477s; final focused race including amount/currency/date, competing decisions and actual bulk snapshot failure passed 7.605s (/private/tmp/hyb340-transport-final-race.log). go vet ./... passed. Console tests passed 929 tests; console build passed with the existing chunk-size advisory. The isolated API binary built successfully. The complete go test -short ./... run passed all 81 packages, including server 135.029s (/private/tmp/hyb340-transport-all.log).

Actual HTTP tests use governed v7 USD15 capture fixtures and separately provisioned requester, decision permission and module amount limits. They create a pending request, read historical exact amount/author/requester/policy under separate View permission, and record one real module approval. Original capture author and requester cannot approve, including through a different requester. No default submission/decision grants are inferred; submission does not grant review permission. Requests/decisions after source or policy changes refuse; new same-scope requests supersede old ones, and terminal requests cannot be revived with a new key. Same-key approved receipts require current permission; historical review remains explicitly unchecked after permission revocation.

Tests also cover closed reject reasons, reject without amount authority, exact USD15 versus 14.99 limit, wrong currency, expired/future date authority, competing request revisions and competing approve/reject, output/sidecar preflight without superseding the prior request, defensive sidecars, nondisclosing missing/foreign/corrupt bindings, hold/step-up and unchanged other command clients. Actual bulk preparation propagates pending-request tally failure without a snapshot or revision change, then succeeds with count one and rechecked=false. Requester suspension is not automatic financial cancellation.

# Findings and corrections

The first member tests accidentally used a helper that fetched an administrator user list before submitting the command; they failed before reaching the intended boundary. The corrected tests use the supplied current revision with the actual member session. They then exposed a real missing SelfApprovalDenied entry in the server boundary error map: the boundary returned internal_error instead of 403. Adding the exact mapping fixed both original-author and requester refusal tests (0.584s). A later fixture repeatedly opened administrator sessions and evicted its retained decider under the existing session cap; the policy-staleness assertion now opens a current decider session. No session or permission rule was relaxed. An initial build command used a nonexistent command directory; the successful build uses apps/api/cmd/hb-api. These failed attempts are not passing evidence.

# Actual rendered scope

The tracked tools/financial-charge-impact-qa.mjs ran against isolated synthetic API 8897 and worktree frontend 8896. /private/tmp/hyb340-financial-charge-ui/evidence.json reports nine passed EN/KO/AR cases at 1440/1024/390, no captured errors, correct direction, localized actual-request label and explicit requester-permission note, and no page overflow. The actual browser fixture has zero pending financial requests; it does not execute a financial request or decision. Nonzero pending count and financial effects are verified by HTTP tests, not claimed as browser behavior. Existing permission proposals in the fixture remain a separate category.

Actual Claude CLI ui-designer inspected context plus row crops for EN 1440, KO 1024/390 and AR 1024/390 (five variants, ten images) and found no blockers. It observed a separate actual-request row, readable wrapping/RTL, neutral separators and full-width layout. It read the style rule in full and only keyword-matched portions of the transport plan; no full-plan visual review is claimed. The other four automated variants were not visually inspected. Evidence: /private/tmp/hyb340-financial-charge-visual-review.txt. Optional sentence-versus-phrase rhythm was left unchanged to preserve the explicit, independently reviewed permission distinction. Keyboard, screen reader, computed-border and native-language acceptance are unverified; the change adds one row and explicit copy within the existing layout. Owner FIX was not changed.

# Remaining integration

This transport remains in-memory and synthetic. B09 listing/action integration with a truthful amount-basis contract, downstream current checks and execution, ERP/issue/delivery/recognition, persistence and full W-23 remain outstanding. No main merge, deployment, real-account/customer-data operation, payment or external message occurred. Tests ran on macOS; other operating systems are not certified by this result.
