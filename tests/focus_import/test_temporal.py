import copy
import csv
import io
import json
import random
from datetime import timedelta
from decimal import Decimal
from fractions import Fraction

import pytest

from modules.focus_import import temporal
from modules.focus_import.model import ADDITIVE, MAX_ROWS, Rejected, canonical, decode, digest, domain, domains_digest, instant, tenant_id, versions


def source_row(provider='AWS', cost='10', start='2024-09-01T00:00:00Z', end='2024-09-02T00:00:00Z'):
    return {
        'provider': provider, 'service': 'Synthetic compute', 'currency': 'USD', 'account': 'synthetic-account',
        'subaccount': 'synthetic-subaccount', 'consumed_unit': 'Hours', 'pricing_unit': 'Hours',
        'billing_start': '2024-09-01T00:00:00Z', 'billing_end': '2024-10-01T00:00:00Z',
        'charge_start': start, 'charge_end': end, 'billed_cost': cost, 'effective_cost': cost,
        'list_cost': cost, 'contracted_cost': '', 'consumed_quantity': '2', 'pricing_quantity': '2',
        'list_unit_price': '5', 'contracted_unit_price': '',
    }


def encoded(rows, version='1.0'):
    mapping = versions()[version]['columns']
    output = io.StringIO(newline='')
    writer = csv.DictWriter(output, fieldnames=list(mapping.values()))
    writer.writeheader()
    for row in rows:
        writer.writerow({mapping[k]: v for k, v in row.items()})
    return output.getvalue().encode()


def configuration(rows=None):
    rows = rows or [source_row(), source_row(provider='Microsoft', cost='20')]
    data = encoded(rows)
    normalized = decode(data, versions()['1.0'])
    tenants = []
    for key, provider in [('north', 'AWS'), ('south', 'Microsoft')]:
        assignments = {canonical(domain(r)) for r in normalized if r['provider'] == provider}
        tenants.append({
            'key': key, 'label': 'Synthetic ' + key, 'timezone': 'UTC', 'reporting_currency': 'USD',
            'currencies': ['USD', None], 'retention_days': 30, 'actor': 'reader-' + key, 'active': True,
            'assignments': [json.loads(a) for a in sorted(assignments)],
        })
    return {
        'source': 'synthetic-temporal', 'version': '1.0', 'revision': 1, 'tenants': tenants,
        'snapshot': {'file_digest': digest(data), 'row_count': len(normalized), 'domains_digest': domains_digest(normalized)},
    }


def descriptor(data, grain='day', revision=1):
    rows = []
    try:
        rows = decode(data, versions()['1.0'])
    except Rejected as exc:
        assert str(exc) == 'empty_input'
    end = '2024-09-02T00:00:00Z' if grain == 'day' else '2024-10-01T00:00:00Z'
    scope = temporal.window('2024-09-01T00:00:00Z', end, grain)
    return {
        'window': scope, 'revision': revision, 'file_digest': digest(data), 'row_count': len(rows),
        'domains_digest': domains_digest(rows), 'complete_through': end, 'finalized': True,
    }


def policy():
    return {
        'revision': 1, 'basis': 'received_at', 'hot_days': 2, 'total_days': 30,
        'grace_days': 2, 'backup_days': 7, 'reject_before': '2024-01-01T00:00:00Z',
        'approved_by': 'lifecycle-approver',
    }


@pytest.mark.parametrize(('start', 'end', 'grain'), [
    ('2024-02-29T00:00:00Z', '2024-03-01T00:00:00Z', 'day'),
    ('2024-02-01T00:00:00Z', '2024-03-01T00:00:00Z', 'month'),
    ('2024-12-01T00:00:00Z', '2025-01-01T00:00:00Z', 'month'),
])
def test_calendar_window_identity(start, end, grain):
    observed = temporal.window(start, end, grain)
    assert observed['start'] == start.replace('Z', '+00:00')
    assert observed['end'] == end.replace('Z', '+00:00')
    assert observed == temporal.window(observed['start'], observed['end'], grain)
    assert len(observed['key']) == 64


