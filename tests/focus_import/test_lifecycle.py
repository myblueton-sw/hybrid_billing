import copy
import json
from concurrent.futures import ThreadPoolExecutor
from fractions import Fraction
from threading import Event
from uuid import UUID

import pytest

from modules.focus_import import archive, authority, collection, control, lifecycle, warehouse
from modules.focus_import.model import Rejected, canonical, digest, domains_digest, decode, tenant_id, versions
from modules.focus_import.service import Coordinator, initialize
from modules.focus_import.temporal import window
from test_temporal import configuration, descriptor, encoded, policy, source_row


@pytest.fixture
def temporal_stack(stack,monkeypatch,tmp_path):
    monkeypatch.setattr(control,'DIRECTORY',tmp_path/'controls')
    cfg=configuration()
    for t in cfg['tenants']:t.update(company_id='company-one',company_label='Same display')
    cfg['views']=[{'actor':'reader-company','company_id':'company-one','tenant_keys':['north','south'],'active':True}]
    initialize(cfg)
    policies={t['key']:{**policy(),'basis':'window_end','hot_days':1,'total_days':10,'grace_days':1} for t in cfg['tenants']}
    return stack[0],cfg,policies


def activated(stack,grain='day'):
    env,cfg,policies=stack
    lifecycle.activate('operator',grain,policies)
    return env,cfg,policies


def delivery(data,grain='day',revision=1,start=None,end=None):
    d=descriptor(data,grain,revision)
    if start:
        d['window']=window(start,end,grain);d['complete_through']=end
    return d


def received(data,d):
    r=collection.register('operator',data,'1.0',d)
    item=collection.collect('importer',r['delivery_id'],data)
    collection.confirm('approver',r['delivery_id'],item['manifest_digest'])
    collection.publish('approver',r['delivery_id'])
    return r['delivery_id'],item


def measure(result,name='billed_cost'):
    return sum((Fraction(g['measures'][name]['coefficient'] or 0,10**18) for g in result['groups'].values()),Fraction(0))


def slice_id(item,key):return item['batch_id']+':'+str(tenant_id(key))


def test_two_days_correction_empty_snapshot_and_replay(temporal_stack):
    activated(temporal_stack)
    day1=encoded([source_row(cost='10'),source_row(provider='Microsoft',cost='20')])
    first,item=received(day1,delivery(day1))
    day2=encoded([source_row(cost='30',start='2024-09-02T00:00:00Z',end='2024-09-03T00:00:00Z')])
    received(day2,delivery(day2,start='2024-09-02T00:00:00Z',end='2024-09-03T00:00:00Z'))
    view=collection.query('reader-company',grain='day',limit=1)
    assert view['row_count']==3 and measure(view)==60
    assert measure(collection.query('reader-north'))==40
    next_page=collection.query('reader-company',grain='day',limit=1,cursor=view['next_cursor'],expected_snapshot=view['snapshot'])
    assert next_page['rows'][0]['window_id']!=view['rows'][0]['window_id'] or next_page['rows'][0]['ordinal']!=view['rows'][0]['ordinal']
    assert collection.collect('importer',first,day1)['duplicate']
    changed=encoded([source_row(cost='7'),source_row(provider='Microsoft',cost='20')])
    received(changed,delivery(changed,revision=2))
    assert measure(collection.query('reader-company'))==57
    with pytest.raises(Rejected,match='snapshot_changed'):collection.query('reader-company',grain='day',cursor=view['next_cursor'],expected_snapshot=view['snapshot'])
    with pytest.raises(Rejected,match='delivery_revision'):collection.register('operator',changed,'1.0',delivery(changed,revision=1))
    empty=encoded([]);received(empty,delivery(empty,revision=3))
    assert measure(collection.query('reader-company'))==30
    state=lifecycle.status('operator')
    assert len(state['slices'])==8
    assert state['slices'][slice_id(item,'north')]['state']=='hot'


