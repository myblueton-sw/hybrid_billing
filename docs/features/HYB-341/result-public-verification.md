---
type: VerificationRecord
title: Public result evidence commands and unresolved impact presentation
description: Actual HTTP result governance and recovery with distinct M01 pending and unknown-work categories.
status: draft
---

# Implemented scope

HYB-341, machine:mac_mini, follows the independently reviewed [public plan](result-public-plan.md). Root implements seven result commands through the existing command engine and actual financialERP instance. Proposal preserves independent original JSON; confirmation, revocation and internal application retain the reviewed core boundaries. Dedicated Cost inspection/history expose retained evidence and historical receipts, explicitly leaving current eligibility unchecked. Mutation replies omit source/cost data. Hold, CSRF, current session/rights, step-up, revision and bounded output are enforced.

Before cached confirmation/application outcomes, the boundary rechecks current rights and evidence. Completed applications additionally validate the current coherent ledger snapshot against the historical receipt without writing. Unknown operations remain unknown when only request_id changes; recovery requires a new idempotency_key and current revision while retaining result identity, digest and applying actor. Inspection never settles an unknown operation. Response admission happens before mutation; bounded projection failures return empty outcomes.

The actual default runtime assembly test creates its document and preparation through HTTP before registering, confirming and applying independently specified result rows. Other targeted tests install a separate actual ERP registry into the server; these are not claimed as default assembly evidence. No external ERP commands, approval consumption, issuance or delivery occur.

# M01 scope and independent review

Two fail-closed impact stores separately count pending evidence by original proposer and receipt-empty applications by applying actor. Revoked/superseded evidence does not erase unknown ownership. Enumeration validates records and reverse ownership indexes; missing/corrupt state in supported SeedSample refuses preview rather than reporting zero; SeedNone remains unsupported for result commands and has empty impact tallies. Recorded receipts are excluded from pending work. The public preview test observes both counts as one and an enumeration failure as unavailable.

M01 displays separate EN/KO/AR labels and notes. Proposal confirmation rechecks the proposer's right; recovery rechecks the applying actor/current evidence while preserving possible earlier effects. Existing layout, neutral separators and typography remain. This is an impact-label change, not the result-management screen.

refusal_review passed public source corrections, impact registration and the preview fixture. It found nonempty outcomes on projection errors; root corrected both handler and projection paths. Actual HTTP tests then exposed missing EvidenceStale/PrerequisiteFailed boundary mappings, corrected to the established command status/next-action contract. The reviewer inspected source only. platform_refusals reviewed terminology before implementation and directly inspected twelve context captures across all locales at 1440/390, passing the two rows. It did not execute tests or certify native usage.

Actual Claude CLI ui-designer provided preliminary guidance and a final bounded PASS after inspecting five context images, two row crops and the evidence JSON. Root retained TW-reviewed wording rather than the preliminary owning-operator phrase or optional bold extra line. Claude noted no blocking layout issue; local font loading and keyboard behavior were not established by its image review. Reports: /private/tmp/hyb341-result-public-design.txt and /private/tmp/hyb341-result-public-render-review.txt.

# Executed evidence

| Check | Result | Local evidence |
| --- | --- | --- |
| Initial HTTP journey and three interrupted-write boundaries | Passed, 0.815s | /private/tmp/hyb341-result-public-http.log |
| Projection bounds corrections | Passed, 1.016s | /private/tmp/hyb341-result-public-projection.log |
| Corrected result/source/hold race | Passed, 16.310s | /private/tmp/hyb341-result-public-corrected-race.log |
| Expanded result race | Passed, 17.005s | /private/tmp/hyb341-result-public-expanded-race.log |
| Default runtime HTTP assembly | Passed, 0.489s | /private/tmp/hyb341-result-public-runtime.log |
| Registered impact public preview | Passed, 1.094s | /private/tmp/hyb341-result-impact-integration.log |
| Console tests | 1057 passed, 18474.70025ms | /private/tmp/hyb341-result-public-console.log |
| Console typecheck/build | Passed, build 1.50s; existing large-chunk advisory | /private/tmp/hyb341-result-public-build.log |
| Actual M01 browser matrix | Passed, 28.075s; 18 cases, errors empty, zero bulk execution | /private/tmp/hyb341-result-evidence-render.log and /private/tmp/hyb341-result-evidence-renders/evidence.json |
| Server vet, including final registered impact | Passed | /private/tmp/hyb341-result-public-vet.log and /private/tmp/hyb341-result-public-vet-final.log |
| Final history/envelope/impact race | Passed, 6.642s | /private/tmp/hyb341-result-public-final-edges.log |
| Proposal-store coverage guard after registration | Passed, 0.481s | /private/tmp/hyb341-result-public-store-coverage.log |
| Final full Go short regression | Passed all 83 packages; server 170.203s | /private/tmp/hyb341-result-public-full-final.log |

The browser matrix covers EN/KO/AR at 1440/1024/390, one real pending proposal and one actual interrupted application. Nine false-recheck cases inject only the unknown-application display flag; the proposal false fallback and zero-count new rows are not rendered by this fixture. It verifies wrapping/local clipping, direction, note weight and no page overflow. Readback verifies two counts and grant count, not exhaustive ledger equality. Screenshots are scrolled viewport captures, not whole-page acceptance.

The first full Go run started before impact registration and failed only the proposal-store coverage guard: result_evidence.propose had no registered store in that compiled snapshot (server 251.309s). The required stores were added without weakening the guard; final regression is separate. The first impact fault test reused an already prepared draft and returned permission_denied before enumeration. Root corrected it to stage a new draft; production behavior was unchanged. A browser-fixture generation command initially ran from the console directory and failed to locate root-relative files; the corrected root-directory command generated the actual fixture. These failures are not passes.

# Remaining original goal

Protected result readiness/context, complete dedicated result/provisioning UI journeys, replacement-revision protocol, StageERPFinal, issuance and delivery remain unfinished. Full rendered result-management design and independent QA remain required. No owner FIX, native-language certification, production persistence/policy, external ERP, merge or deployment occurred. All executed environment claims remain local macOS/Chromium only.
