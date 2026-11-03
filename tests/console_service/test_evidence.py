import copy
import csv
import io
import json
import os
import sys
import threading
import unittest
import urllib.request
import urllib.error
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0,str(Path(__file__).resolve().parents[2]))
from modules.focus_local.evidence import Evidence, source_fields
from modules.focus_local.contract import MAX_BYTES, Rejected, canonical, controls, digest, load_registry, parse, rows_digest
from modules.focus_local.http_service import LocalServer
from modules.focus_local.storage import pinned_contract


class Projection(unittest.TestCase):
    def test_logical_record_source_absence_empty_null_and_inert_text(self):
        contract=load_registry()['1.3']
        raw=b'BillingAccountId,ServiceName,ResourceId,BilledCost,EffectiveCost,PrivateExtra\n"line\none","<img src=https://invalid.test/x>",=1+1,0.000000,\\N,excluded\nsecond,,plain,1e-6,,excluded\n'
        normalized={f['name']:None for f in contract.fields}
        first={f['name']:f for f in source_fields(raw,contract,normalized,1)}
        second={f['name']:f for f in source_fields(raw,contract,normalized,2)}
        self.assertEqual(first['billing_account_id']['source_text'],'line\none')
        self.assertEqual(second['billing_account_id']['source_text'],'second')
        self.assertEqual(first['effective_cost']['source_text'],'\\N')
        self.assertEqual(second['service_name']['source_text'],'')
        self.assertFalse(first['pricing_unit']['source_present'])
        self.assertIsNone(first['pricing_unit']['source_text'])
        self.assertEqual(first['resource_id']['source_text'],'=1+1')
        self.assertEqual(second['billed_cost']['source_text'],'1e-6')
        self.assertNotIn('excluded',json.dumps(first))
        large=b'BillingAccountId\n'+b'x\n'*9999+b'last\n'
        self.assertEqual(source_fields(large,contract,normalized,10000)[0]['source_text'],'last')


