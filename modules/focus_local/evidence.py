import csv
import io
import re
from datetime import datetime, timezone

from .contract import MAX_BYTES, MAX_ROWS, Rejected, controls, digest, parse, rows_digest
from .source_data import FIELDS, SourceData, wire
from .storage import pinned_contract


def source_fields(raw, contract, normalized, ordinal):
    source = next(row for index, row in enumerate(csv.DictReader(io.StringIO(raw.decode('utf-8'), newline=''), strict=True), 1) if index == ordinal)
    return [{'name': f['name'], 'source_name': f['source'],
             'source_present': f['source'] is not None and f['source'] in source,
             'source_text': source.get(f['source']), 'normalized': wire(normalized[f['name']])}
            for f in contract.fields if f['name'] in FIELDS]


class Evidence(SourceData):
    def inspect_row(self, release_id, expected_manifest, ordinal):
        if not isinstance(expected_manifest, str) or not re.fullmatch('[0-9a-f]{64}', expected_manifest) or type(ordinal) is not int or not 1 <= ordinal <= MAX_ROWS:
            raise Rejected('request_invalid')
        a = self.authority
        with a.inspection() as cap:
            release = self.inspected_release(release_id, expected_manifest, cap)
            manifest = release['manifest']
            if ordinal > manifest['row_count']:
                raise Rejected('request_invalid')
            contract = pinned_contract(release)
            key = f'{a.scope[0]}/{a.scope[1]}/originals/{release["release_id"]}/{release["file_digest"]}.csv'
            if release['original_key'] != key or manifest['original_key'] != key:
                raise Rejected('original_reference')
            raw = self.store.bounded_original(key, MAX_BYTES)
            if len(raw) != manifest['file_bytes'] or digest(raw) != release['file_digest']:
                raise Rejected('original_digest')
            normalized = parse(raw, contract, {'account': cap['account'], 'subaccount': cap['subaccount'], 'start': cap['period_start'], 'end': cap['period_end']})
            if len(normalized) != manifest['row_count'] or rows_digest(normalized) != manifest['rows_digest'] or controls(normalized, contract.fields) != manifest['controls']:
                raise Rejected('original_manifest')
            return {'release_id': str(release['release_id']), 'snapshot': expected_manifest, 'ordinal': str(ordinal),
                'row_count': len(normalized), 'source_revision': str(release['source_revision']),
                'version': manifest['version'], 'registry_digest': release['registry_digest'],
                'original_digest': release['file_digest'], 'original_bytes': len(raw),
                'method': 'pinned-original-record-extract-v1', 'classification': 'local-source-evidence',
                'warehouse_integrity': 'not-revalidated', 'retention': 'unknown', 'public_document': 'unconnected',
                'observed_at': datetime.now(timezone.utc).isoformat(),
                'scope': {'workspace': str(a.scope[1]), 'account': cap['account'],
                          'period_start': cap['period_start'].isoformat(), 'period_end': cap['period_end'].isoformat()},
                'fields': source_fields(raw, contract, normalized[ordinal - 1], ordinal)}
