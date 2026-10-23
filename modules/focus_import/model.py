import csv
import io
import json
import re
from datetime import datetime, timezone
from decimal import Decimal
from pathlib import Path
from uuid import UUID, uuid5
from zoneinfo import ZoneInfo

from modules.focus_local.contract import ROOT, Rejected, canonical, digest


MAX_BYTES = 16 * 1024 * 1024
MAX_ROWS = 10000
SCALE = 18
LIMIT = 10**38
INSTALLATION = UUID('1a0ea32d-ec92-47f8-82c9-d4f7b97ed6fe')
SOURCE = 'official-focus-sample'
SOURCE_SHA = 'e91e5ac7edf01ed2c9d926f37ef7dc1ae2aae97956fea8da6c9ee488b1c2839e'
REVISION = 'adbdd17a132984d6e8583c149c236d2199c3f5bc'
SOURCE_URL = f'https://raw.githubusercontent.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS-Sample-Data/{REVISION}/FOCUS-1.0/focus_sample.csv'
NUMERIC = ['billed_cost','effective_cost','list_cost','contracted_cost','consumed_quantity','pricing_quantity','list_unit_price','contracted_unit_price']
ADDITIVE = NUMERIC[:6]
TIMES = ['billing_start','billing_end','charge_start','charge_end']
STRINGS = ['provider','service','currency','account','subaccount','consumed_unit','pricing_unit']
COLUMNS = STRINGS + TIMES + NUMERIC + ['missing_reasons','source_json']
BINDINGS = ROOT / '.local/focus-import-bindings.json'


def versions(path=None):
    path = path or ROOT/'packages/contracts/focus_import/versions.json'
    data = json.loads(Path(path).read_text())
    if set(data) != {'versions'} or not isinstance(data['versions'],dict) or not data['versions']:
        raise Rejected('mapping_registry')
    for version, entry in data['versions'].items():
        if not re.fullmatch(r'\d+\.\d+',version) or set(entry) != {'classification','source_timezone','null_tokens','columns'}:
            raise Rejected('mapping_registry')
        if entry['classification'] != 'exploratory-query-only' or entry['source_timezone'] != 'UTC':
            raise Rejected('mapping_registry')
        if not isinstance(entry['null_tokens'],list) or not all(isinstance(x,str) for x in entry['null_tokens']):
            raise Rejected('mapping_registry')
        if set(entry['columns']) != set(STRINGS+TIMES+NUMERIC) or not all(isinstance(v,str) and re.fullmatch('[A-Za-z][A-Za-z0-9_]{0,63}',v) for v in entry['columns'].values()):
            raise Rejected('mapping_registry')
    return data['versions']


def number(text):
    if not isinstance(text,str) or len(text)>128 or not re.fullmatch(r'-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?',text):
        raise Rejected('decimal_syntax')
    value=Decimal(text)
    sign,digits,exponent=value.as_tuple()
    if abs(exponent)>128 or exponent < -SCALE:
        raise Rejected('decimal_scale')
    coefficient=int(''.join(map(str,digits)) or '0')*10**(exponent+SCALE)
    if coefficient>=LIMIT:
        raise Rejected('decimal_magnitude')
    return decimal(-coefficient if sign else coefficient)


def coefficient(value):
    sign,digits,exponent=value.as_tuple()
    if exponent < -SCALE:
        raise Rejected('decimal_scale')
    n=int(''.join(map(str,digits)) or '0')*10**(exponent+SCALE)
    return -n if sign else n


def fixed(n):
    whole,fraction=divmod(abs(n),10**SCALE)
    return ('-' if n<0 else '')+f'{whole}.{fraction:018d}'


def decimal(n):
    return Decimal(fixed(n))


