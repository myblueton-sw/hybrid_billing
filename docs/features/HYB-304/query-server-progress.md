---
type: ImplementationRecord
title: HYB-304 M05 session query foundation
description: Records the implemented server query/index/detail foundation and the remaining full M05 integration work.
status: stable
sources:
  - resource: ./plan.md
  - resource: ./query-baseline.md
  - resource: ./query-server-evidence.json
---

# Final status

The remaining M05 work listed below was completed later in HYB-304: the screen reads the bounded session, selection and operation routes and `/me` no longer carries the populations ([session UI integration](session-ui-integration.md), [M05 boundaries](m05-boundaries.md)). HYB-304 closed on 2026-11-27 ([closing audit](closing-audit.md)).

# Scope implemented in the ticket branch

Root, machine mac_mini. This is the server foundation of HYB-304 increment A, not completion of M05 or M02–M05. Current UI still uses the old unbounded `/me` payload; this change alone does not satisfy UI-04. No PR merge or ticket Done is claimed.

Authenticated `GET /api/v1/me/sessions` supports current/other/all, the current permitted workspace, signed-in time bounds, bounded search over visible handles/timestamps, page size and continuation. It returns a bounded page, normalized filters, reference time, canonical generation/order, exact or lower-bound matching-record count, scan exhaustion and installation settings provenance. Matching recorded rows can include inactive/unavailable evidence; their count is not an exact active-session total. Fractional bounds preserve nanoseconds; unsupported precision and malformed/duplicate/unknown query input are rejected.

Sessions are indexed by canonical user, identity epoch and workspace. Opening resolves the identity and user epoch after successful module session creation; this indexing does not grant workspace admission. Normal sign-in still accepts the applicable invitation and checks membership/policy before returning credentials. The per-owner ordered source and idle/authentication deadline indexes are maintained under the session mutex; query reads copy only bounded batches. This does not claim constant-time index writes.

Process-local signed cursors bind owner/incarnation/workspace/caller, query and page size, index/domain revisions, reference time and the earliest relevant expiry/lifetime boundary. New pages reauthorize. Changed, expired, malformed or foreign continuations require an explicit fresh query and return the existing409 read-conflict response. Target state uses read-only Peek, preserving other sessions’ idle deadlines. Current-session pinning does not repeat across continuation pages.

Authenticated known-ID detail rechecks the same owner/workspace/epoch independently of the currently displayed page. Foreign and missing targets share the same404 response. The new operations are documented in the existing `x-self` API contract. Page size50, scan1000 and lifetime300seconds are bounded changeable installation defaults exposed through the existing CLI/settings mechanism.

# Findings corrected before integration

- Initial fail-first baseline: three real-handler404 failures; no absent-import or compilation failure counted.
- Intermediate independent source FAIL: query decoding silently dropped malformed filters, idle expiry was labelled revoked, MaxAge equality could escape cursor invalidation, and accepted fractional bounds lost precision. Root added executable failures before correcting each. Ordinary blocked/method/MFA policy failures are distinct from stale authentication.
- New edge tests also exposed use of the generic error helper for cursor conflicts, producing500; corrected to the existing read-conflict helper.
- Initial complete server regression exposed nine invitation/restart/acceptance failures from requiring Peek admission too early during session creation. Three representative unchanged-main tests passed; root moved canonical binding to identity resolution/user epoch and retained admission after invitation acceptance. All nine previous failures and query boundaries passed targeted recheck.
- Known-ID detail and API-contract tests failed before their handlers/declarations were added. Sparse scan, independent lifecycle model, concurrent readers/writes and a171-session bounded query verify source progression and retention of exact/inexact count meaning.

# Independent review provenance

`/root/hyb310_review`, read-only: revised design PASS after target reconstruction/identity/detail and global resource-accounting corrections. Intermediate actual query/index source FAIL was corrected; bounded actual-source re-review PASS covered query/index/detail, canonical opening/binding and authority separation. A separate bounded OpenAPI review PASS checked request/response shapes and error mappings. Reviewer executed no tests, mutations or services in these reviews; all run evidence below belongs to root. Final bounded evidence/test review PASS verified20source/test hashes and30artifact hashes before the last race artifact was appended. Final race execution/result remains root evidence.

# Verification checkpoint

Final foundation server/CLI short regression PASS (including schema and population additions). Darwin vet and Linux/Windows static vet PASS. The later readonly-fixture strengthening changes one assertion target only; final targeted race verification after that change PASS with29test results including subtests and no failures, separately recorded in the manifest. No earlier or narrower pass certifies the entire increment.

Targeted mutation batch:13of14 killed initially. The surviving read-only-target mutation exposed a fixture that checked an older session while the first page actually showed a newer one. Root made the query explicitly select the session whose deadline was measured, observed failure on the known mutant, then killed that mutant in a fresh private-cache batch. All14decision guards are covered by the recorded runs; this is not a claim that all14were rerun after the fixture change. Every runtime source edit was restored byte-for-byte and batch private Go caches were removed. The fixture-only fail-first mutant confirmation used the ordinary test cache; the recorded mutation batch and survivor recheck used disposable private caches.

# Remaining required work

- Replace `/me` full session/expanded-operation downloads with bounded summaries/current metadata and update strict console consumers.
- Implement the approved immutable all-other-session selection, global prepared/accepted reference accounting, reserved-before-effect admission, replay/expiry and cleanup retention.
- Bound operation/result/not-included detail pages and preserve known-ID command eligibility and held/new-session semantics.
- Implement the screen query controls, history/late-response handling, full-set confirmation and independent EN/KO/AR rendered/copy review at four widths; update the actual field binding register/map and glossary with the completed paths.
- Complete required package coverage and scoped mutations/reviews for the entire increment before PR integration. Then continue M04/M03/M02 without closing the full parent prematurely.

Only isolated synthetic test data was used. Existing user API, FOCUS readers/storage, journals, credentials and browser acceptance records were not changed. Actual runtime host is macOS ARM64 with Go1.25.7. Cross-platform static checks are distinct from actual Linux/Windows execution. No deployment, financial action, real account/customer data or external dispatch occurred.
