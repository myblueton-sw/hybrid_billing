---
type: SecurityTestPlan
title: MSP customer isolation scenario design and verification
description: Defines positive and adversarial synthetic scenarios for ingestion, database roles and queries while keeping unimplemented customer portal paths unverified.
---

# Scope and ownership

Ticket HB-32; accountable owner Seung Woo Park; machine mac_mini. Root owns scenario design, test harness/report and integration. Independent DBA and security/QA reviewers inspect design and actual results. Base is merged main `8153975c19fe27e32569e5416ef5f9298362ec08`; prior HB-31 query evidence is in PR27 and does not certify these new cases.

Owned files: `tests/focus_import/msp_isolation/`, `docs/work/HB-32-msp-isolation/`, `docs/validation/reports/HB-32-msp-isolation/`. A reproduced existing scope defect may receive a narrowly reviewed fix in `modules/focus_import/collection.py` or its scope helper, after independent design review; no new HTTP/auth/customer-portal feature is implied.

The pass criterion is absence of unauthorized rows, values, identities, totals, counts, graph objects, delivery periods/batch references and managed original bytes. A harmless empty row set with another customer's metadata still fails. Hash/opaque reference disclosure and unrelated snapshot invalidation must be assessed, not dismissed because no cost row crossed the boundary.

# Synthetic installation and expected ownership

| Scope | Company identity / display label | Source rows | Exact authorized billed total |
| --- | --- | ---: | ---: |
| Tenant A1 | company-a / Equal Company | AWS10, AWS12, Google Cloud3 | USD25 |
| Tenant A2 | company-a / Equal Company | AWS7 | USD7 |
| Tenant B | company-b / Equal Company | AWS100, Google Cloud200 | USD300 |
| Explicit company-A audience | Only A1+A2, not B | 4 | USD32 |

All services use the same synthetic service name. Accounts share the synthetic billing payer string; subaccounts distinguish customers, and identical account/subaccount strings across AWS/GCP remain provider-qualified. Sources contain distinct synthetic resource/private markers. A's delivery month is August2026, B's September2026, so unauthorized period/empty-slice disclosure is observable. Register three separate strict consumed1.3 connections (A1,A2,B), not one global source connection whose legitimate disclosure already includes every customer. A1/A2 are separate tenant access scopes; this does not implement department-level human permissions inside a tenant. Customer readers are trusted local test personas backed by actual scoped database roles, not authenticated portal users.

# Scenario matrix

