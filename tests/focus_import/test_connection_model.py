from contextlib import contextmanager
from types import SimpleNamespace

import pytest

from modules.focus_import import collection,connections
from modules.focus_import.model import Rejected,digest,tenant_id
from test_collection_scopes import definition
from test_temporal import configuration
from tools import focus_lifecycle as cli


def test_expected_calendar_and_domain_scope():
    state={'connections':{},'windows':{},'deliveries':{}}
    cfg=configuration()
    connections.add(state,definition(),cfg)
    visible={str(tenant_id('north')):{}}
    missing=connections.coverage(state,visible,cfg,None,None,'2024-09-01T00:00:00+00:00')
    assert len(missing)==30 and {c['state'] for c in missing}=={'missing'}
    late=connections.coverage(state,visible,cfg,None,None,'2024-10-02T00:00:00+00:00')
    assert {c['state'] for c in late}=={'late'}
    assert connections.coverage(state,{str(tenant_id('south')):{}},cfg,None,None,'2024-10-02T00:00:00+00:00')==[]
    with pytest.raises(Rejected,match='economic_overlap'):connections.add(state,definition('duplicate'),cfg)
    with pytest.raises(Rejected,match='domain_authority'):connections.add(state,{**definition('unknown'),'domains':[['AWS','synthetic-account',None]]},cfg)


class File:
    suffix='.csv'
    def __init__(self,name,regular=True):self.name=name;self.regular=regular
    def is_file(self):return self.regular
    def __lt__(self,other):return self.name<other.name


class Inbox:
    def __init__(self,paths):self.paths=paths
    def iterdir(self):return iter(self.paths)


@pytest.fixture
def scheduler(monkeypatch):
    journal=SimpleNamespace(state={'deliveries':{'wanted':{'state':'registered','descriptor':{'file_digest':digest(b'wanted')}}}})
    @contextmanager
    def access(*args):yield journal,None,None
    monkeypatch.setattr(collection,'access',access)
    def read(path):
        if path.name=='000.csv':raise Rejected('input_size')
        return b'wanted' if path.name=='zzz.csv' else b'other'
    monkeypatch.setattr(cli,'bounded',read)
    calls=[]
    monkeypatch.setattr(collection,'collect',lambda *args:calls.append(args) or {'delivery_id':args[1]})
    return calls


def test_fair_scan_after_unregistered_oversized_files(scheduler):
    result=cli.run_due('importer',Inbox([File(f'{n:03}.csv') for n in range(100)]+[File('zzz.csv')]))
    assert result['registered_waiting']==0 and len(result['collected'])==1 and len(scheduler)==1


def test_scan_bounds_all_directory_entries(scheduler):
    with pytest.raises(Rejected,match='inbox_scan_bound'):
        cli.run_due('importer',Inbox([File(f'{n:05}',False) for n in range(10001)]))
    assert not scheduler
