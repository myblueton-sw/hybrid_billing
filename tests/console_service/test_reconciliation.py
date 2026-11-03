import io
import json
import os
import sys
import threading
import unittest
import urllib.error
import urllib.request
from pathlib import Path
from unittest.mock import patch

import psycopg

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from modules.focus_local.contract import Rejected
from modules.focus_local.database import Authority
from modules.focus_local.http_service import LocalServer
from modules.focus_local.reconciliation import Reconciliation, comparisons
from modules.focus_local.storage import Storage

ROOT = Path(__file__).resolve().parents[2]


def group(coefficient, missing=0):
    return {'rows': 2, 'measures': {'billed_cost': {'coefficient': coefficient, 'present': 2-missing, 'missing': missing}}}


class Comparison(unittest.TestCase):
    def test_exact_large_and_negative_differences(self):
        key = '["USD","Hours","Hours"]'
        for n in (0, 1, -1, 9999999999999999999999999):
            r = comparisons({key: group(n)}, {key: group(n-1)})[0]
            self.assertEqual(r['difference'], '-0.000001')
            self.assertEqual(r['state'], 'difference')

    def test_missing_is_not_zero_and_units_never_combined(self):
        a, b = '["USD","Hours","Hours"]', '["EUR","GB","GB"]'
        rows = comparisons({a: group(0, 1), b: group(0)}, {a: group(0), b: group(0)})
        self.assertEqual(len(rows), 2)
        usd = next(r for r in rows if r['currency'] == 'USD')
        self.assertEqual(usd['state'], 'incomplete')
        self.assertIsNone(usd['difference'])
        missing = comparisons({b: group(0)}, {})[0]
        self.assertIsNone(missing['actual'])
        self.assertIsNone(missing['actual_missing'])

    def test_original_read_is_bounded_and_closed(self):
        store = object.__new__(Storage)
        from unittest.mock import Mock
        store.s3 = Mock()
        store.bucket = 'synthetic'
        body = io.BytesIO(b'123456')
        store.s3.get_object.return_value = {'ContentLength': 6, 'Body': body}
        with self.assertRaises(Rejected):
            store.bounded_original('key', 4)
        self.assertTrue(body.closed)


