import fcntl
import json
import os
import subprocess

from runtime import DIRECTORY, ROOT, fingerprints, isolated
from modules.focus_import import collection, customer_usage, model, authority, warehouse
from modules.focus_local.database import environment


def main():
    run = ROOT / '.local/runs/HB-34-usage-projection'
    run.mkdir(parents=True, exist_ok=True)
    before = fingerprints()
    with isolated():
        result = collection.query('reader-msp33-a2-r4', '2026-08-01T00:00:00Z', '2026-09-01T00:00:00Z', limit=10)
        head = json.loads((DIRECTORY / 'control/head.json').read_text())
        profile = {'kind': 'source-usage-v1', 'installation': head['installation'], 'source': 'hb33-reader-revocation-r4',
            'actor': 'reader-msp33-a2-r4', 'company_id': 'company-a', 'tenant_id': str(model.tenant_id('msp33-a2-r4')),
            'snapshot': result['snapshot'], 'start': '2026-08-01T00:00:00Z', 'end': '2026-09-01T00:00:00Z',
            'grain': 'month', 'services': {'Synthetic Shared Compute': 'Compute'}, 'units': {'Hours': 'hours'}}
        safe = customer_usage.query(json.dumps(profile), 10, lambda value: value)
        assert len(safe['rows']) == 1 and safe['rows'][0]['Values']['quantity'] == '2.000000000000000000'
        assert safe['rows'][0]['Values']['service'] == 'Compute'
        encoded = json.dumps(safe)
        assert all(value not in encoded for value in ['PRIVATE-', 'synthetic-shared-payer', 'billed_cost', 'source_json', 'account_context'])
        with authority.connect(environment()) as conn:
            pg_version = conn.execute('SHOW server_version').fetchone()['server_version']
        ch = warehouse.client(environment(), 'probe_admin')
        try: ch_version = ch.query('SELECT version()').result_rows[0][0]
        finally: ch.close()
    profile_path = run / 'profile.json'
    profile_path.write_text(json.dumps(profile))
    python = str(ROOT / 'tests/engine_probes/.venv/bin/python')
    bridge = str(ROOT / 'tests/customer_usage/bridge.py')
    process = subprocess.Popen([python, bridge], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True)
    try:
        process.stdin.write(json.dumps({'profile': json.dumps(profile), 'limit': 10}) + '\n')
        process.stdin.flush()
        ready = json.loads(process.stdout.readline())
        assert ready['status'] == 'ready'
        with (DIRECTORY / 'focus-coordinator.lock').open('a') as stream:
            try: fcntl.flock(stream, fcntl.LOCK_EX | fcntl.LOCK_NB)
            except BlockingIOError: pass
            else: raise AssertionError('source protection released before Go authorization')
        process.stdin.write('release\n'); process.stdin.flush(); process.stdin.close()
        assert json.loads(process.stdout.readline()) == {'status': 'released'}
        assert process.wait(timeout=10) == 0 and process.stdout.read() == ''
        with (DIRECTORY / 'focus-coordinator.lock').open('a') as stream:
            fcntl.flock(stream, fcntl.LOCK_EX | fcntl.LOCK_NB)
            fcntl.flock(stream, fcntl.LOCK_UN)
    finally:
        if process.poll() is None: process.kill(); process.wait()
    env = {**os.environ, 'HB34_USAGE_PROFILE': str(profile_path), 'HB34_USAGE_PYTHON': python, 'HB34_USAGE_BRIDGE': bridge}
    result = subprocess.run(['go', 'test', '-race', '-count=1', '-run', '^TestEngineGateway$', './modules/customerusage'], cwd=ROOT, env=env, capture_output=True, text=True, timeout=90)
    print(result.stdout, end='')
    if result.returncode:
        print('Go engine integration failed; diagnostics retained locally')
        (run / 'go-error.txt').write_text(result.stderr)
        return 1
    assert fingerprints() == before
    evidence = {'status': 'PASS', 'postgres': pg_version, 'clickhouse': ch_version, 'synthetic_rows': 1,
        'source_lock_held_until_release': True, 'go_engine_integration': True,
        'default_and_prior_bindings_controls_unchanged': True, 'database_writes': False,
        'external_authentication': 'NOT_RUN', 'source': 'HB-33-r4 synthetic read-only'}
    (run / 'results.json').write_text(json.dumps(evidence, indent=2) + '\n')
    print(json.dumps(evidence))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
