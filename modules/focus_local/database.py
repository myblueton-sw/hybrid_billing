import hashlib
from contextlib import contextmanager
from datetime import datetime, timezone
from uuid import NAMESPACE_DNS, uuid5

import psycopg
from psycopg import sql
from psycopg.rows import dict_row

from .contract import ROOT, Rejected, digest


INSTALLATION = uuid5(NAMESPACE_DNS, 'hybrid-billing.local.synthetic')
PERIOD_START = datetime(2026, 10, 1, tzinfo=timezone.utc)
PERIOD_END = datetime(2026, 11, 1, tzinfo=timezone.utc)


class LocalSecrets(dict):
    def __repr__(self):
        return '<local credentials redacted>'


def environment():
    values = LocalSecrets()
    for line in (ROOT / 'deploy/local/verification/.env').read_text().splitlines():
        key, sep, value = line.partition('=')
        if sep:
            values[key] = value
    return values


def password(env, domain, key):
    return hashlib.sha256((domain + ':' + env[key]).encode()).hexdigest()


def persona(actor):
    if actor not in {f'{role}-{tenant}' for role in ('loader', 'approver', 'reader') for tenant in ('a', 'b')}:
        raise Rejected('unknown_local_actor')
    tenant = actor[-1]
    workspace = uuid5(INSTALLATION, 'workspace-' + tenant)
    connection = uuid5(workspace, 'monthly-synthetic-cost-usage')
    return INSTALLATION, workspace, connection


def connect(env, admin=False):
    return psycopg.connect(host='127.0.0.1', port=15432, dbname='hb',
        user=env['PG_ADMIN_USER'] if admin else 'hb_local_app',
        password=env['PG_ADMIN_PASSWORD'] if admin else password(env, 'pg-app', 'PG_ADMIN_PASSWORD'),
        autocommit=True, row_factory=dict_row, connect_timeout=5)


def initialize(env):
    migration = (ROOT / 'migrations/local_focus/001-authority.sql').read_bytes()
    with connect(env, admin=True) as conn:
        with conn.transaction():
            conn.execute('SELECT pg_advisory_xact_lock(270001)')
            exists = conn.execute("SELECT to_regclass('hb_local.migration') AS value").fetchone()['value']
            if exists:
                saved = conn.execute('SELECT digest FROM hb_local.migration WHERE version=1').fetchone()
                if not saved or saved['digest'] != digest(migration):
                    raise Rejected('migration_digest')
            else:
                conn.execute(migration.decode())
                conn.execute('INSERT INTO hb_local.migration VALUES (1,%s)', (digest(migration),))
            if not conn.execute("SELECT 1 FROM pg_roles WHERE rolname='hb_local_app'").fetchone():
                conn.execute(sql.SQL('CREATE ROLE hb_local_app LOGIN PASSWORD {} NOSUPERUSER NOBYPASSRLS').format(sql.Literal(password(env, 'pg-app', 'PG_ADMIN_PASSWORD'))))
            for tenant in ('a', 'b'):
                installation, workspace, connection = persona('loader-' + tenant)
                conn.execute('INSERT INTO hb_local.connection VALUES (%s,%s,%s,%s,%s,%s,%s,1,true) ON CONFLICT DO NOTHING',
                    (installation, workspace, connection, 'synthetic-account-' + tenant, 'synthetic-sub-' + tenant, PERIOD_START, PERIOD_END))
                conn.execute('INSERT INTO hb_local.publication_scope(installation_id,workspace_id,connection_id) VALUES (%s,%s,%s) ON CONFLICT DO NOTHING', (installation, workspace, connection))
                for role, rights in [('loader', ['ingest']), ('approver', ['confirm', 'publish', 'export']), ('reader', ['query'])]:
                    conn.execute('INSERT INTO hb_local.principal VALUES (%s,%s,%s,%s,%s) ON CONFLICT DO NOTHING', (installation, workspace, role + '-' + tenant, connection, rights))
            role = conn.execute("SELECT oid,rolsuper,rolbypassrls,rolcreaterole,rolcreatedb,rolreplication FROM pg_roles WHERE rolname='hb_local_app'").fetchone()
            if any(role[name] for name in ['rolsuper','rolbypassrls','rolcreaterole','rolcreatedb','rolreplication']):
                raise Rejected('incompatible_pg_role')
            if conn.execute('SELECT 1 FROM pg_auth_members WHERE member=%s', (role['oid'],)).fetchone():
                raise Rejected('incompatible_pg_membership')
            if conn.execute('SELECT 1 FROM pg_class WHERE relowner=%s', (role['oid'],)).fetchone():
                raise Rejected('incompatible_pg_owner')
            tables = ['connection','principal','release','publication_scope','operation','confirmation','publication_decision','outbox','quarantine']
            for table in tables:
                ident = sql.Identifier('hb_local', table)
                conn.execute(sql.SQL('ALTER TABLE {} ENABLE ROW LEVEL SECURITY').format(ident))
                conn.execute(sql.SQL('ALTER TABLE {} FORCE ROW LEVEL SECURITY').format(ident))
                if conn.execute('SELECT 1 FROM pg_policies WHERE schemaname=%s AND tablename=%s AND policyname<>%s', ('hb_local',table,'tenant_scope')).fetchone():
                    raise Rejected('incompatible_pg_policy')
                conn.execute(sql.SQL('DROP POLICY IF EXISTS tenant_scope ON {}').format(ident))
                conn.execute(sql.SQL("CREATE POLICY tenant_scope ON {} USING (installation_id = NULLIF(current_setting('hb.installation',true),'')::uuid AND workspace_id = NULLIF(current_setting('hb.workspace',true),'')::uuid) WITH CHECK (installation_id = NULLIF(current_setting('hb.installation',true),'')::uuid AND workspace_id = NULLIF(current_setting('hb.workspace',true),'')::uuid)").format(ident))
            conn.execute('GRANT USAGE ON SCHEMA hb_local TO hb_local_app')
            conn.execute('GRANT UPDATE(lock_token) ON hb_local.connection TO hb_local_app')
            conn.execute('GRANT SELECT ON ALL TABLES IN SCHEMA hb_local TO hb_local_app')
            conn.execute('GRANT INSERT ON hb_local.release,hb_local.operation,hb_local.confirmation,hb_local.publication_decision,hb_local.outbox,hb_local.quarantine TO hb_local_app')
            conn.execute('GRANT UPDATE(state) ON hb_local.release,hb_local.operation TO hb_local_app')
            conn.execute('GRANT UPDATE(revision,release_id) ON hb_local.publication_scope TO hb_local_app')
            conn.execute('GRANT UPDATE(state,object_key,object_digest) ON hb_local.outbox TO hb_local_app')
            verify_privileges(conn, tables + ['migration'])


