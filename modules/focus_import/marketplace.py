import re
from datetime import datetime, timezone
from zoneinfo import ZoneInfo

from .collection import query as collected_query
from .marketplace_fields import PROFILE, Mapping, classify
from .model import Rejected, canonical, coefficient, digest
from .overview import BASES, PAGE, Projection as OverviewProjection, account_id, evidence, selection as overview_selection


KEYS = {'provider', 'period', 'account', 'q', 'basis', 'after', 'account_after', 'snapshot', 'row'}


def selection(params):
    if not isinstance(params, dict) or set(params) - KEYS:
        raise Rejected('request_invalid')
    selected, expected = overview_selection({k: v for k, v in params.items() if k != 'row'})
    selected.pop('basis')
    row = params.get('row', '')
    if not isinstance(row, str) or row and not re.fullmatch('[a-f0-9]{64}', row):
        raise Rejected('request_invalid')
    if row and expected is None:
        raise Rejected('snapshot_required')
    selected['row'] = row
    return selected, expected


def costs(rows):
    result = {'rows': len(rows)}
    for basis in BASES:
        values = [int(r[basis]) for r in rows if r[basis] is not None]
        if sum(abs(v) for v in values) >= 10 ** 50:
            raise Rejected('marketplace_projection_bound')
        result[basis] = {'coefficient': str(sum(values)) if values else None,
                         'present': len(values), 'missing': len(rows) - len(values)}
    return result


def amounts(rows):
    return [{'currency': currency, **costs([r for r in rows if r['currency'] == currency])}
            for currency in sorted({r['currency'] for r in rows})]


