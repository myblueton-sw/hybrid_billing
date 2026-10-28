import hashlib
import json
import re
import select
import sys
from uuid import UUID

from . import collection
from .model import Rejected, canonical, coefficient, fixed, instant


FIELDS = ['service', 'quantity', 'unit', 'usage_start', 'usage_end', 'measurement_state']
PROFILE_KEYS = {'kind', 'installation', 'source', 'actor', 'company_id', 'tenant_id', 'snapshot', 'start', 'end', 'grain', 'services', 'units'}
MAX_INPUT = 65536
MAX_OUTPUT = 1048576


def checked_profile(raw):
    if not isinstance(raw, str) or len(raw.encode()) > MAX_INPUT:
        raise Rejected('usage_profile')
    profile = json.loads(raw)
    if not isinstance(profile, dict) or set(profile) != PROFILE_KEYS or profile['kind'] != 'source-usage-v1':
        raise Rejected('usage_profile')
    for key in ['installation', 'tenant_id']:
        if str(UUID(profile[key])) != profile[key]:
            raise Rejected('usage_profile')
    for key in ['company_id', 'source', 'actor']:
        if not isinstance(profile[key], str) or not 1 <= len(profile[key]) <= 256:
            raise Rejected('usage_profile')
    if not re.fullmatch('[a-f0-9]{64}', profile['snapshot']) or profile['grain'] not in ['day', 'month']:
        raise Rejected('usage_profile')
    if not all(isinstance(profile[k], str) and profile[k].endswith('Z') for k in ['start', 'end']) or instant(profile['start']) >= instant(profile['end']):
        raise Rejected('usage_profile')
    for key in ['services', 'units']:
        values = profile[key]
        if not isinstance(values, dict) or not 1 <= len(values) <= 100:
            raise Rejected('usage_profile')
        if any(not isinstance(k, str) or not 1 <= len(k) <= 4096 or not isinstance(v, str) or not 1 <= len(v) <= 128 or any(ord(c) < 32 for c in v) for k, v in values.items()):
            raise Rejected('usage_profile')
    return profile, hashlib.sha256(raw.encode()).hexdigest()


class Projection:
    def __init__(self, raw, deliver):
        self.profile, self.revision = checked_profile(raw)
        self.deliver = deliver

    def preflight(self, journal, coordinator, context, slices):
        p = self.profile
        a = coordinator.a
        if not journal.head or journal.head['installation'] != p['installation'] or a.is_view or str(a.tenant) != p['tenant_id'] or a.actor != p['actor'] or a.bindings['source'] != p['source']:
            raise Rejected('usage_binding')
        if context['properties'].get('company_id') != p['company_id'] or not context['active']:
            raise Rejected('usage_binding')
        if any(item['state'] != 'hot' or item['tenant_id'] != p['tenant_id'] for _, _, item in slices):
            raise Rejected('usage_source_state')

    def row(self, batch, ordinal, row):
        p = self.profile
        if str(row['tenant_id']) != p['tenant_id'] or type(ordinal) is not int or ordinal <= 0:
            raise Rejected('usage_scope')
        start, end = row['charge_start'], row['charge_end']
        if not instant(p['start']) <= start < end <= instant(p['end']):
            raise Rejected('usage_interval')
        if row['service'] not in p['services'] or row['consumed_unit'] is not None and row['consumed_unit'] not in p['units']:
            raise Rejected('usage_unmapped')
        quantity = row['consumed_quantity']
        reference = hashlib.sha256(canonical([p['installation'], p['tenant_id'], str(batch), ordinal]).encode()).hexdigest()
        return {'Scope': {'CompanyID': p['company_id'], 'TenantID': p['tenant_id']}, 'ID': reference,
            'Values': {'service': p['services'][row['service']], 'quantity': '' if quantity is None else fixed(coefficient(quantity)),
                'unit': '' if row['consumed_unit'] is None else p['units'][row['consumed_unit']],
                'usage_start': start.isoformat(), 'usage_end': end.isoformat(),
                'measurement_state': 'missing' if quantity is None else 'source-reported'}}

    def consume(self, result, all_rows):
        if result['snapshot'] != self.profile['snapshot'] or result['classification'] != 'exploratory-query-only':
            raise Rejected('usage_snapshot')
        projected = {}
        for _, batch, ordinal, _, row in all_rows:
            key = (batch, ordinal)
            if key in projected:
                raise Rejected('usage_duplicate')
            projected[key] = self.row(batch, ordinal, row)
        rows = [projected[(row['batch_id'], row['ordinal'])] for row in result['rows']]
        return self.deliver({'status': 'ready', 'revision': self.revision, 'rows': rows})


def query(raw, limit, deliver):
    if type(limit) is not int or not 1 <= limit <= 100:
        raise Rejected('usage_limit')
    consumer = Projection(raw, deliver)
    p = consumer.profile
    return collection.query(p['actor'], p['start'], p['end'], p['grain'], limit,
        expected_snapshot=p['snapshot'], _consumer=consumer)


def main():
    try:
        line = sys.stdin.buffer.readline(MAX_INPUT + 1)
        if not line.endswith(b'\n') or len(line) > MAX_INPUT:
            raise Rejected('usage_input')
        request = json.loads(line)
        if not isinstance(request, dict) or set(request) != {'profile', 'limit'}:
            raise Rejected('usage_input')

        def deliver(value):
            data = json.dumps(value, separators=(',', ':'), ensure_ascii=True).encode() + b'\n'
            if len(data) > MAX_OUTPUT:
                raise Rejected('usage_output')
            sys.stdout.buffer.write(data)
            sys.stdout.buffer.flush()
            if not select.select([sys.stdin.buffer], [], [], 30)[0] or sys.stdin.buffer.readline(16) != b'release\n':
                raise Rejected('usage_release')
            sys.stdout.buffer.write(b'{"status":"released"}\n')
            sys.stdout.buffer.flush()

        query(request['profile'], request['limit'], deliver)
        return 0
    except Exception:
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
