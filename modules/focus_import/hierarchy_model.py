import copy
import json
import re
from collections import defaultdict
from datetime import datetime, timezone
from uuid import UUID, uuid5

from .model import ADDITIVE, INSTALLATION, LIMIT, MAX_ROWS, ROOT, Rejected, canonical, coefficient, controls, digest, tenant_id
from .temporal import utc


REGISTRY = ROOT / 'packages/contracts/focus_import/native_hierarchy.json'
PROVIDERS = {'aws', 'azure', 'gcp', 'oci', 'vmware', 'alibaba'}
ALIASES = {'AWS': 'aws', 'Amazon Web Services': 'aws', 'Microsoft': 'azure', 'Azure': 'azure', 'Microsoft Azure': 'azure', 'Google Cloud': 'gcp', 'GCP': 'gcp', 'Oracle': 'oci', 'OCI': 'oci', 'VMware': 'vmware', 'Alibaba Cloud': 'alibaba', 'Alibaba': 'alibaba'}
NODE_FIELDS = {'key', 'provider', 'partition', 'namespace', 'kind', 'native_id', 'label', 'aliases'}
EDGE_FIELDS = {'child', 'parent', 'relation', 'valid_from', 'valid_to'}
BINDING_FIELDS = {'provider', 'account', 'subaccount', 'tenant_key', 'valid_from', 'valid_to', 'roles'}


def text(value, maximum=4096):
    return isinstance(value, str) and 0 < len(value) <= maximum and not any(ord(c) < 32 for c in value)


def validate_profile(result):
    try:
        expected = {'schema_version', 'max_nodes', 'max_edges', 'max_bindings', 'max_depth', 'max_bytes', 'categories', 'axes', 'providers'}
        if not isinstance(result, dict) or set(result) != expected or type(result['schema_version']) is not int or result['schema_version'] != 1 or set(result['providers']) != PROVIDERS:
            raise Rejected('hierarchy_registry')
        for key in ['max_nodes', 'max_edges', 'max_bindings', 'max_depth', 'max_bytes']:
            if type(result[key]) is not int or not 0 < result[key] <= 1048576:
                raise Rejected('hierarchy_registry')
        for key in ['categories', 'axes']:
            if not isinstance(result[key], list) or not result[key] or len(set(result[key])) != len(result[key]) or any(not text(v, 64) for v in result[key]):
                raise Rejected('hierarchy_registry')
        for profile in result['providers'].values():
            if set(profile) != {'kinds', 'relations', 'roles'} or not profile['kinds']:
                raise Rejected('hierarchy_registry')
            for kind, spec in profile['kinds'].items():
                if not text(kind, 64) or set(spec) != {'category', 'namespace'} or spec['category'] not in result['categories'] or spec['namespace'] not in ['global', 'organization', 'directory', 'billing-account', 'project', 'tenancy', 'account', 'service-instance']:
                    raise Rejected('hierarchy_registry')
            seen = set()
            for relation in profile['relations']:
                if set(relation) != {'relation', 'axis', 'children', 'parents', 'max_parents'} or not text(relation['relation'], 64) or relation['relation'] in seen or relation['axis'] not in result['axes']:
                    raise Rejected('hierarchy_registry')
                seen.add(relation['relation'])
                for side in ['children', 'parents']:
                    if not relation[side] or any(k not in profile['kinds'] for k in relation[side]):
                        raise Rejected('hierarchy_registry')
                if relation['max_parents'] is not None and (type(relation['max_parents']) is not int or relation['max_parents'] != 1):
                    raise Rejected('hierarchy_registry')
            if set(profile['roles']) != {'billing', 'usage', 'resource', 'identity'} or any(not isinstance(v, list) or any(k not in profile['kinds'] for k in v) for v in profile['roles'].values()):
                raise Rejected('hierarchy_registry')
        return result
    except (OSError, ValueError, TypeError, KeyError):
        raise Rejected('hierarchy_registry') from None


