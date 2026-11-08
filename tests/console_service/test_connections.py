import os
import json
import sys
import threading
import unittest
import urllib.request
import urllib.error
from pathlib import Path
from unittest.mock import patch, Mock

import psycopg

sys.path.insert(0,str(Path(__file__).resolve().parents[2]))
from modules.focus_local.connections import Connections
from modules.focus_local.database import Authority, persona
from modules.focus_local.contract import Rejected
from modules.focus_local.http_service import LocalServer


class Permission(unittest.TestCase):
    def test_missing_right_inactive_and_reader_rejected(self):
        for actor,principal,cap in [('loader-a',{'rights':['confirm']},{'active':True}),('approver-a',{'rights':['ingest']},{'active':True}),('loader-a',{'rights':['ingest']},{'active':False}),('reader-a',{'rights':['query']},{'active':True})]:
            a=object.__new__(Authority);a.actor=actor;a.scope=persona(actor);a.conn=Mock()
            a.conn.execute.return_value.fetchone.side_effect=[principal,cap]
            with self.assertRaises(Rejected):a.metadata_capability()


class Integration(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if os.environ.get('HB_CONSOLE_INTEGRATION')!='isolated':raise unittest.SkipTest('Requires existing isolated fixtures')

    def test_actual_metadata_role_actions_and_no_content(self):
        for actor,actions in [('loader-a',['imports']),('approver-a',['source-data','reconciliation'])]:
            s=Connections(actor)
            try:
                data=s.detail();self.assertEqual(data['actions'],actions)
                self.assertEqual(data['connection'],str(persona(actor)[2]));self.assertEqual(data['completeness'],'unknown')
                self.assertEqual([c['version'] for c in data['contracts']],['1.2','1.3'])
                self.assertTrue(data['holds']['quarantine'])
                self.assertIsNotNone(data['latest_candidate'])
                self.assertEqual(set(data['latest_candidate']),{'release_id','source_revision','state','created_at','manifest_digest','version'})
                for forbidden in ('billed_cost','effective_cost','row_count','original_key','password','secret','endpoint'):
                    self.assertNotIn(forbidden,json.dumps(data))
            finally:s.close()

    def test_exact_foreign_connection_and_reader_denied(self):
        for actor in ('loader-a','approver-a'):
            s=Connections(actor)
            try:
                self.assertEqual(s.detail(str(persona(actor)[2]))['connection'],str(persona(actor)[2]))
                for identity in (str(persona('loader-b')[2]),'invalid'):
                    with self.assertRaisesRegex(Rejected,'connection_unavailable'):s.detail(identity)
            finally:s.close()
        s=Connections('reader-a')
        try:
            with self.assertRaisesRegex(Rejected,'permission'):s.detail()
        finally:s.close()

    def test_read_only_recheck_scope_reset_and_unchanged_state(self):
        s=Connections('loader-a');a=s.authority
        try:
            with a.metadata_inspection() as cap:
                initial=cap.copy();before=a.conn.execute('SELECT count(*) AS n FROM hb_local.release').fetchone()['n']
            s.detail()
            with a.metadata_inspection():self.assertEqual(a.conn.execute('SELECT count(*) AS n FROM hb_local.release').fetchone()['n'],before)
            with self.assertRaises(psycopg.errors.ReadOnlySqlTransaction):
                with a.metadata_inspection():a.conn.execute('UPDATE hb_local.connection SET lock_token=lock_token WHERE connection_id=%s',(a.scope[2],))
            self.assertIn(a.conn.execute("SELECT current_setting('hb.workspace',true) AS value").fetchone()['value'],('',None))
            with patch.object(a,'metadata_capability',side_effect=[initial,Rejected('permission')]),self.assertRaisesRegex(Rejected,'permission'):s.detail()
            with patch.object(a,'metadata_capability',side_effect=[initial,{**initial,'policy_revision':initial['policy_revision']+1}]),self.assertRaisesRegex(Rejected,'policy_changed'):s.detail()
        finally:s.close()

    def test_existing_confirm_inspection_still_rejects_loader(self):
        a=Authority('loader-a')
        try:
            with self.assertRaisesRegex(Rejected,'permission'):
                with a.inspection():pass
        finally:a.close()

    def test_unresolved_operation_is_visible(self):
        s=Connections('loader-b')
        try:self.assertTrue(s.detail()['holds']['unresolved'])
        finally:s.close()

    def test_http_role_partition_origin_and_mutation(self):
        build=Path(__file__).resolve().parents[2]/'apps/console/dist'
        for actor in ('loader-a','approver-a'):
            server=LocalServer(0,actor,build);thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
            try:
                for suffix,headers,data,expected in [('',{'X-HB-Local-Client':'1'},None,200),('',{},None,403),('',{'X-HB-Local-Client':'1','Origin':server.origin},b'{}',405),('/'+str(persona('loader-b')[2]),{'X-HB-Local-Client':'1'},None,403)]:
                    try:
                        with urllib.request.urlopen(urllib.request.Request(server.origin+'/api/local-connections'+suffix,headers=headers,data=data)) as response:status=response.status;body=json.load(response)
                    except urllib.error.HTTPError as error:status=error.code;body=json.load(error)
                    self.assertEqual(status,expected,body)
            finally:server.shutdown();server.server_close();thread.join()

if __name__=='__main__':unittest.main()
