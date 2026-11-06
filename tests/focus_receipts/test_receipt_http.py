import copy
import http.client
import json
import threading

import pytest

from modules.focus_import.overview_http import OverviewServer


@pytest.fixture
def receipt_server(monkeypatch, receipt_fixture):
    fixture = receipt_fixture
    fixture.add_delivery()
    calls = []
    def read(actor, params, deliver):
        assert actor == fixture.authority.actor
        calls.append(params)
        return deliver(fixture.render(params))
    monkeypatch.setattr('modules.focus_import.overview_http.receipts', read)
    server = OverviewServer(0, fixture.authority.actor, 8808)
    worker = threading.Thread(target=server.serve_forever, daemon=True)
    worker.start()
    yield server, fixture, calls
    server.shutdown()
    server.server_close()
    worker.join()


def fetch(receipt_server, query='', method='GET', headers=None, path='/api/focus-receipts'):
    server = receipt_server[0]
    connection = http.client.HTTPConnection('127.0.0.1', server.server_port)
    connection.request(method, path + ('?' + query if query else ''), headers={'X-HB-Local-Client': '1', **(headers or {})})
    response = connection.getresponse()
    result = response.status, response.read(), dict(response.getheaders())
    connection.close()
    return result[0], json.loads(result[1]) if result[1] else None, result[2]


def test_actual_handler_get_projection_no_transient_success_id(receipt_server):
    before = copy.deepcopy(receipt_server[1].state)
    for headers in [{}, {'Origin': 'http://127.0.0.1:8808', 'Sec-Fetch-Site': 'same-origin'}]:
        status, body, metadata = fetch(receipt_server, headers=headers)
        assert status == 200 and body['profile'] == 'retained-focus-receipts-v1'
        assert body['total'] == 1 and 'request_id' not in body
        assert metadata['Cache-Control'] == 'no-store' and metadata['X-Content-Type-Options'] == 'nosniff'
    assert receipt_server[1].state == before


@pytest.mark.parametrize('headers', [
    {'Origin': 'https://evil.example'}, {'Origin': 'null'}, {'Host': 'evil.example'},
    {'Sec-Fetch-Site': 'cross-site'}, {'Sec-Fetch-Site': 'same-site'}, {'X-HB-Local-Client': ''},
    {'Content-Length': '1'},
])
def test_untrusted_requests_never_reach_projection(receipt_server, headers):
    expected = 400 if 'Content-Length' in headers else 403
    assert fetch(receipt_server, headers=headers)[0] == expected
    assert receipt_server[2] == []


@pytest.mark.parametrize('query', [
    'actor=operator', 'tenant=north-a', 'company=north', 'view=wrong', 'q=A', 'q=' + 'a' * 129,
    'after=-1', 'after=01', 'after=10001', 'q=a&q=b', 'view=imports&q=a&after=0&snapshot=x&permission_revision=y&extra=x',
])
def test_invalid_query_never_returns_evidence(receipt_server, query):
    status, body, _ = fetch(receipt_server, query)
    assert status == 400 and set(body) == {'error', 'request_id'}
    assert 'records' not in body


def test_stale_missing_pins_and_safe_errors(receipt_server):
    assert fetch(receipt_server, 'after=25')[0] == 409
    status, body, _ = fetch(receipt_server, 'snapshot=' + 'a' * 64 + '&permission_revision=' + 'b' * 64)
    assert status == 409 and body['error'] == 'query_snapshot_changed'
    assert len(body['request_id']) == 32


def test_denied_methods_and_parallel_slot_bound(receipt_server):
    for method in ['POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS', 'HEAD']:
        status, body, _ = fetch(receipt_server, method=method)
        assert status == 405
        if method != 'HEAD':
            assert set(body) == {'error', 'request_id'} and body['error'] == 'method_denied'
    assert receipt_server[2] == []
    for _ in range(2):
        receipt_server[0].slots.acquire()
    try:
        status, body, _ = fetch(receipt_server)
        assert status == 503 and body['error'] == 'source_busy'
        assert receipt_server[2] == []
    finally:
        for _ in range(2):
            receipt_server[0].slots.release()


def test_new_endpoint_wire_byte_bound_and_sanitized_failure(receipt_server, monkeypatch):
    def oversized(actor, params, deliver):
        return deliver({'payload': 'x' * (2 * 1024 * 1024)})
    monkeypatch.setattr('modules.focus_import.overview_http.receipts', oversized)
    status, body, _ = fetch(receipt_server)
    assert status == 503 and set(body) == {'error', 'request_id'} and body['error'] == 'response_bound'
    def unavailable(actor, params, deliver):
        raise RuntimeError('PRIVATE-PASSWORD-DO-NOT-RETURN')
    monkeypatch.setattr('modules.focus_import.overview_http.receipts', unavailable)
    status, body, _ = fetch(receipt_server)
    assert status == 503 and body['error'] == 'collected_unavailable'
    assert 'PRIVATE' not in str(body)


def test_old_get_dispatch_and_unknown_path_preserved(receipt_server, monkeypatch):
    monkeypatch.setattr('modules.focus_import.overview_http.overview', lambda actor, params, deliver: deliver({'existing': 'overview'}))
    assert fetch(receipt_server, path='/api/focus-overview')[:2] == (200, {'existing': 'overview'})
    assert fetch(receipt_server, path='/api/focus-receipts/foreign')[0] == 404
    assert fetch(receipt_server, path='/api/focus%2dreceipts')[0] == 404


def test_malformed_url_never_crashes_or_queries(receipt_server):
    server = receipt_server[0]
    connection = http.client.HTTPConnection('127.0.0.1', server.server_port)
    connection.putrequest('GET', 'http://[', skip_host=True)
    connection.putheader('Host', server.origin.removeprefix('http://'))
    connection.putheader('X-HB-Local-Client', '1')
    connection.endheaders()
    response = connection.getresponse()
    body = json.loads(response.read())
    connection.close()
    assert response.status == 400 and body['error'] == 'request_invalid'
    assert receipt_server[2] == []


def test_duplicate_length_and_transfer_encoding_are_denied(receipt_server):
    server = receipt_server[0]
    connection = http.client.HTTPConnection('127.0.0.1', server.server_port)
    connection.putrequest('GET', '/api/focus-receipts')
    connection.putheader('X-HB-Local-Client', '1')
    connection.putheader('Content-Length', '0')
    connection.putheader('Content-Length', '1')
    connection.endheaders()
    response = connection.getresponse()
    response.read()
    connection.close()
    assert response.status == 400
    assert fetch(receipt_server, headers={'Transfer-Encoding': 'chunked'})[0] == 400
    assert receipt_server[2] == []
