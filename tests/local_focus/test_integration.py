import json
import io
from contextlib import contextmanager
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from uuid import uuid4

import pytest
import pyarrow as pa
import pyarrow.parquet as pq

from modules.focus_local import database, storage
from modules.focus_local.contract import REGISTRY, Rejected, digest
from modules.focus_local.pipeline import Coordinator, initialize
from test_contract import fixture


def publish(loader,approver,version='1.3',revision=1,data=None):
    result=loader.ingest(data or fixture(version),version,revision)
    approver.confirm(result['release_id'],result['manifest_digest'])
    approver.publish(result['release_id'])
    return result


@pytest.mark.parametrize('version',['1.2','1.3'])
def test_real_end_to_end_and_role_boundaries(local_stack,version):
    loader=Coordinator('loader-a');approver=Coordinator('approver-a');reader=Coordinator('reader-a');other=Coordinator('reader-b')
    try:
        result=loader.ingest(fixture(version),version,1)
        with pytest.raises(Rejected,match='confirmation_required'):approver.publish(result['release_id'])
        with pytest.raises(Rejected,match='permission'):loader.confirm(result['release_id'],result['manifest_digest'])
        with pytest.raises(Rejected,match='confirmation_subject'):approver.confirm(result['release_id'],'0'*64)
        approver.confirm(result['release_id'],result['manifest_digest'])
        assert approver.publish(result['release_id'])['scope_revision']==1
        observed=reader.query()
        assert observed['row_count']==2 and observed['version']==version
        assert next(iter(observed['groups'].values()))['measures']['billed_cost']['coefficient']==200000
        assert other.query()['row_count']==0
        with loader.authority.transaction('ingest'):
            release=loader.authority.release(result['release_id'])
            contract=storage.pinned_contract(release)
        direct=storage.client(local_stack,'hb_local_reader_b')
        try:assert direct.query(f'SELECT count() FROM hb_local.{contract.table}').result_rows[0][0]==0
        finally:direct.close()
        with other.authority.transaction('query'):
            assert other.authority.conn.execute('SELECT count(*) AS n FROM hb_local.release').fetchone()['n']==0
        with loader.authority.transaction('ingest'):
            release=loader.authority.release(result['release_id'])
            assert loader.store.get(release['original_key'])==fixture(version)
        exported=approver.export(result['release_id'])
        assert digest(approver.store.get(exported['object_key']))==exported['sha256']
        assert approver.export(result['release_id'])['duplicate']
        assert loader.ingest(fixture(version),version,1)['duplicate']
        assert approver.publish(result['release_id'])['duplicate']
        with pytest.raises(Rejected,match='revision_conflict'):loader.ingest(fixture(version,BilledCost='9'),version,1)
        initialize()
    finally:
        for c in [loader,approver,reader,other]:c.close()


def test_quarantine_original_and_reject_scope(local_stack):
    loader=Coordinator('loader-a');approver=Coordinator('approver-a')
    try:
        data=fixture(BilledCost='')
        result=loader.ingest(data,'1.3',1)
        assert result['state']=='quarantined' and result['reason']=='required_value'
        with loader.authority.transaction('ingest'):
            row=loader.authority.conn.execute('SELECT * FROM hb_local.quarantine').fetchone()
            assert loader.store.get(row['object_key'])==data
            assert loader.authority.conn.execute('SELECT count(*) AS n FROM hb_local.release').fetchone()['n']==0
        assert loader.ingest(data,'1.3',1)['duplicate']
        assert loader.ingest(fixture(),'0.0',2)['reason']=='unsupported_version'
        with pytest.raises(Rejected,match='source_scope'):loader.ingest(fixture(BillingAccountId='other',BilledCost=''),'1.3',3)
        with pytest.raises(Rejected,match='quarantine_hold'):approver.confirm(result['receipt_id'],'0'*64)
    finally:loader.close();approver.close()


def test_pending_operation_survives_failure_and_session_loss(local_stack,monkeypatch):
    loader=Coordinator('loader-a')
    original=loader.store.insert
    def interrupted(*args):
        original(*args)
        raise Rejected('injected_after_insert')
    monkeypatch.setattr(loader.store,'insert',interrupted)
    with pytest.raises(Rejected,match='injected_after_insert'):loader.ingest(fixture(),'1.3',1)
    loader.close()
    replacement=Coordinator('loader-a');approver=Coordinator('approver-a')
    try:
        with pytest.raises(Rejected,match='operation_unresolved'):replacement.ingest(fixture(),'1.3',2)
        with pytest.raises(Rejected,match='operation_unresolved'):approver.publish(str(uuid4()))
    finally:replacement.close();approver.close()


