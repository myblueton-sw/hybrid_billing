import re
from datetime import datetime, timedelta, timezone as utc_timezone
from zoneinfo import ZoneInfo, ZoneInfoNotFoundError

from .model import ADDITIVE, LIMIT, MAX_ROWS, Rejected, canonical, coefficient, controls, decode, digest, domains_digest, instant, tenant_id, validate_config, versions


DELIVERY_FIELDS = {'window', 'revision', 'file_digest', 'row_count', 'domains_digest', 'complete_through', 'finalized'}
POLICY_FIELDS = {'revision', 'basis', 'hot_days', 'total_days', 'grace_days', 'backup_days', 'reject_before', 'approved_by'}


def utc(text):
    if isinstance(text, datetime):
        if text.tzinfo is None or text.utcoffset() != timedelta(0):
            raise Rejected('explicit_utc_required')
        return text.astimezone(utc_timezone.utc)
    if not isinstance(text, str) or not re.search(r'(?:Z|\+00:00)$', text):
        raise Rejected('explicit_utc_required')
    return instant(text)


def window(start, end, grain):
    begin, finish = utc(start), utc(end)
    if grain not in ['day', 'month'] or begin.time().isoformat() != '00:00:00' or finish.time().isoformat() != '00:00:00':
        raise Rejected('delivery_window')
    try:
        expected = begin + timedelta(days=1) if grain == 'day' else begin.replace(year=begin.year + (begin.month == 12), month=begin.month % 12 + 1)
    except (ValueError, OverflowError):
        raise Rejected('delivery_window') from None
    if finish != expected or grain == 'month' and begin.day != 1:
        raise Rejected('delivery_window')
    body = {'start': begin.isoformat(), 'end': finish.isoformat(), 'grain': grain}
    return {**body, 'key': digest(canonical(body).encode())}


def checked_window(value):
    if not isinstance(value, dict) or set(value) != {'start', 'end', 'grain', 'key'}:
        raise Rejected('delivery_window')
    result = window(value['start'], value['end'], value['grain'])
    if value != result:
        raise Rejected('delivery_window')
    return result


def overlaps(left, right):
    left, right = checked_window(left), checked_window(right)
    return utc(left['start']) < utc(right['end']) and utc(right['start']) < utc(left['end'])


def validate_delivery(data, version, config, descriptor):
    if not isinstance(descriptor, dict) or set(descriptor) != DELIVERY_FIELDS:
        raise Rejected('delivery_descriptor')
    scope = checked_window(descriptor['window'])
    if type(descriptor['revision']) is not int or not 0 < descriptor['revision'] < 2**63:
        raise Rejected('delivery_revision')
    if type(descriptor['row_count']) is not int or not 0 <= descriptor['row_count'] <= MAX_ROWS or type(descriptor['finalized']) is not bool:
        raise Rejected('delivery_descriptor')
    if any(not isinstance(descriptor[n], str) or not re.fullmatch('[a-f0-9]{64}', descriptor[n]) for n in ['file_digest', 'domains_digest']):
        raise Rejected('delivery_descriptor')
    begin, finish, through = utc(scope['start']), utc(scope['end']), utc(descriptor['complete_through'])
    if not begin < through <= finish or descriptor['finalized'] and through != finish:
        raise Rejected('delivery_completeness')
    registry = versions()
    if not isinstance(version, str) or version not in registry:
        raise Rejected('unsupported_version')
    validate_config(config)
    if not isinstance(data, bytes) or digest(data) != descriptor['file_digest']:
        raise Rejected('delivery_digest')
    try:
        rows = decode(data, registry[version])
    except Rejected as exc:
        if str(exc) != 'empty_input' or descriptor['row_count'] != 0:
            raise
        rows = []
    if len(rows) != descriptor['row_count'] or domains_digest(rows) != descriptor['domains_digest']:
        raise Rejected('delivery_coverage')
    owners = {}
    for tenant in config['tenants']:
        for assignment in tenant['assignments']:
            key = canonical(assignment[:3])
            previous = owners.get(key)
            if previous is not None and previous['key'] != tenant['key']:
                raise Rejected('ambiguous_delivery_owner')
            owners[key] = tenant
    for row in rows:
        tenant = owners.get(canonical([row[n] for n in ['provider', 'account', 'subaccount']]))
        if not tenant or not tenant['active'] or row['currency'] not in tenant['currencies']:
            raise Rejected('unmapped_or_denied_source')
        start_name, end_name = ('charge_start', 'charge_end') if scope['grain'] == 'day' else ('billing_start', 'billing_end')
        if row[start_name] < begin or row[end_name] > finish:
            raise Rejected('row_outside_delivery_window')
        if scope['grain'] == 'day' and row['charge_end'] > through:
            raise Rejected('row_after_complete_through')
        row['tenant_id'] = tenant_id(tenant['key'])
    return rows


def validate_policy(policy):
    if not isinstance(policy, dict) or set(policy) != POLICY_FIELDS:
        raise Rejected('retention_policy')
    if type(policy['revision']) is not int or not 0 < policy['revision'] < 2**63 or policy['basis'] not in ['received_at', 'window_end']:
        raise Rejected('retention_policy')
    if any(type(policy[n]) is not int or not 0 <= policy[n] <= 36500 for n in ['hot_days', 'total_days', 'grace_days', 'backup_days']):
        raise Rejected('retention_policy')
    if policy['hot_days'] > policy['total_days'] or policy['total_days'] < 1:
        raise Rejected('retention_policy')
    if not isinstance(policy['approved_by'], str) or not re.fullmatch('[a-z][a-z0-9-]{0,63}', policy['approved_by']):
        raise Rejected('retention_policy')
    utc(policy['reject_before'])
    return policy


def eligible(origin, policy, now):
    validate_policy(policy)
    origin = utc(origin)
    utc(now)
    try:
        hot_until = origin + timedelta(days=policy['hot_days'])
        delete_after = origin + timedelta(days=policy['total_days'] + policy['grace_days'])
        backup_until = delete_after + timedelta(days=policy['backup_days'])
    except (ValueError, OverflowError):
        raise Rejected('retention_date_range') from None
    return {'hot_until': hot_until.isoformat(), 'delete_after': delete_after.isoformat(), 'backup_until': backup_until.isoformat()}


def summarize(rows, timezone, grain):
    if grain not in ['day', 'month']:
        raise Rejected('reporting_grain')
    try:
        zone = ZoneInfo(timezone)
    except (ValueError, TypeError, ZoneInfoNotFoundError):
        raise Rejected('reporting_timezone') from None
    if len(rows) > MAX_ROWS:
        raise Rejected('selection_row_limit')
    for name in ADDITIVE:
        if sum(abs(coefficient(row[name])) for row in rows if row[name] is not None) >= LIMIT:
            raise Rejected('selection_aggregate_bound')
    periods = {}
    for row in rows:
        start = row['charge_start']
        if not isinstance(start, datetime) or start.tzinfo is None:
            raise Rejected('reporting_timestamp')
        local = start.astimezone(zone)
        key = local.strftime('%Y-%m-%d' if grain == 'day' else '%Y-%m')
        periods.setdefault(key, []).append(row)
    return {key: controls(periods[key]) for key in sorted(periods)}