def test_monthly_cumulative_replaces_and_granularity_rejects(temporal_stack):
    activated(temporal_stack,'month')
    a=encoded([source_row(cost='10')]);first,_=received(a,delivery(a,'month'))
    b=encoded([source_row(cost='25')]);received(b,delivery(b,'month',2))
    q=collection.query('reader-north');assert measure(q)==25 and q['row_count']==1
    assert len(lifecycle.status('operator')['windows'])==1
    with pytest.raises(Rejected,match='granularity'):collection.register('operator',a,'1.0',delivery(a))
    with pytest.raises(Rejected,match='permission'):collection.register('reader-north',a,'1.0',delivery(a,'month',3))
    assert collection.collect('importer',first,a)['duplicate']


def test_unknown_collect_survives_restore_and_legacy_entrypoints(temporal_stack,monkeypatch):
    activated(temporal_stack)
    data=encoded([source_row()]);d=delivery(data)
    receipt=collection.register('operator',data,'1.0',d)['delivery_id']
    original=warehouse.Warehouse.insert
    def failed(self,batch,rows):
        original(self,batch,rows)
        raise RuntimeError('synthetic lost acknowledgement')
    monkeypatch.setattr(warehouse.Warehouse,'insert',failed)
    with pytest.raises(RuntimeError):collection.collect('importer',receipt,data)
    with pytest.raises(Rejected):collection.query('reader-north')
    with pytest.raises(Rejected):lifecycle.restore('operator','old-backup')
    c=Coordinator('reader-north')
    try:
        with pytest.raises(Rejected):c.query()
    finally:c.close()


def test_archive_recall_controls_and_immutable_origin(temporal_stack):
    activated(temporal_stack)
    data=encoded([source_row(cost='10.000000000000000001'),source_row(provider='Microsoft',cost='20')]);_,item=received(data,delivery(data))
    identity=slice_id(item,'north');before=lifecycle.status('operator')['slices'][identity]
    with pytest.raises(Rejected,match='hot_retention'):lifecycle.archive_slice('operator',identity,at='2024-09-02T00:00:00Z')
    archived=lifecycle.archive_slice('operator',identity,at='2024-09-03T00:00:00Z')
    assert archived['row_count']==1
    assert collection.query('reader-north')['row_count']==1
    assert measure(collection.query('reader-company'))==Fraction('30.000000000000000001')
    assert lifecycle.recall('operator',identity)['row_count']==1
    after=lifecycle.status('operator')['slices'][identity]
    assert after['origin_received']==before['origin_received'] and after['origin_window_end']==before['origin_window_end']
    assert measure(collection.query('reader-north'))==Fraction('10.000000000000000001')


def test_corrupt_archive_denies_without_changing_catalog(temporal_stack):
    activated(temporal_stack)
    data=encoded([source_row()]);_,item=received(data,delivery(data));identity=slice_id(item,'north')
    lifecycle.archive_slice('operator',identity,at='2024-09-03T00:00:00Z')
    with lifecycle.access('operator','status') as (j,conn,store,tenants):
        key=j.state['slices'][identity]['archive']['key'];j.s3.put_object(Bucket=warehouse.BUCKET,Key=key,Body=b'broken')
    with pytest.raises(Rejected,match='archive_digest'):collection.query('reader-north')
    with pytest.raises(Rejected,match='archive_digest'):lifecycle.recall('operator',identity)
    assert lifecycle.status('operator')['slices'][identity]['state']=='archived'


def test_hold_dependency_preview_change_and_shared_original(temporal_stack):
    activated(temporal_stack)
    data=encoded([source_row(),source_row(provider='Microsoft',cost='20')]);_,item=received(data,delivery(data))
    a=slice_id(item,'north');b=slice_id(item,'south')
    lifecycle.restriction('operator',b,'legal-case','Synthetic hold')
    p=lifecycle.preview('operator',identities=[a,b]);assert len(p['targets'])==1 and p['held'][0]['reason']=='hold'
    with pytest.raises(Rejected,match='permission'):lifecycle.approve('operator',p['manifest_digest'])
    lifecycle.approve('lifecycle-approver',p['manifest_digest'])
    lifecycle.restriction('operator',a,'issued-proof','Synthetic dependency',kind='reference')
    with pytest.raises(Rejected,match='manifest_changed'):lifecycle.delete('operator',p['manifest_digest'])
    lifecycle.restriction('operator',a,'issued-proof','Synthetic release',kind='reference',release=True)
    p=lifecycle.preview('operator',identities=[a,b]);lifecycle.approve('lifecycle-approver',p['manifest_digest'])
    result=lifecycle.delete('operator',p['manifest_digest'])
    assert result['succeeded']==1 and result['held']==1 and result['stores'][0]['shared_original_remaining']
    assert measure(collection.query('reader-south'))==20
    with pytest.raises(Rejected,match='expired'):collection.query('reader-north')
    with pytest.raises(Rejected,match='expired'):collection.query('reader-company')
    with pytest.raises(Rejected,match='too_late'):lifecycle.restriction('operator',a,'late','Late hold')
    with pytest.raises(Rejected,match='permission'):lifecycle.delete('reader-north',p['manifest_digest'])


