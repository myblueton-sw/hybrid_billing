import csv
import hashlib
import json
from collections import Counter
from datetime import datetime, timezone
from decimal import Decimal, Inexact, Rounded, localcontext
from pathlib import Path
from uuid import UUID, uuid5
from zoneinfo import ZoneInfo


ROOT = Path(__file__).resolve().parents[3]
NULLS = {'', 'NULL', 'null', '\\N'}
FIELDS = {
    'provider': 'ProviderName', 'service': 'ServiceName', 'currency': 'BillingCurrency',
    'account': 'BillingAccountId', 'subaccount': 'SubAccountId',
    'consumed_unit': 'ConsumedUnit', 'pricing_unit': 'PricingUnit',
    'billing_start': 'BillingPeriodStart', 'billing_end': 'BillingPeriodEnd',
    'charge_start': 'ChargePeriodStart', 'charge_end': 'ChargePeriodEnd',
    'billed_cost': 'BilledCost', 'effective_cost': 'EffectiveCost',
    'list_cost': 'ListCost', 'contracted_cost': 'ContractedCost',
    'consumed_quantity': 'ConsumedQuantity', 'pricing_quantity': 'PricingQuantity',
    'list_unit_price': 'ListUnitPrice', 'contracted_unit_price': 'ContractedUnitPrice',
}
MEASURES = ['billed_cost', 'effective_cost', 'list_cost', 'contracted_cost', 'consumed_quantity', 'pricing_quantity']
PRICES = ['list_unit_price', 'contracted_unit_price']
TIMES = ['billing_start', 'billing_end', 'charge_start', 'charge_end']
DIMENSIONS = ['provider', 'service', 'currency', 'consumed_unit', 'pricing_unit']
INSTALLATION_NAMESPACE = UUID('1a0ea32d-ec92-47f8-82c9-d4f7b97ed6fe')


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def instant(value):
    parsed = datetime.fromisoformat(value.replace('Z', '+00:00'))
    return parsed.replace(tzinfo=timezone.utc) if parsed.tzinfo is None else parsed.astimezone(timezone.utc)


def nullable(value):
    return None if value is None or value in NULLS else value


def coefficient(value):
    if value is None:
        return None
    with localcontext() as context:
        context.prec = 100
        context.traps[Inexact] = True
        context.traps[Rounded] = True
        scaled = Decimal(value) * Decimal(10) ** 18
        assert scaled == scaled.to_integral_value()
        return int(scaled)


def load():
    directory = ROOT / '.local/datasets/focus'
    data = (directory / 'six-provider.csv').read_bytes()
    provenance = json.loads((directory / 'six-provider.provenance.json').read_text())
    assert sha(data) == provenance['combined_sha256']
    assert provenance['declared_version'] == '1.0'
    config = json.loads((directory / 'six-provider.config.json').read_text())
    assert config['snapshot']['file_digest'] == sha(data)
    assert config['version'] == '1.0'
    rows = list(csv.DictReader(data.decode().splitlines(keepends=True)))
    assert len(rows) == config['snapshot']['row_count'] == provenance['combined_rows'] == 1123
    tenants = {t['key']: t for t in config['tenants']}
    for ordinal, source in enumerate(rows, 1):
        domain = [source['ProviderName'], source['BillingAccountId'], nullable(source['SubAccountId']),
            instant(source['BillingPeriodStart']).isoformat(), instant(source['BillingPeriodEnd']).isoformat()]
        owners = [t['key'] for t in config['tenants'] if t['active'] and domain in t['assignments']]
        assert len(owners) == 1
        source['_tenant'] = owners[0]
        source['_ordinal'] = ordinal
    return rows, tenants, config, provenance


def lexical(source):
    return {k: v for k, v in source.items() if not k.startswith('_')}


def fingerprint(source):
    return sha(canonical(lexical(source)).encode())


def row_counter(rows):
    return Counter((r['_tenant'], r['_window_id'], r['_batch_id'], r['_ordinal'], fingerprint(r)) for r in rows)


