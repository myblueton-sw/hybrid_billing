---
type: VerificationRecord
title: Current approval prerequisite verification
description: Records non-consuming current approval checks and the remaining W-23 integration work.
status: draft
---

# Scope and result

[HYB-225](https://linear.app/myblueton/issue/HYB-225) adds `Registry.CheckCurrent` as the first prerequisite in the [plan](./plan.md). This verifies the current shared approval rules without consuming an execution key or replaying historical success. It does not authenticate a caller, reserve an execution, prove downstream readiness or complete W-23 integration. The registry's monotonic clock watermark may advance.

# Executed checks

Executed on macOS arm64 with Go 1.26.4 in the isolated ticket worktree. Other operating systems remain unverified.

| Check | Result |
| --- | --- |
| `go test -race ./modules/approvals -count=1` | PASS, 1.846 seconds |
| Same module race run with coverage | PASS, 2.122 seconds; module statement coverage 99.7%; CheckCurrent 100% |
| `go vet ./modules/approvals` | PASS |
| Disposable current-check mutation baseline | PASS |
| Four targeted decision/state mutations | All four killed by intended assertions; no build failures counted as kills |

The tests exercise repeated and concurrent non-consumption, identifiers already used with different execution content, historical success after authority revocation, exact expiry and UTC authority date boundaries, clock rollback, expected subject dimensions, missing and foreign requests, invalid forms, pending/rejected/withdrawn/superseded states, item exclusion and concurrent revocation. Existing Execute still enforces consumption and historical idempotent replay independently.

The portable Python runner copies only required source and Go manifests into a temporary directory. The four mutations skip current checks, write an execution key during a check, bypass checks after historical success and skip workspace/actor validation. Root inspected each assertion failure: eligibility/expiry refusal bypassed, changed records, revoked authority accepted and invalid input mishandled respectively. These four mutations are focused evidence, not exhaustive mutation coverage.

Local executed logs are `/private/tmp/hyb225-check-tests.log`, `/private/tmp/hyb225-check-final.log`, `/private/tmp/hyb225-check-coverage.out` and `/private/tmp/hyb225-current-mutants/`. They are local verification artifacts, not committed product files.

# Independent reviews

The read-only reviewer `/root/refusal_review` independently reviewed the public contract before implementation and returned scoped PASS. The reviewer required ID shape validation without reservation, explicit authentication and downstream-execution limits, and no historical-success bypass.

After implementation the same independent reviewer inspected code and tests and returned scoped PASS with no blocker. The reviewer inspected the final coverage log but did not execute tests; the executed results above were run by root. A further independent read of the mutation runner and verification record found no blocking verification flaw; root corrected the reported mutation-to-failure ordering. The verdict covers this prerequisite only, not HTTP, producer integration, persistence, production policy or complete ticket acceptance.

# Remaining work

Service-owned immutable producer binding, authenticated grants, stage prerequisites and execution boundaries remain open. Adjustments, protected changes, recognition attribution, screens, persistence decisions and independent system QA remain in the original ticket scope. The follow-on integration design is under independent review. No external effect, deployment, merge or owner screen acceptance occurred.

# Internal rating candidate increment

The next implementation adds an internal, currently unwired `chargeCandidates` store. It invokes the real rating compiler/calculator, retains the complete input and computed result in a versioned SHA-256 envelope, and returns fresh decoded copies. Exact input replay works at capacity; changed content under the same workspace/run identity conflicts. Source labels are unverified provenance, and rating exclusions/holds remain visible. There is no approval creation, HTTP command, Server seed or financial effect from this store.

Independent producer review identified the missing assembled charge-run record and prohibited using an approval subject as its own evidence. Concrete design review approved the bounded internal candidate boundary with invalid UTF-8 rejection and a file-specific decimal-import exception. Root incorporated both before implementation. First implementation review found three test gaps (complete-envelope digest, held-rule preservation, computed-output limit) and the need to constrain transient serialization claims. Root added those assertions and a preflight raw-string budget/class count. JSON allocation may exceed the retained byte cap due to escaping and structural overhead; no streaming memory guarantee is claimed.

Executed focused race/source checks: `go test -race ./apps/api/server -run 'TestChargeCandidate|TestSourceRules' -count=1 -coverprofile=...` passed in 2.398 seconds. The passing tests include literal exact charge results, profile provenance, nested alias protection, identity/provenance/policy/input digest changes, cross-workspace isolation, malformed inputs and UTF-8, extreme declarations, conflict/capacity/replay, concurrent differing inputs, excluded and held results, and output overflow that leaves the identity available. `go vet ./apps/api/server ./modules/approvals` and diff whitespace checks passed. Local logs/profile: `/private/tmp/hyb225-candidate-final.log` and `/private/tmp/hyb225-candidate-coverage.out`.

Focused coverage is 0.9% of the entire server package, not full-server verification. Candidate helper function coverage ranges from 80% to 100%; defensive JSON error branches and some validation branches remain uncovered. HYB-337's separate full regression is still running and cannot certify this different branch. Independent corrective re-review by /root/refusal_review returned scoped PASS with no remaining blocker. The reviewer inspected the changed assertions, bounds and final root-run test log; no reviewer test execution is claimed. This accepts only the standalone internal candidate increment, not assembled service behavior or the full ticket.

# Normalized producer to rating integration

The internal adapter now reads actual closed normalization runs and their complete paged rows, the exact source/mapping revision, and actual organization contract/billing records before invoking the candidate store and rating engine. It selects the declared cost basis and typed dimension fields, requires explicit class mapping, preserves full economic row identities and provenance, and refuses incomplete or inconsistent inputs. Candidate envelope schema v2 includes the typed normalized provenance. The producer owns its private normalization/organization registries and mutex; assembly writes must hold that mutex. No registry handles or financial command are exposed via HTTP.

The first test run failed in fixture preparation because a normalization Window requires mapped period fields. Root traced the refusal to normalize.Open and added real start/end fields and source values; the billed-path fixture then passed. Independent change review found missing expected SourceVersion comparison and unsafe JSON hashing of malformed UTF-8 in unselected source text. Root added the exact mapping-key check and validation of every cell before hashing, plus regression cases.

Final executed command: `go test -race ./apps/api/server -run 'TestNormalizedCharges|TestChargeCandidate|TestSourceRules' -count=1`; PASS, 2.524 seconds. `go vet ./apps/api/server ./modules/approvals` and diff whitespace checks passed. Root logs: `/private/tmp/hyb225-normalized-tests.log` (initial failure), `/private/tmp/hyb225-normalized-fixture.log` (root-cause correction), `/private/tmp/hyb225-normalized-final.log` (final passing scope).

Tests use real normalization declarations, mappings, chunks and close operations, and real organization registry writes/reads. Literal outcomes cover billed 5, effective 2, transformed billed 50 and quantity-priced 20; one-row pages preserve two equal locator keys in different documents. Refusals cover missing/null cells, mixed currency, scope/workspace mismatch, absent class/policy/attribution, mapping/source-version mismatch, suspended workspace/profile/party, contract customer/issuer/path mismatch, malformed unselected text and missing quantity binding. Failed captures leave the candidate ID reusable. Changed mappings alter provenance without changing economic IDs. Active legal-entity parent enforcement is separately tested at helper level, not as an internal-chargeback journey.

Read-only reviewer /root/refusal_review completed design, change and corrective reviews and returned scoped PASS after inspecting the final code/tests and root's log. The reviewer did not execute tests. This proves only the tested internal producer-to-rating path. It does not prove intake-manifest linkage, external account ownership, authenticated operator confirmation, effective policy authority, financial approval readiness, production calendars, HTTP assembly or persistence. Whole-UTC-day synthetic profile support is not universal jurisdiction/calendar support. Broader original W-23 scope remains open.