def registry():
    try:
        return validate_profile(json.loads(REGISTRY.read_text()))
    except (OSError, ValueError, TypeError):
        raise Rejected('hierarchy_registry') from None


def registry_digest():
    return digest(canonical(registry()).encode())


def pinned_profile(body):
    try:
        profile = body['profile']
        if digest(canonical(profile).encode()) != body['registry_digest']:
            raise Rejected('hierarchy_profile_digest')
        profile = validate_profile(profile)
        bindings, nodes, edges = body['bindings'], body['nodes'], body['edges']
        if any(not isinstance(items, list) for items in [bindings, nodes, edges]) or len(bindings) > min(MAX_ROWS, profile['max_bindings']) or len(nodes) > profile['max_nodes'] + 2 * len(bindings) or len(edges) > profile['max_edges'] + len(bindings):
            raise Rejected('hierarchy_snapshot_bound')
        return profile
    except (KeyError, TypeError, ValueError):
        raise Rejected('hierarchy_profile_digest') from None


def provider_key(raw):
    if not isinstance(raw, str):
        raise Rejected('hierarchy_provider')
    result = raw if raw in PROVIDERS else ALIASES.get(raw)
    if result is None:
        raise Rejected('hierarchy_provider')
    return result


def interval(item):
    start, end = item['valid_from'], item['valid_to']
    for value in [start, end]:
        if value is not None:
            if not isinstance(value, str):
                raise Rejected('hierarchy_validity')
            utc(value)
    if start is not None and end is not None and utc(start) >= utc(end):
        raise Rejected('hierarchy_validity')
    return (utc(start) if start is not None else datetime.min.replace(tzinfo=timezone.utc), utc(end) if end is not None else datetime.max.replace(tzinfo=timezone.utc))


def intersecting(left, right):
    a, b = interval(left), interval(right)
    return a[0] < b[1] and b[0] < a[1]


def covers(item, row):
    return item['valid_from'] is not None and item['valid_to'] is not None and utc(item['valid_from']) <= row['charge_start'] and row['charge_end'] <= utc(item['valid_to'])


def node_id(node):
    return str(uuid5(INSTALLATION, canonical([node[k] for k in ['provider', 'partition', 'kind', 'namespace', 'native_id']])))


def namespace(node, profile):
    kind = profile['kinds'][node['kind']]['namespace']
    value = node['namespace']
    if kind == 'global':
        if value != 'global':
            raise Rejected('hierarchy_namespace')
    elif not value.startswith(kind + ':') or not text(value[len(kind) + 1:]):
        raise Rejected('hierarchy_namespace')
    if kind == 'service-instance':
        try:
            UUID(value.split(':', 1)[1])
        except ValueError:
            raise Rejected('hierarchy_namespace') from None
    if node['provider'] == 'vmware' and node['kind'] == 'service_instance':
        try:
            UUID(node['native_id'])
        except ValueError:
            raise Rejected('hierarchy_namespace') from None


def namespace_edge(child, parent, profile):
    child_scope = profile['kinds'][child['kind']]['namespace']
    parent_scope = profile['kinds'][parent['kind']]['namespace']
    anchors = {'organization': 'organization', 'directory': 'resource_directory', 'billing-account': 'billing_account', 'project': 'project', 'tenancy': 'tenancy', 'account': 'account', 'service-instance': 'service_instance'}
    if child_scope != 'global' and parent['kind'] == anchors[child_scope] and child['namespace'] != child_scope + ':' + parent['native_id']:
        raise Rejected('hierarchy_namespace_edge')
    if child_scope != 'global' and child_scope == parent_scope and child['namespace'] != parent['namespace']:
        raise Rejected('hierarchy_namespace_edge')
    if child['provider'] == 'azure' and child['kind'] == 'management_group' and parent['kind'] == 'entra_tenant' and child['namespace'] != 'directory:' + parent['native_id']:
        raise Rejected('hierarchy_namespace_edge')


