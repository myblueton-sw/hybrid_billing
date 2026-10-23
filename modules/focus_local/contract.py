import csv
import hashlib
import io
import json
import re
from dataclasses import dataclass
from datetime import datetime, timezone
from decimal import Decimal
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
REGISTRY = ROOT / 'packages/contracts/focus'
MAX_ROWS = 10_000
MAX_BYTES = 8 * 1024 * 1024
SCALE = 6
MEASURES = ('billed_cost', 'effective_cost', 'list_cost', 'contracted_cost')
IDENTIFIER = re.compile(r'[a-z][a-z0-9_]{0,62}\Z')
NUMBER = re.compile(r'(-?)([0-9]+)(?:\.([0-9]+))?(?:[eE]([+-]?[0-9]+))?\Z')


class Rejected(Exception):
    pass


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=True)


def digest(value):
    return hashlib.sha256(value).hexdigest()


def predicate(rule, row):
    op, value = next(iter(rule.items()))
    if op == 'always':
        return value
    if op == 'all':
        return all(predicate(x, row) for x in value)
    if op == 'any':
        return any(predicate(x, row) for x in value)
    if op == 'not':
        return not predicate(value, row)
    if op == 'is_null':
        return row[value] is None
    if op == 'eq':
        return row[value[0]] == value[1]
    if op == 'in':
        return row[value[0]] in value[1]
    raise Rejected('registry_predicate')


def validate_predicate(rule, names, depth=0):
    if depth > 8 or not isinstance(rule, dict) or len(rule) != 1:
        raise Rejected('registry_predicate')
    op, value = next(iter(rule.items()))
    if op == 'always' and type(value) is bool:
        return
    if op in ('all', 'any') and isinstance(value, list) and 0 < len(value) <= 16:
        for item in value:
            validate_predicate(item, names, depth + 1)
        return
    if op == 'not':
        validate_predicate(value, names, depth + 1)
        return
    if op == 'is_null' and isinstance(value, str) and value in names:
        return
    if op in ('eq', 'in') and isinstance(value, list) and len(value) == 2:
        if isinstance(value[0], str) and value[0] in names and (
            (op == 'eq' and isinstance(value[1], str)) or
            (op == 'in' and isinstance(value[1], list) and all(isinstance(x, str) for x in value[1]))
        ):
            return
    raise Rejected('registry_predicate')


@dataclass(frozen=True)
class Contract:
    entry: dict
    sha256: str
    table: str

    @property
    def fields(self):
        return self.entry['fields']


def load_registry(directory=REGISTRY):
    try:
        return _load_registry(directory)
    except (ValueError, KeyError, TypeError, AttributeError):
        raise Rejected('registry_format') from None


