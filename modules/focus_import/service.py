from datetime import datetime
from uuid import UUID, uuid4
from zoneinfo import ZoneInfo

from psycopg.types.json import Jsonb

from modules.focus_local.database import environment
from . import authority, warehouse, hierarchy_store
from .control import guarded, locked, Journal
from .intake import input_format
from .model import ADDITIVE, BINDINGS, MAX_BYTES, Rejected, assign, canonical, controls, decode, demo_config, digest, row_digest, serialized, tenant_id, validate_config, versions


def initialize(config):
    env=environment()
    with locked(),authority.connect(env) as conn:
        journal=Journal(env,conn);journal.ready()
        if journal.state and journal.state.get('retired'):raise Rejected('installation_retired')
        config=authority.initialize(config,env)
        warehouse.initialize(env,config)
    return {'state':'initialized','tenants':len(config['tenants']),'version':config['version'],'classification':'exploratory-query-only'}


class Coordinator:
    def __init__(self,actor):
        self.a=authority.Authority(actor)
        self.store=warehouse.Warehouse(self.a.env)

    def close(self):self.a.close()

    def mapping(self,version):
        m=versions().get(version)
        if m is None:raise Rejected('unsupported_version')
        return m

    def check(self,batch,cfg,gate):
        if batch['config_digest']!=cfg['digest'] or batch['mapping_digest']!=digest(canonical(self.mapping(batch['manifest']['version'])).encode()):raise Rejected('profile_changed')
        if batch['manifest_digest']!=digest(canonical(batch['manifest']).encode()):raise Rejected('manifest_digest')
        for tid,part in batch['manifest']['slices'].items():
            if part.get('hierarchy_digest'):hierarchy_store.verify_original(self.a.conn,tid,batch['batch_id'],part['hierarchy_digest'])
        current=str(gate['current_batch']) if gate['current_batch'] else None
        if batch['state']!='sealed' and current!=batch['manifest']['predecessor']:raise Rejected('publication_conflict')

    @guarded
    def ingest(self,data,version,revision,delivery=None):
        if type(revision) is not int or not 0<revision<2**63:raise Rejected('source_revision')
        a=self.a
        with a.transaction('ingest',allow_unfinished=True) as (gate,cfg):
            if delivery is not None and getattr(self,'_authorized_delivery',None) is not delivery:raise Rejected('registered_delivery_required')
            if delivery is None and version!=cfg['body']['version']:raise Rejected('configured_version')
            mapping=self.mapping(version)
            mapping_sha=digest(canonical(mapping).encode())
            sha=digest(data)
            previous=a.conn.execute('SELECT * FROM hb_focus.batch WHERE source=%s AND source_revision=%s',(gate['source'],revision)).fetchone()
            if previous:
                self.check(previous,cfg,gate)
                if previous['file_digest']!=sha:raise Rejected('revision_conflict')
                if previous['state'] not in ['verified','sealed']:raise Rejected('operation_unresolved')
                return self.result(previous,True)
            if gate['busy']:raise Rejected('operation_unresolved')
            if gate['current_batch']:
                prior=a.conn.execute('SELECT source_revision FROM hb_focus.batch WHERE batch_id=%s',(gate['current_batch'],)).fetchone()
                if revision<=prior['source_revision']:raise Rejected('stale_source_revision')
            if delivery is None:
                if sha!=cfg['body']['snapshot']['file_digest']:raise Rejected('snapshot_digest')
                rows=assign(decode(data,mapping),cfg['body'])
            else:
                from .temporal import validate_delivery
                rows=validate_delivery(data,version,cfg['body'],delivery,consumed=bool(getattr(self,'_delivery_semantic',None)))
            batch_id=uuid4()
            slices={};hierarchies={}
            from .hierarchy_model import snapshot
            catalog=self._journal.state.get('hierarchy_catalog') if self._journal and self._journal.state else None
            for tenant in cfg['body']['tenants']:
                tid=tenant_id(tenant['key'])
                selected=[r for r in rows if r['tenant_id']==tid]
                body=snapshot(selected,tenant['key'],gate['source'],catalog)
                hierarchies[str(tid)]=body
                slices[str(tid)]={'row_count':len(selected),'row_digest':row_digest(selected),'controls':controls(selected),'hierarchy_digest':hierarchy_store.signature(body)}
            manifest={'version':version,'classification':mapping['classification'],'mapping':mapping,'mapping_digest':mapping_sha,
                'config_digest':cfg['digest'],'numeric_profile':'exact-decimal38-scale18-bounded-absolute-sums','source':gate['source'],
                'source_revision':revision,'row_count':len(rows),'file_digest':sha,'file_bytes':len(data),'input_format':input_format(data),'slices':slices,
                'predecessor':str(gate['current_batch']) if gate['current_batch'] else None}
            semantic=getattr(self,'_delivery_semantic',None)
            if semantic:
                from .semantics import validate_rows
                validate_rows(rows,version,semantic)
                manifest['semantic_profile']=semantic
            manifest_sha=digest(canonical(manifest).encode())
            a.conn.execute('INSERT INTO hb_focus.batch VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)',(batch_id,gate['source'],revision,sha,cfg['digest'],mapping_sha,Jsonb(manifest),manifest_sha,'pending'))
            for tid,item in slices.items():
                metadata={'version':version,'classification':mapping['classification'],'config_digest':cfg['digest'],'mapping_digest':mapping_sha,'coefficient_scale':18,'hierarchy_digest':item['hierarchy_digest'],'properties_digest':digest(canonical(next(t for t in cfg['body']['tenants'] if tenant_id(t['key'])==UUID(tid))).encode())}
                a.conn.execute('INSERT INTO hb_focus.slice VALUES (%s,%s,%s,%s,%s,%s)',(UUID(tid),batch_id,item['row_count'],item['row_digest'],Jsonb(item['controls']),Jsonb(metadata)))
            for tid,body in hierarchies.items():hierarchy_store.persist(a.conn,tid,batch_id,body)
            a.conn.execute('UPDATE hb_focus.gate SET busy=true WHERE source=%s',(gate['source'],))
        try:
            with a.transaction('ingest',allow_unfinished=True) as (gate,cfg):
                batch=a.conn.execute('SELECT * FROM hb_focus.batch WHERE batch_id=%s',(batch_id,)).fetchone()
                self.check(batch,cfg,gate)
                self.store.original(batch_id,data)
                self.store.insert(batch_id,rows)
                for tid,item in slices.items():self.store.verify(batch_id,UUID(tid),item)
                self.check(batch,cfg,gate)
                a.conn.execute("UPDATE hb_focus.batch SET state='verified' WHERE batch_id=%s",(batch_id,))
            return self.result({**batch,'state':'verified'})
        except BaseException:
            try:
                with a.transaction('ingest',allow_unfinished=True):a.conn.execute("UPDATE hb_focus.batch SET state='unknown' WHERE batch_id=%s AND state='pending'",(batch_id,))
            except Exception:pass
            raise

    def batch(self,batch_id):
        try:batch_id=UUID(str(batch_id))
        except ValueError:raise Rejected('batch_identity') from None
        batch=self.a.conn.execute('SELECT * FROM hb_focus.batch WHERE batch_id=%s',(batch_id,)).fetchone()
        if not batch:raise Rejected('batch_unavailable')
        return batch

    @guarded
    def confirm(self,batch_id,manifest_digest):
        a=self.a
        with a.transaction('confirm',allow_unfinished=True) as (gate,cfg):
            batch=self.batch(batch_id);self.check(batch,cfg,gate)
            if batch['state'] not in ['verified','sealed'] or batch['manifest_digest']!=manifest_digest:raise Rejected('confirmation_digest_or_state')
            for tid,item in batch['manifest']['slices'].items():self.store.verify(batch['batch_id'],UUID(tid),item)
            a.conn.execute('INSERT INTO hb_focus.confirmation VALUES (%s,%s,%s) ON CONFLICT DO NOTHING',(batch['batch_id'],manifest_digest,a.role))
            return self.result(batch)

    @guarded
    def publish(self,batch_id):
        a=self.a
        with a.transaction('publish',allow_unfinished=True) as (gate,cfg):
            batch=self.batch(batch_id);self.check(batch,cfg,gate)
            if batch['state']=='sealed':return self.result(batch,True)
            confirmed=a.conn.execute('SELECT manifest_digest FROM hb_focus.confirmation WHERE batch_id=%s',(batch['batch_id'],)).fetchone()
            if batch['state']!='verified' or not confirmed or confirmed['manifest_digest']!=batch['manifest_digest']:raise Rejected('confirmation_required')
            for tid,item in batch['manifest']['slices'].items():self.store.verify(batch['batch_id'],UUID(tid),item)
            for tid in sorted(batch['manifest']['slices']):
                a.conn.execute('INSERT INTO hb_focus.publication VALUES (%s,%s,%s) ON CONFLICT (tenant_id,source) DO UPDATE SET batch_id=excluded.batch_id',(UUID(tid),gate['source'],batch['batch_id']))
            changed=a.conn.execute('UPDATE hb_focus.gate SET current_batch=%s,revision=revision+1,busy=false WHERE source=%s AND current_batch IS NOT DISTINCT FROM %s AND revision<9223372036854775807',(batch['batch_id'],gate['source'],gate['current_batch'])).rowcount
            if changed!=1:raise Rejected('publication_conflict')
            a.conn.execute("UPDATE hb_focus.batch SET state='sealed' WHERE batch_id=%s",(batch['batch_id'],))
            a.conn.execute('INSERT INTO hb_focus.decision VALUES (%s,%s,%s,%s)',(gate['source'],batch['batch_id'],gate['current_batch'],a.role))
            return self.result({**batch,'state':'sealed'})

    @guarded
    def query(self,limit=20,after=0,provider=None,service=None,expected_batch=None,expected_hierarchy=None):
        if type(limit) is not int or not 1<=limit<=100 or type(after) is not int or after<0:raise Rejected('query_page')
        if after and expected_batch is None:raise Rejected('query_batch_required')
        if after and expected_hierarchy is None:raise Rejected('query_hierarchy_required')
        from .views import query_view,source_provider
        provider=source_provider(provider)
        a=self.a
        with a.transaction('query') as (gate,tenant):
            if self._journal and self._journal.state:
                if self._journal.state['windows']:raise Rejected('period_query_required')
                members=[t['tenant_id'] for t in tenant['tenants']] if a.is_view else [a.tenant]
                for tid in members:
                    item=self._journal.state['slices'].get(str(gate['current_batch'])+':'+str(tid))
                    if item and item['state']!='hot':raise Rejected('period_query_required_or_expired')
            if a.is_view:return query_view(self,gate,tenant,limit,after,provider,service,expected_batch,expected_hierarchy)
            props=tenant['properties']
            result={'tenant':{k:props[k] for k in ['key','label','timezone','reporting_currency','retention_days']},
                'currency_conversion':False,'retention_enforcement':False,'coefficient_scale':18,'row_count':0,'groups':{},'rows':[]}
            pub=a.conn.execute('SELECT s.* FROM hb_focus.publication p JOIN hb_focus.slice s ON s.tenant_id=p.tenant_id AND s.batch_id=p.batch_id WHERE p.source=%s AND p.tenant_id=%s',(gate['source'],a.tenant)).fetchone()
            if not pub:return result
            if pub['batch_id']!=gate['current_batch']:raise Rejected('publication_pointer')
            if expected_batch is not None and str(pub['batch_id'])!=str(expected_batch):raise Rejected('query_batch_changed')
            if digest(canonical(props).encode())!=pub['metadata']['properties_digest']:raise Rejected('tenant_properties_changed')
            rows=self.store.verify(pub['batch_id'],a.tenant,pub,reader=True)
            selected=[r for r in rows if (provider is None or r['provider']==provider) and (service is None or r['service']==service)]
            graph_digest,graph=hierarchy_store.load(a.conn,a.tenant,pub['batch_id'],hierarchy_store.expected(self._journal,pub['batch_id'],a.tenant,pub['metadata'].get('hierarchy_digest')))
            graph=hierarchy_store.view_body(graph,selected)
            graph_pin=digest(canonical({'batch':str(pub['batch_id']),'tenant':str(a.tenant),'digest':graph_digest,'provider':provider,'service':service,'actor':a.actor}).encode())
            if expected_hierarchy is not None and expected_hierarchy!=graph_pin:raise Rejected('query_hierarchy_changed')
            page=[r for r in selected if r['ordinal']>after][:limit]
            display=[]
            for r in page:
                item=serialized({k:v for k,v in r.items() if k!='tenant_id'})
                item['account_context']=hierarchy_store.context(graph,r['ordinal'])
                item['billing_start_local']=r['billing_start'].astimezone(ZoneInfo(props['timezone'])).isoformat()
                item['billing_end_local']=r['billing_end'].astimezone(ZoneInfo(props['timezone'])).isoformat()
                display.append(item)
            return {**result,**pub['metadata'],'hierarchy_snapshot':graph_pin,'account_hierarchies':{str(pub['batch_id'])+':'+str(a.tenant):{'digest':graph_digest,**graph}},'batch_id':str(pub['batch_id']),'row_count':len(selected),'groups':controls(selected),'rows':display,'next_after':page[-1]['ordinal'] if page and any(r['ordinal']>page[-1]['ordinal'] for r in selected) else None}

    @staticmethod
    def result(batch,duplicate=False):
        return {'batch_id':str(batch['batch_id']),'state':batch['state'],'manifest_digest':batch['manifest_digest'],
            'row_count':batch['manifest']['row_count'],'version':batch['manifest']['version'],'input_format':batch['manifest'].get('input_format','csv'),'classification':batch['manifest']['classification'],'duplicate':duplicate}
