import copy
import csv
import io
import json
import random
from fractions import Fraction

import pytest

from modules.focus_local.contract import REGISTRY, ROOT, Rejected, coefficient, controls, decimal_value, load_registry, parse
from modules.focus_local.database import PERIOD_END, PERIOD_START


CAP = {'account':'synthetic-account-a','subaccount':'synthetic-sub-a','start':PERIOD_START,'end':PERIOD_END}


def fixture(version='1.3', all_rows=False, **updates):
    data = (ROOT / f'tests/fixtures/focus/synthetic-{version}.csv').read_text()
    source = csv.DictReader(io.StringIO(data))
    headers = source.fieldnames
    rows = list(source)
    for row in (rows if all_rows else rows[:1]):row.update(updates)
    output = io.StringIO(newline='')
    writer = csv.DictWriter(output,fieldnames=headers)
    writer.writeheader()
    writer.writerows(rows)
    return output.getvalue().encode()


@pytest.mark.parametrize('version',['1.2','1.3'])
def test_supported_version_parse_and_exact_control(version):
    contract = load_registry()[version]
    rows = parse(fixture(version),contract,CAP)
    group = next(iter(controls(rows,contract.fields).values()))
    assert group['measures']['billed_cost'] == {'coefficient':200000,'missing':0,'present':2}
    assert 'list_unit_price' not in group['measures']
    assert rows[0]['service_provider_name'] == 'SyntheticProvider'
    assert rows[0]['host_provider_name'] == (None if version=='1.2' else 'SyntheticHost')


@pytest.mark.parametrize('text',['NaN','Infinity','1e999','1.0000001','1000000000000','1e-7',' 1','1,1','+1'])
def test_invalid_numeric_profile(text):
    with pytest.raises(Rejected):
        decimal_value(text)


def test_generated_numbers_match_fraction_oracle():
    rng = random.Random(27001)
    total = 0
    oracle = Fraction(0)
    for _ in range(2000):
        value = rng.randint(-10**17,10**17)
        text = f'{"-" if value<0 else ""}{abs(value)//10**6}.{abs(value)%10**6:06d}'
        actual = decimal_value(text)
        assert Fraction(actual) == Fraction(value,10**6)
        total += coefficient(actual)
        oracle += Fraction(text)
    assert Fraction(total,10**6) == oracle
    assert decimal_value('1e-3') == decimal_value('0.001')


@pytest.mark.parametrize('updates,reason',[
    ({'BilledCost':''},'required_value'),
    ({'BillingAccountId':'other-account','BilledCost':''},'source_scope'),
    ({'BillingPeriodEnd':'2026-12-01T00:00:00Z'},'source_period'),
    ({'ConsumedQuantity':''},'required_value'),
    ({'SkuPriceId':''},'inapplicable_value'),
    ({'HostProviderName':''},'required_value'),
    ({'ListUnitPrice':'-1'},'negative_unit_price'),
    ({'ListUnitPrice':'0.2'},'price_product'),
    ({'ChargeCategory':'usage'},'enum_value'),
])
def test_consumed_field_rules(updates,reason):
    with pytest.raises(Rejected,match=reason):
        parse(fixture(**updates),load_registry()['1.3'],CAP)


def test_missing_sum_is_null_and_units_are_separate():
    contract=load_registry()['1.3']
    rows=parse(fixture(),contract,CAP)
    rows[0]['consumed_quantity']=None
    rows[1]['consumed_quantity']=None
    rows[1]['pricing_unit']='Requests'
    groups=controls(rows,contract.fields)
    assert len(groups)==2
    assert all(g['measures']['consumed_quantity']['coefficient'] is None for g in groups.values())


@pytest.mark.parametrize('broken', ['unit','timestamp','predicate','duplicate','price_aggregation'])
def test_registry_rejects_invalid_contract_before_io(tmp_path,broken):
    entry=json.loads((REGISTRY/'1.3.json').read_text())
    if broken=='unit':
        entry['fields']=[f for f in entry['fields'] if f['name']!='consumed_unit']
    elif broken=='timestamp':
        next(f for f in entry['fields'] if f['name']=='charge_period_start')['type']='string'
    elif broken=='predicate':
        entry['fields'][0]['nonnull_when']={'eval':'unsafe'}
    elif broken=='duplicate':
        entry['fields'].append(copy.deepcopy(entry['fields'][0]))
    else:
        next(f for f in entry['fields'] if f['name']=='list_unit_price')['aggregate']='invalid'
    (tmp_path/'entry.json').write_text(json.dumps(entry))
    with pytest.raises(Rejected):load_registry(tmp_path)


def test_registry_extension_without_version_branch(tmp_path):
    entry=json.loads((REGISTRY/'1.3.json').read_text())
    entry['version']='9.9'
    (tmp_path/'new.json').write_text(json.dumps(entry))
    contract=load_registry(tmp_path)['9.9']
    assert len(parse(fixture(),contract,CAP))==2


@pytest.mark.parametrize('version',['1.2','1.3'])
@pytest.mark.parametrize('updates',[
    {'ChargeCategory':'Tax','SkuPriceId':'','ConsumedQuantity':'','ConsumedUnit':'','PricingQuantity':'','PricingUnit':'','ListUnitPrice':'','ContractedUnitPrice':''},
    {'ChargeCategory':'Purchase','ConsumedQuantity':'','ConsumedUnit':''},
    {'CommitmentDiscountStatus':'Unused','ConsumedQuantity':'','ConsumedUnit':''},
    {'ChargeClass':'Correction','SkuPriceId':'','ConsumedQuantity':'','ConsumedUnit':'','PricingQuantity':'','PricingUnit':'','ListUnitPrice':'','ContractedUnitPrice':''},
])
def test_positive_version_applicability(version,updates):
    assert len(parse(fixture(version,all_rows=True,**updates),load_registry()[version],CAP))==2
