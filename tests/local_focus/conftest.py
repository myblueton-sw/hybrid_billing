import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from modules.focus_local import database, storage
from modules.focus_local.pipeline import initialize


@pytest.fixture
def local_stack():
    env = database.environment()
    reset_namespace(env)
    yield env
    reset_namespace(env)


def reset_namespace(env):
    with database.connect(env, admin=True) as conn:
        conn.execute('DROP SCHEMA IF EXISTS hb_local CASCADE')
    admin = storage.client(env, 'probe_admin')
    try:
        admin.command('DROP DATABASE IF EXISTS hb_local')
        admin.command('ALTER SETTINGS PROFILE hb_local_locked SETTINGS ' + storage.LOCKED_SETTINGS)
        admin.command('REVOKE ALTER ON hb_local.* FROM hb_local_reader_a')
    finally:
        admin.close()
    initialize(env)
