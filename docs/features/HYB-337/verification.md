---
type: VerificationRecord
title: B09 approval read binding verification in progress
description: Actual implementation checks and outstanding acceptance evidence for the phase 6 read increment.
status: draft
---

# Implemented and reviewed boundary

The Go service serves /api/approval-review from modules/approvals, with synthetic module-governed requests, exact decimal strings, current workspace gate checks and session/query/revision-bound HMAC pagination. Unknown validation, dependencies, limits and ERP applicability remain unknown. No financial command is enabled. Unsupported recognition, adjustment, policy and delegation views return unconnected rather than fabricated empty results. The proxy routes B09 to Go, and its environment label describes read-only review rather than an always-unbound producer.

Independent reviewer refusal_review required explicit withdrawn/multiple-decision/expiry semantics, then passed the constrained design. Implementation review required UTF-16 query limits, non-binary-fraction count assertions and an exact decimal import exception only for approvalseed.go composition. Updated source/TW and runner review passed within scope; this is not a final full QA pass or human native-language acceptance. The binding register and glossary map actual fields and constraints.

# Current execution evidence

Paths below are temporary local evidence, not portable committed artifacts. Environment: macOS arm64, Go 1.26.4, Node 25.9.0 and local Chromium through Playwright. Existing user API/preview services were preserved. Browser automation explicitly intercepts /api/** to disposable ticket APIs; frontend 8836 alone is not a standalone isolated configuration.

| Check | Actual outcome |
| --- | --- |
| Focused Go approval/source-rule/access-shape checks | PASS, /private/tmp/hyb337-focused-final.log |
| Authenticated empty-installation HTTP | PASS, /private/tmp/hyb337-empty-http.log |
| Policy-revision/approval-count test strengthening | PASS, /private/tmp/hyb337-policy-tests.log |
| B09 parser and proxy tests | 21 PASS, /private/tmp/hyb337-client-final.log |
| Full client suite | 913 PASS, /private/tmp/hyb337-client-all.log |
| Final console build | PASS with existing bundle-size warning, /private/tmp/hyb337-build-final.log |
| Go vet all packages | PASS, exit 0, /private/tmp/hyb337-vet-all.log |
| Broad server suite | FAIL/TIMEOUT at default 10 minutes in existing TestScaleFilterListsOverHTTP; /private/tmp/hyb337-server-tests.log |
| Race-enabled short suite, all packages | FAIL/TIMEOUT at aggregate 10-minute deadline; four regression expectations corrected below |
| Full non-short race suite | INTERRUPTED after scale bottleneck diagnosis, /private/tmp/hyb337-race-full.log; not PASS |
| Primary browser journey | 9 PASS, /private/tmp/hyb337-render-reviewed/evidence.json |
| Expanded browser lifecycle | PASS, /private/tmp/hyb337-journey-settled/evidence.json |
| Claude rendered review | 35/36 initial expanded captures passed; defective Arabic desktop capture replaced and targeted re-review PASS, below |
| Decision mutations | 10 intended assertion failures after targeted corrections; initial and rerun evidence retained separately |

The broad server run's access-shape failure expected only FOCUS grants. The assertion now verifies both new approval grants/catalog entry and existing FOCUS grants, and the focused rerun passed. The separate timeout occurred in the pre-existing 100,000-user membership scale test, with its identity transition stack recorded. Neither timeout nor a short-suite pass establishes full regression acceptance.

# Browser behavior and rendered evidence

The primary runner activates and signs in a synthetic administrator, receives 25 module-backed rows, programmatically focuses the first row and activates it with Enter, observes detail-heading focus, verifies the exact amount string and checks document overflow. This does not prove keyboard reachability from page entry. It awaits document.fonts.ready, which does not independently prove successful loading of every font.

Current visual evidence is the 36 PNG set at /private/tmp/hyb337-render-reviewed: inbox, detail, amounts and blockers, each in EN/KO/AR at 1440/1024/390. It supersedes the earlier 18-viewport set for the expanded review scope. Claude CLI ui-designer inspected all 36 and found no design or financial-truth blocker, but rejected b09-blockers-ar-1440 because of a blank paint band. Root reproduced the defective image observation. Waiting two animation frames and 100 ms after scrolling produced a clean replacement at /private/tmp/hyb337-ar-recapture/b09-blockers-ar-1440.png. Root inspected it, and actual Claude targeted re-review passed /private/tmp/hyb337-ar-rereview.txt. The other 35 retain their recorded review; the replacement does not invent a full re-review of them.

Reviewed wording is implemented: Exact request count disambiguates counts from money; Period labels the actual target.period field; a middle dot separates synthetic provenance and selection. The known producer_unbound notice is localized separately from the blocked-state badge. All unknown amounts and authority/prerequisite evidence remain unknown. Existing composition/CSS is retained; no new external layout pattern was adopted. Tablet detail distance and shared-header limitations remain nonblocking observations. Owner and human language acceptance are not claimed.

The --journey runner at /private/tmp/hyb337-journey-settled supersedes the older lifecycle evidence. It passed paging, empty search, failed-read row removal, retry, late filter response handling, denied-read row removal and actual sign-out/sign-in as a second actor. It awaits held-route fulfillment completion, then two frames, asserts the expected current row remains and records zero command POSTs. Both route.fulfill calls returned successfully; that does not prove an aborted old document consumed a response. Navigation destroys the old document: this is navigation/session isolation, not an arbitrary in-document session-change race.

Harness corrections were evidence-driven: collapsed full-table controls were excluded from visible row counts; identical fragment navigation was replaced with a fresh document between width cases; sign-out completion is awaited before the next activation navigation. Failure artifacts omit exception messages to avoid activation URL fragments.

# API and mutation scope

API tests cover anonymous/member refusal, current gate revocation and narrow-grant refusal, exact decimals/nulls, 25-row paging, query/session/revision mismatch, tampered cursor, malformed UTF-8 and UTF-16 bounds, module unavailable, unsupported state before filtering, historical approved/rejected projection, supersession, partial-policy rejection and metadata/index disagreement. Authenticated seed-none HTTP tests use explicit test-only grants, prove empty reads and foreign-workspace index exclusion, and keep recognition unavailable. They do not assert default seed-none read grants or a full second-workspace lifecycle.

Mutation experiments copy source into a disposable directory; the working tree and running service binaries are unchanged. The initial results are /private/tmp/hyb337-mutants/results.json. Survivors exposed missing tests for a policy revision with unchanged approval count and a two-approval policy before any decision. Those tests now pass. The initial skip-gate mutation failed compilation because its variable became unused; the corrected mutant preserves the variable. The targeted rerun at /private/tmp/hyb337-mutants-rereview passed its baseline and killed all three. Root inspected every final selected log: each failed the intended TestApprovalRead assertion with no panic/signal/compile error. Combined evidence therefore kills all ten decision mutants without treating the initial compile failure as a kill. Only an intended TestApprovalRead assertion failure counts as a kill; compilation, infrastructure failure and timeout do not. Final file-level coverage and the independently reviewed remaining defensive branches are recorded in the classification section below; overall coverage is not 100%.

# Outstanding before increment acceptance

Finish race and full regression results and coverage reconciliation; complete final source/document review. Inspect actual staged changes before committing and opening a draft PR. No PR, merge or owner screen acceptance is claimed for HYB-337 yet. Cross-stage command binding and HYB-225 producer integration remain unfinished under the active overall goal; this read increment does not complete B09 or phase 6.

Disposable HYB-337 APIs on 8881–8888 were stopped after their completed runs; only their specifically verified process IDs were terminated. An unrelated Docker listener on 8888 was preserved. The ticket Vite process on 8836 remains available for further isolated tests. No current user-runtime B09 binding is claimed.

# Reachable coverage and defensive branches

The final focused run at /private/tmp/hyb337-coverage-final.log passed. Statement coverage in the three new production files is: approvalread.go 81/81 (100%); approvalrows.go 43/45 (95.56%); approvalseed.go 32/39 (82.05%). These are file-level figures, not the full existing server package percentage. No new Go package is introduced, and no 100% overall claim is made.

Independent classification identified reachable missing cases, now tested: invalid view/kind, denied-but-still-present session, both kind filters, unsupported issue-stage request rejected even under a nonmatching filter, superseded previously approved request, and itemized-subject rejection before filtering. The two uncovered projection returns defend against approved/rejected records without a decision or with inconsistent verdict/actor/exclusions. Public module transitions append a decision before ending/approving a request and reject author/requester decisions (modules/approvals/requests.go, covered by TestApprovedSubjectExecutesOnce and TestSeparationOfDuties). Nonempty entries are rejected earlier, so the downstream exclusion defense cannot be reached for supported subjects. Tests do not alter private module state or use unsafe access to manufacture impossible records.

The seven uncovered seed statements are error returns: fixed valid decimal-profile construction, administrator-grant insertion, workspace opening, policy proposal/confirmation, request creation and rejection recording. Supported fresh startup uses a valid constant profile, fresh registry, existing seeded administrators, two policies and 30 requests within fixed limits. Duplicate initialization is tested to stop without appending rows. There is no supported partial-initialization retry or configurable financial seed population. Remaining guards protect failures if these assumptions change; no injected production fault API or artificial wrapper was added merely to raise coverage. Independent reviewer refusal_review recalculated these counts and passed this classification, with no new implementation blocker. This is not a waiver of full regression or proof of unexecuted paths.

# Broad regression reconciliation

The race-short run reached the aggregate ten-minute deadline at TestSeed and reported four affected expectation groups before timeout: the integration actor call-site guard did not recognize approvals.Confirm during seed assembly; the integration client shape lacked the new approval-console grant; ledger replay counters still used the old seed baseline; and the fixed grant-change call-site set omitted seedApprovals. These are recorded failures, not omitted checks. The fixes preserve exact scope: one file/function/receiver/method exception for approvals confirmation, one seed grant-call entry, both exact console grants and a uniform +1 replay baseline with unchanged relative replay effects. Independent refusal_review passed these test changes.

The targeted race rerun passed all four affected groups and the approval tests: /private/tmp/hyb337-regression-fixes.log, 4.271 seconds. The subsequent full non-short race suite was interrupted after the diagnosed dependency below. No scale test was removed, shortened or silently skipped.

## Full regression interruption and diagnosed dependency

Root stopped this branch's unchanged full race run after noninvasive sampling identified repeated full-workspace administrator scans in TestScaleFilterListsOverHTTP ordinary-member reinstatement. The run is INTERRUPTED, not PASS; its cmd package result does not certify the server or remaining suite. The source remained unchanged during the run. [HYB-338](https://linear.app/myblueton/issue/HYB-338) owns the isolated correction: exhaustive enumerated predicate equivalence and independent security review passed, and the full identity race suite passed 54.453 seconds on that separate branch. The unchanged API scale test subsequently passed in 108.744 seconds with HYB-338 and HYB-339 together. The standalone HYB-338 scale run was interrupted, not passed. Integration and a fresh full regression are required before either consumer branch can claim completion.


# Composed candidate and approval reads

This branch now includes HYB-225 and the independently reviewed HYB-338/HYB-339 dependencies. Both read registries and seed steps remain separate. The combined seed baseline is 23. Candidate data retains Cost/ViewCost; approval records retain General/View. Four explicit user/client revocation combinations deny only the removed capability. Both endpoints retain separate seed-none tests, and no candidate-to-approval binding or financial command was added.

Independent corrective source/TW review by refusal_review passed. The reviewer inspected source, not executed tests; suggested detail-endpoint and lock-ownership test strengthening remains a follow-up. Three M03 capabilities have localized labels and exact read phrases, also consumed by P02. Human native-language acceptance remains unverified.

| Composed check | Actual outcome |
| --- | --- |
| Focused approval/candidate/access/integration/source/seed/replay race checks | PASS, 21.719 seconds, /private/tmp/hyb337-composed-focused-3.log |
| Full frontend suite | 916 PASS, /private/tmp/hyb337-composed-console.log |
| Console build | PASS, existing bundle warning, /private/tmp/hyb337-composed-build.log |
| Full race suite | RUNNING, /private/tmp/hyb337-composed-race-full.log; not yet accepted |
| Three-capability rendered matrix | PASS, 18 route cases plus selected forms, /private/tmp/hyb337-composed-renders/evidence.json |
| Three exact request payloads and pending locks | PASS, /private/tmp/hyb337-composed-interactions/evidence.json; injected503 before backend writes |
| Reorder/removal/expiry/ceiling/error-link states | PASS, controlled response fixtures, /private/tmp/hyb337-composed-state/evidence.json |
| Composed B09 rendered matrix | PASS, nine route cases and36 images, /private/tmp/hyb337-composed-b09-renders/evidence.json |
| Actual Claude M03/P02 visual re-review | Scoped PASS on six named images, /private/tmp/hyb337-composed-visual-review.txt |
| Actual Claude B09 composed re-review | Scoped PASS on five named images, /private/tmp/hyb337-composed-b09-visual-review.txt |

The initial composed focused run failed on the inherited 22-step expectation; the exact combined baseline and replay expectations were corrected. Actual rendering uses isolated API 8882 and frontend 8838 with explicit browser route interception; 8838 alone does not establish that isolated API binding.


The Claude review did not inspect all captures, computed CSS or browser behavior. Its Arabic mobile P02 screenshot began partway through the approval card; that specific card wording is outside the six-image visual PASS. Automated body-label assertions cover localized availability, not that cropped card's visual placement. Root inspected the Arabic mobile selected approval form and the composed Arabic desktop B09 blocker capture. Current unknown prerequisite/read-only semantics remain visible. No native-language, assistive-technology or owner acceptance is claimed.

During local rebase conflict resolution, an overbroad conflict-marker expression briefly truncated four files. Formatting rejected the result. Root reconstructed those files directly from the three Git revisions with bounded markers before running the focused checks above. No malformed tree was pushed or treated as verified; the final ticket commit is assembled from the corrected composition.
