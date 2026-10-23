import copy
from contextlib import contextmanager
from datetime import datetime, timezone
from uuid import UUID
from zoneinfo import ZoneInfo

from . import authority, archive, control, hierarchy_store
from .model import ADDITIVE, MAX_ROWS, Rejected, canonical, coefficient, controls, digest, serialized, tenant_id, versions
from .service import Coordinator
from .temporal import summarize, validate_delivery, window
from .views import source_provider


def now():return datetime.now(timezone.utc).isoformat()


@contextmanager
def access(actor,right):
    allowed={'register':{'operator'},'collect':{'importer'},'confirm':{'approver'},'publish':{'approver'},'query':None}
    if right not in allowed or allowed[right] is not None and actor not in allowed[right]:raise Rejected('permission')
    with control.locked():
        if actor=='operator':
            from modules.focus_local.database import environment
            env=environment();conn=authority.connect(env);coordinator=None
        else:
            coordinator=Coordinator(actor);env=coordinator.a.env;conn=coordinator.a.conn
        try:
            journal=control.Journal(env,conn)
            state=journal.ready()
            if state is None:raise Rejected('lifecycle_not_activated')
            if actor in state['revoked_actors']:raise Rejected('reader_revoked')
            if coordinator:
                coordinator._control_nested=True;coordinator._journal=journal
            yield journal,coordinator,conn
        finally:
            if coordinator:coordinator.close()
            else:conn.close()


def register(actor,data,version,descriptor):
    with access(actor,'register') as (j,c,conn):
        cfg=conn.execute('SELECT * FROM hb_focus.configuration WHERE id=1').fetchone()
        rows=validate_delivery(data,version,cfg['body'],descriptor)
        w=descriptor['window'];key=w['key'];state=j.state
        if w['grain']!=state['grain']:raise Rejected('source_granularity')
        if w['start']<state['reject_before']:raise Rejected('source_window_expired_hold')
        for existing in state['windows'].values():
            e=existing['window']
            if e['key']!=key and w['start']<e['end'] and e['start']<w['end']:raise Rejected('overlapping_window')
        target=state['windows'].get(key)
        latest=target['revisions'][-1] if target else None
        if target and target.get('retired'):raise Rejected('source_window_expired_hold')
        receipt=digest(canonical({'version':version,'descriptor':descriptor}).encode())
        if receipt in state['deliveries']:return {'delivery_id':receipt,'duplicate':True}
        if latest and (descriptor['revision']<=state['deliveries'][latest]['descriptor']['revision'] or state['deliveries'][latest]['state']!='sealed'):raise Rejected('delivery_revision_or_pending')
        if not target:
            target={'window':w,'first_received_at':now(),'current':None,'revisions':[],'retired':False};state['windows'][key]=target
        target['revisions'].append(receipt)
        state['deliveries'][receipt]={'descriptor':descriptor,'version':version,'mapping_digest':digest(canonical(versions()[version]).encode()),'state':'registered','batch_id':None,'transport_revision':None,'predecessor':target['current'],'received_at':target['first_received_at']}
        j.save(state)
        return {'delivery_id':receipt,'row_count':len(rows),'duplicate':False}


