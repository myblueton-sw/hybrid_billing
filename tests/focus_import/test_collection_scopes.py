import copy
import csv
import io
import json
from datetime import datetime, timedelta, timezone
from fractions import Fraction
from pathlib import Path
from uuid import UUID

import pytest

from modules.focus_import import collection, connections, lifecycle, receipts, warehouse
from modules.focus_import.model import ROOT, Rejected, canonical, decode, digest, domain, domains_digest, tenant_id, versions
from modules.focus_import.service import initialize
from modules.focus_import.temporal import window
from test_temporal import configuration, encoded, policy, source_row


def data(provider='AWS', subaccount='synthetic-subaccount', cost='10', **changes):
    row=next(csv.DictReader(io.StringIO((ROOT/'tests/fixtures/focus/synthetic-1.3.csv').read_text())))
    row.update(ProviderName=provider,ServiceProviderName=provider,BillingAccountId='synthetic-account',SubAccountId=subaccount,
        BillingPeriodStart='2024-09-01T00:00:00Z',BillingPeriodEnd='2024-10-01T00:00:00Z',
        ChargePeriodStart='2024-09-01T00:00:00Z',ChargePeriodEnd='2024-09-02T00:00:00Z',BilledCost=cost,EffectiveCost=cost)
    row.update(changes)
    out=io.StringIO(newline='');writer=csv.DictWriter(out,fieldnames=list(row));writer.writeheader();writer.writerow(row)
    return out.getvalue().encode()


def definition(key='aws',provider='AWS',grain='day',subaccount='synthetic-subaccount'):
    return {'id':key,'dataset':'cost_usage','coverage_key':'primary','version':'1.3','grain':grain,
        'start':'2024-09-01T00:00:00Z','end':'2024-10-01T00:00:00Z','lag_seconds':0,
        'domains':[[provider,'synthetic-account',subaccount]],'semantic_profile':'focus-consumed-v1','max_replacements':1}


def descriptor(content,cid='aws',grain='day',revision=1,zero=None):
    rows=[]
    try:rows=decode(content,versions()['1.3'],conditional=True)
    except Rejected as exc:
        if str(exc)!='empty_input':raise
    finish='2024-09-02T00:00:00Z' if grain=='day' else '2024-10-01T00:00:00Z'
    return {'connection_id':cid,'window':window('2024-09-01T00:00:00Z',finish,grain),'revision':revision,
        'file_digest':digest(content),'row_count':len(rows),'domains_digest':domains_digest(rows),
        'complete_through':finish,'finalized':True,'zero_domains':zero or []}


@pytest.fixture
def scoped_stack(stack):
    cfg=configuration()
    cfg['tenants'][0]['assignments'].append(['AWS','synthetic-account',None,'2024-09-01T00:00:00+00:00','2024-10-01T00:00:00+00:00'])
    for t in cfg['tenants']:t['company_id']='company-one'
    cfg['views']=[{'actor':'reader-company','company_id':'company-one','tenant_keys':['north','south'],'active':True}]
    initialize(cfg)
    policies={t['key']:{**policy(),'basis':'window_end','hot_days':1,'total_days':36500} for t in cfg['tenants']}
    lifecycle.activate('operator','day',policies)
    return stack[0],cfg


def published(content,d):
    registered=collection.register('operator',content,'1.3',d)
    item=collection.collect('importer',registered['delivery_id'],content)
    collection.confirm('approver',registered['delivery_id'],item['manifest_digest'])
    collection.publish('approver',registered['delivery_id'])
    return registered,item


def total(result):
    return sum((Fraction(g['measures']['billed_cost']['coefficient'] or 0,10**18) for g in result['groups'].values()),Fraction())


