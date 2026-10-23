import io
import re
from datetime import timezone
from uuid import UUID

import boto3
import clickhouse_connect
import pyarrow as pa
import pyarrow.parquet as pq
from botocore.config import Config

from .contract import Contract, MAX_ROWS, Rejected, canonical, coefficient, controls, digest, rows_digest
from .database import password, persona


BUCKET = 'hb-local-focus'
TYPES = {'string': 'Nullable(String)', 'json': 'Nullable(String)',
         'decimal': 'Nullable(Decimal(18,6))', 'timestamp': "Nullable(DateTime64(6,'UTC'))"}
ARROW_TYPES = {'string': pa.string(), 'json': pa.string(),
               'decimal': pa.decimal128(18, 6), 'timestamp': pa.timestamp('us', tz='UTC')}
LOCKED_SETTINGS = 'join_use_nulls=1 CONST, input_format_null_as_default=0 CONST, input_format_csv_empty_as_default=1 CONST, input_format_parquet_allow_missing_columns=0 CONST, serialize_query_plan=0 CONST, async_insert=0 CONST'


def client(env, user='hb_local_writer'):
    secret = env['CH_ADMIN_PASSWORD'] if user == 'probe_admin' else password(env, user, 'CH_LOADER_PASSWORD' if user == 'hb_local_writer' else 'CH_READER_PASSWORD')
    return clickhouse_connect.get_client(host='127.0.0.1', port=18123,
        username=user, password=secret, autogenerate_session_id=False,
        send_receive_timeout=20, connect_timeout=5)


def object_client(env):
    return boto3.client('s3', endpoint_url='http://127.0.0.1:18333',
        aws_access_key_id=env['S3_ACCESS_KEY'], aws_secret_access_key=env['S3_SECRET_KEY'],
        region_name='us-east-1', config=Config(s3={'addressing_style': 'path'},
            retries={'max_attempts': 0}, connect_timeout=5, read_timeout=20))


def initialize(env, registry):
    admin = client(env, 'probe_admin')
    try:
        admin.command('CREATE DATABASE IF NOT EXISTS hb_local')
        admin.command('CREATE SETTINGS PROFILE IF NOT EXISTS hb_local_locked SETTINGS ' + LOCKED_SETTINGS)
        for user in ['hb_local_writer', 'hb_local_reader_a', 'hb_local_reader_b']:
            secret = password(env, user, 'CH_LOADER_PASSWORD' if user == 'hb_local_writer' else 'CH_READER_PASSWORD')
            verifier = __import__('hashlib').sha256(secret.encode()).hexdigest()
            admin.command(f"CREATE USER IF NOT EXISTS {user} IDENTIFIED WITH sha256_hash BY '{verifier}' SETTINGS PROFILE 'hb_local_locked'")
        for contract in registry.values():
            columns = ','.join(f"`{f['name']}` {TYPES[f['type']]}" for f in contract.fields)
            table = contract.table
            admin.command(f'CREATE TABLE IF NOT EXISTS hb_local.{table} (installation_id UUID,workspace_id UUID,release_id UUID,attempt UInt64,ordinal UInt64,{columns}) ENGINE=MergeTree PARTITION BY toYYYYMM(charge_period_start) ORDER BY (installation_id,workspace_id,release_id,attempt,ordinal) SETTINGS allow_nullable_key=1')
            admin.command(f'GRANT SELECT, INSERT ON hb_local.{table} TO hb_local_writer')
            extra = admin.query(f"SELECT count() FROM system.row_policies WHERE database='hb_local' AND table='{table}' AND short_name NOT IN ('coordinator','reader_a','reader_b')").result_rows[0][0]
            if extra:
                raise Rejected('incompatible_ch_policy')
            admin.command(f'CREATE ROW POLICY IF NOT EXISTS coordinator ON hb_local.{table} FOR SELECT USING 1 TO hb_local_writer')
            admin.command(f'ALTER ROW POLICY coordinator ON hb_local.{table} FOR SELECT USING 1 TO hb_local_writer')
            for tenant in ('a', 'b'):
                installation, workspace, _ = persona('reader-' + tenant)
                user = 'hb_local_reader_' + tenant
                admin.command(f'GRANT SELECT ON hb_local.{table} TO {user}')
                admin.command(f"CREATE ROW POLICY IF NOT EXISTS reader_{tenant} ON hb_local.{table} FOR SELECT USING installation_id=toUUID('{installation}') AND workspace_id=toUUID('{workspace}') TO {user}")
                admin.command(f"ALTER ROW POLICY reader_{tenant} ON hb_local.{table} FOR SELECT USING installation_id=toUUID('{installation}') AND workspace_id=toUUID('{workspace}') TO {user}")
        for user in ['hb_local_writer', 'hb_local_reader_a', 'hb_local_reader_b']:
            for row in admin.query('SHOW GRANTS FOR ' + user).result_rows:
                match = re.fullmatch(r'GRANT ([A-Z, ]+) ON hb_local\.(facts_[a-f0-9]{20}) TO ' + user, row[0].replace('`',''))
                if not match or not set(match[1].split(', ')) <= ({'SELECT','INSERT'} if user == 'hb_local_writer' else {'SELECT'}):
                    raise Rejected('incompatible_ch_grants')
            effective = client(env,user)
            try:
                required = {'join_use_nulls':'1','input_format_null_as_default':'0','input_format_csv_empty_as_default':'1','input_format_parquet_allow_missing_columns':'0','serialize_query_plan':'0','async_insert':'0'}
                values = effective.query("SELECT name,value,readonly FROM system.settings WHERE name IN (" + ','.join("'"+name+"'" for name in required) + ')').result_rows
                if {name:value for name,value,_ in values} != required or not all(locked for _,_,locked in values):
                    raise Rejected('incompatible_ch_settings')
            finally:
                effective.close()
    finally:
        admin.close()
    s3 = object_client(env)
    try:
        s3.create_bucket(Bucket=BUCKET)
    except s3.exceptions.BucketAlreadyOwnedByYou:
        pass