def collect(actor,delivery_id,data):
    with access(actor,'collect') as (j,c,conn):
        d=j.state['deliveries'].get(delivery_id)
        if not d or digest(data)!=d['descriptor']['file_digest']:raise Rejected('registered_delivery_required')
        if digest(canonical(versions().get(d['version'])).encode())!=d['mapping_digest']:raise Rejected('mapping_changed')
        if d['state'] in ['verified','sealed']:return {'delivery_id':delivery_id,'batch_id':d['batch_id'],'state':d['state'],'duplicate':True}
        with c.a.transaction('ingest') as (gate,cfg):
            validate_delivery(data,d['version'],cfg['body'],d['descriptor'])
            latest=conn.execute('SELECT max(source_revision) AS n FROM hb_focus.batch').fetchone()['n'] or 0
            d['transport_revision']=latest+1
        with j.effect('collect',delivery_id):
            c._authorized_delivery=d['descriptor']
            result=c.ingest(data,d['version'],d['transport_revision'],delivery=d['descriptor'])
            d['batch_id']=result['batch_id'];d['manifest_digest']=result['manifest_digest'];d['state']='verified'
            for tid,part in conn.execute('SELECT manifest FROM hb_focus.batch WHERE batch_id=%s',(UUID(result['batch_id']),)).fetchone()['manifest']['slices'].items():
                identity=result['batch_id']+':'+tid
                j.state['slices'][identity]={**part,'tenant_id':tid,'batch_id':result['batch_id'],'window':d['descriptor']['window'],'origin_received':d['received_at'],'origin_window_end':d['descriptor']['window']['end'],'state':'hot','archive':None,'holds':{},'references':{},'deletion':None}
            from .lifecycle import metadata
            j.state['metadata']=metadata(conn)
        return {'delivery_id':delivery_id,**result}


def confirm(actor,delivery_id,manifest_digest):
    with access(actor,'confirm') as (j,c,conn):
        d=j.state['deliveries'].get(delivery_id)
        if not d or not d['batch_id']:raise Rejected('delivery_unavailable')
        result=c.confirm(d['batch_id'],manifest_digest)
        from .lifecycle import metadata
        j.state['metadata']=metadata(conn);j.save(j.state)
        return result


def publish(actor,delivery_id):
    with access(actor,'publish') as (j,c,conn):
        d=j.state['deliveries'].get(delivery_id)
        if not d or d['state'] not in ['verified','sealed']:raise Rejected('delivery_unavailable')
        if d['state']=='sealed':return {'delivery_id':delivery_id,'state':'sealed','duplicate':True}
        target=j.state['windows'][d['descriptor']['window']['key']]
        if target['current']!=d['predecessor']:raise Rejected('window_predecessor')
        with j.effect('publish',delivery_id):
            result=c.publish(d['batch_id']);target['current']=delivery_id;d['state']='sealed'
            from .lifecycle import metadata
            j.state['metadata']=metadata(conn)
        return {'delivery_id':delivery_id,**result}


def selected_slices(state,members,start,end):
    result=[]
    for key,w in sorted(state['windows'].items()):
        if not w['current'] or start and w['window']['end']<=start or end and w['window']['start']>=end:continue
        d=state['deliveries'][w['current']]
        if d['state']!='sealed':raise Rejected('window_not_published')
        for tid in sorted(members):
            item=state['slices'].get(d['batch_id']+':'+tid)
            if not item:raise Rejected('slice_missing')
            if item['state']=='deleted':raise Rejected('period_expired_unavailable')
            result.append((key,d,item))
    return result


