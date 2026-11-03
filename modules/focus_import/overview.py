import json
import re
from collections import Counter
from datetime import datetime, timezone
from zoneinfo import ZoneInfo

from .collection import query as collected_query
from .model import Rejected, canonical, coefficient, digest


PAGE = 25
BASES = ('billed_cost', 'effective_cost')
KEYS = {'period', 'provider', 'account', 'basis', 'q', 'after', 'account_after', 'snapshot'}


def selection(params):
    if not isinstance(params, dict) or set(params) - KEYS or any(not isinstance(v, str) or len(v) > 4096 for v in params.values()):
        raise Rejected('request_invalid')
    body = {'period': params.get('period', 'all'), 'provider': params.get('provider', ''),
            'account': params.get('account', ''), 'basis': params.get('basis', 'billed_cost'), 'q': params.get('q', '').strip()}
    if body['period'] != 'all' and not re.fullmatch(r'\d{4}-(?:0[1-9]|1[0-2])', body['period']):
        raise Rejected('request_invalid')
    if body['basis'] not in BASES or len(body['q']) > 128 or body['account'] and not re.fullmatch('[a-f0-9]{64}', body['account']):
        raise Rejected('request_invalid')
    for key in ['after', 'account_after']:
        value = params.get(key, '0')
        if not re.fullmatch('[0-9]{1,5}', value) or int(value) > 10000:
            raise Rejected('request_invalid')
        body[key] = int(value)
    expected = params.get('snapshot')
    if expected is not None and not re.fullmatch('[a-f0-9]{64}', expected):
        raise Rejected('request_invalid')
    if (body['after'] or body['account_after']) and expected is None:
        raise Rejected('snapshot_required')
    return body, expected


def measure(rows, name):
    values = [coefficient(r[name]) for r in rows if r[name] is not None]
    return {'coefficient': str(sum(values)) if values else None, 'present': len(values), 'missing': len(rows) - len(values)}


def amounts(rows):
    return [{'currency': currency, 'rows': len(group), **{basis: measure(group, basis) for basis in BASES}}
            for currency in sorted({r['currency'] for r in rows})
            for group in [[r for r in rows if r['currency'] == currency]]]


def account_id(tenant, row):
    return digest(canonical([tenant, row['provider'], row['account'], row['subaccount']]).encode())


def evidence(row):
    try:
        raw = row['source_json']
        raw = json.loads(raw) if isinstance(raw, str) else raw
        value = raw.get('x_HB_EvidenceClass', 'unclassified')
        return value if value in ('official-public', 'official-public-subset', 'synthetic-contract-fixture') else 'unclassified'
    except (TypeError, ValueError, AttributeError):
        return 'unclassified'


