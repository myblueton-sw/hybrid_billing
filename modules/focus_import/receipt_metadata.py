import re

from . import connections
from .model import Rejected, canonical, digest, tenant_id
from .temporal import checked_window, utc


RECORD_FIELDS = (
    'id', 'kind', 'connection', 'stage', 'window_start', 'window_end',
    'source_revision', 'focus_version', 'format', 'declared_rows', 'file_bytes',
    'file_digest', 'mapping_digest', 'received_at', 'complete_through', 'finalized',
    'reason', 'root_receipt', 'replaced_by', 'attempts', 'known_zero', 'published',
)
DELIVERY_STAGES = {'registered', 'verified', 'sealed'}
QUARANTINE_STAGES = {'quarantined', 'replacement-requested', 'escalated', 'resolved', 'deleted'}
COVERAGE_STAGES = DELIVERY_STAGES | {'quarantined', 'replacement-requested', 'escalated', 'missing', 'late'}


def sha(value):
    return isinstance(value, str) and re.fullmatch('[a-f0-9]{64}', value) is not None


def fail():
    raise Rejected('receipt_evidence_unavailable')


def number(value, minimum, maximum):
    if type(value) is not int or not minimum <= value <= maximum:
        fail()
    return str(value)


def nullable_number(value, minimum, maximum):
    return str(value) if type(value) is int and minimum <= value <= maximum else None


def timestamp(value):
    try:
        result = utc(value).isoformat()
    except (Rejected, TypeError, ValueError, OverflowError):
        fail()
    if len(result) > 40:
        fail()
    return result


def blank(identity, kind, stage):
    if not sha(identity):
        fail()
    return {**dict.fromkeys(RECORD_FIELDS), 'id': identity, 'kind': kind, 'stage': stage}


def configuration(state, authority):
    records = state.get('metadata', {}).get('configuration', [])
    if len(records) != 1 or not isinstance(records[0].get('body'), dict):
        fail()
    saved = records[0]
    expected = digest(canonical(saved['body']).encode())
    if saved.get('digest') != expected or state.get('config_digest') != expected or authority.bindings.get('config_digest') != expected or authority.bindings.get('source') != saved['body'].get('source'):
        fail()
    return saved['body']


def members_for(authority, context, config):
    if authority.is_view:
        audience = context['audience']
        if audience.get('actor') != authority.actor or not audience.get('active'):
            fail()
        members = {str(t['tenant_id']): t['properties'] for t in context['tenants'] if t.get('active')}
        if len(members) != len(context['tenants']):
            fail()
        kind = 'company'
    else:
        members = {str(authority.tenant): context['properties']}
        if not context.get('active') or context['properties'].get('actor') != authority.actor:
            fail()
        kind = 'tenant'
    if not 1 <= len(members) <= 10000:
        fail()
    current = {str(tenant_id(t['key'])): t for t in config['tenants'] if t['active']}
    if any(current.get(tid) != props for tid, props in members.items()):
        fail()
    return kind, members


def authorized_connections(state, config, members):
    ownership = connections.owners(config)
    result = {}
    for cid, connection in state.get('connections', {}).items():
        if not connections.token(cid) or connection.get('id') != cid:
            continue
        domains = connection.get('domains')
        if not isinstance(domains, list) or not 1 <= len(domains) <= 10000:
            continue
        owners = [ownership.get(canonical(d)) for d in domains]
        if any(owner is None for owner in owners):
            continue
        affected = {str(tenant_id(owner['key'])) for owner in owners}
        if not affected or not affected <= set(members):
            continue
        if len({canonical(d) for d in domains}) != len(domains):
            continue
        result[cid] = connection
    return result


def descriptor_for(value):
    if not isinstance(value, dict):
        fail()
    scope = checked_window(value.get('window'))
    number(value.get('revision'), 1, 2**63 - 1)
    number(value.get('row_count'), 0, 10000)
    if not sha(value.get('file_digest')) or not sha(value.get('domains_digest')) or type(value.get('finalized')) is not bool:
        fail()
    through = utc(value.get('complete_through'))
    if not utc(scope['start']) < through <= utc(scope['end']) or value['finalized'] and through != utc(scope['end']):
        fail()
    return scope


def version_for(value, registry):
    if not isinstance(value, str) or len(value) > 8 or not re.fullmatch(r'\d{1,2}\.\d{1,2}', value) or value not in registry:
        fail()
    return value


