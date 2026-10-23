import copy
import csv
import io
import json
from datetime import datetime,timezone
from fractions import Fraction
from uuid import UUID

import psycopg
from psycopg.types.json import Jsonb
import pytest

from modules.focus_import import authority,collection,control,hierarchy_admin,hierarchy_store,lifecycle,warehouse
from modules.focus_import.model import ROOT,Rejected,canonical,decode,digest,domain,domains_digest,tenant_id,versions
from modules.focus_import.service import Coordinator,initialize
from modules.focus_import.temporal import window


PROVIDERS={'aws':'AWS','azure':'Microsoft','gcp':'Google Cloud','oci':'Oracle','vmware':'VMware','alibaba':'Alibaba Cloud'}
START='2024-09-01T00:00:00Z'
END='2024-10-01T00:00:00Z'


def fixture_catalog():
    return json.loads((ROOT/'tests/fixtures/focus/native-hierarchy/six-provider.json').read_text())


def source_data(catalog):
    columns=versions()['1.0']['columns'];records=[];seen=set()
    for b in catalog['bindings']:
        token=canonical([b['provider'],b['account'],b['subaccount'],b['tenant_key']])
        if token in seen:continue
        seen.add(token);n=len(records)+1
        row={'provider':PROVIDERS[b['provider']],'service':'Synthetic native hierarchy','currency':'CNY' if b['provider']=='alibaba' else 'USD','account':b['account'],'subaccount':b['subaccount'],'consumed_unit':'Hours','pricing_unit':'Hours','billing_start':START,'billing_end':END,'charge_start':START,'charge_end':'2024-09-02T00:00:00Z','billed_cost':str(n)+'.000000000000000001','effective_cost':str(n),'list_cost':str(n),'contracted_cost':'','consumed_quantity':'1.25','pricing_quantity':'1.25','list_unit_price':'0.75','contracted_unit_price':''}
        records.append({columns[k]:v for k,v in row.items()})
    stream=io.StringIO(newline='');writer=csv.DictWriter(stream,fieldnames=list(columns.values()));writer.writeheader();writer.writerows(records)
    return stream.getvalue().encode()


def config_for(catalog,data):
    rows=decode(data,versions()['1.0']);tenants=[]
    owners={canonical([PROVIDERS[b['provider']],b['account'],b['subaccount']]):b['tenant_key'] for b in catalog['bindings']}
    for key in sorted({b['tenant_key'] for b in catalog['bindings']}):
        assignments=sorted({canonical(domain(r)) for r in rows if owners[canonical([r['provider'],r['account'],r['subaccount']])]==key})
        tenants.append({'key':key,'label':'Synthetic '+key,'timezone':'UTC','reporting_currency':'USD','currencies':['USD','CNY',None],'retention_days':30,'actor':'reader-'+key,'active':True,'assignments':[json.loads(x) for x in assignments],'company_id':'other' if 'foreign' in key else 'same-company','company_label':'Same display label'})
    members=[t['key'] for t in tenants if t['company_id']=='same-company']
    return {'source':'synthetic-native-hierarchy','version':'1.0','revision':1,'tenants':tenants,'snapshot':{'file_digest':digest(data),'row_count':len(rows),'domains_digest':domains_digest(rows)},'views':[{'actor':'reader-native-company','company_id':'same-company','tenant_keys':members,'active':True}]}


def descriptor(data,revision=1):
    rows=decode(data,versions()['1.0'])
    return {'window':window(START,END,'month'),'revision':revision,'file_digest':digest(data),'row_count':len(rows),'domains_digest':domains_digest(rows),'complete_through':END,'finalized':True}


def policies(cfg):
    return {t['key']:{'revision':1,'basis':'window_end','hot_days':0,'total_days':10,'grace_days':0,'backup_days':30,'reject_before':'2020-01-01T00:00:00Z','approved_by':'lifecycle-approver'} for t in cfg['tenants']}


