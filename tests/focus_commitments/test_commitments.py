import http.client
import json
import threading
from datetime import datetime, timezone
from decimal import Decimal
from types import SimpleNamespace

import pytest

from modules.focus_import.commitment_fields import Mapping
from modules.focus_import.commitments import Projection, amounts, commitments, costs, selection
from modules.focus_import.model import Rejected, canonical, digest, versions
from modules.focus_import.overview import account_id
from modules.focus_import.overview_http import OverviewServer


def row(source_id='sp-1', account='acct', subaccount=None, currency='USD', provider='AWS', billed='100', effective='80', status='Used', extra=None):
    raw = {'CommitmentDiscountId': source_id, 'CommitmentDiscountType': 'Savings Plan',
           'CommitmentDiscountCategory': 'Spend', 'CommitmentDiscountStatus': status,
           'ChargeCategory': 'Usage', 'ChargeClass': 'Regular', 'ChargeFrequency': 'Usage-Based',
           'private_secret': 'never-project'}
    raw.update(extra or {})
    return {'provider': provider, 'account': account, 'subaccount': subaccount, 'service': 'Compute',
            'currency': currency, 'billed_cost': Decimal(billed) if billed is not None else None,
            'effective_cost': Decimal(effective) if effective is not None else None,
            **{name: datetime(2024, 9 if name.endswith('start') else 10, 1, tzinfo=timezone.utc)
               for name in ('charge_start', 'charge_end', 'billing_start', 'billing_end')},
            'source_json': raw}


def project(params=None, rows=None, declared=None, snapshot='a' * 64, deliver=None):
    rows = rows if rows is not None else [('w', 'batch', 1, 'north-a', row())]
    p = Projection(params or {}, deliver)
    declared = declared if declared is not None else {(r[3], r[1]): '1.0' for r in rows}
    names = ('north-a', 'north-b')
    tenants = [{'tenant_id': name + '-id', 'properties': {'key': name, 'timezone': 'UTC'}} for name in names]
    scopes = [('w', {'version': version, 'batch_id': batch, 'mapping_digest': digest(canonical(versions()[version]).encode())},
               {'batch_id': batch, 'tenant_id': tenant + '-id'}) for (tenant, batch), version in declared.items()]
    p.preflight(None, SimpleNamespace(a=SimpleNamespace(is_view=True)), {'audience': {'company_id': 'company-north'}, 'tenants': tenants}, scopes)
    result = {'classification': 'exploratory-query-only', 'period_attribution': 'charge-start-in-tenant-timezone-no-proration', 'snapshot': snapshot, 'coverage': []}
    return p.consume(result, rows)


def test_complete_groups_and_distinct_identity_scopes():
    rows = [('w', 'batch', i, tenant, value) for i, (tenant, value) in enumerate([
        ('north-a', row()), ('north-b', row()), ('north-a', row(account='other')),
        ('north-a', row(subaccount='sub')), ('north-a', row(currency='CNY')), ('north-a', row(provider='VMware')),
        ('north-a', row(source_id='sp-2')), ('north-a', row())])]
    d = project(rows=rows)
    assert d['group_count'] == 7 and d['identified_rows'] == 8
    assert len({g['id'] for g in d['groups']}) == 7
    combined = next(g for g in d['groups'] if g['rows'] == 2)
    assert combined['billed_cost']['coefficient'] == str(200 * 10 ** 18)
    assert 'savings' not in d and all('savings' not in g for g in d['groups'])
    assert 'private_secret' not in str(d) and 'never-project' not in str(d) and 'source_json' not in str(d)
    assert d['financial_authority'] is False


@pytest.mark.parametrize('value,state', [(None, 'source_null'), ('NULL', 'source_null'), ('null', 'source_null'), (r'\N', 'source_null'), ('', 'source_null'), ('  ', 'blank'), (3, 'invalid_type'), ({'x': 1}, 'invalid_type')])
def test_pinned_null_tokens_are_not_identifiers(value, state):
    d = project(rows=[('w', 'batch', 0, 'north-a', row(source_id=value))])
    assert d['group_count'] == d['identified_rows'] == 0
    assert d['unidentified_rows'] == d['unidentified_reasons'][state] == 1
    assert sum(d['unidentified_reasons'].values()) == 1
    assert d['unidentified_amounts'][0]['billed_cost']['coefficient'] == str(100 * 10 ** 18)


