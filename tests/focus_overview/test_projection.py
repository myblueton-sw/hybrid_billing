from datetime import datetime, timezone
from decimal import Decimal
from types import SimpleNamespace

import pytest

from modules.focus_import.model import Rejected
from modules.focus_import.overview import Projection, account_id, overview


def row(account='same', subaccount=None, provider='AWS', value='1.000000000000000001', start='2024-04-01T00:30:00+00:00', currency='USD', evidence='official-public'):
    return {'provider': provider, 'account': account, 'subaccount': subaccount, 'service': 'Compute', 'currency': currency,
            'billed_cost': Decimal(value) if value is not None else None, 'effective_cost': Decimal('-0.5'),
            'charge_start': datetime.fromisoformat(start), 'charge_end': datetime(2024, 5, 1, tzinfo=timezone.utc),
            'consumed_quantity': Decimal('0.000000000000000001'), 'pricing_quantity': None, 'list_cost': Decimal('0'), 'contracted_cost': None,
            'list_unit_price': Decimal('-0.000000000000000001'), 'contracted_unit_price': None,
            'consumed_unit': 'Hours', 'pricing_unit': 'Requests', 'billing_start': datetime(2024, 1, 1, tzinfo=timezone.utc), 'billing_end': datetime(2024, 2, 1, tzinfo=timezone.utc),
            'source_json': {'x_HB_EvidenceClass': evidence, 'secret_extra': 'never-project'}}


def project(params=None, rows=None, snapshot='a' * 64, deliver=None):
    p = Projection(params or {}, deliver)
    tenants = [{'properties': {'key': name, 'timezone': zone}} for name, zone in [('north-a', 'America/Los_Angeles'), ('north-b', 'UTC')]]
    p.preflight(None, SimpleNamespace(a=SimpleNamespace(is_view=True)), {'audience': {'company_id': 'north'}, 'tenants': tenants}, [('window', {'version': '1.0', 'batch_id': 'batch'}, {})])
    result = {'classification': 'exploratory-query-only', 'period_attribution': 'charge-start-in-tenant-timezone-no-proration', 'snapshot': snapshot, 'coverage': []}
    return p.consume(result, rows if rows is not None else [('window', 'batch', 0, 'north-a', row())])


def test_exact_signed_missing_and_separate_currency():
    rows = [('w', 'b', i, 'north-a', r) for i, r in enumerate([row(), row(value='-1'), row(value=None), row(value='0', currency='CNY')])]
    d = project(rows=rows)
    usd = next(a for a in d['amounts'] if a['currency'] == 'USD')
    assert usd['billed_cost'] == {'coefficient': '1', 'present': 2, 'missing': 1}
    assert usd['effective_cost']['coefficient'] == '-1500000000000000000'
    assert d['amounts'][0]['billed_cost']['coefficient'] == '0'
    assert d['row_count'] == 4
    assert 'secret_extra' not in str(d) and 'source_json' not in str(d)
    assert project(rows=[('w', 'b', 0, 'north-a', row(value=None))])['amounts'][0]['billed_cost']['coefficient'] is None


def test_timezone_and_no_delivery_window_proration():
    rows = [('w', 'b', i, tenant, row()) for i, tenant in enumerate(['north-a', 'north-b'])]
    d = project({'period': '2024-03'}, rows)
    assert d['row_count'] == 1
    assert d['rows'][0]['charge_month'] == '2024-03'
    assert [m['period'] for m in d['months']] == ['2024-03', '2024-04']
    assert d['amounts'][0]['billed_cost']['coefficient'] == '1000000000000000001'


def test_account_identity_and_foreign_scope():
    base = row()
    assert len({account_id(t, r) for t, r in [('north-a', base), ('north-b', base), ('north-a', row(subaccount='x')), ('north-a', row(account='other', subaccount='x')), ('north-a', row(provider='Oracle'))]}) == 5
    d = project()
    assert d['accounts'][0]['sub_account'] is None
    assert d['accounts'][0]['native_ancestry'] == 'unverified'
    with pytest.raises(Rejected, match='scope_denied'):
        project(rows=[('w', 'b', 0, 'foreign', base)])
    with pytest.raises(Rejected, match='selection_unavailable'):
        project({'account': 'f' * 64})
    with pytest.raises(Rejected, match='selection_unavailable'):
        project({'period': '2026-10'})


def test_snapshot_filters_pages_and_search_do_not_change_totals():
    rows = [('w', 'b', i, 'north-a', row(account=f'acct-{i:03}')) for i in range(51)]
    first = project(rows=rows)
    assert len(first['accounts']) == len(first['rows']) == 25
    second = project({'after': '25', 'account_after': '25', 'snapshot': first['snapshot']}, rows)
    assert set(a['id'] for a in first['rows']).isdisjoint(a['id'] for a in second['rows'])
    assert second['amounts'] == first['amounts']
    found = project({'q': 'acct-050'}, rows)
    assert found['account_matches'] == 1 and found['row_count'] == 51
    for changes in [{'basis': 'effective_cost'}, {'q': 'acct'}, {'period': '2024-03'}]:
        with pytest.raises(Rejected, match='query_snapshot_changed'):
            project({**changes, 'snapshot': first['snapshot']}, rows)
    with pytest.raises(Rejected, match='query_snapshot_changed'):
        project({'snapshot': first['snapshot']}, rows, snapshot='b' * 64)
    with pytest.raises(Rejected, match='snapshot_required'):
        project({'after': '25'}, rows)
    with pytest.raises(Rejected, match='request_invalid'):
        project({'actor': 'foreign'}, rows)


def test_publication_remains_in_collection_consumer(monkeypatch):
    active = []
    def collected(actor, limit, _consumer):
        active.append(actor)
        result = project(deliver=_consumer.deliver)
        active.clear()
        return result
    monkeypatch.setattr('modules.focus_import.overview.collected_query', collected)
    def deliver(body):
        assert active == ['reader-company-north']
        assert body['financial_authority'] is False
        return 'published'
    assert overview('reader-company-north', {}, deliver) == 'published'


def test_extended_row_contract_is_exact_and_unit_specific():
    d = project()
    r = d['rows'][0]
    assert r['consumed_quantity'] == '1' and r['consumed_unit'] == 'Hours'
    assert r['pricing_quantity'] is None and r['pricing_unit'] == 'Requests'
    assert r['list_unit_price'] == '-1' and r['list_cost'] == '0'
    assert r['contracted_cost'] is None
    assert r['billing_start'] == '2024-01-01T00:00:00+00:00'
    assert 'source_json' not in r


def test_extended_projection_rejects_old_content_identity():
    from modules.focus_import.model import canonical, digest
    old = digest(canonical({'projection': 1, 'source': 'a' * 64, 'selection': {'period':'all','provider':'','account':'','basis':'billed_cost','q':''}}).encode())
    with pytest.raises(Rejected, match='query_snapshot_changed'):
        project({'snapshot': old})


def test_quantity_with_missing_unit_does_not_invent_one():
    r = row()
    r['consumed_unit'] = None
    d = project(rows=[('w','b',0,'north-a',r)])
    assert d['rows'][0]['consumed_quantity'] == '1'
    assert d['rows'][0]['consumed_unit'] is None
    assert 'consumed_quantity' not in d['amounts'][0]