def test_backup_restore_old_checkpoint_replays_tombstone_and_revoke(temporal_stack):
    env,cfg,policies=activated(temporal_stack)
    data=encoded([source_row(),source_row(provider='Microsoft',cost='20')]);_,item=received(data,delivery(data));a=slice_id(item,'north');b=slice_id(item,'south')
    backup=lifecycle.backup('operator',at='2024-09-03T00:00:00Z')
    with authority.connect(env) as conn:old=control.checkpoint(conn)
    lifecycle.restriction('operator',b,'case','Post-backup hold')
    p=lifecycle.preview('operator',identities=[a]);lifecycle.approve('lifecycle-approver',p['manifest_digest']);removed=lifecycle.delete('operator',p['manifest_digest'])
    assert removed['stores'][0]['managed_backups']==[backup['backup_id']]
    lifecycle.revoke('operator','reader-north')
    ch=warehouse.client(env,'probe_admin')
    try:ch.command('TRUNCATE TABLE hb_focus.fact')
    finally:ch.close()
    with authority.connect(env) as conn:conn.execute('UPDATE hb_focus.control_checkpoint SET sequence=%s,digest=%s WHERE id=1',(old['sequence'],old['digest']))
    with pytest.raises(Rejected,match='control_checkpoint'):collection.query('reader-south')
    restored=lifecycle.restore('operator',backup['backup_id'],at='2024-09-04T00:00:00Z');assert restored['latest_controls_applied']
    assert measure(collection.query('reader-south'))==20
    with pytest.raises(Rejected):collection.query('reader-north')
    with pytest.raises(Rejected):collection.query('reader-company')
    state=lifecycle.status('operator')
    assert state['slices'][a]['state']=='deleted' and 'case' in state['slices'][b]['holds']
    assert lifecycle.prune_backups('operator',at='2024-09-21T00:00:00Z')['removed_backups']==[backup['backup_id']]
    with pytest.raises(Rejected,match='backup_unavailable'):lifecycle.restore('operator',backup['backup_id'])


def test_journal_missing_local_intent_and_mirror_mismatch_failclosed(temporal_stack):
    activated(temporal_stack)
    head=control.DIRECTORY/'head.json';data=head.read_bytes();head.unlink()
    with pytest.raises(Rejected,match='journal_missing'):collection.query('reader-north')
    head.write_bytes(data)
    (control.DIRECTORY/'999999999999.json').write_text('{}')
    with pytest.raises(Rejected,match='uncommitted_intent'):collection.query('reader-north')
    (control.DIRECTORY/'999999999999.json').unlink()
    with lifecycle.access('operator','status') as (j,conn,store,tenants):
        j.s3.put_object(Bucket=warehouse.BUCKET,Key=f"control/{j.head['installation']}/999999999999-extra.json",Body=b'{}')
    with pytest.raises(Rejected,match='mirror_head'):collection.query('reader-north')


