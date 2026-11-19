---
type: Verification
title: HYB-333 combined credential-issue verification
description: Records actual test and browser results, independent reviews and fail-first repairs of credential issue tracking, secret pickup metadata and held-state presentation.
status: draft
sources:
  - resource: ./plan.md
  - resource: ./visual-spec.md
  - resource: ./implementation.md
---

# Scope and result

The current candidate in `/private/tmp/hb333`, branch `fix/HYB-333-m04-unknown-issue`, base `73745d6`, passed scoped security and QA/TW change review after the repairs below. Root owns implementation and test execution on `mac_mini`; independent reviewers were read-only. Owner FIX and native-language approval are not claimed.

# Findings repaired

| Finding | Before | Repair and verification |
| --- | --- | --- |
| Unknown issue/create lost diagnostic metadata | Available stable code/request reference absent from neutral held block | Extracted `Held.tsx`; both forms pass their retained request and error; display only escaped request ID and code, mono/LTR. Six EN/KO/AR render assertions failed before, passed after. No failure alert is substituted for an unknown outcome. |
| Issued read-only fields referenced a missing lead | Held block was idle while inputs described `svc-held-lead` | Issued explanation has `svc-issued-lead`, referenced by those fields. Independent source and fresh-render review passed. |
| Credential ID and pickup timestamp crossed table cells | Character-range assertions failed in six desktop locale/width combinations | Scoped ID wrapping and independently isolated date/time runs. Only Intl-generated directional controls are removed; user strings remain untouched. Final cell-boundary checks passed. |
| Arabic date screenshot reading was ambiguous | Image inspection could not reliably determine numeric order | Browser measured each character's x coordinate: logical and visual values both `19/11/2026`, strictly increasing at visible 1440/1280 pickup dates. Reviewer read the coordinates and resolved the uncertainty. |

# Executed checks

| Check | Observed result |
| --- | --- |
| `go test -short ./...` | PASS, exit 0 |
| `go vet ./...` | PASS, exit 0 |
| `go test -short -race ./modules/identity ./apps/api/server` | PASS; identity 4.283 s, server 361.537 s; no race report |
| Short coverage, identity/server | 99.0% / 100.0% reported |
| `npm test` | 830 passed, zero failures |
| Final `npm run build` | PASS, TypeScript and Vite; existing native-loader/chunk warnings remain |
| Six held render regressions plus store/model/unload suites | 37 passed in the focused run; fail-first held assertions recorded above |
| `tools/service-accounts-qa.mjs`, all 20 scenarios | 114 passed, zero failures, no unexpected page errors |
| Strengthened leave/selector/narrow/in-flight retry scenarios | 42 passed; complete actual issue request bodies remain byte-identical across retries/remounts |
| Final EN/KO/AR render and cell-boundary run, 1440/1280/1024/390 | 41 passed, zero failures; includes character-position evidence for visible Arabic pickup dates |
| Critical pickup mutation cases | Four killed, no invalid/surviving mutants; baseline passed |
| Docs-format tests / `git diff --check` / changed-file size limit | PASS, two format tests; maintained files below 1,500 lines |

The unchanged-build comparison is established by the predecessor HYB-316 behavior and the new tests' missing-feature surface; the resumed repair assertions above have direct failing-before/passing-after evidence. Not every inherited browser guard was rerun on the unchanged predecessor, and no blanket red-first claim is made for them.

# Independent review

- **Security/change: PASS.** Reviewed read-only `identity.Peek`, live-session/current-permission pickup estimates, collection/revoke/retire/sweep, original request/revision/body preservation, fresh-read sequencing, sign-out release and DOM/storage disclosure. Metadata exposes neither secret nor pickup reference nor idempotency key.
- **QA/TW: PASS in scoped source/fresh-render review.** The independent reviewer inspected three languages, representative desktop/mobile renders and the cell/character-position records. It did not execute root's browser runs. KO/AR remain AI drafts without native approval.

# Evidence and limits

Runtime: macOS arm64, Go 1.25.7, Node 26.7.0, installed Chromium headless 1243. Local synthetic browser logs/screenshots: `/private/tmp/hb-combined-qa/333/`. The final all-scenario, retry and render logs are distinguished from earlier failed tool/defect runs in `verification-evidence.json`. Mutation evidence: `/private/tmp/hyb310-333-mutation-result.json`; copied files restored by SHA, parent source unchanged, private cache removed. Temporary evidence is not promised durable.

This adds no credential authentication, rotation, use history or owner transfer. Plan residuals remain. No real account, customer data, message, deployment or financial operation was used. Actual Windows/Linux execution, assistive technology and native-language approval remain unverified. Claude CLI review was unavailable under its reported usage limit; separate-agent review is not relabelled as Claude work. No owner FIX, merged integration or ticket completion is established by this report.
