import json
import sys
from contextlib import ExitStack, contextmanager
from pathlib import Path
from unittest.mock import patch

import clickhouse_connect
import psycopg
from psycopg import sql


ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))
sys.path.append(str(ROOT / 'tests/focus_import/msp_isolation'))

import fixture_data
import preflight as prior_preflight
import runtime as prior_runtime
from modules.focus_import import archive, authority, control, lifecycle, model, service, warehouse
from modules.focus_local.database import environment


DATABASE = 'hb33_reader_revocation_20261028'
BUCKET = 'hb33-reader-revocation-20261028'
PORT = 18125
DIRECTORY = ROOT / '.local/runs/HB-33-revocation'
KEYS = ['msp33-a1', 'msp33-a2', 'msp33-b']
ACTORS = {key: 'reader-' + key for key in KEYS}
COMPANY = 'reader-msp33-company-a'
OTHER_COMPANY = 'reader-msp33-company-b'
SOURCE = 'hb33-reader-revocation'


def configure(run_number):
    global DATABASE, BUCKET, PORT, DIRECTORY, KEYS, ACTORS, COMPANY, OTHER_COMPANY, SOURCE
    if type(run_number) is not int or not 0 <= run_number <= 9:
        raise ValueError('run number must be an integer from 0 through 9')
    suffix = '-r' + str(run_number) if run_number else ''
    DATABASE = 'hb33_reader_revocation_20261028' + ('_r' + str(run_number) if run_number else '')
    BUCKET = 'hb33-reader-revocation-20261028' + suffix
    PORT = 18125 + run_number
    DIRECTORY = ROOT / ('.local/runs/HB-33-revocation' + suffix)
    KEYS = [key + suffix for key in ['msp33-a1', 'msp33-a2', 'msp33-b']]
    ACTORS = {key: 'reader-' + key for key in KEYS}
    COMPANY = 'reader-msp33-company-a' + suffix
    OTHER_COMPANY = 'reader-msp33-company-b' + suffix
    SOURCE = 'hb33-reader-revocation' + suffix


def save(name, body):
    path = DIRECTORY / name
    path.write_text(json.dumps(body, sort_keys=True, indent=2) + '\n')


def preflight():
    original = clickhouse_connect.get_client

    def clickhouse(*args, **kwargs):
        assert kwargs.get('host') == '127.0.0.1' and kwargs.get('port') == 18124
        return original(*args, **{**kwargs, 'port': PORT})

    with patch.object(prior_preflight, 'DATABASE', DATABASE), patch.object(prior_preflight, 'BUCKET', BUCKET), patch.object(clickhouse_connect, 'get_client', clickhouse):
        return prior_preflight.inspect()


def reserve_roles(config):
    roles = sorted(model.reader_scopes(config))
    with authority.connect(environment()) as connection:
        assert not connection.execute('SELECT rolname FROM pg_roles WHERE rolname=ANY(%s)', (roles,)).fetchall(), 'reader role collision'
    owner = json.loads((DIRECTORY / 'owner.json').read_text())
    owner['reader_roles'] = roles
    save('owner.json', owner)


@contextmanager
def fixtures():
    original = fixture_data.configuration

    def configuration(rows):
        config = original(rows)
        config['source'] = SOURCE
        config['views'].append({'actor': OTHER_COMPANY, 'company_id': 'company-b', 'tenant_keys': [KEYS[2]], 'active': True})
        return config

    with ExitStack() as stack:
        for name, value in [('KEYS', KEYS), ('ACTORS', ACTORS), ('COMPANY', COMPANY), ('configuration', configuration)]:
            stack.enter_context(patch.object(fixture_data, name, value))
        stack.enter_context(patch.object(prior_runtime, 'reserve_roles', reserve_roles))
        yield fixture_data


@contextmanager
def isolated(create=False):
    before = prior_runtime.default_status()
    bindings_before = model.BINDINGS.read_bytes()
    env = environment()
    roles_before = prior_runtime.shared_roles(env)
    owner_path = DIRECTORY / 'owner.json'
    if create:
        assert not DIRECTORY.exists(), 'existing HB-33 run directory'
        gates = preflight()
        assert gates['status'] == 'PASS', 'namespace preflight failed'
        DIRECTORY.mkdir(parents=True, mode=0o700)
        save('preflight.json', gates)
        with authority.connect(env) as connection:
            connection.execute(sql.SQL('CREATE DATABASE {}').format(sql.Identifier(DATABASE)))
        save('owner.json', {'ticket': 'HB-33', 'database': DATABASE, 'bucket': BUCKET, 'clickhouse_port': PORT})
    owner = json.loads(owner_path.read_text())
    assert owner['ticket'] == 'HB-33' and owner['database'] == DATABASE and owner['bucket'] == BUCKET and owner['clickhouse_port'] == PORT
    pg_connect = psycopg.connect
    ch_connect = clickhouse_connect.get_client

    def postgres(*args, **kwargs):
        assert not args and kwargs.get('host') == '127.0.0.1' and kwargs.get('port') == 15432 and kwargs.get('dbname') == 'hb'
        return pg_connect(**{**kwargs, 'dbname': DATABASE})

    def clickhouse(*args, **kwargs):
        assert not args and kwargs.get('host') == '127.0.0.1' and kwargs.get('port') == 18123
        return ch_connect(**{**kwargs, 'port': PORT})

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
                    assert config['source'] == SOURCE
            with fixtures() as data:
                yield env, data
    finally:
        after = prior_runtime.default_status()
        roles_after = prior_runtime.shared_roles(env)
        assert before == after and bindings_before == model.BINDINGS.read_bytes() and roles_before == roles_after
        save('default-preservation.json', {'default_runtime_unchanged': True, 'journal_sequence': after['sequence'],
            'status_sha256': model.digest(model.canonical(after).encode()), 'bindings_sha256': model.digest(bindings_before),
            'shared_roles_unchanged': True, 'database': DATABASE, 'bucket': BUCKET, 'clickhouse_port': PORT})
