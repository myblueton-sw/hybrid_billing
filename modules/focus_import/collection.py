import copy
from contextlib import contextmanager
from datetime import datetime, timezone
from uuid import UUID
from zoneinfo import ZoneInfo

from . import authority, archive, control, hierarchy_store, connections
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


def configure_connection(actor,body):
    with access(actor,'register') as (j,c,conn):
        cfg=conn.execute('SELECT * FROM hb_focus.configuration WHERE id=1').fetchone()
        result=connections.add(j.state,body,cfg['body']);j.save(j.state)
        return {'state':'registered','connection':result}


def register(actor,data,version,descriptor):
    with access(actor,'register') as (j,c,conn):
        cfg=conn.execute('SELECT * FROM hb_focus.configuration WHERE id=1').fetchone()
        connection=None;projection_error=None
        try:
            if isinstance(descriptor,dict) and 'connection_id' in descriptor:connection=connections.get(j.state,descriptor)
            if connection:scope_rows=connections.admit_source(data,version,cfg['body'],descriptor,connection)
        except Rejected as exc:
            from .receipts import record
            return record(j,data,version,descriptor,str(exc),cfg['body'])
        try:
            rows=validate_delivery(data,version,cfg['body'],descriptor,consumed=bool(connection and connection.get('semantic_definition')))
            if connection:connections.admission(rows,connection,descriptor,version)
        except Rejected as exc:
            if not connection:raise
            rows=scope_rows;projection_error=exc
        w=descriptor['window'];key=connections.identity(connection,w);state=j.state
        if connection is None:
            if w['grain']!=state['grain']:raise Rejected('source_granularity')
            if any(w['start']<c['end'] and c['start']<w['end'] for c in state.get('connections',{}).values()):raise Rejected('legacy_connection_overlap')
        if w['start']<state['reject_before']:raise Rejected('source_window_expired_hold')
        for existing in state['windows'].values():
            e=existing['window']
            if existing.get('connection_id')!=(connection['id'] if connection else None):continue
            if key!=connections.identity(connection,e) and w['start']<e['end'] and e['start']<w['end']:raise Rejected('overlapping_window')
        target=state['windows'].get(key)
        latest=target['revisions'][-1] if target else None
        if target and target.get('retired'):raise Rejected('source_window_expired_hold')
        receipt=digest(canonical({'version':version,'descriptor':descriptor}).encode())
        if receipt in state['deliveries']:return {'delivery_id':receipt,'duplicate':True}
        if latest and (descriptor['revision']<=state['deliveries'][latest]['descriptor']['revision'] or state['deliveries'][latest]['state']!='sealed'):raise Rejected('delivery_revision_or_pending')
        from . import receipts
        plans=[];predecessor=target['current'] if target else None
        if latest:
            previous=state['deliveries'][latest]['descriptor']
            regression=utc_time(descriptor['complete_through'])<utc_time(previous['complete_through']) or previous['finalized'] and not descriptor['finalized']
            if regression:
                if not connection:raise Rejected('reopening_requires_explicit_connection')
                plans.append(receipts.authorize(j,descriptor.get('reopen_plan'),'reopen',connection,descriptor,predecessor))
        closed_receipt=any(connection and q['connection_id']==connection['id'] and q.get('descriptor') and q['descriptor']['window']==w and q.get('intake_closed') for q in state.get('receipts',{}).values())
        if closed_receipt:raise Rejected('receipt_scope_closed')
        exhausted=any(connection and q['connection_id']==connection['id'] and q.get('descriptor') and q['descriptor']['window']==w and q.get('lineage_status')=='escalated' for q in state.get('receipts',{}).values())
        if exhausted:raise Rejected('replacement_escalated')
        pending_quarantine=[q for q in state.get('receipts',{}).values() if connection and q['connection_id']==connection['id'] and q.get('descriptor') and q['descriptor']['window']==w and q['state'] in ['quarantined','replacement-requested','escalated'] and (q['file_digest']!=digest(data) or receipts.bare_descriptor(q['descriptor'])!=receipts.bare_descriptor(descriptor))]
        if pending_quarantine and not descriptor.get('replacement_plan'):raise Rejected('replacement_approval_required')
        if descriptor.get('replacement_plan'):
            plans.append(receipts.authorize(j,descriptor['replacement_plan'],'replacement',connection,descriptor,predecessor))
        if connection:
            try:
                if projection_error:raise projection_error
                connections.semantic(rows,connection)
            except Rejected as exc:
                result=receipts.record(j,data,version,descriptor,str(exc),cfg['body'],connection,True,plans)
                return result
        if not target:
            origin=next((state['receipts'][p['body']['receipt_id']]['origin'] for p in plans if p['kind']=='replacement'),now())
            target={'window':w,'first_received_at':origin,'current':None,'revisions':[],'retired':False};state['windows'][key]=target
            if connection:target['connection_id']=connection['id']
        target['revisions'].append(receipt)
        state['deliveries'][receipt]={'descriptor':descriptor,'version':version,'mapping_digest':digest(canonical(versions()[version]).encode()),'state':'registered','batch_id':None,'transport_revision':None,'predecessor':target['current'],'received_at':target['first_received_at']}
        if connection:
            state['deliveries'][receipt]['semantic_definition']=connection.get('semantic_definition')
            state['deliveries'][receipt]['connection_digest']=digest(canonical(connection).encode())
        result={'delivery_id':receipt,'row_count':len(rows),'duplicate':False}
        for plan in plans:receipts.consume(j,plan,result)
        j.save(state)
        return result


