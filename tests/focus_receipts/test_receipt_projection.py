import copy
from contextlib import contextmanager
from types import SimpleNamespace

import pytest

from modules.focus_import.model import Rejected, canonical, digest, tenant_id
from modules.focus_import.receipt_metadata import RECORD_FIELDS
from modules.focus_import.receipt_projection import MAX_RESPONSE_BYTES, project, receipts, selection


ENVELOPE = {'schema', 'profile', 'read_only', 'financial_authority', 'commands', 'view', 'selection', 'audience',
            'member_count', 'auth_scope', 'permission_revision', 'snapshot', 'observed_at', 'total', 'next_after', 'capabilities', 'records'}


def test_scoped_actual_metadata_and_closed_projection(receipt_fixture):
    f = receipt_fixture
    own = f.add_delivery()
    f.add_delivery(connection='south')
    f.add_delivery(connection='mixed', positive={'north-a': 1, 'south-a': 1})
    f.add_quarantine(connection='south')
    before = copy.deepcopy(f.state)
    body = f.render()
    assert set(body) == ENVELOPE
    assert body['total'] == 1 and body['records'][0]['id'] == own
    row = body['records'][0]
    assert set(row) == set(RECORD_FIELDS) and len(RECORD_FIELDS) == 22
    assert row['declared_rows'] == '2' and row['file_bytes'] == '128' and row['source_revision'] == '1'
    assert row['stage'] == 'sealed' and row['format'] == 'csv'
    assert all(row[k] is None for k in ['published', 'finalized', 'complete_through', 'known_zero', 'attempts'])
    assert body['read_only'] is True and body['financial_authority'] is False and body['commands'] == 'unconnected'
    assert body['member_count'] == 2 and body['audience'] == 'company'
    assert not any(secret in str(body) for secret in ['DO-NOT-EXPOSE', 'slices', 'tenant_keys', 'assignments', 'policies', 'manifest'])
    assert f.state == before


def test_legacy_full_positive_membership_and_exact_manifest(receipt_fixture):
    f = receipt_fixture
    own = f.add_delivery(connection=None, positive={'north-a': 2})
    f.add_delivery(revision=2, connection=None, positive={'north-a': 1, 'south-a': 1})
    f.add_delivery(revision=3, connection=None, count=0, positive={})
    f.add_delivery(revision=4, connection=None, stage='registered')
    body = f.render()
    assert [r['id'] for r in body['records']] == [own]
    assert body['records'][0]['connection'] is None
    batch = next(b for b in f.state['metadata']['batch'] if b['batch_id'] == f.state['deliveries'][own]['batch_id'])
    batch['manifest']['row_count'] = 3
    with pytest.raises(Rejected, match='receipt_evidence_unavailable'):
        f.render()


def test_foreign_and_unowned_activity_never_changes_pins_or_search(receipt_fixture):
    f = receipt_fixture
    f.add_delivery()
    before = f.render()
    foreign = f.add_delivery(revision=2, connection='south')
    f.add_quarantine(connection='south')
    rejected = f.add_quarantine(revision=3, admitted=False, stage='rejected')
    f.state['global_sequence'] = 999999
    f.state['metadata']['foreign-secret'] = {'raw': 'unrelated'}
    after = f.render()
    for key in ['records', 'total', 'snapshot', 'permission_revision', 'auth_scope']:
        assert before[key] == after[key]
    assert f.render({'q': foreign})['total'] == 0
    assert f.render({'view': 'quarantine', 'q': rejected})['total'] == 0
    f.state['receipts'][rejected]['descriptor'] = {'invalid': 'unowned'}
    assert f.render({'view': 'quarantine'})['total'] == 0


def test_foreign_configuration_changes_do_not_change_public_pins(receipt_fixture):
    f = receipt_fixture
    f.add_delivery(stage='registered')
    before = f.render()
    f.config['tenants'][2]['timezone'] = 'Asia/Seoul'
    changed = digest(canonical(f.config).encode())
    f.state['config_digest'] = changed
    f.state['metadata']['configuration'] = [{'body': copy.deepcopy(f.config), 'digest': changed}]
    f.authority.bindings['config_digest'] = changed
    after = f.render()
    assert before['snapshot'] == after['snapshot']
    assert before['permission_revision'] == after['permission_revision']


