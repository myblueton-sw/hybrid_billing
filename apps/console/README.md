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

B01 at `/#/cost-pools?lang=ko` now opens the infrastructure cost composition candidate. It uses the same protected collected-source reader, defaults to the actual synthetic VMware source and provides source selection, cost categories, whole-row inclusion declarations and an explicitly browser-only review draft. A specific charge-start month is required to save; this is not an approved cost ledger or server policy. See the [HB-101 scope and design](../../docs/features/HB-101/infrastructure-cost.md) and [verification](../../docs/features/HB-101/verification.md). User-facing financial terminology follows the [product glossary](../../docs/ui/terminology/product-glossary.md). Each source row shows its billing account, subaccount, row ordinal in the delivery, sample provenance and full row ID, and row amounts distinguish Missing from Exact zero with the S04 wording; see the [HB-153 plan](../../docs/features/HB-153/plan.md) and [verification](../../docs/features/HB-153/verification.md).

B03 at `/#/allocations?lang=ko` provides a source-backed allocation review editor. The [HB-103 design](../../docs/features/HB-103/allocation-policies.md) and [verification](../../docs/features/HB-103/verification.md) define whole-source denominators, provisional exact arithmetic, browser-only metadata drafts and session comparisons. These do not activate approved allocation, legal-organization ownership, production rounding or server history.

## B04 sales pricing policies

The `/rates` screen uses an explicitly selected, freshly authorized collected FOCUS row for a manual unit-price arithmetic review. It requires PricingQuantity/PricingUnit and keeps source provider prices separate from selling inputs. Five views cover the rule, ordered calculation, unconnected contract inheritance, separate target adjustment and session comparison. Browser save retains explicitly entered financial parameters; it is not private policy storage or a financial command. See [HB-104 design](../../docs/features/HB-104/sales-pricing.md) and [verification](../../docs/features/HB-104/verification.md).

Run `node tools/sales-pricing-qa.mjs --url http://127.0.0.1:8805/ --playwright-module <installed-module> --chromium <browser-executable> --output <local-evidence-directory>` from this directory against the already running collected-source reader. No retained namespace reset is needed. Portable runtime design does not certify untested operating systems. Source row values use the S04 source-field terms and value states; see the [HB-157 plan](../../docs/features/HB-157/plan.md) and [verification](../../docs/features/HB-157/verification.md).

## B05 FX and rounding policies

The `/exchange-rates` screen reviews conversion of one authorized collected source-cost row. Manual rate and quotation unit preserve direction, explicit markup and a target-currency flat fee precede one selected rounding increment/method, and six signed rounding modes use exact rational arithmetic. UTC dates/maximum age and the unconverted-source choice are user declarations, not a verified rate feed or approved eligibility. The local browser draft intentionally retains entered financial parameters. Server history, contract precedence and financial approval remain unconnected. See [HB-105 design](../../docs/features/HB-105/fx-rounding.md) and [verification](../../docs/features/HB-105/verification.md).

From this directory, run `node tools/fx-rounding-qa.mjs --url http://127.0.0.1:8805/ --playwright-module <installed-module> --chromium <browser-executable> --output <local-evidence-directory>` against the existing reader. Do not reset retained stores. Portable browser/runtime interfaces are used; other operating systems and browser engines require actual verification. Source row values use the S04 source-field terms and value states through the shared components in `src/modules/sources/collected-focus/SourceValues.tsx`; see the [HB-158 plan](../../docs/features/HB-158/plan.md) and [verification](../../docs/features/HB-158/verification.md).

## B06 commitment costs and benefits

The `/commitments` screen reads observed commitment identifiers from `/api/focus-commitments`. It shows separate full-selection Billed/Effective costs, source Used/Unused/unknown classifications, bounded group/account/charge pages and raw field-state evidence. It never infers savings, utilization percentages, legal ownership, unused burden or completed lifetime reconciliation. Provider benefits, internal allocation and customer discounts remain separate; purchase/optimization/financial activation are unconnected. See [HB-106 design](../../docs/features/HB-106/commitments.md) and [verification](../../docs/features/HB-106/verification.md).

The new endpoint uses the existing trusted local collected-reader startup (`tools/focus_overview.py`) and authorization/configuration prerequisites. It adds no data import/schema migration. Run `node tools/commitment-benefits-qa.mjs --url http://127.0.0.1:8805/ --playwright-module <installed-module> --chromium <browser-executable> --output <local-evidence-directory>` from this directory. The exact isolated backend command is `python -m pytest -q tests/focus_commitments/test_commitments.py tests/focus_overview/test_projection.py tests/focus_overview/test_http.py` from the repository root in the prepared backend test environment. Do not run retained-source reset suites. No cross-OS or production-throughput claim follows from these commands.

## B07 Marketplace source evidence

