---
type: ScreenDirectory
title: Whole-frontend screen ownership
description: Assigns each of the 112 existing design routes to a proposed business module and concrete React screen path.
status: draft
---

# Screen directory

This table renders [screen-ownership.json](screen-ownership.json). [Architecture](structure.md) defines the boundaries. Paths are proposed, not existing implementation or owner acceptance. Original registry mappings and related routes remain in the JSON. Semantic item IDs must be carried from each actual screen design; this route table does not replace the item checklist.

## design-review

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| overview | Hybrid Billing — complete review | `apps/console/src/modules/design-review/pages/overview/Screen.tsx` |
| contents | Complete screen index | `apps/console/src/modules/design-review/pages/contents/Screen.tsx` |
| demo-scenarios | Synthetic scenarios | `apps/console/src/modules/design-review/pages/demo-scenarios/Screen.tsx` |

## identity

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| entry-start | Welcome to Hybrid Billing | `apps/console/src/modules/identity/pages/entry-start/Screen.tsx` |
| entry-bootstrap | Register the first administrator | `apps/console/src/modules/identity/pages/entry-bootstrap/Screen.tsx` |
| entry-invite | Join by invitation | `apps/console/src/modules/identity/pages/entry-invite/Screen.tsx` |
| entry-login | Sign in | `apps/console/src/modules/identity/pages/entry-login/Screen.tsx` |
| entry-challenge | Verify identity or recover access | `apps/console/src/modules/identity/pages/entry-challenge/Screen.tsx` |
| entry-scope | Choose an authorized workspace | `apps/console/src/modules/identity/pages/entry-scope/Screen.tsx` |
| entry-roles | Roles and delegated permissions | `apps/console/src/modules/identity/pages/entry-roles/Screen.tsx` |
| entry-sessions | Sessions and access recovery | `apps/console/src/modules/identity/pages/entry-sessions/Screen.tsx` |
| users | Users and invitations | `apps/console/src/modules/identity/pages/users/Screen.tsx` |
| identity-policy | Identity providers and sign-in policy | `apps/console/src/modules/identity/pages/identity-policy/Screen.tsx` |
| service-accounts | Service accounts and credential references | `apps/console/src/modules/identity/pages/service-accounts/Screen.tsx` |
| personal-security | Personal security and notifications | `apps/console/src/modules/identity/pages/personal-security/Screen.tsx` |

## onboarding

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| setup-progress | Set up your first billing workspace | `apps/console/src/modules/onboarding/pages/setup-progress/Screen.tsx` |
| setup-profile | Business and billing foundation | `apps/console/src/modules/onboarding/pages/setup-profile/Screen.tsx` |
| setup-attribution | Connect customers to billing accounts | `apps/console/src/modules/onboarding/pages/setup-attribution/Screen.tsx` |
| setup-connectors | Choose the first data connection | `apps/console/src/modules/onboarding/pages/setup-connectors/Screen.tsx` |
| setup-readtest | Verify permission with an actual read | `apps/console/src/modules/onboarding/pages/setup-readtest/Screen.tsx` |
| setup-schedule | Configure the first collection | `apps/console/src/modules/onboarding/pages/setup-schedule/Screen.tsx` |
| setup-first | Validate the first collected data | `apps/console/src/modules/onboarding/pages/setup-first/Screen.tsx` |
| setup-readiness | Ready for the first billing cycle? | `apps/console/src/modules/onboarding/pages/setup-readiness/Screen.tsx` |

## operations

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| operations | What is preventing issuance? | `apps/console/src/modules/operations/pages/operations/Screen.tsx` |
| core-index-close | Selected portfolio close workbench | `apps/console/src/modules/operations/pages/core-index-close/Screen.tsx` |
| my-tasks | My tasks and notifications | `apps/console/src/modules/operations/pages/my-tasks/Screen.tsx` |

## organizations

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| customer-detail | Customer account detail | `apps/console/src/modules/organizations/pages/customer-detail/Screen.tsx` |
| organizations | Organization relationships | `apps/console/src/modules/organizations/pages/organizations/Screen.tsx` |
| organization-impact | Relationship change impact | `apps/console/src/modules/organizations/pages/organization-impact/Screen.tsx` |

## sources

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| index-intake | Resolve incomplete supplier evidence | `apps/console/src/modules/sources/pages/index-intake/Screen.tsx` |
| v4-inbox | Supplier invoice inbox | `apps/console/src/modules/sources/pages/v4-inbox/Screen.tsx` |
| v4-extract | Review PDF, extracted fields and AI findings | `apps/console/src/modules/sources/pages/v4-extract/Screen.tsx` |
| v4-differences | Resolve PDF versus collected-data differences | `apps/console/src/modules/sources/pages/v4-differences/Screen.tsx` |
| v4-audit | Accept evidence with an immutable audit trail | `apps/console/src/modules/sources/pages/v4-audit/Screen.tsx` |
| core-index-sources | Supplier invoice reconciliation | `apps/console/src/modules/sources/pages/core-index-sources/Screen.tsx` |
| core-index-aws | AWS source detail | `apps/console/src/modules/sources/pages/core-index-aws/Screen.tsx` |
| core-index-variants | Provider billing structures | `apps/console/src/modules/sources/pages/core-index-variants/Screen.tsx` |
| index-mapping | Assign the account and inspect the impact | `apps/console/src/modules/sources/pages/index-mapping/Screen.tsx` |
| asset-measurements | Asset and measurement history | `apps/console/src/modules/sources/pages/asset-measurements/Screen.tsx` |

