import json
import re
from collections import Counter
from datetime import datetime, timezone

from . import collection, receipt_projection
from .model import Rejected, canonical, digest, versions
from .overview import evidence
from .temporal import checked_window


PROFILE = 'retained-focus-collection-status-v1'
MAX_RESPONSE_BYTES = 2 * 1024 * 1024
PROVENANCE = ('official-public', 'official-public-subset', 'synthetic-contract-fixture', 'unclassified')


def selection(params):
    if not isinstance(params, dict) or params:
        raise Rejected('request_invalid')


def unavailable():
    raise Rejected('collection_status_unavailable')


class Projection:
    def __init__(self, actor, deliver=None):
        self.actor, self.deliver = actor, deliver

    def preflight(self, journal, coordinator, context, scopes):
        self.observed_at = datetime.now(timezone.utc).isoformat()
        receipts = receipt_projection.project(self.actor, {}, coordinator.a, context, journal.state, self.observed_at)
        self.scope = {key: receipts[key] for key in ('audience', 'member_count', 'auth_scope', 'permission_revision')}
        if coordinator.a.is_view:
            self.members = {str(t['tenant_id']): t['properties']['key'] for t in context['tenants']}
        else:
            self.members = {str(coordinator.a.tenant): context['properties']['key']}
        self.deliveries = {}
        for window, delivery, item in scopes:
            tenant = self.members.get(str(item.get('tenant_id')))
            batch = item.get('batch_id')
            if tenant is not None and isinstance(window, str) and isinstance(batch, str):
                self.deliveries.setdefault((window, batch, tenant), []).append(delivery)

    def consume(self, result, all_rows):
        if result.get('classification') != 'exploratory-query-only' or type(result.get('row_count')) is not int or result['row_count'] != len(all_rows):
            unavailable()
        if len(all_rows) > 10000:
            raise Rejected('selection_bound')
        registry = versions()
        providers, periods, provenance = Counter(), Counter(), Counter()
        batches, seen = {}, set()
        for window, batch, ordinal, tenant, row in all_rows:
            declarations = self.deliveries.get((window, batch, tenant), [])
            if len(declarations) != 1 or self.members.get(str(row.get('tenant_id'))) != tenant:
                raise Rejected('permission')
            identity = (window, batch, ordinal, tenant)
            if identity in seen or type(ordinal) is not int or ordinal < 1:
                unavailable()
            seen.add(identity)
            delivery = declarations[0]
            version = delivery.get('version')
            if delivery.get('batch_id') != batch or not isinstance(version, str) or not re.fullmatch(r'\d{1,2}\.\d{1,2}', version) or version not in registry:
                unavailable()
            try:
                source_window = checked_window(delivery['descriptor']['window'])
            except (Rejected, KeyError, TypeError, ValueError, OverflowError):
                unavailable()
            period = (source_window['start'], source_window['end'])
            declaration = (version, period)
            if batch in batches and batches[batch] != declaration:
                unavailable()
            batches[batch] = declaration
            provider = row.get('provider')
            if not isinstance(provider, str) or not 1 <= len(provider) <= 256:
                unavailable()
            providers[provider] += 1
            periods[period] += 1
            provenance[evidence(row)] += 1
        if len(providers) > 128 or len(periods) > 120:
            raise Rejected('selection_bound')
        source_versions = sorted({version for version, _ in batches.values()})
        if len(source_versions) > 128:
            raise Rejected('selection_bound')
        body = {'schema': 1, 'profile': PROFILE, 'read_only': True, 'financial_authority': False,
                **self.scope, 'observed_at': self.observed_at, 'row_count': len(all_rows), 'delivery_count': len(batches),
                'versions': source_versions, 'provenance': {key: provenance[key] for key in PROVENANCE},
                'providers': [{'provider': key, 'row_count': providers[key]} for key in sorted(providers)],
                'periods': [{'window_start': start, 'window_end': end, 'row_count': count}
                            for (start, end), count in sorted(periods.items())],
                'upstream_completeness': 'unverified', 'source_classification': 'exploratory-query-only',
                'original_availability': 'not-checked'}
        body['snapshot'] = digest(canonical({key: value for key, value in body.items() if key != 'observed_at'}).encode())
        if len(json.dumps(body).encode('utf-8')) > MAX_RESPONSE_BYTES:
            raise Rejected('response_bound')
        return self.deliver(body) if self.deliver is not None else body


def collection_status(actor, params, deliver=None):
    selection(params)
    return collection.query(actor, limit=1, _consumer=Projection(actor, deliver))
