import http.client
import json
import socket
import threading
import time

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


class RecordingSlots(threading.BoundedSemaphore):
    def __init__(self, value):
        super().__init__(value)
        self.waiting = threading.Semaphore(0)
        self.released = threading.Semaphore(0)
        self.seen = threading.local()
        self.steps = 0

    def acquire(self, blocking=True, timeout=None):
        if timeout is not None:
            self.steps += 1
            if not getattr(self.seen, 'waiting', False):
                self.seen.waiting = True
                self.waiting.release()
        return super().acquire(blocking, timeout)

    def release(self, n=1):
        super().release(n)
        self.released.release()


@pytest.fixture
def held(server):
    srv, calls = server
    srv.slots = RecordingSlots(2)
    srv.slot_wait = 30
    for _ in range(2):
        srv.slots.acquire()
    state = {'held': 2}

    def release(count=1):
        for _ in range(count):
            srv.slots.release()
            assert srv.slots.released.acquire(timeout=5)
            state['held'] -= 1

    yield srv, calls, release
    release(state['held'])


def background(server, path='/api/focus-overview'):
    result = []
    worker = threading.Thread(target=lambda: result.append(fetch(server, path)), daemon=True)
    worker.start()
    return worker, result


def test_request_waits_for_a_slot_and_is_served(server, held):
    srv, calls, release = held
    worker, result = background(server)
    assert srv.slots.waiting.acquire(timeout=5)
    assert calls == [] and worker.is_alive()
    release()
    worker.join(5)
    assert result[0][0] == 200 and result[0][1] == {'ok': True} and len(calls) == 1
    assert srv.slots.released.acquire(timeout=5)


def test_wait_ends_with_source_busy_and_no_query(server, held):
    srv, calls, _ = held
    srv.slot_wait = 0.05
    status, body, _ = fetch(server)
    assert status == 503 and body['error'] == 'source_busy' and calls == []
    assert srv.slots.waiting.acquire(blocking=False)


def test_full_waiting_places_answer_busy_without_waiting(server, held):
    srv, calls, release = held
    for _ in range(2):
        srv.admission.acquire()
    workers = [background(server) for _ in range(6)]
    for _ in range(6):
        assert srv.slots.waiting.acquire(timeout=5)
    status, body, _ = fetch(server)
    assert status == 503 and body['error'] == 'source_busy'
    assert all(worker.is_alive() for worker, _ in workers) and calls == []
    assert not srv.slots.waiting.acquire(blocking=False)
    for _ in range(2):
        srv.admission.release()
    release(2)
    for worker, _ in workers:
        worker.join(10)
    assert [result[0][0] for _, result in workers] == [200] * 6 and len(calls) == 6
    for _ in range(6):
        assert srv.slots.released.acquire(timeout=5)


def test_caller_that_left_while_waiting_is_not_queried(server, held):
    srv, calls, release = held
    srv.slot_step = srv.slot_wait
    caller = socket.create_connection(('127.0.0.1', srv.server_port))
    caller.sendall(b'GET /api/focus-overview HTTP/1.1\r\nHost: 127.0.0.1:%d\r\nX-HB-Local-Client: 1\r\n\r\n' % srv.server_port)
    assert srv.slots.waiting.acquire(timeout=5)
    caller.close()
    time.sleep(0.2)
    release()
    assert srv.slots.released.acquire(timeout=5)
    assert calls == []
    assert srv.slots.acquire(blocking=False)
    srv.slots.release()
    assert srv.slots.released.acquire(timeout=5)
    assert fetch(server)[0] == 200 and len(calls) == 1
    assert srv.slots.released.acquire(timeout=5)


def test_slot_is_released_when_a_waited_query_fails(server, held):
    srv, calls, release = held
    worker, result = background(server, '/api/focus-overview?provider=offline')
    assert srv.slots.waiting.acquire(timeout=5)
    release()
    worker.join(5)
    assert result[0][0] == 503 and result[0][1]['error'] == 'collected_unavailable'
    assert srv.slots.released.acquire(timeout=5)
    assert fetch(server)[0] == 200
    assert srv.slots.released.acquire(timeout=5)