def instant(text):
    if not isinstance(text,str) or not re.fullmatch(r'\d{4}-\d\d-\d\d[T ]\d\d:\d\d:\d\d(?:\.\d{1,6})?(?:Z|[+-]\d\d:\d\d)?',text):
        raise Rejected('timestamp_format')
    try:
        d=datetime.fromisoformat(text.replace('Z','+00:00'))
        return (d if d.tzinfo else d.replace(tzinfo=timezone.utc)).astimezone(timezone.utc)
    except ValueError:
        raise Rejected('timestamp_format') from None


def domain(row):
    return [row[k].isoformat() if k in TIMES else row[k] for k in ['provider','account','subaccount','billing_start','billing_end']]


def decode(data, mapping):
    if len(data)>MAX_BYTES:
        raise Rejected('input_size')
    try:
        reader=csv.DictReader(io.StringIO(data.decode('utf-8'),newline=''),strict=True)
        headers=reader.fieldnames or []
        if len(set(headers))!=len(headers) or not set(mapping['columns'].values())<=set(headers):
            raise Rejected('source_headers')
        rows=[]
        for ordinal,source in enumerate(reader,1):
            if ordinal>MAX_ROWS or None in source or any(v is None or len(v)>65536 for v in source.values()):
                raise Rejected('source_shape')
            row={'ordinal':ordinal,'source_json':canonical(source)}
            reasons={}
            for name,column in mapping['columns'].items():
                value=source[column]
                if value in mapping['null_tokens']:
                    reasons[name]='empty' if value=='' else 'literal:'+value
                    value=None
                if name in TIMES:
                    value=instant(value)
                elif name in NUMERIC and value is not None:
                    value=number(value)
                row[name]=value
            if not all(row[n] is not None for n in ['provider','account','subaccount','service']):
                raise Rejected('required_source_scope')
            if row['billing_start']>=row['billing_end'] or row['charge_start']>=row['charge_end']:
                raise Rejected('timestamp_interval')
            if row['currency'] is not None and not re.fullmatch('[A-Z]{3}',row['currency']):
                raise Rejected('currency')
            row['missing_reasons']=canonical(reasons)
            rows.append(row)
        if not rows:
            raise Rejected('empty_input')
        for n in ADDITIVE:
            if sum(abs(coefficient(r[n])) for r in rows if r[n] is not None)>=LIMIT:
                raise Rejected('aggregate_bound')
        return rows
    except (UnicodeError,csv.Error,ValueError,TypeError):
        raise Rejected('source_format') from None


def tenant_id(key):
    return uuid5(INSTALLATION,key)


def validate_config(config):
    if set(config)!={'source','version','revision','tenants','snapshot'} or not isinstance(config['source'],str) or not re.fullmatch('[a-z][a-z0-9-]{0,63}',config['source']):
        raise Rejected('configuration')
    if type(config['revision']) is not int or not 0<config['revision']<2**63 or config['version'] not in versions():
        raise Rejected('configuration_version')
    if not isinstance(config['tenants'],list) or not 1<=len(config['tenants'])<=20:
        raise Rejected('configuration_tenants')
    snapshot=config['snapshot']
    if not isinstance(snapshot,dict) or set(snapshot)!={'file_digest','row_count','domains_digest'} or any(not isinstance(snapshot[k],str) or not re.fullmatch('[a-f0-9]{64}',snapshot[k]) for k in ['file_digest','domains_digest']) or type(snapshot['row_count']) is not int or not 0<snapshot['row_count']<=MAX_ROWS:raise Rejected('snapshot_profile')
    keys=set();actors=set();assignments=set()
    for t in config['tenants']:
        if set(t)!={'key','label','timezone','reporting_currency','currencies','retention_days','actor','active','assignments'}:
            raise Rejected('tenant_properties')
        if not isinstance(t['key'],str) or not re.fullmatch('[a-z][a-z0-9-]{0,40}',t['key']) or t['key'] in keys:
            raise Rejected('tenant_identity')
        if not isinstance(t['actor'],str) or not re.fullmatch('reader-[a-z0-9-]{1,40}',t['actor']) or t['actor'] in actors:
            raise Rejected('tenant_actor')
        keys.add(t['key']);actors.add(t['actor'])
        try:ZoneInfo(t['timezone'])
        except (ValueError,KeyError,TypeError):raise Rejected('tenant_timezone') from None
        if not isinstance(t['label'],str) or not 1<=len(t['label'])<=100 or type(t['active']) is not bool or type(t['retention_days']) is not int or not 1<=t['retention_days']<=36500:
            raise Rejected('tenant_properties')
        if not isinstance(t['reporting_currency'],str) or not re.fullmatch('[A-Z]{3}',t['reporting_currency']) or not isinstance(t['currencies'],list) or not t['currencies'] or any(x is not None and (not isinstance(x,str) or not re.fullmatch('[A-Z]{3}',x)) for x in t['currencies']):
            raise Rejected('tenant_currency')
        if not isinstance(t['assignments'],list) or len(t['assignments'])>MAX_ROWS:
            raise Rejected('source_assignments')
        for item in t['assignments']:
            if not isinstance(item,list) or len(item)!=5 or not all(isinstance(x,str) and 0<len(x)<=4096 for x in item):
                raise Rejected('source_assignments')
            normalized=item[:3]+[instant(item[3]).isoformat(),instant(item[4]).isoformat()]
            if normalized!=item or instant(item[3])>=instant(item[4]):raise Rejected('source_assignments')
            token=canonical(item)
            if token in assignments:raise Rejected('overlapping_assignment')
            assignments.add(token)
    return config


