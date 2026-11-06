import http.client
import json
import threading
from datetime import datetime, timezone
from decimal import Decimal
from types import SimpleNamespace

import pytest

from modules.focus_import.marketplace import Projection, amounts, costs, marketplace, selection
from modules.focus_import.marketplace_fields import FIELDS, Mapping
from modules.focus_import.model import Rejected, canonical, digest, versions
from modules.focus_import.overview import account_id
from modules.focus_import.overview_http import OverviewServer


def row(publisher='Publisher Ltd', issuer='Market Ltd', account='acct', subaccount=None,
        provider='AWS', currency='USD', billed='0.342', effective='0', extra=None):
    raw = {'PublisherName': publisher, 'InvoiceIssuerName': issuer,
           'ChargeDescription': 'Marketplace software usage', 'ChargeCategory': 'Usage',
           'ChargeClass': None, 'ChargeFrequency': 'Usage-Based', 'SkuId': 'sku-1',
           'CommitmentDiscountId': None, 'private_secret': 'never-project'}
    raw.update(extra or {})
    return {'provider': provider, 'account': account, 'subaccount': subaccount, 'service': 'Software',
            'currency': currency, 'billed_cost': Decimal(billed) if billed is not None else None,
            'effective_cost': Decimal(effective) if effective is not None else None,
            **{name: datetime(2024, 9 if name.endswith('start') else 10, 1, tzinfo=timezone.utc)
               for name in ('charge_start', 'charge_end', 'billing_start', 'billing_end')},
            'source_json': raw}


def project(params=None, rows=None, declared=None, snapshot='a' * 64, deliver=None):
    rows = rows if rows is not None else [('w', 'batch', 1, 'north-a', row())]
    p = Projection(params or {}, deliver)
    declared = declared if declared is not None else {(r[3], r[1]): '1.0' for r in rows}
    tenants = [{'tenant_id': name + '-id', 'properties': {'key': name, 'timezone': 'UTC'}}
               for name in ('north-a', 'north-b')]
    scopes = [('w', {'version': version, 'batch_id': batch,
                     'mapping_digest': digest(canonical(versions()[version]).encode())},
               {'batch_id': batch, 'tenant_id': tenant + '-id'})
              for (tenant, batch), version in declared.items()]
    p.preflight(None, SimpleNamespace(a=SimpleNamespace(is_view=True)),
                {'audience': {'company_id': 'company-north'}, 'tenants': tenants}, scopes)
    result = {'classification': 'exploratory-query-only',
              'period_attribution': 'charge-start-in-tenant-timezone-no-proration',
              'snapshot': snapshot, 'coverage': []}
    return p.consume(result, rows)


def test_exact_name_partition_not_provider_or_keyword_or_legal_authority():
    rows = [('w', 'batch', i, 'north-a', value) for i, value in enumerate([
        row(), row(publisher='Amazon Web Services, Inc.', issuer='Amazon Web Services, Inc.'),
        row(publisher=None), row(issuer=None), row(publisher='Market Ltd', issuer='Market Ltd'),
        row(publisher='publisher ltd', issuer='Publisher Ltd', extra={'ChargeDescription': 'ordinary usage'})])]
    d = project(rows=rows)
    assert d['counts'] == {'candidate': 2, 'no_difference': 2, 'unknown': 2}
    assert d['row_count'] == sum(d['counts'].values()) == 6
    assert len(d['rows']) == 2 and d['financial_authority'] is False
    assert d['candidate_amounts'][0]['billed_cost']['coefficient'] == '684000000000000000'
    assert d['candidate_amounts'][0]['effective_cost']['coefficient'] == '0'
    for forbidden in ('savings', 'fee', 'withheld', 'net_settlement', 'paid', 'offer_id', 'transaction_id', 'source_json', 'private_secret', 'never-project'):
        assert forbidden not in str(d)
    assert set(d['rows'][0]['fields']) == set(FIELDS)


@pytest.mark.parametrize('value,state', [(None, 'source_null'), ('NULL', 'source_null'),
                                        ('null', 'source_null'), (r'\N', 'source_null'),
                                        ('', 'source_null'), ('  ', 'blank'), (0, 'invalid_type'),
                                        ({'secret': 'value'}, 'invalid_type')])
def test_pinned_null_invalid_names_never_become_candidate(value, state):
    d = project(rows=[('w', 'batch', 0, 'north-a', row(issuer=value))])
    assert d['counts'] == {'candidate': 0, 'no_difference': 0, 'unknown': 1}
    assert d['rows'] == [] and d['candidate_amounts'] == []
    assert Mapping().fields(row(issuer=value), '1.0')['issuer'] == {'state': state, 'value': None}
    assert d['unknown_amounts'][0]['rows'] == 1