| ID | Scenario | Expected result / current execution boundary |
| --- | --- | --- |
| I01 | A1,A2,B normal reads | Exact respective rows/totals; no other scope's private marker |
| I02 | Explicit company-A union versus ordinary A1 reader | Union32 only for the explicit audience; A1 never inherits A2 because of company/organization labels |
| I03 | Equal company labels and shared payer | Company B remains separate; payer identity never creates customer membership |
| I04 | Reused account IDs across provider namespaces | AWS and GCP identity is distinct; attribution remains exact |
| I05 | Supply B tenant predicate or forged session-context value on A's PG reader connection | Zero B rows; current role still limits scope |
| I06 | PG role escalation, protected configuration/manifest read and forbidden writes | Denied; tenant/financial metadata remains protected |
| I07 | ClickHouse unfiltered counts/sums, explicit B predicate and UNION/subqueries | Only A-authorized rows/measures; B predicate empty |
| I08 | ClickHouse role/settings escalation | Denied; policy cannot be disabled by the reader |
| I09 | Query aggregates, pages, graph and delivery coverage | Exact allowed union; no B batch IDs, graph keys or September coverage in A results, even if B slices are empty for A |
| I10 | Publish a B-only correction while A page/snapshot is in use | A values, permitted metadata and snapshot remain unchanged; unrelated B activity is not disclosed through invalidation |
| I11 | Same source domain assigned to two customers | Ambiguous mapping rejected before publication; no guessed ownership |
| I12 | B-domain file supplied to A1 connection / reader attempts collection registration | Denied; rejected unauthorized bytes are not retained as an A original/quarantine object |
| I13 | Cross-company audience definition, missing/foreign member or inactive audience | Invalid definition rejected; live member tamper/inactivation denies the whole view, not a silent partial union |
| I14 | Interleaved/concurrent A/B queries and reuse of A's own SQL connection | No actor or predicate contamination; this does not certify an unimplemented shared HTTP session pool |
| I15 | Archive/recall, backup, revoke then restore | Scoped values remain exact; old backup does not restore revoked product-query access |
| I16 | Unsigned S3 GET/LIST against only the new synthetic bucket | Denied or reported as a concrete storage-access limit; no probe of unrelated buckets |
| I17 | Untrusted direct engine metadata / existing SQL session after product-persona revocation | Conditional profile probe. If physical global counts or stale SQL authority remain visible, direct end-user SQL cannot be certified; these roles are internal verification identities and the planned portal forbids direct storage addresses |
| I18 | MSP employee customer grants, department-level human permissions, forged external user/actor, portal downloads and private MSP field projection | NOT_RUN where those API/auth/projection paths do not exist. The internal source_json query is not a customer-safe field projection |
| I19 | Customer-facing search/cache/export/queued-worker revocation | NOT_RUN for absent entry points; never inferred from internal CLI/database-row tests |
| I20 | An authorized A1 zero-usage month | Preserve its legitimate coverage and zero evidence; a metadata fix must not drop every empty slice |
| I21 | An owned period expires versus an unrelated B period | Owned unavailable data must block incomplete reads; unrelated foreign expiry must not block A or reveal B state |

The expected values above are fixed synthetic ground truth; compare exact coefficients/Fraction, NULLs and unit/currency groups without importing production aggregate functions as an oracle. Repeat scalar and metadata assertions on all returned pages. Distinguish core supported-boundary failures from conditional raw-engine probes and absent external-product features.

# Test isolation before writes

Preserve the root1123-row runtime and hb_local. Use a fresh dedicated PostgreSQL database on the existing local verification server, only if its chosen name does not already exist. Use the alternate verification ClickHouse service on18124 only after proving it has no hb_focus database or conflicting hb_focus users/policies. Create a uniquely named synthetic S3 bucket and separate local binding/control/lock paths. Never reset/destroy an unknown namespace.

Process-local transport adapters redirect only the stock PG connection's expected hb/15432 endpoint to the new database and the stock CH client's expected18123 endpoint to18124; schema SQL, roles, RLS, row policies, business validation and query functions remain unchanged. Patch the four binding-path references and all warehouse/archive/control bucket aliases before initialization. Use unique tenant keys to avoid reader-role collisions. Existing shared ingest/approve role attributes and memberships must not be changed; new schema grants affect only synthetic objects in the dedicated database. No .env change or credentials/logging is permitted.

Record namespace ownership, baseline default-runtime status/hash before and after, source hashes, setup gates and actual outcomes. Keep the isolated namespace for review rather than indiscriminately deleting stores or global roles. If the alternate server has foreign objects or cannot be verified, stop those writes and choose a separately reviewed isolated runtime. First run establishes actual failures; only independently reviewed narrow fixes justify targeted reruns.

Independent DBA and security/QA design review PASS, conditional on fail-closed preflight. Both reviews identify a source-inspected potential defect (not yet runtime-reproduced): `selected_slices` selects every window's slice for each reader member, including foreign empty slices, so coverage/batch graph keys and snapshot pins can expose another customer's activity. Reproduce I09/I10 first. The reviewed fix direction is to derive eligible connection/member ownership before slice lookup and metadata/pin generation, preserving authorized known-zero coverage and an explicit legacy/shared-window compatibility rule. Reviewers performed no tests or DB writes.
