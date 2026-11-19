---
type: Verification
title: HYB-306 combined persistent access-history verification
description: Records reproduced storage and disclosure defects, their independently reviewed repairs, race and mutation evidence, coverage, and multilingual rendered verification of personal access history.
status: draft
sources:
  - resource: ./plan.md
  - resource: ./visual-spec.md
  - resource: ./implementation.md
---

# Scope and result

Candidate: `/private/tmp/hb306`, branch `feat/HYB-306-m05-access-history`, base `73745d6`, machine `mac_mini`. Root owns integration/console and execution; the assigned implementer owns the access-history storage repairs. Independent security/data and QA/TW reviewers were read-only. Their actual-change re-reviews found no remaining blocker in the inspected scope; owner acceptance and production readiness are not implied.

# Reproduced and repaired findings

| Finding | Reproduction and repair |
| --- | --- |
| Unaccepted clock jump could delete files through size cleanup | Startup full-bound test failed when an old file disappeared. Both purge and size deletion now require accepted time; a safe append that cannot fit refuses startup/stops recording rather than deleting protected history. |
| Observed legal hold could be lost between rewrite and retention snapshot | Durable-marker ordering and failed-snapshot tests failed before repair. Markers and damaged-file associations now persist and fsync before destructive operations; crash/reopen test preserves the observed hold. |
| Read/maintenance/sequence synchronization was inconsistent | Added continuous Read/Offer/Incarnation/Tick stress. Snapshot/offer/state lock ownership repaired; no lock held across unrelated file I/O. Full module and server race runs passed. |
| Loss counts disclosed another workspace or predecessor | Live and restarted attributed-loss tests failed before repair. Counts use original scope/source-sequence bounds, suppress predecessor-only ranges and mask uncertain legacy/cross-floor attribution. |
| Delayed refused-sign-in summaries acquired a successor identity | Regression reproduced exact predecessor count five after incarnation. Summary `src` retains original source identity, including a failed write converted to a loss note. Live/restart tests pass. |
| Queued/loaded predecessor rows shifted successor cursor | Test observed `NextW=7` for two successor events. Hidden rows no longer allocate/restore successor ordinals: two/three successor events now produce positions two/three live and after restart. |
| Hold snapshot slices aliased protected registry data | A caller mutated all three returned slices and changed authoritative hold scope. `HoldsCovering` now copies Artifacts/Tenants/Classes; regression failed before and passes under the race detector. Independent actual-diff re-review passed. |
| Eight-hour fixture sometimes lost the oldest journal record | Two failures in ten repeated runs: 28,799 instead of 28,800. Async clock reads exposed a synthetic wall-step interleaving. Fixture advance now takes server-clock then wall-source lock; the existing test passed 50 consecutive runs. Product clock behavior is unchanged. |
| Arabic row date/time presentation | Separate LTR date/time isolates plus removal of Intl-only directional marks. Three locale regressions failed before isolation; Arabic mark assertion failed before normalization; all pass after. Browser character coordinates independently resolve logical/visual `19/11/2026` in all eight normal/notes width cases. |

The plan's repair-decision section records internal `src` and encoded `p` attribution compatibility. No public response field was added for that internal bookkeeping.

# Executed checks

| Check | Observed result |
| --- | --- |
| Final `go test -short ./...` | PASS, exit 0 after storage and fixture repairs; new coverage-only tests also passed in their scoped run |
| `go vet ./...` | PASS on macOS arm64 |
| `GOOS=windows GOARCH=amd64 go vet ./...` | PASS, static compatibility only |
| `GOOS=linux GOARCH=amd64 go vet ./...` | PASS, static compatibility only |
| Full `go test -race ./modules/accesshistory -count=1` | PASS, final worker run 22.345 s; subsequent code is coverage-only tests |
| Final server `go test -short -race ./apps/api/server` | PASS, 416.239 s; no race report |
| New integration safety-case race filter | PASS, 3.054 s |
| Hold snapshot mutation-isolation race case | PASS, 1.604 s |
| Final retention full short/race/coverage run | PASS, 65.716 s, 100.0% reported |
| Final short coverage before hold-copy-only delta | Server 100.0% reported; new accesshistory module 80.2%; retention 100.0% in the final full-module short/race run |
| `npm test` | 811 passed, zero failures, including the final directional-mark assertion |
| Final `npm run build` | PASS, TypeScript and Vite; inherited tooling/chunk warnings remain |
| Final M05 browser run | 86 passed, zero failures; EN/KO/AR at 1440/1280/1024/390, normal/failure/recording notes, pagination failure/retry/focus and late old-period response rejection |
| Safety mutation cases | Eight valid mutants killed; one preliminary invalid mutant corrected and then killed; parent SHA unchanged and private caches removed |
| Docs-format tests / `git diff --check` / file-size limit | PASS; two docs-format tests, maintained changed files below 1,500 lines |

Main comparison with identical short-test flags reports server 100.0% and retention 100.0%. Percentages are rounded statement coverage. One server defensive `ErrStart` fallback is not currently reachable because `accesshistory.Open` outward errors are `ErrConfig` or `ErrLocked`; it was retained, not removed to force a number. The new storage module's 80.2% does not claim exhaustive filesystem-failure coverage.

# Review and evidence

- **Security/data actual-change review: PASS**, including deletion gates, marker durability, synchronization, workspace/incarnation attribution, delayed summaries, cursor ordinals and hold-copy isolation. The reviewer did not execute the worker/root commands.
- **QA/TW: PASS in inspected scope**, including source and fresh EN/KO/AR screenshots. Browser character-position evidence resolved ambiguous screenshot date reading; it is distinct from native-language approval.
- Runtime: macOS arm64, Go 1.25.7, Node 26.7.0, installed Chromium headless 1243. Linux/Windows were statically checked, not executed.
- Synthetic raw evidence: `/private/tmp/hb-combined-qa/306/`; hashes and candidate source fingerprint in `verification-evidence.json`. Earlier failed runs remain distinguished from final passing runs. Mutation records: `/private/tmp/hyb306-mutation-result.json`, `hyb306-mutation-incarnation-result.json`, `hyb306-mutation-cursor-result.json`.

# Limits

Plan residuals R1–R13 stand, including no tamper evidence, bounded crash loss, no administrator access-history view, unobserved in-memory P04 holds, coarse device data and bootstrap incarnation behavior. Fixture-backed note/error browser states verify presentation only; real storage behavior is covered by the module/server tests. Actual Windows/Linux operation, native-language/assistive-technology approval and owner FIX remain unverified. Claude CLI review was unavailable due to its reported usage limit; no separate-agent result is relabelled as Claude review. Artifacts under `/private/tmp` are not guaranteed durable. PR integration/merge read-back and ticket completion are separate stages.
