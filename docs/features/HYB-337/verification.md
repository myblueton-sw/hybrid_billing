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

Independent corrective source/TW review by refusal_review passed. The reviewer inspected source, not executed tests; the suggested detail-endpoint and lock-ownership test strengthening was subsequently implemented and its focused race check passed in 2.359 seconds (/private/tmp/hyb337-composed-separation-final.log). Three M03 capabilities have localized labels and exact read phrases, also consumed by P02. Human native-language acceptance remains unverified.

| Composed check | Actual outcome |
| --- | --- |
| Focused approval/candidate/access/integration/source/seed/replay race checks | PASS, 21.719 seconds, /private/tmp/hyb337-composed-focused-3.log |
| Full frontend suite | 916 PASS, /private/tmp/hyb337-composed-console.log |
| Console build | PASS, existing bundle warning, /private/tmp/hyb337-composed-build.log |
| Full race suite | PASS, all 74 packages on snapshot 11cfd10; server 1480.851 seconds, /private/tmp/hyb337-composed-race-full.log |
| Three-capability rendered matrix | PASS, 18 route cases plus selected forms, /private/tmp/hyb337-composed-renders/evidence.json |
| Three exact request payloads and pending locks | PASS, /private/tmp/hyb337-composed-interactions/evidence.json; injected 503 before backend writes |
| Reorder/removal/expiry/ceiling/error-link states | PASS, controlled response fixtures, /private/tmp/hyb337-composed-state/evidence.json |
| Composed B09 rendered matrix | PASS, nine route cases and 36 images, /private/tmp/hyb337-composed-b09-renders/evidence.json |
| Actual Claude M03/P02 visual re-review | Scoped PASS on six named images, /private/tmp/hyb337-composed-visual-review.txt |
| Actual Claude B09 composed re-review | Scoped PASS on five named images, /private/tmp/hyb337-composed-b09-visual-review.txt |

The initial composed focused run failed on the inherited 22-step expectation; the exact combined baseline and replay expectations were corrected. Actual rendering uses isolated API 8882 and frontend 8838 with explicit browser route interception; 8838 alone does not establish that isolated API binding.


The Claude review did not inspect all captures, computed CSS or browser behavior. Its Arabic mobile P02 screenshot began partway through the approval card; that specific card wording is outside the six-image visual PASS. Automated body-label assertions cover localized availability, not that cropped card's visual placement. Root inspected the Arabic mobile selected approval form and the composed Arabic desktop B09 blocker capture. Current unknown prerequisite/read-only semantics remain visible. No native-language, assistive-technology or owner acceptance is claimed.

During local rebase conflict resolution, an overbroad conflict-marker expression briefly truncated four files. Formatting rejected the result. Root reconstructed those files directly from the three Git revisions with bounded markers before running the focused checks above. No malformed tree was pushed or treated as verified; the final ticket commit is assembled from the corrected composition.


