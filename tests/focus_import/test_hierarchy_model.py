import copy
import csv
import io
import json
from datetime import timedelta
from fractions import Fraction

import pytest

from modules.focus_import import hierarchy_model as hierarchy
from modules.focus_import.model import ADDITIVE, ROOT, Rejected, assign, canonical, decode, decimal, digest, domain, domains_digest, tenant_id, versions


RAW = {'aws': 'AWS', 'azure': 'Microsoft', 'gcp': 'Google Cloud', 'oci': 'Oracle', 'vmware': 'VMware', 'alibaba': 'Alibaba Cloud'}


@pytest.fixture
def catalog():
    return json.loads((ROOT / 'tests/fixtures/focus/native-hierarchy/six-provider.json').read_text())


def input_rows(doc):
    sources = []
    for index, binding in enumerate(doc['bindings'], 1):
        cost = '999' if binding['tenant_key'] == 'foreign' else f'{index}.000000000000000001'
        row = {'provider': RAW[binding['provider']], 'service': 'Synthetic compute', 'currency': 'CNY' if binding['provider'] == 'alibaba' else 'USD', 'account': binding['account'], 'subaccount': binding['subaccount'], 'consumed_unit': 'Hours', 'pricing_unit': 'Hours', 'billing_start': '2024-09-01T00:00:00Z', 'billing_end': '2024-10-01T00:00:00Z', 'charge_start': '2024-09-01T00:00:00Z', 'charge_end': '2024-09-02T00:00:00Z', 'billed_cost': cost, 'effective_cost': cost, 'list_cost': cost, 'contracted_cost': '', 'consumed_quantity': '2', 'pricing_quantity': '2', 'list_unit_price': '0.5', 'contracted_unit_price': ''}
        sources.append(row)
    columns = versions()['1.0']['columns']
    stream = io.StringIO(newline='')
    writer = csv.DictWriter(stream, fieldnames=list(columns.values()))
    writer.writeheader()
    for row in sources:
        writer.writerow({columns[k]: v for k, v in row.items()})
    return stream.getvalue().encode()


def configured(doc):
    data = input_rows(doc)
    rows = decode(data, versions()['1.0'])
    tenants = []
    for key in ['north-a', 'north-b', 'foreign']:
        owned = {canonical(domain(row)) for row, binding in zip(rows, doc['bindings']) if binding['tenant_key'] == key}
        tenants.append({'key': key, 'label': 'Synthetic ' + key, 'company_id': 'foreign-company' if key == 'foreign' else 'north-company', 'company_label': 'Same label', 'actor': 'reader-' + key, 'timezone': 'UTC', 'reporting_currency': 'USD', 'currencies': ['USD', 'CNY', None], 'retention_days': 30, 'active': True, 'assignments': [json.loads(v) for v in sorted(owned)]})
    config = {'source': 'synthetic-hierarchy', 'version': '1.0', 'revision': 1, 'snapshot': {'file_digest': digest(data), 'row_count': len(rows), 'domains_digest': domains_digest(rows)}, 'tenants': tenants}
    return config, assign(rows, config)


def fixture_context(doc, provider=None, tenant='north-a'):
    config, rows = configured(doc)
    hierarchy.validate_catalog(doc, config)
    selected = [r for r in rows if r['tenant_id'] == tenant_id(tenant) and (provider is None or hierarchy.provider_key(r['provider']) == provider)]
    return selected, hierarchy.snapshot(selected, tenant, config['source'], doc)


def node(doc, key):
    return next(n for n in doc['nodes'] if n['key'] == key)


def edge(doc, child, relation=None):
    return next(e for e in doc['edges'] if e['child'] == child and (relation is None or e['relation'] == relation))


def new_node(doc, key, provider, kind, native_id, namespace='global'):
    original = next(n for n in doc['nodes'] if n['provider'] == provider)
    item = {'key': key, 'provider': provider, 'partition': original['partition'], 'namespace': namespace, 'kind': kind, 'native_id': native_id, 'label': 'Synthetic ' + key, 'aliases': []}
    doc['nodes'].append(item)
    return item