def test_independent_grains_correction_and_visible_coverage(scoped_stack):
    collection.configure_connection('operator',definition())
    collection.configure_connection('operator',definition('azure','Microsoft','month'))
    a=data();published(a,descriptor(a))
    b=data('Microsoft',cost='20');published(b,descriptor(b,'azure','month'))
    company=collection.query('reader-company');assert company['row_count']==2 and total(company)==30
    paged=collection.query('reader-company',limit=1)
    collection.configure_connection('operator',definition('account_only',subaccount=None))
    with pytest.raises(Rejected,match='snapshot_changed'):collection.query('reader-company',cursor=paged['next_cursor'],expected_snapshot=paged['snapshot'])
    assert total(collection.query('reader-north'))==10 and total(collection.query('reader-south'))==20
    assert {c['connection_id'] for c in collection.query('reader-north')['expected_coverage']}=={'aws','account_only'}
    assert {c['connection_id'] for c in collection.query('reader-south')['expected_coverage']}=={'azure'}
    assert len(company['expected_coverage'])==31
    assert {c['state'] for c in company['expected_coverage']}=={'sealed','late'}
    assert company['rows'][0]['semantic_context']['classification']=='consumed-fields-validated'
    changed=data(cost='7');published(changed,descriptor(changed,revision=2))
    assert total(collection.query('reader-company'))==27
    assert total(collection.query('reader-south'))==20
    with pytest.raises(Rejected,match='overlap'):collection.configure_connection('operator',definition('duplicate'))


def test_zero_attestation_and_missing_domains(scoped_stack):
    body=definition();body['domains'].append(['Microsoft','synthetic-account','synthetic-subaccount'])
    collection.configure_connection('operator',body)
    a=data();d=descriptor(a)
    result=collection.register('operator',a,'1.3',d)
    assert result['state']=='quarantined' and result['reason']=='expected_domain_missing'
    status=lifecycle.status('operator');q=status['receipts'][result['receipt_id']]
    assert q['key'] is not None and q['descriptor']==d
    assert warehouse.object_client(scoped_stack[0]).get_object(Bucket=warehouse.BUCKET,Key=q['key'])['Body'].read()==a
    proof={'domain':body['domains'][1],'reason':'zero_usage','evidence_digest':'a'*64}
    d['zero_domains']=[proof]
    with pytest.raises(Rejected,match='replacement_approval_required'):collection.register('operator',a,'1.3',d)
    approval={'connection_id':'aws','descriptor':d,'reason':'confirm_zero_evidence','expires_at':(datetime.now(timezone.utc)+timedelta(hours=1)).isoformat(),'receipt_id':result['receipt_id']}
    sha=approved_plan('replacement',approval);d['replacement_plan']=sha
    registered,_=published(a,d)
    assert collection.register('operator',a,'1.3',d)['duplicate']
    assert lifecycle.status('operator')['receipts'][result['receipt_id']]['attempts']==1
    view=collection.query('reader-south');assert view['row_count']==0
    own=[c for c in view['expected_coverage'] if c['window']['start']==d['window']['start']]
    assert len(own)==1 and own[0]['known_zero'] and not own[0]['upstream_verification']
    for zeros in [[proof,proof],[{**proof,'domain':body['domains'][0]}],[{**proof,'domain':['Foreign','x','y']}]]:
        with pytest.raises(Rejected,match='zero_attestation'):connections.admission(decode(a,versions()['1.3']),body,{**d,'zero_domains':zeros},'1.3')


def test_null_subaccount_real_graph_query_archive_restore(scoped_stack):
    collection.configure_connection('operator',definition(subaccount=None))
    content=data(subaccount='');registered,item=published(content,descriptor(content))
    view=collection.query('reader-north');row=view['rows'][0]
    assert row['subaccount'] is None
    assert row['account_context']['usage_scope_status']=='account-only-subaccount-absent'
    graph=next(iter(view['account_hierarchies'].values()))
    assert len(graph['nodes'])==1 and graph['edges']==[]
    assert row['semantic_context']['fields']['sub_account_id'] is None
    identity=item['batch_id']+':'+str(tenant_id('north'))
    backup=lifecycle.backup('operator')
    lifecycle.archive_slice('operator',identity,at='2024-09-03T00:00:00Z')
    assert total(collection.query('reader-north'))==10
    lifecycle.restore('operator',backup['backup_id'])
    assert collection.query('reader-north')['rows'][0]['subaccount'] is None
    denied=data(subaccount='other');d=descriptor(denied)
    rejected=collection.register('operator',denied,'1.3',d)
    assert rejected['state']=='rejected' and rejected['reason']=='unmapped_or_denied_source'


def approved_plan(kind,body):
    result=receipts.prepare('operator',kind,body)
    lifecycle.approve('lifecycle-approver',result['manifest_digest'])
    return result['manifest_digest']


