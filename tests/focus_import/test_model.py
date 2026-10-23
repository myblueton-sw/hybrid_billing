import copy
import csv
import io
import json
import random
from decimal import Decimal
from fractions import Fraction

import pytest

from modules.focus_import.model import *


def test_decimal_generated_oracle():
    randomizer=random.Random(270023)
    for _ in range(2000):
        places=randomizer.randrange(19)
        n=randomizer.randrange(-10**18,10**18)
        text=('-' if n<0 else '')+str(abs(n)//10**places)+('.'+str(abs(n)%10**places).zfill(places) if places else '')
        observed=number(text)
        assert Fraction(observed)==Fraction(text)
        assert Fraction(coefficient(observed),10**18)==Fraction(text)


@pytest.mark.parametrize('value',['NaN','Infinity','1.0000000000000000001','1e100','1e-19','100000000000000000000',' 1','1_000'])
def test_numeric_rejections(value):
    with pytest.raises(Rejected):number(value)


def test_partition_and_null_controls(official):
    config=demo_config(official)
    rows=assign(decode(official,versions()['1.0']),config)
    assert len(rows)==1000
    assert len({r['ordinal'] for r in rows})==1000
    assert {t['key']:sum(r['tenant_id']==tenant_id(t['key']) for r in rows) for t in config['tenants']}=={'demo-a':942,'demo-b':58}
    assert sum(r['contracted_cost'] is None for r in rows)==7
    assert sum(r['consumed_quantity'] is None for r in rows)==1
    assert all('literal:NULL' in r['missing_reasons'] for r in rows if r['contracted_cost'] is None)
    for n in ADDITIVE:
        expected=sum((Fraction(json.loads(r['source_json'])[versions()['1.0']['columns'][n]]) for r in rows if r[n] is not None),Fraction(0))
        observed=sum(Fraction(g['measures'][n]['coefficient'] or 0,10**18) for g in controls(rows).values())
        assert expected==observed
    assert all('list_unit_price' not in g['measures'] for g in controls(rows).values())


@pytest.mark.parametrize('change',['overlap','unknown','timezone','currency','actor','period'])
def test_config_guards(official,change):
    c=demo_config(official)
    if change=='overlap':c['tenants'][1]['assignments'].append(c['tenants'][0]['assignments'][0])
    if change=='unknown':c['tenants'][0]['extra']=True
    if change=='timezone':c['tenants'][0]['timezone']='Invalid/Zone'
    if change=='currency':c['tenants'][0]['currencies']=['usd']
    if change=='actor':c['tenants'][0]['actor']='importer'
    if change=='period':c['tenants'][0]['assignments'][0][3]='not a timestamp'
    with pytest.raises(Rejected):validate_config(c)


@pytest.mark.parametrize('version',['1.0','1.2','1.3'])
def test_mapping_dispatch_preserves_declared_version(official,version):
    mapping=versions()[version]
    reader=csv.DictReader(io.StringIO(official.decode()))
    header=list(reader.fieldnames)
    if version=='1.3':header[header.index('ProviderName')]='ServiceProviderName'
    stream=io.StringIO(newline='');writer=csv.DictWriter(stream,fieldnames=header);writer.writeheader()
    for row in reader:
        if version=='1.3':row['ServiceProviderName']=row.pop('ProviderName')
        writer.writerow(row)
    rows=decode(stream.getvalue().encode(),mapping)
    assert len(rows)==1000
    assert rows[0]['provider']=='AWS'


def test_unmapped_full_file_and_all_missing(official):
    c=demo_config(official);rows=decode(official,versions()['1.0'])
    target=canonical(domain(rows[-1]))
    for t in c['tenants']:t['assignments']=[d for d in t['assignments'] if canonical(d)!=target]
    with pytest.raises(Rejected,match='unmapped'):assign(rows,c)
    rows=decode(official,versions()['1.0'])
    rows=[r for r in rows if r['provider']=='Oracle']
    for group in controls(rows).values():
        assert group['measures']['contracted_cost']['coefficient'] is None
        assert group['measures']['contracted_cost']['present']==0


def test_duplicate_headers_overflow_and_forged_tenant(official):
    with pytest.raises(Rejected,match='source_headers'):decode(b'ProviderName,ProviderName\na,b\n',versions()['1.0'])
    reader=csv.DictReader(io.StringIO(official.decode()));header=reader.fieldnames+['tenant_id']
    output=io.StringIO(newline='');writer=csv.DictWriter(output,fieldnames=header);writer.writeheader()
    for row in reader:writer.writerow({**row,'tenant_id':'attacker'})
    rows=assign(decode(output.getvalue().encode(),versions()['1.0']),demo_config(official))
    assert {r['tenant_id'] for r in rows}=={tenant_id('demo-a'),tenant_id('demo-b')}


def test_snapshot_coverage_and_aggregate_overflow(official):
    config=demo_config(official);rows=decode(official,versions()['1.0'])
    with pytest.raises(Rejected,match='snapshot_coverage'):assign(rows[:-1],config)
    source=json.loads(rows[0]['source_json']);source['BilledCost']='60000000000000000000'
    output=io.StringIO(newline='');w=csv.DictWriter(output,fieldnames=list(source));w.writeheader();w.writerow(source);w.writerow(source)
    with pytest.raises(Rejected,match='aggregate_bound'):decode(output.getvalue().encode(),versions()['1.0'])


def test_extra_registry_version_without_runtime_branch(tmp_path,official):
    registry=versions();registry['9.9']=copy.deepcopy(registry['1.0'])
    path=tmp_path/'versions.json';path.write_text(json.dumps({'versions':registry}))
    assert len(decode(official,versions(path)['9.9']))==1000


def test_provider_qualifies_equal_account_identity(official):
    rows=decode(official,versions()['1.0'])[:2]
    rows[1]['provider']='Microsoft'
    for k in ['account','subaccount','billing_start','billing_end']:rows[1][k]=rows[0][k]
    config=demo_config(official)
    for t,r in zip(config['tenants'],rows):t['assignments']=[domain(r)]
    config['snapshot'].update(row_count=2,domains_digest=domains_digest(rows))
    assigned=assign(rows,config)
    assert assigned[0]['tenant_id']!=assigned[1]['tenant_id']
    config['tenants'][0]['currencies']=['KRW']
    with pytest.raises(Rejected,match='denied'):assign(rows,config)