@pytest.mark.parametrize(('start', 'end', 'grain'), [
    ('2024-02-29T00:00:00', '2024-03-01T00:00:00Z', 'day'),
    ('2024-02-29T00:00:00+09:00', '2024-03-01T00:00:00Z', 'day'),
    ('2024-02-29T00:00:00Z', '2024-03-02T00:00:00Z', 'day'),
    ('2024-02-29T00:00:00Z', '2024-02-29T00:00:00Z', 'day'),
    ('2024-02-01T00:00:00Z', '2024-03-01T00:00:00Z', 'year'),
    ('2024-02-02T00:00:00Z', '2024-03-02T00:00:00Z', 'month'),
    ('2024-02-01T00:00:00Z', '2024-02-29T00:00:00Z', 'month'),
    ('2024-02-01T00:00:00.000001Z', '2024-02-02T00:00:00Z', 'day'),
])
def test_invalid_calendar_windows(start, end, grain):
    with pytest.raises(Rejected):
        temporal.window(start, end, grain)


def test_half_open_overlap_includes_month_and_excludes_adjacent_day():
    day = temporal.window('2024-09-01T00:00:00Z', '2024-09-02T00:00:00Z', 'day')
    next_day = temporal.window('2024-09-02T00:00:00Z', '2024-09-03T00:00:00Z', 'day')
    month = temporal.window('2024-09-01T00:00:00Z', '2024-10-01T00:00:00Z', 'month')
    assert temporal.overlaps(day, day)
    assert temporal.overlaps(day, month)
    assert temporal.overlaps(month, next_day)
    assert not temporal.overlaps(day, next_day)
    assert day['key'] != month['key']


def test_delivery_assigns_provider_qualified_domains_preserving_equal_rows():
    sources = [source_row(), source_row(), source_row(provider='Microsoft', cost='20')]
    data = encoded(sources)
    rows = temporal.validate_delivery(data, '1.0', configuration(sources), descriptor(data))
    assert [r['tenant_id'] for r in rows] == [tenant_id('north'), tenant_id('north'), tenant_id('south')]
    assert [r['ordinal'] for r in rows] == [1, 2, 3]
    assert rows[0]['source_json'] == rows[1]['source_json']
    assert rows[0]['contracted_cost'] is None


def test_header_only_empty_complete_correction():
    data = encoded([])
    observed = temporal.validate_delivery(data, '1.0', configuration(), descriptor(data, revision=2))
    assert observed == []
    bad = descriptor(data, revision=2)
    bad['row_count'] = 1
    with pytest.raises(Rejected, match='empty_input'):
        temporal.validate_delivery(data, '1.0', configuration(), bad)


@pytest.mark.parametrize('version', ['1.0', '1.2', '1.3'])
def test_delivery_version_dispatch(version):
    data = encoded([source_row()], version)
    reference = encoded([source_row()])
    desc = descriptor(reference)
    desc['file_digest'] = digest(data)
    rows = temporal.validate_delivery(data, version, configuration(), desc)
    assert rows[0]['provider'] == 'AWS'
    assert rows[0]['billed_cost'] == Decimal('10')


def test_extra_registry_version_has_no_runtime_version_branch(monkeypatch):
    registry = versions()
    registry['9.9'] = copy.deepcopy(registry['1.0'])
    monkeypatch.setattr(temporal, 'versions', lambda: registry)
    data = encoded([source_row()])
    assert temporal.validate_delivery(data, '9.9', configuration(), descriptor(data))[0]['provider'] == 'AWS'
    with pytest.raises(Rejected, match='unsupported_version'):
        temporal.validate_delivery(data, '8.8', configuration(), descriptor(data))


