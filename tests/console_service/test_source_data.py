import json
import threading
import urllib.request
import urllib.error
import copy
import os
import sys
import unittest
from decimal import Decimal
from pathlib import Path
from unittest.mock import Mock, patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from modules.focus_local.contract import Rejected, canonical, digest, load_registry
from modules.focus_local.database import persona
from modules.focus_local.source_data import SourceData, warehouse_page, wire


class Paging(unittest.TestCase):
    def setUp(self):
        self.contract = load_registry()['1.2']
        self.release = {'release_id': '00000000-0000-0000-0000-000000000001', 'registry_digest': self.contract.sha256,
                        'manifest': {'registry': self.contract.entry, 'table': self.contract.table, 'row_count': 101}}

    def test_boundaries_and_parameterized_tenant_reader(self):
        for after, count, next_after in [(0, 50, '50'), (50, 50, '100'), (100, 1, None)]:
            ch = Mock()
            ch.query.side_effect = [Mock(result_rows=[(101,101,1,101)]), Mock(result_rows=[(i,)+tuple(None for f in self.contract.fields if f['name'] in __import__('modules.focus_local.source_data',fromlist=['FIELDS']).FIELDS) for i in range(after+1,min(101,after+51)+1)])]
            with patch('modules.focus_local.source_data.client', return_value=ch) as client:
                rows, cursor = warehouse_page({},persona('approver-a'),self.release,after)
            self.assertEqual(len(rows),count)
            self.assertEqual(cursor,next_after)
            self.assertEqual(rows[0]['ordinal'],str(after+1))
            self.assertEqual(client.call_args.args[1],'hb_local_reader_a')
            self.assertEqual(ch.query.call_args.kwargs['parameters']['limit'],51)
            self.assertIn('ordinal>{after:UInt64}',ch.query.call_args.args[0])
            self.assertTrue(ch.close.called)

    def test_duplicate_missing_and_out_of_range_ordinals_rejected(self):
        for stats in [(101,100,1,101),(100,100,1,101),(101,101,0,100)]:
            ch=Mock(); ch.query.return_value.result_rows=[stats]
            with patch('modules.focus_local.source_data.client',return_value=ch), self.assertRaisesRegex(Rejected,'storage_shape'):
                warehouse_page({},persona('approver-a'),self.release,0)
            self.assertEqual(ch.query.call_count,1)

    def test_exact_amounts_and_null(self):
        self.assertEqual(wire(Decimal('999999999999.999999')),'999999999999.999999')
        self.assertEqual(wire(Decimal('-0.000001')),'-0.000001')
        self.assertEqual(wire(Decimal('0')),'0.000000')
        self.assertIsNone(wire(None))


class Integration(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if os.environ.get('HB_CONSOLE_INTEGRATION') != 'isolated':
            raise unittest.SkipTest('Requires isolated synthetic fixtures')
        cls.service=SourceData('approver-a')
        cls.c=cls.service.candidates()['candidates'][0]

    @classmethod
    def tearDownClass(cls):
        cls.service.close()

    def test_actual_rows_and_read_only(self):
        s=self.service; c=self.c
        def counts():
            with s.authority.inspection():
                return [s.authority.conn.execute('SELECT count(*) AS n FROM hb_local.'+t).fetchone()['n'] for t in ('release','confirmation','publication_decision','outbox')]
        before=counts()
        p=s.page(c['release_id'],c['manifest_digest'],0)
        self.assertEqual(p['classification'],'local-source-inspection')
        self.assertEqual(p['rows'][0]['ordinal'],'1')
        self.assertIsInstance(p['rows'][0]['values']['billed_cost'],str)
        self.assertNotIn('tags',p['rows'][0]['values'])
        self.assertNotIn('original_key',str(p))
        self.assertEqual(counts(),before)

    def test_foreign_scope_and_loader_denied(self):
        for actor in ('approver-b','loader-a'):
            s=SourceData(actor)
            try:
                with self.assertRaises(Rejected): s.page(self.c['release_id'],self.c['manifest_digest'],0)
            finally:s.close()

    def test_stale_invalid_and_corrupt_snapshot(self):
        s=self.service;c=self.c
        with self.assertRaisesRegex(Rejected,'evidence_changed'):s.page(c['release_id'],'0'*64,0)
        for after in (-1,10000,True):
            with self.assertRaisesRegex(Rejected,'request_invalid'):s.page(c['release_id'],c['manifest_digest'],after)
        with s.authority.inspection(): original=s.authority.release(c['release_id'])
        bad=copy.deepcopy(original);bad['manifest']['row_count']+=1
        with patch.object(s.authority,'release',return_value=bad),self.assertRaisesRegex(Rejected,'manifest_digest'):
            s.page(c['release_id'],c['manifest_digest'],0)
        bad=copy.deepcopy(original);bad['state']='pending'
        with patch.object(s.authority,'release',return_value=bad),self.assertRaisesRegex(Rejected,'input_unavailable'):
            s.page(c['release_id'],c['manifest_digest'],0)
        bad=copy.deepcopy(original);bad['manifest']['workspace_id']=str(persona('approver-b')[1]);bad['manifest_digest']=digest(canonical(bad['manifest']).encode())
        with patch.object(s.authority,'release',return_value=bad),self.assertRaisesRegex(Rejected,'manifest_scope'):
            s.page(c['release_id'],bad['manifest_digest'],0)

    def test_http_role_origin_snapshot_and_method(self):
        from modules.focus_local.http_service import LocalServer
        build=Path(__file__).resolve().parents[2]/'apps/console/dist'
        c=self.c
        path='/api/local-source-data/pages/'+c['release_id']+'/'+c['manifest_digest']+'/0'
        for actor in ('approver-a','loader-a'):
            server=LocalServer(0,actor,build)
            thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
            try:
                for suffix, headers, data, expected in [
                    (path,{'X-HB-Local-Client':'1'},None,200 if actor=='approver-a' else 403),
                    (path,{},None,403),
                    (path,{'X-HB-Local-Client':'1','Origin':'http://foreign.invalid'},None,403),
                    (path,{'X-HB-Local-Client':'1','Origin':server.origin},b'{}',405 if actor=='approver-a' else 403),
                    (path.replace(c['manifest_digest'],'0'*64),{'X-HB-Local-Client':'1'},None,409 if actor=='approver-a' else 403),
                ]:
                    req=urllib.request.Request(server.origin+suffix,headers=headers,data=data)
                    try:
                        with urllib.request.urlopen(req) as response:
                            status=response.status;body=json.load(response)
                    except urllib.error.HTTPError as error:
                        status=error.code;body=json.load(error)
                    self.assertEqual(status,expected,body)
            finally:server.shutdown();server.server_close();thread.join()

    def test_capability_rechecked_after_page(self):
        s=self.service
        with s.authority.inspection() as cap: original=cap.copy()
        with patch.object(s.authority,'inspection_capability',side_effect=[original,Rejected('permission')]), self.assertRaisesRegex(Rejected,'permission'):
            s.page(self.c['release_id'],self.c['manifest_digest'],0)


if __name__=='__main__':unittest.main()
