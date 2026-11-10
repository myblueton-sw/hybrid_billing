import copy
import json
from datetime import datetime, timedelta, timezone
from types import SimpleNamespace

import pytest

from modules.focus_import import collection_status as status
from modules.focus_import.model import Rejected, canonical, digest, tenant_id, versions
from modules.focus_import.receipt_projection import project as receipt_project
from modules.focus_import.temporal import window


FIELDS = {'schema', 'profile', 'read_only', 'financial_authority', 'audience', 'member_count', 'auth_scope',
          'permission_revision', 'snapshot', 'observed_at', 'row_count', 'delivery_count', 'versions', 'provenance',
          'providers', 'periods', 'upstream_completeness', 'source_classification', 'original_availability'}


def test_exact_public_summary_reuses_receipt_authority_and_never_exposes_private_query_data(status_fixture):
    f = status_fixture
    f.scope()
    body = f.status([f.row()])
    receipt = receipt_project(f.authority.actor, {}, f.authority, f.context, f.state, f.at)
    assert set(body) == FIELDS and len(FIELDS) == 19
    assert all(body[key] == receipt[key] for key in ['audience', 'member_count', 'auth_scope', 'permission_revision'])
    assert body['profile'] == 'retained-focus-collection-status-v1' and body['schema'] == 1
    assert body['read_only'] is True and body['financial_authority'] is False
    assert body['row_count'] == body['delivery_count'] == 1 and body['versions'] == ['1.0']
    assert body['providers'] == [{'provider': 'AWS', 'row_count': 1}]
    assert body['periods'] == [{'window_start': f.window['start'], 'window_end': f.window['end'], 'row_count': 1}]
    assert body['provenance'] == {'official-public': 1, 'official-public-subset': 0, 'synthetic-contract-fixture': 0, 'unclassified': 0}
    assert body['upstream_completeness'] == 'unverified' and body['original_availability'] == 'not-checked'
    assert body['source_classification'] == 'exploratory-query-only'
    assert 'DO-NOT-EXPOSE' not in json.dumps(body) and 'PRIVATE' not in json.dumps(body)
    assert body['snapshot'] == digest(canonical({k: v for k, v in body.items() if k not in {'snapshot', 'observed_at'}}).encode())


def test_only_rows_that_contribute_supply_delivery_version_and_merged_source_periods(status_fixture):
    f = status_fixture
    f.scope(batch='batch-a')
    f.scope(batch='batch-b', tenant='north-b', version='1.2')
    f.scope(batch='zero-slice', version='UNRELATED-FOREIGN-VERSION', period={'PRIVATE': 'FOREIGN-PERIOD'})
    rows = [f.row(provider='Azure', batch='batch-a', ordinal=1),
            f.row(provider='AWS', batch='batch-a', ordinal=2, provenance='official-public-subset'),
            f.row(provider='AWS', batch='batch-b', tenant='north-b', provenance='synthetic-contract-fixture')]
    body = f.status(rows)
    assert body['row_count'] == 3 and body['delivery_count'] == 2 and body['versions'] == ['1.0', '1.2']
    assert body['providers'] == [{'provider': 'AWS', 'row_count': 2}, {'provider': 'Azure', 'row_count': 1}]
    assert body['periods'] == [{'window_start': f.window['start'], 'window_end': f.window['end'], 'row_count': 3}]
    assert sum(body['provenance'].values()) == sum(p['row_count'] for p in body['providers']) == sum(p['row_count'] for p in body['periods']) == body['row_count']
    assert len(body['periods']) <= body['delivery_count'] <= body['row_count']
    assert 0 < len(body['versions']) <= body['delivery_count']


def test_empty_verified_selection_is_exact_zero_not_fabricated_delivery_metadata(status_fixture):
    f = status_fixture
    f.scope(version='IGNORED-ZERO-SLICE')
    body = f.status([])
    assert body['row_count'] == body['delivery_count'] == 0
    assert body['versions'] == body['providers'] == body['periods'] == []
    assert body['provenance'] == dict.fromkeys(status.PROVENANCE, 0)


def test_foreign_only_delivery_with_authorized_zero_slices_has_no_public_footprint(status_fixture):
    f = status_fixture
    baseline = f.status([])
    delivery_id = f.add_delivery(connection=None, count=9, positive={'south-a': 9})
    foreign = f.state['deliveries'][delivery_id]
    f.scopes = [('PRIVATE-ZERO-WINDOW', foreign, {'tenant_id': str(tenant_id('north-a')), 'batch_id': foreign['batch_id']})]
    first = f.status([])
    foreign.update(version='99.99', received_at='PRIVATE-FOREIGN-TIME', file_bytes=999999, private='PRIVATE-FOREIGN')
    foreign['descriptor']['window'] = {'PRIVATE': 'FOREIGN-WINDOW'}
    second = f.status([], result={'classification': 'exploratory-query-only', 'row_count': 0, 'snapshot': 'CHANGED-PRIVATE-SOURCE-SNAPSHOT'})
    for key in FIELDS - {'observed_at'}:
        assert baseline[key] == first[key] == second[key]


