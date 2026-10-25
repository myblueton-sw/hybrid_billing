import io
from datetime import timezone
from uuid import UUID

import pyarrow as pa
import pyarrow.parquet as pq

from .model import COLUMNS, NUMERIC, STRINGS, TIMES, Rejected, controls, digest, row_digest
from .warehouse import BUCKET


SCHEMA=pa.schema([pa.field('tenant_id',pa.string(),nullable=False),pa.field('ordinal',pa.uint64(),nullable=False)]+[
    pa.field(n,pa.timestamp('us',tz='UTC') if n in TIMES else pa.decimal128(38,18) if n in NUMERIC else pa.string(),nullable=n not in TIMES) for n in COLUMNS])


def encode(rows):
    records=[{**r,'tenant_id':str(r['tenant_id'])} for r in rows]
    table=pa.Table.from_pylist(records,schema=SCHEMA)
    output=io.BytesIO();pq.write_table(table,output,compression='zstd')
    return output.getvalue()


def decode(data,expected):
    table=pq.read_table(io.BytesIO(data))
    if table.schema!=SCHEMA:raise Rejected('archive_schema')
    rows=table.to_pylist()
    for r in rows:
        r['tenant_id']=UUID(r['tenant_id'])
        for n in TIMES:r[n]=r[n].astimezone(timezone.utc)
    if len(rows)!=expected['row_count'] or row_digest(rows)!=expected['row_digest'] or controls(rows)!=expected['controls']:raise Rejected('archive_controls')
    return rows


def put(s3,key,data):
    try:s3.put_object(Bucket=BUCKET,Key=key,Body=data,IfNoneMatch='*',Metadata={'sha256':digest(data)})
    except Exception as exc:
        if getattr(exc,'response',{}).get('Error',{}).get('Code') not in ['PreconditionFailed','412']:raise
    if s3.get_object(Bucket=BUCKET,Key=key)['Body'].read()!=data:raise Rejected('archive_object_digest')


def read(s3,item):
    data=s3.get_object(Bucket=BUCKET,Key=item['archive']['key'])['Body'].read()
    if digest(data)!=item['archive']['digest']:raise Rejected('archive_digest')
    return decode(data,item)


def verify_rows(rows,item):
    if len(rows)!=item['row_count'] or row_digest(rows)!=item['row_digest'] or controls(rows)!=item['controls']:raise Rejected('archive_controls')
    return rows
