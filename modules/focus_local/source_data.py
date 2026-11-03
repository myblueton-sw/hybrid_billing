import re
from datetime import datetime, timezone
from decimal import Decimal

from .contract import MAX_ROWS, Rejected, coefficient, fixed
from .reconciliation import Reconciliation
from .storage import client, pinned_contract
from .database import persona


FIELDS = ('billing_account_id', 'sub_account_id', 'service_name', 'resource_id',
          'charge_period_start', 'charge_period_end', 'consumed_quantity', 'consumed_unit',
          'pricing_quantity', 'pricing_unit', 'billed_cost', 'effective_cost',
          'list_cost', 'contracted_cost', 'billing_currency', 'charge_category')
PAGE_SIZE = 50


def wire(value):
    if isinstance(value, Decimal):
        return fixed(coefficient(value))
    if isinstance(value, datetime):
        return value.replace(tzinfo=value.tzinfo or timezone.utc).astimezone(timezone.utc).isoformat()
    return value


def warehouse_page(env, scope, release, after):
    contract = pinned_contract(release)
    names = [f['name'] for f in contract.fields if f['name'] in FIELDS]
    tenant = 'a' if scope[1] == persona('reader-a')[1] else 'b'
    ch = client(env, 'hb_local_reader_' + tenant)
    clause = 'installation_id={i:UUID} AND workspace_id={w:UUID} AND release_id={r:UUID} AND attempt=1'
    parameters = {'i': str(scope[0]), 'w': str(scope[1]), 'r': str(release['release_id']), 'after': after, 'limit': PAGE_SIZE + 1}
    try:
        stats = ch.query(f'SELECT count(),uniqExact(ordinal),min(ordinal),max(ordinal) FROM hb_local.{contract.table} WHERE {clause}', parameters=parameters).result_rows[0]
        count = release['manifest']['row_count']
        if type(count) is not int or not 0 < count <= MAX_ROWS or tuple(stats) != (count, count, 1, count):
            raise Rejected('storage_shape')
        columns = ','.join('`' + name + '`' for name in names)
        rows = ch.query(f'SELECT ordinal,{columns} FROM hb_local.{contract.table} WHERE {clause} AND ordinal>{{after:UInt64}} ORDER BY ordinal LIMIT {{limit:UInt32}}', parameters=parameters).result_rows
        ordinals = [row[0] for row in rows]
        if ordinals != list(range(after + 1, min(count, after + PAGE_SIZE + 1) + 1)):
            raise Rejected('storage_shape')
        values = [{'ordinal': str(row[0]), 'values': {name: wire(value) for name, value in zip(names, row[1:])}} for row in rows[:PAGE_SIZE]]
        return values, str(rows[PAGE_SIZE - 1][0]) if len(rows) > PAGE_SIZE else None
    finally:
        ch.close()


class SourceData(Reconciliation):
    def context(self):
        result = super().context()
        return {**result, 'profile': 'synthetic-local-source-data-v1', 'classification': 'local-source-inspection'}

    def page(self, release_id, expected_manifest, after):
        if not isinstance(expected_manifest, str) or not re.fullmatch('[0-9a-f]{64}', expected_manifest) or type(after) is not int or not 0 <= after < MAX_ROWS:
            raise Rejected('request_invalid')
        a = self.authority
        with a.inspection() as cap:
            release = self.inspected_release(release_id, expected_manifest, cap)
            manifest = release['manifest']
            if after >= manifest['row_count']:
                raise Rejected('request_invalid')
            rows, next_after = warehouse_page(a.env, a.scope, release, after)
            return {'candidate': self.metadata(release, cap), 'snapshot': expected_manifest,
                'after': str(after), 'next_after': next_after, 'rows': rows,
                'fields': [f['name'] for f in pinned_contract(release).fields if f['name'] in FIELDS],
                'classification': 'local-source-inspection', 'content_integrity': 'not-revalidated',
                'observed_at': datetime.now(timezone.utc).isoformat()}

    def inspected_release(self, release_id, expected_manifest, cap):
        a = self.authority
        release = a.release(release_id)
        self.check_registry(release)
        manifest = release['manifest']
        if release['manifest_digest'] != expected_manifest:
            raise Rejected('evidence_changed')
        for key, value in zip(('installation_id', 'workspace_id', 'connection_id'), a.scope):
            if manifest[key] != str(value):
                raise Rejected('manifest_scope')
        if manifest['policy_revision'] != cap['policy_revision']:
            raise Rejected('policy_changed')
        if release['state'] not in ('verified', 'sealed'):
            raise Rejected('input_unavailable')
        if manifest['file_digest'] != release['file_digest'] or manifest['source_revision'] != release['source_revision'] or manifest['attempt'] != 1:
            raise Rejected('manifest_identity')
        return release