After that full-run binary was compiled, only the grant-separation test was strengthened to acquire the domain lock for revocation and assert candidate detail as well as list reads. Production source is unchanged. Its focused rerun passed separately in 2.359 seconds; the full original run subsequently passed all 74 packages. Draft [PR249](https://github.com/myblueton-sw/hybrid_billing/pull/249) is based on HYB-225; no merge or completed ticket is claimed.

Independent final corrective review by refusal_review inspected the test diff, verification record and focused log and returned narrow PASS. It executed no tests and inspected no images in that review. The full snapshot 11cfd10 regression subsequently passed all 74 packages.


# Full composition baseline and next dependency update

The completed full race log contains exactly the same 74 package names as go list ./..., each PASS, and the test processes are terminal. The server result is 1480.851 seconds. This baseline covers production snapshot 11cfd10, with the later test-only strengthening verified separately. It does not certify new parent HYB-225 lifecycle code 8995695.

The next dependency update incorporates the separately reviewed recorded-lifecycle check into candidate capture. Approval records, gates and financial unknowns remain separate; no candidate-to-approval binding is introduced. Verify unchanged combined seed/counters, candidate and B09 exact values, user/client grant isolation and relevant package regression after rebasing. Existing rendered evidence covers unchanged screens; no new visual composition is intended.

## Lifecycle dependency verification

The separately reviewed lifecycle dependency was integrated without console/proxy source changes relative to the rendered 11cfd10 build. Independent refusal_review confirmed separate approval/candidate registries, gates and seed steps. Focused composed race checks passed 59.639 seconds (/private/tmp/hyb337-lifecycle-composed-focused.log). All 74 packages passed short-mode race regression, server 630.288 seconds, exit 0 (/private/tmp/hyb337-lifecycle-composed-short.log), with an exact go list package-set match. The source snapshot at run start was b1a7078; the subsequent rebase onto parent 19a37a6 added only documented date-test/evidence/plan changes, with no production difference. The extra date cases were separately verified in the parent.

Short mode skips or reduces long timing/scale/property workloads. The earlier full 11cfd10 baseline remains distinct; neither run certifies the later uncommitted intake-source work in HYB-225. Financial commands and approval readiness remain unconnected. No merge or owner acceptance is claimed.


## Intake source dependency composition

Parent source increment 64a3de3 is incorporated in composed snapshot bc79ec2. Relative to the earlier composed head 30ec33d, changed files belong exclusively to the parent source assembly, candidate fixtures/provenance and HYB-225 records. Console/proxy source is unchanged, so prior rendered evidence remains scoped to those unchanged screens. No candidate-to-approval link or new financial command is introduced.

Root ran focused composed race checks covering ApprovalRead, ApprovalSeed, CandidateRead, SourceAssembly, SeedReplay, SourceRules and grant/counter test names: PASS 9.655 seconds, /private/tmp/hyb337-source-composed-focused.log. A fresh all-package short-mode race suite on bc79ec2 is running in /private/tmp/hyb337-source-composed-short.log. The parent full race on 64a3de3 is also running separately. Neither pending run is a pass, and earlier full/short results do not certify the new intake implementation.

The source-composed short suite finished with FAIL (server 477.457 seconds): TestIntegrationActorsAndEffectsAreConfined rejected the new internal intake.Publish call in assembleSourceLocked. Other package results do not convert that failure to a pass. The generic integration-effect AST restriction requires an independently reviewed exact intake-only exception; correction and rerun are pending.

The first guard propagation copied the parent's test file and accidentally omitted HYB-337's existing approvalSeed exception. Its focused run correctly failed on seedApprovals.Confirm (3.966 seconds, /private/tmp/hyb337-source-guard-fix.log). Root restored that branch-specific predicate and added only the separately reviewed intake exception. Independent corrective review confirmed the final diff preserves approvalSeed verbatim. The corrected focused race suite passed 5.405 seconds (/private/tmp/hyb337-source-guard-final.log); broader corrected regression remains pending.

Corrected all-package short-mode race regression on 4947638 completed with exit0 and PASS for all74 packages (server464.208 seconds), /private/tmp/hyb337-source-corrected-short.log. Root compared the exact package-name set with go list ./...:74 expected,74 passed, no missing/extra packages or FAIL entries. This supersedes the failed source-composed short attempt as corrected evidence, while retaining that failure record. Short-mode timing/scale/property exclusions still apply; this does not certify new inputreview work in HYB-225.

## Governed input, pricing and authored capture composition

Root locally integrated reviewed parent HYB-225 a7261d3 into the existing HYB-337 feature branch. Five conflict files were combined by exact bounded conflict blocks: server assembly/catalog, the integration source-effect guard, shared access labels/tests and the product glossary. The service retains one financial approvals registry and its separate demonstration index, alongside current input/pricing registries and authored v7 candidates. Approval General/View remains distinct from candidate/input/pricing Cost access. All previous exact seed/publication/confirmation/capability source exceptions remain independently present. B09 stays read-only and does not bind demonstration approval requests to captured candidates.

The first focused composed race failed 20.455s (`hyb337-governed-composed-focused.log`) because the exact sample seed count was still 23. The combined service adds the input-review step and has 24; root corrected both startup expectations, held-lock count 22→23 and each exact ledger replay expectation by one. Assertions and replay distinctions were preserved. Corrected focused race passed 61.862s (`hyb337-governed-composed-focused2.log`), covering approval/candidate/input/pricing/seed/source/hold boundaries and v5/v6 compatibility plus v7 author capture. Independent refusal_review inspected source, corrected counter semantics and that terminal log, returning scoped PASS without executing tests.

Console tests passed 927 (`hyb337-governed-console.log`); production build/typecheck passed (`hyb337-governed-build.log`) with existing bundle-size and future native-config-loader advisories. Vet passed (`hyb337-governed-vet.log`). No new layout, wording or third-party assets were introduced; shared copy combines previously reviewed families.

Actual isolated CLI APIs and Vite/Chromium verified the composed service. B09 passed nine EN/KO/AR × 1440/1024/390 cases with exact USD1234.56789012 demonstration values, detail focus, no page overflow and computed border evidence (`/private/tmp/hyb337-governed-b09/evidence.json`). A separate actual journey passed paging, failures clearing rows, retry, empty state, late-response protection, denial clearing, navigation and sign-out followed by a second actor (`hyb337-governed-b09-journey/evidence.json`). M03/P02 passed 18 locale/viewport cases with distinct approval/candidate capabilities (`hyb337-governed-access/evidence.json`). The inherited pricing runner passed 36 cases plus actual separate-user input confirmation, pricing approval, member capture and authorized Cost read of exact USD15 with matching review/publication linkage (`hyb337-governed-pricing/evidence.json`). Screenshots in that runner precede policy approval. No financial amount approval or bulk revocation was executed.

Root inspected Arabic mobile B09 blockers and Korean tablet M03 approval-capability form. Actual Claude CLI ui-designer read 11 assigned PNGs plus four evidence JSON files and returned scoped visual PASS (`hyb337-governed-visual-review.txt`). It observed explicit read-only/unknown B09 states, exact unrounded English amounts, RTL IDs, neutral separators and distinct approval/candidate labels. Native select labels can truncate, with full helper text visible. That review did not directly see input/pricing Cost labels or Arabic amounts; the subsequent targeted reviews below address pricing labels and Arabic amounts, while input labels remain outside this new visual certification. No full-locale, assistive-technology, other-OS, native-language or owner acceptance is claimed.

Final composed all-package short suite passed with exit0 (`hyb337-governed-all-short.log`), including server138.656s. Root compared exact package names against go list ./...: 79 expected, 79 distinct PASS, no missing/extra packages or FAIL entries. Runtime/test source was unchanged during this final run. Short mode skips or reduces longer timing, scale and property workloads; this is not a new all-package full-race claim. Independent reviewer inspected the log and confirmed 79 passes, without independently executing go list or tests.

The second actual Claude pass inspected 11 additional existing images (`hyb337-governed-visual-followup.txt`). Arabic B09 exact amount/RTL and scoped style passed, but the full-page M01 image omitted the visible cropped policy row and tall grant images were too scaled to read pricing Cost labels. Root therefore extended only the QA runner: select the actual inspection grant by returned ID, assert localized kind/action/evidence inside that row, scroll it into view, wait two animation frames and require complete viewport containment. The runner records both row crops and viewport context; M01 receives the same contextual evidence. No app/CSS changes were made to manufacture a visual pass.

The first expanded capture run passed 36 cases plus the complete capture journey (`hyb337-governed-pricing2/evidence.json`). After independent review strengthened row-local wording and viewport assertions, the exact final runner passed again with exit0 at `hyb337-governed-pricing3/evidence.json`: 36 cases, zero errors, 18 explicitly verified inspection-row contexts and exact USD15 capture. Root inspected the expanded English desktop M01 context and confirmed the pricing policy count 1 appears beside permission-request count 0 in the same surrounding panel. Final targeted Claude re-review passed after inspecting 11 new row/context captures and the final evidence JSON (`hyb337-governed-visual-final.txt`). It directly confirmed policy count 1 in context, readable pricing Cost/evidence labels in EN1440/KO1024/AR390, neutral separators and RTL grant IDs. Other locale/width combinations retain automated assertions only. Existing time-zone wording and off-scope layout observations remain unaccepted follow-up findings, not silently certified fixes. Input Cost grants are provisioned after screenshots, so no new direct visual certification of those labels is claimed. USD15 remains actual HTTP journey evidence, distinct from the B09 demonstration amount images.

Independent final runner/evidence review returned scoped PASS without executing tests or viewing images. Root stopped all six owned isolated QA APIs after their terminal successful journeys; the original owner services were preserved. The integration is a local feature-stack update only; no main/PR merge, production deployment, financial amount approval or owner FIX was performed. The next financial integration should use a separately tracked child branch atop HYB-337, preserving the acyclic HYB-225 → HYB-337 dependency and reusing its one approvals registry.