def test_private_source_snapshot_scope_order_and_observation_time_do_not_change_public_snapshot(status_fixture):
    f = status_fixture
    f.scope(batch='a')
    f.scope(batch='b', tenant='north-b')
    rows = [f.row(batch='a'), f.row(batch='b', tenant='north-b', provider='Azure')]
    first = f.status(rows)
    f.scopes.reverse()
    reader = f.projection()
    reader.observed_at = '2026-11-10T12:00:00.123456+00:00'
    second = reader.consume({'classification': 'exploratory-query-only', 'row_count': 2, 'snapshot': 'NEW-PRIVATE-HASH'}, list(reversed(rows)))
    assert first['snapshot'] == second['snapshot'] and first['observed_at'] != second['observed_at']
    rows[1][4]['source_json']['private'] = 'CHANGED-HIDDEN'
    assert f.status(rows)['snapshot'] == first['snapshot']
    rows[1][4]['provider'] = 'GCP'
    assert f.status(rows)['snapshot'] != first['snapshot']


def test_periods_are_source_windows_sorted_and_merged_not_charge_months(status_fixture):
    f = status_fixture
    feb = window('2026-02-01T00:00:00+00:00', '2026-03-01T00:00:00+00:00', 'month')
    f.scope(batch='feb', period=feb)
    f.scope(batch='jan')
    rows = [f.row(batch='feb'), f.row(batch='jan')]
    for row in rows:
        row[4]['charge_start'] = '2021-04-11T00:00:00Z'
    assert f.status(rows)['periods'] == [
        {'window_start': f.window['start'], 'window_end': f.window['end'], 'row_count': 1},
        {'window_start': feb['start'], 'window_end': feb['end'], 'row_count': 1}]


@pytest.mark.parametrize('value', [None, {}, {'x_HB_EvidenceClass': 'customer-secret'}, {'x_HB_EvidenceClass': ['official-public']}, 'invalid-json', '[]'])
def test_unclassified_provenance_never_echoes_unknown_source_values(status_fixture, value):
    f = status_fixture
    f.scope()
    row = f.row()
    row[4]['source_json'] = value
    body = f.status([row])
    assert body['provenance'] == {'official-public': 0, 'official-public-subset': 0, 'synthetic-contract-fixture': 0, 'unclassified': 1}
    assert 'customer-secret' not in str(body)


def test_json_source_provenance_and_all_four_counts_are_exact(status_fixture):
    f = status_fixture
    f.scope()
    rows = [f.row(ordinal=i + 1, provenance=label) for i, label in enumerate(status.PROVENANCE)]
    rows[0][4]['source_json'] = '{"x_HB_EvidenceClass":"official-public","private":"PRIVATE"}'
    body = f.status(rows)
    assert body['provenance'] == dict.fromkeys(status.PROVENANCE, 1)


@pytest.mark.parametrize('params', [None, [], '', {'q': ''}, {'actor': 'operator'}, {'snapshot': 'a' * 64}, {'provider': 'AWS'}, {'after': '0'}])
def test_no_query_inputs_are_accepted_before_guard_opens(monkeypatch, params):
    def forbidden(*args, **kwargs):
        raise AssertionError('guard must not open')
    monkeypatch.setattr(status.collection, 'query', forbidden)
    with pytest.raises(Rejected, match='request_invalid'):
        status.collection_status('reader-company-north', params)


@pytest.mark.parametrize('change', ['unknown-tenant', 'wrong-row-tenant', 'missing-scope', 'batch-mismatch', 'duplicate-row', 'conflicting-batch-period'])
def test_unverified_or_inconsistent_row_identity_fails_closed(status_fixture, change):
    f = status_fixture
    f.scope()
    row = f.row()
    rows = [row]
    if change == 'unknown-tenant': rows = [f.row(tenant='south-a')]
    if change == 'wrong-row-tenant': row[4]['tenant_id'] = str(tenant_id('north-b'))
    if change == 'missing-scope': f.scopes = []
    if change == 'batch-mismatch': f.scopes[0][1]['batch_id'] = 'PRIVATE-OTHER-BATCH'
    if change == 'duplicate-row': rows.append(row)
    if change == 'conflicting-batch-period':
        f.scope(key='other-window', period=window('2026-02-01T00:00:00Z', '2026-03-01T00:00:00Z', 'month'))
        rows.append(f.row(key='other-window', ordinal=1))
    with pytest.raises(Rejected):
        f.status(rows)


