import copy
import http.client
import json
import threading

import pytest

from modules.focus_import.model import Rejected
from modules.focus_import.overview_http import OverviewServer


@pytest.fixture
def connection_server(monkeypatch, connection_fixture):
    f, calls = connection_fixture, []
    def read(actor, params, deliver):
        assert actor == f.authority.actor
        calls.append(params)
        return deliver(f.render(params))
    monkeypatch.setattr('modules.focus_import.overview_http.connection_metadata', read)
    server = OverviewServer(0, f.authority.actor, 8817)
    worker = threading.Thread(target=server.serve_forever, daemon=True)
    worker.start()
    yield server, f, calls
    server.shutdown()
    server.server_close()
    worker.join()


def fetch(server, query='', method='GET', headers=None, path='/api/focus-connections'):
    connection = http.client.HTTPConnection('127.0.0.1', server[0].server_port)
    connection.request(method, path + ('?' + query if query else ''), headers={'X-HB-Local-Client': '1', **(headers or {})})
    response = connection.getresponse()
    status, body, metadata = response.status, response.read(), dict(response.getheaders())
    connection.close()
    return status, json.loads(body) if body else None, metadata


def raw(server, path='/api/focus-connections', headers=()):
    connection = http.client.HTTPConnection('127.0.0.1', server[0].server_port)
    connection.putrequest('GET', path, skip_host=True)
    for key, value in [('Host', f'127.0.0.1:{server[0].server_port}'), ('X-HB-Local-Client', '1'), *headers]:
        connection.putheader(key, value)
    connection.endheaders()
    response = connection.getresponse()
    status, body = response.status, json.loads(response.read())
    connection.close()
    return status, body


def test_actual_guarded_get_uses_exact_scope_and_readonly_headers(connection_server):
    before = copy.deepcopy(connection_server[1].state)
    for headers in [{}, {'Origin': 'http://127.0.0.1:8817', 'Sec-Fetch-Site': 'same-origin'}, {'Origin': connection_server[0].origin}]:
        status, body, metadata = fetch(connection_server, headers=headers)
        assert status == 200 and body['profile'] == 'retained-focus-connections-v1' and body['total'] == 1
        assert 'request_id' not in body and metadata['Cache-Control'] == 'no-store'
        assert metadata['Content-Type'] == 'application/json' and metadata['X-Content-Type-Options'] == 'nosniff'
        assert metadata['X-Frame-Options'] == 'DENY' and metadata['Referrer-Policy'] == 'no-referrer'
    assert connection_server[1].state == before


@pytest.mark.parametrize('headers', [
    {'Origin': 'https://evil.example'}, {'Origin': 'null'}, {'Host': 'evil.example'},
    {'Sec-Fetch-Site': 'cross-site'}, {'Sec-Fetch-Site': 'same-site'}, {'X-HB-Local-Client': ''},
    {'Content-Length': '1'}, {'Transfer-Encoding': 'chunked'},
])
def test_hostile_headers_never_reach_projection(connection_server, headers):
    status, body, _ = fetch(connection_server, headers=headers)
    assert status == (400 if 'Content-Length' in headers or 'Transfer-Encoding' in headers else 403)
    assert set(body) == {'error', 'request_id'} and connection_server[2] == []


@pytest.mark.parametrize('headers,status', [
    ([('Host', 'evil.example')], 403), ([('Origin', 'http://127.0.0.1:8817')] * 2, 403),
    ([('Sec-Fetch-Site', 'same-origin')] * 2, 403), ([('X-HB-Local-Client', '1')], 403),
    ([('Content-Length', '0')] * 2, 400),
])
def test_duplicate_security_headers_are_rejected(connection_server, headers, status):
    assert raw(connection_server, headers=headers)[0] == status
    assert connection_server[2] == []