def test_quarantine_replacement_and_original_backup(scoped_stack):
    env,cfg=scoped_stack;collection.configure_connection('operator',definition())
    invalid=data(BilledCost='');d=descriptor(invalid)
    bad=collection.register('operator',invalid,'1.3',d);assert bad['state']=='quarantined'
    q=lifecycle.status('operator')['receipts'][bad['receipt_id']]
    s3=warehouse.object_client(env)
    assert s3.get_object(Bucket=warehouse.BUCKET,Key=q['key'])['Body'].read()==invalid
    backup=lifecycle.backup('operator')
    s3.delete_object(Bucket=warehouse.BUCKET,Key=q['key'])
    lifecycle.restore('operator',backup['backup_id'])
    assert s3.get_object(Bucket=warehouse.BUCKET,Key=q['key'])['Body'].read()==invalid
    good=data();new=descriptor(good)
    with pytest.raises(Rejected,match='replacement_approval'):collection.register('operator',good,'1.3',new)
    plan={'connection_id':'aws','descriptor':new,'reason':'correct_source','expires_at':(datetime.now(timezone.utc)+timedelta(hours=1)).isoformat(),'receipt_id':bad['receipt_id']}
    sha=approved_plan('replacement',plan)
    new['replacement_plan']=sha;published(good,new)
    q2=lifecycle.status('operator')['receipts'][bad['receipt_id']]
    assert q2['state']=='resolved' and q2['attempts']==1 and q2['origin']==q['origin']
    latest=lifecycle.status('operator')
    successor=next(v for v in latest['deliveries'].values() if v['state']=='sealed')
    assert latest['slices'][successor['batch_id']+':'+str(tenant_id('north'))]['origin_received']==q['origin']
    assert collection.register('operator',good,'1.3',new)['duplicate']
    assert lifecycle.status('operator')['receipts'][bad['receipt_id']]['attempts']==1
    with pytest.raises(Rejected):collection.register('operator',data(cost='11'),'1.3',{**descriptor(data(cost='11'),revision=2),'replacement_plan':sha})
    receipts.retention('operator',bad['receipt_id'],hold='legal_hold')
    assert not receipts.retention('operator',bad['receipt_id'],at='2126-10-01T00:00:00+00:00')['eligible']
    assert total(collection.query('reader-north'))==10


def test_reopening_requires_exact_separate_approval(scoped_stack,monkeypatch):
    collection.configure_connection('operator',definition(grain='month'))
    original=data();published(original,descriptor(original,grain='month'))
    changed=data(cost='7');d=descriptor(changed,grain='month',revision=2)
    d.update(finalized=False,complete_through='2024-09-10T00:00:00Z')
    with pytest.raises(Rejected,match='approval'):collection.register('operator',changed,'1.3',d)
    body={'connection_id':'aws','descriptor':d,'reason':'source_reopened','expires_at':(datetime.now(timezone.utc)+timedelta(hours=1)).isoformat(),'receipt_id':None}
    sha=approved_plan('reopen',body)
    with monkeypatch.context() as controlled:
        controlled.setattr(collection,'now',lambda:body['expires_at'])
        with pytest.raises(Rejected,match='approval_changed'):collection.register('operator',changed,'1.3',{**d,'reopen_plan':sha})
    wrong={**d,'reopen_plan':sha,'complete_through':'2024-09-11T00:00:00Z'}
    with pytest.raises(Rejected,match='approval_changed'):collection.register('operator',changed,'1.3',wrong)
    d['reopen_plan']=sha;published(changed,d)
    assert total(collection.query('reader-north'))==7
    assert not collection.query('reader-north')['expected_coverage'][0]['finalized']