def new_edge(doc, child, parent, relation):
    item = {'child': child, 'parent': parent, 'relation': relation, 'valid_from': '2024-01-01T00:00:00Z', 'valid_to': '2025-01-01T00:00:00Z'}
    doc['edges'].append(item)
    return item


def single_group(result):
    assert result['additive'] is True
    assert result['unresolved_rows'] == 0
    assert len(result['groups']) == 1
    return next(iter(result['groups']))


def test_registry_has_six_closed_profiles():
    reg = hierarchy.registry()
    assert set(reg['providers']) == {'aws', 'azure', 'gcp', 'oci', 'vmware', 'alibaba'}
    assert reg['providers']['aws']['kinds']['organizational_unit']['namespace'] == 'organization'
    assert reg['providers']['vmware']['kinds']['virtual_machine']['namespace'] == 'service-instance'
    assert reg['providers']['alibaba']['kinds']['resource_group']['namespace'] == 'account'
    assert len(hierarchy.registry_digest()) == 64
    with pytest.raises(Rejected, match='hierarchy_provider'):
        hierarchy.provider_key('Unrecognized native CSP')


def test_agreement_and_vendor_variant_relations_are_typed(catalog):
    config, _ = configured(catalog)
    new_node(catalog, 'azure-department', 'azure', 'department', 'synthetic-ea-department', 'billing-account:synthetic-mca-account')
    new_node(catalog, 'azure-ea-account', 'azure', 'enrollment_account', 'synthetic-ea-account', 'billing-account:synthetic-mca-account')
    new_node(catalog, 'azure-csp-customer', 'azure', 'csp_customer', 'synthetic-csp-customer', 'billing-account:synthetic-mca-account')
    new_edge(catalog, 'azure-department', 'azure-billing', 'department_of')
    new_edge(catalog, 'azure-ea-account', 'azure-department', 'enrollment_account_of')
    new_edge(catalog, 'azure-csp-customer', 'azure-billing', 'customer_of')
    new_node(catalog, 'oci-child-tenancy', 'oci', 'tenancy', 'ocid1.tenancy.oc1..syntheticchild')
    new_edge(catalog, 'oci-child-tenancy', 'oci-tenancy', 'child_tenancy_of')
    new_node(catalog, 'gcp-sub-billing', 'gcp', 'billing_account', 'DDDDDD-EEEEEE-FFFFFF')
    new_edge(catalog, 'gcp-sub-billing', 'gcp-billing', 'sub_billing_account_of')
    for kind in ['ram_role', 'ram_group']:
        new_node(catalog, 'ali-' + kind, 'alibaba', kind, 'synthetic-' + kind, 'account:100000000002')
        new_edge(catalog, 'ali-' + kind, 'ali-member', 'identity_of')
    ns = node(catalog, 'vm-machine')['namespace']
    new_node(catalog, 'vm-compute-resource', 'vmware', 'compute_resource', 'domain-c8', ns)
    new_node(catalog, 'vm-vapp', 'vmware', 'vapp', 'resgroup-vapp', ns)
    new_edge(catalog, 'vm-compute-resource', 'vm-dc', 'compute_resource_in')
    new_edge(catalog, 'vm-vapp', 'vm-compute-resource', 'vapp_in')
    hierarchy.validate_catalog(catalog, config)


def test_six_provider_snapshots_and_unknown_binding_are_explicit(catalog):
    rows, body = fixture_context(catalog)
    assert len(rows) == 7
    assert body['status'] == 'mixed'
    assert {n['provider'] for n in body['nodes']} == set(RAW)
    assert sum(bool(b['native_roles']) for b in body['bindings']) == 6
    assert all(b['focus_billing'] != b['focus_usage'] for b in body['bindings'])
    unknown = next(b for b in body['bindings'] if not b['native_roles'])
    assert unknown['status'] == 'focus-only'
    assert 'organization' in unknown['unresolved_axes']
    assert body['observed_at'] == catalog['observed_at']
    assert json.loads(canonical(body)) == body


