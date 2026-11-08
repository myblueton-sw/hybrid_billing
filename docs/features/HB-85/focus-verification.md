---
type: VerificationReport
title: HB-85 collected FOCUS integration verification and runbook
description: Records actual retained-source reads, independent change review and the storage outage that prevents final end-to-end acceptance.
status: draft
---

# Scope and verdict

Tracking: HB-85, branch `feat/HB-85-commercial-overview`, draft PR77, pending HB-76 dependency. Root implements; separate financial and architecture reviewers inspect source. The [integration design](collected-focus.md) and [composition](commercial-overview.md) are current; [earlier verification](verification.md) describes a different generated-fixture revision.

Implementation and isolated verification pass. The final successfully built candidate is source fingerprint`c43029ac98d7`; App banners and all U01 legacy mode/scenario variants use the collected path without a generated fallback. **The former storage blocker was resolved on 2026-11-04; the complete collected-data journey suite now passes.** Do not represent six successful earlier capture sessions as completion of the later interrupted journey suite. Owner screen FIX, complete U01 planning/business fitness, merge and deployment remain pending.

# Actual retained source read-back

The original `hb-verification` stack retained its volumes, journal and source objects. Its original ports were occupied by a distinct `hb141-integration` stack. Root inspected resolved Compose metadata without publishing credentials, confirmed original image digests/volume identity, and started only the original three services on alternative loopback ports. No initialization, reseed, schema reset or volume deletion was performed; the other stack was left untouched.

The actual `collection.query` and new overview projection succeeded against `reader-company-north`:

| Observation | Result |
| --- | --- |
| Authorized rows | 1,121; two foreign-company rows from the 1,123-row source pack excluded |
| Observed tenant/provider/billing/subaccount combinations | 78 |
| Providers | AWS, Microsoft, Google Cloud, Oracle, VMware, Alibaba Cloud |
| Charge-start months | 2024-03, 2024-09, 2024-10, attributed in tenant timezone |
| Currencies | USD and CNY, never combined |
| Public sample rows | 1,115: original-public 1,000 plus pinned-public-subset 115 |
| Explicitly synthetic authorized rows | 6 |
| USD BilledCost | 27.37875272899 across 1,119 rows |
| USD EffectiveCost | 18.72651418586 across 1,119 rows |
| CNY BilledCost / EffectiveCost | 11 / 11 across two rows |
| FOCUS version / authority | 1.0 / exploratory query-only sample |
| Source slices | Six, all source-finalized; not financial closing |

The all-period default is deliberate: providers do not cover an aligned reporting month. Internal company/tenant membership is synthetic sample authorization, not an observed FOCUS legal entity. Missing native ancestry remains unverified. Original identifiers, downloaded public source and secrets are not committed.

# Actual tests and independent reviews

Host: macOS arm64; Python3.11.7; Node26.7.0; npm11.19.0; Chromium headless shell153.0.8010.12. Other operating systems/browser engines remain UNVERIFIED.

| Check | Actual result |
| --- | --- |
| `python -m pytest tests/focus_overview -q` | PASS, 33 tests; isolated mocks/pure projection, no retained-store reset fixtures |
| `npm test --prefix apps/console` | PASS, 85 tests, including exact18-place values, signed minimum coefficients, calendar gaps, rendered partial/null states and DTO validation |
| `npm run build --prefix apps/console` | PASS, TypeScript/Vite; existing bundle-size advisory remains |
| `node apps/console/tools/verify-build.mjs` | PASS, 438 generated artifact hashes including HTML/seven notices |
| Actual browser capture phase | Six EN/KO/AR variants at1440/390, 1,121 rows,78 account combinations,6 providers, bounded25-row mapping/detail pages, exact cost and review control present; no page errors or viewport overflow |
| Actual journey phase | INCOMPLETE: hostile/missing proxy header, browser actor, forged account, stale basis continuation and EffectiveCost switch passed before the Microsoft-filter query failed during storage outage |
| Final post-review renders/journeys | PASS on2026-11-04: six locale/layout captures and the complete collected query journey suite |

The first capture attempt's English-font assertion failed in a Korean compact view because it checked a family not used by that locale. The runner now checks the actual locale family. The subsequent six captures passed. A later source change moved unknown markers to the zero baseline, corrected table alignment and added normal refresh; those changes still require fresh runtime captures.

Independent financial design review conditionally approved the protected collected path. Actual change review passed backend precision/attribution/identity but initially failed three frontend issues: partial/all-missing display, duplicate minimum-value ticks and omitted visible-page/currency review scope. Root fixed them; the separate reviewer re-reviewed and returned scoped PASS, independently executing four focused numeric/render tests. Full controller fitness was not approved.

Independent architecture/security design and actual-change reviews returned scoped PASS: trusted loopback ports, strict HTTP boundary, narrowed filters, source/selection snapshots, response publication within collection protection, allowlisted DTO and obsolete-response rejection. The reviewer found a search-whitespace echo mismatch; root normalized before request identity and the separate re-review passed. No retained-store runtime access was claimed by either reviewer.