def test_absent_unmapped_and_bad_source_shape_boundaries():
    r = row()
    del r['source_json']['PublisherName']
    assert Mapping().fields(r, '1.0')['publisher']['state'] == 'absent'
    assert project(rows=[('w', 'batch', 0, 'north-a', r)])['counts']['unknown'] == 1
    fields = Mapping().fields(row(), '1.2')
    assert fields['publisher'] == {'state': 'unmapped_version', 'value': 'Publisher Ltd'}
    assert project(declared={('north-a', 'batch'): '1.2'})['counts']['unknown'] == 1
    for raw in ('not-json', '[]', [], None):
        value = row()
        value['source_json'] = raw
        with pytest.raises(Rejected, match='source_shape'):
            Mapping().fields(value, '1.0')
    raw_json = row()
    raw_json['source_json'] = json.dumps(raw_json['source_json'])
    assert Mapping().fields(raw_json, '1.0')['issuer']['state'] == 'value'
    for version in ('1.0', '1.2'):
        with pytest.raises(Rejected, match='field_bound'):
            Mapping().fields(row(publisher='x' * 4097), version)
    assert len(Mapping().fields(row(publisher='x' * 4096), '1.0')['publisher']['value']) == 4096


def test_exact_signed_missing_zero_and_per_currency_before_paging():
    rows = [('w', 'batch', i, 'north-a', value) for i, value in enumerate([
        row(billed='100', effective='80'), row(billed='-100', effective=None, extra={'ChargeCategory': 'Credit'}),
        row(billed=None, effective='-80'), row(billed=None, effective=None),
        row(currency='EUR', billed='4', effective='3')])]
    d = project(rows=rows)
    usd = next(a for a in d['candidate_amounts'] if a['currency'] == 'USD')
    eur = next(a for a in d['candidate_amounts'] if a['currency'] == 'EUR')
    assert usd == {'currency': 'USD', 'rows': 4,
                   'billed_cost': {'coefficient': '0', 'present': 2, 'missing': 2},
                   'effective_cost': {'coefficient': '0', 'present': 2, 'missing': 2}}
    assert eur['billed_cost']['coefficient'] == str(4 * 10 ** 18)
    missing = project(rows=[('w', 'batch', 0, 'north-a', row(billed=None, effective=None))])
    assert missing['candidate_amounts'][0]['billed_cost']['coefficient'] is None
    assert amounts([]) == [] and project(rows=[])['candidate_amounts'] == []


def test_scope_canonical_identity_and_per_tenant_batch_versions():
    rows = [('w', 'same-batch', 1, 'north-a', row()), ('w', 'same-batch', 1, 'north-b', row()),
            ('w', 'same-batch', 2, 'north-a', row(account='other', subaccount='sub'))]
    d = project(rows=rows, declared={('north-a', 'same-batch'): '1.0', ('north-b', 'same-batch'): '1.2'})
    assert d['counts'] == {'candidate': 2, 'no_difference': 0, 'unknown': 1}
    assert len({r['id'] for r in d['rows']}) == 2
    account = account_id('north-a', rows[2][4])
    selected = project({'account': account}, rows=rows)
    assert selected['row_count'] == 1 and selected['rows'][0]['sub_account'] == 'sub'
    assert selected['rows'][0]['account'] == account
    with pytest.raises(Rejected, match='scope_denied'):
        project(rows=[('w', 'batch', 0, 'foreign', row())], declared={})
    with pytest.raises(Rejected, match='row_identity'):
        project(rows=[('w', 'batch', 0, 'north-a', row())] * 2)