def manifest_for(state, delivery, descriptor, source):
    batch_id = delivery.get('batch_id')
    if batch_id is None:
        return None
    batches = [b for b in state.get('metadata', {}).get('batch', []) if b.get('batch_id') == batch_id]
    if len(batches) != 1:
        fail()
    batch = batches[0]
    manifest = batch.get('manifest')
    if not isinstance(manifest, dict) or not sha(batch.get('manifest_digest')):
        fail()
    if digest(canonical(manifest).encode()) != batch['manifest_digest'] or delivery.get('manifest_digest') != batch['manifest_digest']:
        fail()
    if batch.get('source') != source or manifest.get('source') != source or batch.get('file_digest') != descriptor['file_digest'] or manifest.get('file_digest') != descriptor['file_digest']:
        fail()
    if manifest.get('row_count') != descriptor['row_count'] or manifest.get('version') != delivery.get('version') or manifest.get('mapping_digest') != delivery.get('mapping_digest'):
        fail()
    if batch.get('source_revision') != delivery.get('transport_revision') or manifest.get('source_revision') != delivery.get('transport_revision'):
        fail()
    if manifest.get('config_digest') != state['config_digest'] or batch.get('config_digest') != state['config_digest']:
        fail()
    if batch.get('state') not in {'verified', 'sealed'} or delivery['state'] == 'sealed' and batch['state'] != 'sealed':
        fail()
    return manifest


def legacy_candidate(state, item, members):
    batches = [b for b in state.get('metadata', {}).get('batch', []) if b.get('batch_id') == item.get('batch_id')]
    if len(batches) != 1 or not isinstance(batches[0].get('manifest'), dict):
        return False
    slices = batches[0]['manifest'].get('slices')
    if not isinstance(slices, dict) or not slices:
        return False
    affected = set()
    for tid, part in slices.items():
        count = part.get('row_count') if isinstance(part, dict) else None
        if type(count) is not int or not 0 <= count <= 10000:
            return False
        if count:
            affected.add(tid)
    return bool(affected) and affected <= set(members)


def legacy_owned(manifest, descriptor, members):
    if manifest is None or descriptor['row_count'] == 0:
        return False
    slices = manifest.get('slices')
    if not isinstance(slices, dict) or not slices:
        return False
    affected = set()
    total = 0
    for tid, part in slices.items():
        count = part.get('row_count') if isinstance(part, dict) else None
        if type(count) is not int or not 0 <= count <= 10000:
            return False
        total += count
        if count:
            affected.add(tid)
    return total == descriptor['row_count'] and bool(affected) and affected <= set(members)


def zero_attestations(descriptor, connection):
    expected = {canonical(d) for d in connection['domains']}
    zeros = descriptor.get('zero_domains', [])
    if not isinstance(zeros, list) or len(zeros) > len(expected):
        fail()
    seen = set()
    for item in zeros:
        if not isinstance(item, dict) or set(item) != {'domain', 'reason', 'evidence_digest'}:
            fail()
        key = canonical(item['domain'])
        if key not in expected or key in seen or not connections.token(item['reason']) or not sha(item['evidence_digest']):
            fail()
        seen.add(key)
    if descriptor['row_count'] == 0 and seen != expected:
        fail()


def delivery_records(state, config, members, owned, registry):
    records = {}
    ownership = connections.owners(config)
    for identity, item in state.get('deliveries', {}).items():
        descriptor = item.get('descriptor')
        if not isinstance(descriptor, dict):
            continue
        cid = descriptor.get('connection_id')
        if cid is not None and cid not in owned:
            continue
        if cid is None and (item.get('batch_id') is None or not legacy_candidate(state, item, members)):
            continue
        if item.get('state') not in DELIVERY_STAGES:
            fail()
        scope = descriptor_for(descriptor)
        version = version_for(item.get('version'), registry)
        if identity != digest(canonical({'version': version, 'descriptor': descriptor}).encode()):
            fail()
        mapping = digest(canonical(registry[version]).encode())
        if item.get('mapping_digest') != mapping:
            fail()
        if cid is not None:
            connection = owned[cid]
            if connections.get(state, descriptor) != connection or item.get('connection_digest') != digest(canonical(connection).encode()) or connection['version'] != version:
                continue
            zero_attestations(descriptor, connection)
        manifest = manifest_for(state, item, descriptor, config['source'])
        if cid is None and not legacy_owned(manifest, descriptor, members):
            continue
        if cid is not None and manifest is not None:
            expected = {str(tenant_id(ownership[canonical(d)]['key'])) for d in owned[cid]['domains']}
            parts = manifest.get('slices')
            if not isinstance(parts, dict) or any(not isinstance(v, dict) or type(v.get('row_count')) is not int or not 0 <= v['row_count'] <= 10000 for v in parts.values()):
                continue
            affected = {tid for tid, part in parts.items() if part['row_count'] > 0}
            if not affected <= expected or sum(part['row_count'] for part in parts.values()) != descriptor['row_count']:
                continue
        if item['state'] != 'registered' and manifest is None:
            fail()
        record = blank(identity, 'delivery', item['state'])
        record.update(connection=cid, window_start=scope['start'], window_end=scope['end'],
                      source_revision=number(descriptor['revision'], 1, 2**63 - 1), focus_version=version,
                      declared_rows=number(descriptor['row_count'], 0, 10000), file_digest=descriptor['file_digest'],
                      mapping_digest=mapping, received_at=timestamp(item.get('received_at')))
        if manifest is not None:
            format = manifest.get('input_format')
            if format is not None and format not in {'csv', 'parquet'}:
                fail()
            size = number(manifest['file_bytes'], 0, 16777216) if manifest.get('file_bytes') is not None else None
            record.update(format=format, file_bytes=size)
        records[identity] = record
    return records


