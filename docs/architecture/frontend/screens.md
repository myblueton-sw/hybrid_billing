---
type: ScreenDirectory
title: Whole-frontend screen ownership
description: Assigns each of the 112 existing design routes to a proposed feature and concrete React screen path.
status: draft
---

# Screen directory

This table renders [screen-ownership.json](screen-ownership.json). [Architecture](structure.md) defines the boundaries. Paths are proposed, not existing implementation or owner acceptance. Original registry mappings and related routes remain in the JSON. Semantic item IDs must be carried from each actual screen design; this route table does not replace the item checklist.

## design-review

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| overview | Hybrid Billing — complete review | `apps/console/src/features/design-review/screens/overview/Screen.tsx` |
| contents | Complete screen index | `apps/console/src/features/design-review/screens/contents/Screen.tsx` |
| demo-scenarios | Synthetic scenarios | `apps/console/src/features/design-review/screens/demo-scenarios/Screen.tsx` |

## identity

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| entry-start | Welcome to Hybrid Billing | `apps/console/src/features/identity/screens/entry-start/Screen.tsx` |
| entry-bootstrap | Register the first administrator | `apps/console/src/features/identity/screens/entry-bootstrap/Screen.tsx` |
| entry-invite | Join by invitation | `apps/console/src/features/identity/screens/entry-invite/Screen.tsx` |
| entry-login | Sign in | `apps/console/src/features/identity/screens/entry-login/Screen.tsx` |
| entry-challenge | Verify identity or recover access | `apps/console/src/features/identity/screens/entry-challenge/Screen.tsx` |
| entry-scope | Choose an authorized workspace | `apps/console/src/features/identity/screens/entry-scope/Screen.tsx` |
| entry-roles | Roles and delegated permissions | `apps/console/src/features/identity/screens/entry-roles/Screen.tsx` |
| entry-sessions | Sessions and access recovery | `apps/console/src/features/identity/screens/entry-sessions/Screen.tsx` |
| users | Users and invitations | `apps/console/src/features/identity/screens/users/Screen.tsx` |
| identity-policy | Identity providers and sign-in policy | `apps/console/src/features/identity/screens/identity-policy/Screen.tsx` |
| service-accounts | Service accounts and credential references | `apps/console/src/features/identity/screens/service-accounts/Screen.tsx` |
| personal-security | Personal security and notifications | `apps/console/src/features/identity/screens/personal-security/Screen.tsx` |

## onboarding

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| setup-progress | Set up your first billing workspace | `apps/console/src/features/onboarding/screens/setup-progress/Screen.tsx` |
| setup-profile | Business and billing foundation | `apps/console/src/features/onboarding/screens/setup-profile/Screen.tsx` |
| setup-attribution | Connect customers to billing accounts | `apps/console/src/features/onboarding/screens/setup-attribution/Screen.tsx` |
| setup-connectors | Choose the first data connection | `apps/console/src/features/onboarding/screens/setup-connectors/Screen.tsx` |
| setup-readtest | Verify permission with an actual read | `apps/console/src/features/onboarding/screens/setup-readtest/Screen.tsx` |
| setup-schedule | Configure the first collection | `apps/console/src/features/onboarding/screens/setup-schedule/Screen.tsx` |
| setup-first | Validate the first collected data | `apps/console/src/features/onboarding/screens/setup-first/Screen.tsx` |
| setup-readiness | Ready for the first billing cycle? | `apps/console/src/features/onboarding/screens/setup-readiness/Screen.tsx` |

## operations

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| operations | What is preventing issuance? | `apps/console/src/features/operations/screens/operations/Screen.tsx` |
| core-index-close | Selected portfolio close workbench | `apps/console/src/features/operations/screens/core-index-close/Screen.tsx` |
| my-tasks | My tasks and notifications | `apps/console/src/features/operations/screens/my-tasks/Screen.tsx` |

## organizations

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| customer-detail | Customer account detail | `apps/console/src/features/organizations/screens/customer-detail/Screen.tsx` |
| organizations | Organization relationships | `apps/console/src/features/organizations/screens/organizations/Screen.tsx` |
| organization-impact | Relationship change impact | `apps/console/src/features/organizations/screens/organization-impact/Screen.tsx` |

## sources

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| index-intake | Resolve incomplete supplier evidence | `apps/console/src/features/sources/screens/index-intake/Screen.tsx` |
| v4-inbox | Supplier invoice inbox | `apps/console/src/features/sources/screens/v4-inbox/Screen.tsx` |
| v4-extract | Review PDF, extracted fields and AI findings | `apps/console/src/features/sources/screens/v4-extract/Screen.tsx` |
| v4-differences | Resolve PDF versus collected-data differences | `apps/console/src/features/sources/screens/v4-differences/Screen.tsx` |
| v4-audit | Accept evidence with an immutable audit trail | `apps/console/src/features/sources/screens/v4-audit/Screen.tsx` |
| core-index-sources | Supplier invoice reconciliation | `apps/console/src/features/sources/screens/core-index-sources/Screen.tsx` |
| core-index-aws | AWS source detail | `apps/console/src/features/sources/screens/core-index-aws/Screen.tsx` |
| core-index-variants | Provider billing structures | `apps/console/src/features/sources/screens/core-index-variants/Screen.tsx` |
| index-mapping | Assign the account and inspect the impact | `apps/console/src/features/sources/screens/index-mapping/Screen.tsx` |
| asset-measurements | Asset and measurement history | `apps/console/src/features/sources/screens/asset-measurements/Screen.tsx` |