def test_full_scope_paging_account_search_and_selected_row_membership():
    rows = [('w', 'batch', i, 'north-a', row(account=f'acct-{i:03}')) for i in range(52)]
    first = project(rows=rows)
    second = project({'after': '25', 'account_after': '25', 'snapshot': first['snapshot']}, rows)
    assert len(first['rows']) == len(first['accounts']) == 25
    assert first['counts'] == second['counts'] == {'candidate': 52, 'no_difference': 0, 'unknown': 0}
    assert first['candidate_amounts'] == second['candidate_amounts']
    assert set(r['id'] for r in first['rows']).isdisjoint(r['id'] for r in second['rows'])
    found = project({'q': '  acct-051  '}, rows)
    assert found['selection']['q'] == 'acct-051' and found['account_matches'] == 1
    assert found['candidate_amounts'] == first['candidate_amounts'] and found['counts'] == first['counts']
    absent = project({'q': 'no-such-account'}, rows)
    assert absent['accounts'] == [] and absent['counts']['candidate'] == 52
    selected = project({'row': first['rows'][0]['id'], 'snapshot': first['snapshot']}, rows)
    assert selected['selected'] == first['rows'][0]
    for params in ({'row': 'f' * 64}, {'row': first['rows'][0]['id'], 'after': '25'}):
        with pytest.raises(Rejected, match='selection_unavailable'):
            project({**params, 'snapshot': first['snapshot']}, rows)
    for params in ({'after': '53'}, {'account_after': '53'}):
        with pytest.raises(Rejected, match='query_page'):
            project({**params, 'snapshot': first['snapshot']}, rows)


@pytest.mark.parametrize('params,reason', [({'row': 'a' * 64}, 'snapshot_required'),
    ({'after': '25'}, 'snapshot_required'), ({'account_after': '25'}, 'snapshot_required'),
    ({'row': 'bad'}, 'request_invalid'), ({'row': None}, 'request_invalid'),
    ({'basis': 'net_settlement'}, 'request_invalid'), ({'tenant': 'north-a'}, 'request_invalid'),
    ({'q': 'x' * 129}, 'request_invalid'), ({'after': '10001'}, 'request_invalid'),
    ({'account': 'bad'}, 'request_invalid'), ({'period': '2024-13'}, 'request_invalid')])
def test_query_validation(params, reason):
    with pytest.raises(Rejected, match=reason):
        selection(params)


def test_snapshot_source_filter_and_mapping_pins():
    first = project()
    assert project({'basis': 'effective_cost'})['snapshot'] == first['snapshot']
    assert first['snapshot'] != first['source_snapshot']
    for params in ({'q': 'acct'}, {'provider': 'AWS'}, {'period': '2024-09'}):
        with pytest.raises(Rejected, match='query_snapshot_changed'):
            project({**params, 'snapshot': first['snapshot']})
    with pytest.raises(Rejected, match='query_snapshot_changed'):
        project({'snapshot': first['snapshot']}, snapshot='b' * 64)
    mapped = project(declared={('north-a', 'batch'): '1.2'})
    assert mapped['snapshot'] != first['snapshot'] and len(first['mapping_digest']) == 64


def test_preflight_missing_conflicting_and_invalid_delivery():
    p = Projection({})
    coordinator = SimpleNamespace(a=SimpleNamespace(is_view=False, tenant='tenant-id'))
    context = {'properties': {'key': 'north-a', 'timezone': 'UTC'}}
    entry = {'version': '1.0', 'batch_id': 'batch', 'mapping_digest': digest(canonical(versions()['1.0']).encode())}
    item = {'tenant_id': 'tenant-id', 'batch_id': 'batch'}
    p.preflight(None, coordinator, context, [('w', entry, item)])
    assert p.batch_versions == {('north-a', 'batch'): '1.0'}
    for changed, reason in (({**entry, 'mapping_digest': 'f' * 64}, 'delivery_mapping'),
                            ({**entry, 'version': 'unsupported'}, 'delivery_mapping'),
                            ({**entry, 'batch_id': 'other'}, 'delivery_scope')):
        with pytest.raises(Rejected, match=reason):
            p.preflight(None, coordinator, context, [('w', changed, item)])
    with pytest.raises(Rejected, match='delivery_scope'):
        p.preflight(None, coordinator, context, [('w', entry, {**item, 'tenant_id': 'other'})])
    other = {'version': '1.2', 'batch_id': 'batch', 'mapping_digest': digest(canonical(versions()['1.2']).encode())}
    with pytest.raises(Rejected, match='delivery_conflict'):
        p.preflight(None, coordinator, context, [('w', entry, item), ('w', other, item)])
    with pytest.raises(Rejected, match='delivery_missing'):
        project(declared={})


def test_signed_input_and_projection_defensive_bounds():
    d = project(rows=[('w', 'batch', 0, 'north-a', row(billed='-99999999999999999999.999999999999999999'))])
    assert d['candidate_amounts'][0]['billed_cost']['coefficient'] == '-' + '9' * 38
    for value in ('100000000000000000000', '-100000000000000000000'):
        with pytest.raises(Rejected, match='input_bound'):
            project(rows=[('w', 'batch', 0, 'north-a', row(billed=value))])
    with pytest.raises(Rejected, match='projection_bound'):
        costs([{'billed_cost': str(10 ** 50), 'effective_cost': None}])


