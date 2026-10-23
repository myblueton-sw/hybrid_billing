import copy
import json
from fractions import Fraction
from uuid import UUID

import psycopg
import pytest
from psycopg.types.json import Jsonb

from modules.focus_import import authority,warehouse
from modules.focus_import.model import *
from modules.focus_import.service import Coordinator,initialize


def loaded(official):
    c=Coordinator('importer')
    try:return c.ingest(official,'1.0',1)
    finally:c.close()


def sealed(official):
    item=loaded(official);a=Coordinator('approver')
    try:a.confirm(item['batch_id'],item['manifest_digest']);a.publish(item['batch_id'])
    finally:a.close()
    return item


def query(actor,**kw):
    r=Coordinator(actor)
    try:return r.query(**kw)
    finally:r.close()


def test_real_official_storage_query_properties_isolation_and_replay(stack,official):
    env,config=stack
    empty=copy.deepcopy(config['tenants'][1]);empty.update(key='empty',actor='reader-empty',assignments=[])
    config['tenants'].append(empty)
    initialize(config);item=sealed(official)
    assert loaded(official)['duplicate']
    a=query('reader-demo-a',limit=100);b=query('reader-demo-b',limit=100);e=query('reader-empty')
    assert (a['row_count'],b['row_count'],e['row_count'])==(942,58,0)
    assert a['tenant']['timezone']=='Asia/Seoul' and b['tenant']['timezone']=='UTC'
    assert a['tenant']['retention_days']==180 and b['tenant']['retention_days']==365
    assert a['rows'][0]['billing_start_local'].endswith('+09:00')
    assert a['currency_conversion'] is False and a['retention_enforcement'] is False
    assert {r['provider'] for r in a['rows']}=={'AWS'}
    assert {r['provider'] for r in b['rows']}=={'Microsoft','Oracle'}
    assert 'original_key' not in canonical(a) and 'shared-originals' not in canonical(a)
    assert query('reader-demo-a',provider='Microsoft')['row_count']==0
    assert query('reader-demo-b',provider='Oracle')['row_count']==7
    for tenant,count in [('demo-a',942),('demo-b',58),('empty',0)]:
        role=authority.reader_role(tenant_id(tenant))
        with authority.connect(env,role) as conn:
            assert {r['tenant_id'] for r in conn.execute('SELECT tenant_id FROM hb_focus.tenant')}=={tenant_id(tenant)}
            assert sum(r['row_count'] for r in conn.execute('SELECT row_count FROM hb_focus.slice'))==count
            with pytest.raises(psycopg.errors.InsufficientPrivilege):conn.execute('SELECT * FROM hb_focus.batch')
            with pytest.raises(psycopg.errors.InsufficientPrivilege):conn.execute('DELETE FROM hb_focus.publication')
        ch=warehouse.client(env,role)
        try:
            assert ch.query('SELECT count() FROM hb_focus.fact').result_rows[0][0]==count
            assert ch.query('SELECT count() FROM hb_focus.fact WHERE tenant_id={t:UUID}',parameters={'t':str(tenant_id('demo-b' if tenant=='demo-a' else 'demo-a'))}).result_rows[0][0]==0
            with pytest.raises(Exception):ch.command('TRUNCATE TABLE hb_focus.fact')
        finally:ch.close()
    ch=warehouse.client(env)
    try:assert ch.query('SELECT count() FROM hb_focus.fact').result_rows[0][0]==1000
    finally:ch.close()
    key=f"shared-originals/{item['batch_id']}/{SOURCE_SHA}.csv"
    assert warehouse.object_client(env).get_object(Bucket=warehouse.BUCKET,Key=key)['Body'].read()==official
    source=assign(decode(official,versions()['1.0']),config)
    for tenant,result in [('demo-a',a),('demo-b',b)]:
        expected=[r for r in source if r['tenant_id']==tenant_id(tenant)]
        for n in ADDITIVE:
            original=sum((Fraction(json.loads(r['source_json'])[versions()['1.0']['columns'][n]]) for r in expected if r[n] is not None),Fraction(0))
            observed=sum(Fraction(g['measures'][n]['coefficient'] or 0,10**18) for g in result['groups'].values())
            assert original==observed
    assert any(json.loads(r['missing_reasons']).get('contracted_cost')=='literal:NULL' and r['contracted_cost'] is None for r in b['rows'])


