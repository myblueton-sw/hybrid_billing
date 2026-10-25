import copy
from uuid import UUID

from . import authority, hierarchy_store
from .model import Rejected, canonical, digest


def source_state(j,conn):
    from .hierarchy_model import registry_digest
    rows=conn.execute('SELECT tenant_id,batch_id,row_count,row_digest,metadata FROM hb_focus.slice ORDER BY tenant_id,batch_id').fetchall()
    heads=conn.execute('SELECT * FROM hb_focus.native_head ORDER BY tenant_id,batch_id').fetchall()
    return {'config_digest':j.state['config_digest'],'catalog_digest':digest(canonical(j.state.get('hierarchy_catalog')).encode()),'registry_digest':registry_digest(),'parts':[{**{k:str(r[k]) for k in ['tenant_id','batch_id']},'row_digest':r['row_digest']} for r in rows],'heads':[{k:str(v) for k,v in r.items()} for r in heads],'lifecycle_digest':digest(canonical(j.state['slices']).encode())}


def prepare(actor,catalog=None):
    from .hierarchy_model import validate_catalog
    from .lifecycle import access
    with access(actor,'hierarchy-plan') as (j,conn,store,tenants):
        cfg=conn.execute('SELECT body FROM hb_focus.configuration WHERE id=1').fetchone()['body']
        if catalog is not None:
            catalog=validate_catalog(catalog,cfg)
            old=j.state.get('hierarchy_catalog')
            if old and catalog['revision']<=old['revision']:raise Rejected('hierarchy_catalog_revision')
        elif j.state.get('hierarchy_catalog') is not None:catalog=j.state['hierarchy_catalog']
        body={'catalog':catalog,'predecessor':source_state(j,conn)}
        sha=digest(canonical(body).encode());j.state['plans'][sha]={'kind':'hierarchy','body':body,'requester':actor,'approver':None,'state':'prepared'};j.save(j.state)
        return {'manifest_digest':sha,'catalog_revision':catalog['revision'] if catalog else None,'mode':'approved-source-metadata' if catalog else 'observed-focus-only'}


def apply(actor,sha):
    from .hierarchy_model import snapshot
    from .lifecycle import access, metadata
    from . import archive
    with access(actor,'hierarchy-apply') as (j,conn,store,tenants):
        plan=j.state['plans'].get(sha)
        if not plan or plan['kind']!='hierarchy' or not plan['approver'] or plan['state']!='prepared' or digest(canonical(plan['body']).encode())!=sha:raise Rejected('hierarchy_approval_required')
        if plan['body']['predecessor']!=source_state(j,conn):raise Rejected('hierarchy_plan_changed')
        catalog=plan['body']['catalog'];cfg=conn.execute('SELECT body FROM hb_focus.configuration WHERE id=1').fetchone()['body']
        keys={str(t['tenant_id']):t['properties']['key'] for t in tenants}
        targets=[];frozen=0
        for identity,item in sorted(j.state['slices'].items()):
            if item['state']=='deleted':continue
            tid=item['tenant_id'];batch=item['batch_id']
            head=conn.execute('SELECT snapshot_digest FROM hb_focus.native_head WHERE tenant_id=%s AND batch_id=%s',(UUID(tid),UUID(batch))).fetchone()
            previous=None
            if head:
                old_sha,previous=hierarchy_store.load(conn,tid,batch,item.get('hierarchy_digest'))
                if previous['status']!='focus-only':frozen+=1;continue
            rows=archive.read(j.s3,item) if item['state']=='archived' else store.verify(UUID(batch),UUID(tid),item)
            body=snapshot(rows,keys[tid],cfg['source'],catalog)
            if previous is not None and body['status']=='focus-only':
                item['hierarchy_digest']=old_sha;continue
            if previous is not None:
                body={**body,'supersedes':old_sha,'attribution_mode':'supplemental-after-intake'}
            targets.append((identity,body))
        with j.effect('hierarchy-apply',sha):
            for identity,body in targets:
                item=j.state['slices'][identity]
                item['hierarchy_digest']=hierarchy_store.persist(conn,item['tenant_id'],item['batch_id'],body,replace=True)
            j.state['hierarchy_catalog']=copy.deepcopy(catalog)
            plan['state']='executed';j.state['metadata']=metadata(conn)
        return {'state':'normalized','updated_slices':len(targets),'frozen_typed_slices':frozen,'catalog_revision':catalog['revision'] if catalog else None,'profiles':'six-provider-typed-registry','native_parent_inference':False}
