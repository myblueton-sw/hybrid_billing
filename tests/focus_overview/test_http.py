import http.client
import json
import threading

import pytest

from modules.focus_import.model import Rejected
from modules.focus_import.overview_http import OverviewServer


@pytest.fixture
def server(monkeypatch):
    calls = []
    def query(actor, params, deliver):
        from modules.focus_import.overview import selection
        selection(params)
        calls.append((actor, params))
        if params.get('provider') == 'revoked':
            raise Rejected('reader_revoked')
        if params.get('provider') == 'offline':
            raise RuntimeError('secret driver detail')
        return deliver({'ok': True})
    monkeypatch.setattr('modules.focus_import.overview_http.overview', query)
    srv = OverviewServer(0, 'reader-company-north', 8805)
    worker = threading.Thread(target=srv.serve_forever, daemon=True)
    worker.start()
    yield srv, calls
    srv.shutdown()
    srv.server_close()
    worker.join()


def fetch(server, path='/api/focus-overview', method='GET', headers=None):
    srv, _ = server
    connection = http.client.HTTPConnection('127.0.0.1', srv.server_port)
    connection.request(method, path, headers={'X-HB-Local-Client': '1', **(headers or {})})
    response = connection.getresponse()
    status, body, metadata = response.status, response.read(), dict(response.getheaders())
    connection.close()
    return status, json.loads(body) if body else None, metadata


def test_valid_direct_and_proxy_origin(server):
    for headers in [{}, {'Origin': 'http://127.0.0.1:8805', 'Sec-Fetch-Site': 'same-origin'}, {'Origin': server[0].origin}]:
        status, body, metadata = fetch(server, headers=headers)
        assert status == 200 and body == {'ok': True}
        assert metadata['Cache-Control'] == 'no-store'
    assert len(server[1]) == 3


@pytest.mark.parametrize('headers', [
    {'Origin': 'https://evil.example'}, {'Origin': 'null'}, {'Host': 'evil.example'},
    {'Sec-Fetch-Site': 'cross-site'}, {'Sec-Fetch-Site': 'same-site'}, {'X-HB-Local-Client': ''},
    {'Origin': 'https://evil.example', 'X-Forwarded-Host': '127.0.0.1:8805'},
])
def test_hostile_requests_never_query(server, headers):
    assert fetch(server, headers=headers)[0] == 403
    assert server[1] == []


@pytest.mark.parametrize('query', ['actor=foreign', 'company=foreign', 'basis=x', 'q=a&q=b', 'q=' + 'a' * 129, 'after=25', 'x=1&y=2&z=3&a=4&b=5&c=6&d=7&e=8&f=9'])
def test_bad_query(server, query):
    status = fetch(server, path='/api/focus-overview?' + query)[0]
    assert status in (400, 409)
    assert server[1] == []


def test_error_sanitization_and_methods(server):
    assert fetch(server, path='/api/focus-overview?provider=revoked')[0] == 403
    status, body, _ = fetch(server, path='/api/focus-overview?provider=offline')
    assert status == 503 and body['error'] == 'collected_unavailable'
    assert 'secret' not in str(body)
    for method in ['POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS', 'HEAD']:
        assert fetch(server, method=method)[0] == 405