def test_permissions_confirmation_and_revision_conflict(stack,official):
    initialize(stack[1]);item=loaded(official)
    reader=Coordinator('reader-demo-a');loader=Coordinator('importer');approver=Coordinator('approver')
    try:
        with pytest.raises(Rejected,match='permission'):reader.ingest(official,'1.0',1)
        with pytest.raises(Rejected,match='permission'):loader.confirm(item['batch_id'],item['manifest_digest'])
        with pytest.raises(Rejected,match='confirmation_required'):approver.publish(item['batch_id'])
        with pytest.raises(Rejected,match='confirmation_digest'):approver.confirm(item['batch_id'],'0'*64)
        with pytest.raises(Rejected,match='revision_conflict'):loader.ingest(official+b'\n','1.0',1)
        with pytest.raises(Rejected,match='configured_version'):loader.ingest(official,'7.7',2)
        with pytest.raises(Rejected,match='operation_unresolved'):reader.query()
        approver.confirm(item['batch_id'],item['manifest_digest']);approver.publish(item['batch_id'])
        assert reader.query()['row_count']==942
    finally:reader.close();loader.close();approver.close()


def test_unmapped_and_currency_reject_before_storage(stack,official):
    env,c=stack
    c['tenants'][1]['assignments']=[]
    initialize(c);loader=Coordinator('importer')
    try:
        with pytest.raises(Rejected,match='unmapped'):loader.ingest(official,'1.0',1)
        with authority.connect(env) as conn:assert conn.execute('SELECT count(*) AS n FROM hb_focus.batch').fetchone()['n']==0
    finally:loader.close()


def test_unknown_partial_insert_blocks_reuse_and_reads(stack,official,monkeypatch):
    env,c=stack;initialize(c);loader=Coordinator('importer')
    original=loader.store.insert
    def partial(batch,rows):original(batch,rows[:10]);raise RuntimeError('injected partial effect')
    monkeypatch.setattr(loader.store,'insert',partial)
    try:
        with pytest.raises(RuntimeError):loader.ingest(official,'1.0',1)
        with authority.connect(env) as conn:assert conn.execute('SELECT state FROM hb_focus.batch').fetchone()['state']=='unknown'
        with pytest.raises(Rejected,match='operation_unresolved'):loader.ingest(official,'1.0',1)
        with pytest.raises(Rejected,match='operation_unresolved'):query('reader-demo-a')
    finally:loader.close()


def test_atomic_publication_rollback(stack,official):
    env,c=stack;initialize(c);item=loaded(official);approver=Coordinator('approver')
    try:
        approver.confirm(item['batch_id'],item['manifest_digest'])
        with authority.connect(env) as conn:
            conn.execute("CREATE FUNCTION hb_focus.test_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected'; END $$")
            conn.execute('CREATE TRIGGER test_fail BEFORE INSERT ON hb_focus.decision FOR EACH ROW EXECUTE FUNCTION hb_focus.test_fail()')
        with pytest.raises(psycopg.Error):approver.publish(item['batch_id'])
        with authority.connect(env) as conn:
            assert conn.execute('SELECT count(*) AS n FROM hb_focus.publication').fetchone()['n']==0
            assert conn.execute('SELECT state FROM hb_focus.batch').fetchone()['state']=='verified'
            assert conn.execute('SELECT busy FROM hb_focus.gate').fetchone()['busy']
            conn.execute('DROP TRIGGER test_fail ON hb_focus.decision')
        approver.publish(item['batch_id'])
        assert query('reader-demo-b')['row_count']==58
    finally:approver.close()


def test_policy_and_mapping_change_deny_effects(stack,official,monkeypatch):
    env,c=stack;initialize(c);item=loaded(official);approver=Coordinator('approver')
    try:
        import modules.focus_import.service as service
        existing=service.versions
        def changed():
            v=copy.deepcopy(existing());v['1.0']['null_tokens'].append('changed');return v
        monkeypatch.setattr(service,'versions',changed)
        with pytest.raises(Rejected,match='profile_changed'):approver.confirm(item['batch_id'],item['manifest_digest'])
        monkeypatch.setattr(service,'versions',existing)
        with authority.connect(env) as conn:conn.execute('UPDATE hb_focus.tenant SET active=false WHERE tenant_id=%s',(tenant_id('demo-b'),))
        with pytest.raises(Rejected,match='tenant_policy'):approver.confirm(item['batch_id'],item['manifest_digest'])
        with pytest.raises(Rejected,match='tenant_inactive'):query('reader-demo-b')
    finally:approver.close()


def test_reinitialization_checks_configuration_and_privileges(stack,official):
    env,c=stack;initialize(c)
    altered=copy.deepcopy(c);altered['tenants'][0]['timezone']='UTC'
    with pytest.raises(Rejected,match='configuration_changed'):initialize(altered)
    with authority.connect(env) as conn:conn.execute('GRANT UPDATE(active) ON hb_focus.tenant TO hb_focus_ingest')
    with pytest.raises(Rejected,match='incompatible_pg_privileges'):initialize(c)


def test_new_revision_preserves_rows_and_current_pointer(stack,official):
    env,c=stack;initialize(c);first=sealed(official)
    loader=Coordinator('importer');a=Coordinator('approver')
    try:
        second=loader.ingest(official,'1.0',2)
        a.confirm(second['batch_id'],second['manifest_digest']);a.publish(second['batch_id'])
        assert query('reader-demo-a')['batch_id']==second['batch_id']
        assert a.publish(first['batch_id'])['duplicate']
        assert query('reader-demo-a')['batch_id']==second['batch_id']
        with pytest.raises(Rejected,match='source_revision'):loader.ingest(official,'1.0',0)
    finally:loader.close();a.close()


