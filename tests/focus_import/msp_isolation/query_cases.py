import copy
import json
from concurrent.futures import ThreadPoolExecutor
from fractions import Fraction
from threading import Lock

from fixture_data import ACTORS, COMPANY, END, KEYS, START, billed, configuration, descriptor, encoded, publish
from runtime import DIRECTORY, save
from modules.focus_import import collection, lifecycle
from modules.focus_import import authority, warehouse
from modules.focus_import.model import Rejected, canonical, digest, domains_digest, tenant_id, validate_config
from modules.focus_import.temporal import window


class Queries:
    def __init__(self, env):
        self.env = env
        self.seed = json.loads((DIRECTORY / 'seed.json').read_text())
        self.config = self.seed['config']
        self.rows = self.seed['rows']
        self.log_lock = Lock()
        self.preopened_pg = None
        self.preopened_ch = None
        self.expected = {ACTORS[KEYS[0]]: (3, Fraction(25)), ACTORS[KEYS[1]]: (1, Fraction(7)),
            ACTORS[KEYS[2]]: (2, Fraction(300)), COMPANY: (4, Fraction(32))}

    def reset_b(self):
        status = lifecycle.status('operator')
        revision = max(d['descriptor']['revision'] for d in status['deliveries'].values() if d['descriptor'].get('connection_id') == KEYS[2] + '-feed')
        publish(KEYS[2], self.rows[KEYS[2]], revision + 1)

    def query(self, actor, **options):
        result = collection.query(actor, **options)
        with self.log_lock:
            with (DIRECTORY / 'responses.jsonl').open('a') as stream:
                stream.write(json.dumps({'actor': actor, 'options': options, 'result': result}, sort_keys=True) + '\n')
        return result

    def check(self, actor, result):
        allowed = set(KEYS[:2]) if actor == COMPANY else {key for key in KEYS if ACTORS[key] == actor}
        count, total = self.expected[actor]
        assert result['row_count'] == count and billed(result) == total
        assert set(result['tenant_breakdown']) == allowed
        assert all(row['tenant_key'] in allowed for row in result['rows'])
        assert all(row['tenant_id'] == str(tenant_id(row['tenant_key'])) for row in result['rows'])
        for key in set(KEYS) - allowed:
            body = json.dumps(result)
            assert 'PRIVATE-' + key not in body
            assert 'synthetic-sub-' + key not in body
        status = lifecycle.status('operator')
        expected_graphs = set()
        expected_periods = set()
        for scope in status['windows'].values():
            cid = scope.get('connection_id')
            owners = {key for key in KEYS if cid in {key + '-feed', key + '-zero'}}
            owners &= allowed
            if not owners or not scope['current']:
                continue
            delivery = status['deliveries'][scope['current']]
            for owner in owners:
                expected_graphs.add(delivery['batch_id'] + ':' + str(tenant_id(owner)))
            expected_periods.add((scope['window']['start'], scope['window']['end']))
        assert set(result['account_hierarchies']) == expected_graphs
        assert all((c['window']['start'], c['window']['end']) in expected_periods for c in result['coverage'])
        assert all(c['tenant_key'] in allowed for c in result['coverage'])
        assert all(c['tenant_key'] in allowed for c in result['expected_coverage'])
        return allowed

    def pages(self, actor):
        first = self.query(actor, limit=1)
        self.check(actor, first)
        seen = []
        current = first
        while True:
            self.check(actor, current)
            seen.extend((row['window_id'], row['batch_id'], row['tenant_id'], row['ordinal']) for row in current['rows'])
            if current['next_cursor'] is None:
                break
            assert len(seen) < 20
            current = self.query(actor, limit=1, cursor=current['next_cursor'], expected_snapshot=first['snapshot'])
        assert len(seen) == len(set(seen)) == first['row_count']
        return {'rows': len(seen), 'billed_cost': str(billed(first)), 'snapshot': first['snapshot']}

    def normal(self):
        return {actor: self.pages(actor) for actor in self.expected}

    def provider_identity(self):
        actor = ACTORS[KEYS[0]]
        aws = self.query(actor, provider='AWS')
        gcp = self.query(actor, provider='GCP')
        assert (aws['row_count'], billed(aws)) == (2, 22)
        assert (gcp['row_count'], billed(gcp)) == (1, 3)
        assert {r['account'] for r in aws['rows']} == {r['account'] for r in gcp['rows']}
        assert {r['subaccount'] for r in aws['rows']} == {r['subaccount'] for r in gcp['rows']}
        return {'aws_rows': 2, 'gcp_rows': 1, 'same_strings_provider_qualified': True}

    def correction(self):
        first = self.query(COMPANY, limit=1)
        self.check(COMPANY, first)
        status = lifecycle.status('operator')
        revision = max(d['descriptor']['revision'] for d in status['deliveries'].values() if d['descriptor'].get('connection_id') == KEYS[2] + '-feed')
        changed = copy.deepcopy(self.rows[KEYS[2]])
        changed[0].update(BilledCost='101', EffectiveCost='101')
        publish(KEYS[2], changed, revision + 1)
        self.expected[ACTORS[KEYS[2]]] = (2, Fraction(301))
        resumed = self.query(COMPANY, limit=1, cursor=first['next_cursor'], expected_snapshot=first['snapshot'])
        self.check(COMPANY, resumed)
        assert resumed['snapshot'] == first['snapshot']
        self.check(ACTORS[KEYS[2]], self.query(ACTORS[KEYS[2]]))
        return {'a_snapshot_unchanged': True, 'b_corrected_cost': '301'}

    def ambiguous(self):
        invalid = copy.deepcopy(self.config)
        invalid['tenants'][2]['assignments'] = copy.deepcopy(invalid['tenants'][0]['assignments'])
        try:
            validate_config(invalid)
        except Rejected as error:
            assert str(error) == 'overlapping_assignment'
            return {'reason': str(error)}
        raise AssertionError('ambiguous mapping accepted')

    def wrong_source(self):
        wrong = copy.deepcopy(self.rows[KEYS[2]])
        for row in wrong:
            row.update(BillingPeriodStart=START, BillingPeriodEnd=END, ChargePeriodStart=START, ChargePeriodEnd='2026-08-02T00:00:00Z')
        wrong[0]['x_PrivateMarker'] = 'ILLEGAL-B-NEW-RAW'
        data = encoded(wrong)
        body = descriptor(KEYS[2], wrong, revision=20)
        body['connection_id'] = KEYS[0] + '-feed'
        body['window'] = window(START, END, 'month')
        body['complete_through'] = END
        result = collection.register('operator', data, '1.3', body)
        assert result['state'] == 'rejected'
        assert result['reason'] == 'connection_domain_authority'
        status = lifecycle.status('operator')
        receipt = status['receipts'][result['receipt_id']]
        assert receipt.get('key') is None and 'ILLEGAL-B-NEW-RAW' not in canonical(receipt)
        try:
            collection.register(ACTORS[KEYS[0]], data, '1.3', body)
        except Rejected as error:
            assert str(error) == 'permission'
        else:
            raise AssertionError('reader registered a source')
        return {'state': result['state'], 'reason': result['reason'], 'retained_raw_key': None, 'reader_registration_denied': True}

    def concurrency(self):
        actors = [ACTORS[KEYS[0]], ACTORS[KEYS[2]], COMPANY, ACTORS[KEYS[1]]]
        with ThreadPoolExecutor(max_workers=2) as executor:
            results = list(executor.map(lambda actor: (actor, self.query(actor)), actors))
        for actor, result in results:
            self.check(actor, result)
        return {'personas': len(results), 'contamination': False}

    def known_zero(self):
        key = KEYS[0]
        status = lifecycle.status('operator')
        original = status['connections'][key + '-feed']
        body = {name: copy.deepcopy(value) for name, value in original.items() if name != 'semantic_definition'}
        body.update(id=key + '-zero', start='2026-07-01T00:00:00Z', end=START)
        collection.configure_connection('operator', body)
        data = encoded([])
        scope = window(body['start'], body['end'], 'month')
        descriptor_body = {'connection_id': body['id'], 'window': scope, 'revision': 1, 'file_digest': digest(data),
            'row_count': 0, 'domains_digest': domains_digest([]), 'complete_through': body['end'], 'finalized': True,
            'zero_domains': [{'domain': domain, 'reason': 'synthetic_zero', 'evidence_digest': digest(canonical(domain).encode())} for domain in body['domains']]}
        registered = collection.register('operator', data, '1.3', descriptor_body)
        existing = lifecycle.status('operator')['deliveries'][registered['delivery_id']]
        if existing['state'] != 'sealed':
            collected = collection.collect('importer', registered['delivery_id'], data)
            collection.confirm('approver', registered['delivery_id'], collected['manifest_digest'])
            collection.publish('approver', registered['delivery_id'])
        result = self.query(ACTORS[key])
        self.check(ACTORS[key], result)
        zero = [c for c in result['expected_coverage'] if c['connection_id'] == body['id']]
        assert len(zero) == 2 and all(c['known_zero'] for c in zero)
        assert any(c['window']['start'].startswith('2026-07-01') for c in result['coverage'])
        self.check(ACTORS[KEYS[2]], self.query(ACTORS[KEYS[2]]))
        return {'authorized_zero_domains': len(zero), 'original_rows': result['row_count'], 'foreign_zero_disclosure': False}

    def archive_and_restore(self):
        status = lifecycle.status('operator')
        target = next(d for d in status['deliveries'].values() if d['descriptor'].get('connection_id') == KEYS[0] + '-feed')
        identity = target['batch_id'] + ':' + str(tenant_id(KEYS[0]))
        lifecycle.archive_slice('operator', identity, at='2026-09-03T00:00:00Z')
        self.check(ACTORS[KEYS[0]], self.query(ACTORS[KEYS[0]]))
        self.check(ACTORS[KEYS[2]], self.query(ACTORS[KEYS[2]]))
        lifecycle.recall('operator', identity)
        backup = lifecycle.backup('operator')
        cursor = self.query(ACTORS[KEYS[0]], limit=1)
        role = authority.reader_role(tenant_id(KEYS[0]))
        self.preopened_pg = authority.connect(self.env, role)
        self.preopened_ch = warehouse.client(self.env, role)
        assert self.preopened_pg.execute('SELECT coalesce(sum(row_count),0) AS n FROM hb_focus.slice').fetchone()['n'] == 3
        assert self.preopened_ch.query('SELECT count() FROM hb_focus.fact').result_rows[0][0] == 3
        lifecycle.revoke('operator', ACTORS[KEYS[0]])
        for options in [{}, {'cursor': cursor['next_cursor'], 'expected_snapshot': cursor['snapshot']}]:
            try:
                self.query(ACTORS[KEYS[0]], **options)
            except Rejected as error:
                assert str(error) == 'reader_revoked'
            else:
                raise AssertionError('revoked reader accepted')
        lifecycle.restore('operator', backup['backup_id'])
        try:
            self.query(ACTORS[KEYS[0]])
        except Rejected as error:
            assert str(error) == 'reader_revoked'
        else:
            raise AssertionError('restore resurrected access')
        self.check(ACTORS[KEYS[1]], self.query(ACTORS[KEYS[1]]))
        self.check(ACTORS[KEYS[2]], self.query(ACTORS[KEYS[2]]))
        return {'archive_recall_scoped': True, 'revoked_after_restore': True, 'old_cursor_denied': True}

    def close_engine_probes(self):
        if self.preopened_pg:
            self.preopened_pg.close()
        if self.preopened_ch:
            self.preopened_ch.close()

    def expiry(self):
        clock = '2300-01-01T00:00:00Z'
        for key in [KEYS[2], KEYS[1]]:
            status = lifecycle.status('operator')
            scope = next(w for w in status['windows'].values() if w.get('connection_id') == key + '-feed')
            delivery = status['deliveries'][scope['current']]
            identity = delivery['batch_id'] + ':' + str(tenant_id(key))
            preview = lifecycle.preview('operator', at=clock, identities=[identity])
            assert len(preview['targets']) == 1 and preview['held'] == []
            lifecycle.approve('lifecycle-approver', preview['manifest_digest'])
            lifecycle.delete('operator', preview['manifest_digest'], at=clock)
            try:
                self.query(ACTORS[key])
            except Rejected as error:
                assert str(error) == 'period_expired_unavailable'
            else:
                raise AssertionError('owned expired period silently omitted')
            if key == KEYS[2]:
                self.check(ACTORS[KEYS[1]], self.query(ACTORS[KEYS[1]]))
        return {'foreign_expiry_ignored': True, 'owned_expiry_blocked': True,
            'synthetic_deletion_clock': clock, 'physical_all_copy_erasure_claimed': False}
