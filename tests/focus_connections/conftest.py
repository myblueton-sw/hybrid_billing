import copy
from types import SimpleNamespace

import pytest

from modules.focus_import.connection_projection import project
from modules.focus_import.model import canonical, digest, tenant_id


class ConnectionFixture:
    def __init__(self):
        self.at = '2026-02-02T00:00:00+00:00'
        tenants = [self.tenant(key, company) for key, company in [('north-a', 'north'), ('north-b', 'north'), ('south-a', 'south')]]
        self.config = {'source': 'connection-test', 'tenants': tenants}
        self.authority = SimpleNamespace(actor='reader-company-north', is_view=True, bindings={})
        self.context = {'audience': {'actor': self.authority.actor, 'active': True},
                        'tenants': [{'tenant_id': tenant_id(t['key']), 'properties': copy.deepcopy(t), 'active': True} for t in tenants[:2]]}
        self.state = {'metadata': {'configuration': []}, 'connections': {},
                      'policies': {t['key']: {'revision': 1, 'private': 'HIDDEN-POLICY-VALUE'} for t in tenants},
                      'deliveries': {'PRIVATE-DELIVERY': {'body': 'DO-NOT-LOAD'}},
                      'receipts': {'PRIVATE-RECEIPT': {'key': 'PRIVATE-OBJECT-PATH'}}}
        self.bind_config()
        self.add_connection('north', ['north-a', 'north-b'])
        self.add_connection('south', ['south-a'])
        self.add_connection('mixed', ['north-a', 'south-a'])

    def tenant(self, key, company):
        return {'key': key, 'actor': 'reader-' + key, 'company_id': company, 'active': True,
                'timezone': 'UTC', 'assignments': [['AWS', key, None, 'USD']], 'currencies': ['USD']}

    def bind_config(self):
        value = digest(canonical(self.config).encode())
        self.state['config_digest'] = value
        self.state['metadata']['configuration'] = [{'digest': value, 'body': copy.deepcopy(self.config)}]
        self.authority.bindings = {'config_digest': value, 'source': self.config['source']}

    def add_connection(self, cid, keys, start='2026-01-01T00:00:00+00:00', end='2026-02-01T00:00:00+00:00', grain='month'):
        self.state['connections'][cid] = {'id': cid, 'dataset': 'HIDDEN-DATASET', 'coverage_key': 'HIDDEN-COVERAGE',
            'version': '1.0', 'grain': grain, 'start': start, 'end': end, 'lag_seconds': 0,
            'domains': [t['assignments'][0][:3] for t in self.config['tenants'] if t['key'] in keys],
            'semantic_profile': 'exploratory-query-only', 'max_replacements': 3,
            'capabilities': {'HIDDEN-CAPABILITY': 'PRIVATE-CONFIGURATION'}}

    def render(self, params=None):
        return project(self.authority.actor, params if params is not None else {}, self.authority, self.context, self.state, self.at)


@pytest.fixture
def connection_fixture():
    return ConnectionFixture()
