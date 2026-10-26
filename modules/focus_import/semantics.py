import copy
import json
import re
from datetime import datetime
from decimal import Decimal

from modules.focus_local.contract import key_value, predicate, validate_predicate
from .model import MAX_ROWS, ROOT, Rejected, canonical, coefficient, digest, fixed, instant, number


REGISTRY = ROOT / 'packages/contracts/focus_import/consumed.json'
DECIMALS = {'billed_cost', 'effective_cost', 'list_cost', 'contracted_cost', 'consumed_quantity', 'pricing_quantity', 'list_unit_price', 'contracted_unit_price'}
TIMESTAMPS = {'billing_period_start', 'billing_period_end', 'charge_period_start', 'charge_period_end'}
STRINGS = {'billing_account_id', 'sub_account_id', 'billing_currency', 'charge_category', 'charge_class', 'service_name', 'service_category', 'commitment_discount_status', 'sku_price_id', 'resource_id', 'provider_name_legacy', 'publisher_name_legacy', 'invoice_issuer_name', 'consumed_unit', 'pricing_unit', 'service_provider_name', 'host_provider_name'}
NAMES = DECIMALS | TIMESTAMPS | STRINGS | {'tags'}
CAPABILITIES_V1 = {'measured_usage', 'list_prices', 'contracted_prices', 'host_null_services'}
CONSTRUCTS = {'tags': 'tags', 'resource_id': 'resource_ids', 'commitment_discount_status': 'commitment_discounts', 'sku_price_id': 'sku_prices', 'sub_account_id': 'subaccounts'}
CAPABILITIES = CAPABILITIES_V1 | set(CONSTRUCTS.values())
NUMBER = re.compile(r'-?\d+(?:\.\d+)?(?:[eE]-?\d+)?\Z')
STAMP = re.compile(r'\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,6})?Z\Z')
STAMP_LEGACY = re.compile(r'\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,6})?(?:Z|[+-]\d\d:\d\d)\Z')


def capabilities(value, revision=2):
    expected = CAPABILITIES_V1 if revision == 1 else CAPABILITIES
    if not isinstance(value, dict) or set(value) != expected:
        raise Rejected('semantic_capabilities')
    if any(type(value[k]) is not bool for k in expected - {'host_null_services'}):
        raise Rejected('semantic_capabilities')
    names = value['host_null_services']
    if not isinstance(names, list) or len(names) > 100 or any(not isinstance(n, str) or not 1 <= len(n) <= 4096 for n in names) or len(set(names)) != len(names):
        raise Rejected('semantic_capabilities')
    return value


def profile(version, capabilities=None):
    try:
        if REGISTRY.stat().st_size > 256000:
            raise Rejected('semantic_registry')
        registry = json.loads(REGISTRY.read_text())
        if set(registry) != {'versions'} or version not in registry['versions']:
            raise Rejected('semantic_version')
        definition = copy.deepcopy(registry['versions'][version])
        if capabilities is not None:
            definition['capabilities'] = copy.deepcopy(capabilities)
            if definition['validator_revision'] == 2 and set(capabilities) == CAPABILITIES_V1:
                definition['capabilities'].update({name: True for name in CONSTRUCTS.values()})
        result = {'id': 'focus-consumed-v1', 'version': version, 'definition': definition,
            'digest': digest(canonical(definition).encode())}
        return checked_profile(result, version)
    except (OSError, ValueError, TypeError, KeyError):
        raise Rejected('semantic_registry') from None


