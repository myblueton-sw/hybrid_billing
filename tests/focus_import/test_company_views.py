import copy
import json
from pathlib import Path
from fractions import Fraction

import psycopg
import pytest

from modules.focus_import import authority,warehouse
from modules.focus_import.model import *
from modules.focus_import.service import Coordinator,initialize


@pytest.fixture
def pack(stack):
    data=(ROOT/'.local/datasets/focus/six-provider.csv').read_bytes()
    config=json.loads((ROOT/'.local/datasets/focus/six-provider.config.json').read_text())
    validate_config(config)
    assert digest(data)==config['snapshot']['file_digest']
    return stack[0],config,data


def publish(data):
    loader=Coordinator('importer');approver=Coordinator('approver')
    try:
        item=loader.ingest(data,'1.0',1);approver.confirm(item['batch_id'],item['manifest_digest']);approver.publish(item['batch_id']);return item
    finally:loader.close();approver.close()


def read(actor,**kw):
    r=Coordinator(actor)
    try:return r.query(**kw)
    finally:r.close()


def test_all_six_and_authorized_company_union(pack):
    env,c,data=pack;initialize(c);item=publish(data)
    results={actor:read(actor,limit=100) for actor in ['reader-north-a','reader-north-b','reader-foreign','reader-company-north']}
    assert {k:v['row_count'] for k,v in results.items()}=={'reader-north-a':1112,'reader-north-b':9,'reader-foreign':2,'reader-company-north':1121}
    combined=results['reader-company-north']
    assert set(combined['tenant_breakdown'])=={'north-a','north-b'}
    assert {json.loads(k)[0] for k in combined['groups']}=={'AWS','Microsoft','Google Cloud','Oracle','VMware','Alibaba Cloud'}
    assert {json.loads(k)[2] for k in combined['groups']}=={'USD','CNY'}
    assert read('reader-company-north',provider='GCP')['row_count']==115
    assert read('reader-company-north',provider='Azure')['row_count']==51
    assert read('reader-company-north',provider='OCI')['row_count']==7
    assert read('reader-company-north',provider='VMware')['row_count']==3
    assert read('reader-company-north',provider='Alibaba')['row_count']==3
    assert read('reader-north-a',provider='OCI')['row_count']==0
    role=authority.reader_role(view_id('reader-company-north'))
    with authority.connect(env,role) as conn:
        assert {r['tenant_id'] for r in conn.execute('SELECT tenant_id FROM hb_focus.tenant')}=={tenant_id('north-a'),tenant_id('north-b')}
        assert sum(r['row_count'] for r in conn.execute('SELECT row_count FROM hb_focus.slice'))==1121
        assert conn.execute('SELECT count(*) AS n FROM hb_focus.audience').fetchone()['n']==1
        with pytest.raises(psycopg.errors.InsufficientPrivilege):conn.execute('INSERT INTO hb_focus.audience_member VALUES (%s,%s)',('reader-company-north',tenant_id('foreign')))
        with pytest.raises(psycopg.errors.InsufficientPrivilege):conn.execute('SELECT * FROM hb_focus.batch')
    ch=warehouse.client(env,role)
    try:
        assert ch.query('SELECT count() FROM hb_focus.fact').result_rows[0][0]==1121
        assert ch.query('SELECT count() FROM hb_focus.fact WHERE tenant_id={t:UUID}',parameters={'t':str(tenant_id('foreign'))}).result_rows[0][0]==0
        with pytest.raises(Exception):ch.command('TRUNCATE TABLE hb_focus.fact')
    finally:ch.close()
    rows=[];page=combined
    while True:
        rows.extend(page['rows'])
        if page['next_after'] is None:break
        page=read('reader-company-north',limit=100,after=page['next_after'],expected_batch=page['batch_id'],expected_hierarchy=page['hierarchy_snapshot'])
    assert len(rows)==len({r['ordinal'] for r in rows})==1121
    assert {r['tenant_key'] for r in rows}=={'north-a','north-b'}
    for key,group in combined['groups'].items():
        provider,service,currency,consumed,pricing=json.loads(key)
        selected=[r for r in rows if [r[n] for n in ['provider','service','currency','consumed_unit','pricing_unit']]==[provider,service,currency,consumed,pricing]]
        for n in ADDITIVE:
            lexical=versions()['1.0']['columns'][n]
            expected=sum((Fraction(json.loads(r['source_json'])[lexical]) for r in selected if r[n] is not None),Fraction(0))
            measure=group['measures'][n]
            assert expected==Fraction(measure['coefficient'] or 0,10**18)
            assert measure['missing']==sum(r[n] is None for r in selected)
    observed=assign(decode(data,versions()['1.0']),c)
    assert len(observed)==1123
    assert sum(json.loads(r['source_json'])['x_HB_EvidenceClass']=='synthetic-contract-fixture' for r in observed)==8
    assert sum(r['provider']=='Google Cloud' for r in observed)==115


