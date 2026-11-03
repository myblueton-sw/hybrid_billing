---
type: Implementation
title: Source reconciliation service screen
description: Implements actual source and storage consistency diagnostics for S06 without inventing input eligibility.
---

# Scope and design

HB95 plan: fixed approver loopback process; new READ ONLY PostgreSQL inspection transaction checks existing confirm right and active capability, sets RLS scope, and shares connection advisory lock. No change to existing mutation guards. Reconciliation GET profile/list/detail (metadata only). POST candidate/revalidate binds expected manifest digest and computes original byte/hash, pinned registry/manifest controls, parsed original replay, ClickHouse row hash/count and SQL aggregates. Result is ephemeral source/storage consistency, no independent billing-validation or eligibility; always exposes missing independent evidence/atomic confirm-publication as blockers plus current quarantine/unknown/policy conditions. Exact decimal strings by currency/quantity units; null/missing retained, cap output comparison rows100 with explicit total/truncation. No confirmation/publication/rejection endpoints. Root sole writer, independent design/source and actual CLI visual review. Scope S06 source tab; sales/allocation/ERP/parallel remain existing unconnected candidates. Reuse HB141 isolated services read-only, no original dataset reset.

HB95 starts from main 36394ed and explicitly incorporates pending HB141 (2ae9c66) as a stacked dependency. No main merge or owner FIX is implied. Root owns implementation/QA; separate read-only architect reviews authority, decimal meaning and command boundaries.

Canonical S06 confirmation/publication remains unavailable: current confirmation and publication commit separately and do not establish independent cross-validation, complete source/provider evidence, canonical idempotency or confirm/reject serialization. Diagnostic matching cannot create awaiting_confirmation or eligible. Source replay uses the same pinned parser as ingestion; SQL aggregation is an independent aggregation path over the same extracted data, not independent billing evidence.

# Interface and screen contract

The fixed approver process exposes these same-origin routes under `/api/local-reconciliation`; every call requires `X-HB-Local-Client: 1`, and POST additionally requires the exact server Origin. Actor and scope never come from browser parameters. Loader processes deny this namespace; approver processes deny ingestion.

| Method/path | Behavior |
| --- | --- |
| GET `/profile` | Server-selected account, workspace, connection and period; explicitly no publication capability. |
| GET `/candidates` | Latest 25 scoped release metadata records, more flag and current holds. |
| GET `/candidates/{release-id}` | Scoped exact candidate metadata; foreign/unavailable IDs receive generic denial. |
| POST `/candidates/{release-id}/revalidate` | Bounded JSON containing only expected_manifest, a lowercase 64-character digest. Re-read and compare original, pinned manifest/registry and stored evidence. |

A changed expected digest returns 409. Invalid authority returns 403. Infrastructure failure returns 503 without internal exception text. Source integrity rejection returns an explicit unavailable result, not matched zero. Failed client requests clear protected profile/list/selection/result and require a fresh authorized lookup. Cancellation plus epochs discard late responses. Locale and route remounts remove temporary results. Revalidation does not mutate source objects, warehouse rows, confirmation/publication records or outbox state.

Exact integer coefficients are converted to six-place decimal strings. The browser never parses or rounds financial values. Each currency/consumption-unit/pricing-unit group remains distinct. Missing groups and missing measures retain null/explicit missing counts; differences are unavailable for incomplete groups. Comparison summaries cover the entire comparison set before the first 100 display rows are selected. Registry/source parsing uses the existing local FOCUS contract; it does not independently validate provider billing.