## analytics

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| cost-pivot | Cost pivot by project, service and SKU | `apps/console/src/features/analytics/screens/cost-pivot/Screen.tsx` |
| raw-explorer | Raw cost lookup and export | `apps/console/src/features/analytics/screens/raw-explorer/Screen.tsx` |
| reports | Scoped billing reports | `apps/console/src/features/analytics/screens/reports/Screen.tsx` |

## pricing

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| auto-portfolio | Contract pricing operations | `apps/console/src/features/pricing/screens/auto-portfolio/Screen.tsx` |
| supplier-register | Contracts and rates | `apps/console/src/features/pricing/screens/supplier-register/Screen.tsx` |
| marketplace-register | Marketplace ISV costs and contracts | `apps/console/src/features/pricing/screens/marketplace-register/Screen.tsx` |
| marketplace-commitment | Commitment credit and ISV purchase planning | `apps/console/src/features/pricing/screens/marketplace-commitment/Screen.tsx` |
| supplier-aws | AWS · Contract evidence | `apps/console/src/features/pricing/screens/supplier-aws/Screen.tsx` |
| supplier-azure | Azure · Contract evidence | `apps/console/src/features/pricing/screens/supplier-azure/Screen.tsx` |
| supplier-gcp | GCP · Contract evidence | `apps/console/src/features/pricing/screens/supplier-gcp/Screen.tsx` |
| supplier-oci | OCI purchase contract | `apps/console/src/features/pricing/screens/supplier-oci/Screen.tsx` |
| supplier-alibaba | Alibaba Cloud · Contract evidence | `apps/console/src/features/pricing/screens/supplier-alibaba/Screen.tsx` |
| customer-contract | Contract lifecycle and effective versions | `apps/console/src/features/pricing/screens/customer-contract/Screen.tsx` |
| auto-rulebook | Northstar customer price book | `apps/console/src/features/pricing/screens/auto-rulebook/Screen.tsx` |
| auto-editor | Define a pricing rule | `apps/console/src/features/pricing/screens/auto-editor/Screen.tsx` |
| auto-exclusions | Discount interaction and exclusions | `apps/console/src/features/pricing/screens/auto-exclusions/Screen.tsx` |
| auto-mapping | Validate automatic matching | `apps/console/src/features/pricing/screens/auto-mapping/Screen.tsx` |
| auto-bulk | Bulk price application preview | `apps/console/src/features/pricing/screens/auto-bulk/Screen.tsx` |
| auto-exceptions | Pricing exceptions | `apps/console/src/features/pricing/screens/auto-exceptions/Screen.tsx` |
| auto-activation | Approve and schedule contract pricing | `apps/console/src/features/pricing/screens/auto-activation/Screen.tsx` |
| sales-policy | A default, explicit exceptions, one-off benefits | `apps/console/src/features/pricing/screens/sales-policy/Screen.tsx` |
| sales-impact | See the price before activating the rule | `apps/console/src/features/pricing/screens/sales-impact/Screen.tsx` |
| core-index-allocation | Cost allocation and retained costs | `apps/console/src/features/pricing/screens/core-index-allocation/Screen.tsx` |
| core-index-pricing | Customer selling calculation | `apps/console/src/features/pricing/screens/core-index-pricing/Screen.tsx` |
| index-pricing | Change a scoped contract rule | `apps/console/src/features/pricing/screens/index-pricing/Screen.tsx` |
| index-seats | Explain a mid-cycle seat adjustment | `apps/console/src/features/pricing/screens/index-seats/Screen.tsx` |
| index-commitments | Keep commitments and credits distinct | `apps/console/src/features/pricing/screens/index-commitments/Screen.tsx` |
| index-fx | Verify conversion, tax and document grouping | `apps/console/src/features/pricing/screens/index-fx/Screen.tsx` |
| onprem-pools | On-premises cost pools | `apps/console/src/features/pricing/screens/onprem-pools/Screen.tsx` |
| supplier-naver | NAVER Cloud contract evidence · unverified | `apps/console/src/features/pricing/screens/supplier-naver/Screen.tsx` |
| supplier-kt | KT Cloud contract evidence · unverified | `apps/console/src/features/pricing/screens/supplier-kt/Screen.tsx` |