def pinned_contract(release):
    entry = release['manifest']['registry']
    sha = digest(canonical(entry).encode())
    shape = sorted((f['name'], f['type']) for f in entry['fields'])
    table = 'facts_' + digest(canonical(shape).encode())[:20]
    if sha != release['registry_digest'] or table != release['manifest']['table'] or not re.fullmatch('facts_[a-f0-9]{20}', table):
        raise Rejected('manifest_registry')
    return Contract(entry, sha, table)


class Storage:
    def __init__(self, env):
        self.env = env
        self.s3 = object_client(env)

    def put(self, key, data):
        self.s3.put_object(Bucket=BUCKET, Key=key, Body=data, IfNoneMatch='*', Metadata={'sha256': digest(data)})
        observed = self.get(key)
        if digest(observed) != digest(data):
            raise Rejected('object_digest')

    def get(self, key):
        return self.s3.get_object(Bucket=BUCKET, Key=key)['Body'].read()

    def insert(self, scope, release_id, contract, rows):
        names = [f['name'] for f in contract.fields]
        values = [[scope[0], scope[1], release_id, 1, ordinal] + [row[n] for n in names] for ordinal, row in enumerate(rows, 1)]
        ch = client(self.env)
        try:
            ch.insert('hb_local.' + contract.table, values,
                column_names=['installation_id','workspace_id','release_id','attempt','ordinal'] + names)
        finally:
            ch.close()

    def rows(self, scope, release, reader=False):
        contract = pinned_contract(release)
        user = 'hb_local_reader_' + ('a' if scope[1] == persona('reader-a')[1] else 'b') if reader else 'hb_local_writer'
        ch = client(self.env, user)
        try:
            names = [f['name'] for f in contract.fields]
            query = ','.join('`' + n + '`' for n in names)
            result = ch.query(f'SELECT {query} FROM hb_local.{contract.table} WHERE installation_id={{i:UUID}} AND workspace_id={{w:UUID}} AND release_id={{r:UUID}} AND attempt=1 ORDER BY ordinal LIMIT {MAX_ROWS + 1}', parameters={'i': str(scope[0]), 'w': str(scope[1]), 'r': str(release['release_id'])})
            if len(result.result_rows) > MAX_ROWS:
                raise Rejected('row_limit')
            rows = [dict(zip(names, r)) for r in result.result_rows]
            for row in rows:
                for field in contract.fields:
                    value = row[field['name']]
                    if field['type'] == 'timestamp' and value is not None and value.tzinfo is None:
                        row[field['name']] = value.replace(tzinfo=timezone.utc)
            return rows
        finally:
            ch.close()

    def verify(self, scope, release):
        contract = pinned_contract(release)
        rows = self.rows(scope, release)
        if len(rows) != release['manifest']['row_count'] or rows_digest(rows) != release['manifest']['rows_digest'] or controls(rows, contract.fields) != release['manifest']['controls'] or self.aggregates(scope, release) != release['manifest']['controls']:
            raise Rejected('loaded_controls')
        return rows

    def aggregates(self, scope, release, reader=False):
        contract = pinned_contract(release)
        user = 'hb_local_reader_' + ('a' if scope[1] == persona('reader-a')[1] else 'b') if reader else 'hb_local_writer'
        ch = client(self.env, user)
        try:
            numeric = [f['name'] for f in contract.fields if f['type'] == 'decimal' and f['aggregate'] == 'sum']
            expressions = ','.join(f'sum(`{n}`),countIf(isNull(`{n}`))' for n in numeric)
            result = ch.query(f'SELECT billing_currency,consumed_unit,pricing_unit,count(),{expressions} FROM hb_local.{contract.table} WHERE installation_id={{i:UUID}} AND workspace_id={{w:UUID}} AND release_id={{r:UUID}} AND attempt=1 GROUP BY billing_currency,consumed_unit,pricing_unit', parameters={'i': str(scope[0]), 'w': str(scope[1]), 'r': str(release['release_id'])})
            grouped = {}
            total_rows = 0
            for row in result.result_rows:
                total_rows += row[3]
                grouped[canonical(list(row[:3]))] = {'rows': row[3], 'measures': {
                    name: {'coefficient': coefficient(row[4 + 2 * index]) if row[3] != row[5 + 2 * index] else None, 'missing': row[5 + 2 * index], 'present': row[3] - row[5 + 2 * index]}
                    for index, name in enumerate(numeric)}}
            if total_rows > MAX_ROWS:
                raise Rejected('row_limit')
            return grouped
        finally:
            ch.close()

    def export(self, scope, release):
        contract = pinned_contract(release)
        rows = self.verify(scope, release)
        schema = pa.schema([pa.field(f['name'], ARROW_TYPES[f['type']], nullable=True) for f in contract.fields])
        table = pa.Table.from_pylist(rows, schema=schema)
        buffer = io.BytesIO()
        pq.write_table(table, buffer)
        data = buffer.getvalue()
        key = f'{scope[0]}/{scope[1]}/exports/{release["release_id"]}/{digest(data)}.parquet'
        self.put(key, data)
        restored = pq.read_table(io.BytesIO(self.get(key)))
        if restored.schema != schema or restored.num_rows != len(rows) or rows_digest(restored.to_pylist()) != release['manifest']['rows_digest'] or controls(restored.to_pylist(), contract.fields) != release['manifest']['controls']:
            raise Rejected('parquet_controls')
        return key, digest(data)