def test_atomic_predecessor_race_and_new_revision(local_stack):
    loader=Coordinator('loader-a');approver=Coordinator('approver-a')
    try:
        first=loader.ingest(fixture(),'1.3',1)
        second=loader.ingest(fixture(BilledCost='2'),'1.3',2)
        for item in [first,second]:approver.confirm(item['release_id'],item['manifest_digest'])
        def attempt(item):
            worker=Coordinator('approver-a')
            try:return worker.publish(item['release_id'])
            except Rejected as exc:return str(exc)
            finally:worker.close()
        with ThreadPoolExecutor(max_workers=2) as pool:results=list(pool.map(attempt,[first,second]))
        assert sum(isinstance(r,dict) for r in results)==1
        assert 'predecessor_conflict' in results
        newer=publish(loader,approver,revision=3,data=fixture(BilledCost='4'))
        assert newer['state']=='verified'
        with loader.authority.transaction('ingest'):
            assert loader.authority.conn.execute('SELECT count(*) AS n FROM hb_local.publication_decision').fetchone()['n']==2
    finally:loader.close();approver.close()


def test_policy_change_and_registry_digest(local_stack,tmp_path):
    for path in REGISTRY.glob('*.json'):(tmp_path/path.name).write_bytes(path.read_bytes())
    loader=Coordinator('loader-a',registry_dir=tmp_path);approver=Coordinator('approver-a',registry_dir=tmp_path)
    try:
        item=loader.ingest(fixture(),'1.3',1)
        entry=json.loads((tmp_path/'1.3.json').read_text());entry['revision']=2
        (tmp_path/'1.3.json').write_text(json.dumps(entry))
        with pytest.raises(Rejected,match='registry_changed'):approver.confirm(item['release_id'],item['manifest_digest'])
        entry['revision']=1;(tmp_path/'1.3.json').write_text(json.dumps(entry))
        approver.confirm(item['release_id'],item['manifest_digest'])
        with database.connect(local_stack,admin=True) as conn:conn.execute('UPDATE hb_local.connection SET policy_revision=2 WHERE connection_id=%s',(loader.authority.scope[2],))
        with pytest.raises(Rejected):approver.publish(item['release_id'])
        with database.connect(local_stack,admin=True) as conn:conn.execute('UPDATE hb_local.connection SET active=false WHERE connection_id=%s',(loader.authority.scope[2],))
        reader=Coordinator('reader-a')
        try:
            with pytest.raises(Rejected,match='policy_unavailable'):reader.query()
        finally:reader.close()
    finally:loader.close();approver.close()


def test_dynamic_registry_real_load(local_stack,tmp_path):
    entry=json.loads((REGISTRY/'1.3.json').read_text());entry['version']='9.9'
    (tmp_path/'extra.json').write_text(json.dumps(entry))
    initialize(registry_dir=tmp_path)
    loader=Coordinator('loader-a',registry_dir=tmp_path);approver=Coordinator('approver-a',registry_dir=tmp_path);reader=Coordinator('reader-a',registry_dir=tmp_path)
    try:
        item=loader.ingest(fixture(),'9.9',1)
        approver.confirm(item['release_id'],item['manifest_digest']);approver.publish(item['release_id'])
        entry['revision']=2;(tmp_path/'extra.json').write_text(json.dumps(entry))
        assert reader.query()['version']=='9.9'
    finally:loader.close();approver.close();reader.close()


def test_database_and_clickhouse_security_reinitialization(local_stack):
    with database.connect(local_stack,admin=True) as conn:conn.execute('ALTER ROLE hb_local_app BYPASSRLS')
    with pytest.raises(Rejected,match='incompatible_pg_role'):initialize()
    with database.connect(local_stack,admin=True) as conn:conn.execute('ALTER ROLE hb_local_app NOBYPASSRLS')
    with database.connect(local_stack,admin=True) as conn:conn.execute('GRANT UPDATE(active) ON hb_local.connection TO hb_local_app')
    with pytest.raises(Rejected,match='incompatible_pg_privileges'):initialize()
    with database.connect(local_stack,admin=True) as conn:conn.execute('REVOKE UPDATE(active) ON hb_local.connection FROM hb_local_app')
    admin=storage.client(local_stack,'probe_admin')
    try:
        admin.command('ALTER SETTINGS PROFILE hb_local_locked SETTINGS input_format_csv_empty_as_default=0 CONST')
        with pytest.raises(Rejected,match='incompatible_ch_settings'):initialize()
        admin.command('ALTER SETTINGS PROFILE hb_local_locked SETTINGS ' + storage.LOCKED_SETTINGS)
        admin.command('GRANT ALTER ON hb_local.* TO hb_local_reader_a')
        with pytest.raises(Rejected,match='incompatible_ch_grants'):initialize()
        admin.command('REVOKE ALTER ON hb_local.* FROM hb_local_reader_a')
        initialize()
        reader=storage.client(local_stack,'hb_local_reader_a')
        try:
            for query in ['SELECT 1 SETTINGS input_format_csv_empty_as_default=0','CREATE TABLE hb_local.forbidden(x UInt8) ENGINE=Memory',"SELECT * FROM s3(hb_objects,filename='anything',format='CSV',structure='x UInt8')"]:
                with pytest.raises(Exception):reader.query(query)
        finally:reader.close()
    finally:admin.close()