def demo_config(data):
    if digest(data)!=SOURCE_SHA:
        raise Rejected('official_fixture_digest')
    mapping=versions()['1.0']
    rows=decode(data,mapping)
    tenants=[]
    for key,providers,tz,retention in [('demo-a',{'AWS'},'Asia/Seoul',180),('demo-b',{'Microsoft','Oracle'},'UTC',365)]:
        tuples={canonical(domain(r)) for r in rows if r['provider'] in providers}
        tenants.append({'key':key,'label':'Synthetic '+key,'timezone':tz,'reporting_currency':'USD','currencies':['USD',None],
            'retention_days':retention,'actor':'reader-'+key,'active':True,'assignments':[json.loads(x) for x in sorted(tuples)]})
    return validate_config({'source':SOURCE,'version':'1.0','revision':1,'tenants':tenants,'snapshot':{'file_digest':digest(data),'row_count':len(rows),'domains_digest':domains_digest(rows)}})


def domains_digest(rows):
    counts={}
    for r in rows:
        key=canonical(domain(r));counts[key]=counts.get(key,0)+1
    return digest(canonical(counts).encode())


def assign(rows,config):
    owners={canonical(d):t for t in config['tenants'] for d in t['assignments']}
    for row in rows:
        t=owners.get(canonical(domain(row)))
        if not t or not t['active'] or row['currency'] not in t['currencies']:
            raise Rejected('unmapped_or_denied_source')
        row['tenant_id']=tenant_id(t['key'])
    if len(rows)!=config['snapshot']['row_count'] or domains_digest(rows)!=config['snapshot']['domains_digest']:raise Rejected('snapshot_coverage')
    return rows


def serialized(row):
    return {k:(fixed(coefficient(v)) if isinstance(v,Decimal) else v.isoformat() if isinstance(v,datetime) else str(v) if isinstance(v,UUID) else v) for k,v in row.items()}


def row_digest(rows):
    return digest(canonical([serialized(r) for r in rows]).encode())


def controls(rows):
    groups={}
    for r in rows:
        key=canonical([r[k] for k in ['provider','service','currency','consumed_unit','pricing_unit']])
        g=groups.setdefault(key,{'rows':0,'measures':{n:{'coefficient':None,'present':0,'missing':0} for n in ADDITIVE}})
        g['rows']+=1
        for n in ADDITIVE:
            m=g['measures'][n]
            if r[n] is None:m['missing']+=1
            else:m['present']+=1;m['coefficient']=(m['coefficient'] or 0)+coefficient(r[n])
    return groups
