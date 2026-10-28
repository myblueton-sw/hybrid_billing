import json
import sys
from contextlib import ExitStack, contextmanager
from pathlib import Path
from unittest.mock import patch

import clickhouse_connect
import psycopg

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

from modules.focus_import import archive, authority, control, lifecycle, model, service, warehouse


DIRECTORY = ROOT / '.local/runs/HB-33-revocation-r4'
DATABASE = 'hb33_reader_revocation_20261028_r4'
BUCKET = 'hb33-reader-revocation-20261028-r4'
PORT = 18129


def fingerprints():
    paths = [model.BINDINGS, DIRECTORY / 'bindings.json', DIRECTORY / 'owner.json']
    paths += list((ROOT / '.local/focus-lifecycle').glob('*'))
    paths += list((DIRECTORY / 'control').glob('*'))
    return {str(p): model.digest(p.read_bytes()) for p in paths if p.is_file()}


@contextmanager
def isolated():
    before = fingerprints()
    owner = json.loads((DIRECTORY / 'owner.json').read_text())
    assert owner['ticket'] == 'HB-33' and owner['database'] == DATABASE and owner['bucket'] == BUCKET and owner['clickhouse_port'] == PORT
    pg_connect, ch_connect = psycopg.connect, clickhouse_connect.get_client

    def postgres(*args, **kwargs):
        assert not args and kwargs['host'] == '127.0.0.1' and kwargs['port'] == 15432 and kwargs['dbname'] == 'hb'
        return pg_connect(**{**kwargs, 'dbname': DATABASE})

    def clickhouse(*args, **kwargs):
        assert not args and kwargs['host'] == '127.0.0.1' and kwargs['port'] == 18123
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
            yield
    finally:
        assert fingerprints() == before, 'existing bindings/control evidence changed'