def _load_registry(directory):
    registry = {}
    for path in sorted(Path(directory).glob('*.json')):
        if path.stat().st_size > 128_000:
            raise Rejected('registry_size')
        entry = json.loads(path.read_text())
        if not isinstance(entry, dict) or set(entry) != {'version', 'revision', 'profile', 'sources', 'fields', 'checks'}:
            raise Rejected('registry_shape')
        version = entry['version']
        if not isinstance(version, str) or not re.fullmatch(r'[0-9]+\.[0-9]+', version) or version in registry:
            raise Rejected('registry_version')
        if type(entry['revision']) is not int or entry['revision'] < 1 or entry['profile'] != 'synthetic-local-v1':
            raise Rejected('registry_profile')
        if not isinstance(entry['sources'], list) or not entry['sources']:
            raise Rejected('registry_sources')
        fields = entry['fields']
        if not isinstance(fields, list) or not 1 <= len(fields) <= 100:
            raise Rejected('registry_fields')
        names = set()
        for field in fields:
            if not isinstance(field, dict) or set(field) - {'source', 'name', 'type', 'header', 'enum', 'nonnull_when', 'null_when', 'aggregate', 'nonnegative'}:
                raise Rejected('registry_field')
            if not {'source', 'name', 'type', 'header'} <= set(field):
                raise Rejected('registry_field')
            name = field['name']
            if not isinstance(name, str) or not IDENTIFIER.fullmatch(name) or name in names or name in ('installation_id', 'workspace_id', 'release_id', 'attempt', 'ordinal'):
                raise Rejected('registry_name')
            names.add(name)
            if field['type'] not in ('string', 'decimal', 'timestamp', 'json') or type(field['header']) is not bool:
                raise Rejected('registry_type')
            source = field['source']
            if source is not None and (not isinstance(source, str) or not re.fullmatch(r'[A-Za-z][A-Za-z0-9_]{0,63}', source)):
                raise Rejected('registry_source')
            if source is None and field['header']:
                raise Rejected('registry_source')
            if 'enum' in field and (not isinstance(field['enum'], list) or not all(isinstance(x, str) for x in field['enum'])):
                raise Rejected('registry_enum')
            if field['type'] == 'decimal' and field.get('aggregate') not in ('sum', 'none'):
                raise Rejected('registry_aggregation')
            if 'nonnegative' in field and type(field['nonnegative']) is not bool:
                raise Rejected('registry_nonnegative')
        required = {x: 'decimal' for x in MEASURES}
        required.update({x: 'string' for x in ['billing_currency','billing_account_id','sub_account_id','consumed_unit','pricing_unit']})
        required.update({x: 'timestamp' for x in ['billing_period_start','billing_period_end','charge_period_start','charge_period_end']})
        kinds = {f['name']: f['type'] for f in fields}
        if any(kinds.get(name) != kind for name, kind in required.items()):
            raise Rejected('registry_core')
        if any(f.get('aggregate') != 'sum' for f in fields if f['name'] in MEASURES):
            raise Rejected('registry_aggregation')
        for field in fields:
            for key in ('nonnull_when', 'null_when'):
                if key in field:
                    validate_predicate(field[key], names)
        if not isinstance(entry['checks'], list) or len(entry['checks']) > 32:
            raise Rejected('registry_checks')
        for check in entry['checks']:
            if not isinstance(check, dict) or set(check) != {'product','when'} or not isinstance(check['product'], list) or len(check['product']) != 3:
                raise Rejected('registry_checks')
            if any(not isinstance(x, str) or kinds.get(x) != 'decimal' for x in check['product']):
                raise Rejected('registry_checks')
            validate_predicate(check['when'], names)
        shape = sorted((f['name'], f['type']) for f in fields)
        registry[version] = Contract(entry, digest(canonical(entry).encode()), 'facts_' + digest(canonical(shape).encode())[:20])
    if not registry:
        raise Rejected('registry_empty')
    return registry


def decimal_value(text):
    if not isinstance(text, str) or len(text) > 128:
        raise Rejected('decimal_syntax')
    match = NUMBER.fullmatch(text)
    if not match:
        raise Rejected('decimal_syntax')
    sign, integer, fraction, exponent = match.groups()
    exponent = int(exponent or 0)
    fraction = fraction or ''
    scale = len(fraction) - exponent
    if abs(exponent) > 12 or scale > SCALE:
        raise Rejected('decimal_scale')
    digits = (integer + fraction).lstrip('0') or '0'
    places = SCALE - scale
    if len(digits) + places > 18:
        raise Rejected('decimal_magnitude')
    coefficient = int(digits) * 10 ** places * (-1 if sign else 1)
    return Decimal(fixed(coefficient))


def coefficient(value):
    sign, digits, exponent = value.as_tuple()
    result = int(''.join(map(str, digits)) or '0') * 10 ** (exponent + SCALE)
    return -result if sign else result


def fixed(value):
    sign = '-' if value < 0 else ''
    whole, fraction = divmod(abs(value), 10 ** SCALE)
    return f'{sign}{whole}.{fraction:06d}'


