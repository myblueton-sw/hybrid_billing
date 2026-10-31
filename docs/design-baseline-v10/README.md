---
type: DesignBaseline
title: Frozen v10 baseline and modular design management
description: Records owner acceptance of v10 as the base design and manages subsequent work by independently versioned functional modules.
status: stable
---

# Owner decision

On 2026-10-29 the owner explicitly fixed v10 as the base design and requested modular management because the complete book had grown. [HB-44](https://linear.app/hybrid-billing/issue/HB-44) records this decision. The baseline is the 89-page bilingual book from source commit `e955befffa54a9cefb392984ab7716a67c8e05b0` (HB-43 / PR40). Original PDF paths and SHA-256 values are frozen in [registry.json](registry.json).

Baseline acceptance is a design-management decision. It does not certify runtime behavior, resolve every detailed design choice, select authentication products, define CFRC, waive independent review or complete the [56 delivery checklist items](../reviews/HB-43-onboarding/checklist.md). Prior draft labels and pending-owner-approval text inside the frozen historical book describe its creation state; this later owner decision supersedes only the baseline-acceptance status. Do not rewrite frozen pages to erase that history.

# Modules

| ID | Responsibility | Baseline pages |
| --- | --- | --- |
| 01-overview | Guide and complete index | 1–2 |
| 02-access | Registration, login and permissions | 3–10 |
| 03-onboarding | Initial connection and collection | 11–18 |
| 04-work | Monthly operations and customer profile | 19–21 |
| 05-sources | Source/PDF evidence and cost exploration | 22–32 |
| 06-contracts | Contracts, pricing, Marketplace and allocation | 33–57 |
| 07-reconcile | Calculation, discrepancies and approval | 58–66 |
| 08-issue | Issuance, assignment and ERP | 67–71 |
| 09-history | Monthly history and correction | 72–77 |
| 10-customer | Customer portal and invoice explanations | 78–86 |
| 11-settings | Setup and license | 87–89 |
| 12-vmware-onprem | Dedicated VMware/on-prem detail | Planned; no delivered pages |

The registry supplies stable screen IDs and checklist references; original page numbers are lookup aids, not entity identity. Checklist wildcards refer to existing prefixes, never new checked acceptance claims. The 11 delivered modules partition all89 pages once per locale. VMware is a separate planned extension: existing connector candidates, allocation and usage examples are context, not a completed vCenter/inventory/metering/rating/reconciliation journey. Its gaps are enumerated in the registry and must receive a separate ticket before design execution.

# Change management

1. Keep active v10 baseline artifacts and its frozen source commit immutable. Do not edit HB-43 page generators to implement later module changes; reproduce the baseline from its pinned commit when required.
2. Create a ticket and owned module revision for each change, for example `02-access/r2` or the first `12-vmware-onprem/r1`. The module owns its maintained screen definitions; reference shared assets explicitly instead of modifying historical snapshots.
3. Each revision records baseline, previous revision, owner/ticket, changed stable screen IDs, reason, checklist IDs, dependencies, reviewer, verification and approval status. Root integrates shared dependencies; one writer owns each file.
4. Preserve cross-module links by stable screen ID. Identify affected source/target modules and review contracts together before changing navigation or shared behavior. A module approval is scoped, not approval of every other module.
5. Review only affected modules and relevant regression paths. Financial/tenant/permission changes still require independent review and actual runtime evidence before checklist completion.
6. Assemble a later complete edition only from an explicit manifest pinning every compatible module revision. Do not silently mix latest files or replace v10. VMware gaps can be filled independently without claiming the original baseline contained them.
7. Once the owner fixes a replacement as final, migrate references/build inputs and delete the superseded copies from the current tree under the [final-document cleanup rule](../../.agents/rules/workflow.md#final-document-cleanup). Use Git history for prior tracked editions, not archive copies. This later owner instruction supersedes indefinite current-tree retention; v10 remains active until its scope is actually replaced. An AWS-only r2 draft does not replace the complete v10 book.

Root owns registry integration. Domain owners and implementation tickets for future revisions are unassigned until claimed; no work or authority is invented. Source/billing contracts and the existing checklist remain canonical rather than duplicated per module.

# Local review package

Run `python3 docs/design-baseline-v10/build.py` with Poppler tools and the exact original PDFs available. The builder rejects mismatched hashes, splits existing pages without redesign, checks full extracted-text equality and first-page pixel equality per module, and verifies exactly-once coverage. Original PDF hashes are rechecked after generation. Module container bytes differ from the original PDF; page content is retained.

Open `output/designs/v10-modules/index.html`, then select Korean or English. The package contains11 PDFs per locale, with original baseline page numbering and unchanged historical header/navigation. Embedded PDF links retain original targets and are not independently remapped; use the package index to select modules or the original HTML book for complete navigation. This is a design review package, not a customer deployment.

The portable archive is `output/designs/hybrid-billing-v10-modules.zip`. Generated files remain outside Git; maintained registry, decision and builder are tracked. [Verification](verification.md) records actual results.

# Proposed module revisions

[03-onboarding r2: multi-provider FOCUS onboarding](../design-modules/03-onboarding/r2/README.md) defines common source selection, provider-specific setup, data/schema validation and background collection under HB-49/HB-51. It is a reviewed design proposal with runtime acceptance pending, not a replacement for frozen v10 pages or an assembled new edition.

[02-access r2: login, LDAP and federated SSO](../design-modules/02-access/r2/authentication-integration.md) supplies the HB-60 protocol and configuration proposal for LDAPv3, SAML 2.0 and OIDC, with session, mapping and recovery boundaries. It is planning-only; actual support and acceptance remain pending in the [verification plan](../design-modules/02-access/r2/authentication-verification.md).

# HB-50 rule verification

The owner clarified that explanatory code comments are prohibited and accepted final documents replace older copies in the current tree. Root updated AGENTS.md, the workflow rule and this management policy; root editorial review and document QA apply to this small rule clarification, not an independent review claim. Existing runtime, rights and evidence-retention contracts are unchanged. No bulk deletion is claimed: the current r2 onboarding proposal is a draft and does not supersede the full v10 baseline. Finalization must enumerate the actual replacement set and remove it, including obsolete generated copies.

Validation: PASS for root editorial scope review, rule-link target/anchor, existing OKF metadata and git diff --check. This ticket changes policy text only; no code or document deletion was executed.

# Planning coverage review

[HB-54 coverage audit](coverage-audit.md) distinguishes missing/partial operational screen journeys, existing coverage and owner-deferred detail. It does not alter the frozen baseline, certify runtime acceptance or authorize the deferred visual redesign.

[HB-55 focused settings/admin/overview check](admin-overview-audit.md) narrows the audit to dedicated missing and partial management views; it does not change any screen.
