import json
import re
from datetime import datetime, timezone

from . import collection, connections
from .model import Rejected, canonical, digest, versions
from .receipt_metadata import authorized_connections, configuration, members_for
from .temporal import utc


PROFILE = 'retained-focus-connections-v1'
MAX_RESPONSE_BYTES = 2 * 1024 * 1024
VIEWS = {'connections', 'capabilities', 'security', 'diagnostics'}
QUERY_KEYS = {'view', 'q', 'after', 'snapshot', 'permission_revision'}
CAPABILITIES = {'connections': 'fully-owned-registrations-only', 'capability_evidence': 'registration-only',
                'provider_authentication': 'unconnected', 'diagnostics': 'unconnected'}


def selection(params):
    if not isinstance(params, dict) or set(params) - QUERY_KEYS or any(not isinstance(v, str) for v in params.values()):
        raise Rejected('request_invalid')
    view, query, after = params.get('view', 'connections'), params.get('q', ''), params.get('after', '0')
    if view not in VIEWS or not re.fullmatch('[a-z0-9:_-]{0,128}', query) or not re.fullmatch('0|[1-9][0-9]{0,4}', after) or int(after) > 10000:
        raise Rejected('request_invalid')
    if any(key in params and not re.fullmatch('[a-f0-9]{64}', params[key]) for key in ['snapshot', 'permission_revision']):
        raise Rejected('request_invalid')
    if ('snapshot' in params) != ('permission_revision' in params) or int(after) and 'snapshot' not in params:
        raise Rejected('snapshot_required')
    return view, query, int(after)


def fail():
    raise Rejected('connection_evidence_unavailable')


def number(value, minimum, maximum):
    if type(value) is not int or not minimum <= value <= maximum:
        fail()
    return str(value)


def registration(connection, source, registry):
    cid = connection.get('id')
    version, grain, semantic = connection.get('version'), connection.get('grain'), connection.get('semantic_profile')
    if not connections.token(cid) or not isinstance(version, str) or not re.fullmatch(r'\d{1,2}\.\d{1,2}', version) or version not in registry:
        fail()
    if not isinstance(grain, str) or grain not in {'day', 'month'} or not isinstance(semantic, str) or semantic not in {'focus-consumed-v1', 'exploratory-query-only'}:
        fail()
    try:
        start, end = utc(connection.get('start')).isoformat(), utc(connection.get('end')).isoformat()
        connections.calendar({'start': start, 'end': end, 'grain': grain})
    except (Rejected, TypeError, ValueError, OverflowError):
        fail()
    return {'id': digest(canonical({'profile': PROFILE, 'source': source, 'connection': cid}).encode()),
            'connection': cid, 'focus_version': version, 'grain': grain, 'window_start': start, 'window_end': end,
            'registered_domains': number(len(connection['domains']), 1, 10000), 'semantic_profile': semantic,
            'allowed_lateness_seconds': number(connection.get('lag_seconds'), 0, 604800),
            'replacement_limit': number(connection.get('max_replacements'), 1, 10)}


def project(actor, params, authority, context, state, at):
    view, query, after = selection(params)
    if authority.actor != actor:
        raise Rejected('permission')
    config = configuration(state, authority)
    audience, members = members_for(authority, context, config)
    source = config.get('source')
    if not isinstance(source, str) or not source:
        fail()
    owned = authorized_connections(state, config, members)
    registry = versions()
    records = [registration(value, source, registry) for value in owned.values()]
    policies = []
    for member in sorted(members):
        policy = state.get('policies', {}).get(members[member]['key'])
        if not isinstance(policy, dict):
            fail()
        policies.append({'member': member, 'revision': number(policy.get('revision'), 1, 2**63 - 1)})
    auth_scope = digest(canonical({'profile': PROFILE, 'source': source, 'actor': actor,
                                   'audience': audience, 'members': sorted(members)}).encode())
    permission_revision = digest(canonical({'auth_scope': auth_scope, 'policies': policies}).encode())
    supported = view in {'connections', 'capabilities'}
    records = [r for r in records if r['id'].startswith(query) or r['connection'].startswith(query)] if supported else []
    records.sort(key=lambda r: r['connection'])
    if len(records) > 10000:
        raise Rejected('selection_unavailable')
    total = len(records) if supported else None
    snapshot = digest(canonical({'profile': PROFILE, 'permission_revision': permission_revision,
                                'view': view, 'q': query, 'capabilities': CAPABILITIES,
                                'total': total, 'records': records}).encode())
    if 'snapshot' in params and (params['snapshot'] != snapshot or params['permission_revision'] != permission_revision):
        raise Rejected('query_snapshot_changed')
    if after > len(records) or not supported and after:
        raise Rejected('query_page')
    page = records[after:after + 25]
    next_after = after + 25 if len(page) == 25 and after + 25 < len(records) else None
    body = {'schema': 1, 'profile': PROFILE, 'read_only': True, 'financial_authority': False, 'commands': 'unconnected',
            'view': view, 'selection': {'q': query, 'after': after}, 'audience': audience, 'member_count': len(members),
            'auth_scope': auth_scope, 'permission_revision': permission_revision, 'snapshot': snapshot, 'observed_at': at,
            'total': total, 'next_after': next_after, 'capabilities': dict(CAPABILITIES), 'records': page}
    if len(json.dumps(body).encode('utf-8')) > MAX_RESPONSE_BYTES:
        raise Rejected('response_bound')
    return body


def connection_metadata(actor, params, deliver):
    selection(params)
    with collection.access(actor, 'query') as (journal, coordinator, conn):
        if coordinator is None:
            raise Rejected('permission')
        with coordinator.a.transaction('query') as (gate, context):
            body = project(actor, params, coordinator.a, context, journal.state, datetime.now(timezone.utc).isoformat())
            return deliver(body)
