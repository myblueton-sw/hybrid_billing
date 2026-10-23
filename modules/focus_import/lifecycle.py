import base64
import copy
import json
import os
from contextlib import contextmanager, nullcontext
from datetime import datetime, timezone
from uuid import UUID, uuid4

from psycopg import sql

from modules.focus_local.database import environment
from . import archive, authority, control, warehouse
from .collection import now
from .hierarchy_store import TABLES as NATIVE_TABLES
from .model import BINDINGS, Rejected, canonical, digest, instant, tenant_id
from .temporal import eligible, validate_policy


METADATA_TABLES=['configuration','tenant','gate','batch','slice','publication','confirmation','decision','audience','audience_member']+NATIVE_TABLES
RIGHTS={name:{'operator'} for name in ['activate','archive','recall','hold','reference','preview','delete','status','backup','restore','prune-backups','revoke','policy-preview','policy-apply','retire-preview','retire','hierarchy-plan','hierarchy-apply']}
RIGHTS['approve']={'lifecycle-approver'}


def metadata(conn):
    return {table:[r['value'] for r in conn.execute(sql.SQL('SELECT row_to_json(t) AS value FROM {} t').format(sql.Identifier('hb_focus',table)))] for table in METADATA_TABLES}


@contextmanager
def access(actor,right,repair=False,restore_request=None):
    if actor not in RIGHTS.get(right,set()):raise Rejected('permission')
    env=environment()
    with control.locked(),authority.connect(env) as conn:
        j=control.Journal(env,conn)
        state=j.load(repair=repair)
        if state and state.get('retired') and right!='status':raise Rejected('installation_retired')
        if right!='activate' and state is None:raise Rejected('lifecycle_not_activated')
        if state and state.get('operation'):raise Rejected('operation_unresolved')
        if right=='restore':
            if restore_request is None:raise Rejected('backup_request_required')
            j.restore_body=validated_backup(j,*restore_request)
        bootstrapped=False
        if right=='restore' and control.checkpoint(conn) is None:
            cfg=state['metadata']['configuration'][0]['body']
            authority.initialize(cfg,env)
            with conn.transaction():
                conn.execute('UPDATE hb_focus.gate SET busy=true')
                for name in state['revoked_actors']:
                    conn.execute("UPDATE hb_focus.tenant SET active=false WHERE properties->>'actor'=%s",(name,))
                    conn.execute('UPDATE hb_focus.audience SET active=false WHERE actor=%s',(name,))
                conn.execute('INSERT INTO hb_focus.control_checkpoint VALUES (1,%s,%s,%s)',(UUID(j.head['installation']),j.head['sequence'],j.head['digest']))
            bootstrapped=True
        with (conn.transaction() if right!='retire' else nullcontext()):
            gates=conn.execute('SELECT * FROM hb_focus.gate ORDER BY source FOR UPDATE').fetchall()
            tenants=conn.execute('SELECT * FROM hb_focus.tenant ORDER BY tenant_id FOR UPDATE').fetchall()
            if any(g['busy'] for g in gates) and not bootstrapped or conn.execute("SELECT 1 FROM hb_focus.batch WHERE state<>'sealed'").fetchone():raise Rejected('operation_unresolved')
            yield j,conn,warehouse.Warehouse(env),tenants


