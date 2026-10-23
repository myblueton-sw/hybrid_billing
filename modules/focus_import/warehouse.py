import hashlib
import re
from datetime import timezone
from decimal import localcontext
from uuid import UUID

import clickhouse_connect

from modules.focus_local.database import password
from modules.focus_local.storage import LOCKED_SETTINGS, object_client
from .authority import APPROVER, IMPORTER, reader_role
from .model import ADDITIVE, COLUMNS, NUMERIC, STRINGS, TIMES, Rejected, canonical, coefficient, controls, digest, row_digest, tenant_id


BUCKET='hb-focus-originals'
CH_WRITER='hb_focus_writer'
TYPES={**{n:'Nullable(String)' for n in STRINGS},**{n:"DateTime64(6,'UTC')" for n in TIMES},**{n:'Nullable(Decimal(38,18))' for n in NUMERIC},'missing_reasons':'String','source_json':'String'}


def client(env,user=CH_WRITER):
    secret=env['CH_ADMIN_PASSWORD'] if user=='probe_admin' else password(env,user,'CH_LOADER_PASSWORD' if user==CH_WRITER else 'CH_READER_PASSWORD')
    return clickhouse_connect.get_client(host='127.0.0.1',port=18123,username=user,password=secret,autogenerate_session_id=False,connect_timeout=5,send_receive_timeout=30)


def initialize(env,config):
    admin=client(env,'probe_admin')
    roles={CH_WRITER:None,**{reader_role(tenant_id(t['key'])):tenant_id(t['key']) for t in config['tenants']}}
    try:
        admin.command('CREATE DATABASE IF NOT EXISTS hb_focus')
        admin.command('CREATE SETTINGS PROFILE IF NOT EXISTS hb_focus_locked SETTINGS '+LOCKED_SETTINGS)
        columns=','.join('`'+n+'` '+TYPES[n] for n in COLUMNS)
        admin.command(f'CREATE TABLE IF NOT EXISTS hb_focus.fact (tenant_id UUID,batch_id UUID,ordinal UInt64,{columns}) ENGINE=MergeTree ORDER BY (tenant_id,batch_id,ordinal)')
        for user,tenant in roles.items():
            key='CH_LOADER_PASSWORD' if tenant is None else 'CH_READER_PASSWORD'
            verifier=hashlib.sha256(password(env,user,key).encode()).hexdigest()
            admin.command(f"CREATE USER IF NOT EXISTS {user} IDENTIFIED WITH sha256_hash BY '{verifier}' SETTINGS PROFILE 'hb_focus_locked'")
            grant='SELECT, INSERT' if tenant is None else 'SELECT'
            admin.command(f'GRANT {grant} ON hb_focus.fact TO {user}')
            condition='1' if tenant is None else f"tenant_id=toUUID('{tenant}')"
            admin.command(f'CREATE ROW POLICY IF NOT EXISTS p_{user} ON hb_focus.fact FOR SELECT USING {condition} TO {user}')
            admin.command(f'ALTER ROW POLICY p_{user} ON hb_focus.fact FOR SELECT USING {condition} TO {user}')
            for row in admin.query('SHOW GRANTS FOR '+user).result_rows:
                match=re.fullmatch(r'GRANT ([A-Z, ]+) ON hb_focus\.fact TO '+user,row[0].replace('`',''))
                if not match or not set(match[1].split(', '))<=({'SELECT','INSERT'} if tenant is None else {'SELECT'}):raise Rejected('incompatible_ch_privileges')
            effective=client(env,user)
            try:
                required={'join_use_nulls':'1','input_format_null_as_default':'0','input_format_csv_empty_as_default':'1','input_format_parquet_allow_missing_columns':'0','serialize_query_plan':'0','async_insert':'0'}
                values=effective.query('SELECT name,value,readonly FROM system.settings WHERE name IN ('+','.join("'"+n+"'" for n in required)+')').result_rows
                if {n:v for n,v,_ in values}!=required or not all(r for _,_,r in values):raise Rejected('incompatible_ch_settings')
            finally:effective.close()
        policies={r[0] for r in admin.query("SELECT short_name FROM system.row_policies WHERE database='hb_focus' AND table='fact'").result_rows}
        if policies!={'p_'+r for r in roles}:raise Rejected('incompatible_ch_policy')
    finally:admin.close()
    s3=object_client(env)
    try:s3.create_bucket(Bucket=BUCKET)
    except s3.exceptions.BucketAlreadyOwnedByYou:pass


class Warehouse:
    def __init__(self,env):self.env=env

    def original(self,batch,data):
        s3=object_client(self.env)
        key='shared-originals/'+str(batch)+'/'+digest(data)+'.csv'
        s3.put_object(Bucket=BUCKET,Key=key,Body=data,IfNoneMatch='*',Metadata={'sha256':digest(data)})
        if s3.get_object(Bucket=BUCKET,Key=key)['Body'].read()!=data:raise Rejected('original_digest')
        return key

    def insert(self,batch,rows):
        ch=client(self.env)
        try:
            with localcontext() as context:
                context.prec=80
                ch.insert('hb_focus.fact',[[r['tenant_id'],batch,r['ordinal']]+[r[n] for n in COLUMNS] for r in rows],column_names=['tenant_id','batch_id','ordinal']+COLUMNS)
        finally:ch.close()

    def rows(self,batch,tenant,reader=False):
        ch=client(self.env,reader_role(tenant) if reader else CH_WRITER)
        try:
            names=['tenant_id','ordinal']+COLUMNS
            result=ch.query('SELECT '+','.join('`'+n+'`' for n in names)+' FROM hb_focus.fact WHERE tenant_id={t:UUID} AND batch_id={b:UUID} ORDER BY ordinal LIMIT 10001',parameters={'t':str(tenant),'b':str(batch)})
            if len(result.result_rows)>10000:raise Rejected('row_limit')
            rows=[dict(zip(names,r)) for r in result.result_rows]
            for r in rows:
                for n in TIMES:
                    if r[n].tzinfo is None:r[n]=r[n].replace(tzinfo=timezone.utc)
            return rows
        finally:ch.close()

    def aggregates(self,batch,tenant,reader=False):
        ch=client(self.env,reader_role(tenant) if reader else CH_WRITER)
        try:
            expr=','.join(f'sum(CAST(`{n}` AS Nullable(Decimal(76,18)))),countIf(isNull(`{n}`))' for n in ADDITIVE)
            result=ch.query(f'SELECT provider,service,currency,consumed_unit,pricing_unit,count(),{expr} FROM hb_focus.fact WHERE tenant_id={{t:UUID}} AND batch_id={{b:UUID}} GROUP BY provider,service,currency,consumed_unit,pricing_unit',parameters={'t':str(tenant),'b':str(batch)})
            return {canonical(list(r[:5])):{'rows':r[5],'measures':{n:{'coefficient':coefficient(r[6+2*i]) if r[5]!=r[7+2*i] else None,'present':r[5]-r[7+2*i],'missing':r[7+2*i]} for i,n in enumerate(ADDITIVE)}} for r in result.result_rows}
        finally:ch.close()

    def verify(self,batch,tenant,expected,reader=False):
        rows=self.rows(batch,tenant,reader)
        if len(rows)!=expected['row_count'] or row_digest(rows)!=expected['row_digest'] or controls(rows)!=expected['controls'] or self.aggregates(batch,tenant,reader)!=expected['controls']:
            raise Rejected('warehouse_controls')
        return rows
