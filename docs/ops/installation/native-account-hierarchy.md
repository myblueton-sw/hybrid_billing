---
type: Runbook
title: Native provider account hierarchy normalization
description: Operates typed six-provider account graphs, exact FOCUS bindings and scoped historical hierarchy queries on the trusted local path.
status: stable
---

# Model and scope

Tracking: HB-27; owner Seung Woo Park; machine mac_mini. The owner requires native CSP hierarchy normalization in the current intake/storage/query model. The executable implementation adds six provider profiles, not a universal company/provider/account tree. Internal company and tenant IDs remain permission boundaries; source payer, directory tenant and resource IDs are separately typed external identities. See the [design and official source record](../../work/HB-27-native-hierarchy/brief.md) and [execution report](../../validation/reports/HB-27-native-hierarchy/summary.md).

| Provider | Independent structures represented by the profile |
| --- | --- |
| AWS | Organization/root/OU/accounts, management-account relationship and billing/payment transfer |
| Azure | Entra directory and governance/resource scopes; MCA billing account/profile/invoice section; EA department/enrollment account and MPA/CSP customer relationships |
| GCP | Organization/folder/project ancestry, billing account ownership/payment linkage and reseller sub-billing account |
| OCI | Organization/parent-child tenancies/compartments, subscription mapping and identity domain |
| Alibaba | Resource directory/folders/member accounts, separate management/settlement payer, account-local resource groups and RAM identities |
| VMware | Service-instance-qualified MoRefs, inventory folders/datacenters, compute/resource-pool placement and storage/network attachments |

These are versioned typed data contracts grounded in official documentation. They do not certify a live discovery API or every vendor product/version. AWS management account is not assumed to be the ultimate payer. Azure FOCUS invoice scope is not assumed to be its top-level native billing account. A VMware vCenter is not fabricated as a payer. Missing native roles/parents/effective times stay unknown.

# Identities and immutable persistence

Canonical native node identity includes installation namespace, provider, partition/realm, native kind, required kind namespace and original native ID. Labels/current parents/collector are not keys. Native ID strings retain leading zeros and case; aliases are retained observations, not automatic account merges. Scoped namespaces distinguish AWS organizations, Azure directories, account-local Alibaba IDs and VMware service instances; invalid cross-namespace containment rejects. Full ARM/OCID/MoRef identifiers remain provider-specific evidence rather than being truncated into one account number.

Migration004 adds scoped immutable `native_snapshot`, `native_node`, `native_edge`, `native_binding`, `native_binding_role` and a tenant/batch `native_head`. Composite FKs and forced RLS prevent cross-tenant or cross-snapshot references. Public provider definitions are embedded in each snapshot and digest-verified so later registry edits or unavailable current files cannot reinterpret retained data. Snapshot digest is pinned in new intake manifests, per-tenant metadata and the independent lifecycle journal. CH numeric rows and Parquet shapes remain unchanged.

Every source occurrence retains observed FOCUS invoice and usage scopes plus any explicitly supplied native billing/usage/resource/identity role. Normalized FOCUS references never invent missing organizations, payer ancestry, directory tenants or vCenter inventory. `status=typed-native` means observed native bindings exist, not that all ancestors or roles are complete. Evidence class and observation time distinguish synthetic, operator-supplied native and FOCUS-only observations.

# Provision native metadata

Prerequisite: the [existing local lifecycle](focus-collection-lifecycle.md) is active and healthy, with no unknown operation. Use `tests/engine_probes/.venv/bin/python tools/focus_lifecycle.py` for these commands. Upgrade an existing installation with the existing `tools/focus_data.py init --config PATH` only while its independent journal is current; never remove the journal to reinitialize. Schema-loss maintenance restore also applies current migrations behind its busy gate.