def parse(data, contract, capability):
    if len(data) > MAX_BYTES:
        raise Rejected('input_size')
    try:
        reader = csv.DictReader(io.StringIO(data.decode('utf-8'), newline=''), strict=True)
        headers = reader.fieldnames or []
        if len(headers) != len(set(headers)):
            raise Rejected('duplicate_header')
        if any(f['source'] not in headers for f in contract.fields if f['header']):
            raise Rejected('missing_header')
        rows = []
        for source in reader:
            if len(rows) >= MAX_ROWS or None in source or any(v is None for v in source.values()):
                raise Rejected('row_shape_or_limit')
            sources = {f['name']: f['source'] for f in contract.fields}
            if source.get(sources['billing_account_id']) != capability['account'] or source.get(sources['sub_account_id']) != capability['subaccount']:
                raise Rejected('source_scope')
            row = {}
            for field in contract.fields:
                text = source.get(field['source']) if field['source'] else None
                value = None if text in (None, '', '\\N') else text
                if value is not None:
                    if len(value) > 4096:
                        raise Rejected('field_size')
                    if field['type'] == 'decimal':
                        value = decimal_value(value)
                    elif field['type'] == 'timestamp':
                        if not re.fullmatch(r'\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,6})?(?:Z|[+-]\d\d:\d\d)', value):
                            raise Rejected('timestamp_format')
                        value = datetime.fromisoformat(value.replace('Z', '+00:00'))
                    elif field['type'] == 'json':
                        value = canonical(json.loads(value, parse_constant=lambda _: (_ for _ in ()).throw(Rejected('json_value'))))
                    if 'enum' in field and value not in field['enum']:
                        raise Rejected('enum_value')
                row[field['name']] = value
            for field in contract.fields:
                value = row[field['name']]
                if value is not None and field.get('nonnegative') and value < 0:
                    raise Rejected('negative_unit_price')
                if 'nonnull_when' in field and predicate(field['nonnull_when'], row) and value is None:
                    raise Rejected('required_value')
                if 'null_when' in field and predicate(field['null_when'], row) and value is not None:
                    raise Rejected('inapplicable_value')
            for check in contract.entry['checks']:
                left, right, target = (row[n] for n in check['product'])
                if left is not None and right is not None and predicate(check['when'], row):
                    if target is None or coefficient(left) * coefficient(right) != coefficient(target) * 10**SCALE:
                        raise Rejected('price_product')
            if row['billing_account_id'] != capability['account'] or row['sub_account_id'] != capability['subaccount']:
                raise Rejected('source_scope')
            start, end = capability['start'], capability['end']
            if row['billing_period_start'] != start or row['billing_period_end'] != end:
                raise Rejected('source_period')
            if not start <= row['charge_period_start'] < row['charge_period_end'] <= end:
                raise Rejected('source_period')
            if not re.fullmatch('[A-Z]{3}', row['billing_currency'] or ''):
                raise Rejected('currency')
            rows.append(row)
        if not rows:
            raise Rejected('empty_input')
        return rows
    except (UnicodeError, csv.Error, ValueError, TypeError, OverflowError):
        raise Rejected('input_format') from None


def admit(data, capability):
    if len(data) > MAX_BYTES:
        raise Rejected('source_admission')
    try:
        reader = csv.DictReader(io.StringIO(data.decode('utf-8'),newline=''),strict=True)
        headers = reader.fieldnames or []
        mapping = capability['admission_fields']
        if len(headers) != len(set(headers)) or not set(mapping.values()) <= set(headers):
            raise Rejected('source_admission')
        count = 0
        for row in reader:
            count += 1
            if count > MAX_ROWS or None in row or any(v is None for v in row.values()):
                raise Rejected('source_admission')
            if row[mapping['account']] != capability['account'] or row[mapping['subaccount']] != capability['subaccount']:
                raise Rejected('source_scope')
            values = {}
            for field in ('billing_start','billing_end','charge_start','charge_end'):
                text = row[mapping[field]]
                if len(text)>40 or not re.fullmatch(r'\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,6})?(?:Z|[+-]\d\d:\d\d)',text):
                    raise Rejected('source_admission')
                values[field] = datetime.fromisoformat(text.replace('Z','+00:00'))
            if values['billing_start'] != capability['period_start'] or values['billing_end'] != capability['period_end']:
                raise Rejected('source_period')
            if not capability['period_start'] <= values['charge_start'] < values['charge_end'] <= capability['period_end']:
                raise Rejected('source_period')
        if count == 0:
            raise Rejected('source_admission')
    except (UnicodeError,csv.Error,ValueError,TypeError,KeyError):
        raise Rejected('source_admission') from None


def controls(rows, fields):
    result = {}
    for row in rows:
        key = canonical([row['billing_currency'], row.get('consumed_unit'), row.get('pricing_unit')])
        group = result.setdefault(key, {'rows': 0, 'measures': {f['name']: {'coefficient': 0, 'missing': 0, 'present': 0} for f in fields if f['type'] == 'decimal' and f['aggregate'] == 'sum'}})
        group['rows'] += 1
        for name, item in group['measures'].items():
            value = row[name]
            item['missing'] += value is None
            item['present'] += value is not None
            item['coefficient'] += coefficient(value) if value is not None else 0
    for group in result.values():
        for item in group['measures'].values():
            if item['present'] == 0:
                item['coefficient'] = None
    return result


def rows_digest(rows):
    normalized = []
    for row in rows:
        normalized.append({k: (v.astimezone(timezone.utc).isoformat() if isinstance(v, datetime) else fixed(coefficient(v)) if isinstance(v, Decimal) else v) for k, v in row.items()})
    return digest(canonical(normalized).encode())