def test_failed_replacement_lineage_has_one_budget(scoped_stack):
    collection.configure_connection('operator',definition())
    invalid=data(BilledCost='');bad=collection.register('operator',invalid,'1.3',descriptor(invalid))
    another=data(ServiceCategory='invalid-category');d=descriptor(another)
    body={'connection_id':'aws','descriptor':d,'reason':'replace_bad_source','expires_at':(datetime.now(timezone.utc)+timedelta(hours=1)).isoformat(),'receipt_id':bad['receipt_id']}
    sha=approved_plan('replacement',body);d['replacement_plan']=sha
    failed=collection.register('operator',another,'1.3',d)
    assert failed['state']=='escalated'
    all_receipts=lifecycle.status('operator')['receipts']
    assert all_receipts[bad['receipt_id']]['attempts']==1
    assert all_receipts[failed['receipt_id']]['root_receipt']==bad['receipt_id']
    body['receipt_id']=failed['receipt_id'];body['descriptor']=descriptor(data())
    with pytest.raises(Rejected,match='replacement_receipt'):receipts.prepare('operator','replacement',body)
    removal=receipts.deletion_plan('operator',bad['receipt_id'],at='2124-10-01T00:00:00+00:00')
    lifecycle.approve('lifecycle-approver',removal['manifest_digest'])
    receipts.delete('operator',removal['manifest_digest'],at='2124-10-01T00:00:00+00:00')
    with pytest.raises(Rejected,match='receipt_scope_closed'):collection.register('operator',data(),'1.3',descriptor(data()))


def test_quarantine_lost_ack_reserves_consumption_before_effect(scoped_stack,monkeypatch):
    from modules.focus_import import archive,authority,control
    env,cfg=scoped_stack
    collection.configure_connection('operator',definition())
    invalid=data(BilledCost='');bad=collection.register('operator',invalid,'1.3',descriptor(invalid))
    another=data(ServiceCategory='invalid-category');d=descriptor(another)
    body={'connection_id':'aws','descriptor':d,'reason':'replace_bad_source','expires_at':(datetime.now(timezone.utc)+timedelta(hours=1)).isoformat(),'receipt_id':bad['receipt_id']}
    sha=approved_plan('replacement',body);d['replacement_plan']=sha
    real=archive.put
    def lost(s3,key,content):
        real(s3,key,content)
        if key.startswith('quarantine/'):raise RuntimeError('synthetic lost PUT acknowledgment')
    monkeypatch.setattr(archive,'put',lost)
    with pytest.raises(RuntimeError):collection.register('operator',another,'1.3',d)
    with authority.connect(env) as conn:
        j=control.Journal(env,conn);state=j.load()
        assert state['operation']['kind']=='quarantine'
        assert state['plans'][sha]['state']=='executed'
        assert state['receipts'][bad['receipt_id']]['attempts']==1
    with pytest.raises(Rejected,match='operation_unresolved'):collection.register('operator',another,'1.3',d)


def test_quarantine_retention_policy_basis_and_delete_hold(scoped_stack):
    collection.configure_connection('operator',definition())
    invalid=data(BilledCost='');bad=collection.register('operator',invalid,'1.3',descriptor(invalid))
    retained=receipts.retention('operator',bad['receipt_id'],at='2124-10-01T00:00:00+00:00')
    assert retained['eligible']
    receipts.retention('operator',bad['receipt_id'],hold='case_hold')
    with pytest.raises(Rejected):receipts.deletion_plan('operator',bad['receipt_id'],at='2124-10-01T00:00:00+00:00')
    receipts.retention('operator',bad['receipt_id'],hold='case_hold',release=True)
    plan=receipts.deletion_plan('operator',bad['receipt_id'],at='2124-10-01T00:00:00+00:00')
    lifecycle.approve('lifecycle-approver',plan['manifest_digest'])
    assert receipts.delete('operator',plan['manifest_digest'],at='2124-10-01T00:00:00+00:00')['state']=='deleted'
    with pytest.raises(Rejected,match='receipt_scope_closed'):collection.register('operator',data(),'1.3',descriptor(data()))


def test_scoped_collection_cli_entrypoint(scoped_stack,monkeypatch,capsys,tmp_path):
    from test_intake_integration import invoke
    config=tmp_path/'connection.json';config.write_text(json.dumps(definition()))
    created=invoke(monkeypatch,capsys,'connection-register','--actor','operator','--file',str(config))
    assert created['connection']['semantic_definition']['id']=='focus-consumed-v1'
    source=tmp_path/'delivery.csv';source.write_bytes(data())
    described=invoke(monkeypatch,capsys,'describe','--file',str(source),'--version','1.3','--connection','aws',
        '--start','2024-09-01T00:00:00Z','--end','2024-09-02T00:00:00Z','--grain','day','--revision','1','--complete-through','2024-09-02T00:00:00Z','--finalized')
    desc=tmp_path/'descriptor.json';desc.write_text(json.dumps(described['descriptor']))
    registered=invoke(monkeypatch,capsys,'register','--actor','operator','--file',str(source),'--version','1.3','--descriptor',str(desc))
    item=invoke(monkeypatch,capsys,'run-due','--actor','importer','--inbox',str(tmp_path))['collected'][0]
    invoke(monkeypatch,capsys,'confirm','--actor','approver','--delivery',registered['delivery_id'],'--manifest-digest',item['manifest_digest'])
    invoke(monkeypatch,capsys,'publish','--actor','approver','--delivery',registered['delivery_id'])
    result=invoke(monkeypatch,capsys,'query','--actor','reader-north')
    assert total(result)==10 and result['rows'][0]['semantic_context']['fields']['charge_category']=='Usage'