def query(actor,start=None,end=None,grain='month',limit=20,cursor=None,expected_snapshot=None,provider=None,service=None,hierarchy_node=None,group_kind=None,group_axis=None,group_role='usage'):
    if type(limit) is not int or not 1<=limit<=100 or grain not in ['day','month']:raise Rejected('query_page')
    if hierarchy_node is not None:
        try:hierarchy_node=str(UUID(str(hierarchy_node)))
        except ValueError:raise Rejected('hierarchy_node_identity') from None
    if (group_kind is None)!=(group_axis is None) or group_role not in ['billing','usage','resource','identity']:raise Rejected('hierarchy_group_options')
    if cursor is not None and expected_snapshot is None:raise Rejected('query_snapshot_required')
    from .model import instant
    if start is not None:start=instant(start).isoformat()
    if end is not None:end=instant(end).isoformat()
    if start and end and start>=end:raise Rejected('query_interval')
    provider=source_provider(provider)
    with access(actor,'query') as (j,c,conn):
        with c.a.transaction('query') as (gate,context):
            if c.a.is_view:members={str(t['tenant_id']):t['properties'] for t in context['tenants']}
            else:members={str(c.a.tenant):context['properties']}
            scopes=selected_slices(j.state,members,start,end)
            pins=[{'window':k,'delivery':digest(canonical(d).encode()),'slice':digest(canonical(item).encode())} for k,d,item in scopes]
            snapshot=digest(canonical({'pins':pins,'members':members,'actor':actor,'start':start,'end':end,'grain':grain,'provider':provider,'service':service,'hierarchy_node':hierarchy_node,'group_kind':group_kind,'group_axis':group_axis,'group_role':group_role}).encode())
            if expected_snapshot is not None and snapshot!=expected_snapshot:raise Rejected('query_snapshot_changed')
            all_rows=[];breakdown={};periods={};coverage=[];graphs={};contexts={};account_groups={};unresolved_rows=0
            for key,d,item in scopes:
                tid=item['tenant_id'];props=members[tid]
                if item['state']=='archived':rows=archive.read(j.s3,item)
                else:rows=c.store.verify(UUID(item['batch_id']),UUID(tid),item,reader=c.a.role)
                if any(str(r['tenant_id'])!=tid for r in rows):raise Rejected('slice_scope')
                graph_digest,graph=hierarchy_store.load(conn,tid,item['batch_id'],item.get('hierarchy_digest'))
                if not item.get('hierarchy_digest'):raise Rejected('hierarchy_not_normalized')
                rows=[r for r in rows if (provider is None or r['provider']==provider) and (service is None or r['service']==service)]
                if hierarchy_node is not None:rows=[r for r in rows if hierarchy_node in hierarchy_store.context(graph,r['ordinal'])['visible_nodes']]
                graph=hierarchy_store.view_body(graph,rows)
                graph_key=item['batch_id']+':'+tid
                graphs[graph_key]={'digest':graph_digest,**graph};contexts[graph_key]=graph
                if group_kind is not None:
                    from .hierarchy_model import group_rows
                    grouped=group_rows(rows,graph,group_kind,group_axis,group_role)
                    hierarchy_store.merge_groups(account_groups,grouped);unresolved_rows+=grouped['unresolved_rows']
                periods.setdefault(props['key'],[]).extend(rows)
                for row in rows:all_rows.append((key,item['batch_id'],row['ordinal'],props['key'],row))
                coverage.append({'window':d['descriptor']['window'],'complete_through':d['descriptor']['complete_through'],'finalized':d['descriptor']['finalized'],'tenant_key':props['key'],'state':item['state']})
            if len(all_rows)>MAX_ROWS or any(sum(abs(coefficient(r[4][n])) for r in all_rows if r[4][n] is not None)>=10**38 for n in ADDITIVE):raise Rejected('selection_bound')
            for tid,props in members.items():
                rows=periods.get(props['key'],[])
                breakdown[props['key']]={'row_count':len(rows),'groups':controls(rows),'periods':summarize(rows,props['timezone'],grain),'timezone':props['timezone']}
            all_rows.sort(key=lambda r:r[:3])
            if cursor is not None and (not isinstance(cursor,list) or len(cursor)!=3 or not isinstance(cursor[0],str) or not isinstance(cursor[1],str) or type(cursor[2]) is not int):raise Rejected('query_cursor')
            remaining=[r for r in all_rows if cursor is None or r[:3]>tuple(cursor)]
            page=remaining[:limit]
            display=[{**serialized(row),'tenant_key':key,'window_id':wid,'batch_id':batch,'account_context':hierarchy_store.context(contexts[batch+':'+str(row['tenant_id'])],ordinal)} for wid,batch,ordinal,key,row in page]
            return {'snapshot':snapshot,'account_hierarchies':graphs,'account_groups':account_groups,'unresolved_group_rows':unresolved_rows,'grouping_additive':True,'row_count':len(all_rows),'rows':display,'next_cursor':list(page[-1][:3]) if len(remaining)>limit else None,'groups':controls([r[4] for r in all_rows]),'tenant_breakdown':breakdown,'coverage':coverage,'currency_conversion':False,'retention_enforcement':True,'classification':'exploratory-query-only','range_basis':'delivery-window-overlap','period_attribution':'charge-start-in-tenant-timezone-no-proration'}
