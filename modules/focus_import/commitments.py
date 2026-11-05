import re
from collections import Counter
from datetime import datetime, timezone
from zoneinfo import ZoneInfo

from .collection import query as collected_query
from .commitment_fields import FIELDS, PROFILE, STATES, STATUSES, Mapping, metadata
from .model import Rejected, canonical, coefficient, digest
from .overview import BASES, PAGE, Projection as OverviewProjection, account_id, evidence, selection as overview_selection


KEYS = {'provider', 'period', 'account', 'q', 'basis', 'after', 'account_after', 'snapshot', 'group', 'row_after'}


def selection(params):
    if not isinstance(params, dict) or set(params) - KEYS:
        raise Rejected('request_invalid')
    selected, expected = overview_selection({k: v for k, v in params.items() if k not in ('group', 'row_after')})
    selected.pop('basis')
    group = params.get('group', '')
    row_after = params.get('row_after', '0')
    if not isinstance(group, str) or group and not re.fullmatch('[a-f0-9]{64}', group):
        raise Rejected('request_invalid')
    if not isinstance(row_after, str) or not re.fullmatch('[0-9]{1,5}', row_after) or int(row_after) > 10000:
        raise Rejected('request_invalid')
    if int(row_after) and not group:
        raise Rejected('request_invalid')
    if (group or int(row_after)) and expected is None:
        raise Rejected('snapshot_required')
    selected.update(group=group, row_after=int(row_after))
    return selected, expected


