import argparse
import csv
import io
import json
import sys
from pathlib import Path

from oracle import ROOT, sha

sys.path.insert(0, str(ROOT))

from modules.focus_import import semantics
from modules.focus_import.model import Rejected, decode, versions


PROVIDERS = ['AWS', 'Microsoft', 'Google Cloud', 'Oracle', 'Alibaba Cloud', 'VMware']


def fixture(version, provider):
    with (ROOT / f'tests/fixtures/focus/synthetic-{version}.csv').open(newline='') as stream:
        row = next(csv.DictReader(stream))
    row.update(ProviderName=provider, ServiceName='Synthetic metered compute', PublisherName='Synthetic Publisher',
        InvoiceIssuerName='Synthetic Invoice Issuer')
    if version == '1.3':
        row.update(ServiceProviderName=provider, ProviderName='Synthetic Legacy Provider', HostProviderName='Synthetic Host')
    return row


def encoded(row):
    stream = io.StringIO(newline='')
    writer = csv.DictWriter(stream, fieldnames=list(row))
    writer.writeheader()
    writer.writerow(row)
    return stream.getvalue().encode()


def rejection(data, version):
    try:
        rows = decode(data, versions()[version], conditional=True)
        semantics.validate_rows(rows, version)
    except Rejected as error:
        return str(error)
    raise AssertionError('Expected rejection')


def main():
    import pyarrow as pa
    import pyarrow.parquet as pq

    parser = argparse.ArgumentParser()
    parser.add_argument('--output', type=Path, required=True)
    arguments = parser.parse_args()
    cases = []
    for version in ['1.2', '1.3']:
        for provider in PROVIDERS:
            rows = decode(encoded(fixture(version, provider)), versions()[version], conditional=True)
            context = semantics.validate_rows(rows, version)['contexts'][0]['fields']
            assert rows[0]['provider'] == provider
            assert context['billed_cost'] == '0.300001000000000000'
            assert context['invoice_issuer_name'] == 'Synthetic Invoice Issuer'
            if version == '1.3':
                assert context['service_provider_name'] == provider
                assert context['host_provider_name'] == 'Synthetic Host'
            else:
                assert context['provider_name_legacy'] == provider
                assert context['publisher_name_legacy'] == 'Synthetic Publisher'
                assert context['host_provider_name'] is None
            cases.append({'name': f'profile:{version}:{provider}', 'status': 'PASS', 'source': 'synthetic-contract-fixture'})
    for version in ['1.2', '1.3']:
        row = fixture(version, 'Google Cloud')
        row['ChargeCategory'] = 'usage'
        assert rejection(encoded(row), version) == 'semantic_enum_value'
        cases.append({'name': f'lowercase_charge_category_not_relabelled:{version}', 'status': 'PASS',
            'source': 'synthetic reproduction of observed sample lowercase value', 'outcome': 'semantic_enum_value'})
    row = fixture('1.2', 'AWS')
    table = pa.table({key: pa.array([float(value)], type=pa.float64()) if key == 'BilledCost' else pa.array([value]) for key, value in row.items()})
    output = io.BytesIO()
    pq.write_table(table, output)
    assert rejection(output.getvalue(), '1.2') == 'parquet_type'
    cases.append({'name': 'supplier_style_double_parquet', 'status': 'PASS', 'outcome': 'parquet_type',
        'source': 'synthetic representative physical schema, not an actual AWS export',
        'compatibility_gap': 'Documented supplier DOUBLE columns require an explicitly reviewed exact-value adapter; native acceptance is not verified.'})
    row = fixture('1.2', 'Google Cloud')
    table = pa.table({key: [value] for key, value in row.items()}).append_column('x_Credits', pa.array([[{'amount': '1.0'}]]))
    output = io.BytesIO()
    pq.write_table(table, output)
    assert rejection(output.getvalue(), '1.2') == 'parquet_type'
    cases.append({'name': 'supplier_style_nested_extension', 'status': 'PASS', 'outcome': 'parquet_type',
        'source': 'synthetic representative nested extension, not an actual GCP export',
        'compatibility_gap': 'Nested BigQuery extensions need an explicit transport/flattening contract; no native BigQuery adapter is certified.'})
    arguments.output.parent.mkdir(parents=True, exist_ok=True)
    summary = {'status': 'PASS', 'cases': cases, 'case_count': len(cases), 'runner_sha256': sha(Path(__file__).read_bytes()),
        'scope': 'Pure consumed-profile and physical-type probes; not provider conformance or storage/query integration',
        'pyarrow_version': pa.__version__}
    arguments.output.write_text(json.dumps(summary, sort_keys=True, indent=2) + '\n')
    print(json.dumps({'status': 'PASS', 'cases': len(cases)}))


if __name__ == '__main__':
    main()