def graph_constraints(nodes, edges, reg):
    grouped = defaultdict(list)
    outgoing = defaultdict(list)
    for edge in edges:
        grouped[(edge['child'], edge['relation'])].append(edge)
        outgoing[(edge['axis'], edge['child'])].append(edge)
    for candidates in grouped.values():
        profile = reg['providers'][nodes[candidates[0]['child']]['provider']]
        rule = next(r for r in profile['relations'] if r['relation'] == candidates[0]['relation'])
        if rule['max_parents'] == 1:
            for index, edge in enumerate(candidates):
                if any(intersecting(edge, other) for other in candidates[index + 1:]):
                    raise Rejected('hierarchy_cardinality')
    visits = 0
    def walk(axis, key, path, valid):
        nonlocal visits
        visits += 1
        if visits > 100000 or len(path) > reg['max_depth']:
            raise Rejected('hierarchy_depth')
        for edge in outgoing[(axis, key)]:
            bounds = interval(edge)
            overlap = (max(valid[0], bounds[0]), min(valid[1], bounds[1]))
            if overlap[0] >= overlap[1]:
                continue
            if edge['parent'] in path:
                raise Rejected('hierarchy_cycle')
            walk(axis, edge['parent'], path | {edge['parent']}, overlap)
    whole = (datetime.min.replace(tzinfo=timezone.utc), datetime.max.replace(tzinfo=timezone.utc))
    for axis, key in list(outgoing):
        walk(axis, key, {key}, whole)