The `/marketplace` screen reads the protected `/api/focus-marketplace` projection. It identifies candidate source rows only when mapped issuer and publisher names are present and differ, with full-selection per-currency Billed/Effective amounts before pagination. Equality is no observed name difference; missing or unmapped names remain unknown. Source row references and SKU identifiers do not establish transaction, offer, agreement, legal responsibility, benefit eligibility or customer receivables. Seller revenue, fees, refunds, taxes, withholding, net settlement and disbursement remain separate unconnected evidence. See [HB-107 design](../../docs/features/HB-107/marketplace.md) and [verification](../../docs/features/HB-107/verification.md).

Use the existing trusted local collected-reader startup and environment prerequisites; this screen adds no import or schema migration. From this directory run `node tools/marketplace-qa.mjs --url http://127.0.0.1:8805/ --playwright-module <installed-module> --chromium <browser-executable> --output <local-evidence-directory>`. From the root in the prepared Python environment run `python -m pytest -q tests/focus_marketplace/test_marketplace.py tests/focus_overview/test_projection.py tests/focus_overview/test_http.py tests/focus_commitments/test_commitments.py`. Do not run retained-source reset suites. Portable interfaces are design targets; other OS/browser engines and production capacity require actual verification.

## B08 billing preparation and source comparison

The `/billing-runs` screen reads actual protected collected FOCUS input evidence through the existing overview endpoint. It supplies bounded source/account pages, full-selection per-currency costs and a single explicitly kept in-memory source baseline. Comparison requires the identical protected source reference and authorization/scope identity; changed references, provider/month/account changes, errors and leaving the screen discard it. Missing/partial amounts never produce a zero difference. The baseline is not persisted, an approved policy, frozen target manifest, calculation lock or billing result. The financial run engine/history, claims, approval and ERP recognition producers remain unconnected; no demonstration runs or success counts are displayed. See [HB-109 design](../../docs/features/HB-109/billing-runs.md) and [verification](../../docs/features/HB-109/verification.md). Source row values use the S04 source-field terms and value states (Missing, Exact zero, charge and billing period in UTC, delivery lineage, sample provenance); see the [HB-154 plan](../../docs/features/HB-154/plan.md) and [verification](../../docs/features/HB-154/verification.md).

From this directory run `node tools/billing-runs-qa.mjs --url http://127.0.0.1:8805/ --playwright-module <installed-module> --chromium <browser-executable> --output <local-evidence-directory>` against the prepared protected local reader. The screen adds no backend route, source import, schema or financial mutation. Portable Node/browser interfaces are used; other operating systems and browser engines require actual verification.

## B09 approval inbox and result evidence

`/approvals` now has a dedicated read-only request inbox and result card. The normal view reports the approval service as unconnected and the request count as unknown. No approval producer exists in this increment; the proposed same-origin GET `/api/approval-review` returns 404 through the existing local reader. Live payloads fail closed until a separately reviewed producer binding exists. No approve, reject, delegation, handoff or ERP command is implemented.

Isolated browser QA intercepts supply explicitly labelled synthetic records to exercise filters, bounded pages, actual-actor decision views and per-currency/basis exact decimal evidence. These fixtures never populate the normal screen. See [HB-110 design](../../docs/features/HB-110/approvals.md) and [verification](../../docs/features/HB-110/verification.md). Run `node tools/approval-inbox-qa.mjs --url http://127.0.0.1:8805/ --playwright-module <installed-module> --chromium <browser-executable> --output <local-evidence-directory>` against the prepared local reader. The runner does not create owner screen acceptance or financial writes.

## B10 billing application history

`/claims` has a dedicated read-only application-history screen. Coverage protection and source references come before separate per-currency/basis amounts. Partial applications, original-linked corrections and ERP order/prepayment evidence remain distinct; money credits and quantity reversals never authorize rebilling. No geometry, availability subtraction or financial command is implemented.

The proposed GET `/api/claim-history` has no bound producer. The prepared local reader returns404; normal count remains unknown and no fixtures load. Live responses fail closed until separate producer review. Isolated QA intercepts provide always-labelled synthetic evidence only. Run `node tools/claim-history-qa.mjs --url <prepared-local-frontend> --output <local-evidence-directory> --playwright-module <installed-module> --chromium <browser-executable>` from this directory. See [HB-111 design](../../docs/features/HB-111/claims.md) and [verification](../../docs/features/HB-111/verification.md). This adds no backend, data import/reset, ledger mutation, export or owner acceptance.

## B11 credits and billing inquiries

`/credits-cases` separates credit balances, reservations, billing inquiries, original-linked adjustments and payment-term evidence. Authority/as-of/freshness precede source values; stale balances remain historical observations. Internal notes and recorded public replies are separate read-only disclosures. Consent, holds, due dates, receipts and source overdue state retain separate evidence; no inquiry, reply, credit, receipt, hold or financial command is executed.

