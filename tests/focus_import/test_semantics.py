import copy
import csv
import io
import json
import random
import sys
from decimal import localcontext
from fractions import Fraction

import pytest

from modules.focus_import import semantics
from modules.focus_import.model import ROOT, Rejected, canonical, decode, digest, versions
from modules.focus_local.contract import key_value, load_registry, parse

sys.path.insert(0, str(ROOT / 'tests/local_focus'))
from test_contract import CAP, fixture


def rows(version='1.3', all_rows=False, **updates):
    return [{'ordinal': n, 'source_json': canonical(row)} for n, row in enumerate(
        csv.DictReader(io.StringIO(fixture(version, all_rows=all_rows, **updates).decode())), 1)]


@pytest.mark.parametrize('version', ['1.2', '1.3'])
def test_consumed_profile_validates_without_changing_core_rows(version):
    normalized = decode(fixture(version), versions()[version])
    before = copy.deepcopy(normalized)
    result = semantics.validate_rows(normalized, version)
    assert normalized == before
    assert result['profile']['digest'] == digest(canonical(result['profile']['definition']).encode())
    assert result['contexts'][0]['fields']['billed_cost'] == '0.300001000000000000'
    assert result['contexts'][0]['fields']['service_category'] == 'Compute'
    assert result['contexts'][0]['fields']['tags'] == '{}'
    assert result['contexts'][0]['fields']['host_provider_name'] == (None if version == '1.2' else 'SyntheticHost')
    if version == '1.2':
        assert result['contexts'][0]['missing_reasons']['host_provider_name'] == 'absent_by_version'


@pytest.mark.parametrize('version', ['1.2', '1.3'])
@pytest.mark.parametrize('updates,reason', [
    ({'BilledCost': ''}, 'semantic_required_value'),
    ({'ListUnitPrice': '-1'}, 'semantic_negative_unit_price'),
    ({'ListUnitPrice': '999'}, 'semantic_price_product'),
    ({'ConsumedUnit': ''}, 'semantic_required_value'),
    ({'ChargeCategory': 'usage'}, 'semantic_enum_value'),
    ({'ServiceCategory': 'not-a-category'}, 'semantic_enum_value'),
    ({'BilledCost': '1E+2'}, 'semantic_decimal_syntax'),
    ({'ChargePeriodStart': '2026-10-01 00:00:00'}, 'semantic_timestamp_format'),
])
def test_semantic_failures(version, updates, reason):
    with pytest.raises(Rejected, match=reason):
        semantics.validate_rows(rows(version, **updates), version)


@pytest.mark.parametrize('updates', [{'SkuPriceId': ''}, {'HostProviderName': ''}])
def test_version13_applicability_requires_usable_sku_and_host(updates):
    with pytest.raises(Rejected, match='semantic_(required|inapplicable)_value'):
        semantics.validate_rows(rows(**updates), '1.3')


def test_version12_correction_product_is_not_version13_rule():
    changed = {'ChargeClass': 'Correction', 'ListCost': '999'}
    assert semantics.validate_rows(rows('1.2', **changed), '1.2')['contexts']
    with pytest.raises(Rejected, match='semantic_price_product'):
        semantics.validate_rows(rows('1.3', **changed), '1.3')


@pytest.mark.parametrize('version', ['1.2', '1.3'])
@pytest.mark.parametrize('updates', [
    {'ChargeCategory': 'Tax', 'SkuPriceId': '', 'ConsumedQuantity': '', 'ConsumedUnit': '', 'PricingQuantity': '', 'PricingUnit': '', 'ListUnitPrice': '', 'ContractedUnitPrice': ''},
    {'ChargeCategory': 'Purchase', 'ConsumedQuantity': '', 'ConsumedUnit': ''},
    {'CommitmentDiscountStatus': 'Unused', 'ConsumedQuantity': '', 'ConsumedUnit': ''},
    {'ChargeClass': 'Correction', 'SkuPriceId': '', 'ConsumedQuantity': '', 'ConsumedUnit': '', 'PricingQuantity': '', 'PricingUnit': '', 'ListUnitPrice': '', 'ContractedUnitPrice': ''},
])
def test_version_applicability_positive(version, updates):
    assert len(semantics.validate_rows(rows(version, all_rows=True, **updates), version)['contexts']) == 2


@pytest.mark.parametrize('value', ['', None])
def test_null_subaccount_is_semantically_allowed(value):
    data = rows(SubAccountId='')
    if value is None:
        source = json.loads(data[0]['source_json']); source.pop('SubAccountId')
        data[0]['source_json'] = canonical(source)
    pinned = semantics.profile('1.3')
    if value is None:
        caps = pinned['definition']['capabilities']; caps['subaccounts'] = False
        pinned = semantics.profile('1.3', caps)
    result = semantics.context(data[0], '1.3', pinned)
    assert result['fields']['sub_account_id'] is None


