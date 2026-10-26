import csv
import copy
import re
from datetime import timedelta

from .model import Rejected, canonical, digest, tenant_id
from .temporal import utc, window


OPTIONAL_DELIVERY = {'connection_id', 'zero_domains', 'reopen_plan', 'replacement_plan'}


def token(value):
    return isinstance(value, str) and re.fullmatch('[a-z][a-z0-9_-]{0,63}', value)


def owners(config):
    result = {}
    for tenant in config['tenants']:
        if not tenant['active']:
            continue
        for assignment in tenant['assignments']:
            key = canonical(assignment[:3])
            if key in result and result[key]['key'] != tenant['key']:
                raise Rejected('ambiguous_delivery_owner')
            result[key] = tenant
    return result


def calendar(body):
    start, end = utc(body['start']), utc(body['end'])
    if start >= end:
        raise Rejected('connection_calendar')
    result = []
    while start < end:
        if len(result) >= 3660:
            raise Rejected('connection_calendar_bound')
        finish = start + timedelta(days=1) if body['grain'] == 'day' else start.replace(
            year=start.year + (start.month == 12), month=start.month % 12 + 1, day=1)
        result.append(window(start.isoformat(), finish.isoformat(), body['grain']))
        start = finish
    if start != end:
        raise Rejected('connection_calendar')
    return result


def definition(body, config):
    fields = {'id', 'dataset', 'coverage_key', 'version', 'grain', 'start', 'end', 'lag_seconds',
        'domains', 'semantic_profile', 'max_replacements'}
    if not isinstance(body, dict) or set(body) - {'capabilities'} != fields:
        raise Rejected('connection_definition')
    if any(not token(body[k]) for k in ['id', 'dataset', 'coverage_key']):
        raise Rejected('connection_identity')
    if body['grain'] not in ['day', 'month'] or type(body['lag_seconds']) is not int or not 0 <= body['lag_seconds'] <= 604800:
        raise Rejected('connection_schedule')
    if type(body['max_replacements']) is not int or not 1 <= body['max_replacements'] <= 10:
        raise Rejected('connection_replacement_bound')
    from .model import versions
    if body['version'] not in versions() or body['semantic_profile'] not in ['focus-consumed-v1', 'exploratory-query-only']:
        raise Rejected('connection_profile')
    body = copy.deepcopy(body)
    body['start'], body['end'] = utc(body['start']).isoformat(), utc(body['end']).isoformat()
    calendar(body)
    allowed = owners(config)
    if not isinstance(body['domains'], list) or not 1 <= len(body['domains']) <= 10000:
        raise Rejected('connection_domains')
    seen = set()
    for domain in body['domains']:
        if not isinstance(domain, list) or len(domain) != 3 or any(
                not isinstance(v, str) or not 1 <= len(v) <= 4096 for v in domain[:2]) or domain[2] is not None and (
                not isinstance(domain[2], str) or not 1 <= len(domain[2]) <= 4096):
            raise Rejected('connection_domains')
        key = canonical(domain)
        if key not in allowed or key in seen:
            raise Rejected('connection_domain_authority')
        seen.add(key)
    if body['semantic_profile'] == 'focus-consumed-v1':
        from .semantics import profile
        body['semantic_definition'] = profile(body['version'], body.get('capabilities'))
    elif body.get('capabilities'):
        raise Rejected('connection_profile')
    return body


def add(state, body, config):
    body = definition(body, config)
    existing = state.setdefault('connections', {}).get(body['id'])
    if existing:
        if existing != body:
            raise Rejected('connection_immutable')
        return body
    incoming = {canonical(d) for d in body['domains']}
    for previous in state['connections'].values():
        overlap = body['start'] < previous['end'] and previous['start'] < body['end']
        if overlap and incoming & {canonical(d) for d in previous['domains']}:
            raise Rejected('connection_economic_overlap')
    for previous in state['windows'].values():
        if not previous.get('connection_id') and body['start'] < previous['window']['end'] and previous['window']['start'] < body['end']:
            raise Rejected('connection_legacy_overlap_requires_adoption')
    state['connections'][body['id']] = body
    return body


def get(state, descriptor):
    if not isinstance(descriptor,dict):raise Rejected('delivery_descriptor')
    cid = descriptor.get('connection_id')
    if cid is None:
        if set(descriptor) & OPTIONAL_DELIVERY:
            raise Rejected('connection_required')
        return None
    if not token(cid):raise Rejected('connection_identity')
    result = state.get('connections', {}).get(cid)
    if result is None:
        raise Rejected('connection_unregistered')
    from .temporal import checked_window
    w = checked_window(descriptor.get('window'))
    if w['grain'] != result['grain'] or w['start'] < result['start'] or w['end'] > result['end']:
        raise Rejected('connection_window')
    return result


def identity(connection, scope):
    if connection is None:
        return scope['key']
    return digest(canonical([connection['id'], connection['dataset'], connection['coverage_key'], scope['key']]).encode())


def admission(rows, connection, descriptor, version, complete=True):
    if version != connection['version']:
        raise Rejected('connection_version')
    expected = {canonical(d) for d in connection['domains']}
    observed = {canonical([r[n] for n in ['provider', 'account', 'subaccount']]) for r in rows}
    if not observed <= expected:
        raise Rejected('connection_domain_authority')
    zeros = descriptor.get('zero_domains', [])
    if not isinstance(zeros, list):
        raise Rejected('zero_attestation')
    zero_keys = set()
    for item in zeros:
        if not isinstance(item, dict) or set(item) != {'domain', 'reason', 'evidence_digest'} or not token(item['reason']) or not isinstance(item['evidence_digest'], str) or not re.fullmatch('[a-f0-9]{64}', item['evidence_digest']):
            raise Rejected('zero_attestation')
        key = canonical(item['domain'])
        if key not in expected or key in observed or key in zero_keys:
            raise Rejected('zero_attestation_scope')
        zero_keys.add(key)
    if complete and observed | zero_keys != expected:
        raise Rejected('expected_domain_missing')
    return zeros