def delivery_rows(combined, config, status, directory):
    files = {}
    for path in directory.glob('*.csv'):
        files[sha(path.read_bytes())] = path
    output = []
    pins = []
    for window_id, window in status['windows'].items():
        if not window['current']:
            continue
        delivery = status['deliveries'][window['current']]
        descriptor = delivery['descriptor']
        path = files[descriptor['file_digest']]
        with path.open(newline='') as stream:
            rows = list(csv.DictReader(stream))
        assert len(rows) == descriptor['row_count']
        for ordinal, row in enumerate(rows, 1):
            domain = [row['ProviderName'], row['BillingAccountId'], nullable(row['SubAccountId']),
                instant(row['BillingPeriodStart']).isoformat(), instant(row['BillingPeriodEnd']).isoformat()]
            owners = [t['key'] for t in config['tenants'] if t['active'] and domain in t['assignments']]
            assert len(owners) == 1
            output.append({**row, '_tenant': owners[0], '_ordinal': ordinal,
                '_window_id': window_id, '_batch_id': delivery['batch_id']})
        pins.append({'sha256': descriptor['file_digest'], 'rows': len(rows), 'window': descriptor['window'],
            'revision': descriptor['revision'], 'batch_id': delivery['batch_id']})
    assert Counter((r['_tenant'], fingerprint(r)) for r in combined) == Counter((r['_tenant'], fingerprint(r)) for r in output)
    return output, pins


def groups(rows):
    result = {}
    for source in rows:
        key = canonical([nullable(source[FIELDS[n]]) for n in DIMENSIONS])
        result.setdefault(key, []).append(source)
    output = {}
    for key, members in result.items():
        measures = {}
        for name in MEASURES:
            values = [coefficient(nullable(r[FIELDS[name]])) for r in members]
            present = [v for v in values if v is not None]
            measures[name] = {'coefficient': sum(present) if present else None,
                'present': len(present), 'missing': len(members) - len(present)}
        output[key] = {'rows': len(members), 'measures': measures}
    return output


def periods(rows, zone, grain):
    partitions = {}
    for row in rows:
        key = instant(row['ChargePeriodStart']).astimezone(ZoneInfo(zone)).strftime('%Y-%m-%d' if grain == 'day' else '%Y-%m')
        partitions.setdefault(key, []).append(row)
    return {key: groups(partitions[key]) for key in sorted(partitions)}


def check_detail(row):
    source = json.loads(row['source_json'])
    for name, column in FIELDS.items():
        value = nullable(source.get(column))
        actual = row[name]
        if name in MEASURES + PRICES:
            assert coefficient(actual) == coefficient(value), name
        elif name in TIMES:
            assert instant(actual) == instant(value), name
        else:
            assert actual == value, name
    assert row['tenant_id'] == str(uuid5(INSTALLATION_NAMESPACE, row['tenant_key']))
    return {**source, '_tenant': row['tenant_key'], '_ordinal': row['ordinal'],
        '_window_id': row['window_id'], '_batch_id': row['batch_id']}


def use_cases(rows):
    dimensions = {
        'service_billing_account': ['ProviderName', 'BillingAccountId', 'BillingCurrency', 'ServiceName', 'BillingPeriodStart', 'BillingPeriodEnd'],
        'service_subaccount': ['ProviderName', 'SubAccountId', 'BillingCurrency', 'ServiceName'],
        'charge_category': ['ProviderName', 'BillingCurrency', 'ChargeCategory', 'ChargeClass'],
        'service_category': ['ProviderName', 'BillingCurrency', 'ServiceCategory'],
        'resource_sku': ['ProviderName', 'BillingCurrency', 'ServiceName', 'ResourceId', 'SkuId'],
    }
    output = {}
    for name, columns in dimensions.items():
        buckets = {}
        for row in rows:
            token = canonical([nullable(row.get(k)) for k in columns])
            buckets.setdefault(token, []).append(row)
        aggregates = {}
        for token, members in buckets.items():
            aggregates[token] = {}
            for field in ['BilledCost', 'EffectiveCost']:
                values = [coefficient(nullable(r.get(field))) for r in members]
                aggregates[token][field] = {'coefficient': sum(v for v in values if v is not None) if any(v is not None for v in values) else None,
                    'present': sum(v is not None for v in values), 'missing': sum(v is None for v in values)}
        output[name] = {'dimensions': columns, 'groups': len(aggregates), 'digest': sha(canonical(aggregates).encode())}
    return output
