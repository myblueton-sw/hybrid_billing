---
type: VerificationRecord
title: Charge approval request permission verification
description: Separate governed console and human submission permissions with revocation and recovery evidence.
status: draft
---

# Scope and review

HYB-340 continues after a8960f8, machine:mac_mini. Root implemented the closed financialrequestauthority wrapper, domain4 server adapter, six financial_request_permission commands, current governor boundary before replay, hold/step-up inventory and separate M01/M03/P02 terminology. Independent refusal_review accepted the design and reviewed source without running tests. Its requested unknown-result recovery case after console ceiling revocation was added: a previously applied effect can reconcile historically, while an absent effect cannot be newly applied after ceiling loss. Independent platform_refusals reviewed final source copy, contract and browser evidence without edits or test execution and found no blocker. Actual Claude CLI ui-designer completed a read-only visual review with no blocking findings (/private/tmp/hyb340-request-permission-visual-review.txt).

# Executed checks

The complete go test -short ./... run passed 81 packages (server 130.504s; /private/tmp/hyb340-request-permission-all.log). go vet ./... exited successfully (/private/tmp/hyb340-request-permission-vet.log). Financial authority module race checks passed, including the new financialrequestauthority package (1.376s). Focused server race checks passed (4.133s); the final added ceiling-revocation recovery regression passed in the final focused run (4.533s; /private/tmp/hyb340-request-permission-final-race.log). Console tests passed 929 tests; the console build passed with the existing bundle-size advisory. The isolated API binary build passed.

Tests cover separate console and human RequestApproval authorization, absence of default grants and financial decision authority, independent governors and recipient exclusions, current session/governor checks before replay, exact ceiling containment and cross-domain refusal, preflight capacity, fixed wire capability, SeedNone refusal, M03 human revocation, protected console revocation refusal, pending/unknown impact and fail-closed tally, trust hold and step-up, before/after-effect failure and historical reconciliation. Existing input/pricing authority and direct-human decision permission retain separate domains.

# Actual browser journey

The tracked tools/financial-request-permission-qa.mjs ran against isolated synthetic API 8897 and the worktree frontend 8896. It produced /private/tmp/hyb340-request-permission-ui/evidence.json with outcome passed, 27 cases and no captured errors. M01, M03 and P02 were exercised in EN/KO/AR at 1440/1024/390 widths. Assertions checked direction, localized labels and absence of page overflow. Two pending submission-permission proposals appeared in M01. Actual independent console approval and human approval created separate grants. The human grant was then revoked through M03; historical proposal replay left exactly one matching grant with state revoked and effective false. The financialDecision=false evidence field explicitly means that this journey did not make a financial decision.

The browser run and screenshots are not keyboard, computed-border, screen-reader or native-language certification. No new layout was introduced. Owner screen acceptance was not changed. Claude inspected nine route/locale/width variants: M01, M03 and P02 at EN 1440, KO 1024 and AR 390, using context plus pending/grant/ceiling crops (18 images). It found readable wrapping, mirrored Arabic content, full-width layout and no visible decorative accent borders or clipping of the changed elements. The other 18 automated combinations were not visually inspected by Claude. Independent terminology review recommended explicitly naming submission in the new Korean and Arabic M01 label. Root applied 청구 승인 요청 제출 권한 제안 and مقترحات صلاحية تقديم طلب اعتماد الرسوم, preserving existing M03/P02 action names. The final browser runner repeated all 27 cases successfully with no captured errors (/private/tmp/hyb340-request-permission-ui-final/evidence.json). Final console build passed. Actual Claude corrective review passed four changed M01 variants (KO/AR at 1024/390; eight context/row images), with no blockers (/private/tmp/hyb340-request-permission-final-visual-review.txt). Existing P02 English terms were unchanged. Screenshot observations are not computed CSS measurements.

# Remaining boundary

This increment provisions permission to request charge approval. It creates no financial approval request, decision, amount authority or downstream financial effect. Actual request/decision commands, immutable request-to-candidate binding and authorized bounded financial review remain next. Full W-23 remains active. Testing used macOS and isolated synthetic local data; Windows/Linux runtime, durable storage, external ERP and production operation remain unverified. No merge, deployment, real-account/customer-data operation, payment or external message occurred.