def utc_time(value):
    from .temporal import utc
    return utc(value)


def collect(actor,delivery_id,data):
    with access(actor,'collect') as (j,c,conn):
        d=j.state['deliveries'].get(delivery_id)
        if not d or digest(data)!=d['descriptor']['file_digest']:raise Rejected('registered_delivery_required')
        if digest(canonical(versions().get(d['version'])).encode())!=d['mapping_digest']:raise Rejected('mapping_changed')
        if d['state'] in ['verified','sealed']:return {'delivery_id':delivery_id,'batch_id':d['batch_id'],'state':d['state'],'duplicate':True}
        with c.a.transaction('ingest') as (gate,cfg):
            validated=validate_delivery(data,d['version'],cfg['body'],d['descriptor'],consumed=bool(d.get('semantic_definition')))
            connection=connections.get(j.state,d['descriptor'])
            if connection:
                if d['connection_digest']!=digest(canonical(connection).encode()):raise Rejected('connection_changed')
                connections.admission(validated,connection,d['descriptor'],d['version']);connections.semantic(validated,connection)
            latest=conn.execute('SELECT max(source_revision) AS n FROM hb_focus.batch').fetchone()['n'] or 0
            d['transport_revision']=latest+1
        with j.effect('collect',delivery_id):
            c._authorized_delivery=d['descriptor']
            c._delivery_semantic=d.get('semantic_definition')
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
        target=j.state['windows'][connections.identity(connections.get(j.state,d['descriptor']),d['descriptor']['window'])]
        if target['current']!=d['predecessor']:raise Rejected('window_predecessor')
        with j.effect('publish',delivery_id):
            result=c.publish(d['batch_id']);target['current']=delivery_id;d['state']='sealed'
            from .lifecycle import metadata
            j.state['metadata']=metadata(conn)
        return {'delivery_id':delivery_id,**result}