def test_consumed_parquet_typed_time_and_exact_tags(scoped_stack):
    import pyarrow as pa
    import pyarrow.parquet as pq
    from decimal import Decimal
    from modules.focus_import.model import NUMERIC,TIMES,instant
    collection.configure_connection('operator',definition())
    source=next(csv.DictReader(io.StringIO(data(Tags='{"weight":0.1000000000000000001}').decode())))
    reverse={v:k for k,v in versions()['1.3']['columns'].items()}
    arrays={}
    for key,value in source.items():
        name=reverse.get(key)
        if name in NUMERIC:arrays[key]=pa.array([Decimal(value) if value else None],type=pa.decimal128(38,18))
        elif name in TIMES:arrays[key]=pa.array([instant(value)],type=pa.timestamp('us',tz='UTC'))
        else:arrays[key]=pa.array([value],type=pa.string())
    output=io.BytesIO();pq.write_table(pa.table(arrays),output,compression='zstd');content=output.getvalue()
    published(content,descriptor(content))
    row=collection.query('reader-north')['rows'][0]
    assert row['semantic_context']['fields']['tags']=='{"weight":0.1000000000000000001}'
    assert row['semantic_context']['fields']['charge_period_start']=='2024-09-01T00:00:00+00:00'
    assert json.loads(row['source_json'])['ChargePeriodStart']=='2024-09-01T00:00:00Z'


def test_authorized_broken_numeric_quarantines_but_foreign_last_row_does_not(scoped_stack):
    env,cfg=scoped_stack;collection.configure_connection('operator',definition())
    good=data();broken=data(BilledCost='broken');d={**descriptor(good),'file_digest':digest(broken)}
    q=collection.register('operator',broken,'1.3',d)
    assert q['state']=='quarantined'
    retained=lifecycle.status('operator')['receipts'][q['receipt_id']]
    assert warehouse.object_client(env).get_object(Bucket=warehouse.BUCKET,Key=retained['key'])['Body'].read()==broken
    rows=list(csv.DictReader(io.StringIO(broken.decode())))
    foreign={**rows[0],'BillingAccountId':'unauthorized-account'}
    out=io.StringIO(newline='');writer=csv.DictWriter(out,fieldnames=list(rows[0]));writer.writeheader();writer.writerows(rows+[foreign])
    content=out.getvalue().encode()
    d={**d,'file_digest':digest(content),'row_count':2}
    from modules.focus_import.connections import source_scopes
    d['domains_digest']=domains_digest(source_scopes(content,versions()['1.3'],True))
    denied=collection.register('operator',content,'1.3',d)
    entry=lifecycle.status('operator')['receipts'][denied['receipt_id']]
    assert denied['state']=='rejected' and entry['key'] is None and entry['descriptor'] is None


@pytest.mark.parametrize('kind',['utf8','csv'])
def test_unclassifiable_bytes_get_safe_metadata_only(scoped_stack,monkeypatch,kind):
    from modules.focus_import import archive
    collection.configure_connection('operator',definition())
    source=data();header=source.splitlines(keepends=True)[0]
    broken=b'\xff' if kind=='utf8' else header+b'"unterminated'
    d={**descriptor(source),'file_digest':digest(broken)}
    def no_raw_put(*args):raise AssertionError('unsafe raw quarantine PUT')
    monkeypatch.setattr(archive,'put',no_raw_put)
    result=collection.register('operator',broken,'1.3',d)
    assert result['state']=='rejected' and result['reason']=='source_format'
    q=lifecycle.status('operator')['receipts'][result['receipt_id']]
    assert q['key'] is None and q['descriptor'] is None and not q['admitted']
