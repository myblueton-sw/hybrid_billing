---
type: VerificationRecord
title: B09 candidate-backed request projection verification
description: Actual synthetic request history with exact rated totals, separate actors and explicitly unchecked current eligibility.
status: draft
---

# Scope and independent review

HYB-340 continues after 76b6805, machine:mac_mini. Root implemented a direct bounded union of legacy approval index and service-owned financial bindings, one historical state projector, requester/policy provenance and the rated_candidate_total basis. Independent refusal_review accepted the design before code and passed source review without running tests. It confirmed cross-source duplicate refusal, exact historical policy/state checks, strict old/new paired-field parsing and the opt-in loopback QA boundary. Independent terminology review supplied final translations and found misleading synthetic-example wording, which was corrected in the mixed notice, row tag and help.

Actual Claude CLI ui-designer inspected four earlier baseline renders before implementation and specified actor/policy placement, neutral current-check warning, unchanged layout and explicit synthetic-environment labeling. Final actual Claude visual review passed without blocking findings after inspecting 18 images across six selected state/locale/width variants. No owner screen acceptance was created.

# Executed checks

Initial affected server tests passed 1.600s; new projection/read tests passed 1.921s. Focused race covering projection, existing approval reads and financial transport passed 8.495s (/private/tmp/hyb340-projection-race.log). Parser tests passed 19 tests, including exact old-record compatibility, complete requester/policy pair, rated totals, partial/null/invalid pairs and unknown-basis refusal. Console regression passed 930 tests; final console build and go vet ./... passed. Build retains the existing chunk-size advisory. The full go test -short ./... suite passed across 81 packages (server 138.466s); the opt-in browser case is executed separately, not counted as normal-suite browser coverage.

Actual HTTP and module-backed tests verify mixed gross/rated totals, immutable original author and different requester, exact historical policy and request digest, pending/approved/rejected/superseded states, no inferred current authority/expiry/ERP applicability, duplicate/corrupt binding and combined-cap refusal, foreign-workspace exclusion and snapshot/cursor invalidation after requests/decisions. The read creates no approval or grant and stores no duplicate index.

# Actual rendered verification

The opt-in TestApprovalProjectionRenderedBrowser starts a loopback httptest server using the actual governed chargeWorld fixture. Existing authenticated command paths create service-backed requests, rejected and superseded history. An actual session cookie is passed only through transient child-process environment and removed from the JavaScript environment immediately; it is not written to evidence. The frontend remains the real worktree Vite application on 8896, with API requests forwarded to that isolated server. The fixture uses a fixed synthetic UTC clock and synthetic identities; no owner services/accounts are changed.

The first runner failed because desktop and mobile detail components both exist in the DOM. The unqualified detail locator was ambiguous; it was corrected to visible selectors, matching the existing B09 runner. This was a QA locator failure, not a passing browser run. The corrected run passed 18.563s. After the final row-tag wording correction, the fresh run passed 29.995s (/private/tmp/hyb340-projection-browser-reviewed.log).

Final evidence /private/tmp/hyb340-projection-ui-reviewed/evidence.json reports 20 passed cases and no captured page errors: pending and approved in EN/KO/AR at 1440/1024/390 (18), plus superseded EN 1440 and rejected AR 390. Source responses assert exact state before rendering. Checks include exact USD15, original sample-member author and sample-admin-3 requester, policy label, rated basis, current-check warning, Enter-to-open detail focus, Escape closure, no page overflow and zero computed border widths on the warning. One actual API setup approval transitions the current request before approved renders; the read-only screen does not expose or execute decision controls. The runner records this distinction explicitly.

Waiting for document.fonts.ready is not proof of successful font loading. These checks do not certify canonical font loading, full keyboard accessibility, screen readers or native-language acceptance. Claude inspected detail, facts and amounts images for pending EN 1440, approved KO 390, pending AR 390, approved AR 1024, rejected AR 390 and superseded EN 1440. It saw the warning only in the AR 1024 crop; other language warning visibility and overflow rely on automated checks. It did not execute keyboard checks or inspect all 60 images. Advisory findings were a pre-existing double neutral rule at tablet width and optional hash wrapping. The shorter warning remains paired with explicit unknown fact values and the read-only boundary; it does not assert current eligibility. The rejected actor/time come from the fixture’s actual rejection command, independently of the later setup approval. Acceptance controls were not activated.

# Remaining integration

B09 decision preparation and approval/rejection controls remain separate reviewed work under HYB-340. Historical state and exact amounts are not current authorization or permission to execute. Existing missing FX/ERP producers, downstream execution, persistence and remaining full W-23 stay active. No main merge, deployment, production/customer data, payment or external message occurred.