def verify_privileges(conn, tables):
    mutable = {'connection':{'lock_token'},'release':{'state'},'operation':{'state'},
        'publication_scope':{'revision','release_id'},'outbox':{'state','object_key','object_digest'}}
    insertable = {'release','operation','confirmation','publication_decision','outbox','quarantine'}
    if conn.execute("SELECT has_schema_privilege('hb_local_app','hb_local','CREATE') AS value").fetchone()['value']:
        raise Rejected('incompatible_pg_privileges')
    for table in tables:
        name = 'hb_local.' + table
        for privilege in ('DELETE','TRUNCATE','TRIGGER','REFERENCES'):
            if conn.execute("SELECT has_table_privilege('hb_local_app',%s,%s) AS value", (name,privilege)).fetchone()['value']:
                raise Rejected('incompatible_pg_privileges')
        columns = conn.execute('SELECT column_name FROM information_schema.columns WHERE table_schema=%s AND table_name=%s', ('hb_local',table)).fetchall()
        for column in columns:
            field = column['column_name']
            update = conn.execute("SELECT has_column_privilege('hb_local_app',%s,%s,'UPDATE') AS value", (name,field)).fetchone()['value']
            insert = conn.execute("SELECT has_column_privilege('hb_local_app',%s,%s,'INSERT') AS value", (name,field)).fetchone()['value']
            if (update and field not in mutable.get(table,set())) or (insert and table not in insertable):
                raise Rejected('incompatible_pg_privileges')
    if conn.execute("SELECT 1 FROM information_schema.role_table_grants WHERE grantee='hb_local_app' AND table_schema<>'hb_local'").fetchone():
        raise Rejected('incompatible_pg_privileges')


class Authority:
    def __init__(self, actor, env=None):
        self.actor = actor
        self.scope = persona(actor)
        self.env = env or environment()
        self.conn = connect(self.env)

    def close(self):
        self.conn.close()

    @contextmanager
    def transaction(self, right, allow_unfinished=False):
        with self.conn.transaction():
            installation, workspace, connection = self.scope
            self.conn.execute("SELECT set_config('hb.installation',%s,true),set_config('hb.workspace',%s,true)", (str(installation), str(workspace)))
            lock_key = int.from_bytes(connection.bytes[:8], 'big', signed=True)
            self.conn.execute('SELECT pg_advisory_xact_lock(%s)', (lock_key,))
            principal = self.conn.execute('SELECT rights FROM hb_local.principal WHERE actor=%s AND connection_id=%s', (self.actor, connection)).fetchone()
            if not principal or right not in principal['rights']:
                raise Rejected('permission')
            capability = self.conn.execute('SELECT * FROM hb_local.connection WHERE connection_id=%s FOR UPDATE', (connection,)).fetchone()
            if not capability or not capability['active']:
                raise Rejected('policy_unavailable')
            if not allow_unfinished and self.conn.execute("SELECT 1 FROM hb_local.operation WHERE connection_id=%s AND state<>'terminal'", (connection,)).fetchone():
                raise Rejected('operation_unresolved')
            if right != 'ingest' and self.conn.execute('SELECT 1 FROM hb_local.quarantine WHERE connection_id=%s', (connection,)).fetchone():
                raise Rejected('quarantine_hold')
            yield capability

    def release(self, release_id):
        try:
            from uuid import UUID
            release_id = UUID(str(release_id))
        except ValueError:
            raise Rejected('release_identity') from None
        row = self.conn.execute('SELECT * FROM hb_local.release WHERE release_id=%s AND connection_id=%s', (release_id, self.scope[2])).fetchone()
        if not row:
            raise Rejected('release_unavailable')
        return row

    def mark_unknown(self, operation_id):
        try:
            with self.transaction('ingest' if self.actor.startswith('loader-') else 'export', allow_unfinished=True):
                self.conn.execute("UPDATE hb_local.operation SET state='unknown' WHERE operation_id=%s AND state='pending'", (operation_id,))
        except Exception:
            pass