def test_absent_fields_unmapped_version_and_lexical_visibility():
    r = row()
    del r['source_json']['CommitmentDiscountId']
    assert project(rows=[('w', 'batch', 0, 'north-a', r)])['unidentified_reasons']['absent'] == 1
    mapping = Mapping()
    fields = mapping.fields(row(), '1.2')
    assert fields['id'] == {'state': 'unmapped_version', 'value': 'sp-1'}
    d = project(declared={('north-a', 'batch'): '1.2'})
    assert d['group_count'] == 0 and d['unidentified_reasons']['unmapped_version'] == 1
    assert d['versions'] == ['1.2']
    fields = mapping.fields(row(extra={'CommitmentDiscountName': None, 'CommitmentDiscountStatus': False}), '1.0')
    assert fields['name']['state'] == 'source_null' and fields['status']['state'] == 'invalid_type'
    assert mapping.fields(row(), '1.0')['name']['state'] == 'absent'


def test_status_partition_signed_missing_cancellation_and_no_savings():
    rows = [('w', 'batch', i, 'north-a', r) for i, r in enumerate([
        row(billed='100', effective='80', status='Used'),
        row(billed='-100', effective=None, status='Unused'),
        row(billed=None, effective='-80', status='used'),
        row(billed=None, effective=None, status='NULL')])]
    d = project(rows=rows)
    g = d['groups'][0]
    assert g['billed_cost'] == {'coefficient': '0', 'present': 2, 'missing': 2}
    assert g['effective_cost'] == {'coefficient': '0', 'present': 2, 'missing': 2}
    assert {k: v['rows'] for k, v in g['statuses'].items()} == {'used': 1, 'unused': 1, 'unknown': 2}
    for basis in ('billed_cost', 'effective_cost'):
        assert sum(v[basis]['present'] for v in g['statuses'].values()) == g[basis]['present']
        assert sum(int(v[basis]['coefficient'] or '0') for v in g['statuses'].values()) == 0
    assert g['statuses']['unknown']['billed_cost']['coefficient'] is None
    assert g['metadata']['status']['distinct'] == 3
    one = project()['groups'][0]
    assert one['billed_cost']['coefficient'] == str(100 * 10 ** 18)
    assert one['effective_cost']['coefficient'] == str(80 * 10 ** 18)
    assert one['statuses']['unused']['rows'] == 0
    assert one['statuses']['unused']['effective_cost']['coefficient'] is None


def test_metadata_distinct_count_bound_without_first_value_winner():
    rows = [('w', 'batch', i, 'north-a', row(extra={'CommitmentDiscountName': f'name-{i:02}'})) for i in range(20)]
    metadata = project(rows=rows)['groups'][0]['metadata']['name']
    assert metadata == {'values': [f'name-{i:02}' for i in range(16)], 'distinct': 20, 'truncated': True}
    with pytest.raises(Rejected, match='field_bound'):
        project(rows=[('w', 'batch', 0, 'north-a', row(source_id='x' * 4097))])
    with pytest.raises(Rejected, match='field_bound'):
        Mapping().fields(row(source_id='x' * 4097), '1.3')
    assert project(rows=[('w', 'batch', 0, 'north-a', row(source_id='x' * 4096))])['groups'][0]['source_id'] == 'x' * 4096


def test_full_scope_group_and_account_paging_invariant():
    rows = [('w', 'batch', i, 'north-a', row(source_id=f'sp-{i:03}', account=f'acct-{i:03}')) for i in range(52)]
    first = project(rows=rows)
    second = project({'after': '25', 'account_after': '25', 'snapshot': first['snapshot']}, rows)
    assert len(first['groups']) == len(first['accounts']) == 25
    assert first['identified_amounts'] == second['identified_amounts']
    assert first['identified_rows'] == second['identified_rows'] == 52
    assert set(g['id'] for g in first['groups']).isdisjoint(g['id'] for g in second['groups'])
    search = project({'q': 'acct-051'}, rows)
    assert search['account_matches'] == 1 and search['identified_amounts'] == first['identified_amounts']
    selected = project({'account': account_id('north-a', rows[0][4])}, rows)
    assert selected['row_count'] == 1
    with pytest.raises(Rejected, match='selection_unavailable'):
        project({'after': '25', 'snapshot': first['snapshot'], 'group': first['groups'][0]['id']}, rows)


