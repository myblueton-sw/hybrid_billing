import io
import json
from decimal import Decimal

import pyarrow as pa
import pyarrow.parquet as pq
import pytest

from modules.focus_import import intake
from modules.focus_import.model import NUMERIC, TIMES, Rejected, controls, decode, versions
from test_temporal import encoded, source_row


def parquet(rows, version='1.0', typed=True, extras=None):
    mapping = versions()[version]['columns']
    fields = []
    columns = []
    for name, column in mapping.items():
        values = [r[name] for r in rows]
        if typed and name in NUMERIC:
            values = [Decimal(v) if v else None for v in values]
            t = pa.decimal128(38, 18)
        elif typed and name in TIMES:
            from modules.focus_import.model import instant
            values = [instant(v) for v in values]
            t = pa.timestamp('us', tz='UTC')
        else:
            t = pa.string()
        fields.append(pa.field(column, t)); columns.append(pa.array(values, type=t))
    for name, values in (extras or {}).items():
        array = pa.array(values); fields.append(pa.field(name, array.type)); columns.append(array)
    return table_bytes(pa.Table.from_arrays(columns, schema=pa.schema(fields)))


def table_bytes(table):
    output = io.BytesIO(); pq.write_table(table, output, compression='zstd')
    return output.getvalue()


@pytest.mark.parametrize('version', ['1.0', '1.2', '1.3'])
def test_cross_format_exact_projection(version):
    source = [source_row(cost='10.000000000000000001'), source_row(cost='-0.000000000000000001'), source_row(cost='0')]
    source.append({**source_row(), 'billed_cost': '', 'effective_cost': ''})
    csv_rows = decode(encoded(source, version), versions()[version])
    data = parquet(source, version, extras={'SyntheticExtra': ['x', 'y', 'z', None]})
    rows = decode(data, versions()[version])
    names = list(versions()[version]['columns'])
    assert [[r[n] for n in names] for r in rows] == [[r[n] for n in names] for r in csv_rows]
    assert controls(rows) == controls(csv_rows)
    assert rows[0]['billed_cost'] == Decimal('10.000000000000000001')
    assert json.loads(rows[3]['missing_reasons'])['billed_cost'] == 'typed_null'
    assert json.loads(rows[3]['source_json'])['BilledCost'] is None
    assert json.loads(rows[0]['source_json'])['SyntheticExtra'] == 'x'
    assert intake.input_format(data) == 'parquet'


@pytest.mark.parametrize('version', ['1.0', '1.2', '1.3'])
def test_lexical_parquet_matches_csv(version):
    source = [source_row(), {**source_row(), 'billed_cost': 'NULL'}]
    assert decode(parquet(source, version, typed=False), versions()[version]) == decode(encoded(source, version), versions()[version])


@pytest.mark.parametrize('field,values', [('BilledCost', [0.1]), ('SyntheticExtra', [[1]]), ('SyntheticExtra', [b'bytes']), ('BillingAccountId', [42])])
def test_unsafe_or_scope_types_rejected(field, values):
    table = pq.ParquetFile(io.BytesIO(parquet([source_row()]))).read()
    column = pa.array(values)
    if field in table.column_names: table = table.set_column(table.column_names.index(field), field, column)
    else: table = table.append_column(field, column)
    with pytest.raises(Rejected, match='parquet_type'): decode(table_bytes(table), versions()['1.0'])


def test_missing_duplicate_wrong_version_headers():
    data = parquet([source_row()], '1.3')
    with pytest.raises(Rejected, match='source_headers'): decode(data, versions()['1.2'])
    table = pq.ParquetFile(io.BytesIO(data)).read()
    with pytest.raises(Rejected, match='source_headers'): decode(table_bytes(table.drop(['BillingAccountId'])), versions()['1.3'])
    with pytest.raises(Rejected, match='source_headers'): decode(table_bytes(table.append_column('ServiceName', pa.array(['x']))), versions()['1.3'])


@pytest.mark.parametrize('data', [b'PAR1broken', b'PAR1xxxxPAR1', b'PAR1'])
def test_bad_parquet_is_sanitized(data):
    with pytest.raises(Rejected, match='parquet_format'): decode(data, versions()['1.0'])


def test_parquet_size_bounds(monkeypatch):
    data = parquet([source_row()])
    monkeypatch.setattr(intake, 'MAX_ROWS', 0)
    with pytest.raises(Rejected, match='source_shape'): decode(data, versions()['1.0'])
    monkeypatch.setattr(intake, 'MAX_ROWS', 10000)
    monkeypatch.setattr(intake, 'MAX_CELL', 2)
    with pytest.raises(Rejected, match='source_shape'): decode(data, versions()['1.0'])
    monkeypatch.setattr(intake, 'MAX_CELL', 65536)
    oversized = parquet([source_row()], extras={'SyntheticExtra': ['x' * 20000]})
    assert len(oversized) < 16000
    monkeypatch.setattr(intake, 'MAX_BYTES', 16000)
    with pytest.raises(Rejected, match='parquet_expansion'): decode(oversized, versions()['1.0'])


def test_nanosecond_timestamp_rejected():
    table = pq.ParquetFile(io.BytesIO(parquet([source_row()]))).read()
    name = 'ChargePeriodStart'
    column = pa.array([1725148800000000001], type=pa.timestamp('ns', tz='UTC'))
    table = table.set_column(table.column_names.index(name), name, column)
    with pytest.raises(Rejected, match='parquet_type'): decode(table_bytes(table), versions()['1.0'])


def test_empty_parquet_retains_headers():
    with pytest.raises(Rejected, match='empty_input'): decode(parquet([]), versions()['1.0'])
