---
type: DeliveryPlan
title: Customer access work with authentication selection deferred
description: Separates provider-independent authorization and disclosure work from the owner's deferred authentication decision.
---

# Owner direction and scope

On 2026-10-28 the owner deferred authentication-method selection. Do not ask for a local/OIDC/SAML/LDAP choice again merely to begin independent design work. No provider, credential flow, session duration or recovery policy is selected by this deferral. Login and external-user verification remain deferred, not complete or removed from product scope.

[HB-34](https://linear.app/hybrid-billing/issue/HB-34/implement-authenticated-customer-query-and-export-boundaries) tracks remaining customer access work after merged HB-32/HB-33. Accountable owner: Seung Woo Park. Root owns planning/integration; a separate architecture/security reviewer reviews authorization, disclosure and export boundaries. Machine for this planning increment: mac_mini. This document records sequencing only; it introduces no endpoint, authentication adapter or runtime access grant.

# Independent work sequence

| Order | Deliverable | Acceptance boundary |
| --- | --- | --- |
| 1 | Define the server-internal verified-principal input contract and current company/tenant/action/field grant evaluation. | Missing, forged or stale authority denies. Client tenant/actor IDs and company labels never create membership. Synthetic principals are test inputs, not authenticated users. |
| 2 | Define versioned customer disclosure profiles and typed query/result contracts from existing detail and portal requirements. | Explicit permitted fields; hidden purchase cost, margin, raw source payload/locators and foreign metadata cannot appear through rows, filters, groups, counts, errors or evidence references. FOCUS source cost is not customer selling price. |
| 3 | Design protected export manifests, job checkpoints and download authorization against the same current grants and data revision. | Recheck before work, at execution checkpoints and before release; reduced grants invalidate complete-output claims. Preserve exact decimals, units, currency and amount basis. No raw-store download fallback. |
| 4 | Implement and test reviewed boundaries using synthetic identities/data and internal adapters. | Verify cross-customer, hidden-field, concurrent-scope and revoke-during-work cases. Report internal contract evidence separately from external end-to-end verification. |
| Later | Select and integrate authentication, then connect and verify actual customer entry points. | Principal establishment, session/revocation behavior and adversarial external testing remain required before customer access can be certified. |

Backend remains Go and data processing Python under the [owner's language decisions](../../architecture/current-design-decisions.md). Do not introduce a substitute public Python authentication/API service to avoid the deferred selection. The trusted local FOCUS CLI does not become a public customer gateway merely by adding projection.

# Sources and limits

Use the [identity obligations](../../architecture/security-boundaries/identity-session-permissions.md), [mandatory query boundary](../../contracts/security/access-and-trust.md), [detail/export disclosure contract](../../architecture/usage-detail-lifecycle.md) and authorized local C01/C02 screen evidence. Exact disclosure profiles and supported export limits require scoped design before implementation; this plan does not invent a production allowlist or default lifetime.

HB-33's [internal SQL revocation evidence](../../validation/reports/HB-33-reader-revocation/results.md) remains valid within its stated scope. I18/I19 external-path certification is still NOT_RUN. Deferral does not authorize anonymous/customer-supplied identity bypasses, public deployment, real-account access or customer-data use. Broader HB-34 remains open after this planning increment.

# Review evidence

## Implemented internal increments

PR31 contains the [current-grant query boundary](query-authorization.md), [usage-only projection/source bridge](usage-projection.md) and [protected internal export jobs/download](protected-export.md). Go connects to the existing Python data-processing path and actual local PostgreSQL/ClickHouse with synthetic data while retaining source protection through final Go authorization. [Projection evidence](../../validation/reports/HB-34-query-authorization/usage-projection.md) and [export evidence](../../validation/reports/HB-34-query-authorization/protected-export.md) distinguish actual engine tests from synthetic identity proof. Selling/allocation prices, public customer download routes/formats, durable/distributed job storage and external authentication remain open; the original sequence is not a completed-feature claim.

## Sequencing review

Independent architecture/security review PASS for this sequencing change: provider deferral, synthetic-versus-authenticated evidence, current grants, hidden fields, financial bases and Go/Python ownership remain consistent with existing contracts. Root and reviewer checked frontmatter, relative links and whitespace. No runtime code changed and no runtime tests were run for this document-only increment.