The desktop uses a full-width bounded candidate table, then selected evidence and a full-width comparison. On small screens candidates become labelled stacked rows; original/stored/difference values stack vertically as complete measure cards. A separately expandable comparison table retains all columns and a visible horizontal-scroll instruction. [Cloudscape split-view guidance](https://cloudscape.design/patterns/resource-management/view/split-view/) supports bottom details for wide comparison content. This is a layout reference, not adoption of its theme or a certification of current design quality. Claude CLI provided the concrete layout proposal and separately inspected actual screenshots. Proposed filename/last-check columns were omitted because the API does not supply them. Disconnected modes are stated in prose rather than presenting operational tabs.

| Stable implementation ID | Location and behavior | Verification boundary |
| --- | --- | --- |
| S06:region:source | Actual source reconciliation mode | Existing canonical screen identity retained. |
| S06:local:scope | Server-derived account/workspace/period | No client scope selection or authentication claim. |
| S06:local:candidates | At most 25 newest candidates, explicit more/empty | Selection clears prior comparison. |
| S06:local:revalidate | Digest-bound read-only revalidation action | Actual local synthetic services. |
| S06:local:comparisons | Exact amounts, units, missing values, complete counts | 100-row display bound, no cross-currency total. |
| S06:local:evidence | Seven integrity checks and expandable digests | Ephemeral source/storage diagnostic. |
| S06:local:blockers | Missing independent evidence and current holds | Matching never creates confirmation eligibility. |
| S06:local:async | Loading, failure, revocation, retry and stale-result invalidation | Protected data cleared after failed requests. |

The shared review control remains owner-only and revision/locale/layout/state-bound. No owner FIX was recorded. The source fingerprint changes with shared App/API dependencies; previous local acceptance must not carry into this candidate. S03 keeps the default ingestion API prefix and its existing behavior; other candidate screens remain previews.

# Running and verifying

Use the isolated synthetic services and portable Python/Node prerequisites from [HB-141 integration](../HB-141/integration.md). This task reads the existing isolated profile; it does not reset or initialize it. Credentials remain excluded. Existing local endpoints are fixed development ports, not a production configuration system.

From the repository root:

```sh
npm --prefix apps/console run build
python tools/console_service.py --local-operator --actor approver-a --port 8794
```

Open `http://127.0.0.1:8794/#/reconciliation?lang=ko&service=local`. Stop with Ctrl+C. Rebuild and restart after frontend changes because the asset allowlist is loaded on startup. `approver-b` selects the separate local B fixture at process startup, never through the browser. The canonical S06 input-confirmation workflow remains deferred.

```sh
python tests/console_service/test_reconciliation.py --isolated-profile
npm --prefix apps/console test
```

Run browser verification from `apps/console`:

```sh
node tools/reconciliation-qa.mjs --playwright-module <installed-module> --chromium <executable>
```

The Python tests default to skipping integration without the explicit isolated flag. They require previously ingested synthetic fixtures and existing quarantine/unknown examples; they do not seed or repair a deployment. Runtime tests use real PostgreSQL, ClickHouse and object-store reads. Failure/capability/corruption/truncation injections are explicitly mocked observations, not claimed infrastructure outages or real permission revocations.

# Verification and review evidence

- Nine Python tests PASS: exact large/negative decimal differences, missing/group/currency separation, bounded closed original streams, real source/storage comparison, HTTP role/scope/origin/digest guards, unavailable/corrupt evidence, read-only SQL write rejection and scope reset, preserved mutation holds, capability recheck simulations, complete count-before-truncation and unchanged confirmation/publication/outbox counts after repeated diagnostics.
- Existing 58 frontend tests PASS. TypeScript and production build PASS. The existing large-JavaScript-chunk warning remains; this task does not claim a bundle-performance improvement.
- Six actual renders: KO/EN/AR at 1440 and 390. Eleven browser scenarios PASS, including actual revalidation, keyboard selection/detail focus, changed selection, old-response rejection, 403/503/409 state removal, empty list, retry and reload invalidation. Final mobile checks additionally assert cell children fit within their cells.
- Vite verified immutable foundation hashes. A scoped new-screen CSS inspection passed semantic-token existence, no literal colors/rounded corners/shadows and source-file size limits. This is not an installed repository-wide style check.
- Independent static boundary review initially FAIL for retained protected data after a denied revalidation; root cleared all scoped state on failure, then targeted independent re-review PASS. The reviewer did not execute tests.
- Actual CLI visual review initially FAIL for mobile selected-label overlap. The first CSS correction was overridden by a later shared selector; computed browser height remained 44px. Root increased screen-local specificity and added a geometric regression assertion. The follow-up then identified clipped mobile primary amounts; root added vertically stacked exact values plus a separate expandable full table, consistent with the canonical S06 mobile contract. Final visual re-review is recorded separately below.

Local generated screenshots and browser evidence are under `output/validation/HB-95`, excluded from the PR. The first browser attempt failed because a running server retained the prior build's asset allowlist; restart resolved it. Failed attempts are not counted as passing runs. Actual host: macOS arm64, Python 3.11.7, Node 26.7.0, Chromium headless shell build 1243. Windows/Linux, other browsers, full assistive-technology and native-language acceptance remain UNVERIFIED. Portable entry points are design evidence, not universal OS execution proof.

Final actual Claude CLI visual closure: scoped PASS for the corrected selected-card overlap and complete mobile monetary values. Reviewed EN/KO/AR 390 and KO 1440 screenshots. The reviewer explicitly noted downscaled tall mobile images, unverified scrolling/native-language quality and optional compactness/currency-group-label refinements. Root separately verified actual mobile cell bounds and exact card values in the browser. No owner acceptance was created. Final build manifest: 242 hashes verified.

HB-142 binds live S06 object routes to exact scoped detail reads without automatic revalidation. See the [planning parity and correction record](../HB-142/planning-review.md) for current route verification and remaining integration gaps.