@pytest.mark.parametrize('change', ['digest', 'count', 'domains', 'extra', 'missing', 'revision', 'finalized', 'through', 'window'])
def test_delivery_descriptor_guards(change):
    data = encoded([source_row()])
    desc = descriptor(data)
    if change == 'digest':
        desc['file_digest'] = '0' * 64
    elif change == 'count':
        desc['row_count'] = 2
    elif change == 'domains':
        desc['domains_digest'] = '0' * 64
    elif change == 'extra':
        desc['complete'] = True
    elif change == 'missing':
        del desc['finalized']
    elif change == 'revision':
        desc['revision'] = True
    elif change == 'finalized':
        desc['finalized'] = 'true'
    elif change == 'through':
        desc['complete_through'] = '2024-09-01T12:00:00Z'
    else:
        desc['window']['key'] = '0' * 64
    with pytest.raises(Rejected):
        temporal.validate_delivery(data, '1.0', configuration(), desc)


def test_domain_assignment_cannot_be_ambiguous_across_old_periods():
    config = configuration()
    ambiguous = config['tenants'][0]['assignments'][0][:3] + ['2024-10-01T00:00:00+00:00', '2024-11-01T00:00:00+00:00']
    config['tenants'][1]['assignments'].append(ambiguous)
    data = encoded([source_row()])
    with pytest.raises(Rejected, match='ambiguous_delivery_owner'):
        temporal.validate_delivery(data, '1.0', config, descriptor(data))


@pytest.mark.parametrize('change', ['inactive', 'currency', 'subaccount'])
def test_delivery_denied_domains_and_currency(change):
    config = configuration()
    sources = [source_row()]
    if change == 'inactive':
        config['tenants'][0]['active'] = False
    elif change == 'currency':
        sources[0]['currency'] = 'CNY'
    else:
        sources[0]['subaccount'] = 'unregistered'
    data = encoded(sources)
    with pytest.raises(Rejected, match='unmapped_or_denied_source'):
        temporal.validate_delivery(data, '1.0', config, descriptor(data))


def test_day_spill_and_as_of_guard_but_month_allows_charge_lag():
    data = encoded([source_row(end='2024-09-03T00:00:00Z')])
    with pytest.raises(Rejected, match='row_outside_delivery_window'):
        temporal.validate_delivery(data, '1.0', configuration(), descriptor(data))
    desc = descriptor(data)
    desc['finalized'] = False
    desc['complete_through'] = '2024-09-01T12:00:00Z'
    data = encoded([source_row()])
    desc['file_digest'] = digest(data)
    desc['domains_digest'] = descriptor(data)['domains_digest']
    with pytest.raises(Rejected, match='row_after_complete_through'):
        temporal.validate_delivery(data, '1.0', configuration(), desc)
    data = encoded([source_row(start='2024-10-02T00:00:00Z', end='2024-10-03T00:00:00Z')])
    desc = descriptor(data, 'month')
    desc['finalized'] = False
    desc['complete_through'] = '2024-09-10T00:00:00Z'
    assert temporal.validate_delivery(data, '1.0', configuration(), desc)[0]['charge_start'].month == 10
    outside = source_row()
    outside['billing_start'] = '2024-08-01T00:00:00Z'
    data = encoded([outside])
    with pytest.raises(Rejected, match='row_outside_delivery_window'):
        temporal.validate_delivery(data, '1.0', configuration(), descriptor(data, 'month'))


def test_retention_boundaries_use_immutable_supplied_origin():
    p = policy()
    expected = {
        'hot_until': '2024-09-03T00:00:00+00:00',
        'delete_after': '2024-10-03T00:00:00+00:00',
        'backup_until': '2024-10-10T00:00:00+00:00',
    }
    origin = '2024-09-01T00:00:00Z'
    assert temporal.eligible(origin, p, '2024-09-02T00:00:00Z') == expected
    assert temporal.eligible(origin, p, '2024-10-15T00:00:00Z') == expected
    p['basis'] = 'window_end'
    assert temporal.eligible(origin, p, '2024-08-20T00:00:00Z') == expected