def selected_slices(state,members,start,end):
    result=[];ownership=None
    for key,w in sorted(state['windows'].items()):
        connection_id=w.get('connection_id');visible=set(members)
        if connection_id is not None:
            connection=state.get('connections',{}).get(connection_id)
            if connection is None:raise Rejected('connection_unregistered')
            if ownership is None:
                configuration=state.get('metadata',{}).get('configuration',[])
                if len(configuration)!=1 or not isinstance(configuration[0].get('body'),dict):raise Rejected('connection_owner_unavailable')
                ownership=connections.owners(configuration[0]['body'])
            expected=set()
            for domain in connection['domains']:
                owner=ownership.get(canonical(domain))
                if owner is None:raise Rejected('connection_owner_unavailable')
                expected.add(str(tenant_id(owner['key'])))
            visible&=expected
        if not visible:continue
        if not w['current'] or start and w['window']['end']<=start or end and w['window']['start']>=end:continue
        d=state['deliveries'][w['current']]
        if d['descriptor'].get('connection_id')!=connection_id:raise Rejected('connection_window_identity')
        if d['state']!='sealed':raise Rejected('window_not_published')
        for tid in sorted(visible):
            item=state['slices'].get(d['batch_id']+':'+tid)
            if not item:raise Rejected('slice_missing')
            if item['state']=='deleted':raise Rejected('period_expired_unavailable')
            result.append((key,d,item))
    return result


def query(actor,start=None,end=None,grain='month',limit=20,cursor=None,expected_snapshot=None,provider=None,service=None,hierarchy_node=None,group_kind=None,group_axis=None,group_role='usage',*,_consumer=None):
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
            cfg=j.state['metadata']['configuration'][0]['body']
            expected_coverage=connections.coverage(j.state,members,cfg,start,end,now())
            scopes=selected_slices(j.state,members,start,end)
            pins=[{'window':k,'delivery':digest(canonical(d).encode()),'slice':digest(canonical(item).encode())} for k,d,item in scopes]
            snapshot=digest(canonical({'pins':pins,'expected_coverage':expected_coverage,'members':members,'actor':actor,'start':start,'end':end,'grain':grain,'provider':provider,'service':service,'hierarchy_node':hierarchy_node,'group_kind':group_kind,'group_axis':group_axis,'group_role':group_role}).encode())
            if expected_snapshot is not None and snapshot!=expected_snapshot:raise Rejected('query_snapshot_changed')
            if _consumer is not None:_consumer.preflight(j,c,context,scopes)
            all_rows=[];breakdown={};periods={};coverage=[];graphs={};contexts={};account_groups={};unresolved_rows=0;semantic_contexts={}
            for key,d,item in scopes:
                tid=item['tenant_id'];props=members[tid]
                if item['state']=='archived':rows=archive.read(j.s3,item)
                else:rows=c.store.verify(UUID(item['batch_id']),UUID(tid),item,reader=c.a.role)
                if any(str(r['tenant_id'])!=tid for r in rows):raise Rejected('slice_scope')
                graph_digest,graph=hierarchy_store.load(conn,tid,item['batch_id'],item.get('hierarchy_digest'))
                if not item.get('hierarchy_digest'):raise Rejected('hierarchy_not_normalized')
                if d.get('semantic_definition'):
                    from .semantics import context as semantic_context
                    for r in rows:semantic_contexts[(item['batch_id'],tid,r['ordinal'])]=semantic_context(r,d['version'],d['semantic_definition'])
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
            display=[{**serialized(row),'tenant_key':key,'window_id':wid,'batch_id':batch,'account_context':hierarchy_store.context(contexts[batch+':'+str(row['tenant_id'])],ordinal),'semantic_context':semantic_contexts.get((batch,str(row['tenant_id']),ordinal))} for wid,batch,ordinal,key,row in page]
            result={'expected_coverage':expected_coverage,'legacy_expectations':'operator-asserted-unverified','snapshot':snapshot,'account_hierarchies':graphs,'account_groups':account_groups,'unresolved_group_rows':unresolved_rows,'grouping_additive':True,'row_count':len(all_rows),'rows':display,'next_cursor':list(page[-1][:3]) if len(remaining)>limit else None,'groups':controls([r[4] for r in all_rows]),'tenant_breakdown':breakdown,'coverage':coverage,'currency_conversion':False,'retention_enforcement':True,'classification':'exploratory-query-only','range_basis':'delivery-window-overlap','period_attribution':'charge-start-in-tenant-timezone-no-proration'}
            return _consumer.consume(result,all_rows) if _consumer is not None else result