1. `hierarchy-profiles` displays public supported kinds, categories, relations, axes, namespaces and bounds.
2. Prepare a trusted native inventory manifest. The [synthetic six-provider fixture](../../../tests/fixtures/focus/native-hierarchy/six-provider.json) documents its exact schema. Supply provider/partition/kind/namespace/native ID, labels/aliases, directed typed edges and whole-interval source bindings with explicit observation/evidence class.
3. Run `hierarchy-plan --actor operator --file MANIFEST`. Bindings must match existing provider/account/subaccount ownership exactly. A manifest cannot add tenants, broaden source assignments or derive permission from a shared payer. Run without `--file` to normalize observed FOCUS scopes using the current catalog, which is empty on initial adoption.
4. Run `approve --actor lifecycle-approver --manifest-digest DIGEST`, then `hierarchy-apply --actor operator --manifest-digest DIGEST`. Registry, configuration, predecessor catalog, retained slice/head set or lifecycle changes invalidate a stale plan. The requesting operator cannot self-approve.

The graph source manifest is coordinator-private. Readers receive only persisted projections for their exact source assignments and permitted ancestors. Nodes, edges, role references, aliases, counts and partial-validity warnings are scoped before exposure; no sibling/descendant discovery follows from a shared root. Catalog revision changes apply to future intakes. Already typed historical heads stay frozen. Explicit approved supplemental enrichment may replace an incomplete FOCUS-only head while retaining its original intake snapshot/digest and showing the later observation. Retroactive reparenting of an already typed snapshot requires a source correction/new revision.

A native binding and each traversed edge must cover the full charge interval. Inventory observed_at is not historical valid_from. Partial/unknown intervals remain unresolved; the code neither chooses the latest parent nor invents proration. Single-parent/cardinality and cycle rules apply to their declared relation/axis, not to unrelated billing/resource/identity graphs.

# Query and group

The periodic `query` response now includes `account_context` on each returned fact and `account_hierarchies` with the authorized normalized graph and its pinned public profile. Existing provider/service/window filters run before graph selection. `--hierarchy-node UUID` restricts facts to those whose authorized whole-interval closure includes that node; it does not reveal whether an inaccessible node exists elsewhere.

Examples after approved source normalization:

```sh
tests/engine_probes/.venv/bin/python tools/focus_lifecycle.py query --actor reader-company-north --group-kind role --group-axis binding --group-role billing
tests/engine_probes/.venv/bin/python tools/focus_lifecycle.py query --actor reader-company-north --provider GCP --group-kind organization --group-axis resource --group-role usage
```

Direct role grouping works across distinct native types and falls back to observed FOCUS billing/usage scopes when a native role is unknown. Native-kind or `category:NAME` ancestor grouping follows one requested axis from one requested role; it does not silently bridge management, organization and payment paths. Missing/partial candidates use an explicit unresolved bucket. Multiple applicable ancestors reject additive grouping, including multiple VM networks/datastores. No allocation weights are invented. Each fact is counted once; currency/unit groups and NULLs remain separate. Existing10000-row/full-selection exact bounds still apply.

Periodic pagination uses the normal `--snapshot` and composite cursor, which now also pin selected hierarchy digests and grouping/filter options. Legacy single-batch pagination additionally requires `--expected-hierarchy` from the first response alongside `--expected-batch`; a changed hierarchy rejects continuation. The legacy command still rejects periodic history with `period_query_required`.

# Archive, recovery and final boundaries

Archive/recall preserves native bindings because graph metadata is independent of CH projection/tier. All six normalized table families and pinned profiles join metadata backup/schema-loss restore and source-retirement inventory. Latest holds/revocations/deletions still apply before any restored graph or fact is served. Deleted slices are not enriched or revived. Missing/tampered profiles and unresolved operations fail closed. Caller-owned manifest files remain outside managed deletion, like other input files; shared journal/control payload residuals stay explicit until governed source retirement.

The current public1123-row sample lacks authoritative native organization/identity inventory. Its actual FOCUS invoice/usage IDs are now persisted in normalized scoped form; native parent/role coverage stays FOCUS-only. Full six-provider native shapes are tested separately with deliberately synthetic metadata, not added to public rows as fabricated source facts. Native API collection, production authentication and scale remain outside this trusted single-host implementation.
