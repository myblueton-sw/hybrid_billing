import json
import re
from contextlib import contextmanager
from pathlib import Path
from uuid import UUID

import psycopg
from psycopg import sql
from psycopg.rows import dict_row
from psycopg.types.json import Jsonb

from modules.focus_local.database import environment, password
from .model import BINDINGS, ROOT, Rejected, canonical, digest, tenant_id, validate_config


IMPORTER='hb_focus_ingest'
APPROVER='hb_focus_approve'
TABLES=['configuration','tenant','gate','batch','slice','publication','confirmation','decision']


def reader_role(tenant):
    return 'hb_focus_r_'+UUID(str(tenant)).hex


def connect(env,user=None):
    return psycopg.connect(host='127.0.0.1',port=15432,dbname='hb',user=user or env['PG_ADMIN_USER'],
        password=password(env,user,'PG_ADMIN_PASSWORD') if user else env['PG_ADMIN_PASSWORD'],
        autocommit=True,row_factory=dict_row,connect_timeout=5)


def initialize(config,env=None):
    config=validate_config(config)
    env=env or environment()
    sha=digest(canonical(config).encode())
    migration=(ROOT/'migrations/focus_import/001-authority.sql').read_bytes()
    roles={IMPORTER:None,APPROVER:None,**{reader_role(tenant_id(t['key'])):tenant_id(t['key']) for t in config['tenants']}}
    with connect(env) as conn,conn.transaction():
        conn.execute('SELECT pg_advisory_xact_lock(270002)')
        if not conn.execute("SELECT to_regclass('hb_focus.migration') AS value").fetchone()['value']:
            conn.execute(migration.decode())
            conn.execute('INSERT INTO hb_focus.migration VALUES (1,%s)',(digest(migration),))
        if conn.execute('SELECT digest FROM hb_focus.migration WHERE version=1').fetchone()['digest']!=digest(migration):raise Rejected('migration_digest')
        saved=conn.execute('SELECT digest FROM hb_focus.configuration').fetchone()
        if saved and saved['digest']!=sha:raise Rejected('configuration_changed')
        if not saved:
            conn.execute('INSERT INTO hb_focus.configuration VALUES (1,%s,%s)',(sha,Jsonb(config)))
            conn.execute('INSERT INTO hb_focus.gate(source) VALUES (%s)',(config['source'],))
            for t in config['tenants']:conn.execute('INSERT INTO hb_focus.tenant(tenant_id,properties,active) VALUES (%s,%s,%s)',(tenant_id(t['key']),Jsonb(t),t['active']))
        for role,tenant in roles.items():
            if not conn.execute('SELECT 1 FROM pg_roles WHERE rolname=%s',(role,)).fetchone():
                conn.execute(sql.SQL('CREATE ROLE {} LOGIN NOSUPERUSER NOBYPASSRLS PASSWORD {}').format(sql.Identifier(role),sql.Literal(password(env,role,'PG_ADMIN_PASSWORD'))))
            attributes=conn.execute('SELECT oid,rolsuper,rolbypassrls,rolcreaterole,rolcreatedb,rolreplication FROM pg_roles WHERE rolname=%s',(role,)).fetchone()
            if any(attributes[k] for k in ['rolsuper','rolbypassrls','rolcreaterole','rolcreatedb','rolreplication']) or conn.execute('SELECT 1 FROM pg_auth_members WHERE member=%s',(attributes['oid'],)).fetchone() or conn.execute('SELECT 1 FROM pg_class WHERE relowner=%s',(attributes['oid'],)).fetchone():raise Rejected('incompatible_pg_role')
            conn.execute(sql.SQL('GRANT USAGE ON SCHEMA hb_focus TO {}').format(sql.Identifier(role)))
            for table in TABLES:
                ident=sql.Identifier('hb_focus',table)
                conn.execute(sql.SQL('ALTER TABLE {} ENABLE ROW LEVEL SECURITY').format(ident))
                conn.execute(sql.SQL('ALTER TABLE {} FORCE ROW LEVEL SECURITY').format(ident))
                policy='p_'+role
                allowed=tenant is None or table=='gate' or table in ['tenant','slice','publication']
                if not allowed:continue
                condition=sql.SQL('true') if tenant is None or table=='gate' else sql.SQL('tenant_id={}').format(sql.Literal(tenant))
                conn.execute(sql.SQL('DROP POLICY IF EXISTS {} ON {}').format(sql.Identifier(policy),ident))
                conn.execute(sql.SQL('CREATE POLICY {} ON {} TO {} USING ({}) WITH CHECK ({})').format(sql.Identifier(policy),ident,sql.Identifier(role),condition,condition))
                conn.execute(sql.SQL('GRANT SELECT ON {} TO {}').format(ident,sql.Identifier(role)))
            conn.execute(sql.SQL('GRANT UPDATE(lock_token) ON hb_focus.gate,hb_focus.tenant TO {}').format(sql.Identifier(role)))
        conn.execute('GRANT INSERT ON hb_focus.batch,hb_focus.slice TO hb_focus_ingest')
        conn.execute('GRANT UPDATE(state) ON hb_focus.batch TO hb_focus_ingest,hb_focus_approve')
        conn.execute('GRANT INSERT ON hb_focus.confirmation,hb_focus.decision,hb_focus.publication TO hb_focus_approve')
        conn.execute('GRANT UPDATE(batch_id) ON hb_focus.publication TO hb_focus_approve')
        conn.execute('GRANT UPDATE(current_batch,revision,busy) ON hb_focus.gate TO hb_focus_approve')
        conn.execute('GRANT UPDATE(busy) ON hb_focus.gate TO hb_focus_ingest')
        for role,tenant in roles.items():verify_privileges(conn,role,tenant)
        expected={t:{'p_'+r for r,tid in roles.items() if tid is None or t=='gate' or t in ['tenant','slice','publication']} for t in TABLES}
        for table in TABLES:
            policies={r['policyname'] for r in conn.execute('SELECT policyname FROM pg_policies WHERE schemaname=%s AND tablename=%s',('hb_focus',table))}
            if policies!=expected[table]:raise Rejected('incompatible_pg_policy')
    bindings={'config_digest':sha,'source':config['source'],'actors':{t['actor']:str(tenant_id(t['key'])) for t in config['tenants']}}
    BINDINGS.parent.mkdir(exist_ok=True)
    temporary=BINDINGS.with_suffix('.tmp')
    temporary.write_text(canonical(bindings))
    temporary.chmod(0o600)
    temporary.replace(BINDINGS)
    return config


