import json
import re
from uuid import UUID

from psycopg import sql

from .model import Rejected, tenant_id, view_id


def denied_actors(config, revoked):
    actors = set(revoked)
    keys = {t['key'] for t in config['tenants'] if t['actor'] in actors or not t['active']}
    actors.update(t['actor'] for t in config['tenants'] if t['key'] in keys)
    actors.update(v['actor'] for v in config.get('views', [])
        if not v['active'] or keys.intersection(v['tenant_keys']))
    return actors


def denied_roles(config, revoked):
    from .authority import reader_role
    actors = denied_actors(config, revoked)
    return {reader_role(tenant_id(t['key'])) for t in config['tenants'] if t['actor'] in actors} | {
        reader_role(view_id(v['actor'])) for v in config.get('views', []) if v['actor'] in actors}


def revoke_privileges(conn, roles):
    for role in sorted(roles):
        if not conn.execute('SELECT 1 FROM pg_roles WHERE rolname=%s', (role,)).fetchone():
            continue
        identifier = sql.Identifier(role)
        conn.execute(sql.SQL('REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA hb_focus FROM {}').format(identifier))
        columns = conn.execute("SELECT table_name,column_name FROM information_schema.columns WHERE table_schema='hb_focus'").fetchall()
        tables = {}
        for row in columns:
            tables.setdefault(row['table_name'], []).append(row['column_name'])
        for table, names in tables.items():
            conn.execute(sql.SQL('REVOKE ALL PRIVILEGES ({}) ON TABLE {} FROM {}').format(
                sql.SQL(',').join(map(sql.Identifier, names)), sql.Identifier('hb_focus', table), identifier))
        conn.execute(sql.SQL('REVOKE ALL PRIVILEGES ON SCHEMA hb_focus FROM {}').format(identifier))


def terminate_postgres(conn, roles):
    if not roles:
        return
    identity = conn.execute('SELECT current_user AS name').fetchone()['name']
    if identity in roles:
        raise Rejected('revocation_admin_identity')
    rows = conn.execute('SELECT pg_terminate_backend(pid,5000) AS stopped FROM pg_stat_activity '
        'WHERE datname=current_database() AND usename=ANY(%s) AND pid<>pg_backend_pid()', (sorted(roles),)).fetchall()
    if any(not r['stopped'] for r in rows):
        raise Rejected('revocation_sessions_pending')


def enforce_postgres(conn, config, revoked):
    actors = denied_actors(config, revoked)
    roles = denied_roles(config, revoked)
    terminate_postgres(conn, roles)
    with conn.transaction():
        conn.execute("SET LOCAL lock_timeout='5s'")
        revoke_privileges(conn, roles)
        conn.execute("UPDATE hb_focus.tenant SET active=false WHERE properties->>'actor'=ANY(%s)", (sorted(actors),))
        conn.execute('UPDATE hb_focus.audience SET active=false WHERE actor=ANY(%s)', (sorted(actors),))
    terminate_postgres(conn, roles)


def enforce_clickhouse(env, roles):
    from .warehouse import client
    if not roles:
        return
    if any(not re.fullmatch('hb_focus_r_[0-9a-f]{32}', role) for role in roles):
        raise Rejected('revocation_reader_identity')
    admin = client(env, 'probe_admin')
    try:
        for role in sorted(roles):
            for row in admin.query('SHOW GRANTS FOR ' + role).result_rows:
                grant = row[0].replace('`', '')
                if grant not in {f'GRANT SELECT ON hb_focus.fact TO {role}', f'GRANT USAGE ON *.* TO {role}'}:
                    raise Rejected('revocation_shared_ch_identity')
        for role in sorted(roles):
            admin.command(f'REVOKE SELECT ON hb_focus.fact FROM {role}')
            admin.command(f'ALTER ROW POLICY p_{role} ON hb_focus.fact FOR SELECT USING 0 TO {role}')
        names = ','.join("'" + role + "'" for role in sorted(roles))
        response = admin.raw_query(f'KILL QUERY WHERE user IN ({names}) SYNC',
            fmt='JSON', settings={'max_execution_time': 5})
        killed = json.loads(response)['data'] if response.strip() else []
        if any(row.get('kill_status') not in {'finished', 'unknown_query'} for row in killed):
            raise Rejected('revocation_queries_pending')
        if admin.query(f'SELECT count() FROM system.processes WHERE user IN ({names})').result_rows[0][0]:
            raise Rejected('revocation_queries_pending')
    finally:
        admin.close()


def resume_checkpoint(journal, conn, target):
    from . import control
    saved = control.checkpoint(conn)
    head = journal.head
    if saved and saved['sequence'] == head['sequence'] and saved['digest'] == head['digest']:
        return
    if not saved or saved['sequence'] != head['sequence'] - 1:
        raise Rejected('control_checkpoint')
    entry = json.loads((control.DIRECTORY / f"{head['sequence']:012d}.json").read_text())
    previous = json.loads((control.DIRECTORY / f"{saved['sequence']:012d}.json").read_text())['state']
    operation = journal.state.get('operation') or previous.get('operation')
    if entry['previous'] != saved['digest'] or not operation or operation.get('kind') != 'revoke' or operation.get('target') != target:
        raise Rejected('control_checkpoint')
    if target not in journal.state['revoked_actors']:
        raise Rejected('control_checkpoint')
    conn.execute('UPDATE hb_focus.control_checkpoint SET sequence=%s,digest=%s WHERE id=1 AND installation=%s',
        (head['sequence'], head['digest'], UUID(head['installation'])))


def revoke(actor, target):
    from . import authority, control, lifecycle
    from modules.focus_local.database import environment
    if actor not in lifecycle.RIGHTS['revoke']:
        raise Rejected('permission')
    env = environment()
    with control.locked(), authority.connect(env) as conn:
        journal = control.Journal(env, conn)
        state = journal.load(repair=True)
        if state is None:
            raise Rejected('lifecycle_not_activated')
        if state.get('retired'):
            raise Rejected('installation_retired')
        operation = state.get('operation')
        if operation and (operation.get('kind') != 'revoke' or operation.get('target') != target):
            raise Rejected('operation_unresolved')
        config = state['metadata']['configuration'][0]['body']
        readers = {t['actor'] for t in config['tenants']} | {v['actor'] for v in config.get('views', [])}
        if target not in readers:
            raise Rejected('reader_unavailable')
        resume_checkpoint(journal, conn, target)
        if target not in state['revoked_actors']:
            state['revoked_actors'].append(target)
        if not operation:
            state['operation'] = {'kind': 'revoke', 'target': target}
            journal.save(state)
        enforce_postgres(conn, config, state['revoked_actors'])
        enforce_clickhouse(env, denied_roles(config, state['revoked_actors']))
        state['metadata'] = lifecycle.metadata(conn)
        state['operation'] = None
        journal.save(state)
        return {'state': 'revoked', 'actor': target, 'engine_access': 'denied'}
