import json
import re
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlsplit
from uuid import uuid4

from .model import Rejected
from .overview import overview
from .commitments import commitments


class OverviewServer(ThreadingHTTPServer):
    daemon_threads = True

    def __init__(self, port, actor, ui_port):
        if not re.fullmatch('[a-z][a-z0-9-]{0,63}', actor) or type(ui_port) is not int or not 1 <= ui_port <= 65535:
            raise ValueError('trusted_local_profile')
        self.actor = actor
        self.slots = threading.BoundedSemaphore(2)
        super().__init__(('127.0.0.1', port), Handler)
        self.origin = f'http://127.0.0.1:{self.server_port}'
        self.origins = {self.origin, f'http://127.0.0.1:{ui_port}'}


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def setup(self):
        super().setup()
        self.connection.settimeout(15)

    def reply(self, status, body):
        data = json.dumps(body).encode()
        if len(data) > 8 * 1024 * 1024:
            status, data = 503, b'{"error":"response_bound"}'
        self.send_response(status)
        for key, value in {'Content-Type': 'application/json', 'Content-Length': str(len(data)),
                           'Cache-Control': 'no-store', 'X-Content-Type-Options': 'nosniff',
                           'X-Frame-Options': 'DENY', 'Referrer-Policy': 'no-referrer'}.items():
            self.send_header(key, value)
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        request_id = uuid4().hex
        def failure(status, code):
            self.reply(status, {'error': code, 'request_id': request_id})
        if self.headers.get_all('Host') != [self.server.origin.removeprefix('http://')]:
            return failure(403, 'origin_denied')
        if len(self.headers.get_all('Origin', [])) > 1 or len(self.headers.get_all('Sec-Fetch-Site', [])) > 1:
            return failure(403, 'origin_denied')
        if self.headers.get('Origin') is not None and self.headers.get('Origin') not in self.server.origins:
            return failure(403, 'origin_denied')
        if self.headers.get('Sec-Fetch-Site') not in (None, 'same-origin', 'none') or self.headers.get_all('X-HB-Local-Client') != ['1']:
            return failure(403, 'origin_denied')
        if len(self.path) > 8192 or self.headers.get('Content-Length') not in (None, '0'):
            return failure(400, 'request_invalid')
        path = urlsplit(self.path)
        if path.path not in ('/api/focus-overview', '/api/focus-commitments') or path.scheme or path.netloc or path.fragment:
            return failure(404, 'not_found')
        try:
            values = parse_qs(path.query, keep_blank_values=True, strict_parsing=True, max_num_fields=10 if path.path == '/api/focus-commitments' else 8)
            if any(len(v) != 1 for v in values.values()):
                raise ValueError('duplicate')
            params = {k: v[0] for k, v in values.items()}
        except ValueError:
            return failure(400, 'request_invalid')
        if not self.server.slots.acquire(blocking=False):
            return failure(503, 'source_busy')
        try:
            consumer = commitments if path.path == '/api/focus-commitments' else overview
            consumer(self.server.actor, params, lambda body: self.reply(200, body))
        except Rejected as exc:
            reason = str(exc)
            if reason in ('request_invalid', 'selection_unavailable', 'query_page'):
                failure(400, reason)
            elif reason in ('query_snapshot_changed', 'snapshot_required'):
                failure(409, reason)
            elif reason in ('reader_revoked', 'permission', 'view_unavailable', 'view_member_denied'):
                failure(403, 'scope_denied')
            else:
                failure(503, 'collected_unavailable')
        except Exception:
            failure(503, 'collected_unavailable')
        finally:
            self.server.slots.release()

    def deny_method(self):
        self.reply(405, {'error': 'method_denied'})

    do_POST = deny_method
    do_PUT = deny_method
    do_PATCH = deny_method
    do_DELETE = deny_method
    do_OPTIONS = deny_method
    do_HEAD = deny_method