@pytest.mark.parametrize('change',['foreign','unknown','duplicate','actor_collision'])
def test_invalid_view_configuration_rejected(change):
    c=json.loads((ROOT/'.local/datasets/focus/six-provider.config.json').read_text())
    if change=='foreign':c['views'][0]['tenant_keys'].append('foreign')
    if change=='unknown':c['views'][0]['tenant_keys'].append('missing')
    if change=='duplicate':c['views'][0]['tenant_keys'].append('north-a')
    if change=='actor_collision':c['views'][0]['actor']='reader-north-a'
    with pytest.raises(Rejected):validate_config(c)


def test_view_membership_and_actor_tamper_deny(pack):
    env,c,data=pack;initialize(c);publish(data)
    original=BINDINGS.read_bytes()
    try:
        bindings=json.loads(original);bindings['views'].append('reader-north-a');BINDINGS.write_text(json.dumps(bindings))
        with pytest.raises(Exception):read('reader-north-a')
    finally:BINDINGS.write_bytes(original)
    with authority.connect(env) as conn:conn.execute('DELETE FROM hb_focus.audience_member WHERE actor=%s AND tenant_id=%s',('reader-company-north',tenant_id('north-b')))
    with pytest.raises(Rejected,match='view_members_changed'):read('reader-company-north')
    with pytest.raises(Rejected,match='view_members_changed'):initialize(c)
    assert read('reader-north-a')['row_count']==1112


def test_whole_view_denied_when_member_inactive(pack):
    env,c,data=pack;initialize(c);publish(data)
    with authority.connect(env) as conn:conn.execute('UPDATE hb_focus.tenant SET active=false WHERE tenant_id=%s',(tenant_id('north-b'),))
    with pytest.raises(Rejected,match='view_member_denied'):read('reader-company-north')
    assert read('reader-north-a')['row_count']==1112


@pytest.mark.parametrize('operation',['insert','delete','update'])
def test_membership_insert_serializes_through_parent(pack,monkeypatch,operation):
    from concurrent.futures import ThreadPoolExecutor
    from threading import Event
    import time
    env,c,data=pack;initialize(c);publish(data)
    entered=Event();release=Event();updating=Event();original=warehouse.Warehouse.verify
    def held(self,*args,**kwargs):
        if isinstance(kwargs.get('reader'),str):entered.set();assert release.wait(10)
        return original(self,*args,**kwargs)
    monkeypatch.setattr(warehouse.Warehouse,'verify',held)
    def add_foreign():
        with authority.connect(env) as conn:
            conn.execute("SET application_name='hb27-member-insert'");updating.set()
            if operation=='insert':conn.execute('INSERT INTO hb_focus.audience_member VALUES (%s,%s)',('reader-company-north',tenant_id('foreign')))
            elif operation=='delete':conn.execute('DELETE FROM hb_focus.audience_member WHERE actor=%s AND tenant_id=%s',('reader-company-north',tenant_id('north-b')))
            else:conn.execute('UPDATE hb_focus.audience_member SET tenant_id=%s WHERE actor=%s AND tenant_id=%s',(tenant_id('foreign'),'reader-company-north',tenant_id('north-b')))
    with ThreadPoolExecutor(max_workers=2) as pool:
        query=pool.submit(read,'reader-company-north');assert entered.wait(10)
        addition=pool.submit(add_foreign);assert updating.wait(10)
        try:
            deadline=time.monotonic()+5;blocked=False
            with authority.connect(env) as conn:
                while time.monotonic()<deadline:
                    if conn.execute("SELECT 1 FROM pg_stat_activity WHERE application_name='hb27-member-insert' AND wait_event_type='Lock'").fetchone():blocked=True;break
                    time.sleep(.02)
            assert blocked and not addition.done()
        finally:release.set()
        assert query.result(timeout=10)['row_count']==1121
        addition.result(timeout=10)
    with pytest.raises(Rejected,match='view_members_changed'):read('reader-company-north')


def test_audience_revocation_and_role_denials(pack):
    env,c,data=pack;initialize(c);publish(data)
    reader=Coordinator('reader-company-north')
    try:
        with pytest.raises(Rejected,match='permission'):reader.ingest(data,'1.0',2)
        with pytest.raises(psycopg.errors.InsufficientPrivilege):reader.a.conn.execute('SET ROLE hb_focus_ingest')
        with authority.connect(env) as conn:conn.execute('UPDATE hb_focus.audience SET active=false WHERE actor=%s',('reader-company-north',))
        with pytest.raises(Rejected,match='view_unavailable'):reader.query()
        assert read('reader-north-a')['row_count']==1112
    finally:reader.close()


def test_mixed_publication_batches_deny_whole_view(pack):
    env,c,data=pack;initialize(c);first=publish(data)
    loader=Coordinator('importer');approver=Coordinator('approver')
    try:
        second=loader.ingest(data,'1.0',2);approver.confirm(second['batch_id'],second['manifest_digest']);approver.publish(second['batch_id'])
        with authority.connect(env) as conn:conn.execute('UPDATE hb_focus.publication SET batch_id=%s WHERE tenant_id=%s',(UUID(first['batch_id']),tenant_id('north-b')))
        with pytest.raises(Rejected,match='view_publication_mismatch'):read('reader-company-north')
    finally:loader.close();approver.close()
