---
type: Verification
title: HYB-321 flaky test fixes verification
description: Root's fix and measurements for the flaky TestTrustConcurrentActivation and the timing tests on main, with the tokenmeter report not reproduced.
status: stable
tags: [HYB-321, tests, flaky]
sources:
  - resource: ../HYB-297/verification.md
  - resource: ../HYB-299/verification.md
---

# HYB-321 verification (root, 2026-11-18, main d516b1b)

Test-only change in `apps/api/server` (`trust_test.go`, `timing_test.go`, `reauth_test.go`); no product code changed.

## TestTrustConcurrentActivation

- **Cause:** the test read `expected_revision` for both request bodies before the third administrator signed in. A sign-in can move the revision, so both bodies could carry a stale revision and both activations were refused (`activations 0 0`).
- **Fix:** sign in both actors first, build both bodies, assert the revision did not move while preparing, then fire the two activations concurrently. The failure message now includes both answers.
- **Measured:** `go test -race -count=200 -cpu 1,2,8 -run 'TestTrustConcurrentActivation$'`: original 197 failures of 600, fixed 0 of 600; `-count=300` without race: fixed 0 failures.

## TestTimingIndistinguishable and TestReauthenticationTiming

- **Cause (inferred):** the login test measured each case in its own contiguous block of 20 calls, so a load spike during one block skewed that case's median; the reauthentication test always measured the right password before the wrong one.
- **Fix:** both tests now interleave the cases in rotating order and take 41 samples per case; the 1.5× threshold is unchanged, so what the tests prove is unchanged.
- **Measured:** 3 consecutive runs of the two tests pass; under 10 busy CPU loops, 8 runs each of the original and fixed binaries both passed; 3 full runs of the server package with the original and fixed binaries all passed. The original flake (seen during `go test ./...` with browser QA running) was not reproduced, so the fix is supported by reasoning rather than a before/after failure count.

## modules/tokenmeter

Reported once by the HYB-299 implementer without the test name. `go test -count=30 ./modules/tokenmeter` and `go test -race -count=10 -cpu 1,4 ./modules/tokenmeter` passed every run; not reproduced, no change made. Residual: if it recurs, capture the test name.

## Checks

`gofmt` clean, `go vet ./apps/api/server` ok; the server package tests pass.

## Review

Independent change review: PASS. It confirmed the cause from the code (a successful sign-in increments the global revision, `signin.go:286`), that no assertion or threshold was weakened, and that the record is truthful. Residual noted by the reviewer (pre-existing): both bodies share one global expected revision, so the test does not separately prove the epoch-level guard; the module-level test `TestConcurrentActivationsOnOneEpoch` covers that.

