import json
import mimetypes
import re
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlsplit
from uuid import uuid4

from .contract import MAX_BYTES, Rejected
from .intake_view import Intake


class LocalServer(ThreadingHTTPServer):
    daemon_threads = True

    def __init__(self, port, actor, build, env=None):
        if actor not in ('loader-a', 'loader-b'):
            raise ValueError('loader_required')
        self.actor, self.env = actor, env
        self.build = Path(build).resolve()
        manifest = json.loads((self.build / 'build-manifest.json').read_text())
        self.assets = {item['path'] for item in manifest['files']}
        super().__init__(('127.0.0.1', port), Handler)
        self.origin = f'http://127.0.0.1:{self.server_port}'


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def setup(self):
        super().setup()
        self.connection.settimeout(15)

    def reply(self, status, body, kind='application/json'):
        data = json.dumps(body).encode() if kind == 'application/json' else body
        self.send_response(status)
        self.send_header('Content-Type', kind)
        self.send_header('Content-Length', str(len(data)))
        self.send_header('Cache-Control', 'no-store')
        self.send_header('X-Content-Type-Options', 'nosniff')
        self.send_header('X-Frame-Options', 'DENY')
        self.send_header('Referrer-Policy', 'no-referrer')
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        self.handle_request(False)

    def do_POST(self):
        self.handle_request(True)

    def handle_request(self, mutation):
        request_id = uuid4().hex
        def failure(status, code):
            self.reply(status, {'error': code, 'request_id': request_id})
        expected = self.server.origin.removeprefix('http://')
        if self.headers.get_all('Host') != [expected]:
            return failure(403, 'origin_denied')
        origin = self.headers.get('Origin')
        if origin is not None and origin != self.server.origin:
            return failure(403, 'origin_denied')
        if self.headers.get('Sec-Fetch-Site') not in (None, 'same-origin', 'none'):
            return failure(403, 'origin_denied')
        path = urlsplit(self.path)
        if path.query or path.scheme or path.netloc or '%' in path.path:
            return failure(400, 'request_invalid')
        if not path.path.startswith('/api/'):
            if mutation:
                return failure(405, 'method_denied')
            asset = path.path.lstrip('/') or 'index.html'
            if asset not in self.server.assets:
                return failure(404, 'not_found')
            target = (self.server.build / asset).resolve()
            if not target.is_relative_to(self.server.build) or not target.is_file():
                return failure(404, 'not_found')
            return self.reply(200, target.read_bytes(), mimetypes.guess_type(asset)[0] or 'application/octet-stream')
        if self.headers.get('X-HB-Local-Client') != '1':
            return failure(403, 'origin_denied')
        if mutation and origin != self.server.origin:
            return failure(403, 'origin_denied')
        allowed = path.path in ('/api/local-imports', '/api/local-imports/profile') or re.fullmatch(r'/api/local-imports/[0-9a-f-]{36}', path.path)
        if not allowed or mutation and path.path != '/api/local-imports':
            return failure(404, 'not_found')
        data = None
        if mutation:
            lengths = self.headers.get_all('Content-Length') or []
            if self.headers.get('Transfer-Encoding') or len(lengths) != 1 or not lengths[0].isdigit():
                return failure(400, 'request_invalid')
            length = int(lengths[0])
            if not 0 < length <= MAX_BYTES:
                return failure(413, 'file_size')
            if self.headers.get('Content-Type') != 'text/csv':
                return failure(415, 'csv_required')
            version, revision = self.headers.get('X-Focus-Version', ''), self.headers.get('X-Source-Revision', '')
            if not re.fullmatch(r'[1-9][0-9]{0,18}', revision) or int(revision) >= 2**63 or len(version) > 32:
                return failure(400, 'request_invalid')
            try:
                data = self.rfile.read(length)
            except TimeoutError:
                return failure(408, 'request_timeout')
            if len(data) != length:
                return failure(400, 'request_invalid')
        service = None
        try:
            service = Intake(self.server.actor, self.server.env)
            if mutation:
                result = service.receive(data, version, int(revision))
            elif path.path.endswith('/profile'):
                result = service.context()
            elif path.path == '/api/local-imports':
                result = service.receipts()
            else:
                result = service.receipts(path.path.rsplit('/', 1)[1])
            self.reply(200, {**result, 'request_id': request_id})
        except Rejected as exc:
            reason = str(exc)
            if reason in ('permission', 'policy_unavailable', 'release_unavailable', 'release_identity'):
                failure(403, 'scope_denied')
            elif reason in ('revision_conflict', 'stale_source_revision', 'operation_unresolved'):
                failure(409, reason)
            elif reason in ('source_scope', 'source_period', 'source_admission', 'source_fields', 'unsupported_version'):
                failure(422, reason)
            else:
                failure(422, 'validation_rejected')
        except Exception:
            failure(503, 'service_unavailable')
        finally:
            if service:
                service.close()
