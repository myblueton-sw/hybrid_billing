import argparse
import json
import subprocess
import sys
import time
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path

from oracle import FIELDS, MEASURES, ROOT, canonical, check_detail, delivery_rows, groups, instant, load, periods, row_counter, sha, use_cases


ALIASES = {'AWS': 'AWS', 'Microsoft': 'Azure', 'Google Cloud': 'GCP', 'Oracle': 'OCI', 'Alibaba Cloud': 'Alibaba', 'VMware': 'VMware'}


class Verification:
    def __init__(self, output, delivery_directory):
        self.output = output
        self.output.mkdir(parents=True, exist_ok=False)
        self.commands = []
        self.cases = []
        self.rows, self.tenants, self.config, self.provenance = load()
        self.python = ROOT / 'tests/engine_probes/.venv/bin/python'
        self.delivery_directory = delivery_directory

    def execute(self, *arguments, expected='ok', reason=None):
        start = time.monotonic()
        command = [str(self.python), 'tools/focus_lifecycle.py', *arguments]
        completed = subprocess.run(command, cwd=ROOT, capture_output=True, text=True, timeout=120)
        try:
            result = json.loads(completed.stdout)
        except ValueError:
            result = {'status': 'invalid_response'}
        self.commands.append({'arguments': list(arguments), 'exit': completed.returncode,
            'seconds': round(time.monotonic() - start, 3), 'result': result})
        (self.output / 'commands.json').write_text(json.dumps(self.commands, sort_keys=True))
        assert result.get('status') == expected, (arguments, result.get('status'), result.get('reason'))
        assert completed.returncode == (0 if expected == 'ok' else 2)
        if reason:
            assert result.get('reason') == reason, (reason, result.get('reason'))
        return result

    def source(self, actor, provider=None, service=None):
        if actor in {t['actor'] for t in self.tenants.values()}:
            allowed = {t['key'] for t in self.tenants.values() if t['actor'] == actor}
        else:
            view = next(v for v in self.config['views'] if v['actor'] == actor and v['active'])
            allowed = set(view['tenant_keys'])
            assert all(self.tenants[t]['company_id'] == view['company_id'] for t in allowed)
        return [r for r in self.rows if r['_tenant'] in allowed and (provider is None or r['ProviderName'] == provider)
            and (service is None or r['ServiceName'] == service)], allowed

    def check_page(self, result, expected, allowed, grain):
        assert result['row_count'] == len(expected)
        assert result['groups'] == groups(expected)
        assert result['currency_conversion'] is False
        assert result['range_basis'] == 'delivery-window-overlap'
        assert result['period_attribution'] == 'charge-start-in-tenant-timezone-no-proration'
        assert set(result['tenant_breakdown']) == allowed
        for key in allowed:
            subset = [r for r in expected if r['_tenant'] == key]
            actual = result['tenant_breakdown'][key]
            assert actual['row_count'] == len(subset)
            assert actual['groups'] == groups(subset)
            assert actual['periods'] == periods(subset, self.tenants[key]['timezone'], grain)

    def query(self, name, actor, provider=None, service=None, grain='month', extra=(), expected_rows=None, page_size=100):
        expected, allowed = self.source(actor, provider, service)
        if expected_rows is not None:
            expected = expected_rows
        arguments = ['query', '--actor', actor, '--limit', str(page_size), '--grain', grain]
        if provider:
            arguments += ['--provider', ALIASES.get(provider, provider)]
        if service:
            arguments += ['--service', service]
        arguments += list(extra)
        actual = []
        identities = []
        cursor = None
        snapshot = None
        pages = 0
        while True:
            current = arguments + (['--cursor', json.dumps(cursor), '--snapshot', snapshot] if cursor is not None else [])
            result = self.execute(*current)
            self.check_page(result, expected, allowed, grain)
            if snapshot is not None:
                assert result['snapshot'] == snapshot
            snapshot = result['snapshot']
            for row in result['rows']:
                assert row['tenant_key'] in allowed
                assert row['semantic_context'] is None
                identities.append((row['window_id'], row['batch_id'], row['ordinal'], row['tenant_id']))
                actual.append(check_detail(row))
            pages += 1
            cursor = result['next_cursor']
            if cursor is None:
                break
            assert pages < 100
        assert len(identities) == len(set(identities))
        assert row_counter(actual) == row_counter(expected)
        expected_cases, actual_cases = use_cases(expected), use_cases(actual)
        assert expected_cases == actual_cases
        case = {'name': name, 'status': 'PASS', 'rows': len(actual), 'pages': pages,
            'source_values_digest': sha(canonical(sorted(row_counter(actual).items())).encode()),
            'groups_digest': sha(canonical(result['groups']).encode()), 'use_cases': actual_cases,
            'snapshot': snapshot, 'grain': grain}
        self.cases.append(case)
        print(json.dumps({'case': name, 'rows': len(actual), 'pages': pages, 'status': 'PASS'}), flush=True)
        return result, actual

    def run(self):
        before = self.execute('status', '--actor', 'operator')
        self.rows, delivery_pins = delivery_rows(self.rows, self.config, before, self.delivery_directory)
        self.query('company_union', 'reader-company-north')
        self.query('tenant_a', 'reader-north-a')
        self.query('tenant_b', 'reader-north-b')
        self.query('foreign_only', 'reader-foreign')
        for provider in sorted(ALIASES):
            self.query('provider:' + provider, 'reader-company-north', provider, page_size=73)
            for service in sorted({r['ServiceName'] for r in self.rows if r['ProviderName'] == provider}):
                self.query('service:' + provider + ':' + service, 'reader-company-north', provider, service)
            self.query('daily:' + provider, 'reader-company-north', provider, grain='day')
        self.query('unknown_provider', 'reader-company-north', 'Not A Provider')
        self.query('unknown_service', 'reader-company-north', 'AWS', 'Not A Service')
        self.query('other_tenant_provider', 'reader-north-b', 'AWS')
        self.query('foreign_alibaba', 'reader-foreign', 'Alibaba Cloud')
        self.query('empty_service_string', 'reader-company-north', extra=('--service', ''), expected_rows=[])
        sample = self.execute('query', '--actor', 'reader-company-north', '--provider', 'AWS', '--limit', '1')
        cursor = json.dumps(sample['next_cursor'])
        self.execute('query', '--actor', 'reader-company-north', '--provider', 'AWS', '--cursor', cursor,
            expected='rejected', reason='query_snapshot_required')
        for actor, provider, snapshot in [('reader-company-north', 'AWS', '0' * 64),
            ('reader-company-north', 'GCP', sample['snapshot']), ('reader-north-a', 'AWS', sample['snapshot'])]:
            self.execute('query', '--actor', actor, '--provider', provider, '--cursor', cursor, '--snapshot', snapshot,
                expected='rejected', reason='query_snapshot_changed')
        self.execute('query', '--actor', 'importer', expected='rejected', reason='permission')
        self.cases.append({'name': 'snapshot_and_role_negatives', 'status': 'PASS', 'asserted_rejections': 5})
        for start, end, name in [('2024-09-01T00:00:00Z', '2024-10-01T00:00:00Z', 'billing_window'),
            ('2024-09-15T00:00:00Z', '2024-09-16T00:00:00Z', 'overlap_not_row_clip'),
            ('2024-10-01T00:00:00Z', '2024-11-01T00:00:00Z', 'exclusive_boundary'),
            ('2024-08-01T00:00:00Z', '2024-09-01T00:00:00Z', 'before_boundary')]:
            selected = set()
            for window_id, window in before['windows'].items():
                scope = window['window']
                if window['current'] and instant(scope['start']) < instant(end) and instant(scope['end']) > instant(start):
                    selected.add(window_id)
            slices = [r for r in self.rows if r['_tenant'] in {'north-a', 'north-b'} and r['_window_id'] in selected]
            self.query(name, 'reader-company-north', extra=('--start', start, '--end', end), expected_rows=slices)
        after = self.execute('status', '--actor', 'operator')
        assert before == after
        self.cases.append({'name': 'runtime_unchanged', 'status': 'PASS', 'journal_sequence': after['sequence']})
        source_summary = {}
        for provider in sorted(ALIASES):
            rows = [r for r in self.rows if r['ProviderName'] == provider]
            source_summary[provider] = {'all_rows': len(rows),
                'company_rows': sum(r['_tenant'] != 'foreign' for r in rows),
                'services': sorted({r['ServiceName'] for r in rows}),
                'charge_categories': dict(Counter(r['ChargeCategory'] for r in rows)),
                'evidence_classes': sorted({r['x_HB_EvidenceClass'] for r in rows}),
                'source_use_cases': use_cases(rows),
                'nullable_numeric': {k: sum(r.get(FIELDS[k]) in {'', 'NULL', 'null', '\\N'} for r in rows) for k in MEASURES},
                'negative_billed_rows': sum(r['BilledCost'] not in {'', 'NULL', 'null', '\\N'} and r['BilledCost'].startswith('-') for r in rows)}
        evidence = {'status': 'PASS', 'cases': self.cases, 'case_count': len(self.cases), 'command_count': len(self.commands),
            'declared_version': '1.0', 'provenance': self.provenance, 'providers': source_summary,
            'delivery_pins': delivery_pins,
            'journal_sequence': after['sequence'], 'runtime_unchanged': True,
            'commands_sha256': sha((self.output / 'commands.json').read_bytes()),
            'oracle_sha256': sha(Path(__file__).with_name('oracle.py').read_bytes()),
            'runner_sha256': sha(Path(__file__).read_bytes()),
            'use_case_mode': 'offline derivation from verified returned source_json, not new native query endpoints',
            'source_data_committed': False}
        (self.output / 'summary.json').write_text(json.dumps(evidence, sort_keys=True, indent=2) + '\n')
        print(json.dumps({'status': 'PASS', 'cases': len(self.cases), 'commands': len(self.commands), 'output': str(self.output)}), flush=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--delivery-directory', type=Path, required=True)
    arguments = parser.parse_args()
    verification = Verification(arguments.output, arguments.delivery_directory)
    try:
        verification.run()
    except Exception as error:
        (arguments.output / 'failure.json').write_text(json.dumps({'error': type(error).__name__, 'detail': str(error)}) + '\n')
        raise


if __name__ == '__main__':
    main()