def verify_privileges(conn,role,tenant):
    inserts={'batch','slice'} if role==IMPORTER else {'confirmation','decision','publication'} if role==APPROVER else set()
    updates={'gate':{'lock_token'},'tenant':{'lock_token'}}
    if tenant is None:
        updates['batch']={'state'}
        updates['gate'].add('busy')
    if role==APPROVER:updates.update({'publication':{'batch_id'},'gate':{'lock_token','current_batch','revision','busy'}})
    if conn.execute("SELECT has_schema_privilege(%s,'hb_focus','CREATE') AS v",(role,)).fetchone()['v']:raise Rejected('incompatible_pg_privileges')
    for table in TABLES+['migration']:
        name='hb_focus.'+table
        for privilege in ['DELETE','TRUNCATE','TRIGGER','REFERENCES']:
            if conn.execute('SELECT has_table_privilege(%s,%s,%s) AS v',(role,name,privilege)).fetchone()['v']:raise Rejected('incompatible_pg_privileges')
        if tenant is not None and table not in ['tenant','slice','publication','gate'] and conn.execute("SELECT has_table_privilege(%s,%s,'SELECT') AS v",(role,name)).fetchone()['v']:raise Rejected('incompatible_pg_privileges')
        for col in conn.execute("SELECT column_name FROM information_schema.columns WHERE table_schema='hb_focus' AND table_name=%s",(table,)):
            for privilege,allowed in [('INSERT',table in inserts),('UPDATE',col['column_name'] in updates.get(table,set()))]:
                if not allowed and conn.execute('SELECT has_column_privilege(%s,%s,%s,%s) AS v',(role,name,col['column_name'],privilege)).fetchone()['v']:raise Rejected('incompatible_pg_privileges')
    if conn.execute("SELECT 1 FROM information_schema.role_table_grants WHERE grantee=%s AND table_schema<>'hb_focus'",(role,)).fetchone():raise Rejected('incompatible_pg_privileges')


class Authority:
    def __init__(self,actor,env=None):
        self.env=env or environment()
        self.bindings=json.loads(BINDINGS.read_text())
        self.actor=actor
        self.tenant=None
        if actor in ['importer','approver']:self.role=IMPORTER if actor=='importer' else APPROVER
        elif actor in self.bindings['actors']:
            self.tenant=UUID(self.bindings['actors'][actor]);self.role=reader_role(self.tenant)
        else:raise Rejected('unknown_local_actor')
        self.conn=connect(self.env,self.role)

    def close(self):self.conn.close()

    @contextmanager
    def transaction(self,right,allow_unfinished=False):
        if right=='ingest' and self.actor!='importer' or right in ['confirm','publish'] and self.actor!='approver' or right=='query' and self.tenant is None:
            raise Rejected('permission')
        with self.conn.transaction():
            gate=self.conn.execute('SELECT * FROM hb_focus.gate WHERE source=%s FOR '+('SHARE' if right=='query' else 'UPDATE'),(self.bindings['source'],)).fetchone()
            if not gate:raise Rejected('source_unavailable')
            if self.tenant is None:
                cfg=self.conn.execute('SELECT * FROM hb_focus.configuration WHERE id=1').fetchone()
                if cfg['digest']!=self.bindings['config_digest'] or digest(canonical(cfg['body']).encode())!=cfg['digest']:raise Rejected('configuration_changed')
                tenants=self.conn.execute('SELECT tenant_id,properties,active FROM hb_focus.tenant ORDER BY tenant_id FOR SHARE').fetchall()
                expected={tenant_id(t['key']):(t,t['active']) for t in cfg['body']['tenants']}
                if {t['tenant_id']:(t['properties'],t['active']) for t in tenants}!=expected:raise Rejected('tenant_policy_changed')
                if not allow_unfinished and self.conn.execute("SELECT 1 FROM hb_focus.batch WHERE source=%s AND state IN ('pending','unknown','verified')",(gate['source'],)).fetchone():raise Rejected('operation_unresolved')
                yield gate,cfg
            else:
                t=self.conn.execute('SELECT * FROM hb_focus.tenant WHERE tenant_id=%s FOR SHARE',(self.tenant,)).fetchone()
                if not t or not t['active']:raise Rejected('tenant_inactive')
                if t['properties']['actor']!=self.actor:raise Rejected('actor_binding')
                if gate['busy']:raise Rejected('operation_unresolved')
                yield gate,t
