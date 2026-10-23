import json
from uuid import UUID

import pytest

from modules.focus_import import authority, collection, lifecycle, warehouse
from modules.focus_import.model import digest, tenant_id
from test_intake import parquet
from test_lifecycle import temporal_stack, measure
from test_temporal import source_row
from tools import focus_lifecycle as cli


def invoke(monkeypatch, capsys, *args):
    monkeypatch.setattr('sys.argv', ['focus_lifecycle.py', *args])
    assert cli.main() == 0
    result = json.loads(capsys.readouterr().out)
    assert result['status'] == 'ok'
    return result


@pytest.mark.parametrize('version', ['1.0', '1.2', '1.3'])
def test_parquet_cli_storage_isolation_archive_restore(temporal_stack, monkeypatch, capsys, tmp_path, version):
    env, cfg, policies = temporal_stack
    for p in policies.values(): p['total_days'] = 36500
    lifecycle.activate('operator', 'day', policies)
    source = [source_row(cost='10.000000000000000001'), source_row(cost='-0.000000000000000001'),
        {**source_row(), 'billed_cost': '', 'effective_cost': ''}, source_row(provider='Microsoft', cost='20')]
    data = parquet(source, version, extras={'SyntheticNote': ['a', 'b', None, 'd']})
    path = tmp_path / 'delivery.parquet'; path.write_bytes(data)
    (tmp_path / 'unregistered.parquet').write_bytes(parquet([source_row(cost='999')], version))
    args = ['--file', str(path), '--version', version]
    described = invoke(monkeypatch, capsys, 'describe', *args, '--start', '2024-09-01T00:00:00Z', '--end', '2024-09-02T00:00:00Z',
        '--grain', 'day', '--revision', '1', '--complete-through', '2024-09-02T00:00:00Z', '--finalized')
    descriptor = tmp_path / 'descriptor.json'; descriptor.write_text(json.dumps(described['descriptor']))
    registered = invoke(monkeypatch, capsys, 'register', '--actor', 'operator', *args, '--descriptor', str(descriptor))
    due = invoke(monkeypatch, capsys, 'run-due', '--actor', 'importer', '--inbox', str(tmp_path))
    assert len(due['collected']) == 1
    item = due['collected'][0]; identity = item['batch_id'] + ':' + str(tenant_id('north'))
    delivery = registered['delivery_id']
    invoke(monkeypatch, capsys, 'confirm', '--actor', 'approver', '--delivery', delivery, '--manifest-digest', item['manifest_digest'])
    invoke(monkeypatch, capsys, 'publish', '--actor', 'approver', '--delivery', delivery)
    company = invoke(monkeypatch, capsys, 'query', '--actor', 'reader-company')
    north = invoke(monkeypatch, capsys, 'query', '--actor', 'reader-north')
    south = invoke(monkeypatch, capsys, 'query', '--actor', 'reader-south')
    assert company['row_count'] == 4 and north['row_count'] == 3 and south['row_count'] == 1
    assert measure(company) == 30 and measure(north) == 10 and measure(south) == 20
    assert {r['provider'] for r in north['rows']} == {'AWS'}
    assert {r['provider'] for r in south['rows']} == {'Microsoft'}
    assert north['rows'][0]['billed_cost'] == '10.000000000000000001'
    assert north['rows'][2]['billed_cost'] is None
    assert json.loads(north['rows'][2]['missing_reasons'])['billed_cost'] == 'typed_null'
    duplicate = invoke(monkeypatch, capsys, 'collect', '--actor', 'importer', '--delivery', delivery, '--file', str(path))
    assert duplicate['duplicate']
    with authority.connect(env) as conn:
        manifest = conn.execute('SELECT manifest FROM hb_focus.batch WHERE batch_id=%s', (UUID(item['batch_id']),)).fetchone()['manifest']
    assert manifest['input_format'] == 'parquet' and manifest['file_digest'] == digest(data)
    key = 'shared-originals/' + item['batch_id'] + '/' + digest(data) + '.parquet'
    s3 = warehouse.object_client(env)
    assert s3.get_object(Bucket=warehouse.BUCKET, Key=key)['Body'].read() == data
    backup = invoke(monkeypatch, capsys, 'backup', '--actor', 'operator')
    lifecycle.archive_slice('operator', identity, at='2024-09-03T00:00:00Z')
    assert measure(collection.query('reader-company')) == 30
    s3.delete_object(Bucket=warehouse.BUCKET, Key=key)
    invoke(monkeypatch, capsys, 'restore', '--actor', 'operator', '--backup', backup['backup_id'])
    assert s3.get_object(Bucket=warehouse.BUCKET, Key=key)['Body'].read() == data
    restored = invoke(monkeypatch, capsys, 'query', '--actor', 'reader-company')
    assert restored['row_count'] == 4 and measure(restored) == 30