The proposed GET `/api/credit-case-review` has no bound producer. Actual normal service returns404/unknown counts and no fixtures; live payloads fail closed pending separate producer/authorization/disclosure review. QA uses always-labelled isolated synthetic interception only. From this directory run `node tools/credits-cases-qa.mjs --url <prepared-local-frontend> --output <local-evidence-directory> --playwright-module <installed-module> --chromium <browser-executable>`. See [HB-112 design](../../docs/features/HB-112/credits-cases.md) and [verification](../../docs/features/HB-112/verification.md). No balance conservation, date calculation, public-message transmission, source import/reset or owner acceptance is certified.

## HB-144 remaining32 dedicated frontend candidates

The remaining32 canonical screens now have dedicated identity, organization, work, source, financial delivery, reporting, administration and customer-portal surfaces. They use actual module `pages` paths; genuinely shared local form/evidence/table primitives live under `shared/domain-screen-ui`. Explicit group wiring remains in `app/screen-groups`. Existing17 leaf candidates and the Marketplace hold remain separate.

At the HB-144 base, these32 accept no new producer responses. HB-145 binds S03 as described below; the other31 remain unchanged. Unknown values remain unknown; local input review, clearing, scope changes, document/category composition and navigation never create accounts, policies, financial results, messages or jobs. The explicit existing `/imports?service=local` integration remains available separately. Owner review uses an unverified requested context and value-free edit/action revision; implementation and tests never create owner FIX.

From this directory run `npm test`, `npm run build`, `node tools/verify-build.mjs`, then `npm start`. Run the portable browser matrix with `node tools/remaining-qa.mjs --url <local-frontend> --output <local-output> --playwright-module <installed-module> --chromium <browser-executable>`. It checks146 canonical views across KO/EN/AR and desktop/mobile without fixture toggles, external calls or non-GET commands. See [scope and planning trace](../../docs/features/HB-144/implementation-plan.md), [per-screen completion](../../docs/features/HB-144/completion.json), [relocation evidence](../../docs/features/HB-144/file-relocation.json) and [actual verification](../../docs/features/HB-144/verification.md). Backend binding, full workflows, real authorization, native-language acceptance, other OS/browser execution and owner FIX remain pending.

## S03 retained collection receipt metadata

Default `/#/imports?lang=ko` reads fixed GET `/api/focus-receipts` through the prepared trusted-local reader. It provides permitted receipt metadata, canonical detail, identifier-prefix filtering, snapshot/permission-pinned25-record pages and refresh. Quarantine excludes unowned rejected receipts entirely. Registered completeness expectations may be unavailable; OCR/extraction has no producer. Historical publication, declared rows and explicit all-domain zero attestations do not establish financial approval, current source availability or whole-CSP completeness.

The current development proxy maps this endpoint to loopback8810; older source APIs retain their existing8806 mapping. For the prepared retained public/synthetic environment, start the existing `tools/focus_overview.py` reader with `--local-operator --actor <verified-reader> --port 8810 --ui-port 8808` and the verified existing storage port arguments. Reuse its authorized configuration and control directory; do not initialize, seed, reset or import. The [HB-145 design](../../docs/features/HB-145/receipt-binding.md), [backend record](../../docs/features/HB-145/backend.md) and [integrated verification](../../docs/features/HB-145/verification.md) describe the protection boundary and exact prerequisites/limits. A static frontend alone does not supply receipt evidence.

From this directory run `npm test`, `npm run build`, `node tools/verify-build.mjs`, then `node tools/receipt-reader-qa.mjs --url <prepared-local-frontend> --output <local-evidence-directory> --playwright-module <installed-module> --chromium <browser-executable>`. The runner separates actual retained reads from visibly labelled isolated transport fixtures, issues no non-GET commands and does not confirm owner review. Legacy `service=local` remains a separate existing flow. Native Windows/Linux, production authentication and write durability are not certified by this increment.

## S01 retained connection registrations

Default `/connections` now reads fixed GET `/api/focus-connections` from the same protected retained metadata reader as S03. Only fully owned registered connections contribute rows, counts, search or public pins. A registration does not establish external authentication, diagnostics, provider conformance or billing readiness; legacy deliveries never imply a registration. Security and diagnostics remain explicitly unsupported with unknown totals. The explicit `service=local` and existing object routes are preserved separately.

The existing loopback8810 proxy also serves this endpoint. Start the prepared reader with the verified existing storage ports and the actual frontend's `--ui-port`; do not initialize, reset, seed or import to populate the screen. The environment verified by HB-147 has0 attributable registrations and2 legacy deliveries, which are separate observations. See the [contract](../../docs/features/HB-147/connections.md), [backend record](../../docs/features/HB-147/backend.md) and [verification](../../docs/features/HB-147/verification.md).

Run `node tools/retained-connections-qa.mjs --url <prepared-local-frontend> --expected-total <verified-current-count> --output <local-evidence-directory> --playwright-module <installed-module> --chromium <browser-executable>` from this directory. The optional expected-total is an explicit QA assertion, not a product assumption; use0 only for the verified empty retained environment. The runner distinguishes actual protected reads from isolated synthetic interception and never confirms owner review or issues commands. Source connection configuration, credentials, diagnostics and live CSP authentication remain unconnected.