def test_response_delivery_inside_collected_consumer(monkeypatch):
    active = []
    def collected(actor, limit, _consumer):
        assert limit == 1 and isinstance(_consumer, Projection)
        active.append(actor)
        result = project(deliver=_consumer.deliver)
        active.clear()
        return result
    monkeypatch.setattr('modules.focus_import.marketplace.collected_query', collected)
    def deliver(body):
        assert active == ['reader-company-north'] and body['financial_authority'] is False
        return 'delivered'
    assert marketplace('reader-company-north', {}, deliver) == 'delivered'



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
    monkeypatch.setattr('modules.focus_import.overview_http.marketplace', query)
    srv = OverviewServer(0, 'reader-company-north', 8805)
    worker = threading.Thread(target=srv.serve_forever, daemon=True)
    worker.start()
    yield srv, calls
    srv.shutdown()
    srv.server_close()
    worker.join()


def fetch(server, path='/api/focus-marketplace', headers=None, method='GET'):
    srv, _ = server
    conn = http.client.HTTPConnection('127.0.0.1', srv.server_port)
    conn.request(method, path, headers={'X-HB-Local-Client': '1', **(headers or {})})
    response = conn.getresponse()
    status, metadata, raw = response.status, dict(response.getheaders()), response.read()
    conn.close()
    return status, json.loads(raw) if raw else None, metadata


def test_http_route_bounds_fixed_actor_snapshot_and_sanitized_failures(server):
    query = 'provider=&period=all&account=&q=&basis=billed_cost&after=0&account_after=0&snapshot=' + 'a' * 64 + '&row='
    status, body, headers = fetch(server, '/api/focus-marketplace?' + query)
    assert status == 200 and body == {'ok': True}
    assert headers['Cache-Control'] == 'no-store' and headers['X-Content-Type-Options'] == 'nosniff'
    assert server[1][0][0] == 'reader-company-north'
    assert fetch(server, '/api/focus-marketplace?' + query + '&extra=1')[0] == 400
    assert fetch(server, '/api/focus-overview?' + query)[0] == 400
    assert fetch(server, '/api/focus-commitments?' + query)[0] == 400
    assert fetch(server, '/api/focus-marketplace?q=a&q=b')[0] == 400
    assert fetch(server, '/api/focus-marketplace?tenant=foreign')[0] == 400
    assert fetch(server, '/api/focus-marketplace?after=25')[0] == 409
    assert fetch(server, '/api/focus-marketplace?row=' + 'f' * 64)[0] == 409
    assert fetch(server, '/api/focus-marketplace?provider=revoked')[0] == 403
    status, body, _ = fetch(server, '/api/focus-marketplace?provider=secret-error')
    assert status == 503 and set(body) == {'error', 'request_id'} and 'private' not in str(body)
    for method in ('POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS', 'HEAD'):
        assert fetch(server, method=method)[0] == 405


@pytest.mark.parametrize('headers', [{'Host': 'evil.example'}, {'Origin': 'https://evil.example'},
                                    {'Origin': 'null'}, {'Sec-Fetch-Site': 'cross-site'},
                                    {'X-HB-Local-Client': ''}, {'Content-Length': '1'}])
def test_http_hostile_requests_do_not_query(server, headers):
    assert fetch(server, headers=headers)[0] in (400, 403) and server[1] == []


@pytest.mark.parametrize('name,values', [('Host', ['127.0.0.1', 'evil.example']),
    ('Origin', ['http://127.0.0.1:8805', 'http://127.0.0.1:8805']),
    ('Sec-Fetch-Site', ['same-origin', 'same-origin']),
    ('X-HB-Local-Client', ['1', '1'])])
def test_http_duplicate_security_headers_reject_without_query(server, name, values):
    srv, calls = server
    conn = http.client.HTTPConnection('127.0.0.1', srv.server_port)
    conn.putrequest('GET', '/api/focus-marketplace', skip_host=name == 'Host')
    if name != 'X-HB-Local-Client':
        conn.putheader('X-HB-Local-Client', '1')
    for value in values:
        conn.putheader(name, value)
    conn.endheaders()
    response = conn.getresponse()
    assert response.status == 403 and calls == []
    response.read()
    conn.close()


def test_shared_http_slots_protect_marketplace_route(server):
    srv, calls = server
    assert srv.slots.acquire(False) and srv.slots.acquire(False)
    try:
        assert fetch(server)[0] == 503 and calls == []
    finally:
        srv.slots.release()
        srv.slots.release()
