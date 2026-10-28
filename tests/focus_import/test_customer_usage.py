import json
from datetime import datetime, timezone
from decimal import Decimal
from types import SimpleNamespace
from uuid import UUID

import pytest

from modules.focus_import import customer_usage as usage
from modules.focus_import.model import Rejected


TENANT = '00000000-0000-0000-0000-000000000001'
INSTALLATION = '00000000-0000-0000-0000-000000000002'
BATCH = '00000000-0000-0000-0000-000000000003'


def profile():
    return {'kind': 'source-usage-v1', 'installation': INSTALLATION, 'source': 'synthetic',
        'actor': 'reader-synthetic', 'company_id': 'company-a', 'tenant_id': TENANT, 'snapshot': 'a' * 64,
        'start': '2026-08-01T00:00:00Z', 'end': '2026-09-01T00:00:00Z', 'grain': 'month',
        'services': {'PRIVATE-SOURCE-SERVICE': 'Compute'}, 'units': {'PRIVATE-SOURCE-UNIT': 'hours'}}


def row(quantity=Decimal('12345678901234567890.000000000000000001')):
    return {'tenant_id': UUID(TENANT), 'ordinal': 1, 'service': 'PRIVATE-SOURCE-SERVICE',
        'consumed_quantity': quantity, 'consumed_unit': 'PRIVATE-SOURCE-UNIT',
        'charge_start': datetime(2026, 8, 1, tzinfo=timezone.utc),
        'charge_end': datetime(2026, 8, 2, tzinfo=timezone.utc),
        'billed_cost': Decimal('999.99'), 'account': 'PRIVATE-PAYER', 'source_json': 'PRIVATE-RAW'}


def source():
    journal = SimpleNamespace(head={'installation': INSTALLATION})
    authority = SimpleNamespace(is_view=False, tenant=UUID(TENANT), actor='reader-synthetic', bindings={'source': 'synthetic'})
    context = {'properties': {'company_id': 'company-a'}, 'active': True}
    slices = [('window', {}, {'state': 'hot', 'tenant_id': TENANT})]
    return journal, SimpleNamespace(a=authority), context, slices


def result(rows):
    return {'snapshot': 'a' * 64, 'classification': 'exploratory-query-only',
        'rows': [{'batch_id': BATCH, 'ordinal': r['ordinal']} for r in rows],
        'groups': {'PRIVATE-TOTAL': 99999}, 'account_hierarchies': {'PRIVATE-PAYER': 1}}


def test_projection_removes_hidden_operands_and_preserves_exact_values():
    consumer = usage.Projection(json.dumps(profile()), lambda value: value)
    consumer.preflight(*source())
    rows = [row()]
    value = consumer.consume(result(rows), [('window', BATCH, 1, 'tenant', rows[0])])
    assert set(value) == {'status', 'revision', 'rows'}
    assert set(value['rows'][0]['Values']) == set(usage.FIELDS)
    assert value['rows'][0]['Values'] == {'service': 'Compute', 'quantity': '12345678901234567890.000000000000000001',
        'unit': 'hours', 'usage_start': '2026-08-01T00:00:00+00:00', 'usage_end': '2026-08-02T00:00:00+00:00', 'measurement_state': 'source-reported'}
    serialized = json.dumps(value)
    for marker in ['PRIVATE', '999.99', 'account', 'source_json', BATCH, 'window']:
        assert marker not in serialized
    assert len(value['rows'][0]['ID']) == 64


@pytest.mark.parametrize('quantity,expected,state', [(None, '', 'missing'), (Decimal('0'), '0.000000000000000000', 'source-reported')])
def test_missing_and_zero_remain_distinct(quantity, expected, state):
    consumer = usage.Projection(json.dumps(profile()), lambda value: value)
    value = consumer.row(BATCH, 1, row(quantity))['Values']
    assert value['quantity'] == expected and value['measurement_state'] == state
    missing_unit = row(quantity)
    missing_unit['consumed_unit'] = None
    assert consumer.row(BATCH, 1, missing_unit)['Values']['unit'] == ''


