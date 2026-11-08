import unittest

from oracle import coefficient, groups


class OracleChecks(unittest.TestCase):
    def test_decimal_trailing_zeros_and_full_precision(self):
        self.assertEqual(coefficient('0.100000000000000000'), 100000000000000000)
        self.assertEqual(coefficient('99999999999999999999.999999999999999999'), 10**38 - 1)
        self.assertEqual(coefficient('-1e-18'), -1)
        self.assertIsNone(coefficient(None))
        with self.assertRaises(AssertionError):
            coefficient('1e-19')

    def test_null_zero_currency_and_unit_boundaries(self):
        base = {'ProviderName': 'Synthetic', 'ServiceName': 'Synthetic service', 'BillingCurrency': 'USD',
            'ConsumedUnit': 'Hours', 'PricingUnit': 'Hours', 'BilledCost': '', 'EffectiveCost': '',
            'ListCost': '', 'ContractedCost': '', 'ConsumedQuantity': '', 'PricingQuantity': ''}
        empty = next(iter(groups([base]).values()))['measures']['billed_cost']
        zero = next(iter(groups([{**base, 'BilledCost': '0'}]).values()))['measures']['billed_cost']
        self.assertEqual(empty, {'coefficient': None, 'present': 0, 'missing': 1})
        self.assertEqual(zero, {'coefficient': 0, 'present': 1, 'missing': 0})
        separated = groups([base, {**base, 'BillingCurrency': 'CNY'}, {**base, 'ConsumedUnit': 'GB'}])
        self.assertEqual(len(separated), 3)


if __name__ == '__main__':
    unittest.main()