def test_policy_change_requires_distinct_approval_and_floor_blocks_replay(temporal_stack):
    env,cfg,policies=activated(temporal_stack)
    updated={**policies['north'],'revision':2,'reject_before':'2025-01-01T00:00:00Z','total_days':20}
    proposed=lifecycle.policy_preview('operator','north',updated)
    with pytest.raises(Rejected,match='approval_required'):lifecycle.policy_apply('operator',proposed['manifest_digest'])
    lifecycle.approve('lifecycle-approver',proposed['manifest_digest']);lifecycle.policy_apply('operator',proposed['manifest_digest'])
    data=encoded([source_row()])
    with pytest.raises(Rejected,match='expired_hold'):collection.register('operator',data,'1.0',delivery(data))
    with pytest.raises(Rejected,match='permission'):
        c=Coordinator('reader-north')
        try:
            with c.a.transaction('delete'):pass
        finally:c.close()


def test_hold_serializes_before_delete_and_new_reference_blocks(temporal_stack):
    activated(temporal_stack)
    data=encoded([source_row()]);_,item=received(data,delivery(data));a=slice_id(item,'north')
    p=lifecycle.preview('operator',identities=[a]);lifecycle.approve('lifecycle-approver',p['manifest_digest'])
    ready=Event();release=Event()
    def held():
        with lifecycle.access('operator','hold') as (j,conn,store,tenants):
            j.state['slices'][a]['holds']['race']={'reason':'Synthetic hold','actor':'operator','at':collection.now()};j.save(j.state);ready.set();assert release.wait(10)
    with ThreadPoolExecutor(max_workers=2) as pool:
        hold=pool.submit(held);assert ready.wait(10)
        deletion=pool.submit(lifecycle.delete,'operator',p['manifest_digest'])
        assert not deletion.done();release.set();hold.result(timeout=10)
        with pytest.raises(Rejected,match='manifest_changed'):deletion.result(timeout=10)
    assert collection.query('reader-north')['row_count']==1


def test_restore_recovers_original_and_archived_bytes(temporal_stack):
    activated(temporal_stack)
    data=encoded([source_row(),source_row(provider='Microsoft',cost='20')]);_,item=received(data,delivery(data));a=slice_id(item,'north')
    lifecycle.archive_slice('operator',a,at='2024-09-03T00:00:00Z')
    b=lifecycle.backup('operator',at='2024-09-03T00:00:00Z')
    with lifecycle.access('operator','status') as (j,conn,store,tenants):
        raw=lifecycle.original_for(j,item['batch_id']);arc=j.state['slices'][a]['archive']['key']
        lifecycle.delete_key(j.s3,raw);lifecycle.delete_key(j.s3,arc)
    lifecycle.restore('operator',b['backup_id'],at='2024-09-04T00:00:00Z')
    assert measure(collection.query('reader-north'))==10
    with lifecycle.access('operator','status') as (j,conn,store,tenants):assert j.s3.get_object(Bucket=warehouse.BUCKET,Key=raw)['Body'].read()==data


def test_restore_rejects_independent_head_behind_observed_checkpoint(temporal_stack):
    env,cfg,policies=activated(temporal_stack)
    data=encoded([source_row()]);_,item=received(data,delivery(data));b=lifecycle.backup('operator',at='2024-09-03T00:00:00Z')
    with authority.connect(env) as conn:conn.execute("UPDATE hb_focus.control_checkpoint SET sequence=sequence+1 WHERE id=1")
    with pytest.raises(Rejected,match='control_rollback'):lifecycle.restore('operator',b['backup_id'])


def test_rearchive_cycle_and_backup_clock_do_not_reset(temporal_stack):
    activated(temporal_stack)
    data=encoded([source_row()]);_,item=received(data,delivery(data));a=slice_id(item,'north')
    lifecycle.archive_slice('operator',a,at='2024-09-03T00:00:00Z');lifecycle.recall('operator',a)
    assert lifecycle.archive_slice('operator',a,at='2024-09-04T00:00:00Z')['state']=='archived'
    first=lifecycle.backup('operator',at='2024-09-03T00:00:00Z');second=lifecycle.backup('operator',at='2024-09-05T00:00:00Z')
    assert first['expires_at']==second['expires_at']=='2024-09-20T00:00:00+00:00'


def test_legacy_initialize_denies_unknown_lifecycle_operation(temporal_stack):
    env,cfg,policies=activated(temporal_stack)
    with lifecycle.access('operator','status') as (j,conn,store,tenants):
        j.state['operation']={'kind':'synthetic-insert','target':'uncertain'};j.save(j.state)
    with pytest.raises(Rejected,match='operation_unresolved'):initialize(cfg)