@pytest.mark.parametrize('change', [
    lambda r: r.update(tenant_id=UUID(INSTALLATION)),
    lambda r: r.update(service='unmapped'),
    lambda r: r.update(consumed_unit='unmapped'),
    lambda r: r.update(charge_start=datetime(2026, 7, 31, tzinfo=timezone.utc)),
    lambda r: r.update(charge_end=datetime(2026, 9, 2, tzinfo=timezone.utc)),
])
def test_invalid_row_outside_page_denies_entire_result(change):
    consumer = usage.Projection(json.dumps(profile()), lambda value: pytest.fail('invalid projection released'))
    first, second = row(), row()
    second['ordinal'] = 2
    change(second)
    with pytest.raises(Rejected):
        consumer.consume(result([first]), [('window', BATCH, 1, 'tenant', first), ('window', BATCH, 2, 'tenant', second)])


@pytest.mark.parametrize('case', ['installation', 'tenant', 'company', 'source', 'actor', 'audience', 'inactive', 'archived', 'deleted'])
def test_source_preflight_checks_bindings_and_hot_state(case):
    consumer = usage.Projection(json.dumps(profile()), lambda value: value)
    journal, coordinator, context, slices = source()
    if case == 'installation': journal.head['installation'] = TENANT
    elif case == 'tenant': coordinator.a.tenant = UUID(INSTALLATION)
    elif case == 'company': context['properties']['company_id'] = 'company-b'
    elif case == 'source': coordinator.a.bindings['source'] = 'foreign'
    elif case == 'actor': coordinator.a.actor = 'foreign'
    elif case == 'audience': coordinator.a.is_view = True
    elif case == 'inactive': context['active'] = False
    else: slices[0][2]['state'] = case
    with pytest.raises(Rejected): consumer.preflight(journal, coordinator, context, slices)


def test_snapshot_and_duplicate_rows_fail_closed():
    consumer = usage.Projection(json.dumps(profile()), lambda value: pytest.fail('invalid released'))
    data = row()
    body = result([data])
    body['snapshot'] = 'b' * 64
    with pytest.raises(Rejected): consumer.consume(body, [('window', BATCH, 1, 'tenant', data)])
    with pytest.raises(Rejected): consumer.consume(result([data]), [('window', BATCH, 1, 'tenant', data)] * 2)


def test_profile_allowlist_and_query_pinning(monkeypatch):
    p = profile()
    p['billed_cost'] = True
    with pytest.raises(Rejected): usage.checked_profile(json.dumps(p))
    p = profile()
    def query(actor, start, end, grain, limit, **kwargs):
        assert actor == 'reader-synthetic' and start == p['start'] and end == p['end'] and grain == 'month' and limit == 2
        assert kwargs['expected_snapshot'] == 'a' * 64
        assert isinstance(kwargs['_consumer'], usage.Projection)
        return 'observed'
    monkeypatch.setattr(usage.collection, 'query', query)
    assert usage.query(json.dumps(p), 2, lambda value: value) == 'observed'


def test_export_includes_full_selection_beyond_page_and_never_truncates():
    rows = []
    for ordinal in range(1, 102):
        value = row(Decimal('2'))
        value['ordinal'] = ordinal
        rows.append(value)
    all_rows = [('window', BATCH, r['ordinal'], 'tenant', r) for r in rows]
    page = result(rows[:1])
    page['row_count'] = len(all_rows)
    consumer = usage.Projection(json.dumps(profile()), lambda value: value, 'export', 101)
    exported = consumer.consume(page, all_rows)
    assert exported['selection'] == 'complete-supported-selection' and exported['row_count'] == 101
    assert len(exported['rows']) == 101 and len({r['ID'] for r in exported['rows']}) == 101
    bounded = usage.Projection(json.dumps(profile()), lambda value: pytest.fail('truncated export released'), 'export', 100)
    with pytest.raises(Rejected): bounded.consume(page, all_rows)
    page['row_count'] = 102
    with pytest.raises(Rejected): consumer.consume(page, all_rows)


def test_export_mode_pins_full_row_limit_but_keeps_source_page_bounded(monkeypatch):
    def query(actor, start, end, grain, limit, **kwargs):
        assert limit == 100 and kwargs['_consumer'].mode == 'export' and kwargs['_consumer'].limit == 500
        return 'full-request'
    monkeypatch.setattr(usage.collection, 'query', query)
    assert usage.query(json.dumps(profile()), 500, lambda value: value, 'export') == 'full-request'
    for mode, limit in [('other', 10), ('export', 10001), ('page', 101)]:
        with pytest.raises(Rejected): usage.query(json.dumps(profile()), limit, lambda value: value, mode)
