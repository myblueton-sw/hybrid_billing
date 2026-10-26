import copy
import json

from fixture_data import ACTORS, B_START, COMPANY, KEYS, billed, initialize, publish
from runtime import DIRECTORY, isolated, save
from modules.focus_import import collection
from modules.focus_import.model import Rejected, digest


def main():
    results = []
    with isolated(create=True):
        config, rows, batches = initialize()
        save('seed.json', {'config': config, 'rows': rows, 'batches': batches})
        view = collection.query(COMPANY, limit=1)
        assert view['row_count'] == 4 and billed(view) == 32
        foreign_periods = [c['window']['start'] for c in view['coverage'] if c['window']['start'].startswith(B_START[:10])]
        foreign_batches = [key for key in view['account_hierarchies'] if key.startswith(batches[KEYS[2]]['batch_id'] + ':')]
        results.append({'id': 'I09', 'status': 'FAIL' if foreign_periods or foreign_batches else 'PASS',
            'authorized_rows': view['row_count'], 'authorized_billed_cost': str(billed(view)),
            'foreign_coverage_count': len(foreign_periods), 'foreign_batch_graph_count': len(foreign_batches),
            'foreign_private_rows_visible': any(KEYS[2] in row['source_json'] for row in view['rows'])})
        changed = copy.deepcopy(rows[KEYS[2]])
        changed[0].update(BilledCost='101', EffectiveCost='101')
        publish(KEYS[2], changed, revision=2)
        try:
            resumed = collection.query(COMPANY, limit=1, cursor=view['next_cursor'], expected_snapshot=view['snapshot'])
            assert billed(resumed) == 32
            outcome = 'PASS'
            reason = None
        except Rejected as error:
            outcome = 'FAIL'
            reason = str(error)
        results.append({'id': 'I10', 'status': outcome, 'reason': reason, 'change_scope': 'customer-b-only'})
        save('reproduction.json', {'base_commit': '8153975c19fe27e32569e5416ef5f9298362ec08',
            'source_sha256': digest((DIRECTORY.parents[2] / 'modules/focus_import/collection.py').read_bytes()),
            'results': results, 'namespace': 'isolated-HB-32'})
    print(json.dumps({'status': 'REPRODUCED' if any(r['status'] == 'FAIL' for r in results) else 'NOT_REPRODUCED', 'results': results}))


if __name__ == '__main__':
    main()