@pytest.fixture
def native(stack):
    catalog=fixture_catalog()
    for e in catalog['edges']:e.update(valid_from='2024-01-01T00:00:00Z',valid_to='2025-01-01T00:00:00Z')
    for b in catalog['bindings']:b.update(valid_from='2024-01-01T00:00:00Z',valid_to='2025-01-01T00:00:00Z')
    data=source_data(catalog);cfg=config_for(catalog,data);initialize(cfg)
    lifecycle.activate('operator','month',policies(cfg))
    return stack[0],cfg,catalog,data


def install(catalog):
    plan=hierarchy_admin.prepare('operator',catalog);lifecycle.approve('lifecycle-approver',plan['manifest_digest']);return hierarchy_admin.apply('operator',plan['manifest_digest'])


def published(data,revision=1):
    receipt=collection.register('operator',data,'1.0',descriptor(data,revision))['delivery_id']
    batch=collection.collect('importer',receipt,data);collection.confirm('approver',receipt,batch['manifest_digest']);collection.publish('approver',receipt)
    return batch


def cost(result):
    return sum((Fraction(g['measures']['billed_cost']['coefficient'] or 0,10**18) for g in result['groups'].values()),Fraction(0))


def actor_for(cfg,provider):
    mapping={PROVIDERS[b['provider']]:b['tenant_key'] for b in fixture_catalog()['bindings']}
    return 'reader-'+mapping[provider]


def test_six_native_profiles_persist_without_cost_multiplication(native):
    env,cfg,catalog,data=native;install(catalog);batch=published(data)
    totals=collection.query('reader-native-company',limit=100)
    assert totals['row_count']==sum(len(t['assignments']) for t in cfg['tenants'] if t['company_id']=='same-company')
    assert {n['provider'] for body in totals['account_hierarchies'].values() for n in body['nodes']}==set(PROVIDERS)
    assert all(body['status']=='typed-native' for body in totals['account_hierarchies'].values() if body['bindings'])
    assert all(r['account_context']['native_roles'] for r in totals['rows'])
    for token,group in totals['groups'].items():
        selected=[r for r in totals['rows'] if [r[k] for k in ['provider','service','currency','consumed_unit','pricing_unit']]==json.loads(token)]
        lexical=versions()['1.0']['columns']['billed_cost']
        expected=sum((Fraction(json.loads(r['source_json'])[lexical]) for r in selected),Fraction(0))
        assert Fraction(group['measures']['billed_cost']['coefficient'],10**18)==expected
    for body in totals['account_hierarchies'].values():
        ids={n['id'] for n in body['nodes']}
        assert all(e['child'] in ids and e['parent'] in ids for e in body['edges'])
    with authority.connect(env) as conn:
        assert conn.execute('SELECT count(*) AS n FROM hb_focus.native_binding').fetchone()['n']==len(decode(data,versions()['1.0']))
        assert conn.execute('SELECT count(*) AS n FROM hb_focus.native_binding_role').fetchone()['n']>=len(decode(data,versions()['1.0']))
        assert conn.execute('SELECT manifest FROM hb_focus.batch WHERE batch_id=%s',(UUID(batch['batch_id']),)).fetchone()['manifest']['slices']


def test_direct_rls_scoped_references_and_reader_write_denial(native):
    env,cfg,catalog,data=native;install(catalog);published(data)
    tenant=cfg['tenants'][0];tid=tenant_id(tenant['key']);role=authority.reader_role(tid)
    with authority.connect(env,role) as conn:
        for table in hierarchy_store.TABLES:
            assert {r['tenant_id'] for r in conn.execute('SELECT tenant_id FROM hb_focus.'+table)}<={tid}
        with pytest.raises(psycopg.errors.InsufficientPrivilege):conn.execute('DELETE FROM hb_focus.native_head')
        with pytest.raises(psycopg.errors.InsufficientPrivilege):conn.execute("UPDATE hb_focus.native_head SET snapshot_digest=repeat('0',64)")
    with authority.connect(env) as conn:
        heads=conn.execute('SELECT * FROM hb_focus.native_head ORDER BY tenant_id').fetchall();assert len(heads)>=2
        own=[r['node_id'] for r in conn.execute('SELECT node_id FROM hb_focus.native_node WHERE tenant_id=%s AND batch_id=%s AND snapshot_digest=%s',(heads[0]['tenant_id'],heads[0]['batch_id'],heads[0]['snapshot_digest']))]
        other=next(r['node_id'] for r in conn.execute('SELECT node_id FROM hb_focus.native_node WHERE tenant_id=%s AND batch_id=%s AND snapshot_digest=%s',(heads[1]['tenant_id'],heads[1]['batch_id'],heads[1]['snapshot_digest'])) if r['node_id'] not in own)
        with pytest.raises(psycopg.errors.ForeignKeyViolation):
            conn.execute('INSERT INTO hb_focus.native_edge VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)',(heads[0]['tenant_id'],heads[0]['batch_id'],heads[0]['snapshot_digest'],'cross-tenant',own[0],other,'resource_parent','resource',Jsonb({'id':'cross-tenant','child':str(own[0]),'parent':str(other)})))


