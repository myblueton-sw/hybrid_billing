from uuid import UUID
from zoneinfo import ZoneInfo

from . import hierarchy_store
from .model import Rejected, canonical, controls, digest, serialized


ALIASES={'Azure':'Microsoft','OCI':'Oracle','GCP':'Google Cloud','Alibaba':'Alibaba Cloud'}


def source_provider(value):
    return ALIASES.get(value,value)


def authorize_view(authority,gate):
    if gate['busy']:raise Rejected('operation_unresolved')
    conn=authority.conn
    audience=conn.execute('SELECT * FROM hb_focus.audience WHERE actor=%s FOR SHARE',(authority.actor,)).fetchone()
    if not audience or not audience['active'] or audience['source']!=gate['source'] or audience['config_digest']!=authority.bindings['config_digest']:
        raise Rejected('view_unavailable')
    members={r['tenant_id'] for r in conn.execute('SELECT tenant_id FROM hb_focus.audience_member WHERE actor=%s',(authority.actor,))}
    if not members or members!=set(audience['expected_tenants']):raise Rejected('view_members_changed')
    tenants=conn.execute('SELECT * FROM hb_focus.tenant WHERE tenant_id=ANY(%s) ORDER BY tenant_id FOR SHARE',(list(members),)).fetchall()
    if {t['tenant_id'] for t in tenants}!=members or any(not t['active'] or t['properties'].get('company_id')!=audience['company_id'] for t in tenants):raise Rejected('view_member_denied')
    return {'audience':audience,'tenants':tenants}


def query_view(coordinator,gate,context,limit,after,provider,service,expected_batch,expected_hierarchy=None):
    a=coordinator.a
    members={t['tenant_id']:t for t in context['tenants']}
    audience=context['audience']
    pubs=a.conn.execute('SELECT s.* FROM hb_focus.publication p JOIN hb_focus.slice s ON p.tenant_id=s.tenant_id AND p.batch_id=s.batch_id WHERE p.source=%s AND p.tenant_id=ANY(%s)',(gate['source'],list(members))).fetchall()
    base={'view_actor':a.actor,'company_id':audience['company_id'],'currency_conversion':False,'retention_enforcement':False,'coefficient_scale':18,'row_count':0,'groups':{},'rows':[],'tenant_breakdown':{}}
    if gate['current_batch'] is None and not pubs:return base
    if {p['tenant_id'] for p in pubs}!=set(members) or any(p['batch_id']!=gate['current_batch'] or p['metadata']['config_digest']!=audience['config_digest'] for p in pubs):raise Rejected('view_publication_mismatch')
    batch=str(gate['current_batch'])
    if expected_batch is not None and str(expected_batch)!=batch:raise Rejected('query_batch_changed')
    all_rows=[];meta=None;graphs={};graph_pins=[];contexts={}
    for pub in pubs:
        tid=pub['tenant_id'];props=members[tid]['properties']
        if digest(canonical(props).encode())!=pub['metadata']['properties_digest']:raise Rejected('tenant_properties_changed')
        common={k:pub['metadata'][k] for k in ['version','classification','mapping_digest','config_digest']}
        if meta is not None and meta!=common:raise Rejected('view_profile_mismatch')
        meta=common
        rows=coordinator.store.verify(pub['batch_id'],tid,pub,reader=a.role)
        selected=[r for r in rows if (provider is None or r['provider']==provider) and (service is None or r['service']==service)]
        graph_digest,graph=hierarchy_store.load(a.conn,tid,pub['batch_id'],hierarchy_store.expected(coordinator._journal,pub['batch_id'],tid,pub['metadata'].get('hierarchy_digest')))
        graph=hierarchy_store.view_body(graph,selected)
        graphs[str(pub['batch_id'])+':'+str(tid)]={'digest':graph_digest,**graph};contexts[tid]=graph
        graph_pins.append([str(tid),str(pub['batch_id']),graph_digest])
        base['tenant_breakdown'][props['key']]={'row_count':len(selected),'timezone':props['timezone'],'company_id':props['company_id'],'groups':controls(selected)}
        all_rows.extend(selected)
    if len({r['ordinal'] for r in all_rows})!=len(all_rows):raise Rejected('duplicate_view_ordinal')
    graph_pin=digest(canonical({'graphs':sorted(graph_pins),'provider':provider,'service':service,'actor':a.actor}).encode())
    if expected_hierarchy is not None and expected_hierarchy!=graph_pin:raise Rejected('query_hierarchy_changed')
    all_rows.sort(key=lambda r:r['ordinal'])
    page=[r for r in all_rows if r['ordinal']>after][:limit]
    output=[]
    for row in page:
        props=members[row['tenant_id']]['properties']
        item=serialized({k:v for k,v in row.items() if k!='tenant_id'})
        item['tenant_key']=props['key']
        item['account_context']=hierarchy_store.context(contexts[row['tenant_id']],row['ordinal'])
        item['billing_start_local']=row['billing_start'].astimezone(ZoneInfo(props['timezone'])).isoformat()
        item['billing_end_local']=row['billing_end'].astimezone(ZoneInfo(props['timezone'])).isoformat()
        output.append(item)
    return {**base,**(meta or {}),'hierarchy_snapshot':graph_pin,'account_hierarchies':graphs,'batch_id':batch,'row_count':len(all_rows),'groups':controls(all_rows),'rows':output,'next_after':page[-1]['ordinal'] if page and any(r['ordinal']>page[-1]['ordinal'] for r in all_rows) else None}