def test_focus_only_does_not_infer_native_ids_or_unknown_provider(catalog):
    _, rows = configured(catalog)
    rows = [r for r in rows if r['tenant_id'] == tenant_id('north-a')]
    body = hierarchy.snapshot(rows, 'north-a', 'synthetic-hierarchy')
    assert body['status'] == 'focus-only'
    assert body['catalog_digest'] is None
    assert {n['kind'] for n in body['nodes']} == {'focus_billing_scope', 'focus_usage_scope'}
    assert all(not b['native_roles'] for b in body['bindings'])
    unknown = copy.deepcopy(rows[:1])
    unknown[0]['provider'] = 'Future FOCUS source'
    body = hierarchy.snapshot(unknown, 'north-a', 'synthetic-hierarchy')
    assert {n['provider'] for n in body['nodes']} == {'unknown'}
    assert {n['raw_provider'] for n in body['nodes']} == {'Future FOCUS source'}


def test_shared_payer_never_exposes_sibling_and_manager_is_explicit(catalog):
    rows, body = fixture_context(catalog, 'aws')
    ids = {n['id'] for n in body['nodes']}
    assert hierarchy.node_id(node(catalog, 'aws-payer')) in ids
    assert hierarchy.node_id(node(catalog, 'aws-usage-a')) in ids
    assert hierarchy.node_id(node(catalog, 'aws-usage-b')) not in ids
    assert hierarchy.node_id(node(catalog, 'aws-foreign')) not in ids
    assert hierarchy.node_id(node(catalog, 'aws-manager')) in ids
    valid = [r for r in rows if r['subaccount'] == 'synthetic-aws-usage']
    result = hierarchy.group_rows(valid, body, 'organization', 'organization')
    assert single_group(result) == hierarchy.node_id(node(catalog, 'aws-org'))


@pytest.mark.parametrize('change', ['tenant', 'account', 'role-provider', 'unknown-kind', 'unknown-provider', 'namespace', 'cross-edge'])
def test_catalog_cannot_widen_or_confuse_authority(catalog, change):
    config, _ = configured(catalog)
    if change == 'tenant':
        catalog['bindings'][0]['tenant_key'] = 'foreign'
    elif change == 'account':
        catalog['bindings'][0]['account'] = 'unassigned-account'
    elif change == 'role-provider':
        catalog['bindings'][0]['roles']['usage'] = 'gcp-project'
    elif change == 'unknown-kind':
        catalog['nodes'][0]['kind'] = 'universal-cloud-account'
    elif change == 'unknown-provider':
        catalog['nodes'][0]['provider'] = 'future-native'
    elif change == 'namespace':
        node(catalog, 'aws-ou-north')['namespace'] = 'organization:o-different'
    else:
        edge(catalog, 'aws-usage-a')['parent'] = 'gcp-folder'
    with pytest.raises(Rejected):
        hierarchy.validate_catalog(catalog, config)


def test_partial_roles_and_vmware_no_native_billing(catalog):
    rows, body = fixture_context(catalog, 'vmware')
    assert body['status'] == 'typed-native'
    binding = body['bindings'][0]
    assert set(binding['native_roles']) == {'usage', 'resource'}
    assert not any(n['kind'] == 'billing_account' for n in body['nodes'])
    result = hierarchy.group_rows(rows, body, 'billing_account', 'billing', 'billing')
    assert result['unresolved_rows'] == 1
    assert set(result['groups']) == {'unresolved'}
    azure = next(b for b in catalog['bindings'] if b['provider'] == 'azure')
    azure['roles'] = {'identity': 'azure-directory'}
    rows, body = fixture_context(catalog, 'azure')
    assert body['status'] == 'typed-native'
    assert set(body['bindings'][0]['native_roles']) == {'identity'}
    assert hierarchy.group_rows(rows, body, 'subscription', 'resource')['unresolved_rows'] == 1


def test_vmware_multiple_attachments_reject_cost_multiplication(catalog):
    rows, body = fixture_context(catalog, 'vmware')
    for kind in ['network', 'datastore']:
        with pytest.raises(Rejected, match='hierarchy_nonadditive_ancestors'):
            hierarchy.group_rows(rows, body, kind, 'attachment')
    result = hierarchy.group_rows(rows, body, 'cluster', 'placement')
    assert single_group(result) == hierarchy.node_id(node(catalog, 'vm-cluster'))
    assert sum(g['rows'] for g in next(iter(result['groups'].values())).values()) == 1


