import argparse
import json
import time
from datetime import datetime, timezone

import environment as runtime
from environment import ROOT, isolated, save
from modules.focus_import import lifecycle
from modules.focus_import.model import digest


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--phase', choices=['seed', 'baseline', 'verify', 'missing-schema', 'checkpoint'], required=True)
    parser.add_argument('--run-number', type=int, choices=range(10), default=0)
    arguments = parser.parse_args()
    runtime.configure(arguments.run_number)
    from cases import baseline, missing_schema, verify
    run = datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ') + '-' + arguments.phase
    results = []

    def record(name, function):
        started = time.monotonic()
        try:
            observation = function()
            item = {'id': name, 'status': 'PASS', 'observed': observation}
        except Exception as error:
            item = {'id': name, 'status': 'FAIL', 'error_type': type(error).__name__}
            results.append(item)
            save(run + '.json', {'complete': False, 'results': results})
            print(json.dumps(item), flush=True)
            raise
        item['seconds'] = round(time.monotonic() - started, 3)
        results.append(item)
        save(run + '.json', {'complete': False, 'results': results})
        print(json.dumps(item, sort_keys=True), flush=True)
        return observation

    with isolated(create=arguments.phase == 'seed') as (env, fixture):
        if arguments.phase == 'seed':
            config, rows, batches = fixture.initialize()
            backup = lifecycle.backup('operator')
            save('seed.json', {'config': config, 'batches': batches, 'backup': backup})
            record('R00:seed', lambda: {'tenants': 3, 'rows': 6, 'pre_revocation_backup': True})
        elif arguments.phase == 'baseline':
            assert not (runtime.DIRECTORY / 'baseline-complete.json').exists(), 'baseline already executed'
            observation = record('I17b:baseline-reproduction', lambda: baseline(env))
            save('baseline-complete.json', observation)
        elif arguments.phase == 'verify':
            assert json.loads((runtime.DIRECTORY / 'baseline-complete.json').read_text())['defect_reproduced']
            seed = json.loads((runtime.DIRECTORY / 'seed.json').read_text())
            verify(env, seed['config'], seed['backup'], record)
            save('verify-complete.json', {'status': 'PASS'})
        elif arguments.phase == 'missing-schema':
            assert json.loads((runtime.DIRECTORY / 'verify-complete.json').read_text())['status'] == 'PASS'
            seed = json.loads((runtime.DIRECTORY / 'seed.json').read_text())
            record('R11:missing-schema-restore', lambda: missing_schema(env, seed['backup']))
        else:
            assert json.loads((runtime.DIRECTORY / 'verify-complete.json').read_text())['status'] == 'PASS'
            from checkpoint_cases import run as verify_checkpoints
            verify_checkpoints(env, record)
    files = sorted((ROOT / 'tests/focus_import/reader_revocation').glob('*.py'))
    files += [ROOT / 'modules/focus_import' / name for name in ['authority.py', 'warehouse.py', 'lifecycle.py', 'service.py', 'revocation.py'] if (ROOT / 'modules/focus_import' / name).exists()]
    body = {'complete': True, 'phase': arguments.phase, 'run_number': arguments.run_number, 'results': results,
        'source_sha256': {str(path.relative_to(ROOT)): digest(path.read_bytes()) for path in files},
        'preservation': json.loads((runtime.DIRECTORY / 'default-preservation.json').read_text()),
        'not_run': ['missing-schema bootstrap restore'] if arguments.phase != 'missing-schema' else []}
    save(run + '.json', body)
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