class Projection(OverviewProjection):
    def __init__(self, params, deliver=None):
        self.params, self.expected = selection(params)
        self.deliver = deliver
        self.mapping = Mapping()

    def preflight(self, journal, coordinator, context, scopes):
        super().preflight(journal, coordinator, context, scopes)
        members = {str(t['tenant_id']): t['properties']['key'] for t in context['tenants']} if coordinator.a.is_view else {str(coordinator.a.tenant): context['properties']['key']}
        self.batch_versions = {}
        for _, delivery, item in scopes:
            version = delivery.get('version')
            if version not in self.mapping.registry or delivery.get('mapping_digest') != digest(canonical(self.mapping.registry[version]).encode()):
                raise Rejected('marketplace_delivery_mapping')
            tenant = members.get(str(item.get('tenant_id')))
            batch = item.get('batch_id')
            if tenant is None or not isinstance(batch, str) or batch != delivery.get('batch_id'):
                raise Rejected('marketplace_delivery_scope')
            key = (tenant, batch)
            if key in self.batch_versions and self.batch_versions[key] != version:
                raise Rejected('marketplace_delivery_conflict')
            self.batch_versions[key] = version

    def charge(self, item):
        window, batch, ordinal, tenant, row = item
        if tenant not in self.tenants:
            raise Rejected('scope_denied')
        version = self.batch_versions.get((tenant, batch))
        if version is None:
            raise Rejected('marketplace_delivery_missing')
        money = {}
        for basis in BASES:
            value = coefficient(row[basis]) if row[basis] is not None else None
            if value is not None and abs(value) >= 10 ** 38:
                raise Rejected('marketplace_input_bound')
            money[basis] = str(value) if value is not None else None
        return {'id': digest(canonical([window, batch, ordinal, tenant]).encode()),
                'tenant': tenant, 'account': account_id(tenant, row),
                'provider': row['provider'], 'billing_account': row['account'], 'sub_account': row['subaccount'],
                'currency': row['currency'], 'service': row['service'], **money,
                **{name: row[name].isoformat() for name in ('charge_start', 'charge_end', 'billing_start', 'billing_end')},
                'version': version, 'batch': str(batch), 'ordinal': str(ordinal),
                'fields': self.mapping.fields(row, version), 'evidence_class': evidence(row)}

    def consume(self, result, all_rows):
        if result.get('classification') != 'exploratory-query-only' or result.get('period_attribution') != 'charge-start-in-tenant-timezone-no-proration':
            raise Rejected('overview_basis')
        facts = []
        for item in all_rows:
            row = self.charge(item)
            month = item[4]['charge_start'].astimezone(ZoneInfo(self.tenants[item[3]]['timezone'])).strftime('%Y-%m')
            facts.append((month, row))
        providers = sorted({r['provider'] for _, r in facts})
        periods = sorted({month for month, _ in facts})
        currencies = sorted({r['currency'] for _, r in facts})
        if len(providers) > 128 or len(periods) > 120 or len(currencies) > 64 or len(facts) > 10000:
            raise Rejected('selection_bound')
        if len({r['id'] for _, r in facts}) != len(facts):
            raise Rejected('marketplace_row_identity')
        p = self.params
        if p['provider'] and p['provider'] not in providers or p['account'] and p['account'] not in {r['account'] for _, r in facts} or p['period'] != 'all' and p['period'] not in periods:
            raise Rejected('selection_unavailable')
        identity = {k: p[k] for k in ('period', 'provider', 'account', 'q')}
        snapshot = digest(canonical({'profile': PROFILE, 'mapping': self.mapping.digest,
                                     'source': result['snapshot'], 'company': self.company,
                                     'delivery_versions': sorted((tenant, batch, version) for (tenant, batch), version in self.batch_versions.items()),
                                     'selection': identity}).encode())
        if self.expected is not None and self.expected != snapshot:
            raise Rejected('query_snapshot_changed')
        rows = [r for month, r in facts if (not p['provider'] or r['provider'] == p['provider']) and (not p['account'] or r['account'] == p['account']) and (p['period'] == 'all' or month == p['period'])]
        rows.sort(key=lambda r: (r['tenant'], r['batch'], int(r['ordinal']), r['id']))
        classified = {name: [r for r in rows if classify(r) == name] for name in ('candidate', 'no_difference', 'unknown')}
        candidates = classified['candidate']
        after, account_after = p['after'], p['account_after']
        if after > len(candidates):
            raise Rejected('query_page')
        page = candidates[after:after + PAGE]
        selected = None
        if p['row']:
            selected = next((r for r in page if r['id'] == p['row']), None)
            if selected is None:
                raise Rejected('selection_unavailable')
        account_rows = {}
        for row in rows:
            account_rows.setdefault(row['account'], []).append(row)
        accounts = [{**{k: members[0][k] for k in ('tenant', 'provider', 'billing_account', 'sub_account')},
                     'id': key, 'rows': len(members)} for key, members in account_rows.items()]
        accounts.sort(key=lambda r: (r['provider'], r['billing_account'], r['sub_account'] or '', r['tenant']))
        search = p['q'].casefold()
        found = [a for a in accounts if not search or search in ' '.join(str(a[k] or '') for k in ('provider', 'billing_account', 'sub_account', 'tenant')).casefold()]
        if account_after > len(found):
            raise Rejected('query_page')
        windows = result.get('coverage', [])
        body = {'schema': 1, 'scale': 18, 'profile': PROFILE, 'classification': 'collected-marketplace-evidence-only',
                'financial_authority': False, 'snapshot': snapshot, 'source_snapshot': result['snapshot'],
                'mapping_digest': self.mapping.digest, 'company': self.company, 'versions': self.versions,
                'observed_at': datetime.now(timezone.utc).isoformat(), 'selection': identity,
                'period_basis': result['period_attribution'],
                'coverage': {'slices': len(windows), 'finalized_slices': sum(w.get('finalized') is True for w in windows),
                             'complete_through': min((w['complete_through'] for w in windows), default=None),
                             'expectations': 'operator-asserted-unverified'},
                'options': {'providers': providers, 'periods': periods, 'currencies': currencies},
                'authorized_rows': len(facts), 'row_count': len(rows),
                'counts': {name: len(members) for name, members in classified.items()},
                'candidate_amounts': amounts(candidates), 'unknown_amounts': amounts(classified['unknown']),
                'rows': page, 'after': after, 'next_after': after + PAGE if after + PAGE < len(candidates) else None,
                'accounts': found[account_after:account_after + PAGE], 'account_count': len(accounts), 'account_matches': len(found),
                'account_after': account_after, 'next_account_after': account_after + PAGE if account_after + PAGE < len(found) else None,
                'selected': selected}
        return self.deliver(body) if self.deliver is not None else body


def marketplace(actor, params, deliver=None):
    return collected_query(actor, limit=1, _consumer=Projection(params, deliver))
