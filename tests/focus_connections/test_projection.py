import copy
import json
from contextlib import contextmanager
from datetime import datetime, timedelta, timezone
from types import SimpleNamespace

import pytest

from modules.focus_import import connection_projection as reader
from modules.focus_import.model import Rejected, tenant_id


RECORD_KEYS = {'id', 'connection', 'focus_version', 'grain', 'window_start', 'window_end', 'registered_domains',
               'semantic_profile', 'allowed_lateness_seconds', 'replacement_limit'}
ENVELOPE_KEYS = {'schema', 'profile', 'read_only', 'financial_authority', 'commands', 'view', 'selection', 'audience',
                 'member_count', 'auth_scope', 'permission_revision', 'snapshot', 'observed_at', 'total', 'next_after', 'capabilities', 'records'}


def pins(body):
    return {k: body[k] for k in ('auth_scope', 'permission_revision', 'snapshot')}


def page(body, after):
    return {'after': str(after), 'snapshot': body['snapshot'], 'permission_revision': body['permission_revision']}


def test_registered_metadata_is_exact_fully_owned_and_not_mutated(connection_fixture):
    f = connection_fixture
    before = copy.deepcopy(f.state)
    body = f.render()
    assert set(body) == ENVELOPE_KEYS
    assert body['read_only'] is True and body['financial_authority'] is False and body['commands'] == 'unconnected'
    assert body['total'] == 1 and body['member_count'] == 2 and body['audience'] == 'company'
    assert body['selection'] == {'q': '', 'after': 0} and body['next_after'] is None
    record = body['records'][0]
    assert set(record) == RECORD_KEYS
    assert record['connection'] == 'north' and record['registered_domains'] == '2'
    assert record['focus_version'] == '1.0' and record['semantic_profile'] == 'exploratory-query-only'
    assert record['allowed_lateness_seconds'] == '0' and record['replacement_limit'] == '3'
    assert record['window_start'] == '2026-01-01T00:00:00+00:00' and record['window_end'] == '2026-02-01T00:00:00+00:00'
    assert all(len(body[k]) == 64 for k in pins(body)) and len(record['id']) == 64
    text = json.dumps(body)
    assert all(word not in text for word in ['HIDDEN', 'PRIVATE', 'south', 'mixed', 'north-a', 'north-b', 'AWS'])
    assert f.state == before


@pytest.mark.parametrize('view', ['security', 'diagnostics'])
def test_unsupported_is_unknown_not_empty(connection_fixture, view):
    body = connection_fixture.render({'view': view})
    assert body['total'] is None and body['records'] == [] and body['next_after'] is None
    assert body['capabilities'] == {'connections': 'fully-owned-registrations-only', 'capability_evidence': 'registration-only',
                                    'provider_authentication': 'unconnected', 'diagnostics': 'unconnected'}
    with pytest.raises(Rejected, match='query_page'):
        connection_fixture.render({**page(body, 1), 'view': view})


def test_confirmed_zero_does_not_reconstruct_legacy_deliveries(connection_fixture):
    f = connection_fixture
    f.state['connections'].pop('north')
    assert f.state['deliveries'] and f.state['receipts']
    for view in ['connections', 'capabilities']:
        body = f.render({'view': view})
        assert body['total'] == 0 and body['records'] == []


def test_foreign_unowned_duplicate_and_mixed_connections_never_count_or_search(connection_fixture):
    f = connection_fixture
    f.add_connection('duplicate', ['north-a'])
    f.state['connections']['duplicate']['domains'] *= 2
    f.add_connection('unowned', ['north-a'])
    f.state['connections']['unowned']['domains'].append(['AWS', 'NOT-OWNED', None])
    f.add_connection('bad-id', ['north-a'])
    f.state['connections']['bad-id']['id'] = 'wrong'
    f.add_connection('empty-domain', [])
    assert f.render()['total'] == 1
    for query in ['south', 'mixed', 'duplicate', 'unowned', 'bad-id', 'empty-domain']:
        assert f.render({'q': query})['total'] == 0


