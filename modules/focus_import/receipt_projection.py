import json
import re
from datetime import datetime, timezone

from . import collection
from .model import Rejected, canonical, digest, versions
from .receipt_metadata import (
    authorized_connections, configuration, coverage_records, delivery_records,
    lineage, members_for, quarantine_records,
)


PROFILE = 'retained-focus-receipts-v1'
MAX_RESPONSE_BYTES = 2 * 1024 * 1024
VIEWS = {'imports', 'quarantine', 'completeness', 'extraction'}
QUERY_KEYS = {'view', 'q', 'after', 'snapshot', 'permission_revision'}


def selection(params):
    if not isinstance(params, dict) or set(params) - QUERY_KEYS or any(not isinstance(v, str) for v in params.values()):
        raise Rejected('request_invalid')
    view = params.get('view', 'imports')
    query = params.get('q', '')
    after = params.get('after', '0')
    if view not in VIEWS or not re.fullmatch('[a-z0-9:_-]{0,128}', query) or not re.fullmatch('0|[1-9][0-9]{0,4}', after) or int(after) > 10000:
        raise Rejected('request_invalid')
    for key in ['snapshot', 'permission_revision']:
        if key in params and not re.fullmatch('[a-f0-9]{64}', params[key]):
            raise Rejected('request_invalid')
    if ('snapshot' in params) != ('permission_revision' in params) or int(after) and 'snapshot' not in params:
        raise Rejected('snapshot_required')
    return view, query, int(after)


def project(actor, params, authority, context, state, at):
    view, query, after = selection(params)
    if authority.actor != actor:
        raise Rejected('permission')
    config = configuration(state, authority)
    audience, members = members_for(authority, context, config)
    owned = authorized_connections(state, config, members)
    registry = versions()
    deliveries = delivery_records(state, config, members, owned, registry)
    quarantines = quarantine_records(state, config, owned, registry)
    lineage(state, quarantines, deliveries)
    coverage = coverage_records(state, config, members, owned, deliveries, quarantines, at, registry)
    capabilities = {'receipts': 'fully-authorized-metadata-only',
                    'completeness': 'registered-expectations-only' if coverage is not None else 'unavailable',
                    'extraction': 'unconnected'}
    auth_scope = digest(canonical({'profile': PROFILE, 'source': config['source'], 'actor': actor, 'audience': audience, 'members': sorted(members)}).encode())
    permission_revision = digest(canonical({'auth_scope': auth_scope, 'members': members, 'connections': owned,
                                         'policies': {p['key']: state.get('policies', {}).get(p['key']) for p in members.values()}}).encode())
    records = list(deliveries.values()) if view == 'imports' else list(quarantines.values()) if view == 'quarantine' else coverage if view == 'completeness' else None
    supported = records is not None
    records = [r for r in records or [] if r['id'].startswith(query) or r['connection'] is not None and r['connection'].startswith(query)]
    records.sort(key=lambda r: (r['window_start'] or '', r['source_revision'] is not None, int(r['source_revision'] or '0'), r['id']), reverse=True)
    if len(records) > 10000:
        raise Rejected('selection_unavailable')
    total = len(records) if supported else None
    snapshot = digest(canonical({'profile': PROFILE, 'permission_revision': permission_revision, 'view': view, 'q': query,
                                'capabilities': capabilities, 'total': total, 'records': records}).encode())
    if 'snapshot' in params and (params['snapshot'] != snapshot or params['permission_revision'] != permission_revision):
        raise Rejected('query_snapshot_changed')
    if after > len(records) or not supported and after:
        raise Rejected('query_page')
    page = records[after:after + 25]
    next_after = after + 25 if len(page) == 25 and after + 25 < len(records) else None
    body = {'schema': 1, 'profile': PROFILE, 'read_only': True, 'financial_authority': False, 'commands': 'unconnected',
            'view': view, 'selection': {'q': query, 'after': after}, 'audience': audience, 'member_count': len(members),
            'auth_scope': auth_scope, 'permission_revision': permission_revision, 'snapshot': snapshot, 'observed_at': at,
            'total': total, 'next_after': next_after, 'capabilities': capabilities, 'records': page}
    if len(json.dumps(body).encode('utf-8')) > MAX_RESPONSE_BYTES:
        raise Rejected('response_bound')
    return body


def receipts(actor, params, deliver):
    selection(params)
    with collection.access(actor, 'query') as (journal, coordinator, conn):
        if coordinator is None:
            raise Rejected('permission')
        with coordinator.a.transaction('query') as (gate, context):
            at = datetime.now(timezone.utc).isoformat()
            body = project(actor, params, coordinator.a, context, journal.state, at)
            return deliver(body)