def semantic(rows, connection):
    if connection['semantic_profile'] == 'exploratory-query-only':
        return None
    from .semantics import validate_rows
    return validate_rows(rows, connection['version'], connection['semantic_definition'])


def coverage(state, members, config, start, end, at):
    allowed = owners(config)
    result = []
    for connection in state.get('connections', {}).values():
        domains = [(d, allowed.get(canonical(d))) for d in connection['domains']]
        visible = [(d, t) for d, t in domains if t and str(tenant_id(t['key'])) in members]
        if not visible:
            continue
        for w in calendar(connection):
            if start and w['end'] <= start or end and w['start'] >= end:
                continue
            key = identity(connection, w)
            target = state['windows'].get(key)
            latest = state['deliveries'][target['revisions'][-1]] if target and target['revisions'] else None
            current = state['deliveries'][target['current']] if target and target['current'] else None
            due = (utc(w['end']) + timedelta(seconds=connection['lag_seconds'])).isoformat()
            for domain, tenant in visible:
                if len(result) >= 10000:
                    raise Rejected('coverage_selection_bound')
                pending=[q for q in state.get('receipts',{}).values() if q.get('admitted') and q['connection_id']==connection['id'] and q.get('descriptor') and q['descriptor']['window']==w and tenant['key'] in q.get('tenant_keys',[]) and q['state'] in ['quarantined','replacement-requested','escalated']]
                stage = pending[-1]['state'] if pending else latest['state'] if latest else 'late' if utc(at) > utc(due) else 'missing'
                zeros = current['descriptor'].get('zero_domains', []) if current else []
                zero = next((z for z in zeros if z['domain'] == domain), None)
                result.append({'connection_id': connection['id'], 'dataset': connection['dataset'], 'window': w,
                    'domain': domain, 'tenant_key': tenant['key'], 'due_at': due, 'state': stage,
                    'published': current is not None, 'finalized': current['descriptor']['finalized'] if current else False,
                    'complete_through': current['descriptor']['complete_through'] if current else None,
                    'known_zero': zero is not None, 'zero_evidence': zero, 'upstream_verification': False,
                    'expectation_basis': 'operator-registered-calendar-and-domains'})
    return result


def source_scopes(data,mapping,consumed=False):
    try:return _source_scopes(data,mapping,consumed)
    except (UnicodeError,csv.Error,ValueError,TypeError):raise Rejected('source_format') from None


def _source_scopes(data,mapping,consumed=False):
    from .intake import sources
    from .model import TIMES,instant,MAX_ROWS
    tokens=['',r'\N'] if consumed else mapping['null_tokens']
    required=[mapping['columns'][n] for n in ['provider','account','currency']+TIMES]
    selected={**mapping,'required_columns':required}
    rows=[]
    for ordinal,source in enumerate(sources(data,selected),1):
        if ordinal>MAX_ROWS:raise Rejected('source_shape')
        row={'ordinal':ordinal}
        for name in ['provider','account','subaccount','currency']+TIMES:
            value=source.get(mapping['columns'][name])
            if value is None or value in tokens:value=None
            row[name]=instant(value) if name in TIMES else value
        if row['provider'] is None or row['account'] is None or row['billing_start']>=row['billing_end'] or row['charge_start']>=row['charge_end']:
            raise Rejected('required_source_scope')
        rows.append(row)
    return rows


def admit_source(data,version,config,descriptor,connection):
    from .model import versions,domains_digest
    from .temporal import checked_window,DELIVERY_FIELDS
    if not isinstance(descriptor,dict) or set(descriptor)-OPTIONAL_DELIVERY!=DELIVERY_FIELDS:raise Rejected('delivery_descriptor')
    if type(descriptor['revision']) is not int or not 0<descriptor['revision']<2**63 or type(descriptor['finalized']) is not bool:raise Rejected('delivery_descriptor')
    if version not in versions() or version!=connection['version']:raise Rejected('connection_version')
    if not isinstance(data,bytes) or digest(data)!=descriptor.get('file_digest'):raise Rejected('delivery_digest')
    w=checked_window(descriptor['window'])
    through=utc(descriptor['complete_through'])
    if not utc(w['start'])<through<=utc(w['end']) or descriptor['finalized'] and through!=utc(w['end']):raise Rejected('delivery_completeness')
    rows=source_scopes(data,versions()[version],consumed=bool(connection.get('semantic_definition')))
    if type(descriptor.get('row_count')) is not int or len(rows)!=descriptor['row_count'] or domains_digest(rows)!=descriptor.get('domains_digest'):
        raise Rejected('delivery_coverage')
    allowed=owners(config)
    for row in rows:
        tenant=allowed.get(canonical([row[n] for n in ['provider','account','subaccount']]))
        if not tenant or row['currency'] not in tenant['currencies']:raise Rejected('unmapped_or_denied_source')
        begin,end=('charge_start','charge_end') if w['grain']=='day' else ('billing_start','billing_end')
        if row[begin]<utc(w['start']) or row[end]>utc(w['end']):raise Rejected('row_outside_delivery_window')
        if w['grain']=='day' and row['charge_end']>utc(descriptor['complete_through']):raise Rejected('row_after_complete_through')
        row['tenant_id']=tenant_id(tenant['key'])
    admission(rows,connection,descriptor,version,complete=False)
    return rows