def test_foreign_and_hidden_registration_policy_changes_do_not_change_pins(connection_fixture):
    f = connection_fixture
    before = pins(f.render())
    f.state['connections']['south'].update(version='MALFORMED', private='FOREIGN-PRIVATE')
    f.state['connections']['mixed']['lag_seconds'] = -99
    f.state['connections']['north'].update(dataset='OTHER-HIDDEN', coverage_key='OTHER-HIDDEN',
                                          capabilities={'secret': 'OTHER-HIDDEN'}, semantic_definition={'raw': 'OTHER-HIDDEN'})
    f.state['policies']['north-a']['private'] = 'OTHER-HIDDEN'
    f.state['policies']['south-a']['revision'] = 999
    f.state['metadata']['private'] = {'raw': 'OTHER-HIDDEN'}
    f.state['deliveries']['OTHER-PRIVATE'] = {'raw': 'OTHER-HIDDEN'}
    f.at = '2026-03-01T00:00:00+00:00'
    assert pins(f.render()) == before


def test_hidden_config_and_authorized_domain_value_changes_keep_public_pins(connection_fixture):
    f = connection_fixture
    before = pins(f.render())
    f.config['private'] = 'UNEXPOSED'
    for item in f.config['tenants']:
        if item['key'] == 'north-a':
            item['assignments'][0][1] = 'renamed-authorized-account'
            item['private'] = 'UNEXPOSED'
    for tenant in f.context['tenants']:
        if tenant['properties']['key'] == 'north-a':
            tenant['properties'] = copy.deepcopy(next(t for t in f.config['tenants'] if t['key'] == 'north-a'))
    for connection in f.state['connections'].values():
        for domain in connection['domains']:
            if domain[1] == 'north-a':
                domain[1] = 'renamed-authorized-account'
    f.bind_config()
    assert pins(f.render()) == before


def test_visible_policy_and_membership_changes_invalidate_only_relevant_pins(connection_fixture):
    f = connection_fixture
    initial = f.render()
    f.state['connections']['north']['lag_seconds'] = 123
    changed = f.render()
    assert changed['auth_scope'] == initial['auth_scope'] and changed['permission_revision'] == initial['permission_revision']
    assert changed['snapshot'] != initial['snapshot']
    with pytest.raises(Rejected, match='query_snapshot_changed'):
        f.render(page(initial, 0))
    f.state['policies']['north-a']['revision'] = 2
    policy = f.render()
    assert policy['auth_scope'] == changed['auth_scope'] and policy['permission_revision'] != changed['permission_revision']
    with pytest.raises(Rejected, match='query_snapshot_changed'):
        f.render(page(changed, 0))
    f.context['tenants'].pop()
    member = f.render()
    assert member['auth_scope'] != policy['auth_scope'] and member['permission_revision'] != policy['permission_revision']
    assert member['total'] == 0


@pytest.mark.parametrize('value', [None, {}, {'revision': 0}, {'revision': True}, {'revision': '1'}, {'revision': 2**63}])
def test_missing_or_invalid_current_policy_revision_fails_closed(connection_fixture, value):
    connection_fixture.state['policies']['north-a'] = value
    with pytest.raises(Rejected, match='connection_evidence_unavailable'):
        connection_fixture.render()


@pytest.mark.parametrize('change', [
    {'version': '9.9'}, {'version': None}, {'grain': 'year'}, {'grain': []}, {'semantic_profile': 'approved'}, {'semantic_profile': {}},
    {'lag_seconds': -1}, {'lag_seconds': 604801}, {'lag_seconds': True}, {'max_replacements': 0},
    {'max_replacements': 11}, {'max_replacements': '3'}, {'start': '2026-01-02T00:00:00Z'},
    {'start': '2026-01-01T01:00:00Z'}, {'end': '2026-01-01T00:00:00Z'},
    {'end': '2026-02-02T00:00:00Z'}, {'end': '2026-02-01'},
    {'grain': 'day', 'start': '2000-01-01T00:00:00Z', 'end': '2011-01-01T00:00:00Z'},
])
def test_invalid_owned_registration_fails_closed(connection_fixture, change):
    connection_fixture.state['connections']['north'].update(change)
    with pytest.raises(Rejected, match='connection_evidence_unavailable'):
        connection_fixture.render()


