import json

from .model import Rejected, canonical, digest, versions


PROFILE = 'commitment-source-evidence-sums-v1'
FIELDS = {'id': 'CommitmentDiscountId', 'name': 'CommitmentDiscountName',
          'type': 'CommitmentDiscountType', 'category': 'CommitmentDiscountCategory',
          'status': 'CommitmentDiscountStatus', 'charge_category': 'ChargeCategory',
          'charge_class': 'ChargeClass', 'charge_frequency': 'ChargeFrequency'}
STATES = ('source_null', 'absent', 'blank', 'unmapped_version', 'invalid_type')
STATUSES = {'Used': 'used', 'Unused': 'unused'}


class Mapping:
    def __init__(self):
        self.registry = versions()
        self.digest = digest(canonical({'profile': PROFILE, 'registry': self.registry,
                                        'mapped_versions': ['1.0'], 'fields': FIELDS,
                                        'statuses': STATUSES}).encode())

    def fields(self, row, version):
        raw = row['source_json']
        try:
            raw = json.loads(raw) if isinstance(raw, str) else raw
        except (ValueError, TypeError):
            raise Rejected('commitment_source_shape') from None
        if not isinstance(raw, dict):
            raise Rejected('commitment_source_shape')
        result = {}
        for name, source in FIELDS.items():
            if version != '1.0':
                value = raw.get(source)
                if isinstance(value, str) and len(value) > 4096:
                    raise Rejected('commitment_field_bound')
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
                    raise Rejected('commitment_field_bound')
                elif value in self.registry[version]['null_tokens']:
                    state, value = 'source_null', None
                elif not value.strip():
                    state, value = 'blank', None
                else:
                    state = 'value'
            result[name] = {'state': state, 'value': value}
        return result


def metadata(rows, field):
    values = sorted({r['fields'][field]['value'] for r in rows if r['fields'][field]['state'] == 'value'})
    return {'values': values[:16], 'distinct': len(values), 'truncated': len(values) > 16}