def validated(doc, config=None):
    reg = registry()
    fields = {'schema_version', 'revision', 'observed_at', 'evidence_class', 'evidence_ref', 'nodes', 'edges', 'bindings'}
    try:
        if not isinstance(doc, dict) or set(doc) != fields or type(doc['schema_version']) is not int or doc['schema_version'] != 1 or type(doc['revision']) is not int or not 0 < doc['revision'] < 2**63:
            raise Rejected('hierarchy_catalog')
        if len(canonical(doc).encode()) > reg['max_bytes'] or doc['evidence_class'] not in ['synthetic', 'operator-supplied-native'] or not text(doc['evidence_ref'], 512) or not isinstance(doc['observed_at'], str):
            raise Rejected('hierarchy_catalog')
        utc(doc['observed_at'])
        for key, bound in [('nodes', 'max_nodes'), ('edges', 'max_edges'), ('bindings', 'max_bindings')]:
            if not isinstance(doc[key], list) or len(doc[key]) > reg[bound]:
                raise Rejected('hierarchy_catalog_bound')
        nodes = {}
        identities, aliases = set(), set()
        for node in doc['nodes']:
            if not isinstance(node, dict) or set(node) != NODE_FIELDS or not text(node['key'], 64) or node['key'] in nodes or node['provider'] not in PROVIDERS:
                raise Rejected('hierarchy_node')
            profile = reg['providers'][node['provider']]
            if node['kind'] not in profile['kinds'] or any(not text(node[k], 4096 if k in ['native_id', 'namespace'] else 256) for k in ['partition', 'namespace', 'native_id', 'label']):
                raise Rejected('hierarchy_node')
            namespace(node, profile)
            identity = node_id(node)
            if identity in identities or not isinstance(node['aliases'], list) or len(node['aliases']) > 32:
                raise Rejected('hierarchy_identity')
            identities.add(identity)
            for alias in node['aliases']:
                if not isinstance(alias, dict) or set(alias) != {'kind', 'value'} or not text(alias['kind'], 64) or not text(alias['value']):
                    raise Rejected('hierarchy_alias')
                token = canonical([node[k] for k in ['provider', 'partition', 'kind', 'namespace']] + [alias['kind'], alias['value']])
                if token in aliases:
                    raise Rejected('hierarchy_alias_collision')
                aliases.add(token)
            nodes[node['key']] = node
        edges, seen = [], set()
        for edge in doc['edges']:
            if not isinstance(edge, dict) or set(edge) != EDGE_FIELDS or edge['child'] not in nodes or edge['parent'] not in nodes or edge['child'] == edge['parent']:
                raise Rejected('hierarchy_edge')
            child, parent = nodes[edge['child']], nodes[edge['parent']]
            if child['provider'] != parent['provider'] or child['partition'] != parent['partition']:
                raise Rejected('hierarchy_cross_provider')
            profile = reg['providers'][child['provider']]
            relation = next((r for r in profile['relations'] if r['relation'] == edge['relation']), None)
            if relation is None or child['kind'] not in relation['children'] or parent['kind'] not in relation['parents']:
                raise Rejected('hierarchy_relation')
            namespace_edge(child, parent, profile)
            interval(edge)
            token = canonical(edge)
            if token in seen:
                raise Rejected('hierarchy_duplicate_edge')
            seen.add(token)
            edges.append({**edge, 'axis': relation['axis']})
        graph_constraints(nodes, edges, reg)
        owners = {}
        if config is not None:
            for tenant in config['tenants']:
                for assignment in tenant['assignments']:
                    try:
                        provider = provider_key(assignment[0])
                    except Rejected:
                        continue
                    token = canonical([provider, *assignment[1:3]])
                    if token in owners and owners[token] != tenant['key']:
                        raise Rejected('hierarchy_ambiguous_owner')
                    owners[token] = tenant['key']
        grouped = defaultdict(list)
        for binding in doc['bindings']:
            if not isinstance(binding, dict) or set(binding) != BINDING_FIELDS or binding['provider'] not in PROVIDERS or any(not text(binding[k]) for k in ['account', 'subaccount', 'tenant_key']):
                raise Rejected('hierarchy_binding')
            roles = binding['roles']
            if not isinstance(roles, dict) or not roles or set(roles) - {'billing', 'usage', 'resource', 'identity'}:
                raise Rejected('hierarchy_binding_roles')
            for role, key in roles.items():
                if key not in nodes or nodes[key]['provider'] != binding['provider'] or nodes[key]['kind'] not in reg['providers'][binding['provider']]['roles'][role]:
                    raise Rejected('hierarchy_binding_roles')
            interval(binding)
            token = canonical([binding[k] for k in ['provider', 'account', 'subaccount']])
            if config is not None and owners.get(token) != binding['tenant_key']:
                raise Rejected('hierarchy_binding_authority')
            if any(intersecting(binding, other) for other in grouped[token]):
                raise Rejected('hierarchy_binding_overlap')
            grouped[token].append(binding)
        return reg, nodes, edges
    except (KeyError, TypeError, ValueError, RecursionError):
        raise Rejected('hierarchy_catalog') from None


def validate_catalog(doc, config):
    validated(doc, config)
    return doc


def native_node(node, reg, catalog):
    return {'id': node_id(node), **{k: copy.deepcopy(node[k]) for k in NODE_FIELDS - {'key'}}, 'category': reg['providers'][node['provider']]['kinds'][node['kind']]['category'], 'evidence_class': catalog['evidence_class'], 'observed_at': catalog['observed_at']}


def focus_node(row, kind):
    try:
        provider = provider_key(row['provider'])
    except Rejected:
        provider = 'unknown'
    billing = kind == 'focus_billing_scope'
    node = {'provider': provider, 'partition': 'focus-observed', 'namespace': canonical([row['provider']] if billing else [row['provider'], row['account']]), 'kind': kind, 'native_id': row['account'] if billing else row['subaccount'], 'label': row['account'] if billing else row['subaccount'], 'aliases': []}
    return {'id': node_id(node), **node, 'category': 'invoice_scope' if billing else 'usage_scope', 'raw_provider': row['provider'], 'evidence_class': 'focus-observed'}


