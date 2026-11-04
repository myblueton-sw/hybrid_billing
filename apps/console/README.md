# React console — HB-67

Corrected implementation increment: baseline six-domain shell and all 112 navigation identities, with thirteen distinct React screen bodies: six entry screens (`entry-start`, `entry-login`, `entry-bootstrap`, `entry-invite`, `entry-challenge`, `entry-scope`) plus `contents`, `index-intake`, `operations`, `core-index-close`, `my-tasks`, `customer-detail` and `review-issue`. Other destinations explicitly show implementation pending. The remaining screens stay in the HB-63 design baseline. No production API, identity, billing calculation or financial operation is connected.

## Install and operate

Use Node.js >=22.12 and npm. From this directory, run each command separately:

```sh
npm ci --ignore-scripts
npm run build
node tools/verify-build.mjs
npm test
npm start
```

Open http://127.0.0.1:8787/#entry-start?lang=ko. The default is service entry with a separate pre-workspace layout; append `&mode=unconnected` to inspect the disconnected boundary. Stop with Ctrl+C. Restart with `npm start`. For source development use `npm run dev`. Installation needs registry access or a populated package cache. A built `dist` uses local assets/fonts and can operate without external requests. Serve it over loopback HTTP or HTTPS; review hashing requires a secure browser context. Do not open index.html through file URLs. Vite preview is a local review server, not a production hosting architecture.

`dist` is generated and replaced on build; keep no manual files there. Preserve exported review evidence before changing browser profiles or clearing storage. Install/build errors must be corrected before serving a stale build. Review save failures are visible and do not establish FIX.

OS-neutral npm/Node entry points use relative paths. Verified on macOS arm64, Node 26.7.0, npm 11.19.0, Chromium 153.0.8010.12. Windows, Linux, Node 22 and other browsers remain unverified targets.

## Structure

- `src/app`: shell, composition validation and navigation wiring.
- `src/modules`: domain-owned pages and synthetic source adapter.
- `src/shared`: small shared contracts, review evidence, localization and styles.
- `src/extensions`: separately rendered contributions to versioned slots.
- `src/profiles`: explicit example installation compositions.

Standard and Evidence share the same intake implementation. Evidence adds the `evidence-guide` renderer. These are public synthetic examples shipped together, not private customer installations or an authorization boundary. Unknown profiles/modules/slots, version mismatch, duplicate IDs and dependency cycles fail before screen rendering. Future private extensions require a controlled build distribution; hiding a module is not security.

## Review and verification

Every implemented screen exposes a review button and stable item IDs. FIX is scoped to the source fingerprint, locale, exact width, profile, mode, visible state and item list. Only an explicit owner acknowledgment writes a local record. This is not authenticated or durable project acceptance. Export and reconcile evidence through the ticket/Git process. Backend integration, other variants and native-language acceptance remain excluded.

```sh
npm run qa -- --playwright-module "path to installed playwright" --chromium "path to chromium executable"
npm run qa:work -- --playwright-module "path to installed playwright" --chromium "path to chromium executable"
npm run qa:entry -- --playwright-module "path to installed playwright" --chromium "path to chromium executable"
```

All runners accept `--url`; the original `qa` runner also accepts `--output`. Playwright/browser are QA prerequisites, not application runtime dependencies. Reports/screenshots go to ignored `qa-output`. Tests use isolated browser storage and never record real owner acceptance.

The header account chooser currently offers two synthetic scopes; the full baseline searchable account workflow and background-job engine are not ported. Profiles/data modes are in separate review tools below the screen.

See [design and scope](../../docs/features/HB-67/design.md), [verification](../../docs/features/HB-67/verification.md) and [notices](THIRD_PARTY_NOTICES.md).

Entry forms validate local examples and clear fields; they do not authenticate, register administrators, accept invitations or send recovery requests. Passwords/tokens are not collected. Explicit preview navigation uses fixed synthetic workspaces. Onboarding/customer landing bodies remain pending; the links do not establish a finished end-to-end service. Editable entry review records expire for the current candidate on remount/reload; raw field values are never stored in review evidence.

## Local service screen integrations

[S03 ingestion](../../docs/features/HB-141/integration.md) receives synthetic FOCUS files into actual local services. [S06 reconciliation](../../docs/features/HB-95/reconciliation.md) compares original and stored evidence through a separate fixed-approver loopback process. Both are explicit local-service modes; other screen candidates remain previews. Neither adapter provides production identity or canonical input confirmation/publication.

[HB-142 planning review](../../docs/features/HB-142/planning-review.md) records reception confirmation, exact detail routing, period/currency continuity and remaining integration gaps.

## Collected FOCUS overview

U01 at `/#/home?lang=ko` now queries the authorized collected FOCUS sample service, with exact currency-separated BilledCost/EffectiveCost, observed billing/subaccount mapping and bounded source rows. It does not fall back to generated data for legacy mode/scenario parameters. Operational ledgers remain explicitly unconnected. Follow the [HB-85 runbook](../../docs/features/HB-85/focus-verification.md) for the separate trusted reader process, ports, source provenance and current storage interruption; static preview alone does not supply this data.

The next collected screen is S04 at `/#/source-data?lang=ko`, using the same reader and pinned source as U01. It exposes normalized rows, mapped fields, native consumption/pricing quantities and explicit unavailable history. The [HB-143 verification/runbook](../../docs/features/HB-143/verification.md) distinguishes exact data, scoped reviews and deferred integrations. The existing `service=local` S04 deep links remain on their separate legacy inspection contract.

B01 at `/#/cost-pools?lang=ko` now opens the infrastructure cost composition candidate. It uses the same protected collected-source reader, defaults to the actual synthetic VMware source and provides source selection, cost categories, whole-row inclusion declarations and an explicitly browser-only review draft. A specific charge-start month is required to save; this is not an approved cost ledger or server policy. See the [HB-101 scope and design](../../docs/features/HB-101/infrastructure-cost.md) and [verification](../../docs/features/HB-101/verification.md). User-facing financial terminology follows the [product glossary](../../docs/ui/terminology/product-glossary.md).
