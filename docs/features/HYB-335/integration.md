---
type: VerificationRecord
title: HYB-335 main integration verification
description: Records current-main integration and verification of the local synthetic sign-in helper.
status: stable
sources:
  - resource: ./plan.md
  - resource: ./verification.md
  - resource: ./integration-evidence.json
---

# Scope

PR240 candidate3e08f7f is consolidated with main f3ddf81 using a normal merge. Root owns integration on machine mac_mini; the sole conflict was the package test command. The resolution preserves every current main test and adds the local-preview configuration tests. The original feature source is unchanged. The owner explicitly authorizes reviewed Git merges; no deployment or real-account authority is implied.

# Executed checks

- PASS: 913 console tests, zero failures, including both local-preview boundary tests and all merged HYB-327/330 checks.
- PASS: TypeScript and production build with three sentinel environment values. No sentinel occurs in any production output file. Existing large-bundle warning remains.
- PASS: actual Vite startup with configured synthetic details and a non-loopback host is rejected before serving.
- PASS: 62 real-entry-point browser checks, zero failures, in EN/KO/AR at1440/1280/1024/390. Isolated synthetic API activation; empty fields, keyboard filling, exact values, cleared errors, submit focus, no automatic login request, local-font readiness and viewport containment; explicit ordinary sign-in succeeds and no unexpected JavaScript error was observed.

The first integration harness checked font status before render-induced font loading finished (52 pass/10 font checks fail); it then reported a route-fetch cleanup error after completing the scenarios. The corrected harness waits for fonts.ready after interactions and handles closing requests. No product change was made to satisfy those checks. Existing feature checks are rerun integration evidence; no new product regression is claimed fail-first on main, which does not contain this helper. The original874-test/66-browser record remains candidate-specific evidence.

# Independent review

Reviewer `/root/hyb310_review`: current integration/security PASS, no actionable blocker. Inspected injection/build/preview boundaries, exact server/browser loopback checks, fill-only behavior, original password bytes, source-revision inputs and preserved test manifest. Independently executed both configuration tests PASS; did not rerun production sentinel or full browser/suite checks. Read-only, no edits.

Reviewer `/root/hyb310_copy_render`: bounded TW/render PASS, no blocker. Inspected signin-en-1440, signin-ko-390 and signin-ar-390 renders, accurate fill-only labels, neutral helper button, available-width/mobile containment, Arabic field direction and distinct submit focus. Read root-run62-PASS browser evidence and source fill handler; did not execute suites/builds or inspect ephemeral credential files.

# Evidence and limits

The manifest binds changed source and selected local logs/screenshots by SHA-256. Raw evidence and the ephemeral synthetic API remain outside Git; the helper stores no browser credentials. Only processes started by this check were stopped. Existing user API/Vite/FOCUS services were not restarted or changed.

Actual host macOS ARM64, Node v26.7.0, installed Chromium headless shell. Linux/Windows, full assistive technology, native-language/practitioner and owner screen acceptance remain UNVERIFIED. No configured remote CI workflow is claimed as passed. Remote merge/main read-back and ticket completion follow these checks and will be recorded only after success.