def test_focus_only_backfill_never_invents_native_ids_and_stale_pages_reject(native):
    env,cfg,catalog,data=native;first=published(data)
    page=collection.query('reader-native-company',limit=1)
    assert all(not r['account_context']['native_roles'] for r in page['rows'])
    initial_digest=next(iter(page['account_hierarchies'].values()))['digest']
    install(catalog)
    enriched=collection.query('reader-native-company',limit=1)
    assert any(r['account_context']['native_roles'] for r in enriched['rows'])
    assert enriched['snapshot']!=page['snapshot']
    with pytest.raises(Rejected,match='snapshot_changed'):collection.query('reader-native-company',limit=1,cursor=page['next_cursor'],expected_snapshot=page['snapshot'])
    with authority.connect(env) as conn:
        assert conn.execute('SELECT 1 FROM hb_focus.native_snapshot WHERE snapshot_digest=%s',(initial_digest,)).fetchone()
        assert all(part['hierarchy_digest'] for part in conn.execute('SELECT manifest FROM hb_focus.batch WHERE batch_id=%s',(UUID(first['batch_id']),)).fetchone()['manifest']['slices'].values())


def test_hierarchy_plan_cannot_change_authority_or_ignore_lifecycle_change(native):
    env,cfg,catalog,data=native;first=published(data)
    with pytest.raises(Rejected,match='permission'):hierarchy_admin.prepare(cfg['tenants'][0]['actor'],catalog)
    bad=copy.deepcopy(catalog);bad['bindings'][0]['tenant_key']='unregistered'
    with pytest.raises(Rejected):hierarchy_admin.prepare('operator',bad)
    p=hierarchy_admin.prepare('operator',catalog)
    with pytest.raises(Rejected,match='approval'):hierarchy_admin.apply('operator',p['manifest_digest'])
    lifecycle.approve('lifecycle-approver',p['manifest_digest'])
    identity=first['batch_id']+':'+str(tenant_id(cfg['tenants'][0]['key']))
    lifecycle.restriction('operator',identity,'case','Synthetic hold')
    with pytest.raises(Rejected,match='plan_changed'):hierarchy_admin.apply('operator',p['manifest_digest'])


def test_native_snapshot_survives_archive_and_lost_schema_restore(native):
    env,cfg,catalog,data=native;install(catalog);first=published(data)
    actor=cfg['tenants'][0]['actor'];tid=str(tenant_id(cfg['tenants'][0]['key']));identity=first['batch_id']+':'+tid
    before=collection.query(actor,limit=100)
    lifecycle.archive_slice('operator',identity,at='2024-10-02T00:00:00Z')
    archived=collection.query(actor,limit=100)
    assert archived['account_hierarchies']==before['account_hierarchies'] and archived['groups']==before['groups']
    company_before=collection.query('reader-native-company',limit=100)
    def native_tables():
        with authority.connect(env) as conn:
            return {table:sorted(canonical(r['value']) for r in conn.execute('SELECT row_to_json(t) AS value FROM hb_focus.'+table+' t')) for table in hierarchy_store.TABLES}
    tables_before=native_tables()
    backup=lifecycle.backup('operator',at='2024-10-02T00:00:00Z')
    with authority.connect(env) as conn:conn.execute('DROP SCHEMA hb_focus CASCADE')
    lifecycle.restore('operator',backup['backup_id'],at='2024-10-03T00:00:00Z')
    restored=collection.query(actor,limit=100)
    assert restored['account_hierarchies']==before['account_hierarchies'] and restored['groups']==before['groups']
    assert native_tables()==tables_before
    company_after=collection.query('reader-native-company',limit=100)
    assert company_after['account_hierarchies']==company_before['account_hierarchies'] and company_after['groups']==company_before['groups']