def test_selected_group_current_page_and_detail_paging():
    rows = [('w', 'batch', i, 'north-a', row()) for i in range(51)]
    first = project(rows=rows)
    gid, snapshot = first['groups'][0]['id'], first['snapshot']
    detail = project({'group': gid, 'snapshot': snapshot}, rows)
    next_detail = project({'group': gid, 'snapshot': snapshot, 'row_after': '25'}, rows)
    assert detail['selected']['total_rows'] == 51 and len(detail['selected']['rows']) == 25
    assert next_detail['selected']['after'] == 25
    assert detail['groups'] == next_detail['groups']
    assert set(r['id'] for r in detail['selected']['rows']).isdisjoint(r['id'] for r in next_detail['selected']['rows'])
    assert all(r['group'] == gid for r in next_detail['selected']['rows'])
    assert detail['selected']['rows'][0]['fields']['name'] == {'state': 'absent', 'value': None}
    assert detail['selected']['rows'][0]['version'] == '1.0'
    assert detail['selected']['rows'][0]['charge_start'] == '2024-09-01T00:00:00+00:00'
    with pytest.raises(Rejected, match='selection_unavailable'):
        project({'group': 'f' * 64, 'snapshot': snapshot}, rows)
    with pytest.raises(Rejected, match='query_page'):
        project({'group': gid, 'snapshot': snapshot, 'row_after': '52'}, rows)


@pytest.mark.parametrize('params,reason', [({'group': 'a' * 64}, 'snapshot_required'), ({'after': '25'}, 'snapshot_required'), ({'account_after': '25'}, 'snapshot_required'), ({'row_after': '1'}, 'request_invalid'), ({'group': 'bad'}, 'request_invalid'), ({'basis': 'x'}, 'request_invalid'), ({'tenant': 'north-a'}, 'request_invalid'), ({'q': 'x' * 129}, 'request_invalid'), ({'row_after': '10001'}, 'request_invalid')])
def test_query_rejections(params, reason):
    with pytest.raises(Rejected, match=reason):
        selection(params)


def test_snapshot_pins_profile_mapping_source_and_filters():
    first = project()
    assert project({'basis': 'effective_cost'})['snapshot'] == first['snapshot']
    for params in ({'q': 'acct'}, {'period': '2024-09'}, {'provider': 'AWS'}):
        with pytest.raises(Rejected, match='query_snapshot_changed'):
            project({**params, 'snapshot': first['snapshot']})
    with pytest.raises(Rejected, match='query_snapshot_changed'):
        project({'snapshot': first['snapshot']}, snapshot='b' * 64)
    assert first['snapshot'] != first['source_snapshot'] and len(first['mapping_digest']) == 64


def test_preflight_missing_conflicting_mapping_and_scope_reject():
    p = Projection({})
    ctx = {'properties': {'key': 'north-a', 'timezone': 'UTC'}}
    coordinator = SimpleNamespace(a=SimpleNamespace(is_view=False, tenant='tenant-id'))
    entry = {'version': '1.0', 'batch_id': 'batch', 'mapping_digest': digest(canonical(versions()['1.0']).encode())}
    item = {'tenant_id': 'tenant-id', 'batch_id': 'batch'}
    p.preflight(None, coordinator, ctx, [('w', entry, item)])
    assert p.batch_versions == {('north-a', 'batch'): '1.0'}
    with pytest.raises(Rejected, match='delivery_mapping'):
        p.preflight(None, coordinator, ctx, [('w', {**entry, 'mapping_digest': 'f' * 64}, item)])
    with pytest.raises(Rejected, match='delivery_scope'):
        p.preflight(None, coordinator, ctx, [('w', entry, {**item, 'tenant_id': 'other'})])
    other = {'version': '1.2', 'batch_id': 'batch', 'mapping_digest': digest(canonical(versions()['1.2']).encode())}
    with pytest.raises(Rejected, match='delivery_conflict'):
        p.preflight(None, coordinator, ctx, [('w', entry, item), ('w', other, item)])
    with pytest.raises(Rejected, match='delivery_missing'):
        project(declared={})
    with pytest.raises(Rejected, match='scope_denied'):
        project(rows=[('w', 'batch', 0, 'foreign', row())], declared={})