def test_gcp_resource_and_billing_organization_are_independent(catalog):
    rows, body = fixture_context(catalog, 'gcp')
    resource = single_group(hierarchy.group_rows(rows, body, 'organization', 'resource'))
    billing = single_group(hierarchy.group_rows(rows, body, 'organization', 'billing'))
    assert resource == hierarchy.node_id(node(catalog, 'gcp-resource-org'))
    assert billing == hierarchy.node_id(node(catalog, 'gcp-billing-org'))
    assert resource != billing
    project = next(n for n in body['nodes'] if n['kind'] == 'project')
    assert project['native_id'] == 'synthetic-project-a'
    assert {'kind': 'project_number', 'value': '300000000001'} in project['aliases']


def test_azure_invoice_profile_resource_and_directory_stay_distinct(catalog):
    rows, body = fixture_context(catalog, 'azure')
    binding = body['bindings'][0]
    invoice = next(n for n in body['nodes'] if n['id'] == binding['focus_billing'])
    assert invoice['kind'] == 'focus_billing_scope'
    assert invoice['native_id'] == 'synthetic-azure-profile'
    assert single_group(hierarchy.group_rows(rows, body, 'billing_account', 'billing', 'billing')) == hierarchy.node_id(node(catalog, 'azure-billing'))
    assert single_group(hierarchy.group_rows(rows, body, 'entra_tenant', 'identity')) == hierarchy.node_id(node(catalog, 'azure-directory'))
    assert hierarchy.group_rows(rows, body, 'management_group', 'resource')['unresolved_rows'] == 1


def test_oci_compartment_subscription_and_identity_domains_are_separate(catalog):
    rows, body = fixture_context(catalog, 'oci')
    assert single_group(hierarchy.group_rows(rows, body, 'tenancy', 'resource')) == hierarchy.node_id(node(catalog, 'oci-tenancy'))
    assert single_group(hierarchy.group_rows(rows, body, 'subscription', 'billing', 'billing')) == hierarchy.node_id(node(catalog, 'oci-subscription'))
    assert single_group(hierarchy.group_rows(rows, body, 'tenancy', 'identity', 'identity')) == hierarchy.node_id(node(catalog, 'oci-tenancy'))
    assert hierarchy.group_rows(rows, body, 'subscription', 'billing')['unresolved_rows'] == 1


def test_alibaba_management_account_is_not_invented_member(catalog):
    rows, body = fixture_context(catalog, 'alibaba')
    assert not any(e['child'] == 'ali-manager' and e['relation'] == 'member_of' for e in catalog['edges'])
    assert hierarchy.node_id(node(catalog, 'ali-manager')) in {n['id'] for n in body['nodes']}
    assert single_group(hierarchy.group_rows(rows, body, 'resource_directory', 'organization')) == hierarchy.node_id(node(catalog, 'ali-directory'))


def test_identity_ignores_labels_parents_and_collector(catalog):
    rows, before = fixture_context(catalog, 'gcp')
    original = hierarchy.node_id(node(catalog, 'gcp-project'))
    node(catalog, 'gcp-project')['label'] = 'New display label'
    node(catalog, 'gcp-project')['aliases'].append({'kind': 'display_name', 'value': 'renamed'})
    catalog['revision'] += 1
    catalog['evidence_ref'] = 'Another authorized synthetic collector'
    new_node(catalog, 'gcp-new-folder', 'gcp', 'folder', '200000000002')
    edge(catalog, 'gcp-project')['parent'] = 'gcp-new-folder'
    new_edge(catalog, 'gcp-new-folder', 'gcp-resource-org', 'contained_by')
    _, after = fixture_context(catalog, 'gcp')
    assert hierarchy.node_id(node(catalog, 'gcp-project')) == original
    assert next(n for n in before['nodes'] if n['id'] == original)['label'] != next(n for n in after['nodes'] if n['id'] == original)['label']
    assert before['catalog_digest'] != after['catalog_digest']
    assert next(e for e in before['edges'] if e['child'] == original and e['axis'] == 'resource')['parent'] != next(e for e in after['edges'] if e['child'] == original and e['axis'] == 'resource')['parent']