def activate(actor,grain,policies):
    if grain not in ['day','month']:raise Rejected('source_granularity')
    with access(actor,'activate') as (j,conn,store,tenants):
        if j.state:raise Rejected('lifecycle_already_activated')
        expected={t['properties']['key'] for t in tenants}
        if set(policies)!=expected:raise Rejected('tenant_policy_set')
        for p in policies.values():validate_policy(p)
        cfg=conn.execute('SELECT * FROM hb_focus.configuration WHERE id=1').fetchone()
        state={'config_digest':cfg['digest'],'grain':grain,'reject_before':max(p['reject_before'] for p in policies.values()),'policies':policies,'windows':{},'deliveries':{},'slices':{},'plans':{},'backups':{},'revoked_actors':[],'operation':None,'metadata':metadata(conn)}
        for part in conn.execute('SELECT * FROM hb_focus.slice').fetchall():
            rows=store.verify(part['batch_id'],part['tenant_id'],part)
            end=max((r['billing_end'].isoformat() for r in rows),default=now())
            state['slices'][str(part['batch_id'])+':'+str(part['tenant_id'])]={k:part[k] for k in ['row_count','row_digest','controls']}
            state['slices'][str(part['batch_id'])+':'+str(part['tenant_id'])].update({'tenant_id':str(part['tenant_id']),'batch_id':str(part['batch_id']),'window':None,'origin_received':None,'origin_window_end':end,'state':'hot','archive':None,'holds':{},'references':{},'deletion':None,'hierarchy_digest':part['metadata'].get('hierarchy_digest')})
        j.save(state)
        return {'state':'activated','installation':j.head['installation'],'retained_slices':len(state['slices']),'grain':grain}


def item_for(j,identity):
    item=j.state['slices'].get(identity)
    if not item:raise Rejected('slice_unavailable')
    return item


def policy_for(j,item):
    matches=[t['properties']['key'] for t in j.state['metadata']['tenant'] if t['tenant_id']==item['tenant_id']]
    if len(matches)!=1:raise Rejected('tenant_policy_unavailable')
    return j.state['policies'][matches[0]]


def remove_hot(store,item):
    ch=warehouse.client(store.env,'probe_admin')
    try:
        ch.command('ALTER TABLE hb_focus.fact DELETE WHERE tenant_id={t:UUID} AND batch_id={b:UUID} SETTINGS mutations_sync=2',parameters={'t':item['tenant_id'],'b':item['batch_id']})
    finally:ch.close()
    if store.rows(UUID(item['batch_id']),UUID(item['tenant_id'])):raise Rejected('warehouse_delete_unconfirmed')


def archive_slice(actor,identity,at=None):
    current=instant(at) if at else datetime.now(timezone.utc)
    with access(actor,'archive') as (j,conn,store,tenants):
        item=item_for(j,identity)
        if item['state']=='deleted':raise Rejected('period_expired_unavailable')
        if item['state']=='archived':return {'state':'archived','duplicate':True}
        policy=policy_for(j,item)
        origin=item['origin_received'] if policy['basis']=='received_at' else item['origin_window_end']
        if origin is None:raise Rejected('retention_origin_unknown')
        dates=eligible(origin,policy,current.isoformat())
        if current<instant(dates['hot_until']):raise Rejected('hot_retention_not_elapsed')
        rows=store.verify(UUID(item['batch_id']),UUID(item['tenant_id']),item)
        data=archive.encode(rows);sha=digest(data)
        key=f"archives/{item['tenant_id']}/{item['batch_id']}/{sha}.parquet"
        with j.effect('archive',identity):
            archive.put(j.s3,key,data);archive.decode(j.s3.get_object(Bucket=warehouse.BUCKET,Key=key)['Body'].read(),item)
            item['archive']={'key':key,'digest':sha,'bytes':len(data)}
            remove_hot(store,item);item['state']='archived'
        return {'state':'archived','row_count':len(rows),'sha256':sha,'physical_erasure':'not_claimed'}


def recall(actor,identity):
    with access(actor,'recall') as (j,conn,store,tenants):
        item=item_for(j,identity)
        if item['state']=='deleted':raise Rejected('period_expired_unavailable')
        if item['state']=='hot':return {'state':'hot','duplicate':True}
        rows=archive.read(j.s3,item)
        with j.effect('recall',identity):
            store.insert(UUID(item['batch_id']),rows);store.verify(UUID(item['batch_id']),UUID(item['tenant_id']),item)
            item['state']='hot'
        return {'state':'hot','row_count':len(rows),'retention_origin_unchanged':True}