def encoded_edge(child, parent, relation, axis, valid_from, valid_to, evidence_class):
    body = {'child': child, 'parent': parent, 'relation': relation, 'axis': axis, 'valid_from': valid_from, 'valid_to': valid_to}
    return {'id': digest(canonical(body).encode()), **body, 'evidence_class': evidence_class}


def role_partial_axes(root, outgoing, row, axes):
    partial = set()
    for axis in axes:
        pending, visited = [root], set()
        while pending:
            key = pending.pop()
            if key in visited:
                continue
            visited.add(key)
            for edge in outgoing[key]:
                if edge['axis'] != axis:
                    continue
                if covers(edge, row):
                    pending.append(edge['parent'])
                elif edge['valid_from'] is None or edge['valid_to'] is None or interval(edge)[0] < row['charge_end'] and row['charge_start'] < interval(edge)[1]:
                    partial.add(axis)
    return sorted(partial)


def snapshot(rows, tenant_key, source, catalog=None):
    if not text(tenant_key, 64) or not text(source, 128) or len(rows) > MAX_ROWS:
        raise Rejected('hierarchy_snapshot_scope')
    reg, native, catalog_edges = validated(catalog) if catalog is not None else (registry(), {}, [])
    if len(rows) > reg['max_bindings']:
        raise Rejected('hierarchy_snapshot_bound')
    outgoing = defaultdict(list)
    binding_index = defaultdict(list)
    for edge in catalog_edges:
        outgoing[edge['child']].append(edge)
    if catalog is not None:
        for binding in catalog['bindings']:
            binding_index[(binding['provider'], binding['account'], binding['subaccount'], binding['tenant_key'])].append(binding)
    nodes, edges, bindings, ordinals = {}, {}, [], set()
    for row in rows:
        if type(row['ordinal']) is not int or row['ordinal'] < 1 or row['ordinal'] in ordinals or str(row['tenant_id']) != str(tenant_id(tenant_key)):
            raise Rejected('hierarchy_tenant_scope')
        ordinals.add(row['ordinal'])
        invoice = focus_node(row, 'focus_billing_scope')
        usage = focus_node(row, 'focus_usage_scope') if row['subaccount'] is not None else invoice
        nodes[invoice['id']], nodes[usage['id']] = invoice, usage
        visible_nodes, visible_edges = {invoice['id'], usage['id']}, set()
        if usage['id'] != invoice['id']:
            observed = encoded_edge(usage['id'], invoice['id'], 'focus_invoice_scope', 'focus', row['charge_start'].isoformat(), row['charge_end'].isoformat(), 'focus-observed')
            edges[observed['id']] = observed
            visible_edges.add(observed['id'])
        roles, unresolved, partial = {}, set(), {}
        try:
            provider = provider_key(row['provider'])
        except Rejected:
            provider = None
        matches = [b for b in binding_index[(provider, row['account'], row['subaccount'], tenant_key)] if covers(b, row)]
        if len(matches) > 1:
            raise Rejected('hierarchy_binding_overlap')
        if matches:
            binding = matches[0]
            roles = {role: node_id(native[key]) for role, key in binding['roles'].items()}
            partial = {role: role_partial_axes(key, outgoing, row, reg['axes']) for role, key in binding['roles'].items()}
            pending, visited = list(binding['roles'].values()), set()
            while pending:
                key = pending.pop()
                if key in visited:
                    continue
                visited.add(key)
                if len(visited) > reg['max_nodes']:
                    raise Rejected('hierarchy_snapshot_bound')
                node = native_node(native[key], reg, catalog)
                nodes[node['id']] = node
                visible_nodes.add(node['id'])
                candidates = outgoing[key]
                expected_axes = {r['axis'] for r in reg['providers'][native[key]['provider']]['relations'] if native[key]['kind'] in r['children']}
                valid_axes = set()
                partial_axes = set()
                for edge in candidates:
                    if covers(edge, row):
                        valid_axes.add(edge['axis'])
                        parent = native[edge['parent']]
                        item = encoded_edge(node['id'], node_id(parent), edge['relation'], edge['axis'], edge['valid_from'], edge['valid_to'], catalog['evidence_class'])
                        edges[item['id']] = item
                        visible_edges.add(item['id'])
                        pending.append(edge['parent'])
                    elif edge['valid_from'] is None or edge['valid_to'] is None or interval(edge)[0] < row['charge_end'] and row['charge_start'] < interval(edge)[1]:
                        partial_axes.add(edge['axis'])
                unresolved.update(expected_axes - valid_axes)
                unresolved.update(partial_axes)
        else:
            unresolved.update(reg['axes'])
            unresolved.discard('focus')
        status = 'mixed' if roles and any(partial.values()) else 'typed-native' if roles else 'focus-only'
        bindings.append({'ordinal': row['ordinal'], 'focus_billing': invoice['id'], 'focus_usage': usage['id'], 'native_roles': roles, 'visible_nodes': sorted(visible_nodes), 'visible_edges': sorted(visible_edges), 'unresolved_axes': sorted(unresolved), 'role_partial_axes': partial, 'status': status, 'evidence_class': catalog['evidence_class'] if roles else 'focus-observed', 'charge_start': row['charge_start'].isoformat(), 'charge_end': row['charge_end'].isoformat(), 'row_scope_digest': digest(canonical([row[k] for k in ['provider', 'account', 'subaccount']]).encode())})
        if row['subaccount'] is None:
            bindings[-1]['usage_scope_status'] = 'account-only-subaccount-absent'
    statuses = {b['status'] for b in bindings}
    status = 'typed-native' if statuses == {'typed-native'} else 'mixed' if any(b['native_roles'] for b in bindings) else 'focus-only'
    evidence = {b['evidence_class'] for b in bindings}
    evidence_class = next(iter(evidence)) if len(evidence) == 1 else 'mixed' if evidence else 'focus-observed'
    return {'schema_version': 1, 'profile': copy.deepcopy(reg), 'registry_digest': digest(canonical(reg).encode()), 'catalog_digest': digest(canonical(catalog).encode()) if catalog is not None else None, 'observed_at': catalog['observed_at'] if any(b['native_roles'] for b in bindings) else None, 'evidence_class': evidence_class, 'status': status, 'tenant_key': tenant_key, 'source': source, 'nodes': sorted(nodes.values(), key=lambda n: n['id']), 'edges': sorted(edges.values(), key=lambda e: e['id']), 'bindings': sorted(bindings, key=lambda b: b['ordinal'])}