class Projection:
    def __init__(self, params, deliver=None):
        self.params, self.expected = selection(params)
        self.deliver = deliver

    def preflight(self, journal, coordinator, context, scopes):
        if coordinator.a.is_view:
            self.company = str(context['audience']['company_id'])
            self.tenants = {t['properties']['key']: t['properties'] for t in context['tenants']}
        else:
            self.tenants = {context['properties']['key']: context['properties']}
            self.company = context['properties'].get('company_id')
        self.versions = sorted({d['version'] for _, d, _ in scopes})
        self.delivery_count = len({d['batch_id'] for _, d, _ in scopes})

    def consume(self, result, all_rows):
        if result.get('classification') != 'exploratory-query-only' or result.get('period_attribution') != 'charge-start-in-tenant-timezone-no-proration':
            raise Rejected('overview_basis')
        facts = []
        for window, batch, ordinal, tenant, row in all_rows:
            if tenant not in self.tenants:
                raise Rejected('scope_denied')
            month = row['charge_start'].astimezone(ZoneInfo(self.tenants[tenant]['timezone'])).strftime('%Y-%m')
            facts.append((month, account_id(tenant, row), tenant, window, batch, ordinal, row))
        providers = sorted({r[6]['provider'] for r in facts})
        periods = sorted({r[0] for r in facts})
        currencies = sorted({r[6]['currency'] for r in facts})
        if len(providers) > 128 or len(periods) > 120 or len(currencies) > 64:
            raise Rejected('selection_bound')
        p = self.params
        if p['provider'] and p['provider'] not in providers or p['account'] and p['account'] not in {r[1] for r in facts} or p['period'] != 'all' and p['period'] not in periods:
            raise Rejected('selection_unavailable')
        identity = {k: v for k, v in p.items() if k not in ('after', 'account_after')}
        snapshot = digest(canonical({'projection': 1, 'source': result['snapshot'], 'selection': identity}).encode())
        if self.expected is not None and self.expected != snapshot:
            raise Rejected('query_snapshot_changed')
        narrowed = [r for r in facts if (not p['provider'] or r[6]['provider'] == p['provider']) and (not p['account'] or r[1] == p['account'])]
        selected = [r for r in narrowed if p['period'] == 'all' or r[0] == p['period']]
        rows = [r[6] for r in selected]
        groups = {}
        for item in selected:
            groups.setdefault(item[1], []).append(item)
        accounts = []
        for key, values in groups.items():
            first = values[0]
            row = first[6]
            accounts.append({'id': key, 'tenant': first[2], 'provider': row['provider'], 'billing_account': row['account'],
                             'sub_account': row['subaccount'], 'native_ancestry': 'unverified', 'amounts': amounts([v[6] for v in values]), 'rows': len(values)})
        accounts.sort(key=lambda a: (a['provider'], a['billing_account'], a['sub_account'] or '', a['tenant']))
        search = p['q'].casefold()
        found = [a for a in accounts if not search or search in ' '.join(str(a[k] or '') for k in ['provider', 'billing_account', 'sub_account', 'tenant']).casefold()]
        a = p['account_after']
        after = p['after']
        if a > len(found) or after > len(selected):
            raise Rejected('query_page')
        selected.sort(key=lambda r: (r[3], r[4], r[5], r[2]))
        details = []
        for item in selected[after:after + PAGE]:
            month, key, tenant, window, batch, ordinal, row = item
            details.append({'id': digest(canonical([window, batch, ordinal, tenant]).encode()), 'account': key,
                            'provider': row['provider'], 'billing_account': row['account'], 'sub_account': row['subaccount'],
                            'service': row['service'], 'currency': row['currency'], 'charge_month': month,
                            'charge_start': row['charge_start'].isoformat(), 'charge_end': row['charge_end'].isoformat(),
                            **{basis: str(coefficient(row[basis])) if row[basis] is not None else None for basis in BASES},
                            'evidence_class': evidence(row), 'batch': str(batch), 'ordinal': str(ordinal)})
        windows = result.get('coverage', [])
        body = {'schema': 1, 'scale': 18, 'snapshot': snapshot, 'source_snapshot': result['snapshot'], 'selection': identity,
                'classification': 'collected-focus-sample-query-only', 'financial_authority': False,
                'observed_at': datetime.now(timezone.utc).isoformat(), 'period_basis': result['period_attribution'],
                'company': self.company, 'company_mapping': 'synthetic-sample-authorization',
                'tenants': [{'key': k, 'timezone': v['timezone']} for k, v in sorted(self.tenants.items())],
                'versions': self.versions, 'delivery_count': self.delivery_count,
                'coverage': {'slices': len(windows), 'finalized_slices': sum(w.get('finalized') is True for w in windows),
                             'complete_through': min((w['complete_through'] for w in windows), default=None), 'expectations': 'operator-asserted-unverified'},
                'provenance': dict(Counter(evidence(r) for r in rows)),
                'options': {'providers': providers, 'periods': periods, 'currencies': currencies},
                'authorized_rows': len(facts), 'row_count': len(rows), 'amounts': amounts(rows),
                'providers': [{'provider': provider, 'amounts': amounts([r for r in rows if r['provider'] == provider])} for provider in sorted({r['provider'] for r in rows})],
                'months': [{'period': month, 'amounts': amounts([r[6] for r in narrowed if r[0] == month])} for month in sorted({r[0] for r in narrowed})],
                'account_count': len(accounts), 'account_matches': len(found), 'accounts': found[a:a + PAGE],
                'next_account_after': a + PAGE if a + PAGE < len(found) else None,
                'rows': details, 'next_after': after + PAGE if after + PAGE < len(selected) else None,
                'after': after, 'account_after': a}
        return self.deliver(body) if self.deliver is not None else body


def overview(actor, params, deliver=None):
    return collected_query(actor, limit=1, _consumer=Projection(params, deliver))