def test_shared_payer_projection_never_discloses_other_tenant_siblings(native):
    env,cfg,catalog,data=native;install(catalog);published(data)
    north=collection.query('reader-north-a',provider='AWS',limit=100)
    company=collection.query('reader-native-company',provider='AWS',limit=100)
    foreign=collection.query('reader-foreign',provider='AWS',limit=100)
    nodes=lambda result:{n['native_id'] for body in result['account_hierarchies'].values() for n in body['nodes']}
    native_nodes={n['key']:n['native_id'] for n in catalog['nodes']}
    assert native_nodes['aws-payer'] in nodes(north)&nodes(foreign)
    assert native_nodes['aws-foreign'] not in nodes(north)|nodes(company)
    assert native_nodes['aws-usage-b'] not in nodes(north) and native_nodes['aws-usage-b'] in nodes(company)
    assert all(n['provider']=='aws' for body in north['account_hierarchies'].values() for n in body['nodes'])
    assert company['row_count']==north['row_count']+collection.query('reader-north-b',provider='AWS')['row_count']
    payer=next(n['id'] for body in north['account_hierarchies'].values() for n in body['nodes'] if n['native_id']==native_nodes['aws-payer'])
    filtered=collection.query('reader-north-a',hierarchy_node=payer,limit=100)
    assert filtered['row_count']==north['row_count']


def test_axis_grouping_and_multiple_attachment_rejection(native):
    env,cfg,catalog,data=native;install(catalog);published(data)
    resource=collection.query('reader-north-a',provider='GCP',group_kind='organization',group_axis='resource')
    billing=collection.query('reader-north-a',provider='GCP',group_kind='organization',group_axis='billing',group_role='billing')
    assert len(resource['account_groups'])==len(billing['account_groups'])==1
    assert set(resource['account_groups'])!=set(billing['account_groups'])
    for result in [resource,billing]:
        controls=next(iter(result['account_groups'].values()));assert controls==result['groups']
    with pytest.raises(Rejected,match='nonadditive_ancestors'):collection.query('reader-north-a',provider='VMware',group_kind='datastore',group_axis='attachment')
    vm=collection.query('reader-north-a',provider='VMware')
    assert vm['row_count']==1 and vm['rows'][0]['account_context']['native_roles'].get('billing') is None
    grouped=collection.query('reader-native-company',group_kind='role',group_axis='binding',group_role='billing')
    expected={}
    for controls in grouped['account_groups'].values():
        for key,item in controls.items():
            target=expected.setdefault(key,{'rows':0,'measures':{n:{'coefficient':None,'present':0,'missing':0} for n in item['measures']}})
            target['rows']+=item['rows']
            for name,measure in item['measures'].items():
                t=target['measures'][name];t['present']+=measure['present'];t['missing']+=measure['missing']
                if measure['coefficient'] is not None:t['coefficient']=(t['coefficient'] or 0)+measure['coefficient']
    assert expected==grouped['groups'] and grouped['unresolved_group_rows']==0