class Integration(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if os.environ.get('HB_CONSOLE_INTEGRATION') != 'isolated':
            raise unittest.SkipTest('Requires existing isolated synthetic HB141 fixtures')
        cls.servers = [LocalServer(0, actor, ROOT / 'apps/console/dist') for actor in ('approver-a', 'approver-b', 'loader-a')]
        for server in cls.servers:
            threading.Thread(target=server.serve_forever, daemon=True).start()
        s = Reconciliation('approver-a')
        try:
            cls.candidate = s.candidates()['candidates'][0]
        finally:
            s.close()

    @classmethod
    def tearDownClass(cls):
        for s in cls.servers:
            s.shutdown()
            s.server_close()

    def call(self, suffix, body=None, server=0, headers=None):
        s = self.servers[server]
        h = {'X-HB-Local-Client': '1', 'Origin': s.origin, 'Content-Type': 'application/json'}
        h.update(headers or {})
        req = urllib.request.Request(s.origin + '/api/local-reconciliation' + suffix, data=json.dumps(body).encode() if body is not None else None, headers=h)
        try:
            with urllib.request.urlopen(req) as r:
                return r.status, json.load(r)
        except urllib.error.HTTPError as e:
            return e.code, json.load(e)

    def test_real_source_and_storage_consistency(self):
        r = self.candidate
        code, result = self.call('/candidates/' + r['release_id'] + '/revalidate', {'expected_manifest': r['manifest_digest']})
        self.assertEqual(code, 200, result)
        self.assertEqual(result['state'], 'match')
        self.assertTrue(all(c['passed'] for c in result['checks']))
        self.assertIn('independent_cross_validation_missing', result['blockers'])
        self.assertFalse(result['durable'])
        self.assertTrue(all(v['difference'] == '0.000000' for v in result['comparisons']))

    def test_capability_rechecked_before_result_and_no_publication_writes(self):
        s = Reconciliation('approver-a')
        a = s.authority
        try:
            def snapshot():
                with a.inspection():
                    return {table: a.conn.execute('SELECT count(*) AS n FROM hb_local.' + table).fetchone()['n'] for table in ('confirmation', 'publication_decision', 'outbox')}
            before = snapshot()
            for _ in range(2):
                s.revalidate(self.candidate['release_id'], self.candidate['manifest_digest'])
            self.assertEqual(snapshot(), before)
            with a.inspection() as cap:
                initial = cap.copy()
            with patch.object(a, 'inspection_capability', side_effect=[initial, Rejected('permission')]):
                with self.assertRaisesRegex(Rejected, 'permission'):
                    s.context()
            changed = {**initial, 'policy_revision': initial['policy_revision'] + 1}
            with patch.object(a, 'inspection_capability', side_effect=[initial, changed]):
                with self.assertRaisesRegex(Rejected, 'policy_changed'):
                    s.context()
            with patch.object(a, 'inspection_capability', side_effect=Rejected('policy_unavailable')):
                with self.assertRaisesRegex(Rejected, 'policy_unavailable'):
                    s.context()
        finally:
            s.close()

    def test_complete_counts_before_truncation_and_registry_failure(self):
        s = Reconciliation('approver-a')
        try:
            expanded = [{'state': 'difference'}] * 103 + [{'state': 'incomplete'}] * 4
            with patch('modules.focus_local.reconciliation.comparisons', return_value=expanded):
                r = s.revalidate(self.candidate['release_id'], self.candidate['manifest_digest'])
                self.assertEqual(len(r['comparisons']), 100)
                self.assertEqual((r['comparison_count'], r['mismatch_count'], r['incomplete_count']), (107, 103, 4))
                self.assertTrue(r['truncated'])
            for reason in ('registry_changed', 'manifest_digest'):
                with patch.object(s, 'check_registry', side_effect=Rejected(reason)):
                    r = s.revalidate(self.candidate['release_id'], self.candidate['manifest_digest'])
                    self.assertEqual(r['state'], 'unavailable')
                    self.assertIn(reason, r['blockers'])
        finally:
            s.close()

    def test_scope_digest_role_and_request_guards(self):
        path = '/candidates/' + self.candidate['release_id']
        self.assertEqual(self.call(path, server=1)[0], 403)
        self.assertEqual(self.call('/profile', server=2)[0], 403)
        self.assertEqual(self.call(path + '/revalidate', {'expected_manifest': '0'*64})[0], 409)
        self.assertEqual(self.call('/profile', headers={'Origin':'https://other.invalid'})[0], 403)
        self.assertEqual(self.call('/profile', headers={'X-HB-Local-Client':''})[0], 403)
        self.assertEqual(self.call('/profile?actor=approver-b')[0], 400)
        self.assertEqual(self.call(path + '/confirm', {})[0], 404)
        self.assertEqual(self.call(path + '/revalidate', {'expected_manifest':'0'*64,'actor':'approver-b'})[0], 400)

    def test_unavailable_or_corrupt_evidence_never_matches(self):
        path = '/candidates/' + self.candidate['release_id'] + '/revalidate'
        payload = {'expected_manifest': self.candidate['manifest_digest']}
        with patch.object(Storage, 'bounded_original', return_value=b'corrupt'):
            code, body = self.call(path, payload)
            self.assertEqual(code, 200)
            self.assertEqual(body['state'], 'unavailable')
            self.assertIn('original_digest', body['blockers'])
        with patch.object(Storage, 'rows', side_effect=ConnectionError):
            self.assertEqual(self.call(path, payload)[0], 503)
        with patch.object(Storage, 'aggregates', return_value={}):
            code, body = self.call(path, payload)
            self.assertEqual(code, 200)
            self.assertEqual(body['state'], 'difference')
            self.assertGreater(body['incomplete_count'], 0)
            self.assertTrue(all(r['actual'] is None for r in body['comparisons']))

    def test_inspection_readonly_scope_resets_and_mutation_guards_remain(self):
        a = Authority('approver-a')
        try:
            with self.assertRaises(psycopg.errors.ReadOnlySqlTransaction):
                with a.inspection():
                    self.assertEqual(a.conn.execute('SHOW transaction_read_only').fetchone()['transaction_read_only'], 'on')
                    a.conn.execute('UPDATE hb_local.connection SET lock_token=lock_token WHERE connection_id=%s', (a.scope[2],))
            self.assertEqual(a.conn.execute('SELECT count(*) AS n FROM hb_local.release').fetchone()['n'], 0)
            with a.conn.transaction():
                with self.assertRaisesRegex(Rejected, 'inspection_transaction'):
                    with a.inspection():
                        pass
            with self.assertRaisesRegex(Rejected, 'quarantine_hold'):
                with a.transaction('confirm'):
                    pass
        finally:
            a.close()
        loader = Authority('loader-a')
        try:
            with self.assertRaisesRegex(Rejected, 'permission'):
                with loader.inspection():
                    pass
        finally:
            loader.close()
        self.assertIn('quarantine_hold', self.call('/candidates')[1]['blockers'])
        self.assertIn('operation_unresolved', self.call('/candidates', server=1)[1]['blockers'])


if __name__ == '__main__':
    if '--isolated-profile' in sys.argv:
        sys.argv.remove('--isolated-profile')
        os.environ['HB_CONSOLE_INTEGRATION'] = 'isolated'
    unittest.main()