def restriction(actor,identity,identifier,reason,kind='hold',release=False):
    if kind not in ['hold','reference'] or not isinstance(identifier,str) or not 1<=len(identifier)<=100 or not isinstance(reason,str) or not 1<=len(reason)<=500:raise Rejected('restriction_shape')
    with access(actor,kind) as (j,conn,store,tenants):
        item=item_for(j,identity)
        if item['state']=='deleted':raise Rejected('too_late_deleted')
        field='holds' if kind=='hold' else 'references'
        if release:item[field].pop(identifier,None)
        else:item[field][identifier]={'reason':reason,'actor':actor,'at':now()}
        j.save(j.state)
        return {'state':'released' if release else 'held','kind':kind,'identity':identity}


def preview_body(j,at,identities=None):
    targets=[];held=[]
    for identity,item in sorted(j.state['slices'].items()):
        if identities is not None and identity not in identities or item['state']=='deleted':continue
        policy=policy_for(j,item)
        origin=item['origin_received'] if policy['basis']=='received_at' else item['origin_window_end']
        reason=None
        if origin is None:reason='retention_origin_unknown'
        elif instant(at)<instant(eligible(origin,policy,at)['delete_after']):reason='retention_not_elapsed'
        elif item['holds']:reason='hold'
        elif item['references']:reason='dependency'
        value={'identity':identity,'item_digest':digest(canonical(item).encode()),'policy_digest':digest(canonical(policy).encode()),'row_count':item['row_count']}
        if reason:held.append({**value,'reason':reason})
        else:targets.append(value)
    return {'at':at,'targets':targets,'held':held,'config_digest':j.state['config_digest'],'restrictions_digest':digest(canonical({'slices':j.state['slices'],'policies':j.state['policies'],'backups':j.state['backups']}).encode())}


def preview(actor,at=None,identities=None):
    at=instant(at).isoformat() if at else now()
    with access(actor,'preview') as (j,conn,store,tenants):
        if identities is not None and any(i not in j.state['slices'] for i in identities):raise Rejected('slice_unavailable')
        body=preview_body(j,at,identities)
        sha=digest(canonical(body).encode())
        j.state['plans'][sha]={'kind':'delete','body':body,'requester':actor,'approver':None,'identities':identities,'state':'prepared'}
        j.save(j.state)
        return {'manifest_digest':sha,**body}


def approve(actor,sha):
    with access(actor,'approve') as (j,conn,store,tenants):
        plan=j.state['plans'].get(sha)
        if not plan or digest(canonical(plan['body']).encode())!=sha or plan['requester']==actor or plan['state']!='prepared':raise Rejected('approval_manifest')
        plan['approver']=actor;j.save(j.state)
        return {'state':'approved','manifest_digest':sha}


def object_absent(s3,key):
    try:s3.head_object(Bucket=warehouse.BUCKET,Key=key)
    except Exception as exc:
        code=getattr(exc,'response',{}).get('Error',{}).get('Code')
        if code in ['404','NoSuchKey','NotFound']:return True
        raise
    return False


def delete_key(s3,key):
    s3.delete_object(Bucket=warehouse.BUCKET,Key=key)
    if not object_absent(s3,key):raise Rejected('object_delete_unconfirmed')


def deletion_status(j,store,identity):
    item=item_for(j,identity)
    ch=warehouse.client(store.env,'probe_admin')
    try:
        inactive=ch.query("SELECT count() FROM system.parts WHERE database='hb_focus' AND table='fact' AND active=0").result_rows[0][0]
        mutations=ch.query("SELECT count() FROM system.mutations WHERE database='hb_focus' AND table='fact' AND NOT is_done").result_rows[0][0]
    finally:ch.close()
    backups=[key for key,b in j.state['backups'].items() if b['state']=='retained' and identity in b['slices']]
    try:
        versioned=j.s3.get_bucket_versioning(Bucket=warehouse.BUCKET).get('Status') in ['Enabled','Suspended']
        object_versions='unverified' if versioned else 'disabled'
    except Exception:object_versions='unverified'
    active_absent=not store.rows(UUID(item['batch_id']),UUID(item['tenant_id'])) if item['state']=='deleted' else False
    original_key=original_for(j,item['batch_id'])
    original_remaining=not object_absent(j.s3,original_key)
    return {'identity':identity,'state':item['state'],'active_rows_absent':active_absent,'clickhouse_mutations_pending':mutations,'clickhouse_inactive_parts':inactive,'object_versions':object_versions,'managed_backups':backups,'shared_original_remaining':original_remaining,'retained_shared_control_metadata':True,'policy_complete':False}


