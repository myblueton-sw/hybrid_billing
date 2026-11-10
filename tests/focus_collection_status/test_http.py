import http.client
import json
import threading

import pytest

from modules.focus_import.model import Rejected
from modules.focus_import.overview_http import OverviewServer


@pytest.fixture
def status_server(monkeypatch, status_fixture):
    fixture, calls = status_fixture, []
    fixture.scope()
    def read(actor, params, deliver):
        assert actor == fixture.authority.actor
        calls.append(params)
        from modules.focus_import.collection_status import selection
        selection(params)
        return deliver(fixture.status([fixture.row()]))
    monkeypatch.setattr('modules.focus_import.overview_http.collection_status', read)
    server = OverviewServer(0, fixture.authority.actor, 8817)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    yield server, calls
    server.shutdown()
    server.server_close()
    thread.join()


def fetch(server, path='/api/focus-collection-status', headers=None, method='GET'):
    client = http.client.HTTPConnection('127.0.0.1', server[0].server_port)
    client.request(method, path, headers={'X-HB-Local-Client': '1', **(headers or {})})
    response = client.getresponse()
    code, raw, metadata = response.status, response.read(), dict(response.getheaders())
    client.close()
    return code, json.loads(raw) if raw else None, metadata


def raw(server, path='/api/focus-collection-status', headers=()):
    client = http.client.HTTPConnection('127.0.0.1', server[0].server_port)
    client.putrequest('GET', path, skip_host=True)
    for key, value in [('Host', f'127.0.0.1:{server[0].server_port}'), ('X-HB-Local-Client', '1'), *headers]:
        client.putheader(key, value)
    client.endheaders()
    response = client.getresponse()
    code, body = response.status, json.loads(response.read())
    client.close()
    return code, body


def test_closed_fixed_get_and_security_headers(status_server):
    for headers in [{}, {'Origin': 'http://127.0.0.1:8817', 'Sec-Fetch-Site': 'same-origin'}, {'Origin': status_server[0].origin}]:
        code, body, metadata = fetch(status_server, headers=headers)
        assert code == 200 and body['row_count'] == 1 and body['profile'] == 'retained-focus-collection-status-v1'
        assert 'request_id' not in body and 'source_snapshot' not in body
        assert metadata['Cache-Control'] == 'no-store' and metadata['Content-Type'] == 'application/json'
        assert metadata['X-Content-Type-Options'] == 'nosniff' and metadata['X-Frame-Options'] == 'DENY'


@pytest.mark.parametrize('query', ['q=', 'provider=AWS', 'actor=operator', 'tenant=north-a', 'after=0',
                                  'snapshot=' + 'a' * 64, 'q=a&q=b', 'unexpected', 'q=a&x=b&y=c'])
def test_any_query_parameter_rejects_before_status_consumer(status_server, query):
    code, body, _ = fetch(status_server, path='/api/focus-collection-status?' + query)
    assert code == 400 and body['error'] == 'request_invalid'
    assert set(body) == {'error', 'request_id'} and status_server[1] == []


@pytest.mark.parametrize('headers,code', [({'Host': 'evil.example'}, 403), ({'Origin': 'null'}, 403),
    ({'Origin': 'https://evil.example'}, 403), ({'Sec-Fetch-Site': 'same-site'}, 403),
    ({'Sec-Fetch-Site': 'cross-site'}, 403), ({'X-HB-Local-Client': ''}, 403),
    ({'Content-Length': '1'}, 400), ({'Transfer-Encoding': 'chunked'}, 400)])
def test_existing_request_guards_apply_without_querying(status_server, headers, code):
    actual, body, _ = fetch(status_server, headers=headers)
    assert actual == code and set(body) == {'error', 'request_id'} and status_server[1] == []


@pytest.mark.parametrize('headers,code', [([('Host', 'evil.example')], 403),
    ([('Origin', 'http://127.0.0.1:8817')] * 2, 403), ([('Sec-Fetch-Site', 'same-origin')] * 2, 403),
    ([('X-HB-Local-Client', '1')], 403), ([('Content-Length', '0')] * 2, 400)])
def test_duplicate_security_headers_fail_closed(status_server, headers, code):
    assert raw(status_server, headers=headers)[0] == code
    assert status_server[1] == []


def test_wrong_routes_and_malformed_absolute_uri_do_not_query(status_server):
    for path in ['/api/focus-collection-status/extra', '/api/focus%2dcollection-status',
                 '/api/focus-collection-status#fragment', 'http://evil.example/api/focus-collection-status']:
        assert raw(status_server, path=path)[0] == 404
    assert raw(status_server, path='http://[')[0] == 400
    assert status_server[1] == []


def test_method_denial_and_concurrency_bound(status_server):
    for method in ['POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS', 'HEAD']:
        code, body, _ = fetch(status_server, method=method)
        assert code == 405
        if method != 'HEAD':
            assert body['error'] == 'method_denied' and set(body) == {'error', 'request_id'}
    assert status_server[1] == []
    for _ in range(2): status_server[0].slots.acquire()
    try:
        code, body, _ = fetch(status_server)
        assert code == 503 and body['error'] == 'source_busy' and status_server[1] == []
    finally:
        for _ in range(2): status_server[0].slots.release()


@pytest.mark.parametrize('reason,code,error', [('reader_revoked', 403, 'scope_denied'), ('permission', 403, 'scope_denied'),
    ('view_member_denied', 403, 'scope_denied'), ('collection_status_unavailable', 503, 'collected_unavailable')])
def test_guard_errors_are_sanitized_not_zero_results(status_server, monkeypatch, reason, code, error):
    def rejected(actor, params, deliver):
        raise Rejected(reason)
    monkeypatch.setattr('modules.focus_import.overview_http.collection_status', rejected)
    actual, body, _ = fetch(status_server)
    assert actual == code and body['error'] == error and set(body) == {'error', 'request_id'}


def test_status_wire_byte_bound_source_failure_and_previous_dispatch(status_server, monkeypatch):
    monkeypatch.setattr('modules.focus_import.overview_http.collection_status',
                        lambda actor, params, deliver: deliver({'payload': 'x' * (2 * 1024 * 1024)}))
    code, body, _ = fetch(status_server)
    assert code == 503 and body['error'] == 'response_bound' and set(body) == {'error', 'request_id'}
    def unavailable(actor, params, deliver):
        raise RuntimeError('PRIVATE-SOURCE-DETAIL')
    monkeypatch.setattr('modules.focus_import.overview_http.collection_status', unavailable)
    code, body, _ = fetch(status_server)
    assert code == 503 and body['error'] == 'collected_unavailable' and 'PRIVATE' not in str(body)
    for path, name in [('/api/focus-overview', 'overview'), ('/api/focus-commitments', 'commitments'),
                       ('/api/focus-marketplace', 'marketplace'), ('/api/focus-receipts', 'receipts'),
                       ('/api/focus-connections', 'connection_metadata')]:
        monkeypatch.setattr('modules.focus_import.overview_http.' + name, lambda actor, params, deliver: deliver({'prior': True}))
        assert fetch(status_server, path=path)[:2] == (200, {'prior': True})