def test_current_membership_and_connection_bindings_are_required(receipt_fixture):
    f = receipt_fixture
    f.add_delivery()
    first = f.render()
    f.context['tenants'][0]['active'] = False
    with pytest.raises(Rejected, match='receipt_evidence_unavailable'):
        f.render()
    f.context['tenants'][0]['active'] = True
    f.state['connections']['north']['domains'].append(['AWS', 'south-a', None])
    now = f.render()
    assert now['records'] == [] and now['snapshot'] != first['snapshot']
    assert now['permission_revision'] != first['permission_revision']


def test_quarantine_safe_metadata_and_invalid_declarations_are_nullable(receipt_fixture):
    f = receipt_fixture
    qid = f.add_quarantine(descriptor_changes={'revision': 0, 'row_count': 'invalid'})
    foreign = f.add_quarantine(revision=3, connection='south')
    body = f.render({'view': 'quarantine'})
    assert body['total'] == 1
    row = body['records'][0]
    assert row['id'] == qid and row['reason'] == 'delivery_coverage'
    assert row['source_revision'] is None and row['declared_rows'] is None
    assert row['file_bytes'] is None and row['format'] is None and row['mapping_digest'] is None
    assert row['root_receipt'] == qid and row['attempts'] == '0'
    f.state['receipts'][qid]['replaced_by'] = foreign
    assert f.render({'view': 'quarantine'})['records'][0]['replaced_by'] is None


def test_lineage_requires_independent_scope_window_and_root_counter(receipt_fixture):
    f = receipt_fixture
    root = f.add_quarantine()
    child = f.add_quarantine(revision=2)
    delivery = f.add_delivery(revision=3)
    f.state['receipts'][root]['attempts'] = 2
    f.state['receipts'][child]['root_receipt'] = root
    f.state['receipts'][child]['attempts'] = 0
    f.state['receipts'][child]['replaced_by'] = delivery
    rows = {r['id']: r for r in f.render({'view': 'quarantine'})['records']}
    assert rows[child]['root_receipt'] == root and rows[child]['attempts'] == '2'
    assert rows[child]['replaced_by'] == delivery
    foreign = f.add_quarantine(revision=4, connection='south')
    f.state['receipts'][child]['root_receipt'] = foreign
    rows = {r['id']: r for r in f.render({'view': 'quarantine'})['records']}
    assert rows[child]['root_receipt'] is None and rows[child]['attempts'] is None


def test_coverage_latest_stage_is_separate_from_current_publication(receipt_fixture):
    f = receipt_fixture
    f.add_delivery()
    f.add_delivery(revision=2, stage='registered')
    own = next(r for r in f.render({'view': 'completeness'})['records'] if r['connection'] == 'north')
    assert own['stage'] == 'registered' and own['published'] is True
    assert own['finalized'] is True and own['complete_through'] == f.window['end']
    assert own['known_zero'] is False and own['source_revision'] is None and own['received_at'] is None


def test_missing_and_late_are_registered_expectations_not_zero(receipt_fixture):
    f = receipt_fixture
    body = f.render({'view': 'completeness'})
    assert body['capabilities']['completeness'] == 'registered-expectations-only'
    row = body['records'][0]
    assert row['stage'] == 'late' and row['published'] is False
    assert all(row[k] is None for k in ['finalized', 'complete_through', 'known_zero', 'declared_rows', 'received_at'])
    f.at = '2026-01-02T00:00:00+00:00'
    assert f.render({'view': 'completeness'})['records'][0]['stage'] == 'missing'


def test_no_safe_completeness_and_no_extraction_are_unknown_not_zero(receipt_fixture):
    f = receipt_fixture
    f.state['connections'] = {}
    for view in ['completeness', 'extraction']:
        body = f.render({'view': view})
        assert body['total'] is None and body['records'] == [] and body['next_after'] is None
    assert f.render({'view': 'completeness'})['capabilities']['completeness'] == 'unavailable'
    assert f.render({'view': 'extraction'})['capabilities']['extraction'] == 'unconnected'


def test_explicit_published_zero_domains_only(receipt_fixture):
    f = receipt_fixture
    key = f.add_delivery(count=0, positive={})
    item = f.state['deliveries'][key]
    descriptor = item['descriptor']
    descriptor['zero_domains'] = [{'domain': domain, 'reason': 'source_zero', 'evidence_digest': 'e' * 64} for domain in f.state['connections']['north']['domains']]
    new_key = digest(canonical({'version': '1.0', 'descriptor': descriptor}).encode())
    f.state['deliveries'][new_key] = f.state['deliveries'].pop(key)
    target = next(w for w in f.state['windows'].values() if w.get('connection_id') == 'north')
    target['current'] = new_key
    target['revisions'] = [new_key]
    row = f.render({'view': 'completeness'})['records'][0]
    assert row['known_zero'] is True and row['published'] is True


