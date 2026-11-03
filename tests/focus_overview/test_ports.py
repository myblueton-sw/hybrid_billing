from unittest.mock import patch

import pytest

from modules.focus_local.ports import DEFAULTS, port
from modules.focus_local.contract import Rejected


def test_defaults_and_overrides():
    with patch.dict('os.environ', {}, clear=True):
        assert {s: port({}, s) for s in DEFAULTS} == DEFAULTS
        assert port({'PG_PORT': '25432'}, 'PG') == 25432
    with patch.dict('os.environ', {'HB_LOCAL_PG_PORT': '25432'}):
        assert port({'PG_PORT': 15432}, 'PG') == 25432


@pytest.mark.parametrize('value', [0, 65536, True, None, '', '-1', '1.0', 'localhost:1', '25432x', '1\n'])
def test_invalid_ports(value):
    with patch.dict('os.environ', {}, clear=True), pytest.raises(Rejected, match='local_port'):
        port({'PG_PORT': value}, 'PG')


def test_all_clients_use_trusted_loopback_overrides():
    from modules.focus_local import database, storage
    from modules.focus_import import authority, warehouse
    env = {'PG_PORT': '25432', 'CH_PORT': '28123', 'S3_PORT': '28333',
           'PG_ADMIN_USER': 'test', 'PG_ADMIN_PASSWORD': 'test', 'CH_ADMIN_PASSWORD': 'test',
           'S3_ACCESS_KEY': 'test', 'S3_SECRET_KEY': 'test'}
    with patch.dict('os.environ', {}, clear=True), patch('psycopg.connect') as pg:
        database.connect(env, admin=True)
        assert pg.call_args.kwargs['port'] == 25432
        assert pg.call_args.kwargs['host'] == '127.0.0.1'
        authority.connect(env)
        assert pg.call_args.kwargs['port'] == 25432
    with patch.dict('os.environ', {}, clear=True), patch('clickhouse_connect.get_client') as ch:
        storage.client(env, 'probe_admin')
        assert ch.call_args.kwargs['port'] == 28123
        warehouse.client(env, 'probe_admin')
        assert ch.call_args.kwargs['port'] == 28123
        assert ch.call_args.kwargs['host'] == '127.0.0.1'
    with patch.dict('os.environ', {}, clear=True), patch('boto3.client') as s3:
        storage.object_client(env)
        assert s3.call_args.kwargs['endpoint_url'] == 'http://127.0.0.1:28333'
