---
type: VerificationRecord
title: License verification-key API
description: Dual-controlled public verification-key lifecycle through installation administration.
status: draft
---

# Behavior

The installation command boundary exposes `licenses.propose_key`, `licenses.propose_retirement`, `licenses.propose_compromise`, `licenses.approve_key_change` and `licenses.withdraw_key_change`. Proposals target `license-installation`; approval and withdrawal target `license-change-N`. N is a canonical positive uint64 decimal, returned as a string to preserve values beyond JavaScript integer precision. Resolve checks syntax and workspace only. Human identity and installation-administration authorization precede durable operations. Existing revision, rationale, replay, CSRF and step-up controls apply.

Self-approval returns 403 self_approval_denied; withdrawal by another administrator returns 403 permission_denied. Successful proposal records a pending request; only a separate administrator's approval applies it. Approval rechecks domain conditions, including authoritative retirement time. Key addition and compromise do not invent an external-time prerequisite. Private-key-shaped values and invalid possession proofs are rejected. A key possession proof establishes possession, not legal issuer authority. This API neither generates nor stores private signing material.

Known-ID reads at `/api/v1/licenses/keys/{key}` and `/api/v1/licenses/key-changes/{number}` require the same installation scope and read admission as license status. Allowlisted fields expose issuer, bounds, retirement/compromise flags and proposed action. Fingerprints are lowercase hexadecimal SHA-256 of the decoded public-key bytes. Raw public key, possession proof and filesystem paths are excluded from responses. There is no list or inventory completeness claim. Missing records return not_found; malformed identifiers in initialized configured scope return invalid_request. A storage failure suppresses read data and returns 503. An uncertain mutation uses the existing unknown-operation response with inspect_operation.

Explicit trust-hold policy now covers all nine license commands. Eight refuse during hold; only withdrawal of a pending proposal is allowed, with all normal guards preserved. Shared approval remains refused because its action may add a key. This repairs an explicit-table omission in the earlier four-command increment without relaxing its prior default refusal behavior.

# Verification

Focused API and hold/source-rule tests pass (`go test ./apps/api/server -run '^(TestLicense|TestHold|TestSourceRules)' -count=1`, 2.688 seconds). Root race run passes in 7.107 seconds; independent reviewer race run passes in 6.087 seconds. `go vet ./apps/api/server ./apps/api/cmd/hb-api` passes. New HTTP scenarios cover two-admin proposal/approval followed by signed commercial registration; self-approval and non-proposer withdrawal denial; exact public fingerprint without raw proof; pending proposal retained after closing/reopening the durable facade; compromise invalidating previously accepted rights; unauthorized and CSRF reads; member mutation denial; canonical target and uint64 bounds; missing records; withdrawal; foreign workspace; rejected submitted material absent from data files; retirement unavailable/past-time checks at approval; valid retirement; unknown mutation storage outcome and unavailable reads; held approval refusal and actual permitted proposer withdrawal. The reopen case in the new tests is a durable-facade reopen; the previous API tests independently cover a full server restart.

Independent design, hold-policy extension and actual-change review: PASS, no blocking findings. New command-specific expired-step-up and generic-envelope replay scenarios are not separately exercised; the unchanged shared boundary and earlier license API tests supply that coverage.

The additional bulk integration preserves one entry per pending proposal in a bounded committed internal read. Bulk impact counts selected proposers only, including multiple requests from one person; approved and withdrawn requests are absent. Storage or initialization failure returns no partial impact and leaves the draft unprepared. The license row is explicitly rechecked:false: suspending a proposer neither automatically withdraws the request nor establishes a domain reauthorization check. The existing EN/KO/AR impact screen maps license_key_changes to a human-readable label; no layout or financial meaning changes.

Final expanded scoped tests pass in 3.064 seconds. Root race checks pass for durable tests (8.703 seconds) and API/hold/bulk integration (7.578 seconds). Independent bulk re-review race checks pass for durable tests (3.656 seconds) and focused API integration (2.100 seconds). The first backend re-review identified the missing display label, corrected in the existing copy map. Three PreviewBody rendered-markup tests pass (312.734 ms), including all three locales with the count and recheck qualification. Console typecheck passes. Independent copy/markup re-review: PASS; browser geometry and native-language acceptance for this label are not certified. Final expanded Go vet passes for licensemode, server and CLI.

# Regression investigation

The earlier full server run timed out in TestScaleFilterListsOverHTTP at the default ten-minute limit. Static inspection identifies repeated membership reinstatement followed by a complete administrator scan in a 100000-person fixture. This is evidence of quadratic work, not a measured performance fix or proof of all timeout causes. Identity source is unchanged. Full-scale completion remains unverified.

The first run excluding TestScale completed in 141.507 seconds with two failures: the existing license test used a source-prohibited numeric type token, and four license commands lacked explicit hold rows. Both are corrected within this increment. No failed run is counted as PASS. A subsequent non-scale run finished in 133.106 seconds with one new failure: pending-proposal inventory omitted the licensing namespace. The fallible store catalog and bulk preparation failure path correct that omission without exempting the namespace from coverage.

Final non-scale server regression: PASS, `go test ./apps/api/server -skip '^TestScale' -count=1`, 135.816 seconds. Only explicitly named TestScale tests are excluded; no full-scale PASS is claimed.

# Limits

The local frontend process was found stopped and restarted with the same integrated-preview configuration. A fresh headless browser journey passed: redirected sign-in, overview/S04 with 1121 authorized rows and six providers, anonymous 401, direct-reader 403, notices 200 and no page errors. This rerun is root-executed.

The running local user API is not restarted or upgraded by this change. External authoritative-time provider selection, issuer provenance and live P07 screen binding remain pending. Synthetic authority and grants are used only in tests. Other OS runtime and physical crash behavior remain unverified. No merge, deployment or ticket completion is implied.

Owner Seung Woo Park; root writer; independent read-only reviewer; machine mac_mini; [HYB-328](https://linear.app/myblueton/issue/HYB-328), PR #242.
