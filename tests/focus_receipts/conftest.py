import copy
from types import SimpleNamespace
from uuid import UUID

import pytest

from modules.focus_import.connections import identity
from modules.focus_import.model import canonical, digest, tenant_id, versions
from modules.focus_import.receipt_projection import project
from modules.focus_import.temporal import window


class ReceiptFixture:
    def __init__(self):
        self.at = '2026-02-02T00:00:00+00:00'
        self.window = window('2026-01-01T00:00:00+00:00', '2026-02-01T00:00:00+00:00', 'month')
        tenants = [self.tenant(key, company) for key, company in [('north-a', 'north'), ('north-b', 'north'), ('south-a', 'south')]]
        self.config = {'source': 'receipt-test', 'tenants': tenants}
        config_digest = digest(canonical(self.config).encode())
        self.authority = SimpleNamespace(actor='reader-company-north', is_view=True, bindings={'config_digest': config_digest, 'source': self.config['source']})
        self.context = {'audience': {'actor': self.authority.actor, 'active': True},
                        'tenants': [{'tenant_id': tenant_id(t['key']), 'properties': copy.deepcopy(t), 'active': True} for t in tenants[:2]]}
        self.state = {'config_digest': config_digest, 'metadata': {'configuration': [{'digest': config_digest, 'body': copy.deepcopy(self.config)}], 'batch': []},
                      'deliveries': {}, 'receipts': {}, 'windows': {}, 'connections': {},
                      'policies': {t['key']: {'revision': 1} for t in tenants}}
        self.add_connection('north', ['north-a', 'north-b'])
        self.add_connection('south', ['south-a'])
        self.add_connection('mixed', ['north-a', 'south-a'])

    def tenant(self, key, company):
        return {'key': key, 'actor': 'reader-' + key, 'company_id': company, 'active': True,
                'timezone': 'UTC', 'assignments': [['AWS', key, None, 'USD']], 'currencies': ['USD']}

    def add_connection(self, cid, keys):
        domains = [t['assignments'][0][:3] for t in self.config['tenants'] if t['key'] in keys]
        self.state['connections'][cid] = {'id': cid, 'dataset': cid, 'coverage_key': cid, 'version': '1.0', 'grain': 'month',
                                          'start': self.window['start'], 'end': self.window['end'], 'lag_seconds': 0,
                                          'domains': domains, 'semantic_profile': 'exploratory-query-only', 'max_replacements': 3}

    def descriptor(self, revision=1, connection='north', count=2):
        result = {'window': copy.deepcopy(self.window), 'revision': revision, 'file_digest': digest(f'file-{connection}-{revision}'.encode()),
                  'row_count': count, 'domains_digest': 'd' * 64, 'complete_through': self.window['end'], 'finalized': True}
        if connection is not None:
            result['connection_id'] = connection
        return result

    def add_delivery(self, revision=1, connection='north', count=2, stage='sealed', positive=None):
        descriptor = self.descriptor(revision, connection, count)
        key = digest(canonical({'version': '1.0', 'descriptor': descriptor}).encode())
        mapping = digest(canonical(versions()['1.0']).encode())
        batch_id = str(UUID(int=int(key[:32], 16)))
        delivery = {'descriptor': descriptor, 'version': '1.0', 'mapping_digest': mapping, 'state': stage,
                    'received_at': '2026-01-02T00:00:00+00:00', 'batch_id': None, 'transport_revision': None}
        if connection is not None:
            delivery['connection_digest'] = digest(canonical(self.state['connections'][connection]).encode())
        if stage != 'registered':
            if positive is None:
                target = 'south-a' if connection == 'south' else 'north-a'
                positive = {target: count}
            slices = {str(tenant_id(t['key'])): {'row_count': positive.get(t['key'], 0)} for t in self.config['tenants']}
            manifest = {'version': '1.0', 'source': self.config['source'], 'mapping_digest': mapping,
                        'config_digest': self.state['config_digest'], 'source_revision': revision,
                        'row_count': count, 'file_digest': descriptor['file_digest'], 'file_bytes': 128,
                        'input_format': 'csv', 'slices': slices}
            manifest_digest = digest(canonical(manifest).encode())
            delivery.update(batch_id=batch_id, transport_revision=revision, manifest_digest=manifest_digest)
            self.state['metadata']['batch'].append({'batch_id': batch_id, 'source': self.config['source'],
                'source_revision': revision, 'file_digest': descriptor['file_digest'], 'config_digest': self.state['config_digest'],
                'mapping_digest': mapping, 'manifest': manifest, 'manifest_digest': manifest_digest, 'state': stage})
        self.state['deliveries'][key] = delivery
        connection_body = self.state['connections'].get(connection)
        wid = identity(connection_body, self.window)
        target = self.state['windows'].setdefault(wid, {'window': copy.deepcopy(self.window), 'revisions': [], 'current': None, 'retired': False})
        if connection is not None:
            target['connection_id'] = connection
        target['revisions'].append(key)
        if stage == 'sealed':
            target['current'] = key
        return key

    def add_quarantine(self, revision=1, connection='north', stage='quarantined', admitted=True, descriptor_changes=None):
        descriptor = self.descriptor(revision, connection)
        descriptor.update(descriptor_changes or {})
        descriptor_digest = digest(canonical(descriptor).encode()) if admitted else None
        body = {'file_digest': descriptor['file_digest'] if admitted else digest(f'quarantine-{connection}-{revision}'.encode()), 'version': '1.0', 'connection_id': connection,
                'reason': 'decimal_syntax', 'admitted': admitted, 'descriptor_digest': descriptor_digest}
        key = digest(canonical(body).encode())
        item = {**body, 'descriptor': descriptor if admitted else None, 'state': stage,
                'received_at': '2026-01-03T00:00:00+00:00', 'attempts': 0, 'root_receipt': key, 'replaced_by': None,
                'key': 'DO-NOT-EXPOSE-RAW-PATH', 'holds': {'DO-NOT-EXPOSE': 'timestamp'}}
        if admitted:
            wanted = self.state['connections'][connection]['domains']
            item['tenant_keys'] = sorted(t['key'] for t in self.config['tenants'] if t['assignments'][0][:3] in wanted)
        self.state['receipts'][key] = item
        return key

    def render(self, params=None):
        return project(self.authority.actor, params or {}, self.authority, self.context, self.state, self.at)


@pytest.fixture
def receipt_fixture():
    return ReceiptFixture()