def test_transport_invalid_requests_do_not_wait(server, held):
    srv, calls, _ = held
    started = time.monotonic()
    assert fetch(server, headers={'Origin': 'https://evil.example'})[0] == 403
    assert fetch(server, '/api/focus-overview?q=a&q=b')[0] == 400
    assert fetch(server, '/api/unknown')[0] == 404
    assert time.monotonic() - started < 5 and calls == []
    assert not srv.slots.waiting.acquire(blocking=False)


def test_admission_is_returned_on_every_path(server, held):
    srv, calls, release = held
    srv.admission = RecordingSlots(8)
    srv.slot_wait = 0.05
    for _ in range(10):
        assert fetch(server)[0] == 503
        assert srv.admission.released.acquire(timeout=5)
    release(2)
    assert fetch(server)[0] == 200
    assert srv.slots.released.acquire(timeout=5)
    assert srv.admission.released.acquire(timeout=5)
    assert all(srv.admission.acquire(blocking=False) for _ in range(8))
    assert not srv.admission.acquire(blocking=False)
    for _ in range(8):
        srv.admission.release()


def leave_while_waiting(srv):
    caller = socket.create_connection(('127.0.0.1', srv.server_port))
    caller.sendall(b'GET /api/focus-overview HTTP/1.1\r\nHost: 127.0.0.1:%d\r\nX-HB-Local-Client: 1\r\n\r\n' % srv.server_port)
    assert srv.slots.waiting.acquire(timeout=5)
    caller.close()
    assert srv.admission.released.acquire(timeout=5)


def test_caller_that_left_frees_its_waiting_place_before_a_slot(server, held, capsys):
    srv, calls, _ = held
    srv.admission = RecordingSlots(8)
    srv.slot_step = 0.02
    leave_while_waiting(srv)
    assert calls == [] and capsys.readouterr().err == ''
    assert not srv.slots.released.acquire(blocking=False)
    assert all(srv.admission.acquire(blocking=False) for _ in range(8))
    for _ in range(8):
        srv.admission.release()


def test_callers_that_left_one_after_another_leave_the_places_free(server, held, capsys):
    srv, calls, release = held
    srv.admission = RecordingSlots(8)
    srv.slot_step = 0.02
    for _ in range(8):
        leave_while_waiting(srv)
    assert capsys.readouterr().err == ''
    worker, result = background(server)
    assert srv.slots.waiting.acquire(timeout=5)
    assert calls == []
    release()
    worker.join(5)
    assert result[0][0] == 200 and len(calls) == 1
    assert srv.slots.released.acquire(timeout=5)


def test_total_wait_is_kept_across_steps(server, held):
    srv, calls, _ = held
    srv.slot_wait, srv.slot_step = 0.3, 0.05
    started = time.monotonic()
    status, body, _ = fetch(server)
    assert status == 503 and body['error'] == 'source_busy' and calls == []
    assert time.monotonic() - started >= 0.3 and srv.slots.steps >= 2


LINE = 'Collected FOCUS local reader: a request ended without a complete response.\n'


def test_error_output_is_one_fixed_line(server, capsys):
    srv, _ = server
    try:
        raise RuntimeError('secret driver detail')
    except RuntimeError:
        srv.handle_error(None, ('127.0.0.1', 1))
    assert capsys.readouterr().err == LINE


def test_failed_write_after_a_failed_query_prints_no_detail(server, monkeypatch, capsys):
    def broken(self, status, body):
        raise BrokenPipeError('secret pipe detail')
    monkeypatch.setattr('modules.focus_import.overview_http.Handler.reply', broken)
    with pytest.raises((http.client.RemoteDisconnected, ConnectionError)):
        fetch(server, '/api/focus-overview?provider=offline')
    assert capsys.readouterr().err == LINE