def test_vmware_moref_namespace_prevents_vcenter_collision(catalog):
    first = node(catalog, 'vm-machine')
    second = copy.deepcopy(first)
    second['key'] = 'vm-in-another-center'
    second['namespace'] = 'service-instance:aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee'
    catalog['nodes'].append(second)
    config, _ = configured(catalog)
    hierarchy.validate_catalog(catalog, config)
    assert first['native_id'] == second['native_id']
    assert hierarchy.node_id(first) != hierarchy.node_id(second)
    new_edge(catalog, 'vm-in-another-center', 'vm-folder', 'inventory_parent')
    with pytest.raises(Rejected, match='hierarchy_namespace_edge'):
        hierarchy.validate_catalog(catalog, config)


@pytest.mark.parametrize('change', ['aws-root-organization', 'aws-ou-root', 'vmware-instance', 'azure-management-group', 'azure-directory'])
def test_scoped_containment_requires_same_authoritative_namespace(catalog, change):
    config, _ = configured(catalog)
    if change == 'aws-root-organization':
        new_node(catalog, 'aws-other-org', 'aws', 'organization', 'o-otherorganization')
        edge(catalog, 'aws-root', 'root_of')['parent'] = 'aws-other-org'
    elif change == 'aws-ou-root':
        new_node(catalog, 'aws-other-root', 'aws', 'root', 'r-otherroot', 'organization:o-otherorganization')
        edge(catalog, 'aws-ou-north', 'contained_by')['parent'] = 'aws-other-root'
    elif change == 'vmware-instance':
        new_node(catalog, 'vm-other-instance', 'vmware', 'service_instance', 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee')
        edge(catalog, 'vm-dc', 'inventory_parent')['parent'] = 'vm-other-instance'
    elif change == 'azure-management-group':
        new_node(catalog, 'azure-other-mg', 'azure', 'management_group', 'other-mg', 'directory:00000000-0000-0000-0000-000000000020')
        new_edge(catalog, 'azure-mg', 'azure-other-mg', 'governed_by')
    else:
        new_node(catalog, 'azure-other-directory', 'azure', 'entra_tenant', '00000000-0000-0000-0000-000000000020')
        edge(catalog, 'azure-mg', 'directory_of')['parent'] = 'azure-other-directory'
    with pytest.raises(Rejected, match='hierarchy_namespace_edge'):
        hierarchy.validate_catalog(catalog, config)


def test_alias_collision_rejected_without_alias_identity_equivalence(catalog):
    config, _ = configured(catalog)
    other = new_node(catalog, 'gcp-other-project', 'gcp', 'project', 'synthetic-project-b')
    other['aliases'] = [{'kind': 'project_number', 'value': '300000000001'}]
    with pytest.raises(Rejected, match='hierarchy_alias_collision'):
        hierarchy.validate_catalog(catalog, config)
    other['aliases'][0]['value'] = '300000000002'
    hierarchy.validate_catalog(catalog, config)
    assert hierarchy.node_id(other) != hierarchy.node_id(node(catalog, 'gcp-project'))


@pytest.mark.parametrize('unknown', [True, False])
def test_edge_unknown_or_partial_time_is_not_observation_time(catalog, unknown):
    relation = edge(catalog, 'gcp-project', 'contained_by')
    relation['valid_from'] = None if unknown else '2024-09-01T12:00:00Z'
    rows, body = fixture_context(catalog, 'gcp')
    assert body['status'] == 'mixed'
    assert hierarchy.node_id(node(catalog, 'gcp-folder')) not in {n['id'] for n in body['nodes']}
    result = hierarchy.group_rows(rows, body, 'organization', 'resource')
    assert result['unresolved_rows'] == 1
    assert body['observed_at'].startswith('2026')


def test_binding_must_cover_entire_charge_interval(catalog):
    binding = next(b for b in catalog['bindings'] if b['provider'] == 'gcp')
    binding['valid_from'] = '2024-09-01T12:00:00Z'
    rows, body = fixture_context(catalog, 'gcp')
    assert not body['bindings'][0]['native_roles']
    assert body['status'] == 'focus-only'
    assert body['evidence_class'] == 'focus-observed'
    assert body['observed_at'] is None
    assert hierarchy.group_rows(rows, body, 'project', 'resource')['unresolved_rows'] == 1


def test_disjoint_valid_time_move_and_boundary(catalog):
    old = edge(catalog, 'gcp-project', 'contained_by')
    old['valid_to'] = '2024-09-02T00:00:00Z'
    new_node(catalog, 'gcp-new-folder', 'gcp', 'folder', '200000000002')
    new_edge(catalog, 'gcp-new-folder', 'gcp-billing-org', 'contained_by')
    moved = new_edge(catalog, 'gcp-project', 'gcp-new-folder', 'contained_by')
    moved['valid_from'] = old['valid_to']
    rows, before = fixture_context(catalog, 'gcp')
    assert single_group(hierarchy.group_rows(rows, before, 'organization', 'resource')) == hierarchy.node_id(node(catalog, 'gcp-resource-org'))
    later = copy.deepcopy(rows)
    for row in later:
        row['charge_start'] += timedelta(days=1)
        row['charge_end'] += timedelta(days=1)
    after = hierarchy.snapshot(later, 'north-a', 'synthetic-hierarchy', catalog)
    assert single_group(hierarchy.group_rows(later, after, 'organization', 'resource')) == hierarchy.node_id(node(catalog, 'gcp-billing-org'))
    spanning = copy.deepcopy(rows)
    spanning[0]['charge_end'] += timedelta(days=1)
    partial = hierarchy.snapshot(spanning, 'north-a', 'synthetic-hierarchy', catalog)
    assert hierarchy.group_rows(spanning, partial, 'organization', 'resource')['unresolved_rows'] == 1


@pytest.mark.parametrize('change', ['cardinality', 'cycle', 'binding-overlap', 'dangling', 'relation', 'duplicate-alias', 'empty-roles', 'naive-validity'])
def test_graph_and_binding_constraints(catalog, change):
    config, _ = configured(catalog)
    if change == 'cardinality':
        new_edge(catalog, 'gcp-project', 'gcp-resource-org', 'contained_by')
    elif change == 'cycle':
        new_node(catalog, 'gcp-folder-cycle', 'gcp', 'folder', '200000000003')
        edge(catalog, 'gcp-folder')['parent'] = 'gcp-folder-cycle'
        new_edge(catalog, 'gcp-folder-cycle', 'gcp-folder', 'contained_by')
    elif change == 'binding-overlap':
        catalog['bindings'].append(copy.deepcopy(catalog['bindings'][0]))
    elif change == 'dangling':
        catalog['edges'][0]['parent'] = 'missing'
    elif change == 'relation':
        catalog['edges'][0]['relation'] = 'universal_parent'
    elif change == 'duplicate-alias':
        project = node(catalog, 'gcp-project')
        project['aliases'].append(copy.deepcopy(project['aliases'][0]))
    elif change == 'empty-roles':
        catalog['bindings'][0]['roles'] = {}
    else:
        catalog['edges'][0]['valid_from'] = '2024-01-01T00:00:00'
    with pytest.raises(Rejected):
        hierarchy.validate_catalog(catalog, config)


def test_cross_axis_relationships_do_not_form_universal_tree_cycle(catalog):
    config, _ = configured(catalog)
    new_edge(catalog, 'aws-manager', 'aws-org', 'member_of')
    hierarchy.validate_catalog(catalog, config)
    assert edge(catalog, 'aws-org', 'managed_by')['parent'] == 'aws-manager'


def test_filtered_projection_only_returns_selected_closure(catalog):
    rows, body = fixture_context(catalog)
    azure = next(r for r in rows if r['provider'] == 'Microsoft')
    filtered = hierarchy.select(body, [azure['ordinal']])
    assert filtered['status'] == 'typed-native'
    assert filtered['evidence_class'] == 'synthetic'
    assert {n['provider'] for n in filtered['nodes']} == {'azure'}
    assert len(filtered['bindings']) == 1
    assert all(e['child'] in {n['id'] for n in filtered['nodes']} and e['parent'] in {n['id'] for n in filtered['nodes']} for e in filtered['edges'])
    missing = next(b['ordinal'] for b in body['bindings'] if not b['native_roles'])
    focus_only = hierarchy.select(body, [missing])
    assert focus_only['status'] == 'focus-only'
    assert focus_only['evidence_class'] == 'focus-observed'
    assert focus_only['observed_at'] is None
    assert focus_only['catalog_digest'] == body['catalog_digest']
    empty = hierarchy.select(body, [])
    assert empty['nodes'] == empty['edges'] == empty['bindings'] == []
    with pytest.raises(Rejected, match='hierarchy_selection'):
        hierarchy.select(body, [999])


def test_group_controls_match_fraction_oracle_with_nulls(catalog):
    rows, body = fixture_context(catalog)
    result = hierarchy.group_rows(rows, body, 'category:usage_scope', 'focus')
    for measure in ADDITIVE:
        expected = sum((Fraction(r[measure]) for r in rows if r[measure] is not None), Fraction(0))
        actual = sum((Fraction(g['measures'][measure]['coefficient'] or 0, 10**18) for groups in result['groups'].values() for g in groups.values()), Fraction(0))
        assert expected == actual
        assert sum(g['measures'][measure]['missing'] for groups in result['groups'].values() for g in groups.values()) == sum(r[measure] is None for r in rows)
    assert sum(g['rows'] for groups in result['groups'].values() for g in groups.values()) == len(rows)
    assert all('list_unit_price' not in g['measures'] for groups in result['groups'].values() for g in groups.values())


@pytest.mark.parametrize('role', ['billing', 'usage', 'resource', 'identity'])
def test_direct_role_grouping_preserves_native_kind_and_exact_totals(catalog, role):
    rows, body = fixture_context(catalog)
    result = hierarchy.group_rows(rows, body, 'role', 'binding', role)
    assert sum(g['rows'] for groups in result['groups'].values() for g in groups.values()) == len(rows)
    expected = sum((Fraction(r['billed_cost']) for r in rows), Fraction(0))
    observed = sum((Fraction(g['measures']['billed_cost']['coefficient'], 10**18) for groups in result['groups'].values() for g in groups.values()), Fraction(0))
    assert expected == observed
    assert result['unresolved_rows'] == (0 if role in ['billing', 'usage'] else sum(role not in b['native_roles'] for b in body['bindings']))
    if role == 'billing':
        grouped_kinds = {n['kind'] for n in body['nodes'] if n['id'] in result['groups']}
        assert {'account', 'billing_profile', 'subscription', 'billing_account', 'focus_billing_scope'} <= grouped_kinds


@pytest.mark.parametrize('change', ['tenant', 'account', 'time', 'duplicate'])
def test_group_cannot_rebind_an_occurrence(catalog, change):
    rows, body = fixture_context(catalog, 'gcp')
    altered = copy.deepcopy(rows)
    if change == 'tenant':
        altered[0]['tenant_id'] = tenant_id('foreign')
    elif change == 'account':
        altered[0]['account'] = 'different-account'
    elif change == 'time':
        altered[0]['charge_start'] += timedelta(minutes=1)
    else:
        altered += copy.deepcopy(altered)
    with pytest.raises(Rejected):
        hierarchy.group_rows(altered, body, 'project', 'resource')


def test_bounds_prevent_unbounded_graph_or_aggregate(catalog):
    config, rows = configured(catalog)
    deep = copy.deepcopy(catalog)
    previous = 'gcp-folder'
    for index in range(35):
        key = 'gcp-deep-' + str(index)
        new_node(deep, key, 'gcp', 'folder', str(900000000000 + index))
        new_edge(deep, key, previous, 'contained_by')
        previous = key
    with pytest.raises(Rejected, match='hierarchy_depth'):
        hierarchy.validate_catalog(deep, config)
    huge = copy.deepcopy(catalog)
    huge['nodes'] *= 30
    with pytest.raises(Rejected, match='hierarchy_catalog_bound'):
        hierarchy.validate_catalog(huge, config)
    rows, body = fixture_context(catalog, 'gcp')
    rows[0]['billed_cost'] = decimal(10**38)
    with pytest.raises(Rejected, match='selection_aggregate_bound'):
        hierarchy.group_rows(rows, body, 'project', 'resource')


@pytest.mark.parametrize('change', ['kind', 'axis', 'category', 'bounds'])
def test_retained_grouping_uses_pinned_profile_after_current_registry_changes(catalog, monkeypatch, tmp_path, change):
    rows, body = fixture_context(catalog, 'gcp')
    requests = [('project', 'resource'), ('organization', 'resource'), ('category:usage_scope', 'resource')]
    expected = [hierarchy.group_rows(rows, body, kind, axis) for kind, axis in requests]
    original_profile = copy.deepcopy(body['profile'])
    current = hierarchy.registry()
    if change == 'kind':
        profile = current['providers']['gcp']
        del profile['kinds']['project']
        for relation in profile['relations']:
            for side in ['children', 'parents']:
                relation[side] = [kind for kind in relation[side] if kind != 'project']
        profile['relations'] = [relation for relation in profile['relations'] if relation['children'] and relation['parents']]
        profile['roles'] = {role: [kind for kind in kinds if kind != 'project'] for role, kinds in profile['roles'].items()}
    elif change == 'axis':
        current['axes'].remove('resource')
        for profile in current['providers'].values():
            profile['relations'] = [relation for relation in profile['relations'] if relation['axis'] != 'resource']
    elif change == 'category':
        current['categories'].remove('usage_scope')
        for profile in current['providers'].values():
            for kind in profile['kinds'].values():
                if kind['category'] == 'usage_scope':
                    kind['category'] = 'account'
    else:
        current.update(max_bindings=1, max_nodes=1, max_edges=1, max_depth=1)
    path = tmp_path / 'current-native-hierarchy.json'
    path.write_text(canonical(current))
    monkeypatch.setattr(hierarchy, 'REGISTRY', path)
    assert hierarchy.registry() == current
    assert hierarchy.registry_digest() != body['registry_digest']
    assert hierarchy.pinned_profile(body) == original_profile
    assert [hierarchy.group_rows(rows, body, kind, axis) for kind, axis in requests] == expected
    selected = hierarchy.select(body, [rows[0]['ordinal']])
    assert selected['profile'] == original_profile
    assert selected['registry_digest'] == body['registry_digest']
    assert hierarchy.snapshot(rows, 'north-a', 'synthetic-hierarchy')['profile'] == current


def test_retained_profile_read_survives_current_registry_unavailability(catalog, monkeypatch, tmp_path):
    rows, body = fixture_context(catalog, 'gcp')
    expected = hierarchy.group_rows(rows, body, 'organization', 'resource')
    monkeypatch.setattr(hierarchy, 'REGISTRY', tmp_path / 'unavailable-profile.json')
    assert hierarchy.group_rows(rows, body, 'organization', 'resource') == expected
    assert hierarchy.select(body, [rows[0]['ordinal']])['profile'] == body['profile']
    with pytest.raises(Rejected, match='hierarchy_registry'):
        hierarchy.snapshot(rows, 'north-a', 'synthetic-hierarchy')


@pytest.mark.parametrize('consumer', ['read', 'select', 'group'])
@pytest.mark.parametrize('change', ['profile', 'digest', 'missing'])
def test_pinned_profile_tampering_rejected_before_reads(catalog, consumer, change):
    rows, body = fixture_context(catalog, 'gcp')
    if change == 'profile':
        body['profile']['max_depth'] += 1
    elif change == 'digest':
        body['registry_digest'] = '0' * 64
    else:
        del body['profile']
    with pytest.raises(Rejected, match='hierarchy_profile_digest'):
        if consumer == 'read':
            hierarchy.pinned_profile(body)
        elif consumer == 'select':
            hierarchy.select(body, [rows[0]['ordinal']])
        else:
            hierarchy.group_rows(rows, body, 'project', 'resource')
