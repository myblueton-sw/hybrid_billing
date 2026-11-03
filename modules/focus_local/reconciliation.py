import json
from datetime import datetime, timezone

from .contract import MAX_BYTES, Rejected, canonical, controls, digest, fixed, parse, rows_digest
from .pipeline import Coordinator
from .storage import pinned_contract


LIMIT = 100
BASE_BLOCKERS = ('independent_cross_validation_missing', 'canonical_confirm_publish_unavailable')


def comparisons(expected, actual):
    output = []
    for key in sorted(set(expected) | set(actual)):
        currency, consumed, pricing = json.loads(key)
        left, right = expected.get(key, {}), actual.get(key, {})
        for name in sorted(set(left.get('measures', {})) | set(right.get('measures', {}))):
            e = left.get('measures', {}).get(name)
            a = right.get('measures', {}).get(name)
            ev, av = e.get('coefficient') if e else None, a.get('coefficient') if a else None
            complete = e is not None and a is not None and not e['missing'] and not a['missing']
            output.append({'currency': currency, 'consumed_unit': consumed, 'pricing_unit': pricing,
                'measure': name, 'expected': fixed(ev) if ev is not None else None,
                'actual': fixed(av) if av is not None else None,
                'difference': fixed(av - ev) if complete and ev is not None and av is not None else None,
                'expected_present': e['present'] if e else 0, 'actual_present': a['present'] if a else 0,
                'expected_missing': e['missing'] if e else None, 'actual_missing': a['missing'] if a else None,
                'state': 'match' if complete and e == a and left['rows'] == right['rows'] else 'incomplete' if not complete else 'difference'})
    return output


class Reconciliation(Coordinator):
    def context(self):
        with self.authority.inspection() as cap:
            return {'profile': 'synthetic-local-reconciliation-v1', 'workspace': str(self.authority.scope[1]),
                'connection': str(self.authority.scope[2]), 'account': cap['account'],
                'period_start': cap['period_start'].isoformat(), 'period_end': cap['period_end'].isoformat(),
                'classification': 'source-storage-consistency-only', 'publication_available': False}

    def blockers(self, cap, release=None):
        a = self.authority
        result = list(BASE_BLOCKERS)
        if a.conn.execute('SELECT 1 FROM hb_local.quarantine WHERE connection_id=%s LIMIT 1', (a.scope[2],)).fetchone():
            result.append('quarantine_hold')
        if a.conn.execute("SELECT 1 FROM hb_local.operation WHERE connection_id=%s AND state<>'terminal' LIMIT 1", (a.scope[2],)).fetchone():
            result.append('operation_unresolved')
        if release:
            if release['manifest']['policy_revision'] != cap['policy_revision']:
                result.append('policy_changed')
            if release['state'] not in ('verified', 'sealed'):
                result.append('candidate_unverified')
        return result

    def metadata(self, row, cap):
        return {**self.result(row), 'source_revision': str(row['source_revision']),
            'file_digest': row['file_digest'], 'created_at': row['created_at'].isoformat(),
            'blockers': self.blockers(cap, row)}

    def candidates(self, release_id=None):
        a = self.authority
        with a.inspection() as cap:
            rows = [a.release(release_id)] if release_id else a.conn.execute('SELECT * FROM hb_local.release WHERE connection_id=%s ORDER BY created_at DESC,release_id DESC LIMIT 26', (a.scope[2],)).fetchall()
            return {'candidates': [self.metadata(row, cap) for row in rows[:25]], 'more': len(rows) > 25,
                'blockers': self.blockers(cap)}

    def revalidate(self, release_id, expected_manifest):
        a = self.authority
        with a.inspection() as cap:
            release = a.release(release_id)
            if release['manifest_digest'] != expected_manifest:
                raise Rejected('evidence_changed')
            blockers = self.blockers(cap, release)
            result = {'candidate': self.metadata(release, cap), 'method': 'pinned-source-replay-and-storage-v1',
                'classification': 'source-storage-consistency-only', 'durable': False,
                'observed_at': datetime.now(timezone.utc).isoformat(), 'blockers': blockers,
                'checks': [], 'comparisons': [], 'comparison_count': 0, 'mismatch_count': 0,
                'incomplete_count': 0, 'truncated': False, 'state': 'unavailable'}
            try:
                self.check_registry(release)
                contract = pinned_contract(release)
                manifest = release['manifest']
                expected_key = f'{a.scope[0]}/{a.scope[1]}/originals/{release["release_id"]}/{release["file_digest"]}.csv'
                if release['original_key'] != expected_key or manifest['original_key'] != expected_key:
                    raise Rejected('original_reference')
                for key, value in zip(('installation_id', 'workspace_id', 'connection_id'), a.scope):
                    if manifest[key] != str(value):
                        raise Rejected('manifest_scope')
                raw = self.store.bounded_original(expected_key, MAX_BYTES)
                if len(raw) != manifest['file_bytes'] or digest(raw) != release['file_digest'] or manifest['file_digest'] != release['file_digest']:
                    raise Rejected('original_digest')
                source = parse(raw, contract, {'account': cap['account'], 'subaccount': cap['subaccount'], 'start': cap['period_start'], 'end': cap['period_end']})
                expected = controls(source, contract.fields)
                stored = self.store.rows(a.scope, release)
                actual = self.store.aggregates(a.scope, release)
                checks = {'original_hash': True, 'source_manifest_count': len(source) == manifest['row_count'],
                    'source_manifest_hash': rows_digest(source) == manifest['rows_digest'],
                    'source_manifest_controls': expected == manifest['controls'],
                    'storage_manifest_count': len(stored) == manifest['row_count'],
                    'storage_manifest_hash': rows_digest(stored) == manifest['rows_digest'],
                    'storage_aggregate_controls': controls(stored, contract.fields) == actual}
                items = comparisons(expected, actual)
                mismatch = sum(item['state'] == 'difference' for item in items)
                incomplete = sum(item['state'] == 'incomplete' for item in items)
                if not all(checks.values()) or mismatch:
                    blockers.append('consistency_difference')
                if incomplete:
                    blockers.append('measure_incomplete')
                result.update({'checks': [{'id': k, 'passed': v} for k, v in checks.items()],
                    'comparisons': items[:LIMIT], 'comparison_count': len(items), 'mismatch_count': mismatch,
                    'incomplete_count': incomplete, 'truncated': len(items) > LIMIT,
                    'state': 'match' if all(checks.values()) and not mismatch and not incomplete else 'difference',
                    'evidence': {'original_digest': digest(raw), 'manifest_digest': release['manifest_digest'],
                        'registry_digest': release['registry_digest'], 'source_rows_digest': rows_digest(source),
                        'storage_rows_digest': rows_digest(stored), 'comparison_digest': digest(canonical(items).encode())}})
            except Rejected as exc:
                reason = str(exc)
                known = {'registry_changed', 'manifest_digest', 'manifest_registry', 'original_reference', 'manifest_scope', 'original_digest', 'original_too_large', 'row_limit', 'source_scope', 'source_period'}
                blockers.append(reason if reason in known else 'source_validation_failed')
            return result