def original_for(j,batch):
    matches=[b for b in j.state['metadata']['batch'] if b['batch_id']==batch]
    if len(matches)!=1:raise Rejected('batch_metadata_unavailable')
    return 'shared-originals/'+batch+'/'+matches[0]['file_digest']+'.csv'


def delete(actor,sha,at=None):
    current=instant(at).isoformat() if at else now()
    with access(actor,'delete') as (j,conn,store,tenants):
        plan=j.state['plans'].get(sha)
        if not plan or plan['kind']!='delete' or not plan['approver'] or plan['state']!='prepared':raise Rejected('approval_required')
        if current<plan['body']['at']:raise Rejected('deletion_before_preview')
        body=preview_body(j,plan['body']['at'],plan['identities'])
        if digest(canonical(body).encode())!=sha:raise Rejected('deletion_manifest_changed')
        ids=[t['identity'] for t in body['targets']]
        with j.effect('delete',ids):
            for identity in ids:
                item=item_for(j,identity)
                item['deletion']={'manifest_digest':sha,'at':current,'approver':plan['approver'],'phase':'delete_committed'}
                item['state']='deleted'
            j.save(j.state)
            for identity in ids:
                item=item_for(j,identity)
                remove_hot(store,item)
                if item['archive']:delete_key(j.s3,item['archive']['key'])
                item['deletion']['phase']='active_deleted'
            for batch in {item_for(j,i)['batch_id'] for i in ids}:
                related=[(i,x) for i,x in j.state['slices'].items() if x['batch_id']==batch]
                backups=[b for b in j.state['backups'].values() if b['state']=='retained' and any(i in b['slices'] for i,x in related)]
                if all(x['state']=='deleted' and not x['holds'] and not x['references'] for i,x in related) and not backups:delete_key(j.s3,original_for(j,batch))
            for w in j.state['windows'].values():
                if w['current']:
                    batch=j.state['deliveries'][w['current']]['batch_id']
                    if all(x['state']=='deleted' for x in j.state['slices'].values() if x['batch_id']==batch):w['retired']=True
            plan['state']='executed'
        return {'manifest_digest':sha,'target':len(ids),'succeeded':len(ids),'held':len(body['held']),'failed':0,'pending':0,'stores':[deletion_status(j,store,i) for i in ids]}


def status(actor):
    with access(actor,'status') as (j,conn,store,tenants):
        if j.state.get('retired'):return {'installation':j.head['installation'],'state':'retired','receipt':j.state}
        return {'installation':j.head['installation'],'sequence':j.head['sequence'],'slices':copy.deepcopy(j.state['slices']),'policies':j.state['policies'],'windows':j.state['windows'],'deliveries':j.state['deliveries'],'backups':j.state['backups'],'deletions':[deletion_status(j,store,i) for i,x in j.state['slices'].items() if x['state']=='deleted']}