def quarantine_records(state, config, owned, registry):
    records = {}
    ownership = connections.owners(config)
    for identity, item in state.get('receipts', {}).items():
        if item.get('admitted') is not True:
            continue
        cid = item.get('connection_id')
        if cid not in owned or not isinstance(item.get('descriptor'), dict):
            continue
        if item.get('state') not in QUARANTINE_STAGES:
            fail()
        descriptor = item['descriptor']
        if descriptor.get('connection_id') != cid or connections.get(state, descriptor) != owned[cid]:
            continue
        expected_keys = sorted({owner['key'] for d in owned[cid]['domains'] for owner in [ownership.get(canonical(d))]})
        if item.get('tenant_keys') != expected_keys:
            continue
        scope = checked_window(descriptor.get('window'))
        version = version_for(item.get('version'), registry)
        if not sha(item.get('file_digest')):
            fail()
        bare = {k: v for k, v in descriptor.items() if k not in {'replacement_plan', 'reopen_plan'}}
        if item.get('descriptor_digest') != digest(canonical(bare).encode()):
            continue
        reason = item.get('reason')
        identity_body = {k: item.get(k) for k in ['file_digest', 'version', 'connection_id', 'reason', 'admitted', 'descriptor_digest']}
        if identity != digest(canonical(identity_body).encode()):
            fail()
        if not isinstance(reason, str) or not re.fullmatch('[a-z][a-z0-9_]{0,79}', reason):
            fail()
        record = blank(identity, 'quarantine', item['state'])
        record.update(connection=cid, window_start=scope['start'], window_end=scope['end'],
                      source_revision=nullable_number(descriptor.get('revision'), 1, 2**63 - 1), focus_version=version,
                      declared_rows=nullable_number(descriptor.get('row_count'), 0, 10000), file_digest=item['file_digest'],
                      received_at=timestamp(item.get('received_at')), reason=reason)
        records[identity] = record
    return records


def same_window(left, right):
    return left['connection'] == right['connection'] and left['window_start'] == right['window_start'] and left['window_end'] == right['window_end']


def lineage(state, quarantine, deliveries):
    for identity, record in quarantine.items():
        item = state['receipts'][identity]
        root_id = item.get('root_receipt', identity)
        root = quarantine.get(root_id)
        if root and same_window(record, root):
            record['root_receipt'] = root_id
            record['attempts'] = nullable_number(state['receipts'][root_id].get('attempts'), 0, 10)
        replacement = item.get('replaced_by')
        target = quarantine.get(replacement) or deliveries.get(replacement)
        if target and same_window(record, target):
            record['replaced_by'] = replacement


def coverage_records(state, config, members, owned, deliveries, quarantine, at, registry):
    if not owned:
        return None
    selected = {'connections': owned}
    selected['deliveries'] = {k: state['deliveries'][k] for k in deliveries}
    selected['receipts'] = {k: state['receipts'][k] for k in quarantine}
    selected['windows'] = {k: w for k, w in state.get('windows', {}).items() if w.get('connection_id') in owned}
    for key, target in selected['windows'].items():
        connection = owned[target['connection_id']]
        if connections.identity(connection, checked_window(target['window'])) != key:
            return None
        if target.get('current') is not None and deliveries.get(target['current'], {}).get('stage') != 'sealed':
            return None
        if any(key not in deliveries for key in target.get('revisions', [])) or target.get('current') is not None and target['current'] not in deliveries:
            return None
    coverage = connections.coverage(selected, members, config, None, None, at)
    grouped = {}
    for row in coverage:
        identity = digest(canonical({'connection': row['connection_id'], 'window': row['window']}).encode())
        grouped.setdefault(identity, []).append(row)
    result = []
    for identity, rows in sorted(grouped.items()):
        first = rows[0]
        if any(row['state'] != first['state'] or row['published'] != first['published'] or row['finalized'] != first['finalized'] or row['complete_through'] != first['complete_through'] for row in rows):
            return None
        if first['state'] not in COVERAGE_STAGES:
            fail()
        record = blank(identity, 'coverage', first['state'])
        record.update(connection=first['connection_id'], window_start=first['window']['start'],
                      window_end=first['window']['end'], focus_version=version_for(owned[first['connection_id']]['version'], registry),
                      published=bool(first['published']))
        if first['published']:
            record.update(complete_through=timestamp(first['complete_through']), finalized=first['finalized'],
                          known_zero=all(row['known_zero'] for row in rows))
        result.append(record)
    return result
