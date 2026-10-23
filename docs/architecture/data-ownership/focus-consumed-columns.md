---
type: Data Mapping Contract
title: Consumed FOCUS 1.2 and 1.3 columns
description: Defines the proposed first-path field mappings and function restrictions from pinned official specifications, separately from exploratory FOCUS 1.0 demo observations.
status: draft
---

# Consumed FOCUS fields

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Owner: Seung Woo Park. Machine: `mac_mini`. Proposed mapping revision 1, pending implementation and independent review. Read with [physical storage](focus-physical-schema.md) and [demo inventory](../../validation/reports/HB-27-focus-demo-inventory/summary.md).

## Sources and interpretation

Pinned official source trees: [FOCUS 1.2 at 7296e314](https://github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS_Spec/tree/7296e3146fb931ee32b6496fa6e4200e7f4384f7/specification/columns) and [FOCUS 1.3 at c67b95bd](https://github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS_Spec/tree/c67b95bd40e6fb43a57f74a53a615c4d6d3c118f/specification/datasets/cost_and_usage/columns). Column files generally use lowercase names; the pinned 1.2 exceptions are ProviderName → `provider.md`, PublisherName → `publisher.md`, and InvoiceIssuerName → `invoiceissuer.md`. Use the exact version tree, not guessed paths. These source revisions were inspected, not inferred from the demo. Product checks consumed columns and applicability only; this is no provider-conformance certificate or embedded official validator.

The selected demo has 44 columns and declares 1.0; it is exploratory-only. No compatibility adapter or supported-version end-to-end fixture is delivered here. All columns below have explicit first-path consumers. Other fields stay in the original and are not implicitly exposed or available to formulas. A new consumer needs a reviewed mapping/profile revision. Identifiers, descriptions and tags remain protected even when not numeric.

`R` means a required header and nonnull value for this consumer. `C` means header/value applicability follows the pinned field requirements and activated provider capabilities; NULL is preserved with reason. Required headers are checked before collection; value/semantic/cross-file gates run before publication. An absent-by-version field is distinct from source NULL. Physical Nullable storage does not relax accepted input requirements.

## Mapping inventory

| Source 1.2 | Source 1.3 | Internal field(s) | Gate and allowed function |
| --- | --- | --- | --- |
| BillingAccountId | same | billing_account_id | R; source-account mapping/reconciliation; never tenant identity |
| BillingCurrency | same | billing_currency | R; exact cost grouping; never default USD or FX |
| BillingPeriodStart, BillingPeriodEnd | same | billing_period_start/end | R; half-open billing coverage, no receipt-time substitute |
| ChargePeriodStart, ChargePeriodEnd | same | charge_period_start/end | R; half-open usage/economic period, partition hint only |
| ChargeCategory | same | charge_category | R; versioned exact enum; no silent case folding |
| ChargeClass | same | charge_class | C value; required header; preserve correction vs noncorrection |
| ServiceName, ServiceCategory | same | service_name/category | R; approved service attribution and reporting, names cannot establish contract rates |
| BilledCost | same | billed_cost | R; supplier billed basis only, never customer selling price |
| EffectiveCost | same | effective_cost | R; effective basis only, never add to billed basis |
| ListCost | same | list_cost | R; list comparison only |
| ContractedCost | same | contracted_cost | R; contracted comparison only |
| ConsumedQuantity, ConsumedUnit | same | consumed_quantity/unit | C; actual usage path only with eligible unit/source/service mapping |
| PricingQuantity, PricingUnit | same | pricing_quantity/unit | C values; required headers; supplier pricing volume, not interchangeable with consumed quantity |
| ListUnitPrice, ContractedUnitPrice | same | list_unit_price/contracted_unit_price | C values; source pricing controls only; never adopt as selling tariff without contract |
| CommitmentDiscountStatus | same | commitment_discount_status | C; usage applicability and unused status; not benefit purchase or amortization authority |
| SkuPriceId | same | sku_price_id | C; price identity/applicability, especially 1.3 quantity null rules |
| SubAccountId | same | sub_account_id | C; approved account attribution, absence never grants whole payer access |
| ResourceId | same | resource_id | C; drill-down/approved allocation, not universal economic dedup key |
| Tags | same | tags_source | C; protected validated key/value representation; no automatic dynamic pricing variable |
| ProviderName | ServiceProviderName | service_provider_name | R; 1.3 explicitly replaces ProviderName; retain declared version and original lexical value |
| ProviderName | ProviderName (deprecated) | provider_name_legacy | Required by the pinned source revision; retain independently and check declared-version rules; no precedence fallback masking conflicts |
| PublisherName | PublisherName (deprecated) | publisher_name_legacy | Required by pinned revision; retain publisher identity independently, never use as infrastructure host |
| not defined | HostProviderName | host_provider_name | 1.2: NULL absent_by_version; 1.3 required header, conditional value per infrastructure semantics |
| InvoiceIssuerName | same | invoice_issuer_name | R; invoice issuer independent from provider, publisher and host |

The first-path registry must enumerate each comma-separated source field separately with physical name/type, version, header rule, applicability predicates, disclosure class and enabled functions before loader activation. This table is the human design contract, not executable mapping code. No unit/currency aliases or value substitutions are supplied by name similarity.

## Conditional values and restrictions

All four cost fields are nonnull in the pinned 1.2/1.3 definitions. Missing cost in the 1.0 demo is observed absence, not authority to populate zero or certify a 1.2/1.3 release. In this first path missing mandatory consumed cost holds/quarantines the affected attribution. A later usage-only profile may enable only independent-tariff usage functions through a separately reviewed consumed subset; it cannot claim complete supplier-cost analytics.

Consumed quantity distinguishes nonusage/unused commitments from usable usage and correction cases. The 1.3 rules additionally require NULL when SkuPriceId is NULL. Pricing quantity is NULL for Tax; eligible noncorrection Usage/Purchase values are required, subject to the complete 1.3 SkuPriceId rules. The corresponding units follow quantity applicability and source unit definitions. Evaluate the complete pinned predicates together; if supplied capability/category/price identity makes requirements inconsistent or applicability unknown, hold that scope rather than invent precedence. Corrections never reset stable economic coverage.

Pricing quantity can count commitment purchases or commitment usage differently by purpose: exclude the counterpart set under the declared utilization/cost calculation to avoid double counting. Unit prices are compared with cost/quantity only where versioned predicates apply, in exact Python arithmetic with approved bounds, never CH division or implicit rounding. Monetary measures group by exact currency/measure/basis; quantity groups also require exact unit revision and applicability. Sums include present/missing/not-applicable counts and never establish completeness alone.

1.3 HostProviderName identifies underlying infrastructure. NULL is permitted where no such infrastructure exists or its host is not uniquely determinable (for example particular Tax/Adjustment rows); other cases require the version's host rules. Do not derive HostProviderName from PublisherName. Do not copy InvoiceIssuerName into either provider field. Where both new and deprecated 1.3 fields exist, retain them separately and validate their official meanings; no coalesce hides conflicts. See [ServiceProviderName](https://github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS_Spec/blob/c67b95bd40e6fb43a57f74a53a615c4d6d3c118f/specification/datasets/cost_and_usage/columns/serviceprovidername.md) and [HostProviderName](https://github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS_Spec/blob/c67b95bd40e6fb43a57f74a53a615c4d6d3c118f/specification/datasets/cost_and_usage/columns/hostprovidername.md).

FOCUS NumericFormat permits scientific notation and provider-specific precision; the internal API's fixed decimal syntax is a separate contract. A reviewed bounded source parser converts finite external numbers exactly to canonical fixed point, validates original scale/magnitude and rejects unsupported values before CH. Literal NULL/empty interpretation is format/profile-specific; observed sample tokens do not become universal string-null aliases. No trimming, case normalization or default zero is inferred.

## Evidence and remaining gates

Official source inspection and demo profiling are completed evidence. Provider capability/applicability registry, approved P/S/count bounds, 1.2/1.3 fixtures, full semantic mapping tests and product ingestion are pending. Required acceptance covers exact header spelling, missing mandatory/conditional values, absent-by-version reasons, enum case errors, host/provider/publisher distinction, scientific-number limits, cross-tenant attribution and independently computed complete controls. SaaS/Datadog product scope remains; its 273 demo rows do not prove a native 1.2/1.3 export or connector. Product support is 1.2/1.3 only until an explicit reviewed version decision changes it.