def backup(actor,at=None):
    at=instant(at).isoformat() if at else now()
    with access(actor,'backup') as (j,conn,store,tenants):
        identity=str(uuid4());parts={};originals={};until=[];deadlines={};excluded=[]
        for key,item in j.state['slices'].items():
            if item['state']=='deleted':continue
            p=policy_for(j,item);origin=item['origin_received'] if p['basis']=='received_at' else item['origin_window_end']
            if origin is None:raise Rejected('retention_origin_unknown')
            deadline=eligible(origin,p,at)['backup_until']
            if instant(at)>=instant(deadline):excluded.append(key);continue
            rows=archive.read(j.s3,item) if item['state']=='archived' else store.verify(UUID(item['batch_id']),UUID(item['tenant_id']),item)
            parts[key]=base64.b64encode(archive.encode(rows)).decode();until.append(deadline);deadlines[key]=deadline
            rawkey=original_for(j,item['batch_id'])
            if rawkey not in originals:originals[rawkey]=base64.b64encode(j.s3.get_object(Bucket=warehouse.BUCKET,Key=rawkey)['Body'].read()).decode()
        if not parts:raise Rejected('backup_retention_elapsed')
        body={'installation':j.head['installation'],'control_head':j.head,'metadata':metadata(conn),'slices':copy.deepcopy(j.state['slices']),'parts':parts,'originals':originals}
        data=canonical(body).encode()
        if len(data)>128*1024*1024:raise Rejected('backup_size')
        sha=digest(data);object_key=f'backups/{identity}/{sha}.json'
        with j.effect('backup',identity):
            archive.put(j.s3,object_key,data)
            j.state['backups'][identity]={'key':object_key,'digest':sha,'bytes':len(data),'created_at':at,'expires_at':max(until,default=at),'slices':list(parts),'slice_deadlines':deadlines,'excluded_expired':excluded,'shared_copy_residual':'until-last-slice-deadline','state':'retained'}
            j.state['metadata']=metadata(conn)
        return {'backup_id':identity,'sha256':sha,'slices':len(parts),'expires_at':j.state['backups'][identity]['expires_at']}


def restore_metadata(conn,records):
    conn.execute(sql.SQL('TRUNCATE {} CASCADE').format(sql.SQL(',').join(sql.Identifier('hb_focus',t) for t in METADATA_TABLES)))
    for table in METADATA_TABLES:
        rows=records.get(table,[])
        if table=='gate':rows=[{**r,'current_batch':None} for r in rows]
        for row in rows:
            conn.execute(sql.SQL('INSERT INTO {} SELECT * FROM json_populate_record(NULL::{},%s::json)').format(sql.Identifier('hb_focus',table),sql.Identifier('hb_focus',table)),(canonical(row),))
    for row in records['gate']:conn.execute('UPDATE hb_focus.gate SET current_batch=%s WHERE source=%s',(UUID(row['current_batch']) if row['current_batch'] else None,row['source']))


def validated_backup(j,backup_id,current):
    b=j.state['backups'].get(backup_id)
    if not b or b['state']!='retained':raise Rejected('backup_unavailable')
    if current>=b['expires_at']:raise Rejected('backup_horizon_elapsed')
    if any(current>=deadline and j.state['slices'][identity]['state']!='deleted' for identity,deadline in b['slice_deadlines'].items()):raise Rejected('backup_slice_horizon_elapsed')
    data=j.s3.get_object(Bucket=warehouse.BUCKET,Key=b['key'])['Body'].read()
    if digest(data)!=b['digest']:raise Rejected('backup_digest')
    body=json.loads(data)
    if body['installation']!=j.head['installation']:raise Rejected('installation_mismatch')
    for identity,encoded in body['parts'].items():archive.decode(base64.b64decode(encoded,validate=True),body['slices'][identity])
    return body


