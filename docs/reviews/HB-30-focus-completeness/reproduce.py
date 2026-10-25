import hashlib
import json
import sys
from contextlib import contextmanager
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))
sys.path.insert(0, str(ROOT / 'tests/focus_import'))
sys.path.insert(0, str(ROOT / 'tests/local_focus'))

from test_contract import CAP, fixture
from test_temporal import configuration, descriptor, encoded, source_row
from modules.focus_import import collection, temporal
from modules.focus_import.model import decode, digest, number, versions
from modules.focus_local.contract import decimal_value, load_registry, parse
from tools import focus_lifecycle as cli


results = {'scope': 'pure synthetic audit; no database or runtime effects', 'cases': []}
registry = load_registry()
for version in ['1.2', '1.3']:
    imported = set(versions()[version]['columns'].values())
    strict = {f['source'] for f in registry[version].fields if f['source']}
    results['cases'].append({'case': 'field_inventory', 'version': version, 'mapped_count': len(imported),
        'separate_strict_count': len(strict), 'missing_from_common': sorted(strict - imported)})
results['synthetic_fixture_sha256'] = digest(fixture())
results['cases'].append({'case': 'positive_exponent_sign', 'input': '1E+2',
    'common_import': str(number('1E+2')), 'separate_strict_synthetic': str(decimal_value('1E+2'))})
for name, change in {
    'mandatory_cost_null': {'BilledCost': ''},
    'negative_unit_price': {'ListUnitPrice': '-1'},
    'price_product_mismatch': {'ListUnitPrice': '999'},
    'quantity_without_unit': {'ConsumedUnit': ''},
    'sku_missing': {'SkuPriceId': ''},
    'category_case': {'ChargeCategory': 'usage'},
    'host_missing': {'HostProviderName': ''},
    'null_subaccount': {'SubAccountId': ''},
    'naive_timestamp': {'ChargePeriodStart': '2026-10-01 00:00:00'},
    'bad_service_category': {'ServiceCategory': 'not-a-category'},
    'tags_array': {'Tags': '[]'},
    'duplicate_tag_keys': {'Tags': '{"k":"a","k":"b"}'},
    'tag_numeric_precision': {'Tags': '{"weight":0.1000000000000000001}'},
}.items():
    data = fixture(**change)
    observed = {'case': name, 'input_sha256': digest(data)}
    for label, action in [('common_import', lambda: decode(data, versions()['1.3'])),
                          ('separate_strict_synthetic', lambda: parse(data, registry['1.3'], CAP))]:
        try:
            parsed = action()
            observed[label] = {'result': 'accepted', 'rows': len(parsed)}
            if label == 'separate_strict_synthetic' and name in ['duplicate_tag_keys', 'tag_numeric_precision']:
                observed[label]['normalized_tags'] = parsed[0].get('tags')
        except Exception as exc:
            observed[label] = {'result': 'rejected', 'reason': str(exc)}
    results['cases'].append(observed)

partial = encoded([source_row(cost='7')])
rows = temporal.validate_delivery(partial, '1.0', configuration(), descriptor(partial, 'month', 2))
results['cases'].append({'case': 'finalized_partial_accounts', 'accepted_rows': len(rows),
    'configured_tenants': len(configuration()['tenants']), 'observed_tenants': len({str(r['tenant_id']) for r in rows})})


class File:
    suffix = '.csv'

    def __init__(self, name):
        self.name = name

    def is_file(self):
        return True

    def __lt__(self, other):
        return self.name < other.name


class Inbox:
    def iterdir(self):
        return iter([File(f'{n:03}.csv') for n in range(100)] + [File('zzz.csv')])


journal = SimpleNamespace(state={'deliveries': {'wanted': {'state': 'registered',
    'descriptor': {'file_digest': digest(b'wanted')}}}})


@contextmanager
def access(*args):
    yield journal, None, None


calls = []
with patch.object(collection, 'access', access), patch.object(cli, 'bounded',
        lambda p: b'wanted' if p.name == 'zzz.csv' else b'other'), patch.object(collection, 'collect',
        lambda *args: calls.append(args)):
    scans = [cli.run_due('importer', Inbox()) for _ in range(2)]
results['cases'].append({'case': 'scheduler_starvation', 'cycles': scans, 'collection_calls': len(calls)})

windows = [temporal.window(f'2024-09-0{i}T00:00:00Z', f'2024-09-0{i+1}T00:00:00Z', 'day') for i in (1, 2, 3)]
state = {'windows': {}, 'deliveries': {}, 'slices': {}}
for i, w in enumerate(windows):
    delivery = str(i)
    state['windows'][w['key']] = {'window': w, 'current': None if i == 1 else delivery}
    state['deliveries'][delivery] = {'state': 'registered' if i == 1 else 'sealed',
        'batch_id': delivery, 'descriptor': {'window': w}}
    state['slices'][delivery + ':tenant'] = {'state': 'hot'}
selected = collection.selected_slices(state, {'tenant': {}}, windows[0]['start'], windows[-1]['end'])
results['cases'].append({'case': 'pending_middle_window', 'known_windows': len(windows), 'selected_windows': len(selected)})
class Connection:
    def execute(self, *args):
        return self

    def fetchone(self):
        return {'body': configuration()}


state = {'grain': 'month', 'reject_before': '2000-01-01T00:00:00+00:00', 'windows': {}, 'deliveries': {}, 'slices': {}}
register_journal = SimpleNamespace(state=state, save=lambda body: None)


@contextmanager
def register_access(*args):
    yield register_journal, None, Connection()


aws = encoded([source_row(cost='7')])
azure = encoded([source_row(provider='Microsoft', cost='20')])
with patch.object(collection, 'access', register_access):
    first = collection.register('operator', aws, '1.0', descriptor(aws, 'month', 1))
    first_id = first['delivery_id']
    target = next(iter(state['windows'].values()))
    state['deliveries'][first_id].update(state='sealed', batch_id='aws-batch')
    target['current'] = first_id
    try:
        collection.register('operator', azure, '1.0', descriptor(azure, 'month', 1))
        same_revision = 'accepted'
    except Exception as exc:
        same_revision = str(exc)
    second = collection.register('operator', azure, '1.0', descriptor(azure, 'month', 2))
    second_id = second['delivery_id']
    state['deliveries'][second_id].update(state='sealed', batch_id='azure-batch')
    target['current'] = second_id
    state['slices'] = {'aws-batch:tenant': {'state': 'hot', 'provider': 'AWS'},
        'azure-batch:tenant': {'state': 'hot', 'provider': 'Microsoft'}}
    selected = collection.selected_slices(state, {'tenant': {}}, None, None)
    results['cases'].append({'case': 'independent_provider_same_period', 'same_revision_second_provider': same_revision,
        'higher_revision_second_provider': 'accepted', 'window_count': len(state['windows']),
        'selected_batches_after_modeled_publication': [d['batch_id'] for _, d, _ in selected],
        'selection_is_modeled_post_publication_not_engine_effect': True})
    regressed = descriptor(azure, 'month', 3)
    regressed.update(complete_through='2024-09-10T00:00:00Z', finalized=False)
    reopened = collection.register('operator', azure, '1.0', regressed)
    results['cases'].append({'case': 'finality_regression', 'result': 'registered',
        'previous_finalized': True, 'new_finalized': False, 'new_complete_through': regressed['complete_through'],
        'runtime_effect': 'none; registration used fake journal and connection'})


results['reproduction_sha256'] = hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
print(json.dumps(results, sort_keys=True, indent=2))
