import csv
import io
from uuid import UUID

from .contract import MAX_BYTES, Rejected
from .pipeline import Coordinator


class Intake(Coordinator):
    def context(self):
        a = self.authority
        with a.transaction('ingest', allow_unfinished=True) as cap:
            return {'profile': 'synthetic-local-v1', 'workspace': str(a.scope[1]),
                'connection': str(a.scope[2]), 'account': cap['account'],
                'period_start': cap['period_start'].isoformat(),
                'period_end': cap['period_end'].isoformat(),
                'versions': sorted(self.registry()), 'max_bytes': MAX_BYTES}

    def receive(self, data, version, revision):
        contract = self.registry().get(version)
        if not contract:
            raise Rejected('unsupported_version')
        try:
            headers = next(csv.reader(io.StringIO(data.decode('utf-8')), strict=True))
        except (UnicodeError, csv.Error, StopIteration):
            raise Rejected('source_admission') from None
        allowed = {field['source'] for field in contract.fields}
        if not headers or not set(headers) <= allowed:
            raise Rejected('source_fields')
        return self.ingest(data, version, revision)

    def receipts(self, release_id=None):
        a = self.authority
        with a.transaction('ingest', allow_unfinished=True):
            if release_id is not None:
                try:
                    release_id = UUID(release_id)
                except ValueError:
                    raise Rejected('release_unavailable') from None
                release = a.release(release_id)
                rows = [release]
            else:
                rows = a.conn.execute('SELECT * FROM hb_local.release WHERE connection_id=%s ORDER BY created_at DESC,release_id DESC LIMIT 26', (a.scope[2],)).fetchall()
            releases = []
            for row in rows[:25]:
                operations = a.conn.execute('SELECT state FROM hb_local.operation WHERE connection_id=%s AND release_id=%s AND kind=%s', (a.scope[2], row['release_id'], 'load')).fetchall()
                state = operations[0]['state'] if len(operations) == 1 else 'unknown'
                releases.append({**self.result(row), 'source_revision': str(row['source_revision']),
                    'file_digest': row['file_digest'], 'operation_state': state,
                    'created_at': row['created_at'].isoformat()})
            quarantines = []
            if release_id is None:
                quarantines = a.conn.execute('SELECT q.receipt_id,q.source_revision,q.declared_version,q.reason,o.state AS operation_state FROM hb_local.quarantine q LEFT JOIN hb_local.operation o ON o.quarantine_id=q.receipt_id AND o.connection_id=q.connection_id WHERE q.connection_id=%s ORDER BY q.source_revision DESC,q.receipt_id DESC LIMIT 26', (a.scope[2],)).fetchall()
            return {'releases': releases, 'more_releases': len(rows) > 25,
                'quarantine': [{'receipt_id': str(r['receipt_id']), 'source_revision': str(r['source_revision']),
                    'version': r['declared_version'], 'reason': r['reason'],
                    'operation_state': r['operation_state'] or 'unknown'} for r in quarantines[:25]],
                'more_quarantine': len(quarantines) > 25}