def test_daily_registration_and_limits_keep_exact_zero_and_boundary_values(connection_fixture):
    f = connection_fixture
    f.state['connections']['north'].update(grain='day', start='2026-01-01T00:00:00Z', end='2026-01-03T00:00:00Z',
                                           lag_seconds=604800, max_replacements=10)
    r = f.render()['records'][0]
    assert r['grain'] == 'day' and r['allowed_lateness_seconds'] == '604800' and r['replacement_limit'] == '10'
    assert r['window_end'] == '2026-01-03T00:00:00+00:00'


@pytest.mark.parametrize('params', [
    [], {'actor': 'operator'}, {'tenant': 'north-a'}, {'company': 'north'}, {'url': '/private'}, {'q': 'A'},
    {'q': 'a' * 129}, {'q': 'a/b'}, {'after': '-1'}, {'after': '01'}, {'after': '10001'}, {'after': 0},
    {'view': 'wrong'}, {'snapshot': 'a' * 63}, {'permission_revision': 'A' * 64},
])
def test_closed_query_rejects_unknown_and_noncanonical_values(connection_fixture, params):
    with pytest.raises(Rejected, match='request_invalid'):
        connection_fixture.render(params)


@pytest.mark.parametrize('params', [{'after': '25'}, {'snapshot': 'a' * 64}, {'permission_revision': 'a' * 64}])
def test_paging_requires_paired_pins(connection_fixture, params):
    with pytest.raises(Rejected, match='snapshot_required'):
        connection_fixture.render(params)


def test_bounded_sorted_pages_bind_entire_selection_and_allow_back(connection_fixture):
    f = connection_fixture
    f.state['connections'] = {}
    for index in range(26):
        start = datetime(2026, 1, 1, tzinfo=timezone.utc) + timedelta(days=index)
        f.add_connection(f'north-{index:02}', ['north-a'], start.isoformat(), (start + timedelta(days=1)).isoformat(), 'day')
    first = f.render()
    second = f.render(page(first, 25))
    assert first['total'] == 26 and len(first['records']) == 25 and first['next_after'] == 25
    assert second['total'] == 26 and len(second['records']) == 1 and second['next_after'] is None
    assert first['snapshot'] == second['snapshot'] and f.render(page(first, 0)) == first
    assert [r['connection'] for r in first['records']] == [f'north-{i:02}' for i in range(25)]
    assert second['records'][0]['connection'] == 'north-25'
    f.state['connections']['north-25']['lag_seconds'] = 1
    with pytest.raises(Rejected, match='query_snapshot_changed'):
        f.render(page(first, 25))


def test_prefix_filter_binds_view_and_selection(connection_fixture):
    f = connection_fixture
    r = f.render()['records'][0]
    assert f.render({'q': r['id'][:12]})['total'] == 1
    assert f.render({'q': 'north'})['records'] == [r]
    assert f.render({'q': 'unknown'})['total'] == 0
    body = f.render()
    for extra in [{'q': 'n'}, {'view': 'capabilities'}]:
        with pytest.raises(Rejected, match='query_snapshot_changed'):
            f.render({**page(body, 0), **extra})
    with pytest.raises(Rejected, match='query_page'):
        f.render(page(body, 2))