## calculation

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| auto-job | Automatic calculation and review | `apps/console/src/features/calculation/screens/auto-job/Screen.tsx` |
| auto-trace | Explain every calculated price | `apps/console/src/features/calculation/screens/auto-trace/Screen.tsx` |
| ai-cost-waterfall | AI instance cost and stacked discounts | `apps/console/src/features/calculation/screens/ai-cost-waterfall/Screen.tsx` |
| cost-differences | Precision, AI version and support reconciliation | `apps/console/src/features/calculation/screens/cost-differences/Screen.tsx` |
| core-reconciliation-case | Reconciliation investigation | `apps/console/src/features/calculation/screens/core-reconciliation-case/Screen.tsx` |
| core-reconciliation-recalc | Calculation revision comparison | `apps/console/src/features/calculation/screens/core-reconciliation-recalc/Screen.tsx` |
| index-runs | Run the corrected Aster candidate | `apps/console/src/features/calculation/screens/index-runs/Screen.tsx` |

## issuance

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| core-reconciliation-approval | Financial approval review | `apps/console/src/features/issuance/screens/core-reconciliation-approval/Screen.tsx` |
| index-approval | Resolve feedback before fixed-version issuance | `apps/console/src/features/issuance/screens/index-approval/Screen.tsx` |
| assignment-policy | Issuer and reviewer assignment policy | `apps/console/src/features/issuance/screens/assignment-policy/Screen.tsx` |
| review-issue | Review and issue | `apps/console/src/features/issuance/screens/review-issue/Screen.tsx` |
| document-history | Document action history | `apps/console/src/features/issuance/screens/document-history/Screen.tsx` |
| core-reconciliation-issued | Invoice and follow-up status | `apps/console/src/features/issuance/screens/core-reconciliation-issued/Screen.tsx` |

## delivery

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| erp | ERP connection and document mapping | `apps/console/src/features/delivery/screens/erp/Screen.tsx` |
| file-handoff | File handoff and results | `apps/console/src/features/delivery/screens/file-handoff/Screen.tsx` |
| publication-packages | Disclosure and delivery packages | `apps/console/src/features/delivery/screens/publication-packages/Screen.tsx` |

## history

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| auto-history | Contract and calculation history | `apps/console/src/features/history/screens/auto-history/Screen.tsx` |
| v4-monthly | Monthly billing history | `apps/console/src/features/history/screens/v4-monthly/Screen.tsx` |
| v4-month | September close and evidence history | `apps/console/src/features/history/screens/v4-month/Screen.tsx` |
| v4-variance | Explain the change from August to September | `apps/console/src/features/history/screens/v4-variance/Screen.tsx` |
| core-reconciliation-correction | Post-issue source revision | `apps/console/src/features/history/screens/core-reconciliation-correction/Screen.tsx` |
| index-correction | Close the late-credit correction case | `apps/console/src/features/history/screens/index-correction/Screen.tsx` |
| audit-events | Audit event search | `apps/console/src/features/history/screens/audit-events/Screen.tsx` |

## customer-portal

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| auto-customer | How your contract determines your bill | `apps/console/src/features/customer-portal/screens/auto-customer/Screen.tsx` |
| ai-invoice-detail | Your AI usage invoice and discount detail | `apps/console/src/features/customer-portal/screens/ai-invoice-detail/Screen.tsx` |
| core-customer-overview | Billing overview | `apps/console/src/features/customer-portal/screens/core-customer-overview/Screen.tsx` |
| v4-customer-months | Your monthly billing history | `apps/console/src/features/customer-portal/screens/v4-customer-months/Screen.tsx` |
| customer-history | Your invoices and corrections | `apps/console/src/features/customer-portal/screens/customer-history/Screen.tsx` |
| core-customer-invoice | September invoice | `apps/console/src/features/customer-portal/screens/core-customer-invoice/Screen.tsx` |
| core-customer-usage | Service charges and usage detail | `apps/console/src/features/customer-portal/screens/core-customer-usage/Screen.tsx` |
| customer-inquiry | Ask about a specific charge | `apps/console/src/features/customer-portal/screens/customer-inquiry/Screen.tsx` |
| customer-response | Track the billing explanation | `apps/console/src/features/customer-portal/screens/customer-response/Screen.tsx` |

## administration

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| core-reconciliation-setup | Initial setup and source readiness | `apps/console/src/features/administration/screens/core-reconciliation-setup/Screen.tsx` |
| license-modes | Usage mode and license | `apps/console/src/features/administration/screens/license-modes/Screen.tsx` |
| license-status | License status and registration | `apps/console/src/features/administration/screens/license-status/Screen.tsx` |
| installation-diagnostics | Installation diagnostics | `apps/console/src/features/administration/screens/installation-diagnostics/Screen.tsx` |
| integrations | API, plugins and channels | `apps/console/src/features/administration/screens/integrations/Screen.tsx` |
| recovery-migration | Recovery and migration | `apps/console/src/features/administration/screens/recovery-migration/Screen.tsx` |
| retention-policy | Retention and deletion review | `apps/console/src/features/administration/screens/retention-policy/Screen.tsx` |
| support-matrix | Support matrix and release evidence | `apps/console/src/features/administration/screens/support-matrix/Screen.tsx` |
| community | Contribute | `apps/console/src/features/administration/screens/community/Screen.tsx` |

## assistant

| Screen ID | Purpose | Proposed screen file |
| --- | --- | --- |
| assistant | Billing assistant | `apps/console/src/features/assistant/screens/assistant/Screen.tsx` |
