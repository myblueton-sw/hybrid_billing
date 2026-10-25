import csv
import io
from datetime import datetime, timezone

from modules.focus_local.contract import Rejected


MAX_BYTES = 16 * 1024 * 1024
MAX_ROWS = 10000
MAX_COLUMNS = 256
MAX_CELL = 65536


def input_format(data):
    if not isinstance(data, bytes) or len(data) > MAX_BYTES:
        raise Rejected('input_size')
    return 'parquet' if data.startswith(b'PAR1') else 'csv'


def headers(names, mapping):
    if len(set(names)) != len(names) or not set(mapping.get('required_columns', mapping['columns'].values())) <= set(names):
        raise Rejected('source_headers')


def csv_sources(data, mapping):
    reader = csv.DictReader(io.StringIO(data.decode('utf-8'), newline=''), strict=True)
    headers(reader.fieldnames or [], mapping)
    for ordinal, source in enumerate(reader, 1):
        if ordinal > MAX_ROWS or None in source or any(v is None or len(v) > MAX_CELL for v in source.values()):
            raise Rejected('source_shape')
        yield source


def parquet_sources(data, mapping):
    import pyarrow as pa
    import pyarrow.parquet as pq

    if len(data) < 12 or not data.endswith(b'PAR1'):
        raise Rejected('parquet_format')
    try:
        file = pq.ParquetFile(io.BytesIO(data), thrift_string_size_limit=MAX_BYTES,
            thrift_container_size_limit=100000, arrow_extensions_enabled=False)
        with file:
            meta = file.metadata
            if meta.num_rows > MAX_ROWS or not 1 <= meta.num_columns <= MAX_COLUMNS or meta.num_row_groups > MAX_ROWS + 1:
                raise Rejected('source_shape')
            sizes = [meta.row_group(i).column(j).total_uncompressed_size
                for i in range(meta.num_row_groups) for j in range(meta.num_columns)]
            if any(size < 0 for size in sizes) or sum(sizes) > MAX_BYTES:
                raise Rejected('parquet_expansion')
            schema = file.schema_arrow
            headers(schema.names, mapping)
            numeric = {mapping['columns'][n] for n in mapping['columns'] if n.endswith('_cost') or n.endswith('_quantity') or n.endswith('_price')}
            times = {mapping['columns'][n] for n in ['billing_start', 'billing_end', 'charge_start', 'charge_end']}
            strings = set(mapping['columns'].values()) - numeric - times
            for field in schema:
                t = field.type
                text = pa.types.is_string(t) or pa.types.is_large_string(t)
                exact = pa.types.is_integer(t) or pa.types.is_decimal(t)
                timestamp = pa.types.is_timestamp(t) and t.unit in ['s', 'ms', 'us']
                if not (text or exact or timestamp or pa.types.is_boolean(t) or pa.types.is_null(t)):
                    raise Rejected('parquet_type')
                if field.name in strings and not (text or pa.types.is_null(t)):
                    raise Rejected('parquet_type')
                if field.name in numeric and not (text or exact or pa.types.is_null(t)):
                    raise Rejected('parquet_type')
                if field.name in times and not (text or timestamp or pa.types.is_null(t)):
                    raise Rejected('parquet_type')
            decoded = 0
            for batch in file.iter_batches(batch_size=1, use_threads=False):
                for record in batch.to_pylist():
                    source = {}
                    for name, value in record.items():
                        if isinstance(value, datetime):
                            value = (value if value.tzinfo else value.replace(tzinfo=timezone.utc)).astimezone(timezone.utc).isoformat().replace('+00:00','Z')
                        elif isinstance(value, bool):
                            value = 'true' if value else 'false'
                        elif value is not None:
                            value = str(value)
                        length = len(value.encode('utf-8')) if value is not None else 0
                        if length > MAX_CELL:
                            raise Rejected('source_shape')
                        decoded += length
                        if decoded > MAX_BYTES:
                            raise Rejected('parquet_expansion')
                        source[name] = value
                    yield source
    except (pa.ArrowException, OSError, ValueError, OverflowError):
        raise Rejected('parquet_format') from None


def sources(data, mapping):
    return parquet_sources(data, mapping) if input_format(data) == 'parquet' else csv_sources(data, mapping)
