---
type: ImplementationPlan
title: S04 source-field and row-detail explanation
description: Defines how S04 separates values read from mapped FOCUS source columns from values derived during normalization, using only the current authorized row contract.
status: draft
---

# Scope and ownership

[HB-151](https://linear.app/hybrid-billing/issue/HB-151/refine-s04-source-fields-and-row-detail-to-separate-source-values-from) is a scoped child of S04 HB-93 and follows HB-149/HB-150 from main `9712d7f`. Root (Claude Code) owns planning, frontend implementation, terminology writing, QA and integration. The ui-designer role leads composition and final render comparison. A separate read-only reviewer checks money semantics, authorization/clearing behavior, the actual change and copy. Machine: mac_mini. Onboarding stays deferred. No owner FIX is created.

# Observed baseline

Facts read on 2026-11-11 from the canonical preview and the protected reader through the console proxy:

- `/api/focus-overview` returns 1,121 authorized rows, 78 account scopes, FOCUS version list `["1.0"]`, currencies USD and CNY, and two tenants with time zones `Asia/Seoul` and `UTC`.
- The row DTO supplies 25 allowlisted keys: identity (`id`, `account`), mapped strings, four UTC instants, eight nullable 18-place coefficients, `charge_month`, `evidence_class`, `batch`, `ordinal`.
- Across the retained sample: `contracted_cost` and `contracted_unit_price` are missing on 7 rows; `consumed_quantity`/`consumed_unit` and `list_unit_price` are missing on 1 row; exact zero occurs in every numeric field; 13 rows carry negative costs; 2 rows are CNY-denominated and 1,119 are USD-denominated; consumed and pricing units differ on 116 rows; BilledCost differs from EffectiveCost on 733 rows; the charge-start month differs from the UTC month of `charge_start` on 17 rows; one row's charge period starts outside its billing period.
- The first row on the default page (Oracle) carries BilledCost `0.24`, EffectiveCost exact `0`, missing ContractedCost and missing ContractedUnitPrice. The current screen renders the missing values as `USD · 미확인`, which does not distinguish "missing in the collected data" from "not yet verified", and exact zero has no distinguishing text.
- The stored `missing_reasons`, original column text, row tenant, row mapping version and delivery window are held by the backend but are not part of the authorized row DTO.

# Decision

Frontend-only change. `/api/focus-overview`, its DTO, the projection identity, permissions and storage are unchanged.

Alternatives considered:

1. Extend the DTO with `missing_reasons`, row tenant/time zone and row mapping version. Rejected for this ticket: it is a new evidence contract that changes a protected public projection and needs its own design and tenant/financial review. Recorded below as a deferred scope decision.
2. Infer the row's tenant time zone by testing which scope time zone reproduces `charge_month`. Rejected: inference presented as evidence.
3. Explain only from supplied values and the tracked mapping contract. Chosen.

# Field model

A pure module `collected-focus/fields.ts` classifies each displayed field. It performs no network access and no money arithmetic beyond testing whether a coefficient is zero.

| Origin | Meaning | Fields |
| --- | --- | --- |
| From source columns | Value read from the named FOCUS column through the version mapping; provider conformance is not verified | ProviderName (ServiceProviderName in the 1.3 mapping), ServiceName, BillingAccountId, SubAccountId, BillingCurrency, ConsumedUnit, PricingUnit, BillingPeriodStart/End, ChargePeriodStart/End, BilledCost, EffectiveCost, ListCost, ContractedCost, ConsumedQuantity, PricingQuantity, ListUnitPrice, ContractedUnitPrice |
| Derived by Hybrid Billing | Computed by the product; no FOCUS column holds it | Charge-start month (derived at query time), row ID, account-scope ID |
| Delivery lineage | Identifies the collected delivery and the row position inside it | Delivery ID (`batch`), row ordinal in delivery |
| Sample annotation | `x_HB_EvidenceClass` carried in the collected sample row and allowlisted at query time; an absent or unrecognized value is shown as Unclassified, not as a recorded class | Sample provenance |

Source-column names are a client mirror of `packages/contracts/focus_import/versions.json`; a unit test reads the tracked contract and fails if any version differs. The mirror reflects the currently tracked contract; whether a delivery pins the mapping content used at collection time is unverified and is not claimed. The scope's FOCUS version is stated as the row's mapping version only when the scope has exactly one version; otherwise the versions are listed and the row version is stated as not carried by the row record. A version without a mirrored mapping is reported as "column mapping not available", never guessed.

Handling statements are limited to what the decoder guarantees: text is shown as collected, numerics are exact decimals at scale 18, instants are normalized to UTC. The screen does not claim the original text form, the source time zone, column presence, provider omission, conformance or a universal null-token list.

Value states for the eight numeric row fields:

| State | Condition | Presentation |
| --- | --- | --- |
| Missing | `null` | "Missing" alone in the value cell, without a currency, dash or 0, plus the note "No value in the collected row". One sentence per detail states that the row record does not carry the reason. |
| Exact zero | coefficient equal to zero (`"0"`, defensively `"-0"`) | The exact formatted zero with an "Exact zero" note. |
| Value | any other coefficient | Exact decimal through the existing formatter; negative values keep their sign and are never labeled refund or credit. |

These markers apply to row fields only. Scope totals keep their own wording: a total with no present values reads "no values present", and a zero total is not labeled exact zero because it can be a net of offsetting rows. BillingCurrency is shown as its own field and column and beside present costs and unit prices. Quantities show their own unit; unit prices show PricingUnit; a missing unit is stated even when the number is present. BilledCost and EffectiveCost are separate rows and are never summed, differenced or converted. No denominator, rate, FX or formula is derived.

The missing-values count in the source-fields table covers the 12 nullable mapped fields: SubAccountId, ServiceName, ConsumedUnit, PricingUnit and the eight numerics. The description above the table states that denominator.

Periods are four separate concepts: charge period and billing period (start inclusive, end exclusive by FOCUS definition; shown in UTC; the code enforces only start before end), charge-start month (derived), and delivery lineage. The detail states the UTC calendar month of ChargePeriodStart next to the derived charge-start month as two supplied facts and, when they differ, that the derived month uses the tenant time zone. The tenant time zone is stated only when every tenant in the scope shares one. Otherwise the detail says the row record does not carry its tenant and lists the scope's distinct time zones (time zones only, deduplicated, at most eight shown). The response-level join from `row.account` to `accounts[].tenant` is deliberately not used because it depends on account paging and search. One neutral sentence is shown when the charge period starts outside the billing period. An instant that is not a `+00:00` string is shown raw with "time format not interpreted" and no UTC claim.

Not carried by the row record, and therefore stated and not inferred: original column text, missing reason, row mapping version (when the scope has several), row tenant and time zone (when the scope has several), source delivery window, row-level transformation history, and a link to original evidence. The history tab stays unavailable and no S05 link is added.

# Screen behavior

- Source-fields view: a closed-by-default field mapping reference (field, FOCUS column, handling; grouped by origin) and a table whose header band groups columns by origin: account/detail, provider/service, missing values, currency, charge period, billing period | charge-start month, row ID | delivery ID/row ordinal, sample provenance. Timestamps are non-wrapping `YYYY-MM-DD HH:MM:SS` with UTC in the header.
- Row detail (all row views): a source table (field, FOCUS column, value, note) grouped as costs, quantities and unit prices, periods, account and service; then derived values, delivery lineage, sample annotation and the not-carried list. Close controls at both ends.
- Terminology corrections in the same screen: nulls read "Missing/누락/مفقود" instead of "Unavailable/미확인"; "interval" becomes "period"; the scroll hint no longer calls table columns "source columns"; sample provenance labels follow the glossary and distinguish public sample subsets.
- Unchanged: toolbar, account mapping, 25-row server paging, snapshot handoff with U01, S03 handoff without a period filter, review control, route and tab keys, DTO and validator.

Selection and clearing: selection stays keyed to the complete query key and existing fail-closed clearing is unchanged. One defect found by the design review is fixed: the focus-return target is taken from the clicked control rather than `document.activeElement`. The review's concerns about a selection surviving a tab change do not occur: the application remounts the screen when the tab changes, which clears the selection and reloads the query; the focus effect therefore depends only on the selected row, and the browser suite asserts that a tab change closes the detail.

Stable item IDs: the existing eight `S04:` items are preserved. Added: `S04:reference:field-mapping`, `S04:detail:field-origin`, `S04:state:value-missing-zero`, `S04:state:evidence-unavailable`.

# Structure

`Screen.tsx` shrinks from 1,029 to under 900 lines. New files: `collected-focus/fields.ts` and `fields.test.ts` (field model), and in the page folder `labels.tsx` (shared copy), `FieldReference.tsx`, `FieldsTable.tsx` and `RowDetail.tsx`. The existing browser suite `apps/console/tools/collected-source-qa.mjs` is extended.

# Design review resolution

The independent design review of the first draft returned FAIL without blockers. Resolved in this revision: sample provenance reclassified as a sample annotation (M1); tenant wording narrowed to the row record (M2, option b); single-version scope states the row version and unknown versions are not guessed (M3); handling statements limited and the null-token mirror removed (M4); concept names and existing copy aligned with the glossary, with a glossary update in scope (M5); currency removed from missing values and totals given their own wording (M6); focus-return target fixed (L1), tab-change selection verified cleared by the existing remount (L2, L3); zero classification made numeric (L4); degraded rendering for uninterpreted instants and unmapped versions (L5); missing-count denominator stated (L6). Composition follows the ui-designer specification; its recommendation to drop the currency beside a missing value matches M6.

The owner's standing split assigns implementation to Codex; this increment's implementation was handed to Claude Code by the Codex handoff that the owner forwarded on 2026-11-11.

# Verification

- Unit: mapping mirror equals the tracked contract for every version; origin classes; value-state classification for null, `"0"`, `"-0"`, negative and 33-digit coefficients; missing-field listing; single, multiple and unmapped scope versions; unrecognized provenance; UTC instant parsing; UTC-month comparison with single and multiple time zones; charge start against the billing period.
- Existing console suite, typecheck, build and build verifier; isolated `tests/focus_overview` backend suite unchanged and rerun as regression only. No retained-store reset suite.
- Browser on the retained sample: EN/KO/AR at 1440 and 390; the Oracle row end to end; a row with missing consumed quantity and unit; a CNY row; a row with differing units; a row whose charge-start month differs from the UTC month; stale selection cleared on filter, page and snapshot change; 403, malformed and failed reads clear values and detail; focus restore; full viewport width, neutral computed borders, no page overflow, 16px body.
- Retained source/control file hashes unchanged before and after reads. No ingestion, reset, benchmark or Graphify measurement.
- Independent review of this plan and of the actual change; ui-designer composition and final render comparison; technical-writing review of KO/EN/AR wording in rendered context. AI review is not native-language, practitioner or owner acceptance.

# Deferred scope decisions

- Row-level evidence contract: missing reason, row tenant/time zone, row mapping version, source delivery window. Needs separate design plus tenant and financial review.
- Transformation history, S05 evidence locator, global service/model/quality/attribution filters, export, large-volume performance (HB-34, HB-39) remain deferred planning contracts.

Planning trace: HB-06 lineage, HB-07 exact values, currencies, units, NULLs and period semantics, bounded to the retained sample.
