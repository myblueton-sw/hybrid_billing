import json
import os
import sys
import threading
import time
import unittest
import urllib.error
import urllib.request
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from modules.focus_local.http_service import LocalServer
from modules.focus_local.storage import Storage

ROOT = Path(__file__).resolve().parents[2]


class Integration(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if os.environ.get("HB_CONSOLE_INTEGRATION") != "isolated":
            raise unittest.SkipTest("Requires an explicitly isolated local service profile")
        cls.a = LocalServer(0, 'loader-a', ROOT / 'apps/console/dist')
        cls.b = LocalServer(0, 'loader-b', ROOT / 'apps/console/dist')
        for server in (cls.a, cls.b):
            threading.Thread(target=server.serve_forever, daemon=True).start()
        cls.file = (ROOT / 'tests/fixtures/focus/synthetic-1.3.csv').read_bytes()
        cls.revision = str(time.time_ns())

    @classmethod
    def tearDownClass(cls):
        for server in (cls.a, cls.b):
            server.shutdown()
            server.server_close()

    def call(self, server, path='', data=None, headers=None):
        h = {'X-HB-Local-Client': '1', 'Origin': server.origin}
        if data is not None:
            h.update({'Content-Type': 'text/csv', 'X-Focus-Version': '1.3', 'X-Source-Revision': self.revision})
        h.update(headers or {})
        req = urllib.request.Request(server.origin + '/api/local-imports' + path, data=data, headers=h)
        try:
            with urllib.request.urlopen(req) as r:
                return r.status, json.load(r)
        except urllib.error.HTTPError as e:
            return e.code, json.load(e)

    def test_real_storage_replay_scope_and_restart(self):
        code, first = self.call(self.a, data=self.file)
        self.assertEqual(code, 200, first)
        self.assertEqual(first['state'], 'verified')
        code, same = self.call(self.a, data=self.file)
        self.assertEqual(code, 200)
        self.assertTrue(same['duplicate'])
        self.assertEqual(same['release_id'], first['release_id'])
        code, changed = self.call(self.a, data=self.file.replace(b'0.300001', b'0.300002'))
        self.assertEqual(code, 409)
        self.assertEqual(changed['error'], 'revision_conflict')
        code, detail = self.call(self.a, '/' + first['release_id'])
        self.assertEqual(code, 200)
        self.assertEqual(detail['releases'][0]['operation_state'], 'terminal')
        self.assertEqual(self.call(self.b, '/' + first['release_id'])[0], 403)
        self.assertEqual(self.call(self.b, data=self.file)[0], 422)
        with LocalServer(0, 'loader-a', ROOT / 'apps/console/dist') as restarted:
            threading.Thread(target=restarted.serve_forever, daemon=True).start()
            self.assertEqual(self.call(restarted, '/' + first['release_id'])[1]['releases'][0]['release_id'], first['release_id'])
            restarted.shutdown()

    def test_request_boundaries(self):
        self.assertEqual(self.call(self.a, '?actor=loader-b')[0], 400)
        self.assertEqual(self.call(self.a, headers={'Origin': 'https://other.invalid'})[0], 403)
        self.assertEqual(self.call(self.a, headers={'Host': 'other.invalid'})[0], 403)
        self.assertEqual(self.call(self.a, headers={'X-HB-Local-Client': ''})[0], 403)
        self.assertEqual(self.call(self.a, data=b'x', headers={'Content-Length': str(9*1024*1024)})[0], 413)
        self.assertEqual(self.call(self.a, data=self.file, headers={'X-Source-Revision': '9223372036854775808'})[0], 400)
        extra = self.file.replace(b'BillingAccountId', b'PromptBody', 1)
        self.assertEqual(self.call(self.a, data=extra)[1]['error'], 'source_fields')

    def test_quarantine_has_actual_terminal_operation(self):
        data = self.file.replace(b'0.300001', b'invalid-amount')
        code, result = self.call(self.a, data=data, headers={'X-Source-Revision': str(int(self.revision)+1)})
        self.assertEqual(code, 200, result)
        self.assertEqual(result['state'], 'quarantined')
        saved = self.call(self.a)[1]['quarantine']
        self.assertTrue(any(r['receipt_id']==result['receipt_id'] and r['operation_state']=='terminal' for r in saved))

    def test_unavailable_is_not_empty_success(self):
        with patch('modules.focus_local.http_service.Intake', side_effect=ConnectionError):
            code, body = self.call(self.a)
        self.assertEqual(code, 503)
        self.assertEqual(body['error'], 'service_unavailable')
        self.assertNotIn('releases', body)

    def test_unknown_operation_remains_inspectable(self):
        data = self.file.replace(b'synthetic-account-a', b'synthetic-account-b').replace(b'synthetic-sub-a', b'synthetic-sub-b')
        with patch.object(Storage, 'put', side_effect=ConnectionError):
            self.assertEqual(self.call(self.b, data=data)[0], 503)
        code, saved = self.call(self.b)
        self.assertEqual(code, 200)
        self.assertTrue(any(r['operation_state']=='unknown' for r in saved['releases']))
        self.assertEqual(self.call(self.b, data=data)[1]['error'], 'operation_unresolved')


if __name__ == '__main__':
    if '--isolated-profile' in sys.argv:
        sys.argv.remove('--isolated-profile')
        os.environ['HB_CONSOLE_INTEGRATION'] = 'isolated'
    unittest.main()
