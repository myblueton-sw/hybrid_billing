import sys
from pathlib import Path

import pytest

sys.path.insert(0,str(Path(__file__).resolve().parents[2]))

from modules.focus_import import authority,warehouse
from modules.focus_import.model import BINDINGS,ROOT,SOURCE_SHA,demo_config,digest
from modules.focus_local.database import environment


@pytest.fixture(scope='session')
def official():
    path=ROOT/'.local/datasets/focus/official-1000.csv'
    if not path.exists():path=ROOT/'.local/runs/hb27-official-intake-20261023/focus_sample.csv'
    data=path.read_bytes()
    assert digest(data)==SOURCE_SHA
    return data


@pytest.fixture
def stack(official):
    env=environment()
    saved=BINDINGS.read_bytes() if BINDINGS.exists() else None
    def reset():
        with authority.connect(env) as conn:conn.execute('DROP SCHEMA IF EXISTS hb_focus CASCADE')
        ch=warehouse.client(env,'probe_admin')
        try:
            for row in ch.query("SELECT short_name FROM system.row_policies WHERE database='hb_focus'").result_rows:
                assert row[0].startswith('p_hb_focus_')
                ch.command('DROP ROW POLICY `'+row[0]+'` ON hb_focus.fact')
            ch.command('DROP DATABASE IF EXISTS hb_focus SYNC')
        finally:ch.close()
    reset()
    yield env,demo_config(official)
    reset()
    if saved is not None:BINDINGS.write_bytes(saved)
