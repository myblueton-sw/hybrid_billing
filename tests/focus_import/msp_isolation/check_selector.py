import copy
import unittest

from preflight import ROOT
from modules.focus_import.collection import selected_slices
from modules.focus_import.model import Rejected, tenant_id


class ScopeSelection(unittest.TestCase):
    def setUp(self):
        self.a = str(tenant_id('msp32-a1'))
        self.b = str(tenant_id('msp32-b'))
        self.state = {'windows': {}, 'deliveries': {}, 'slices': {}, 'connections': {},
            'metadata': {'configuration': [{'body': {'tenants': [
                {'key': 'msp32-a1', 'active': True, 'assignments': [['AWS', 'payer', 'a']]},
                {'key': 'msp32-b', 'active': True, 'assignments': [['AWS', 'payer', 'b']]},
            ]}}]}}
        for key, tid, month, sub in [('a', self.a, '08', 'a'), ('b', self.b, '09', 'b')]:
            scope = {'start': '2026-' + month + '-01', 'end': '2026-10-01'}
            self.state['connections'][key] = {'domains': [['AWS', 'payer', sub]]}
            self.state['windows'][key] = {'current': key, 'connection_id': key, 'window': scope}
            self.state['deliveries'][key] = {'state': 'sealed', 'batch_id': key, 'descriptor': {'connection_id': key}}
            for member in [self.a, self.b]:
                self.state['slices'][key + ':' + member] = {'state': 'hot', 'row_count': 0}

    def select(self):
        return selected_slices(self.state, {self.a: {}}, None, None)

    def test_foreign_window_is_excluded_without_mutation(self):
        before = copy.deepcopy(self.state)
        self.assertEqual([key for key, _, _ in self.select()], ['a'])
        self.assertEqual(self.state, before)

    def test_owned_zero_slice_is_preserved(self):
        self.assertEqual(self.select()[0][2]['row_count'], 0)

    def test_foreign_expired_slice_does_not_block(self):
        self.state['slices']['b:' + self.a]['state'] = 'deleted'
        self.assertEqual(len(self.select()), 1)

    def test_owned_expired_slice_fails_closed(self):
        self.state['slices']['a:' + self.a]['state'] = 'deleted'
        with self.assertRaisesRegex(Rejected, 'period_expired_unavailable'):
            self.select()

    def test_foreign_unsealed_state_is_not_inspected(self):
        self.state['deliveries']['b']['state'] = 'registered'
        self.assertEqual(len(self.select()), 1)

    def test_descriptor_connection_must_match(self):
        self.state['deliveries']['a']['descriptor']['connection_id'] = 'b'
        with self.assertRaisesRegex(Rejected, 'connection_window_identity'):
            self.select()

    def test_missing_connection_or_owner_fails_closed(self):
        saved = self.state['connections'].pop('a')
        with self.assertRaisesRegex(Rejected, 'connection_unregistered'):
            self.select()
        self.state['connections']['a'] = saved
        self.state['metadata']['configuration'][0]['body']['tenants'][0]['active'] = False
        with self.assertRaisesRegex(Rejected, 'connection_owner_unavailable'):
            self.select()

    def test_legacy_shared_window_keeps_its_explicit_compatibility(self):
        for key in self.state['windows']:
            self.state['windows'][key].pop('connection_id')
            self.state['deliveries'][key]['descriptor'].pop('connection_id')
        self.assertEqual([key for key, _, _ in self.select()], ['a', 'b'])


if __name__ == '__main__':
    unittest.main()