def test_bounded_pages_use_same_facts_pin_and_no_overlapping_rows(receipt_fixture):
    f = receipt_fixture
    for revision in range(1, 52):
        f.add_delivery(revision=revision)
    first = f.render()
    assert first['total'] == 51 and len(first['records']) == 25 and first['next_after'] == 25
    second = f.render({'after': '25', 'snapshot': first['snapshot'], 'permission_revision': first['permission_revision']})
    assert len(second['records']) == 25 and second['next_after'] == 50
    assert second['snapshot'] == first['snapshot']
    assert not {r['id'] for r in first['records']} & {r['id'] for r in second['records']}
    last = f.render({'after': '50', 'snapshot': first['snapshot'], 'permission_revision': first['permission_revision']})
    assert len(last['records']) == 1 and last['next_after'] is None
    f.add_delivery(revision=52)
    with pytest.raises(Rejected, match='query_snapshot_changed'):
        f.render({'after': '25', 'snapshot': first['snapshot'], 'permission_revision': first['permission_revision']})


@pytest.mark.parametrize('params', [
    {'actor': 'operator'}, {'tenant': 'south'}, {'view': 'results'}, {'q': 'ABC'}, {'q': '../path'}, {'q': 'a' * 129},
    {'after': '-1'}, {'after': '01'}, {'after': '10001'}, {'after': '1.0'}, {'after': '25'},
    {'snapshot': 'a' * 64}, {'permission_revision': 'a' * 64}, {'snapshot': 'bad', 'permission_revision': 'a' * 64},
])
def test_invalid_queries_fail_closed(params):
    with pytest.raises(Rejected):
        selection(params)


def test_changed_record_and_permission_values_invalidate_pins(receipt_fixture):
    f = receipt_fixture
    f.add_delivery(stage='registered')
    first = f.render()
    f.state['policies']['north-a']['revision'] = 2
    with pytest.raises(Rejected, match='query_snapshot_changed'):
        f.render({'snapshot': first['snapshot'], 'permission_revision': first['permission_revision']})
    first = f.render()
    f.state['deliveries'][first['records'][0]['id']]['received_at'] = '2026-01-04T00:00:00+00:00'
    assert f.render()['snapshot'] != first['snapshot']


def test_response_bound_and_out_of_range_offset(receipt_fixture, monkeypatch):
    f = receipt_fixture
    f.add_delivery()
    first = f.render()
    with pytest.raises(Rejected, match='query_page'):
        f.render({'after': '2', 'snapshot': first['snapshot'], 'permission_revision': first['permission_revision']})
    assert MAX_RESPONSE_BYTES == 2 * 1024 * 1024
    monkeypatch.setattr('modules.focus_import.receipt_projection.MAX_RESPONSE_BYTES', 16)
    with pytest.raises(Rejected, match='response_bound'):
        f.render()


def test_read_guard_scopes_callback_and_no_body_or_mutation(receipt_fixture, monkeypatch):
    f = receipt_fixture
    f.add_delivery()
    events = []
    before = copy.deepcopy(f.state)
    @contextmanager
    def transaction(right):
        assert right == 'query'
        events.append('transaction-enter')
        yield {'source': f.config['source']}, f.context
        events.append('transaction-exit')
    @contextmanager
    def access(actor, right):
        assert actor == f.authority.actor and right == 'query'
        events.append('access-enter')
        f.authority.transaction = transaction
        yield SimpleNamespace(state=f.state), SimpleNamespace(a=f.authority), object()
        events.append('access-exit')
    monkeypatch.setattr('modules.focus_import.receipt_projection.collection.access', access)
    monkeypatch.setattr('modules.focus_import.collection.query', lambda *a, **k: pytest.fail('warehouse query must not run'))
    def deliver(body):
        assert events == ['access-enter', 'transaction-enter']
        events.append('deliver')
        return body
    result = receipts(f.authority.actor, {}, deliver)
    assert result['total'] == 1
    assert events == ['access-enter', 'transaction-enter', 'deliver', 'transaction-exit', 'access-exit']
    assert f.state == before