Claude CLI ui-designer actually proposed the collected-data composition using the installed official frontend-design skill and earlier captures. Follow-up visual review of build`4e7573ae3075` passed structure but failed gap-marker placement and text/numeric table alignment. Root corrected their source causes, retained exact digits, adjusted radio alignment and removed doubled separators. This is **not** a final visual PASS; final captures and CLI comparison remain pending.

Generated captures/reports are under `output/validation/HB-85-focus`; they are local, excluded from Git and not promised as durable external artifacts. The interrupted report is not a full-suite PASS. No owner acknowledgement or FIX was created.

# Current storage interruption

During real browser journeys, PostgreSQL closed a protected read unexpectedly, then returned recovery-state connection failures. Subsequent attempts timed out; Docker's metadata endpoint also timed out. These are observed runtime failures, not evidence of absent data or successful recovery. Root did not restart Docker globally, terminate unrelated work or bypass integrity checks. The API returns a sanitized unavailable response and the frontend clears protected figures rather than generating fixtures.

Remaining work: restore responsiveness of the existing retained stack without resetting its namespace; rerun the focused browser suite and post-review captures; obtain final CLI render critique; inspect source/style/PR checks; complete ticket handoff. Preserve the earlier role-fitness findings until authoritative operational ledgers and workflows are implemented.

# Reproduce on the verified authorized environment

Obtain the existing authorized configuration, journal and collected stack; see [six-provider sample lineage](../../ops/installation/six-provider-samples.md), [collection lifecycle](../../ops/installation/focus-collection-lifecycle.md) and [native account hierarchy](../../ops/installation/native-account-hierarchy.md). Their fresh/disposable initialization procedures are not a retained-store outage recovery procedure. `tests/focus_import/conftest.py` resets its namespace; this verification deliberately uses the isolated `tests/focus_overview` directory instead.

From the repository root, using the provisioned Python environment:

```sh
python tools/focus_overview.py --local-operator --actor reader-company-north --port 8806 --ui-port 8805 --pg-port 25432 --ch-port 28123 --s3-port 28333
```

In another terminal, from `apps/console`:

```sh
npm run build
npm start -- --port 8805
```

Open `http://127.0.0.1:8805/#/home?lang=ko`. Vite's dedicated fixed proxy forwards only this overview API to8806 and preserves browser Origin/fetch evidence. The reader actor and storage ports are trusted startup configuration, never browser parameters. Stop each local process with its terminal interrupt; other stacks remain separate.

The portable override is `deploy/local/verification/overview-ports.yaml`; installed Compose2.38.2 was tested. Resolve it with the original Compose project/environment and inspect named volumes/images/ports before starting retained services. Never substitute a fresh project/empty namespace to hide the outage.

Run isolated verification:

```sh
python -m pytest tests/focus_overview -q
npm test --prefix apps/console
npm run build --prefix apps/console
node apps/console/tools/verify-build.mjs
node apps/console/tools/overview-qa.mjs --url http://127.0.0.1:8805/ --playwright-module <installed-module> --chromium <installed-browser> --output output/validation/HB-85-focus
```

The browser runner intentionally expects this documented authorized sample pack. It does not seed data, change permissions, execute financial commands or confirm review. Dependencies are explicit, with fixed loopback host and portable Python/Node browser interfaces. This is a portability design, not Windows/Linux execution evidence.

# Recovery and completed query QA — 2026-11-04

The removed temporary worktree was restored from commit1172307 into a persistent sibling checkout. Existing retained volumes and journal were reused; Docker and the same three services became healthy without namespace initialization or reseeding. Actual source rows/counts/costs were read back again.

Root ran the complete `overview-qa.mjs` suite successfully: six EN/KO/AR1440/390 renders, exact BilledCost/EffectiveCost, provider/month/empty scopes, account-list search invariance, account narrowing, snapshot pages, Back, foreign scope, hostile/missing proxy headers, stale continuation, injected late response, error clearing and retry. The late-response case replays earlier actual protected responses in an isolated browser route, isolating UI response ordering from the deliberately bounded two-request backend; it does not certify backend cancellation. The first resumed runner exposed assertions racing hash navigation; observable URL/query identity and loaded-state matching now replace those early assertions. No financial source behavior was changed.

Claude CLI ui-designer inspected the current KO/EN/AR1440 renders and KO390 account capture and returned scoped visualPASS for all three previous blockers, with no new blockers in that inspected scope. Uninspected mobile chart/source details, native-language and full controller-fitness acceptance remain separate. No ownerFIX was created.

The former outage section above records historical evidence; it is no longer the current runtime status. Windows/Linux and other browser engines remain unverified. Draft PR77 still depends on the unmerged shell work and receives no merge/deployment authorization from this report.