def test_strict_scope_requires_explicit_none_capability():
    data = fixture(all_rows=True, SubAccountId='')
    with pytest.raises(Rejected, match='source_scope'):
        parse(data, load_registry()['1.3'], CAP)
    result = parse(data, load_registry()['1.3'], {**CAP, 'subaccount': None})
    assert all(row['sub_account_id'] is None for row in result)


@pytest.mark.parametrize('value', ['[]', '{"k":{}}', '{"k":[]}', '{"k":"a","k":"b"}', '{"k":NaN}', '{"k":1e99999999999999999999999999999}'])
def test_tags_reject_invalid_shape_duplicate_keys_and_nonfinite(value):
    with pytest.raises(Rejected, match='json_value'):
        key_value(value)
    with pytest.raises(Rejected, match='json_value'):
        semantics.context(rows(Tags=value)[0], '1.3')


def test_tags_preserve_exact_numeric_lexical_value_in_both_paths():
    text = '{ "weight":0.1000000000000000001,"enabled":true,"name":"a","unset":null }'
    assert key_value(text) == text
    assert semantics.context(rows(Tags=text)[0], '1.3')['fields']['tags'] == text
    assert parse(fixture(Tags=text), load_registry()['1.3'], CAP)[0]['tags'] == text


def test_pinned_profile_does_not_reinterpret_current_registry(tmp_path, monkeypatch):
    pinned = semantics.profile('1.3')
    registry = json.loads(semantics.REGISTRY.read_text())
    next(f for f in registry['versions']['1.3']['fields'] if f['name'] == 'service_category')['enum'] = ['Other']
    path = tmp_path / 'changed.json'; path.write_text(json.dumps(registry))
    monkeypatch.setattr(semantics, 'REGISTRY', path)
    with pytest.raises(Rejected, match='semantic_enum_value'):
        semantics.context(rows()[0], '1.3')
    assert semantics.context(rows()[0], '1.3', pinned)['fields']['service_category'] == 'Compute'


@pytest.mark.parametrize('change', ['digest', 'revision', 'fields', 'predicate'])
def test_profile_tampering_or_unsupported_validator_fails_closed(change):
    pinned = semantics.profile('1.3')
    if change == 'digest':
        pinned['digest'] = '0' * 64
    elif change == 'revision':
        pinned['definition']['validator_revision'] = 3
    elif change == 'fields':
        pinned['definition']['fields'].pop()
    else:
        pinned['definition']['fields'][0]['nonnull_when'] = {'unsafe': 'expression'}
    if change != 'digest':
        pinned['digest'] = digest(canonical(pinned['definition']).encode())
    with pytest.raises(Rejected):
        semantics.context(rows()[0], '1.3', pinned)


def test_registry_version_extension_uses_definition_not_version_branch(tmp_path, monkeypatch):
    registry = json.loads(semantics.REGISTRY.read_text())
    registry['versions']['9.9'] = copy.deepcopy(registry['versions']['1.3'])
    registry['versions']['9.9']['version'] = '9.9'
    path = tmp_path / 'extended.json'; path.write_text(json.dumps(registry))
    monkeypatch.setattr(semantics, 'REGISTRY', path)
    assert semantics.context(rows()[0], '9.9')['fields']['host_provider_name'] == 'SyntheticHost'


def test_capabilities_are_explicit_pinned_and_not_row_name_guesses():
    caps = {'measured_usage': False, 'list_prices': False, 'contracted_prices': False, 'host_null_services': ['SyntheticCompute']}
    pinned = semantics.profile('1.3', caps)
    changed = rows(ConsumedQuantity='', ConsumedUnit='', ListUnitPrice='', ContractedUnitPrice='', HostProviderName='')
    result = semantics.context(changed[0], '1.3', pinned)
    assert result['fields']['host_provider_name'] is None
    assert result['missing_reasons']['consumed_quantity'] == 'capability_disabled'
    with pytest.raises(Rejected, match='semantic_disabled_field'):
        semantics.context(rows()[0], '1.3', pinned)
    with pytest.raises(Rejected, match='semantic_capabilities'):
        semantics.profile('1.3', {'measured_usage': False})


