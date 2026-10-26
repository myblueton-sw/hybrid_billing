import json
import subprocess
from contextlib import ExitStack, contextmanager
from pathlib import Path
from unittest.mock import patch

import clickhouse_connect
import psycopg
from psycopg import sql

from preflight import BUCKET, DATABASE, ROOT, inspect
from modules.focus_import import archive, authority, control, lifecycle, model, service, warehouse
from modules.focus_local.database import environment


DIRECTORY = ROOT / '.local/runs/HB-32-isolation'


def default_status():
    result = subprocess.run([str(ROOT / 'tests/engine_probes/.venv/bin/python'), 'tools/focus_lifecycle.py',
        'status', '--actor', 'operator'], cwd=ROOT, capture_output=True, text=True, timeout=120)
    assert result.returncode == 0
    body = json.loads(result.stdout)
    assert body['status'] == 'ok' and body['sequence'] == 58
    return body


def shared_roles(env):
    with authority.connect(env) as connection:
        rows = connection.execute("SELECT rolname,rolsuper,rolbypassrls,rolcreaterole,rolcreatedb,rolreplication FROM pg_roles WHERE rolname IN ('hb_focus_ingest','hb_focus_approve') ORDER BY rolname").fetchall()
        members = connection.execute("SELECT roleid,member,grantor,admin_option FROM pg_auth_members WHERE roleid IN (SELECT oid FROM pg_roles WHERE rolname IN ('hb_focus_ingest','hb_focus_approve')) OR member IN (SELECT oid FROM pg_roles WHERE rolname IN ('hb_focus_ingest','hb_focus_approve')) ORDER BY roleid,member,grantor").fetchall()
    return {'roles': [dict(r) for r in rows], 'memberships': [dict(r) for r in members]}


@contextmanager
def isolated(create=False):
    before = default_status()
    bindings_before = model.BINDINGS.read_bytes()
    env = environment()
    roles_before = shared_roles(env)
    owner_path = DIRECTORY / 'owner.json'
    if create:
        gates = inspect()
        assert gates['status'] == 'PASS', 'namespace preflight failed'
        DIRECTORY.mkdir(parents=True, exist_ok=False, mode=0o700)
        with authority.connect(env) as connection:
            connection.execute(sql.SQL('CREATE DATABASE {}').format(sql.Identifier(DATABASE)))
        owner_path.write_text(json.dumps({'ticket': 'HB-32', 'database': DATABASE, 'bucket': BUCKET,
            'clickhouse_port': 18124, 'preflight': gates}, sort_keys=True, indent=2) + '\n')
    else:
        owner = json.loads(owner_path.read_text())
        assert owner['ticket'] == 'HB-32' and owner['database'] == DATABASE and owner['bucket'] == BUCKET and owner['clickhouse_port'] == 18124
    pg_connect = psycopg.connect
    ch_connect = clickhouse_connect.get_client

    def postgres(*args, **kwargs):
        assert not args and kwargs.get('host') == '127.0.0.1' and kwargs.get('port') == 15432 and kwargs.get('dbname') == 'hb'
        return pg_connect(**{**kwargs, 'dbname': DATABASE})

    def clickhouse(*args, **kwargs):
        assert not args and kwargs.get('host') == '127.0.0.1' and kwargs.get('port') == 18123
        return ch_connect(**{**kwargs, 'port': 18124})

    try:
        with ExitStack() as stack:
            stack.enter_context(patch.object(psycopg, 'connect', postgres))
            stack.enter_context(patch.object(clickhouse_connect, 'get_client', clickhouse))
            for module in [model, authority, service, lifecycle]:
                stack.enter_context(patch.object(module, 'BINDINGS', DIRECTORY / 'bindings.json'))
            for module in [warehouse, archive, control]:
                stack.enter_context(patch.object(module, 'BUCKET', BUCKET))
            stack.enter_context(patch.object(control, 'DIRECTORY', DIRECTORY / 'control'))
            if not create:
                with authority.connect(env) as connection:
                    config = connection.execute('SELECT body FROM hb_focus.configuration WHERE id=1').fetchone()['body']
                    assert config['source'] == 'hb32-msp-isolation'
            yield env
    finally:
        after = default_status()
        roles_after = shared_roles(env)
        assert before == after and bindings_before == model.BINDINGS.read_bytes() and roles_before == roles_after
        proof = {'default_runtime_unchanged': True, 'journal_sequence': after['sequence'],
            'status_sha256': model.digest(model.canonical(after).encode()), 'bindings_sha256': model.digest(bindings_before),
            'shared_roles_unchanged': True, 'database': DATABASE, 'bucket': BUCKET, 'clickhouse_port': 18124}
        (DIRECTORY / 'default-preservation.json').write_text(json.dumps(proof, sort_keys=True, indent=2) + '\n')


def save(name, body):
    (DIRECTORY / name).write_text(json.dumps(body, sort_keys=True, indent=2) + '\n')


def reserve_roles(config):
    roles = sorted(model.reader_scopes(config))
    with authority.connect(environment()) as connection:
        existing = connection.execute('SELECT rolname FROM pg_roles WHERE rolname=ANY(%s)', (roles,)).fetchall()
        assert not existing, 'reader role namespace ownership unavailable'
    owner = json.loads((DIRECTORY / 'owner.json').read_text())
    owner.update(owned_new_pg_reader_roles=roles, shared_existing_pg_roles=['hb_focus_ingest', 'hb_focus_approve'],
        owned_ch_users=[warehouse.CH_WRITER, *roles], owned_ch_database='hb_focus', owned_ch_profile='hb_focus_locked')
    save('owner.json', owner)
