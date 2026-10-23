from uuid import UUID

from psycopg.types.json import Jsonb

from .model import Rejected, canonical, digest


TABLES=['native_snapshot','native_node','native_edge','native_binding','native_binding_role','native_head']


def signature(body):return digest(canonical(body).encode())


def persist(conn,tenant,batch,body,replace=False):
    from .hierarchy_model import pinned_profile
    pinned_profile(body)
    tenant=UUID(str(tenant));batch=UUID(str(batch));sha=signature(body)
    existing=conn.execute('SELECT snapshot_digest FROM hb_focus.native_head WHERE tenant_id=%s AND batch_id=%s',(tenant,batch)).fetchone()
    if existing and existing['snapshot_digest']==sha:return sha
    if existing and not replace:raise Rejected('hierarchy_head_exists')
    saved=conn.execute('SELECT body FROM hb_focus.native_snapshot WHERE tenant_id=%s AND batch_id=%s AND snapshot_digest=%s',(tenant,batch,sha)).fetchone()
    scope=(tenant,batch,sha)
    if saved:
        if saved['body']!=body:raise Rejected('hierarchy_snapshot_collision')
    else:
        conn.execute('INSERT INTO hb_focus.native_snapshot VALUES (%s,%s,%s,%s)',(*scope,Jsonb(body)))
        for n in body['nodes']:
            conn.execute('INSERT INTO hb_focus.native_node VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)',(*scope,UUID(n['id']),n['provider'],n['partition'],n['namespace'],n['kind'],n['native_id'],Jsonb(n)))
        for e in body['edges']:
            conn.execute('INSERT INTO hb_focus.native_edge VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)',(*scope,e['id'],UUID(e['child']),UUID(e['parent']),e['relation'],e['axis'],Jsonb(e)))
        for b in body['bindings']:
            conn.execute('INSERT INTO hb_focus.native_binding VALUES (%s,%s,%s,%s,%s,%s,%s)',(*scope,b['ordinal'],UUID(b['focus_billing']),UUID(b['focus_usage']),Jsonb(b)))
            for role,node in b['native_roles'].items():conn.execute('INSERT INTO hb_focus.native_binding_role VALUES (%s,%s,%s,%s,%s,%s)',(*scope,b['ordinal'],role,UUID(node)))
    if existing:conn.execute('UPDATE hb_focus.native_head SET snapshot_digest=%s WHERE tenant_id=%s AND batch_id=%s',(sha,tenant,batch))
    else:conn.execute('INSERT INTO hb_focus.native_head VALUES (%s,%s,%s)',scope)
    return sha


def load(conn,tenant,batch,expected=None):
    row=conn.execute('SELECT s.* FROM hb_focus.native_head h JOIN hb_focus.native_snapshot s USING(tenant_id,batch_id,snapshot_digest) WHERE h.tenant_id=%s AND h.batch_id=%s',(UUID(str(tenant)),UUID(str(batch)))).fetchone()
    if not row:raise Rejected('hierarchy_snapshot_missing')
    if signature(row['body'])!=row['snapshot_digest'] or expected is not None and row['snapshot_digest']!=expected:raise Rejected('hierarchy_snapshot_changed')
    from .hierarchy_model import pinned_profile
    pinned_profile(row['body'])
    return row['snapshot_digest'],row['body']


def expected(journal,batch,tenant,fallback=None):
    if journal and journal.state:
        part=journal.state.get('slices',{}).get(str(batch)+':'+str(tenant))
        if part and part.get('hierarchy_digest'):return part['hierarchy_digest']
    return fallback


def view_body(body,rows):
    from .hierarchy_model import select
    ordinals={r['ordinal'] for r in rows}
    if not ordinals<={b['ordinal'] for b in body['bindings']}:raise Rejected('hierarchy_occurrence_missing')
    return select(body,ordinals)


def context(body,ordinal):
    matches=[b for b in body['bindings'] if b['ordinal']==ordinal]
    if len(matches)!=1:raise Rejected('hierarchy_occurrence_missing')
    return matches[0]


def query_signature(entries):return signature(sorted(entries))


def merge_groups(target,source):
    for node,value in source['groups'].items():
        groups=target.setdefault(node,{})
        for key,g in value.items():
            if key not in groups:
                groups[key]={'rows':g['rows'],'measures':{name:dict(v) for name,v in g['measures'].items()}}
                continue
            old=groups[key];old['rows']+=g['rows']
            for name,m in g['measures'].items():
                n=old['measures'][name];n['present']+=m['present'];n['missing']+=m['missing']
                if m['coefficient'] is not None:n['coefficient']=(n['coefficient'] or 0)+m['coefficient']


def verify_original(conn,tenant,batch,sha):
    row=conn.execute('SELECT body FROM hb_focus.native_snapshot WHERE tenant_id=%s AND batch_id=%s AND snapshot_digest=%s',(UUID(str(tenant)),UUID(str(batch)),sha)).fetchone()
    if not row or signature(row['body'])!=sha:raise Rejected('hierarchy_snapshot_changed')
    from .hierarchy_model import pinned_profile
    pinned_profile(row['body'])