def test_registered_identical_empty_windows_are_not_lost_by_scheduler(temporal_stack,tmp_path):
    from tools.focus_lifecycle import run_due
    activated(temporal_stack)
    data=encoded([])
    a=collection.register('operator',data,'1.0',delivery(data))['delivery_id']
    b=collection.register('operator',data,'1.0',delivery(data,start='2024-09-02T00:00:00Z',end='2024-09-03T00:00:00Z'))['delivery_id']
    (tmp_path/'empty.csv').write_bytes(data)
    result=run_due('importer',tmp_path)
    assert len(result['collected'])==1 and result['registered_waiting']==1
    first=result['collected'][0]
    assert run_due('importer',tmp_path)['awaiting_publication']
    collection.confirm('approver',first['delivery_id'],first['manifest_digest']);collection.publish('approver',first['delivery_id'])
    second=run_due('importer',tmp_path)['collected'][0]
    assert {first['delivery_id'],second['delivery_id']}=={a,b}


def test_retirement_compacts_shared_payload_and_permanently_denies_reinit(temporal_stack):
    env,cfg,policies=activated(temporal_stack)
    preview=lifecycle.retire_preview('operator');lifecycle.approve('lifecycle-approver',preview['manifest_digest'])
    result=lifecycle.retire('operator',preview['manifest_digest']);assert result['managed_payload_removed']
    assert lifecycle.status('operator')['state']=='retired'
    serialized=''.join(p.read_text() for p in control.DIRECTORY.glob('*.json'))
    assert 'synthetic-account' not in serialized and 'billed_cost' not in serialized
    with authority.connect(env) as conn:assert conn.execute('SELECT count(*) AS n FROM hb_focus.configuration').fetchone()['n']==0
    with pytest.raises(Rejected,match='installation_retired'):initialize(cfg)
    with pytest.raises(Rejected,match='installation_retired'):lifecycle.restore('operator','old')
    with pytest.raises(Rejected,match='installation_retired'):collection.query('reader-north')


def test_interrupted_retirement_keeps_independent_deny_anchor(temporal_stack,monkeypatch):
    activated(temporal_stack)
    p=lifecycle.retire_preview('operator');lifecycle.approve('lifecycle-approver',p['manifest_digest'])
    original=lifecycle.delete_key
    def interrupted(s3,key):
        if key.startswith('control/'):raise RuntimeError('synthetic retirement interruption')
        return original(s3,key)
    monkeypatch.setattr(lifecycle,'delete_key',interrupted)
    with pytest.raises(RuntimeError):lifecycle.retire('operator',p['manifest_digest'])
    with pytest.raises(Rejected,match='retirement_unresolved'):collection.query('reader-north')
    with pytest.raises(Rejected,match='retirement_unresolved'):lifecycle.restore('operator','old')


def test_retirement_rejects_uninventoried_temporary_payload(temporal_stack):
    activated(temporal_stack)
    (control.DIRECTORY/'failed-write.tmp').write_text('Synthetic private payload')
    with pytest.raises(Rejected,match='uninventoried_local_artifact'):lifecycle.retire_preview('operator')


def test_late_purge_failure_cannot_emit_terminal_receipt(temporal_stack,monkeypatch):
    env,cfg,policies=activated(temporal_stack)
    p=lifecycle.retire_preview('operator');lifecycle.approve('lifecycle-approver',p['manifest_digest'])
    def failed_commit(conn,store):
        with conn.transaction():
            conn.execute('TRUNCATE hb_focus.configuration CASCADE')
            raise RuntimeError('Synthetic late transaction failure')
    monkeypatch.setattr(lifecycle,'purge_source',failed_commit)
    with pytest.raises(RuntimeError):lifecycle.retire('operator',p['manifest_digest'])
    with authority.connect(env) as conn:assert conn.execute('SELECT count(*) AS n FROM hb_focus.configuration').fetchone()['n']==1
    anchor=json.loads((control.DIRECTORY/'authority.anchor').read_text())
    assert anchor['state']=='retirement_pending'
    assert all('managed-payload-removed' not in p.read_text() for p in control.DIRECTORY.glob('*.json'))
    with pytest.raises(Rejected,match='retirement_unresolved'):initialize(cfg)


