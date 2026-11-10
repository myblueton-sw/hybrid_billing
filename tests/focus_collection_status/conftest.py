import copy
from types import SimpleNamespace

import pytest

from modules.focus_import.collection_status import Projection
from modules.focus_import.model import tenant_id
from modules.focus_import.temporal import window
from tests.focus_receipts.conftest import ReceiptFixture


class StatusFixture(ReceiptFixture):
    def __init__(self):
        super().__init__()
        self.scopes = []

    def scope(self, batch='PRIVATE-BATCH', key='PRIVATE-WINDOW', tenant='north-a', version='1.0', period=None):
        period = copy.deepcopy(period or self.window)
        delivery = {'batch_id': batch, 'version': version, 'descriptor': {'window': period}, 'private': 'DO-NOT-EXPOSE-DELIVERY'}
        item = {'batch_id': batch, 'tenant_id': str(tenant_id(tenant)), 'private': 'DO-NOT-EXPOSE-SLICE'}
        self.scopes.append((key, delivery, item))
        return delivery

    def row(self, provider='AWS', provenance='official-public', tenant='north-a', batch='PRIVATE-BATCH', key='PRIVATE-WINDOW', ordinal=1):
        return (key, batch, ordinal, tenant,
                {'tenant_id': str(tenant_id(tenant)), 'provider': provider,
                 'account': 'DO-NOT-EXPOSE-ACCOUNT', 'billed_cost': 'DO-NOT-EXPOSE-MONEY',
                 'source_json': {'x_HB_EvidenceClass': provenance, 'private': 'DO-NOT-EXPOSE-SOURCE'}})

    def projection(self, deliver=None):
        reader = Projection(self.authority.actor, deliver)
        reader.preflight(SimpleNamespace(state=self.state), SimpleNamespace(a=self.authority), self.context, self.scopes)
        return reader

    def status(self, rows=None, result=None):
        rows = [] if rows is None else rows
        return self.projection().consume(result if result is not None else {
            'classification': 'exploratory-query-only', 'row_count': len(rows),
            'snapshot': 'DO-NOT-EXPOSE-PRIVATE-SOURCE-SNAPSHOT', 'rows': ['DO-NOT-EXPOSE-RAW-ROWS'],
        }, rows)


@pytest.fixture
def status_fixture():
    return StatusFixture()
