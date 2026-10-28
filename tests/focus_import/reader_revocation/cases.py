import ast
import json
import subprocess
import time
from concurrent.futures import ThreadPoolExecutor
from contextlib import ExitStack
from unittest.mock import patch

import psycopg

from environment import ACTORS, COMPANY, DIRECTORY, KEYS, OTHER_COMPANY, ROOT, save
from modules.focus_import import authority, collection, control, lifecycle, service, warehouse
from modules.focus_import.model import Rejected, tenant_id, view_id


PG_COUNT = 'SELECT coalesce(sum(row_count),0) AS n FROM hb_focus.slice'
CH_COUNT = 'SELECT count() FROM hb_focus.fact'


def role(actor):
    return authority.reader_role(view_id(actor) if actor in [COMPANY, OTHER_COMPANY] else tenant_id(next(k for k in KEYS if ACTORS[k] == actor)))


def pg_denied(connection, query=PG_COUNT):
    try:
        result = connection.execute(query).fetchone()
    except psycopg.Error as error:
        if error.sqlstate in {'42501', '57P01', '08003', '08006'} or connection.closed or connection.broken:
            return {'denied': True, 'sqlstate': error.sqlstate, 'connection_closed': connection.closed}
        raise
    assert result['n'] == 0, 'revoked PostgreSQL rows visible'
    return {'denied': True, 'rows': 0}


def ch_denied(client):
    try:
        rows = client.query(CH_COUNT).result_rows[0][0]
    except Exception as error:
        assert any(code in str(error) for code in ['(ACCESS_DENIED)', '(AUTHENTICATION_FAILED)', '(UNKNOWN_USER)']), 'non-authoritative ClickHouse failure: ' + type(error).__name__
        return {'denied': True, 'error_type': type(error).__name__}
    assert rows == 0, 'revoked ClickHouse rows visible'
    return {'denied': True, 'rows': 0}


def fresh_denied(env, actor):
    with authority.connect(env, role(actor)) as connection:
        pg = pg_denied(connection)
    client = warehouse.client(env, role(actor))
    try:
        ch = ch_denied(client)
    finally:
        client.close()
    try:
        collection.query(actor)
    except (Rejected, psycopg.errors.InsufficientPrivilege):
        pass
    else:
        raise AssertionError('revoked product access succeeded')
    return {'postgres': pg, 'clickhouse': ch, 'product_denied': True}


def unaffected(env):
    result = {}
    for key, expected in [(KEYS[1], 1), (KEYS[2], 2)]:
        actor = ACTORS[key]
        with authority.connect(env, role(actor)) as connection:
            assert connection.execute(PG_COUNT).fetchone()['n'] == expected
        client = warehouse.client(env, role(actor))
        try:
            assert client.query(CH_COUNT).result_rows[0][0] == expected
        finally:
            client.close()
        assert collection.query(actor)['row_count'] == expected
        result[actor] = expected
    return result


def old_revoke():
    source = subprocess.check_output(['git', 'show', '79866f4:modules/focus_import/lifecycle.py'], cwd=ROOT, text=True)
    node = next(n for n in ast.parse(source).body if isinstance(n, ast.FunctionDef) and n.name == 'revoke')
    namespace = dict(vars(lifecycle))
    exec(compile(ast.Module(body=[node], type_ignores=[]), '<HB-32-revoke>', 'exec'), namespace)
    return namespace['revoke']


def baseline(env):
    target = ACTORS[KEYS[0]]
    with authority.connect(env, role(target)) as connection:
        client = warehouse.client(env, role(target))
        try:
            assert connection.execute(PG_COUNT).fetchone()['n'] == 3
            assert client.query(CH_COUNT).result_rows[0][0] == 3
            old_revoke()('operator', target)
            pg = int(connection.execute(PG_COUNT).fetchone()['n'])
            ch = client.query(CH_COUNT).result_rows[0][0]
            assert pg == ch == 3, 'baseline leak did not reproduce'
        finally:
            client.close()
    with authority.connect(env, role(target)) as connection:
        assert connection.execute(PG_COUNT).fetchone()['n'] == 3
    client = warehouse.client(env, role(target))
    try:
        assert client.query(CH_COUNT).result_rows[0][0] == 3
    finally:
        client.close()
    return {'defect_reproduced': True, 'existing_pg_rows': pg, 'reused_ch_rows': ch, 'fresh_credentials_rows': 3}