def restore(actor,backup_id,at=None):
    current=instant(at).isoformat() if at else now()
    with access(actor,'restore',repair=True,restore_request=(backup_id,current)) as (j,conn,store,tenants):
        body=j.restore_body
        with j.effect('restore',backup_id):
            cfg=j.state['metadata']['configuration'][0]['body']
            warehouse.initialize(store.env,cfg)
            restore_metadata(conn,j.state['metadata'])
            required_batches={x['batch_id'] for x in j.state['slices'].values() if x['state']!='deleted'}
            for batch in required_batches:
                rawkey=original_for(j,batch)
                expected=next(b['file_digest'] for b in j.state['metadata']['batch'] if b['batch_id']==batch)
                if object_absent(j.s3,rawkey):
                    if rawkey not in body['originals']:raise Rejected('backup_missing_original')
                    original=base64.b64decode(body['originals'][rawkey],validate=True)
                    if digest(original)!=expected:raise Rejected('backup_original_digest')
                    archive.put(j.s3,rawkey,original)
                if digest(j.s3.get_object(Bucket=warehouse.BUCKET,Key=rawkey)['Body'].read())!=expected:raise Rejected('original_digest')
            for actor_name in j.state['revoked_actors']:
                conn.execute("UPDATE hb_focus.tenant SET active=false WHERE properties->>'actor'=%s",(actor_name,))
                conn.execute('UPDATE hb_focus.audience SET active=false WHERE actor=%s',(actor_name,))
            for identity,item in j.state['slices'].items():
                if item['state']=='deleted':
                    remove_hot(store,item)
                    if item['archive'] and not object_absent(j.s3,item['archive']['key']):delete_key(j.s3,item['archive']['key'])
                    continue
                if item['state']=='archived':
                    if object_absent(j.s3,item['archive']['key']):
                        if identity not in body['parts']:raise Rejected('backup_missing_archive')
                        restored_bytes=base64.b64decode(body['parts'][identity],validate=True)
                        archive.decode(restored_bytes,item)
                        if digest(restored_bytes)!=item['archive']['digest']:raise Rejected('backup_archive_digest')
                        archive.put(j.s3,item['archive']['key'],restored_bytes)
                    archive.read(j.s3,item);continue
                try:store.verify(UUID(item['batch_id']),UUID(item['tenant_id']),item);continue
                except Rejected:pass
                if identity not in body['parts']:raise Rejected('backup_missing_current_dependency')
                rows=archive.decode(base64.b64decode(body['parts'][identity],validate=True),item)
                remove_hot(store,item);store.insert(UUID(item['batch_id']),rows)
                store.verify(UUID(item['batch_id']),UUID(item['tenant_id']),item)
            j.state['metadata']=metadata(conn)
        return {'state':'restored','backup_id':backup_id,'latest_controls_applied':True,'sequence':j.head['sequence']}


def prune_backups(actor,at=None):
    at=instant(at).isoformat() if at else now()
    with access(actor,'prune-backups') as (j,conn,store,tenants):
        expired=[key for key,b in j.state['backups'].items() if b['state']=='retained' and b['expires_at']<=at]
        with j.effect('prune-backups',expired):
            for key in expired:
                b=j.state['backups'][key];delete_key(j.s3,b['key']);b['state']='removed'
            for batch in {x['batch_id'] for x in j.state['slices'].values() if x['state']=='deleted'}:
                related=[(i,x) for i,x in j.state['slices'].items() if x['batch_id']==batch]
                retained=any(b['state']=='retained' and any(i in b['slices'] for i,x in related) for b in j.state['backups'].values())
                if all(x['state']=='deleted' for i,x in related) and not retained:delete_key(j.s3,original_for(j,batch))
        return {'removed_backups':expired}


def revoke(actor,target):
    with access(actor,'revoke') as (j,conn,store,tenants):
        readers={t['properties']['actor'] for t in tenants}|{r['actor'] for r in conn.execute('SELECT actor FROM hb_focus.audience')}
        if target not in readers:raise Rejected('reader_unavailable')
        if target not in j.state['revoked_actors']:j.state['revoked_actors'].append(target)
        j.save(j.state)
        conn.execute("UPDATE hb_focus.tenant SET active=false WHERE properties->>'actor'=%s",(target,))
        conn.execute('UPDATE hb_focus.audience SET active=false WHERE actor=%s',(target,))
        j.state['metadata']=metadata(conn);j.save(j.state)
        return {'state':'revoked','actor':target}


