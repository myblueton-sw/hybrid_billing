import copy
import csv
import io
import json
from fractions import Fraction

from preflight import ROOT
from modules.focus_import import collection, lifecycle, service
from modules.focus_import.model import canonical, decode, digest, domain, domains_digest, versions
from modules.focus_import.temporal import window


KEYS = ['msp32-a1', 'msp32-a2', 'msp32-b']
ACTORS = {key: 'reader-' + key for key in KEYS}
COMPANY = 'reader-msp32-company-a'
START = '2026-08-01T00:00:00Z'
END = '2026-09-01T00:00:00Z'
B_START = END
B_END = '2026-10-01T00:00:00Z'


def encoded(rows):
    with (ROOT / 'tests/fixtures/focus/synthetic-1.3.csv').open(newline='') as stream:
        template = next(csv.DictReader(stream))
    fields = list(template) + ['x_PrivateMarker']
    output = io.StringIO(newline='')
    writer = csv.DictWriter(output, fieldnames=fields)
    writer.writeheader()
    writer.writerows(rows)
    return output.getvalue().encode()


def make_row(key, provider, cost, suffix, marker=None):
    with (ROOT / 'tests/fixtures/focus/synthetic-1.3.csv').open(newline='') as stream:
        row = next(csv.DictReader(stream))
    start, end = (B_START, B_END) if key == KEYS[2] else (START, END)
    charge_end = '2026-09-02T00:00:00Z' if key == KEYS[2] else '2026-08-02T00:00:00Z'
    row.update(ServiceProviderName=provider, ProviderName='Synthetic Legacy Provider',
        BillingAccountId='synthetic-shared-payer', SubAccountId='synthetic-sub-' + key,
        BillingPeriodStart=start, BillingPeriodEnd=end, ChargePeriodStart=start, ChargePeriodEnd=charge_end,
        ServiceName='Synthetic Shared Compute', BilledCost=str(cost), EffectiveCost=str(cost),
        ResourceId='synthetic-private-' + key + '-' + suffix,
        Tags=canonical({'customer': key}), x_PrivateMarker=marker or 'PRIVATE-' + key + '-' + suffix)
    return row


def seed_rows():
    return {
        KEYS[0]: [make_row(KEYS[0], 'AWS', 10, '1'), make_row(KEYS[0], 'AWS', 12, '2'), make_row(KEYS[0], 'Google Cloud', 3, '3')],
        KEYS[1]: [make_row(KEYS[1], 'AWS', 7, '1')],
        KEYS[2]: [make_row(KEYS[2], 'AWS', 100, '1'), make_row(KEYS[2], 'Google Cloud', 200, '2')],
    }


def configuration(rows):
    data = encoded([row for part in rows.values() for row in part])
    normalized = decode(data, versions()['1.3'], conditional=True)
    tenants = []
    for key in KEYS:
        own = decode(encoded(rows[key]), versions()['1.3'], conditional=True)
        tenants.append({'key': key, 'label': 'Same Tenant Label', 'company_id': 'company-b' if key == KEYS[2] else 'company-a',
            'company_label': 'Equal Company', 'timezone': 'UTC', 'reporting_currency': 'USD', 'currencies': ['USD'],
            'retention_days': 36500, 'actor': ACTORS[key], 'active': True,
            'assignments': [json.loads(value) for value in sorted({canonical(domain(row)) for row in own})]})
    return {'source': 'hb32-msp-isolation', 'version': '1.3', 'revision': 1, 'tenants': tenants,
        'snapshot': {'file_digest': digest(data), 'row_count': 6, 'domains_digest': domains_digest(normalized)},
        'views': [{'actor': COMPANY, 'company_id': 'company-a', 'tenant_keys': KEYS[:2], 'active': True}]}


def connection(key, rows):
    source = decode(encoded(rows), versions()['1.3'], conditional=True)
    start, end = (B_START, B_END) if key == KEYS[2] else (START, END)
    return {'id': key + '-feed', 'dataset': 'cost_usage', 'coverage_key': 'primary', 'version': '1.3',
        'grain': 'month', 'start': start, 'end': end, 'lag_seconds': 0,
        'domains': [json.loads(value) for value in sorted({canonical([r['provider'], r['account'], r['subaccount']]) for r in source})],
        'semantic_profile': 'focus-consumed-v1', 'max_replacements': 2}


def descriptor(key, rows, revision=1):
    data = encoded(rows)
    normalized = decode(data, versions()['1.3'], conditional=True)
    start, end = (B_START, B_END) if key == KEYS[2] else (START, END)
    return {'connection_id': key + '-feed', 'window': window(start, end, 'month'), 'revision': revision,
        'file_digest': digest(data), 'row_count': len(rows), 'domains_digest': domains_digest(normalized),
        'complete_through': end, 'finalized': True, 'zero_domains': []}


def publish(key, rows, revision=1):
    data = encoded(rows)
    registered = collection.register('operator', data, '1.3', descriptor(key, rows, revision))
    assert registered.get('delivery_id')
    collected = collection.collect('importer', registered['delivery_id'], data)
    collection.confirm('approver', registered['delivery_id'], collected['manifest_digest'])
    collection.publish('approver', registered['delivery_id'])
    return {**registered, **collected}


def initialize():
    from runtime import reserve_roles
    rows = seed_rows()
    config = configuration(rows)
    reserve_roles(config)
    service.initialize(config)
    policy = {'revision': 1, 'basis': 'window_end', 'hot_days': 1, 'total_days': 36500,
        'grace_days': 0, 'backup_days': 36500, 'reject_before': '2020-01-01T00:00:00Z', 'approved_by': 'lifecycle-approver'}
    lifecycle.activate('operator', 'month', {key: copy.deepcopy(policy) for key in KEYS})
    batches = {}
    for key in KEYS:
        collection.configure_connection('operator', connection(key, rows[key]))
        batches[key] = publish(key, rows[key])
    return config, rows, batches


def billed(result):
    return sum((Fraction(group['measures']['billed_cost']['coefficient'] or 0, 10**18) for group in result['groups'].values()), Fraction())