def verify(env, config, backup, record):
    from modules.focus_import import revocation
    target = ACTORS[KEYS[0]]
    with ExitStack() as stack:
        opened = []
        for actor in [target, COMPANY]:
            connection = stack.enter_context(authority.connect(env, role(actor)))
            connection.execute('PREPARE hb33_read AS ' + PG_COUNT)
            assert connection.execute('EXECUTE hb33_read').fetchone()['n'] == (3 if actor == target else 4)
            client = warehouse.client(env, role(actor))
            stack.callback(client.close)
            opened.append((actor, connection, client))
        long_connection = stack.enter_context(authority.connect(env, role(target)))
        long_connection.execute('BEGIN ISOLATION LEVEL REPEATABLE READ')
        assert long_connection.execute(PG_COUNT).fetchone()['n'] == 3

        def partial_failure():
            with patch.object(revocation, 'enforce_clickhouse', side_effect=RuntimeError('synthetic-engine-unavailable')):
                try:
                    lifecycle.revoke('operator', target)
                except RuntimeError as error:
                    assert str(error) == 'synthetic-engine-unavailable'
                else:
                    raise AssertionError('injected engine failure not observed')
            with authority.connect(env) as connection:
                journal = control.Journal(env, connection)
                journal.load()
                assert journal.state['operation'] and target in journal.state['revoked_actors']
            try:
                collection.query(ACTORS[KEYS[1]])
            except Rejected:
                pass
            else:
                raise AssertionError('pending operation allowed product reads')
            return {'durable_pending': True, 'product_reads_fail_closed': True}

        record('R01:partial-engine-failure', partial_failure)

        def retry_with_inflight():
            slow = warehouse.client(env, role(target))
            admin = warehouse.client(env, 'probe_admin')
            executor = ThreadPoolExecutor(max_workers=1)
            future = executor.submit(slow.query, 'SELECT sleepEachRow(0.1),(SELECT count() FROM hb_focus.fact) AS own_count FROM numbers(200) SETTINGS max_block_size=1,max_threads=1,max_execution_time=25')
            try:
                deadline = time.monotonic() + 10
                active = False
                while time.monotonic() < deadline and not future.done():
                    active = bool(admin.query('SELECT count() FROM system.processes WHERE user={u:String} AND position(query,\'sleepEachRow\')>0',
                        parameters={'u': role(target)}).result_rows[0][0])
                    if active:
                        break
                    time.sleep(0.05)
                if not active and future.done():
                    future.result()
                assert active, 'bounded ClickHouse probe did not enter execution'
                result = lifecycle.revoke('operator', target)
                try:
                    future.result(timeout=10)
                except Exception as error:
                    assert '(QUERY_WAS_CANCELLED)' in str(error), 'in-flight query lacked authoritative cancellation: ' + type(error).__name__
                else:
                    raise AssertionError('in-flight revoked query completed')
                return {**result, 'inflight_query_cancelled': True}
            finally:
                if not future.done():
                    admin.command('KILL QUERY WHERE user={u:String} SYNC', parameters={'u': role(target)})
                executor.shutdown(wait=True)
                admin.close()
                slow.close()

        record('R02:retry-completes', retry_with_inflight)

        def existing():
            observations = {actor: {'prepared_pg': pg_denied(connection, 'EXECUTE hb33_read'), 'reused_ch': ch_denied(client)}
                for actor, connection, client in opened}
            observations['repeatable_read'] = pg_denied(long_connection)
            return observations

        record('R03:existing-sessions', existing)
    record('R04:fresh-tenant-reader', lambda: fresh_denied(env, target))
    record('R05:fresh-dependent-audience', lambda: fresh_denied(env, COMPANY))
    record('R06:unaffected-readers', lambda: unaffected(env))
    record('R07:duplicate-revoke', lambda: lifecycle.revoke('operator', target))

    def reinitialize():
        service.initialize(config)
        return {'tenant': fresh_denied(env, target), 'audience': fresh_denied(env, COMPANY), 'unaffected': unaffected(env)}

    record('R08:ordinary-reinitialize', reinitialize)

    def restore():
        lifecycle.restore('operator', backup['backup_id'])
        return {'tenant': fresh_denied(env, target), 'audience': fresh_denied(env, COMPANY), 'unaffected': unaffected(env)}

    record('R09:old-backup-restore', restore)

    def view_only():
        old_revoke()('operator', OTHER_COMPANY)
        with authority.connect(env, role(OTHER_COMPANY)) as connection:
            client = warehouse.client(env, role(OTHER_COMPANY))
            try:
                assert connection.execute(PG_COUNT).fetchone()['n'] == 2
                assert client.query(CH_COUNT).result_rows[0][0] == 2
                lifecycle.restore('operator', backup['backup_id'])
                existing = {'postgres': pg_denied(connection), 'clickhouse': ch_denied(client)}
            finally:
                client.close()
        restored = fresh_denied(env, OTHER_COMPANY)
        unaffected(env)
        lifecycle.revoke('operator', OTHER_COMPANY)
        return {'legacy_restore_existing': existing, 'legacy_restore_fresh': restored,
            'view': fresh_denied(env, OTHER_COMPANY), 'unaffected': unaffected(env)}

    record('R10:direct-audience-revocation', view_only)


def missing_schema(env, backup):
    from environment import DATABASE, SOURCE
    with authority.connect(env) as connection:
        assert connection.execute('SELECT current_database() AS name').fetchone()['name'] == DATABASE
        assert connection.execute('SELECT body FROM hb_focus.configuration WHERE id=1').fetchone()['body']['source'] == SOURCE
        connection.execute('DROP SCHEMA hb_focus CASCADE')
    lifecycle.restore('operator', backup['backup_id'])
    return {'tenant': fresh_denied(env, ACTORS[KEYS[0]]), 'dependent_audience': fresh_denied(env, COMPANY),
        'direct_audience': fresh_denied(env, OTHER_COMPANY), 'unaffected': unaffected(env)}