## analytics

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| cost-pivot | Cost pivot by project, service and SKU | `apps/console/src/modules/analytics/pages/cost-pivot/Screen.tsx` |
| raw-explorer | Raw cost lookup and export | `apps/console/src/modules/analytics/pages/raw-explorer/Screen.tsx` |
| reports | Scoped billing reports | `apps/console/src/modules/analytics/pages/reports/Screen.tsx` |

## pricing

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| auto-portfolio | Contract pricing operations | `apps/console/src/modules/pricing/pages/auto-portfolio/Screen.tsx` |
| supplier-register | Contracts and rates | `apps/console/src/modules/pricing/pages/supplier-register/Screen.tsx` |
| marketplace-register | Marketplace ISV costs and contracts | `apps/console/src/modules/pricing/pages/marketplace-register/Screen.tsx` |
| marketplace-commitment | Commitment credit and ISV purchase planning | `apps/console/src/modules/pricing/pages/marketplace-commitment/Screen.tsx` |
| supplier-aws | AWS · Contract evidence | `apps/console/src/modules/pricing/pages/supplier-aws/Screen.tsx` |
| supplier-azure | Azure · Contract evidence | `apps/console/src/modules/pricing/pages/supplier-azure/Screen.tsx` |
| supplier-gcp | GCP · Contract evidence | `apps/console/src/modules/pricing/pages/supplier-gcp/Screen.tsx` |
| supplier-oci | OCI purchase contract | `apps/console/src/modules/pricing/pages/supplier-oci/Screen.tsx` |
| supplier-alibaba | Alibaba Cloud · Contract evidence | `apps/console/src/modules/pricing/pages/supplier-alibaba/Screen.tsx` |
| customer-contract | Contract lifecycle and effective versions | `apps/console/src/modules/pricing/pages/customer-contract/Screen.tsx` |
| auto-rulebook | Northstar customer price book | `apps/console/src/modules/pricing/pages/auto-rulebook/Screen.tsx` |
| auto-editor | Define a pricing rule | `apps/console/src/modules/pricing/pages/auto-editor/Screen.tsx` |
| auto-exclusions | Discount interaction and exclusions | `apps/console/src/modules/pricing/pages/auto-exclusions/Screen.tsx` |
| auto-mapping | Validate automatic matching | `apps/console/src/modules/pricing/pages/auto-mapping/Screen.tsx` |
| auto-bulk | Bulk price application preview | `apps/console/src/modules/pricing/pages/auto-bulk/Screen.tsx` |
| auto-exceptions | Pricing exceptions | `apps/console/src/modules/pricing/pages/auto-exceptions/Screen.tsx` |
| auto-activation | Approve and schedule contract pricing | `apps/console/src/modules/pricing/pages/auto-activation/Screen.tsx` |
| sales-policy | A default, explicit exceptions, one-off benefits | `apps/console/src/modules/pricing/pages/sales-policy/Screen.tsx` |
| sales-impact | See the price before activating the rule | `apps/console/src/modules/pricing/pages/sales-impact/Screen.tsx` |
| core-index-allocation | Cost allocation and retained costs | `apps/console/src/modules/pricing/pages/core-index-allocation/Screen.tsx` |
| core-index-pricing | Customer selling calculation | `apps/console/src/modules/pricing/pages/core-index-pricing/Screen.tsx` |
| index-pricing | Change a scoped contract rule | `apps/console/src/modules/pricing/pages/index-pricing/Screen.tsx` |
| index-seats | Explain a mid-cycle seat adjustment | `apps/console/src/modules/pricing/pages/index-seats/Screen.tsx` |
| index-commitments | Keep commitments and credits distinct | `apps/console/src/modules/pricing/pages/index-commitments/Screen.tsx` |
| index-fx | Verify conversion, tax and document grouping | `apps/console/src/modules/pricing/pages/index-fx/Screen.tsx` |
| onprem-pools | On-premises cost pools | `apps/console/src/modules/pricing/pages/onprem-pools/Screen.tsx` |
| supplier-naver | NAVER Cloud contract evidence · unverified | `apps/console/src/modules/pricing/pages/supplier-naver/Screen.tsx` |
| supplier-kt | KT Cloud contract evidence · unverified | `apps/console/src/modules/pricing/pages/supplier-kt/Screen.tsx` |