def test_native_move_only_changes_new_delivery_snapshot(native):
    env,cfg,catalog,data=native;install(catalog);first=published(data)
    before=collection.query('reader-north-a',provider='GCP');old_body=next(v for v in before['account_hierarchies'].values() if v['bindings'])
    updated=copy.deepcopy(catalog);updated['revision']=2
    node=next(n for n in updated['nodes'] if n['key']=='gcp-resource-org');node['native_id']='999999999999';node['label']='Synthetic moved parent'
    installed=install(updated);assert installed['frozen_typed_slices']>0
    unchanged=collection.query('reader-north-a',provider='GCP')
    assert unchanged['account_hierarchies']==before['account_hierarchies']
    second=published(data,revision=2);after=collection.query('reader-north-a',provider='GCP')
    new_body=next(v for v in after['account_hierarchies'].values() if v['bindings'])
    old_project=next(n['id'] for n in old_body['nodes'] if n['kind']=='project');new_project=next(n['id'] for n in new_body['nodes'] if n['kind']=='project')
    assert old_project==new_project and old_body['digest']!=new_body['digest'] and before['groups']==after['groups']
    with authority.connect(env) as conn:assert conn.execute('SELECT 1 FROM hb_focus.native_snapshot WHERE batch_id=%s AND snapshot_digest=%s',(UUID(first['batch_id']),old_body['digest'])).fetchone()


def test_head_tampering_and_legacy_paging_pin_are_rejected(native):
    env,cfg,catalog,data=native;first=published(data)
    before=collection.query('reader-north-a');install(catalog)
    tid=tenant_id('north-a')
    old=next(v['digest'] for k,v in before['account_hierarchies'].items() if k.endswith(str(tid)))
    with authority.connect(env) as conn:conn.execute('UPDATE hb_focus.native_head SET snapshot_digest=%s WHERE tenant_id=%s AND batch_id=%s',(old,tid,UUID(first['batch_id'])))
    with pytest.raises(Rejected,match='hierarchy_snapshot_changed'):collection.query('reader-north-a')


def test_missing_legacy_page_hierarchy_pin_rejects(stack):
    env,unused=stack;catalog=fixture_catalog();data=source_data(catalog);cfg=config_for(catalog,data);initialize(cfg)
    loader=Coordinator('importer');approver=Coordinator('approver');reader=Coordinator('reader-north-a')
    try:
        item=loader.ingest(data,'1.0',1);approver.confirm(item['batch_id'],item['manifest_digest']);approver.publish(item['batch_id'])
        page=reader.query(limit=1)
        with pytest.raises(Rejected,match='hierarchy_required'):reader.query(after=page['next_after'],expected_batch=page['batch_id'])
        continuation=reader.query(limit=1,after=page['next_after'],expected_batch=page['batch_id'],expected_hierarchy=page['hierarchy_snapshot'])
        assert continuation['rows'][0]['ordinal']!=page['rows'][0]['ordinal']
    finally:loader.close();approver.close();reader.close()


def test_native_hierarchy_cli_entrypoint_end_to_end(native,tmp_path,monkeypatch,capsys):
    from tools.focus_lifecycle import main
    env,cfg,catalog,data=native
    def call(args):
        monkeypatch.setattr('sys.argv',['focus_lifecycle.py',*args]);assert main()==0
        return json.loads(capsys.readouterr().out.splitlines()[-1])
    file=tmp_path/'native.json';file.write_text(json.dumps(catalog));csvfile=tmp_path/'input.csv';csvfile.write_bytes(data)
    desc=tmp_path/'descriptor.json';desc.write_text(json.dumps(descriptor(data)))
    assert len(call(['hierarchy-profiles'])['profiles']['providers'])==6
    plan=call(['hierarchy-plan','--actor','operator','--file',str(file)])
    call(['approve','--actor','lifecycle-approver','--manifest-digest',plan['manifest_digest']])
    call(['hierarchy-apply','--actor','operator','--manifest-digest',plan['manifest_digest']])
    receipt=call(['register','--actor','operator','--file',str(csvfile),'--descriptor',str(desc),'--version','1.0'])['delivery_id']
    item=call(['collect','--actor','importer','--delivery',receipt,'--file',str(csvfile)])
    call(['confirm','--actor','approver','--delivery',receipt,'--manifest-digest',item['manifest_digest']]);call(['publish','--actor','approver','--delivery',receipt])
    result=call(['query','--actor','reader-native-company','--limit','100','--group-kind','role','--group-axis','binding','--group-role','billing'])
    assert result['row_count']==sum(len(t['assignments']) for t in cfg['tenants'] if t['company_id']=='same-company')
    assert result['account_groups'] and all(r['account_context']['native_roles'] for r in result['rows'])