def test_admission_scans_entire_file_before_quarantine(local_stack):
    loader=Coordinator('loader-a')
    try:
        data=fixture(BilledCost='invalid')
        left,right=data.rsplit(b'synthetic-account-a',1)
        mixed=left+b'other-account'+right
        for version in ['1.3','0.0']:
            with pytest.raises(Rejected,match='source_scope'):loader.ingest(mixed,version,1)
        with pytest.raises(Rejected,match='source_period'):loader.ingest(fixture(BilledCost='invalid',BillingPeriodEnd='2026-12-01T00:00:00Z'),'1.3',1)
        with pytest.raises(Rejected,match='source_admission'):loader.ingest(b'not a scoped CSV','0.0',1)
        with loader.authority.transaction('ingest'):
            assert loader.authority.conn.execute('SELECT count(*) AS n FROM hb_local.quarantine').fetchone()['n']==0
    finally:loader.close()


def test_unresolved_quarantine_blocks_disclosure_and_publication(local_stack):
    loader=Coordinator('loader-a');approver=Coordinator('approver-a');reader=Coordinator('reader-a')
    try:
        publish(loader,approver)
        assert loader.ingest(fixture(BilledCost=''),'1.3',2)['state']=='quarantined'
        candidate=loader.ingest(fixture(),'1.3',3)
        with pytest.raises(Rejected,match='quarantine_hold'):reader.query()
        with pytest.raises(Rejected,match='quarantine_hold'):approver.confirm(candidate['release_id'],candidate['manifest_digest'])
    finally:loader.close();approver.close();reader.close()


@pytest.mark.parametrize('version',['1.2','1.3'])
def test_nullable_quantities_survive_query_and_parquet(local_stack,version):
    data=fixture(version,all_rows=True,ChargeCategory='Tax',SkuPriceId='',ConsumedQuantity='',ConsumedUnit='',PricingQuantity='',PricingUnit='',ListUnitPrice='',ContractedUnitPrice='')
    loader=Coordinator('loader-a');approver=Coordinator('approver-a');reader=Coordinator('reader-a')
    try:
        item=publish(loader,approver,version=version,data=data)
        group=next(iter(reader.query()['groups'].values()))
        for name in ['consumed_quantity','pricing_quantity']:
            assert group['measures'][name]=={'coefficient':None,'missing':2,'present':0}
        exported=approver.export(item['release_id'])
        table=pq.read_table(io.BytesIO(approver.store.get(exported['object_key'])))
        for name in ['consumed_quantity','pricing_quantity','list_unit_price','contracted_unit_price']:
            assert table.schema.field(name).type==pa.decimal128(18,6)
            assert table.schema.field(name).nullable
            assert table[name].null_count==2
    finally:loader.close();approver.close();reader.close()


def test_quarantine_rechecks_policy_before_object_write(local_stack,monkeypatch):
    loader=Coordinator('loader-a')
    original=loader.authority.transaction
    changed=False
    writes=[]
    @contextmanager
    def transaction(right,allow_unfinished=False):
        nonlocal changed
        if allow_unfinished and not changed:
            with database.connect(local_stack,admin=True) as conn:
                conn.execute('UPDATE hb_local.connection SET policy_revision=2 WHERE connection_id=%s',(loader.authority.scope[2],))
            changed=True
        with original(right,allow_unfinished) as cap:yield cap
    monkeypatch.setattr(loader.authority,'transaction',transaction)
    monkeypatch.setattr(loader.store,'put',lambda *args:writes.append(args))
    try:
        with pytest.raises(Rejected,match='policy_changed'):loader.ingest(fixture(BilledCost=''),'1.3',1)
        assert writes==[]
        with original('ingest',allow_unfinished=True):
            assert loader.authority.conn.execute("SELECT state FROM hb_local.operation").fetchone()['state']=='unknown'
    finally:loader.close()