def policy_preview(actor,tenant,policy):
    validate_policy(policy)
    with access(actor,'policy-preview') as (j,conn,store,tenants):
        previous=j.state['policies'].get(tenant)
        if not previous or policy['revision']<=previous['revision'] or policy['reject_before']<previous['reject_before']:raise Rejected('policy_revision_or_floor')
        body={'tenant':tenant,'policy':policy,'predecessor':digest(canonical(previous).encode()),'impact':[i for i,x in j.state['slices'].items() if x['tenant_id']==str(tenant_id(tenant))],'restrictions_digest':digest(canonical(j.state['slices']).encode())}
        sha=digest(canonical(body).encode());j.state['plans'][sha]={'kind':'policy','body':body,'requester':actor,'approver':None,'state':'prepared'};j.save(j.state)
        return {'manifest_digest':sha,**body}


def policy_apply(actor,sha):
    with access(actor,'policy-apply') as (j,conn,store,tenants):
        plan=j.state['plans'].get(sha)
        if not plan or plan['kind']!='policy' or not plan['approver'] or plan['state']!='prepared':raise Rejected('approval_required')
        b=plan['body']
        if digest(canonical(j.state['policies'][b['tenant']]).encode())!=b['predecessor'] or digest(canonical(j.state['slices']).encode())!=b['restrictions_digest']:raise Rejected('policy_manifest_changed')
        j.state['policies'][b['tenant']]=b['policy'];plan['state']='executed'
        j.state['reject_before']=max(p['reject_before'] for p in j.state['policies'].values());j.save(j.state)
        return {'state':'policy_applied','revision':b['policy']['revision']}


def retirement_inventory(j,conn,store):
    expected={'head.json','authority.anchor'}|{f'{i:012d}.json' for i in range(j.head['first_sequence'],j.head['sequence']+1)}
    if {p.name for p in control.DIRECTORY.iterdir()}!=expected:raise Rejected('retirement_uninventoried_local_artifact')
    binding_files={str(p.relative_to(BINDINGS.parent)):digest(p.read_bytes()) for p in [BINDINGS,BINDINGS.with_suffix('.tmp')] if p.exists()}
    if any(x['state']!='deleted' or x['holds'] or x['references'] for x in j.state['slices'].values()):raise Rejected('retirement_retained_dependency')
    if any(b['state']=='retained' for b in j.state['backups'].values()):raise Rejected('retirement_backup_residual')
    if j.s3.get_bucket_versioning(Bucket=warehouse.BUCKET).get('Status') in ['Enabled','Suspended']:raise Rejected('retirement_object_versions')
    reader=warehouse.client(store.env)
    try:
        if reader.query('SELECT count() FROM hb_focus.fact').result_rows[0][0]:raise Rejected('retirement_active_rows')
    finally:reader.close()
    ch=warehouse.client(store.env,'probe_admin')
    try:
        for table,condition in [('parts','rows>0 OR active=0'),('detached_parts','1'),('mutations','NOT is_done')]:
            if ch.query(f"SELECT count() FROM system.{table} WHERE database='hb_focus' AND table='fact' AND ({condition})").result_rows[0][0]:raise Rejected('retirement_physical_parts_or_mutations')
    finally:ch.close()
    objects=set()
    for item in j.state['slices'].values():
        objects.add(original_for(j,item['batch_id']))
        if item['archive']:objects.add(item['archive']['key'])
    objects.update(b['key'] for b in j.state['backups'].values())
    if any(not object_absent(j.s3,key) for key in objects):raise Rejected('retirement_object_residual')
    business={k:v for k,v in j.state.items() if k not in ['plans','operation']}
    return {'installation':j.head['installation'],'source_digest':j.state['config_digest'],'business_digest':digest(canonical(business).encode()),'objects':sorted(objects),'tables':METADATA_TABLES,'journal_prefix':'control/'+j.head['installation']+'/','bindings':binding_files,'local_entries_through':j.head['sequence'],'control_digest':j.head['digest']}


def retire_preview(actor):
    with access(actor,'retire-preview') as (j,conn,store,tenants):
        body=retirement_inventory(j,conn,store);sha=digest(canonical(body).encode())
        j.state['plans'][sha]={'kind':'retire','body':body,'requester':actor,'approver':None,'state':'prepared'};j.save(j.state)
        return {'manifest_digest':sha,**body}


