import argparse
import json
import time
from datetime import datetime, timezone

import database_cases as databases
from fixture_data import ACTORS, COMPANY, KEYS
from query_cases import Queries
from runtime import DIRECTORY, ROOT, isolated, save
from modules.focus_import.model import digest


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--phase', choices=['pre', 'lifecycle'], required=True)
    arguments = parser.parse_args()
    run = datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ') + '-' + arguments.phase
    results = []

    def record(name, function, kind='core'):
        start = time.monotonic()
        try:
            observed = function()
            result = {'id': name, 'status': 'PASS', 'kind': kind, 'observed': observed}
        except AssertionError as error:
            result = {'id': name, 'status': 'FAIL', 'kind': kind, 'reason': str(error)}
        except Exception as error:
            result = {'id': name, 'status': 'ERROR', 'kind': kind, 'error_type': type(error).__name__}
        result['seconds'] = round(time.monotonic() - start, 3)
        results.append(result)
        save(run + '.json', {'phase': arguments.phase, 'complete': False, 'results': results})
        print(json.dumps(result, sort_keys=True), flush=True)
        return result

    with isolated():
        suite = Queries(None)
        from modules.focus_local.database import environment
        suite.env = environment()
        if arguments.phase == 'pre':
            suite.reset_b()
            for key in KEYS:
                record('I01:' + key, lambda key=key: suite.pages(ACTORS[key]))
            record('I02:explicit-company-union', lambda: suite.pages(COMPANY))
            def labels():
                assert len({t['company_label'] for t in suite.config['tenants']}) == 1
                assert len({t['company_id'] for t in suite.config['tenants']}) == 2
                suite.check(ACTORS[KEYS[0]], suite.query(ACTORS[KEYS[0]]))
                return {'equal_labels_distinct_companies': True, 'ordinary_reader_inherits_no_sibling': True}
            record('I03:equal-label-shared-payer', labels)
            record('I04:provider-qualified-identity', suite.provider_identity)
            record('I05:pg-spoofed-context', lambda: databases.postgres_scope(suite))
            record('I06:pg-privileges', lambda: databases.postgres_privileges(suite))
            record('I07:ch-predicates-union', lambda: databases.clickhouse_scope(suite))
            record('I08:ch-settings', lambda: databases.clickhouse_settings(suite))
            def metadata():
                result = suite.query(COMPANY, limit=1)
                suite.check(COMPANY, result)
                assert len(result['coverage']) == len(result['account_hierarchies'])
                return {'foreign_coverage_count': 0, 'foreign_graph_count': 0, 'authorized_coverage_count': len(result['coverage'])}
            record('I09:metadata-isolation', metadata)
            record('I10:foreign-correction-stable-cursor', suite.correction)
            record('I11:ambiguous-owner', suite.ambiguous)
            record('I12:wrong-source-no-raw', suite.wrong_source)
            record('I13:audience-tamper', lambda: databases.audience_tamper(suite))
            record('I14:concurrent-readers', suite.concurrency)
            record('I16:unsigned-storage', lambda: databases.unsigned_storage(suite), 'storage-adjacent')
            record('I17a:untrusted-engine-metadata', lambda: databases.engine_metadata(suite), 'conditional-direct-SQL')
            if all(r['status'] == 'PASS' for r in results if r['kind'] == 'core'):
                record('I20:owned-known-zero', suite.known_zero)
        else:
            prior = json.loads((DIRECTORY / 'latest-pre.json').read_text())
            assert prior['core_status'] == 'PASS'
            assert prior['source_sha256']['modules/focus_import/collection.py'] == digest((ROOT / 'modules/focus_import/collection.py').read_bytes())
            from fractions import Fraction
            suite.expected[ACTORS[KEYS[2]]] = (2, Fraction(301))
            record('I15:archive-revoke-restore', suite.archive_and_restore)
            if results[-1]['status'] == 'PASS':
                record('I17b:revoked-internal-SQL', lambda: databases.revoked_engine(suite), 'conditional-direct-SQL')
                record('I21:owned-versus-foreign-expiry', suite.expiry)
        suite.close_engine_probes()
        source_paths = ['modules/focus_import/collection.py'] + [str(path.relative_to(ROOT)) for path in sorted((ROOT / 'tests/focus_import/msp_isolation').glob('*.py'))]
        body = {'run': run, 'phase': arguments.phase, 'complete': True, 'results': results,
            'core_status': 'PASS' if all(r['status'] == 'PASS' for r in results if r['kind'] == 'core') else 'FAIL',
            'source_sha256': {path: digest((ROOT / path).read_bytes()) for path in source_paths},
            'response_log_sha256': digest((DIRECTORY / 'responses.jsonl').read_bytes()),
            'external_customer_portal': 'UNVERIFIED_NOT_IMPLEMENTED',
            'not_run': {'I18': 'External user authentication, MSP multi-customer staff grants, department human permissions and customer field projection are not implemented.',
                'I19': 'Customer search/cache/download/export/queued-worker entry points are not implemented.'}}
    body['preservation'] = json.loads((DIRECTORY / 'default-preservation.json').read_text())
    save(run + '.json', body)
    if arguments.phase == 'pre':
        save('latest-pre.json', body)
    print(json.dumps({'run': run, 'phase': arguments.phase, 'core_status': body['core_status'],
        'cases': len(results), 'conditional_failures': sum(r['status'] != 'PASS' for r in results if r['kind'] != 'core'),
        'default_runtime_unchanged': body['preservation']['default_runtime_unchanged']}), flush=True)
    return 0 if body['core_status'] == 'PASS' else 2


if __name__ == '__main__':
    raise SystemExit(main())
