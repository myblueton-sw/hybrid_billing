---
type: Validation Report
title: HB-27 native provider hierarchy execution
description: Records six-provider native graph normalization, scoped persistence and historical grouping tests plus observed-scope migration of the current public sample.
status: stable
---

# Outcome and ownership

HB-27, owner Seung Woo Park, machine mac_mini. Root writes/integrates/executes; an isolated implementer owns the pure model/registry/fixture/tests, and separate read-only DBA, architecture/security and QA reviewers check design/change/evidence. This implements the owner's requirement to normalize CSP-specific hierarchies now within the existing local intake/storage/query path. It does not claim native account API collection or production authentication.

Final suite `20261023T124948Z-f01817`: **197 passed —141 pure and56 actual PG18/CH26.8/S3 integration**, zero failures/errors/skips. All previous122 regression cases remain included. First full run `20261023T124351Z-62bf30` had181 passes and one test-label mismatch: the implementation correctly rejected multiple additive ancestors, while the test expected another error name. That assertion was corrected.

Independent DBA change review also found a real retained-profile defect: querying old graphs consulted today's registry. A valid renamed-axis registry reproduced the failure in `hierarchy-profile-before-fix`. Snapshots now retain the complete canonical public profile, validate its digest and use its original kinds/categories/axes/bounds. Tests cover current registry changes/unavailability and pinned-profile tampering; the important fix passed independent re-review.

# Verified contracts

- Six provider profiles model native type/namespace, aliases, independent organization/resource/billing/identity/management/placement/attachment relations and semantic role bindings. AWS organization-scoped OU/root, Azure directory-scoped management groups, account-local Alibaba IDs and VMware service-instance MoRefs reject invalid namespace edges. Billing transfers/cross-organization payment links remain distinct from containment.
- Exact source assignments determine tenant authority. A native manifest cannot create an account assignment or grant access through a shared payer. Native nodes retain original ID strings and are independent of labels/current parent. Missing native billing/identity roles are not fabricated, including VMware inventory without a payer identity.
- Actual persistence exercises immutable snapshot/node/edge/binding/role/head tables with tenant+batch+snapshot FKs and forced RLS. Direct reader mutation and cross-tenant FK references fail. Persisted bodies themselves are scoped, not only their UI/query projection. Provider/service/node filters remove unrelated bindings/nodes/edges before exposure; same payer does not reveal foreign siblings.
- Grouping counts source occurrences once, preserving exact currency/unit/NULL controls. Semantic billing/usage grouping works across distinct native kinds. GCP resource organization differs from billing organization. Multiple applicable VM datastore/network ancestors reject additive grouping; partial/unknown validity is unresolved. Every attributed edge covers the whole charge interval; observed_at does not invent historical validity.
- Catalog approval pins registry/configuration/predecessor/retained heads/lifecycle state. Typed historical snapshots freeze across catalog moves; only future source revisions use a new graph. Approved supplemental enrichment preserves the original intake pin and invalidates stale pages. Legacy paging requires its hierarchy pin as well.
- Archive/query/backup/schema-loss restore preserve all graph evidence. Recovery compares complete canonical row sets in all six native tables across every tenant/provider, plus company graph/control equality. CLI-entrypoint integration executes profiles→plan→distinct approval→apply→ingest→confirm/publish→semantic-role grouping against actual engines.

# Current collected sample migration

Actual run `20261023T130614Z-74babf`: **39 successful CLI commands** restore the existing verification runtime, apply migration004 through approved maintenance, display profiles, prepare/approve/apply observed-scope normalization, query every page, group invoice roles, create a graph-inclusive backup, restore it and read back the same current scopes/controls.

The current data remains **1123 rows**, north-a1112, north-b9 and foreign2. The authorized company sees1121; foreign scope stays excluded. All original normalized values/row occurrences and complete cost controls match the earlier sample run;972 independent Fraction measure/NULL checks pass. There are6 distinct observed invoice scopes and80 observed usage scopes. Grouping the company by billing role yields6 additive invoice groups without duplication.

All retained revisions are represented:15 snapshots/heads,268 projected nodes,2624 time-qualified edges and3253 exact fact bindings. This3253 count includes original and superseded retained revisions; it is not the current1123 economic row count. `native_binding_role=0` is intentional for this public dataset: no authoritative organization/directory/vCenter inventory was supplied. Normalization preserves the actual FOCUS account IDs with explicit FOCUS-only ancestry instead of manufacturing native parent trees.

Complete six-provider native graph evidence is separately synthetic. It includes shared AWS payer with foreign sibling, distinct Azure MCA/Entra/resource axes, GCP cross-organization billing, OCI compartment/subscription/domain, Alibaba management/member/payer separation and VMware multi-attachment inventory. These fixture assertions do not become observed facts about the public sample.

The new graph-inclusive backup is `120d19db-e114-4c8b-8178-ff16f5b1f58e`; actual restore reaches independent control sequence54. All six normalized-table counts and current query hierarchy snapshots/controls match after restore. Services and the1123-row sample remain queryable. The old `hb_local` namespace and applied migrations001–003 are unchanged.

# Evidence and limits

[Evidence JSON](evidence.json) pins exact run/JUnit/CLI/source hashes. Raw graph/account-bearing command output, catalog manifests, private journal/backup packages and sample ID extracts stay ignored/local. Only synthetic fixtures, code, public profiles and aggregate/hash evidence belong in Git.

Use the [native hierarchy runbook](../../../ops/installation/native-account-hierarchy.md). Profile and source metadata approval are trusted local scenario roles; no real-account inventory, production IAM/API/UI, full vendor compatibility or capacity claim follows. A current native catalog is not permission and cannot retroactively rewrite a typed historical cost snapshot. Ancestor grouping follows one chosen axis; it does not invent allocations or bridge independent graphs. Existing lifecycle restrictions, physical-copy residuals, unknown-operation blocking and10000-row query bounds remain applicable. Account hierarchy normalization is implemented; obtaining missing native metadata still requires an authoritative supplied inventory or a separately implemented/authorized connector.