@pytest.mark.parametrize('query', [
    'actor=operator', 'tenant=north-a', 'company=north', 'view=wrong', 'q=A', 'q=' + 'a' * 129,
    'after=-1', 'after=01', 'after=10001', 'q=a&q=b',
    'view=connections&q=a&after=0&snapshot=x&permission_revision=y&extra=x', 'q',
])
def test_invalid_query_never_returns_records(connection_server, query):
    status, body, _ = fetch(connection_server, query)
    assert status == 400 and set(body) == {'error', 'request_id'} and 'records' not in body


def test_missing_and_stale_pins_are_conflicts(connection_server):
    assert fetch(connection_server, 'after=25')[0] == 409
    status, body, _ = fetch(connection_server, 'snapshot=' + 'a' * 64 + '&permission_revision=' + 'b' * 64)
    assert status == 409 and body['error'] == 'query_snapshot_changed' and len(body['request_id']) == 32


def test_methods_and_slot_bound_do_not_query(connection_server):
    for method in ['POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS', 'HEAD']:
        status, body, _ = fetch(connection_server, method=method)
        assert status == 405
        if method != 'HEAD':
            assert set(body) == {'error', 'request_id'} and body['error'] == 'method_denied'
    assert connection_server[2] == []
    connection_server[0].slot_wait = 0.05
    for _ in range(2): connection_server[0].slots.acquire()
    try:
        status, body, _ = fetch(connection_server)
        assert status == 503 and body['error'] == 'source_busy' and connection_server[2] == []
    finally:
        for _ in range(2): connection_server[0].slots.release()


def test_wire_bound_and_sanitized_errors(connection_server, monkeypatch):
    monkeypatch.setattr('modules.focus_import.overview_http.connection_metadata',
                        lambda actor, params, deliver: deliver({'payload': 'x' * (2 * 1024 * 1024)}))
    status, body, _ = fetch(connection_server)
    assert status == 503 and set(body) == {'error', 'request_id'} and body['error'] == 'response_bound'
    def failure(actor, params, deliver):
        raise RuntimeError('DO-NOT-EXPOSE-PRIVATE')
    monkeypatch.setattr('modules.focus_import.overview_http.connection_metadata', failure)
    status, body, _ = fetch(connection_server)
    assert status == 503 and body['error'] == 'collected_unavailable' and 'PRIVATE' not in str(body)
    def denied(actor, params, deliver):
        raise Rejected('reader_revoked')
    monkeypatch.setattr('modules.focus_import.overview_http.connection_metadata', denied)
    assert fetch(connection_server)[0] == 403


@pytest.mark.parametrize('path,status', [('/api/focus-connections/foreign', 404), ('/api/focus%2dconnections', 404),
                                        ('http://evil.example/api/focus-connections', 403), ('/api/focus-connections#fragment', 404)])
def test_wrong_routes_never_query(connection_server, path, status):
    assert fetch(connection_server, path=path)[0] == status
    assert connection_server[2] == []


def test_malformed_absolute_uri_does_not_crash_or_query(connection_server):
    status, body = raw(connection_server, path='http://[')
    assert status == 400 and body['error'] == 'request_invalid' and connection_server[2] == []
    assert raw(connection_server, path='http://evil.example/api/focus-connections')[0] == 404
    assert connection_server[2] == []


def test_existing_dispatch_and_receipt_wire_bound_remain(connection_server, monkeypatch):
    for path, attr in [('/api/focus-overview', 'overview'), ('/api/focus-commitments', 'commitments'),
                       ('/api/focus-marketplace', 'marketplace'), ('/api/focus-receipts', 'receipts')]:
        monkeypatch.setattr('modules.focus_import.overview_http.' + attr,
                            lambda actor, params, deliver: deliver({'previous': True}))
        assert fetch(connection_server, path=path)[:2] == (200, {'previous': True})
    monkeypatch.setattr('modules.focus_import.overview_http.receipts',
                        lambda actor, params, deliver: deliver({'payload': 'x' * (2 * 1024 * 1024)}))
    assert fetch(connection_server, path='/api/focus-receipts')[1]['error'] == 'response_bound'