@pytest.mark.parametrize('field,value', [
    ('revision', True), ('basis', 'copy_time'), ('hot_days', 31), ('total_days', 0), ('total_days', 1.5),
    ('grace_days', -1), ('backup_days', True), ('backup_days', 36501), ('approved_by', ''),
    ('reject_before', '2024-01-01T00:00:00'),
])
def test_retention_policy_rejects_implicit_or_invalid_values(field, value):
    p = policy()
    p[field] = value
    with pytest.raises(Rejected):
        temporal.validate_policy(p)


def test_reporting_timezone_midnight_and_dst_preserve_occurrences():
    starts = ['2024-11-03T03:59:00Z', '2024-11-03T04:00:00Z', '2024-11-03T05:30:00Z', '2024-11-03T06:30:00Z']
    sources = [source_row(start=s, end=(instant(s) + timedelta(minutes=1)).isoformat()) for s in starts]
    rows = decode(encoded(sources), versions()['1.0'])
    reported = temporal.summarize(rows, 'America/New_York', 'day')
    assert {k: sum(g['rows'] for g in v.values()) for k, v in reported.items()} == {'2024-11-02': 1, '2024-11-03': 3}
    assert sum(g['rows'] for g in temporal.summarize(rows, 'America/New_York', 'month')['2024-11'].values()) == 4
    spanning = decode(encoded([source_row(start='2024-09-01T23:00:00Z', end='2024-09-03T01:00:00Z')]), versions()['1.0'])
    assert list(temporal.summarize(spanning, 'UTC', 'day')) == ['2024-09-01']


def test_generated_reporting_totals_match_independent_fraction_oracle():
    randomizer = random.Random(270031)
    sources = []
    for index in range(800):
        row = source_row(cost=f'{randomizer.randrange(-99999, 100000) / 100:.2f}')
        row['currency'] = randomizer.choice(['USD', 'CNY'])
        row['consumed_unit'] = randomizer.choice(['Hours', 'GB'])
        row['pricing_unit'] = row['consumed_unit']
        row['contracted_cost'] = '' if index % 3 else row['billed_cost']
        row['consumed_quantity'] = '' if index % 5 else '0.000000000000000001'
        row['charge_start'] = f'2024-09-{1 + index % 9:02d}T23:30:00Z'
        row['charge_end'] = f'2024-09-{2 + index % 9:02d}T00:00:00Z'
        sources.append(row)
    rows = decode(encoded(sources), versions()['1.0'])
    observed = temporal.summarize(rows, 'Asia/Seoul', 'month')['2024-09']
    expected = {}
    for row in sources:
        key = canonical([row[k] for k in ['provider', 'service', 'currency', 'consumed_unit', 'pricing_unit']])
        group = expected.setdefault(key, {name: [] for name in ADDITIVE})
        for name in ADDITIVE:
            group[name].append(None if row[name] == '' else Fraction(row[name]))
    assert set(observed) == set(expected)
    for key, measures in expected.items():
        for name, values in measures.items():
            actual = observed[key]['measures'][name]
            present = [value for value in values if value is not None]
            assert actual['present'] == len(present)
            assert actual['missing'] == len(values) - len(present)
            assert (Fraction(actual['coefficient'], 10**18) if actual['coefficient'] is not None else None) == (sum(present, Fraction(0)) if present else None)


def test_selection_bound_is_across_all_deliveries_not_each_one():
    rows = decode(encoded([source_row(cost='60000000000000000000')]), versions()['1.0'])
    assert temporal.summarize(rows, 'UTC', 'month')
    with pytest.raises(Rejected, match='selection_aggregate_bound'):
        temporal.summarize(rows + copy.deepcopy(rows), 'UTC', 'month')
    small = decode(encoded([source_row()]), versions()['1.0'])
    with pytest.raises(Rejected, match='selection_row_limit'):
        temporal.summarize(small * (MAX_ROWS + 1), 'UTC', 'day')


@pytest.mark.parametrize(('timezone', 'grain'), [('Invalid/Zone', 'day'), ('UTC', 'year'), (None, 'month')])
def test_reporting_invalid_scope(timezone, grain):
    with pytest.raises(Rejected):
        temporal.summarize([], timezone, grain)