def test_tenant_actor_never_expands_to_company_scope(receipt_fixture):
    f = receipt_fixture
    f.add_delivery()
    own = f.config['tenants'][0]
    f.authority = SimpleNamespace(actor=own['actor'], is_view=False, tenant=tenant_id(own['key']), bindings=f.authority.bindings)
    f.context = {'active': True, 'properties': copy.deepcopy(own)}
    body = f.render()
    assert body['audience'] == 'tenant' and body['member_count'] == 1
    assert body['total'] == 0 and body['records'] == []
    with pytest.raises(Rejected, match='permission'):
        project('operator', {}, f.authority, f.context, f.state, f.at)


@pytest.mark.parametrize('change', ['missing-attestation', 'wrong-domain', 'duplicate-attestation', 'invalid-evidence'])
def test_published_zero_cannot_be_inferred_without_exact_attestations(receipt_fixture, change):
    f = receipt_fixture
    key = f.add_delivery(count=0, positive={})
    descriptor = f.state['deliveries'][key]['descriptor']
    if change != 'missing-attestation':
        descriptor['zero_domains'] = [{'domain': d, 'reason': 'source_zero', 'evidence_digest': 'e' * 64} for d in f.state['connections']['north']['domains']]
        if change == 'wrong-domain':
            descriptor['zero_domains'][0]['domain'] = ['AWS', 'south-a', None]
        elif change == 'duplicate-attestation':
            descriptor['zero_domains'][1] = descriptor['zero_domains'][0]
        else:
            descriptor['zero_domains'][0]['evidence_digest'] = 'invalid'
        new_key = digest(canonical({'version': '1.0', 'descriptor': descriptor}).encode())
        f.state['deliveries'][new_key] = f.state['deliveries'].pop(key)
        target = next(w for w in f.state['windows'].values() if w.get('connection_id') == 'north')
        target['current'] = new_key
        target['revisions'] = [new_key]
    with pytest.raises(Rejected, match='receipt_evidence_unavailable'):
        f.render({'view': 'completeness'})


@pytest.mark.parametrize('change', ['digest', 'revision', 'date', 'stage', 'rows', 'version'])
def test_owned_malformed_metadata_fails_instead_of_projecting(receipt_fixture, change):
    f = receipt_fixture
    key = f.add_delivery(stage='registered')
    d = f.state['deliveries'][key]
    if change == 'digest':
        d['mapping_digest'] = 'invalid'
    elif change == 'revision':
        d['descriptor']['revision'] = 2**63
    elif change == 'date':
        d['received_at'] = '2026-99-01T00:00:00Z'
    elif change == 'stage':
        d['state'] = 'succeeded'
    elif change == 'rows':
        d['descriptor']['row_count'] = True
    else:
        d['version'] = 'unknown'
    with pytest.raises(Rejected):
        f.render()


def test_explicit_connection_does_not_override_actual_foreign_manifest_members(receipt_fixture):
    f = receipt_fixture
    f.add_delivery(connection='north', positive={'north-a': 1, 'south-a': 1})
    body = f.render()
    assert body['records'] == [] and body['total'] == 0
    complete = f.render({'view': 'completeness'})
    assert complete['total'] is None and complete['capabilities']['completeness'] == 'unavailable'


def test_operator_context_never_projects_whole_journal(monkeypatch):
    @contextmanager
    def access(actor, right):
        assert actor == 'operator' and right == 'query'
        yield SimpleNamespace(state={'private': 'not-projected'}), None, object()
    monkeypatch.setattr('modules.focus_import.receipt_projection.collection.access', access)
    with pytest.raises(Rejected, match='permission'):
        receipts('operator', {}, lambda body: pytest.fail('operator data must not be delivered'))


def test_source_binding_is_part_of_authorized_scope(receipt_fixture):
    f = receipt_fixture
    before = f.render()
    f.authority.bindings['source'] = 'foreign-source'
    with pytest.raises(Rejected, match='receipt_evidence_unavailable'):
        f.render()
    f.config['source'] = 'foreign-source'
    changed = digest(canonical(f.config).encode())
    f.state['config_digest'] = changed
    f.state['metadata']['configuration'] = [{'body': copy.deepcopy(f.config), 'digest': changed}]
    f.authority.bindings['config_digest'] = changed
    after = f.render()
    assert before['auth_scope'] != after['auth_scope']