def select(body, ordinals):
    pinned_profile(body)
    if not isinstance(ordinals, (list, set, tuple)) or any(type(o) is not int or o < 1 for o in ordinals):
        raise Rejected('hierarchy_selection')
    wanted = set(ordinals)
    if len(wanted) != len(ordinals) or not wanted <= {b['ordinal'] for b in body['bindings']}:
        raise Rejected('hierarchy_selection')
    bindings = [b for b in body['bindings'] if b['ordinal'] in wanted]
    nodes = {n for b in bindings for n in b['visible_nodes']}
    edges = {e for b in bindings for e in b['visible_edges']}
    if not nodes <= {n['id'] for n in body['nodes']} or not edges <= {e['id'] for e in body['edges']}:
        raise Rejected('hierarchy_selection_integrity')
    result = copy.deepcopy(body)
    result.update(nodes=[n for n in result['nodes'] if n['id'] in nodes], edges=[e for e in result['edges'] if e['id'] in edges], bindings=copy.deepcopy(bindings))
    if any(e['child'] not in nodes or e['parent'] not in nodes for e in result['edges']):
        raise Rejected('hierarchy_selection_integrity')
    statuses = {b['status'] for b in bindings}
    result['status'] = 'typed-native' if statuses == {'typed-native'} else 'mixed' if any(b['native_roles'] for b in bindings) else 'focus-only'
    evidence = {b['evidence_class'] for b in bindings}
    result['evidence_class'] = next(iter(evidence)) if len(evidence) == 1 else 'mixed' if evidence else 'focus-observed'
    if not any(b['native_roles'] for b in bindings):
        result['observed_at'] = None
    return result