def retire(actor,sha):
    with access(actor,'retire') as (j,conn,store,tenants):
        plan=j.state['plans'].get(sha)
        if not plan or plan['kind']!='retire' or not plan['approver'] or plan['state']!='prepared':raise Rejected('approval_required')
        current=retirement_inventory(j,conn,store)
        if any(current[k]!=plan['body'][k] for k in ['installation','source_digest','business_digest','objects','tables','journal_prefix','bindings']):raise Rejected('retirement_manifest_changed')
        with j.effect('retire',sha):
            j.anchor('retirement_pending')
            purge_source(conn,store)
            with authority.connect(store.env) as verifier:
                if any(verifier.execute(sql.SQL('SELECT 1 FROM {} LIMIT 1').format(sql.Identifier('hb_focus',t))).fetchone() for t in METADATA_TABLES):raise Rejected('retirement_pg_payload')
            for path in [BINDINGS,BINDINGS.with_suffix('.tmp')]:
                if path.exists():path.unlink()
            fd=os.open(BINDINGS.parent,os.O_RDONLY)
            try:os.fsync(fd)
            finally:os.close(fd)
            if any(p.exists() for p in [BINDINGS,BINDINGS.with_suffix('.tmp')]):raise Rejected('retirement_bindings_residual')
            old_head=copy.deepcopy(j.head)
            minimal={'retired':True,'operation':None,'installation':old_head['installation'],'source_digest':plan['body']['source_digest'],'reject_before':'9999-12-31T00:00:00+00:00','approval_digest':sha,'cleanup_inventory_digest':digest(canonical(current).encode()),'prior_head':old_head['digest'],'outcome':'managed-payload-removed','media_erasure':'not-certified','unmanaged_copies':'outside-control'}
            sequence=old_head['sequence']+1
            entry={'installation':old_head['installation'],'sequence':sequence,'previous':old_head['digest'],'state':minimal}
            data=canonical(entry).encode();entry_sha=digest(data)
            control.durable(control.DIRECTORY/f'{sequence:012d}.json',data)
            key=f"control/{old_head['installation']}/{sequence:012d}-{entry_sha}.json"
            archive.put(j.s3,key,data)
            for i in range(old_head['first_sequence'],old_head['sequence']+1):
                path=control.DIRECTORY/f'{i:012d}.json'
                previous_data=path.read_bytes();old_key=f"control/{old_head['installation']}/{i:012d}-{digest(previous_data)}.json"
                delete_key(j.s3,old_key);path.unlink()
            head={'installation':old_head['installation'],'sequence':sequence,'digest':entry_sha,'first_sequence':sequence,'previous_anchor':old_head['digest']}
            control.durable(control.DIRECTORY/'head.json',canonical(head).encode())
            conn.execute('UPDATE hb_focus.control_checkpoint SET sequence=%s,digest=%s WHERE id=1',(sequence,entry_sha))
            j.head=head;j.state=minimal;j.anchor('retired')
        return {'state':'retired','receipt':j.state,'managed_payload_removed':True,'physical_media_erasure':False}


def purge_source(conn,store):
    with conn.transaction():
        conn.execute('SELECT source FROM hb_focus.gate ORDER BY source FOR UPDATE').fetchall()
        conn.execute('SELECT tenant_id FROM hb_focus.tenant ORDER BY tenant_id FOR UPDATE').fetchall()
        conn.execute(sql.SQL('TRUNCATE {} CASCADE').format(sql.SQL(',').join(sql.Identifier('hb_focus',t) for t in METADATA_TABLES)))
        ch=warehouse.client(store.env,'probe_admin')
        try:
            ch.command('TRUNCATE TABLE hb_focus.fact SYNC')
            if ch.query("SELECT count() FROM system.parts WHERE database='hb_focus' AND table='fact'").result_rows[0][0]:raise Rejected('retirement_ch_parts')
        finally:ch.close()