def test_result_count_and_wire_bound_reject_instead_of_truncating(connection_fixture, monkeypatch):
    f = connection_fixture
    f.state['connections'] = {}
    for index in range(10001):
        start = datetime(2000, 1, 1, tzinfo=timezone.utc) + timedelta(days=index)
        f.add_connection(f'north-{index:05}', ['north-a'], start.isoformat(), (start + timedelta(days=1)).isoformat(), 'day')
    with pytest.raises(Rejected, match='selection_unavailable'):
        f.render()
    monkeypatch.setattr(reader, 'MAX_RESPONSE_BYTES', 32)
    with pytest.raises(Rejected, match='response_bound'):
        f.render({'q': 'nothing'})


def test_tenant_audience_and_current_authority_negatives(connection_fixture):
    f = connection_fixture
    f.add_connection('single', ['north-a'])
    tenant = copy.deepcopy(f.config['tenants'][0])
    f.authority = SimpleNamespace(actor=tenant['actor'], tenant=tenant_id(tenant['key']), is_view=False,
                                   bindings=f.authority.bindings)
    f.context = {'active': True, 'properties': tenant}
    assert f.render()['audience'] == 'tenant' and f.render()['member_count'] == 1
    assert [r['connection'] for r in f.render()['records']] == ['single']
    with pytest.raises(Rejected, match='permission'):
        reader.project('operator', {}, f.authority, f.context, f.state, f.at)
    f.context['active'] = False
    with pytest.raises(Rejected):
        f.render()


@pytest.mark.parametrize('change', ['inactive', 'properties', 'binding', 'configuration', 'duplicate-owner'])
def test_current_membership_configuration_and_ownership_fail_closed(connection_fixture, change):
    f = connection_fixture
    if change == 'inactive': f.context['audience']['active'] = False
    if change == 'properties': f.context['tenants'][0]['properties']['company_id'] = 'foreign'
    if change == 'binding': f.authority.bindings['source'] = 'foreign'
    if change == 'configuration': f.state['metadata']['configuration'][0]['body']['source'] = 'foreign'
    if change == 'duplicate-owner':
        f.config['tenants'][2]['assignments'] = copy.deepcopy(f.config['tenants'][0]['assignments'])
        f.bind_config()
    with pytest.raises(Rejected):
        f.render()


def test_reader_delivers_inside_both_guards_without_original_or_warehouse_reads(connection_fixture, monkeypatch):
    f = connection_fixture
    events = []
    @contextmanager
    def transaction(action):
        assert action == 'query'
        events.append('transaction-open')
        yield None, f.context
        events.append('transaction-close')
    f.authority.transaction = transaction
    @contextmanager
    def access(actor, action):
        assert actor == f.authority.actor and action == 'query'
        events.append('access-open')
        yield SimpleNamespace(state=f.state), SimpleNamespace(a=f.authority), None
        events.append('access-close')
    def forbidden(*args, **kwargs):
        raise AssertionError('Forbidden product operation')
    monkeypatch.setattr(reader.collection, 'access', access)
    monkeypatch.setattr(reader.connections, 'definition', forbidden)
    monkeypatch.setattr(reader.connections, 'add', forbidden)
    monkeypatch.setattr(reader.collection, 'query', forbidden)
    def deliver(body):
        assert events == ['access-open', 'transaction-open']
        events.append('deliver')
        return body
    assert reader.connection_metadata(f.authority.actor, {}, deliver)['total'] == 1
    assert events == ['access-open', 'transaction-open', 'deliver', 'transaction-close', 'access-close']


def test_operator_context_and_invalid_query_never_open_authority(connection_fixture, monkeypatch):
    calls = []
    @contextmanager
    def access(actor, action):
        calls.append((actor, action))
        yield None, None, None
    monkeypatch.setattr(reader.collection, 'access', access)
    with pytest.raises(Rejected, match='request_invalid'):
        reader.connection_metadata('operator', {'actor': 'operator'}, lambda body: body)
    assert calls == []
    with pytest.raises(Rejected, match='permission'):
        reader.connection_metadata('operator', {}, lambda body: body)