class Integration(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if os.environ.get('HB_CONSOLE_INTEGRATION')!='isolated':raise unittest.SkipTest('Requires isolated synthetic fixtures')
        cls.s=Evidence('approver-a');cls.c=cls.s.candidates()['candidates'][0]
        with cls.s.authority.inspection() as cap:
            cls.cap=cap.copy();cls.release=cls.s.authority.release(cls.c['release_id'])
        cls.raw=cls.s.store.bounded_original(cls.release['original_key'],MAX_BYTES)

    @classmethod
    def tearDownClass(cls):cls.s.close()

    def inspect(self,ordinal=1,snapshot=None):
        return self.s.inspect_row(self.c['release_id'],snapshot or self.c['manifest_digest'],ordinal)

    def test_actual_s3_source_and_no_writes_or_paths(self):
        def counts():
            with self.s.authority.inspection():
                return [self.s.authority.conn.execute('SELECT count(*) AS n FROM hb_local.'+t).fetchone()['n'] for t in ('release','confirmation','publication_decision','outbox')]
        before=counts();r=self.inspect()
        self.assertEqual(r['original_digest'],digest(self.raw));self.assertEqual(r['original_bytes'],len(self.raw))
        self.assertEqual(r['warehouse_integrity'],'not-revalidated');self.assertEqual(r['retention'],'unknown')
        self.assertNotIn('original_key',json.dumps(r));self.assertNotIn('/originals/',json.dumps(r));self.assertEqual(counts(),before)
        field=next(f for f in r['fields'] if f['name']=='billed_cost');self.assertIsInstance(field['normalized'],str)

    def test_stale_ordinal_and_foreign_scope(self):
        with self.assertRaisesRegex(Rejected,'evidence_changed'):self.inspect(snapshot='0'*64)
        for n in (0,-1,10001,True,self.release['manifest']['row_count']+1):
            with self.assertRaises(Rejected):self.inspect(n)
        for actor in ('loader-a','approver-b'):
            s=Evidence(actor)
            try:
                with self.assertRaises(Rejected):s.inspect_row(self.c['release_id'],self.c['manifest_digest'],1)
            finally:s.close()

    def test_changed_original_key_hash_manifest_and_missing_object(self):
        with patch.object(self.s.store,'bounded_original',return_value=self.raw+b'changed'),self.assertRaisesRegex(Rejected,'original_digest'):self.inspect()
        bad=copy.deepcopy(self.release);bad['original_key']='untrusted/other.csv'
        with patch.object(self.s.authority,'release',return_value=bad),patch.object(self.s.store,'bounded_original') as read,self.assertRaisesRegex(Rejected,'original_reference'):
            self.inspect()
        read.assert_not_called()
        bad=copy.deepcopy(self.release);bad['manifest']['rows_digest']='0'*64;bad['manifest_digest']=digest(canonical(bad['manifest']).encode())
        with patch.object(self.s.authority,'release',return_value=bad),self.assertRaisesRegex(Rejected,'original_manifest'):self.inspect(snapshot=bad['manifest_digest'])
        with patch.object(self.s.store,'bounded_original',side_effect=Rejected('original_too_large')),self.assertRaisesRegex(Rejected,'original_too_large'):self.inspect()
        with patch.object(self.s.store,'bounded_original',side_effect=OSError('synthetic missing object')),self.assertRaises(OSError):self.inspect()

    def test_revocation_after_object_read(self):
        with patch.object(self.s.authority,'inspection_capability',side_effect=[self.cap,Rejected('permission')]),self.assertRaisesRegex(Rejected,'permission'):self.inspect()

    def test_verified_multiline_text_projection(self):
        reader=csv.DictReader(io.StringIO(self.raw.decode(),newline=''));rows=list(reader);headers=reader.fieldnames
        rows[0]['ServiceName']='<img src="https://invalid.test/track">\n=1+1'
        rows[0]['ResourceId']='=HYPERLINK("https://invalid.test")'
        for row in rows:row['UnclassifiedPrivate']='DO-NOT-EXPOSE'
        out=io.StringIO(newline='');writer=csv.DictWriter(out,fieldnames=headers+['UnclassifiedPrivate']);writer.writeheader();writer.writerows(rows);raw=out.getvalue().encode()
        bad=copy.deepcopy(self.release);m=bad['manifest'];bad['file_digest']=digest(raw);m['file_digest']=digest(raw);m['file_bytes']=len(raw)
        key=f'{self.s.authority.scope[0]}/{self.s.authority.scope[1]}/originals/{bad["release_id"]}/{digest(raw)}.csv';bad['original_key']=key;m['original_key']=key
        parsed=parse(raw,pinned_contract(bad),{'account':self.cap['account'],'subaccount':self.cap['subaccount'],'start':self.cap['period_start'],'end':self.cap['period_end']})
        m['rows_digest']=rows_digest(parsed);m['controls']=controls(parsed,pinned_contract(bad).fields);bad['manifest_digest']=digest(canonical(m).encode())
        with patch.object(self.s.authority,'release',return_value=bad),patch.object(self.s.store,'bounded_original',return_value=raw):
            result=self.inspect(snapshot=bad['manifest_digest'])
        self.assertNotIn('DO-NOT-EXPOSE',json.dumps(result));self.assertEqual(next(f for f in result['fields'] if f['name']=='service_name')['source_text'],rows[0]['ServiceName'])

    def test_http_access_method_and_error_redaction(self):
        build=Path(__file__).resolve().parents[2]/'apps/console/dist'
        for actor in ('approver-a','loader-a'):
            server=LocalServer(0,actor,build);thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
            path='/api/local-evidence/rows/'+self.c['release_id']+'/'+self.c['manifest_digest']+'/1'
            try:
                for suffix,headers,data,expected in [(path,{'X-HB-Local-Client':'1'},None,200 if actor=='approver-a' else 403),(path,{},None,403),(path,{'X-HB-Local-Client':'1','Origin':server.origin},b'{}',405 if actor=='approver-a' else 403),(path[:-1]+'0',{'X-HB-Local-Client':'1'},None,400 if actor=='approver-a' else 403)]:
                    try:
                        with urllib.request.urlopen(urllib.request.Request(server.origin+suffix,headers=headers,data=data)) as response:status=response.status;body=json.load(response)
                    except urllib.error.HTTPError as error:status=error.code;body=json.load(error)
                    self.assertEqual(status,expected,body);self.assertNotIn('/originals/',json.dumps(body))
            finally:server.shutdown();server.server_close();thread.join()

if __name__=='__main__':unittest.main()