def checked_profile(pinned, version):
    try:
        if not isinstance(pinned, dict) or set(pinned) != {'id', 'version', 'definition', 'digest'} or pinned['id'] != 'focus-consumed-v1' or pinned['version'] != version:
            raise Rejected('semantic_profile')
        body = pinned['definition']
        if not isinstance(body, dict) or set(body) != {'version', 'revision', 'validator_revision', 'numeric', 'null_tokens', 'sources', 'fields', 'checks', 'capabilities'}:
            raise Rejected('semantic_profile')
        if len(canonical(body)) > 256000 or body['version'] != version or not re.fullmatch(r'\d+\.\d+', version) or type(body['revision']) is not int or body['revision'] < 1 or type(body['validator_revision']) is not int or body['validator_revision'] not in [1, 2]:
            raise Rejected('semantic_profile')
        if body['numeric'] != {'precision': 38, 'scale': 18, 'grammar': 'focus-numeric-v1'}:
            raise Rejected('semantic_profile')
        if digest(canonical(body).encode()) != pinned['digest']:
            raise Rejected('semantic_profile_digest')
        capabilities(body['capabilities'], body['validator_revision'])
        if not isinstance(body['null_tokens'], list) or any(not isinstance(v, str) or len(v) > 16 for v in body['null_tokens']):
            raise Rejected('semantic_profile')
        if not isinstance(body['sources'], list) or not body['sources'] or any(not isinstance(v, str) or len(v) > 4096 for v in body['sources']):
            raise Rejected('semantic_profile')
        fields = body['fields']
        if not isinstance(fields, list) or len(fields) != len(NAMES) or {f['name'] for f in fields} != NAMES:
            raise Rejected('semantic_profile')
        for f in fields:
            if not {'source', 'name', 'type', 'header', 'disclosure', 'functions'} <= set(f) or set(f) - {'source', 'name', 'type', 'header', 'disclosure', 'functions', 'enum', 'nonnull_when', 'null_when', 'aggregate', 'nonnegative'}:
                raise Rejected('semantic_profile')
            expected = 'decimal' if f['name'] in DECIMALS else 'timestamp' if f['name'] in TIMESTAMPS else 'json' if f['name'] == 'tags' else 'string'
            if f['type'] != expected or type(f['header']) is not bool or f['disclosure'] != 'tenant-protected' or not isinstance(f['functions'], list) or not set(f['functions']) <= {'scoped-detail', 'exact-control'}:
                raise Rejected('semantic_profile')
            if f['source'] is not None and (not isinstance(f['source'], str) or not re.fullmatch(r'[A-Za-z][A-Za-z0-9_]{0,63}', f['source'])) or f['source'] is None and f['header']:
                raise Rejected('semantic_profile')
            if 'enum' in f and (not isinstance(f['enum'], list) or not f['enum'] or any(not isinstance(v, str) or len(v) > 4096 for v in f['enum'])):
                raise Rejected('semantic_profile')
            if 'nonnegative' in f and type(f['nonnegative']) is not bool:
                raise Rejected('semantic_profile')
            for key in ['null_when', 'nonnull_when']:
                if key in f:
                    validate_predicate(f[key], NAMES)
        if not isinstance(body['checks'], list) or len(body['checks']) > 32:
            raise Rejected('semantic_profile')
        for check in body['checks']:
            if set(check) != {'product', 'when'} or not isinstance(check['product'], list) or len(check['product']) != 3 or not set(check['product']) <= DECIMALS:
                raise Rejected('semantic_profile')
            validate_predicate(check['when'], NAMES)
        return copy.deepcopy(pinned)
    except (ValueError, TypeError, KeyError, AttributeError):
        raise Rejected('semantic_profile') from None


def decimal_value(value):
    if not isinstance(value, str) or not NUMBER.fullmatch(value) or len(value) > 128:
        raise Rejected('semantic_decimal_syntax')
    exponent = re.search(r'[eE](-?\d+)$', value)
    if exponent and (len(exponent[1]) > 4 or abs(int(exponent[1])) > 128):
        raise Rejected('semantic_decimal_scale')
    return number(value)


def disabled(field, caps):
    return field['name'] in CONSTRUCTS and not caps.get(CONSTRUCTS[field['name']], True) or field['name'] in {'consumed_quantity', 'consumed_unit'} and not caps['measured_usage'] or field['name'] == 'list_unit_price' and not caps['list_prices'] or field['name'] == 'contracted_unit_price' and not caps['contracted_prices']