def test_expired_unpruned_backup_cannot_restore(temporal_stack):
    activated(temporal_stack)
    data=encoded([source_row()]);received(data,delivery(data));b=lifecycle.backup('operator',at='2024-09-03T00:00:00Z')
    with pytest.raises(Rejected,match='backup_horizon_elapsed'):lifecycle.restore('operator',b['backup_id'],at='2024-09-20T00:00:00Z')


def test_restore_rebuilds_lost_authority_schema_under_current_controls(temporal_stack):
    env,cfg,policies=activated(temporal_stack)
    data=encoded([source_row(),source_row(provider='Microsoft',cost='20')]);received(data,delivery(data))
    b=lifecycle.backup('operator',at='2024-09-03T00:00:00Z');lifecycle.revoke('operator','reader-north')
    with authority.connect(env) as conn:conn.execute('DROP SCHEMA hb_focus CASCADE')
    ch=warehouse.client(env,'probe_admin')
    try:ch.command('TRUNCATE TABLE hb_focus.fact')
    finally:ch.close()
    restored=lifecycle.restore('operator',b['backup_id'],at='2024-09-04T00:00:00Z')
    assert restored['latest_controls_applied'] and measure(collection.query('reader-south'))==20
    with pytest.raises(Rejected):collection.query('reader-north')
    with pytest.raises(Rejected):collection.query('reader-company')


def test_missing_schema_bad_backup_does_not_poison_valid_restore(temporal_stack):
    env,cfg,policies=activated(temporal_stack)
    data=encoded([source_row()]);received(data,delivery(data));b=lifecycle.backup('operator',at='2024-09-03T00:00:00Z')
    with authority.connect(env) as conn:conn.execute('DROP SCHEMA hb_focus CASCADE')
    with pytest.raises(Rejected,match='backup_unavailable'):lifecycle.restore('operator','invalid',at='2024-09-04T00:00:00Z')
    with authority.connect(env) as conn:assert conn.execute("SELECT to_regclass('hb_focus.gate') AS value").fetchone()['value'] is None
    assert lifecycle.restore('operator',b['backup_id'],at='2024-09-04T00:00:00Z')['latest_controls_applied']
    assert measure(collection.query('reader-north'))==10


@pytest.mark.parametrize('operation',['archive','delete'])
def test_destructive_lost_ack_survives_new_coordinator(temporal_stack,monkeypatch,operation):
    env,cfg,policies=activated(temporal_stack)
    data=encoded([source_row(),source_row(provider='Microsoft',cost='20')]);_,item=received(data,delivery(data));a=slice_id(item,'north')
    b=lifecycle.backup('operator',at='2024-09-03T00:00:00Z')
    plan=lifecycle.preview('operator',identities=[a]);lifecycle.approve('lifecycle-approver',plan['manifest_digest'])
    original=lifecycle.remove_hot
    def lost_ack(store,part):
        original(store,part)
        raise RuntimeError('Synthetic lost removal acknowledgement')
    monkeypatch.setattr(lifecycle,'remove_hot',lost_ack)
    with pytest.raises(RuntimeError):
        if operation=='archive':lifecycle.archive_slice('operator',a,at='2024-09-03T00:00:00Z')
        else:lifecycle.delete('operator',plan['manifest_digest'])
    store=warehouse.Warehouse(env)
    assert store.rows(UUID(item['batch_id']),tenant_id('north'))==[]
    assert len(store.rows(UUID(item['batch_id']),tenant_id('south')))==1
    with authority.connect(env) as conn:
        j=control.Journal(env,conn);state=j.load(repair=True)
        assert state['operation']['kind']==operation
        if operation=='delete':assert state['slices'][a]['deletion']['phase']=='delete_committed'
    for action in [lambda:collection.query('reader-north'),lambda:lifecycle.recall('operator',a),lambda:lifecycle.restore('operator',b['backup_id'],at='2024-09-04T00:00:00Z'),lambda:initialize(cfg)]:
        with pytest.raises(Rejected):action()