def test_tampered_binding_is_not_authority(stack,official):
    initialize(stack[1]);sealed(official)
    original=BINDINGS.read_bytes()
    try:
        bindings=json.loads(original);bindings['actors']['reader-demo-a']=str(tenant_id('demo-b'))
        BINDINGS.write_text(json.dumps(bindings))
        with pytest.raises(Rejected,match='actor_binding'):query('reader-demo-a')
    finally:BINDINGS.write_bytes(original)


def test_policy_withdrawal_serializes_with_query(stack,official,monkeypatch):
    from concurrent.futures import ThreadPoolExecutor
    from threading import Event
    import time
    env,c=stack;initialize(c);sealed(official)
    entered=Event();release=Event();updating=Event()
    real=warehouse.Warehouse.verify
    def held(self,*args,**kwargs):
        if kwargs.get('reader'):
            entered.set();assert release.wait(10)
        return real(self,*args,**kwargs)
    monkeypatch.setattr(warehouse.Warehouse,'verify',held)
    def deactivate():
        with authority.connect(env) as conn:
            conn.execute("SET application_name='hb27-policy-withdrawal'")
            updating.set()
            conn.execute('UPDATE hb_focus.tenant SET active=false WHERE tenant_id=%s',(tenant_id('demo-a'),))
    with ThreadPoolExecutor(max_workers=2) as pool:
        future=pool.submit(query,'reader-demo-a');assert entered.wait(10)
        change=pool.submit(deactivate);assert updating.wait(10)
        try:
            deadline=time.monotonic()+5;blocked=False
            with authority.connect(env) as conn:
                while time.monotonic()<deadline:
                    if conn.execute("SELECT 1 FROM pg_stat_activity WHERE application_name='hb27-policy-withdrawal' AND wait_event_type='Lock'").fetchone():blocked=True;break
                    time.sleep(.02)
            assert blocked and not change.done()
        finally:release.set()
        assert future.result(timeout=10)['row_count']==942
        change.result(timeout=10)
    with pytest.raises(Rejected,match='tenant_inactive'):query('reader-demo-a')


def test_snapshot_subset_rejected_and_current_preserved(stack,official):
    env,c=stack;initialize(c);sealed(official);loader=Coordinator('importer')
    try:
        with pytest.raises(Rejected,match='snapshot_digest'):loader.ingest(official[:official.find(b'\n',official.find(b'\n')+1)+1],'1.0',2)
        assert query('reader-demo-a')['row_count']==942
        with authority.connect(env) as conn:assert conn.execute('SELECT count(*) AS n FROM hb_focus.batch').fetchone()['n']==1
    finally:loader.close()


def test_pagination_and_source_composite_references(stack,official):
    env,c=stack;initialize(c);first=sealed(official)
    page=query('reader-demo-a',limit=2)
    assert page['next_after'] is not None
    next_page=query('reader-demo-a',limit=2,after=page['next_after'],expected_batch=page['batch_id'])
    assert min(r['ordinal'] for r in next_page['rows'])>page['next_after']
    with pytest.raises(Rejected,match='query_batch_required'):query('reader-demo-a',after=2)
    with authority.connect(env) as conn:
        conn.execute("INSERT INTO hb_focus.gate(source) VALUES ('other-source')")
        with pytest.raises(psycopg.errors.ForeignKeyViolation):conn.execute("UPDATE hb_focus.gate SET current_batch=%s WHERE source='other-source'",(UUID(first['batch_id']),))
        with pytest.raises(psycopg.errors.ForeignKeyViolation):conn.execute("INSERT INTO hb_focus.publication VALUES (%s,'other-source',%s)",(tenant_id('demo-a'),UUID(first['batch_id'])))
    loader=Coordinator('importer');approver=Coordinator('approver')
    try:
        second=loader.ingest(official,'1.0',2)
        approver.confirm(second['batch_id'],second['manifest_digest']);approver.publish(second['batch_id'])
        with pytest.raises(Rejected,match='query_batch_changed'):query('reader-demo-a',after=2,expected_batch=first['batch_id'])
    finally:loader.close();approver.close()


def test_clickhouse_unexpected_privilege_rejected(stack):
    env,c=stack;initialize(c)
    ch=warehouse.client(env,'probe_admin')
    role=authority.reader_role(tenant_id('demo-a'))
    try:
        ch.command(f'GRANT INSERT ON hb_focus.fact TO {role}')
        with pytest.raises(Rejected,match='incompatible_ch_privileges'):initialize(c)
    finally:ch.command(f'REVOKE INSERT ON hb_focus.fact FROM {role}');ch.close()
