import json

from .model import Rejected, canonical, digest, versions


PROFILE = 'marketplace-source-evidence-sums-v1'
FIELDS = {'publisher': 'PublisherName', 'issuer': 'InvoiceIssuerName',
          'description': 'ChargeDescription', 'category': 'ChargeCategory',
          'charge_class': 'ChargeClass', 'frequency': 'ChargeFrequency',
          'sku': 'SkuId', 'commitment': 'CommitmentDiscountId'}


class Mapping:
    def __init__(self):
        self.registry = versions()
        self.digest = digest(canonical({'profile': PROFILE, 'registry': self.registry,
                                        'mapped_versions': ['1.0'], 'fields': FIELDS,
                                        'candidate': 'mapped-present-issuer-publisher-exact-inequality'}).encode())

    def fields(self, row, version):
        raw = row['source_json']
        try:
            raw = json.loads(raw) if isinstance(raw, str) else raw
        except (ValueError, TypeError):
            raise Rejected('marketplace_source_shape') from None
        if not isinstance(raw, dict):
            raise Rejected('marketplace_source_shape')
        result = {}
        for name, source in FIELDS.items():
            if version != '1.0':
                value = raw.get(source)
                if isinstance(value, str) and len(value) > 4096:
                    raise Rejected('marketplace_field_bound')
                state, value = 'unmapped_version', value if isinstance(value, str) else None
            elif source not in raw:
                state, value = 'absent', None
            else:
                value = raw[source]
                if value is None:
                    state, value = 'source_null', None
                elif not isinstance(value, str):
                    state, value = 'invalid_type', None
                elif len(value) > 4096:
                    raise Rejected('marketplace_field_bound')
                elif value in self.registry[version]['null_tokens']:
                    state, value = 'source_null', None
                elif not value.strip():
                    state, value = 'blank', None
                else:
                    state = 'value'
            result[name] = {'state': state, 'value': value}
        return result


def classify(row):
    publisher, issuer = (row['fields'][k] for k in ('publisher', 'issuer'))
    if row['version'] != '1.0' or publisher['state'] != 'value' or issuer['state'] != 'value':
        return 'unknown'
    return 'candidate' if publisher['value'] != issuer['value'] else 'no_difference'
