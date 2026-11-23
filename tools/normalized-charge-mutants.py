import argparse
import json
import pathlib
import re
import shutil
import subprocess
import tempfile

parser = argparse.ArgumentParser()
parser.add_argument('--output', required=True)
args = parser.parse_args()
root = pathlib.Path(__file__).resolve().parents[1]
out = pathlib.Path(args.output).resolve()
out.mkdir(parents=True, exist_ok=True)
producer = 'apps/api/server/approvalproducer.go'
normalized = 'apps/api/server/approvalnormalized.go'
cases = [
    ('source_version', normalized, 'run.Spec.Mapping.SourceVersion != b.SourceVersion || ', ''),
    ('invalid_source_text', normalized, 'if !utf8.ValidString(name) || !utf8.ValidString(cell.Text) {', 'if (!utf8.ValidString(name) || !utf8.ValidString(cell.Text)) && false {'),
    ('customer_attribution', normalized, 'contract.CustomerID != profile.BuyerID || ', ''),
    ('partial_scope', normalized, 'after += len(page.Rows)', 'after = run.Spec.Rows'),
    ('result_digest', producer, 'sha256.Sum256(envelope)', 'sha256.Sum256(encoded)'),
    ('identity_conflict', producer, 'if !bytes.Equal(prior.input, encoded) {', 'if !bytes.Equal(prior.input, encoded) && false {'),
    ('output_bound', producer, 'if len(envelope) > candidateBytes {', 'if len(envelope) > candidateBytes && false {'),
    ('discard_exclusions', producer, 'envelope, err := json.Marshal(chargeEnvelope{', 'result.Excluded = nil\n\tenvelope, err := json.Marshal(chargeEnvelope{'),
]
results = []
with tempfile.TemporaryDirectory(prefix='hb-normalized-charge-') as temp:
    work = pathlib.Path(temp)
    for name in ['modules', 'apps/api', 'packages/primitives']:
        shutil.copytree(root / name, work / name, ignore=shutil.ignore_patterns('__pycache__', '*.pyc'))
    for name in ['go.mod', 'go.sum']:
        if (root / name).exists():
            shutil.copy2(root / name, work / name)

    def run(name):
        result = subprocess.run(
            ['go', 'test', './apps/api/server', '-run', '^Test(NormalizedCharges|ChargeCandidate)', '-count=1', '-timeout=60s'],
            cwd=work, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
        )
        (out / (name + '.log')).write_text(result.stdout)
        return result

    if run('baseline').returncode:
        raise RuntimeError('Baseline failed')
    for name, filename, before, after in cases:
        path = work / filename
        original = path.read_text()
        if original.count(before) != 1:
            raise RuntimeError('Target not unique: ' + name)
        try:
            path.write_text(original.replace(before, after))
            result = run(name)
            if result.returncode == 0:
                outcome = 'survived'
            elif '[build failed]' in result.stdout:
                outcome = 'compile_failed'
            elif re.search(r'^--- FAIL: Test(NormalizedCharges|ChargeCandidate)', result.stdout, re.M) and 'panic:' not in result.stdout and 'signal:' not in result.stdout:
                outcome = 'killed'
            else:
                outcome = 'execution_failed'
            results.append({'name': name, 'outcome': outcome})
            print(name + ': ' + outcome, flush=True)
        finally:
            path.write_text(original)
(out / 'results.json').write_text(json.dumps({'baseline': 'passed', 'mutants': results}, indent=2))
if any(result['outcome'] != 'killed' for result in results):
    raise SystemExit(1)