def source_context(row, pinned):
    body = pinned['definition']; caps = body['capabilities']
    try:
        source = json.loads(row['source_json'])
    except (ValueError, KeyError, TypeError):
        raise Rejected('semantic_source') from None
    if not isinstance(source, dict) or len(source) > 256:
        raise Rejected('semantic_source')
    parsed = {}; reasons = {}
    for field in body['fields']:
        name = field['name']; column = field['source']
        if disabled(field, caps):
            if column in source and source[column] not in [None, *body['null_tokens']]:
                raise Rejected('semantic_disabled_field')
            parsed[name] = None; reasons[name] = 'capability_disabled'
            continue
        if column is not None and field['header'] and column not in source:
            raise Rejected('semantic_missing_header')
        value = source.get(column)
        if value is not None and not isinstance(value, str):
            raise Rejected('semantic_source_type')
        if column is None:
            reasons[name] = 'absent_by_version'
        elif column not in source:
            reasons[name] = 'not_provided'
        elif value is None or value in body['null_tokens']:
            reasons[name] = 'source_null'
        if value is None or value in body['null_tokens']:
            value = None
        elif len(value) > 65536:
            raise Rejected('semantic_field_size')
        elif field['type'] == 'decimal':
            value = decimal_value(value)
        elif field['type'] == 'timestamp':
            stamp = STAMP_LEGACY if body['validator_revision'] == 1 else STAMP
            if not stamp.fullmatch(value):
                raise Rejected('semantic_timestamp_format')
            value = instant(value)
        elif field['type'] == 'json':
            value = key_value(value)
        elif len(value) > 4096:
            raise Rejected('semantic_field_size')
        if value is not None and 'enum' in field and value not in field['enum']:
            raise Rejected('semantic_enum_value')
        parsed[name] = value
    for field in body['fields']:
        name = field['name']; value = parsed[name]
        if disabled(field, caps):
            continue
        if name == 'host_provider_name' and parsed['service_name'] in caps['host_null_services']:
            continue
        if value is not None and field.get('nonnegative') and value < 0:
            raise Rejected('semantic_negative_unit_price')
        if 'nonnull_when' in field and predicate(field['nonnull_when'], parsed) and value is None:
            raise Rejected('semantic_required_value')
        if 'null_when' in field and predicate(field['null_when'], parsed):
            if value is not None:
                raise Rejected('semantic_inapplicable_value')
            reasons[name] = 'not_applicable'
    for check in body['checks']:
        left, right, target = (parsed[n] for n in check['product'])
        if left is not None and right is not None and predicate(check['when'], parsed):
            if target is None or coefficient(left) * coefficient(right) != coefficient(target) * 10**18:
                raise Rejected('semantic_price_product')
    if not re.fullmatch(r'[A-Z]{3}', parsed['billing_currency'] or ''):
        raise Rejected('semantic_currency')
    if parsed['billing_period_start'] >= parsed['billing_period_end'] or parsed['charge_period_start'] >= parsed['charge_period_end']:
        raise Rejected('semantic_timestamp_interval')
    fields = {k: fixed(coefficient(v)) if isinstance(v, Decimal) else v.isoformat() if isinstance(v, datetime) else v for k, v in parsed.items()}
    return {'classification': 'consumed-fields-validated', 'profile_digest': pinned['digest'], 'fields': fields, 'missing_reasons': reasons}


def context(row, version, pinned=None):
    return source_context(row, checked_profile(pinned, version) if pinned is not None else profile(version))


def validate_rows(rows, version, pinned=None):
    selected = checked_profile(pinned, version) if pinned is not None else profile(version)
    if not isinstance(rows, list) or len(rows) > MAX_ROWS:
        raise Rejected('semantic_row_limit')
    return {'profile': selected, 'contexts': [source_context(row, selected) for row in rows]}
