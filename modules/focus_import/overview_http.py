import json
import re
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlsplit
from uuid import uuid4

from .model import Rejected
from .overview import overview
from .commitments import commitments
from .marketplace import marketplace
from .receipt_projection import MAX_RESPONSE_BYTES, receipts
from .connection_projection import connection_metadata
from .collection_status import collection_status


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
        if self.path.split('?', 1)[0] in ('/api/focus-receipts', '/api/focus-connections', '/api/focus-collection-status') and len(data) > MAX_RESPONSE_BYTES:
            status, data = 503, json.dumps({'error': 'response_bound', 'request_id': uuid4().hex}).encode()
        elif len(data) > 8 * 1024 * 1024:
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
        if len(self.path) > 8192 or len(self.headers.get_all('Content-Length', [])) > 1 or self.headers.get('Transfer-Encoding') is not None or self.headers.get('Content-Length') not in (None, '0'):
            return failure(400, 'request_invalid')
        try:
            path = urlsplit(self.path)
        except ValueError:
            return failure(400, 'request_invalid')
        if path.path not in ('/api/focus-overview', '/api/focus-commitments', '/api/focus-marketplace', '/api/focus-receipts', '/api/focus-connections', '/api/focus-collection-status') or path.scheme or path.netloc or path.fragment:
            return failure(404, 'not_found')
        try:
            values = parse_qs(path.query, keep_blank_values=True, strict_parsing=True, max_num_fields={'/api/focus-commitments': 10, '/api/focus-marketplace': 9, '/api/focus-receipts': 5, '/api/focus-connections': 5, '/api/focus-collection-status': 0}.get(path.path, 8))
            if any(len(v) != 1 for v in values.values()):
                raise ValueError('duplicate')
            params = {k: v[0] for k, v in values.items()}
        except ValueError:
            return failure(400, 'request_invalid')
        if not self.server.slots.acquire(blocking=False):
            return failure(503, 'source_busy')
        try:
            consumer = {'/api/focus-commitments': commitments, '/api/focus-marketplace': marketplace, '/api/focus-receipts': receipts, '/api/focus-connections': connection_metadata, '/api/focus-collection-status': collection_status}.get(path.path, overview)
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
        body = {'error': 'method_denied'}
        if self.path.split('?', 1)[0] in ('/api/focus-receipts', '/api/focus-connections', '/api/focus-collection-status'):
            body['request_id'] = uuid4().hex
        self.reply(405, body)

    do_POST = deny_method
    do_PUT = deny_method
    do_PATCH = deny_method
    do_DELETE = deny_method
    do_OPTIONS = deny_method
    do_HEAD = deny_method
