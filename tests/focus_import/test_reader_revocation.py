import json
from types import SimpleNamespace
from uuid import uuid4

import pytest

from modules.focus_import import authority, control, revocation, warehouse
from modules.focus_import.model import Rejected, tenant_id, view_id


@pytest.fixture
def config():
    return {'tenants': [
        {'key': 'a1', 'actor': 'reader-a1', 'active': True},
        {'key': 'a2', 'actor': 'reader-a2', 'active': True},
        {'key': 'b', 'actor': 'reader-b', 'active': True}],
        'views': [{'actor': 'company-a', 'tenant_keys': ['a1', 'a2'], 'active': True},
            {'actor': 'company-b', 'tenant_keys': ['b'], 'active': True}]}


def test_tenant_denial_closes_whole_audience_without_sibling(config):
    assert revocation.denied_actors(config, ['reader-a1']) == {'reader-a1', 'company-a'}
    assert revocation.denied_roles(config, ['reader-a1']) == {
        authority.reader_role(tenant_id('a1')), authority.reader_role(view_id('company-a'))}


def test_view_denial_does_not_revoke_tenants(config):
    assert revocation.denied_actors(config, ['company-a']) == {'company-a'}


def test_inactive_tenant_is_denied_before_initialization(config):
    config['tenants'][0]['active'] = False
    assert revocation.denied_actors(config, []) == {'reader-a1', 'company-a'}


def recovery(tmp_path, monkeypatch, operation, previous_operation=None):
    installation = str(uuid4())
    saved = {'installation': installation, 'sequence': 2, 'digest': 'previous'}
    journal = SimpleNamespace(head={'installation': installation, 'sequence': 3, 'digest': 'current'},
        state={'operation': operation, 'revoked_actors': ['reader-a1']})
    (tmp_path / '000000000003.json').write_text(json.dumps({'previous': 'previous'}))
    (tmp_path / '000000000002.json').write_text(json.dumps({'state': {'operation': previous_operation}}))
    monkeypatch.setattr(control, 'DIRECTORY', tmp_path)
    monkeypatch.setattr(control, 'checkpoint', lambda conn: saved)
    calls = []
    connection = SimpleNamespace(execute=lambda *args: calls.append(args))
    return journal, saved, connection, calls


@pytest.mark.parametrize('completed', [False, True])
def test_only_same_revocation_checkpoint_can_resume(tmp_path, monkeypatch, completed):
    operation = {'kind': 'revoke', 'target': 'reader-a1'}
    journal, saved, connection, calls = recovery(tmp_path, monkeypatch,
        None if completed else operation, operation if completed else None)
    revocation.resume_checkpoint(journal, connection, 'reader-a1')
    assert len(calls) == 1 and calls[0][1][:2] == (3, 'current')


@pytest.mark.parametrize('operation', [None, {'kind': 'restore', 'target': 'reader-a1'},
    {'kind': 'revoke', 'target': 'reader-b'}])
def test_unrelated_checkpoint_is_never_repaired(tmp_path, monkeypatch, operation):
    journal, saved, connection, calls = recovery(tmp_path, monkeypatch, operation)
    with pytest.raises(Rejected, match='control_checkpoint'):
        revocation.resume_checkpoint(journal, connection, 'reader-a1')
    assert not calls


def test_checkpoint_cannot_skip_multiple_durable_entries(tmp_path, monkeypatch):
    journal, saved, connection, calls = recovery(tmp_path, monkeypatch, {'kind': 'revoke', 'target': 'reader-a1'})
    saved['sequence'] = 1
    with pytest.raises(Rejected, match='control_checkpoint'):
        revocation.resume_checkpoint(journal, connection, 'reader-a1')
    assert not calls


@pytest.mark.parametrize('response,running,denied', [
    (b'', 0, False), (b'', 1, True),
    (b'{"data":[{"kill_status":"finished"}]}', 0, False),
    (b'{"data":[{"kill_status":"waiting"}]}', 0, True)])
def test_query_cancellation_requires_completion(monkeypatch, response, running, denied):
    role = authority.reader_role(tenant_id('a1'))
    commands = []
    def query(statement):
        rows = [(f'GRANT SELECT ON hb_focus.fact TO {role}',)] if statement.startswith('SHOW GRANTS') else [(running,)]
        return SimpleNamespace(result_rows=rows)
    client = SimpleNamespace(query=query, command=commands.append,
        raw_query=lambda *args, **kwargs: response, close=lambda: None)
    monkeypatch.setattr(warehouse, 'client', lambda *args: client)
    if denied:
        with pytest.raises(Rejected, match='revocation_queries_pending'):
            revocation.enforce_clickhouse({}, {role})
    else:
        revocation.enforce_clickhouse({}, {role})
    assert commands[0] == f'REVOKE SELECT ON hb_focus.fact FROM {role}'