def group_rows(rows, body, kind, axis, role='usage'):
    reg = pinned_profile(body)
    kinds = {k for p in reg['providers'].values() for k in p['kinds']} | {'focus_billing_scope', 'focus_usage_scope'}
    category = kind[9:] if isinstance(kind, str) and kind.startswith('category:') else None
    direct = kind == 'role' and axis == 'binding'
    if not isinstance(kind, str) or not isinstance(axis, str) or not direct and ((category is None and kind not in kinds or category is not None and category not in reg['categories']) or axis not in reg['axes']) or role not in ['billing', 'usage', 'resource', 'identity']:
        raise Rejected('hierarchy_grouping')
    if len(rows) > min(MAX_ROWS, reg['max_bindings']) or len({r['ordinal'] for r in rows}) != len(rows):
        raise Rejected('hierarchy_grouping_bound')
    for name in ADDITIVE:
        if sum(abs(coefficient(r[name])) for r in rows if r[name] is not None) >= LIMIT:
            raise Rejected('selection_aggregate_bound')
    bindings = {b['ordinal']: b for b in body['bindings']}
    nodes, edges = {n['id']: n for n in body['nodes']}, {e['id']: e for e in body['edges']}
    grouped, unresolved_rows = defaultdict(list), 0
    for row in rows:
        binding = bindings.get(row['ordinal'])
        if binding is None or binding['charge_start'] != row['charge_start'].isoformat() or binding['charge_end'] != row['charge_end'].isoformat() or str(row['tenant_id']) != str(tenant_id(body['tenant_key'])) or binding['row_scope_digest'] != digest(canonical([row[k] for k in ['provider', 'account', 'subaccount']]).encode()):
            raise Rejected('hierarchy_binding_mismatch')
        root = binding['native_roles'].get(role)
        if root is None and role in ['billing', 'usage']:
            root = binding['focus_' + role]
        if direct:
            if root is not None and (root not in binding['visible_nodes'] or root not in nodes):
                raise Rejected('hierarchy_grouping_scope')
            key = root or 'unresolved'
            unresolved_rows += key == 'unresolved'
            grouped[key].append(row)
            continue
        outgoing = defaultdict(list)
        for edge_id in binding['visible_edges']:
            edge = edges[edge_id]
            if edge['axis'] == axis and covers(edge, row):
                outgoing[edge['child']].append(edge['parent'])
        pending, visited, candidates = [(root, 1)] if root is not None else [], set(), set()
        while pending:
            key, depth = pending.pop()
            if depth > reg['max_depth']:
                raise Rejected('hierarchy_depth')
            if key in visited:
                continue
            if key not in binding['visible_nodes'] or key not in nodes:
                raise Rejected('hierarchy_grouping_scope')
            visited.add(key)
            node = nodes[key]
            if node['category'] == category if category is not None else node['kind'] == kind:
                candidates.add(key)
            pending.extend((parent, depth + 1) for parent in outgoing[key])
        if len(candidates) > 1:
            raise Rejected('hierarchy_nonadditive_ancestors')
        key = next(iter(candidates)) if candidates and axis not in binding['role_partial_axes'].get(role, []) else 'unresolved'
        if key == 'unresolved':
            unresolved_rows += 1
        grouped[key].append(row)
    return {'groups': {key: controls(grouped[key]) for key in sorted(grouped)}, 'unresolved_rows': unresolved_rows, 'additive': True}