## calculation

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| auto-job | Automatic calculation and review | `apps/console/src/modules/calculation/pages/auto-job/Screen.tsx` |
| auto-trace | Explain every calculated price | `apps/console/src/modules/calculation/pages/auto-trace/Screen.tsx` |
| ai-cost-waterfall | AI instance cost and stacked discounts | `apps/console/src/modules/calculation/pages/ai-cost-waterfall/Screen.tsx` |
| cost-differences | Precision, AI version and support reconciliation | `apps/console/src/modules/calculation/pages/cost-differences/Screen.tsx` |
| core-reconciliation-case | Reconciliation investigation | `apps/console/src/modules/calculation/pages/core-reconciliation-case/Screen.tsx` |
| core-reconciliation-recalc | Calculation revision comparison | `apps/console/src/modules/calculation/pages/core-reconciliation-recalc/Screen.tsx` |
| index-runs | Run the corrected Aster candidate | `apps/console/src/modules/calculation/pages/index-runs/Screen.tsx` |

## issuance

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| core-reconciliation-approval | Financial approval review | `apps/console/src/modules/issuance/pages/core-reconciliation-approval/Screen.tsx` |
| index-approval | Resolve feedback before fixed-version issuance | `apps/console/src/modules/issuance/pages/index-approval/Screen.tsx` |
| assignment-policy | Issuer and reviewer assignment policy | `apps/console/src/modules/issuance/pages/assignment-policy/Screen.tsx` |
| review-issue | Review and issue | `apps/console/src/modules/issuance/pages/review-issue/Screen.tsx` |
| document-history | Document action history | `apps/console/src/modules/issuance/pages/document-history/Screen.tsx` |
| core-reconciliation-issued | Invoice and follow-up status | `apps/console/src/modules/issuance/pages/core-reconciliation-issued/Screen.tsx` |

## delivery

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| erp | ERP connection and document mapping | `apps/console/src/modules/delivery/pages/erp/Screen.tsx` |
| file-handoff | File handoff and results | `apps/console/src/modules/delivery/pages/file-handoff/Screen.tsx` |
| publication-packages | Disclosure and delivery packages | `apps/console/src/modules/delivery/pages/publication-packages/Screen.tsx` |

## history

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| auto-history | Contract and calculation history | `apps/console/src/modules/history/pages/auto-history/Screen.tsx` |
| v4-monthly | Monthly billing history | `apps/console/src/modules/history/pages/v4-monthly/Screen.tsx` |
| v4-month | September close and evidence history | `apps/console/src/modules/history/pages/v4-month/Screen.tsx` |
| v4-variance | Explain the change from August to September | `apps/console/src/modules/history/pages/v4-variance/Screen.tsx` |
| core-reconciliation-correction | Post-issue source revision | `apps/console/src/modules/history/pages/core-reconciliation-correction/Screen.tsx` |
| index-correction | Close the late-credit correction case | `apps/console/src/modules/history/pages/index-correction/Screen.tsx` |
| audit-events | Audit event search | `apps/console/src/modules/history/pages/audit-events/Screen.tsx` |

## customer-portal

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| auto-customer | How your contract determines your bill | `apps/console/src/modules/customer-portal/pages/auto-customer/Screen.tsx` |
| ai-invoice-detail | Your AI usage invoice and discount detail | `apps/console/src/modules/customer-portal/pages/ai-invoice-detail/Screen.tsx` |
| core-customer-overview | Billing overview | `apps/console/src/modules/customer-portal/pages/core-customer-overview/Screen.tsx` |
| v4-customer-months | Your monthly billing history | `apps/console/src/modules/customer-portal/pages/v4-customer-months/Screen.tsx` |
| customer-history | Your invoices and corrections | `apps/console/src/modules/customer-portal/pages/customer-history/Screen.tsx` |
| core-customer-invoice | September invoice | `apps/console/src/modules/customer-portal/pages/core-customer-invoice/Screen.tsx` |
| core-customer-usage | Service charges and usage detail | `apps/console/src/modules/customer-portal/pages/core-customer-usage/Screen.tsx` |
| customer-inquiry | Ask about a specific charge | `apps/console/src/modules/customer-portal/pages/customer-inquiry/Screen.tsx` |
| customer-response | Track the billing explanation | `apps/console/src/modules/customer-portal/pages/customer-response/Screen.tsx` |

## administration

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| core-reconciliation-setup | Initial setup and source readiness | `apps/console/src/modules/administration/pages/core-reconciliation-setup/Screen.tsx` |
| license-modes | Usage mode and license | `apps/console/src/modules/administration/pages/license-modes/Screen.tsx` |
| license-status | License status and registration | `apps/console/src/modules/administration/pages/license-status/Screen.tsx` |
| installation-diagnostics | Installation diagnostics | `apps/console/src/modules/administration/pages/installation-diagnostics/Screen.tsx` |
| integrations | API, plugins and channels | `apps/console/src/modules/administration/pages/integrations/Screen.tsx` |
| recovery-migration | Recovery and migration | `apps/console/src/modules/administration/pages/recovery-migration/Screen.tsx` |
| retention-policy | Retention and deletion review | `apps/console/src/modules/administration/pages/retention-policy/Screen.tsx` |
| support-matrix | Support matrix and release evidence | `apps/console/src/modules/administration/pages/support-matrix/Screen.tsx` |
| community | Contribute | `apps/console/src/modules/administration/pages/community/Screen.tsx` |

## assistant

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| assistant | Billing assistant | `apps/console/src/modules/assistant/pages/assistant/Screen.tsx` |