def costs(rows):
    result = {'rows': len(rows)}
    for basis in BASES:
        values = [int(r[basis]) for r in rows if r[basis] is not None]
        if sum(abs(v) for v in values) >= 10 ** 50:
            raise Rejected('commitment_projection_bound')
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
                raise Rejected('commitment_delivery_mapping')
            tenant = members.get(str(item.get('tenant_id')))
            batch = item.get('batch_id')
            if tenant is None or not isinstance(batch, str) or batch != delivery.get('batch_id'):
                raise Rejected('commitment_delivery_scope')
            key = (tenant, batch)
            if key in self.batch_versions and self.batch_versions[key] != version:
                raise Rejected('commitment_delivery_conflict')
            self.batch_versions[key] = version

    def charge(self, item):
        window, batch, ordinal, tenant, row = item
        if tenant not in self.tenants:
            raise Rejected('scope_denied')
        version = self.batch_versions.get((tenant, batch))
        if version is None:
            raise Rejected('commitment_delivery_missing')
        fields = self.mapping.fields(row, version)
        money = {}
        for basis in BASES:
            value = coefficient(row[basis]) if row[basis] is not None else None
            if value is not None and abs(value) >= 10 ** 38:
                raise Rejected('commitment_input_bound')
            money[basis] = str(value) if value is not None else None
        return {'id': digest(canonical([window, batch, ordinal, tenant]).encode()),
                'group': '', 'tenant': tenant, 'account': account_id(tenant, row),
                'provider': row['provider'], 'billing_account': row['account'], 'sub_account': row['subaccount'],
                'currency': row['currency'], 'service': row['service'], **money,
                **{name: row[name].isoformat() for name in ('charge_start', 'charge_end', 'billing_start', 'billing_end')},
                'version': version, 'batch': str(batch), 'ordinal': str(ordinal), 'fields': fields,
                'evidence_class': evidence(row)}

    def consume(self, result, all_rows):
        if result.get('classification') != 'exploratory-query-only' or result.get('period_attribution') != 'charge-start-in-tenant-timezone-no-proration':
            raise Rejected('overview_basis')
        facts = []
        for item in all_rows:
            charge = self.charge(item)
            month = item[4]['charge_start'].astimezone(ZoneInfo(self.tenants[item[3]]['timezone'])).strftime('%Y-%m')
            facts.append((month, charge))
        providers = sorted({r['provider'] for _, r in facts})
        periods = sorted({month for month, _ in facts})
        currencies = sorted({r['currency'] for _, r in facts})
        if len(providers) > 128 or len(periods) > 120 or len(currencies) > 64 or len(facts) > 10000:
            raise Rejected('selection_bound')
        p = self.params
        if p['provider'] and p['provider'] not in providers or p['account'] and p['account'] not in {r['account'] for _, r in facts} or p['period'] != 'all' and p['period'] not in periods:
            raise Rejected('selection_unavailable')
        identity = {k: p[k] for k in ('period', 'provider', 'account', 'q')}
        snapshot = digest(canonical({'profile': PROFILE, 'mapping': self.mapping.digest,
                                     'source': result['snapshot'], 'company': self.company,
                                     'selection': identity}).encode())
        if self.expected is not None and self.expected != snapshot:
            raise Rejected('query_snapshot_changed')
        rows = [r for month, r in facts if (not p['provider'] or r['provider'] == p['provider']) and (not p['account'] or r['account'] == p['account']) and (p['period'] == 'all' or month == p['period'])]
        rows.sort(key=lambda r: (r['tenant'], r['batch'], int(r['ordinal']), r['id']))
        grouped, unidentified = {}, []
        for row in rows:
            source_id = row['fields']['id']
            if source_id['state'] != 'value':
                unidentified.append(row)
                continue
            key = digest(canonical([self.company, row['tenant'], row['provider'], row['account'], row['currency'], source_id['value']]).encode())
            row['group'] = key
            grouped.setdefault(key, []).append(row)
        groups = []
        for key, members in grouped.items():
            first = members[0]
            groups.append({'id': key, 'source_id': first['fields']['id']['value'],
                           **{k: first[k] for k in ('tenant', 'account', 'provider', 'billing_account', 'sub_account', 'currency')},
                           **costs(members), 'metadata': {name: metadata(members, name) for name in FIELDS if name != 'id'},
                           'statuses': {status: costs([r for r in members if STATUSES.get(r['fields']['status']['value'] if r['fields']['status']['state'] == 'value' and r['version'] == '1.0' else None, 'unknown') == status]) for status in ('used', 'unused', 'unknown')},
                           'versions': sorted({r['version'] for r in members}),
                           'mapping': {'version_mapped_rows': sum(r['version'] == '1.0' for r in members), 'version_unmapped_rows': sum(r['version'] != '1.0' for r in members)},
                           'intervals': {name: (min if name.endswith('start') else max)(r[name] for r in members) for name in ('charge_start', 'charge_end', 'billing_start', 'billing_end')}})
        groups.sort(key=lambda r: (r['provider'], r['tenant'], r['billing_account'], r['sub_account'] or '', r['currency'], r['source_id'], r['id']))
        after, account_after, row_after = p['after'], p['account_after'], p['row_after']
        page = groups[after:after + PAGE]
        if after > len(groups):
            raise Rejected('query_page')
        detail = None
        if p['group']:
            if p['group'] not in {g['id'] for g in page}:
                raise Rejected('selection_unavailable')
            members = grouped[p['group']]
            if row_after > len(members):
                raise Rejected('query_page')
            detail = {'group': p['group'], 'rows': members[row_after:row_after + PAGE],
                      'after': row_after, 'next_after': row_after + PAGE if row_after + PAGE < len(members) else None,
                      'total_rows': len(members)}
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
        identified = [r for r in rows if r['group']]
        reasons = Counter(r['fields']['id']['state'] for r in unidentified)
        windows = result.get('coverage', [])
        body = {'schema': 1, 'scale': 18, 'profile': PROFILE, 'classification': 'collected-commitment-evidence-only',
                'financial_authority': False, 'snapshot': snapshot, 'source_snapshot': result['snapshot'],
                'mapping_digest': self.mapping.digest, 'company': self.company, 'versions': self.versions,
                'observed_at': datetime.now(timezone.utc).isoformat(), 'selection': identity,
                'period_basis': result['period_attribution'],
                'coverage': {'slices': len(windows), 'finalized_slices': sum(w.get('finalized') is True for w in windows),
                             'complete_through': min((w['complete_through'] for w in windows), default=None),
                             'expectations': 'operator-asserted-unverified'},
                'options': {'providers': providers, 'periods': periods, 'currencies': currencies},
                'authorized_rows': len(facts), 'row_count': len(rows), 'identified_rows': len(identified),
                'unidentified_rows': len(unidentified), 'group_count': len(groups),
                'identified_amounts': amounts(identified), 'unidentified_amounts': amounts(unidentified),
                'unidentified_reasons': {state: reasons[state] for state in STATES},
                'groups': page, 'after': after, 'next_after': after + PAGE if after + PAGE < len(groups) else None,
                'accounts': found[account_after:account_after + PAGE], 'account_count': len(accounts), 'account_matches': len(found),
                'account_after': account_after, 'next_account_after': account_after + PAGE if account_after + PAGE < len(found) else None,
                'selected': detail}
        return self.deliver(body) if self.deliver is not None else body


def commitments(actor, params, deliver=None):
    return collected_query(actor, limit=1, _consumer=Projection(params, deliver))