def test_generated_exact_products_use_fraction_oracle_under_low_decimal_context():
    rng = random.Random(300603)
    with localcontext() as context:
        context.prec = 6
        for _ in range(100):
            price = rng.randint(1, 10**17)
            quantity = rng.randint(-1000, 1000)
            lexical_price = f'{price // 10**18}.{price % 10**18:018d}'
            amount = price * quantity
            lexical_cost = ('-' if amount < 0 else '') + f'{abs(amount) // 10**18}.{abs(amount) % 10**18:018d}'
            result = semantics.context(rows(ListUnitPrice=lexical_price, ContractedUnitPrice=lexical_price,
                PricingQuantity=str(quantity), ListCost=lexical_cost, ContractedCost=lexical_cost)[0], '1.3')
            assert Fraction(result['fields']['list_cost']) == Fraction(price, 10**18) * quantity
            assert Fraction(result['fields']['contracted_cost']) == Fraction(lexical_cost)


@pytest.mark.parametrize('version', ['1.2', '1.3'])
@pytest.mark.parametrize('suffix', ['+00:00', '+01:00', '-05:00'])
def test_consumed_source_timestamp_requires_z(version, suffix):
    changed = rows(version, ChargePeriodStart='2026-10-01T00:00:00' + suffix)
    with pytest.raises(Rejected, match='semantic_timestamp_format'):
        semantics.context(changed[0], version)
    with pytest.raises(Rejected, match='timestamp_format'):
        parse(fixture(version, ChargePeriodStart='2026-10-01T00:00:00' + suffix), load_registry()[version], CAP)


@pytest.mark.parametrize('version', ['1.2', '1.3'])
@pytest.mark.parametrize('category', ['Usage', 'Purchase'])
def test_sku_price_required_when_pricelist_construct_enabled(version, category):
    changed = rows(version, ChargeCategory=category, SkuPriceId='')
    with pytest.raises(Rejected, match='semantic_(required|inapplicable)_value'):
        semantics.context(changed[0], version)
    with pytest.raises(Rejected, match='required_value|inapplicable_value'):
        parse(fixture(version, ChargeCategory=category, SkuPriceId=''), load_registry()[version], CAP)


@pytest.mark.parametrize('version', ['1.2', '1.3'])
def test_tax_forbids_nonnull_sku_even_when_unit_price_capabilities_disabled(version):
    caps = semantics.profile(version)['definition']['capabilities']
    caps.update(measured_usage=False, list_prices=False, contracted_prices=False)
    changed = rows(version, ChargeCategory='Tax', ConsumedQuantity='', ConsumedUnit='',
        PricingQuantity='', PricingUnit='', ListUnitPrice='', ContractedUnitPrice='')
    with pytest.raises(Rejected, match='semantic_inapplicable_value'):
        semantics.context(changed[0], version, semantics.profile(version, caps))


@pytest.mark.parametrize('version', ['1.2', '1.3'])
@pytest.mark.parametrize('column, flag, name', [('Tags', 'tags', 'tags'), ('ResourceId', 'resource_ids', 'resource_id'),
    ('CommitmentDiscountStatus', 'commitment_discounts', 'commitment_discount_status'),
    ('SkuPriceId', 'sku_prices', 'sku_price_id'), ('SubAccountId', 'subaccounts', 'sub_account_id')])
def test_conditional_construct_headers_are_pinned_capabilities(version, column, flag, name):
    source_rows = rows(version, ChargeCategory='Tax', SkuPriceId='', ConsumedQuantity='', ConsumedUnit='',
        PricingQuantity='', PricingUnit='', ListUnitPrice='', ContractedUnitPrice='')
    source = json.loads(source_rows[0]['source_json']); source.pop(column)
    source_rows[0]['source_json'] = canonical(source)
    with pytest.raises(Rejected, match='semantic_missing_header'):
        semantics.context(source_rows[0], version)
    caps = semantics.profile(version)['definition']['capabilities']; caps[flag] = False
    pinned = semantics.profile(version, caps)
    context = semantics.context(source_rows[0], version, pinned)
    assert context['fields'][name] is None
    assert context['missing_reasons'][name] == 'capability_disabled'
    assert pinned['definition']['capabilities'][flag] is False
    assert pinned['digest'] == digest(canonical(pinned['definition']).encode())


def test_retained_revision_one_profile_keeps_its_own_rules():
    pinned = semantics.profile('1.2')
    definition = pinned['definition']; definition['validator_revision'] = 1
    definition['capabilities'] = {k: v for k, v in definition['capabilities'].items() if k in semantics.CAPABILITIES_V1}
    sku = next(f for f in definition['fields'] if f['name'] == 'sku_price_id')
    sku.pop('nonnull_when'); sku.pop('null_when')
    next(f for f in definition['fields'] if f['name'] == 'sub_account_id')['header'] = False
    pinned['digest'] = digest(canonical(definition).encode())
    changed = rows('1.2', SkuPriceId='', ChargePeriodStart='2026-10-01T00:00:00+00:00')
    assert semantics.context(changed[0], '1.2', pinned)['fields']['sku_price_id'] is None
    with pytest.raises(Rejected):
        semantics.context(changed[0], '1.2')
