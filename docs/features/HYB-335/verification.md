---
type: VerificationRecord
title: Local preview autofill verification
description: Records development-only credential injection and sign-in behavior checks.
status: draft
---

# Behavior

The local sign-in screen offers “Fill local sample sign-in details” when all three process variables are explicitly set: `HB_LOCAL_PREVIEW_WORKSPACE`, `HB_LOCAL_PREVIEW_EMAIL`, `HB_LOCAL_PREVIEW_PASSWORD`. Supply only isolated synthetic preview values through the process environment; do not commit values or env files. The helper does not create a sample account: it must match the currently running synthetic API seed.

The control fills the existing fields, clears earlier form errors and focuses Sign in. It does not submit. Normal authentication still decides whether the credentials are valid. The control is disabled during sign-in. It writes no browser storage and preserves password bytes exactly.

Build and preview commands define the configuration as null regardless of environment. Development credential injection requires an explicitly resolved loopback listening host; non-loopback `--host` overrides fail startup. The browser also requires an exact loopback hostname. This helper is not a deployment or authentication feature.

# Executed evidence

- PASS: console typecheck and production build.
- PASS: all 874 console tests; separate reviewer independently ran both configuration-boundary tests.
- PASS: production build with sentinel values in all three preview variables; no sentinel appeared in any production output file.
- PASS: actual development startup with `--host 0.0.0.0` rejected the injected configuration before serving.
- PASS: 66 browser assertions across EN/KO/AR at 1440/390 pixels, including empty initial fields, click/Enter filling, error reset, submit focus, no automatic login request, font readiness, width bounds and successful ordinary login after explicit submit.
- PASS: separate design/change/technical-writing review and inspection of all six rendered variants. The suggested source-revision input coverage was included.
- PASS: staged scope/secret/generated-output inspection and whitespace checks before commit.

Execution: macOS, Node v25.9.0, Chromium 153.0.8010.12, locally installed locked dependencies. Scripts, logs and six screenshots remain generated local evidence under `/private/tmp/hyb335-qa` and `/private/tmp/hyb335-*.log`. No credential values are in this record. Native-language, assistive-technology and Windows/Linux operation are unverified; no owner screen acceptance was submitted.

# Handoff

[HYB-335](https://linear.app/myblueton/issue/HYB-335), owner Seung Woo Park, machine mac_mini. Independent branch from main d7b6a3d; PR review handoff only. No merge or deployment was performed. The user's loopback API and Vite preview remain running for inspection; machine work claim is released at handoff.