@pytest.mark.parametrize('version', [None, 'unsupported', '9.9', '1.000', [], ''])
def test_contributing_delivery_requires_actual_supported_version(status_fixture, version):
    status_fixture.scope(version=version)
    with pytest.raises(Rejected, match='collection_status_unavailable'):
        status_fixture.status([status_fixture.row()])


@pytest.mark.parametrize('ordinal', [0, -1, True, '1'])
def test_verified_file_ordinals_must_be_positive_integers(status_fixture, ordinal):
    status_fixture.scope()
    with pytest.raises(Rejected, match='collection_status_unavailable'):
        status_fixture.status([status_fixture.row(ordinal=ordinal)])


@pytest.mark.parametrize('provider', [None, '', 'x' * 257, {}, 1])
def test_malformed_provider_cannot_enter_counts(status_fixture, provider):
    status_fixture.scope()
    with pytest.raises(Rejected, match='collection_status_unavailable'):
        status_fixture.status([status_fixture.row(provider=provider)])


@pytest.mark.parametrize('source_window', [{}, None, {'start': 'bad', 'end': 'bad'},
                                         {'start': '2026-01-01T00:00:00Z', 'end': '2026-01-01T00:00:00Z', 'grain': 'month', 'key': 'a' * 64}])
def test_contributing_delivery_requires_verified_source_window(status_fixture, source_window):
    f = status_fixture
    f.scope()['descriptor']['window'] = source_window
    with pytest.raises(Rejected, match='collection_status_unavailable'):
        f.status([f.row()])


@pytest.mark.parametrize('result', [{'classification': 'billing-final', 'row_count': 0},
                                  {'classification': 'exploratory-query-only', 'row_count': True},
                                  {'classification': 'exploratory-query-only', 'row_count': 1}, {}])
def test_failed_or_inconsistent_guard_result_is_not_empty_success(status_fixture, result):
    with pytest.raises(Rejected, match='collection_status_unavailable'):
        status_fixture.status([], result=result)


def test_current_receipt_authority_guard_still_runs_in_preflight(status_fixture):
    f = status_fixture
    f.context['audience']['active'] = False
    with pytest.raises(Rejected):
        f.status([])


def test_fixed_consumer_and_delivery_execute_inside_existing_collection_query_guard(status_fixture, monkeypatch):
    f = status_fixture
    f.scope()
    active = []
    def query(actor, limit, _consumer):
        assert actor == f.authority.actor and limit == 1
        active.append('guard')
        _consumer.preflight(SimpleNamespace(state=f.state), SimpleNamespace(a=f.authority), f.context, f.scopes)
        result = _consumer.consume({'classification': 'exploratory-query-only', 'row_count': 1, 'snapshot': 'PRIVATE'}, [f.row()])
        active.clear()
        return result
    monkeypatch.setattr(status.collection, 'query', query)
    def deliver(body):
        assert active == ['guard'] and body['row_count'] == 1
        return 'delivered'
    assert status.collection_status(f.authority.actor, {}, deliver) == 'delivered'
    assert active == []


def test_bounds_reject_instead_of_truncating_providers_periods_rows_and_bytes(status_fixture, monkeypatch):
    f = status_fixture
    f.scope()
    with pytest.raises(Rejected, match='selection_bound'):
        f.status([f.row(provider=f'provider-{i}', ordinal=i + 1) for i in range(129)])
    rows = []
    f.scopes = []
    for i in range(121):
        start = datetime(2026, 1, 1, tzinfo=timezone.utc) + timedelta(days=i)
        f.scope(batch=f'batch-{i}', period=window(start.isoformat(), (start + timedelta(days=1)).isoformat(), 'day'))
        rows.append(f.row(batch=f'batch-{i}'))
    with pytest.raises(Rejected, match='selection_bound'):
        f.status(rows)
    with pytest.raises(Rejected, match='selection_bound'):
        f.status([f.row(ordinal=i + 1) for i in range(10001)])
    monkeypatch.setattr(status, 'MAX_RESPONSE_BYTES', 1)
    with pytest.raises(Rejected, match='response_bound'):
        f.status([])


def test_supported_versions_are_bounded_independently_of_other_groups(status_fixture, monkeypatch):
    f = status_fixture
    registry = {f'{i // 100}.{i % 100}': {} for i in range(129)}
    monkeypatch.setattr(status, 'versions', lambda: registry)
    rows = []
    for i, version in enumerate(registry):
        f.scope(batch=f'batch-{i}', version=version)
        rows.append(f.row(batch=f'batch-{i}'))
    with pytest.raises(Rejected, match='selection_bound'):
        f.status(rows)