def test_signed_input_bounds_and_projection_defensive_bound():
    assert project(rows=[('w', 'batch', 0, 'north-a', row(billed='-99999999999999999999.999999999999999999'))])['groups'][0]['billed_cost']['coefficient'] == '-' + '9' * 38
    for value in ('100000000000000000000', '-100000000000000000000'):
        with pytest.raises(Rejected, match='input_bound'):
            project(rows=[('w', 'batch', 0, 'north-a', row(billed=value))])
    with pytest.raises(Rejected, match='projection_bound'):
        costs([{'billed_cost': str(10 ** 50), 'effective_cost': None}])
    assert costs([{'billed_cost': None, 'effective_cost': None}])['billed_cost']['coefficient'] is None
    assert amounts([]) == []


def test_delivery_occurs_inside_collected_consumer(monkeypatch):
    active = []
    def collected(actor, limit, _consumer):
        assert limit == 1 and isinstance(_consumer, Projection)
        active.append(actor)
        result = project(deliver=_consumer.deliver)
        active.clear()
        return result
    monkeypatch.setattr('modules.focus_import.commitments.collected_query', collected)
    def deliver(body):
        assert active == ['reader-company-north'] and body['profile'] == 'commitment-source-evidence-sums-v1'
        return 'delivered'
    assert commitments('reader-company-north', {}, deliver) == 'delivered'


@pytest.fixture
def server(monkeypatch):
    calls = []
    def query(actor, params, deliver):
        selection(params)
        calls.append((actor, params))
        if params.get('provider') == 'revoked':
            raise Rejected('reader_revoked')
        if params.get('provider') == 'secret-error':
            raise RuntimeError('never reveal private driver detail')
        return deliver({'ok': True})
    monkeypatch.setattr('modules.focus_import.overview_http.commitments', query)
    srv = OverviewServer(0, 'reader-company-north', 8805)
    worker = threading.Thread(target=srv.serve_forever, daemon=True)
    worker.start()
    yield srv, calls
    srv.shutdown()
    srv.server_close()
    worker.join()


def fetch(server, path='/api/focus-commitments', headers=None, method='GET'):
    srv, _ = server
    conn = http.client.HTTPConnection('127.0.0.1', srv.server_port)
    conn.request(method, path, headers={'X-HB-Local-Client': '1', **(headers or {})})
    response = conn.getresponse()
    status, metadata, raw = response.status, dict(response.getheaders()), response.read()
    conn.close()
    return status, json.loads(raw) if raw else None, metadata


def test_http_route_limits_fixed_actor_and_sanitized_errors(server):
    query = 'provider=&period=all&account=&q=&basis=billed_cost&after=0&account_after=0&snapshot=' + 'a' * 64 + '&group=&row_after=0'
    status, body, headers = fetch(server, '/api/focus-commitments?' + query)
    assert status == 200 and body == {'ok': True} and headers['Cache-Control'] == 'no-store'
    assert server[1][0][0] == 'reader-company-north'
    assert fetch(server, '/api/focus-overview?' + query)[0] == 400
    assert fetch(server, '/api/focus-commitments?' + query + '&extra=1')[0] == 400
    assert fetch(server, '/api/focus-commitments?q=a&q=b')[0] == 400
    assert fetch(server, '/api/focus-commitments?provider=revoked')[0] == 403
    status, body, _ = fetch(server, '/api/focus-commitments?provider=secret-error')
    assert status == 503 and body['error'] == 'collected_unavailable' and 'private' not in str(body)
    assert fetch(server, method='POST')[0] == 405


@pytest.mark.parametrize('headers', [{'Host': 'evil.example'}, {'Origin': 'https://evil.example'}, {'Origin': 'null'}, {'Sec-Fetch-Site': 'cross-site'}, {'X-HB-Local-Client': ''}])
def test_http_hostile_requests_do_not_query(server, headers):
    assert fetch(server, headers=headers)[0] == 403 and server[1] == []


def test_shared_http_slots_protect_commitment_route(server):
    srv, calls = server
    srv.slot_wait = 0.05
    assert srv.slots.acquire(False) and srv.slots.acquire(False)
    try:
        assert fetch(server)[0] == 503 and calls == []
    finally:
        srv.slots.release()
        srv.slots.release()
