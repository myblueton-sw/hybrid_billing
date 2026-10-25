import csv
import gzip
import hashlib
import io
import json
import urllib.request
from pathlib import Path

from .model import ROOT, REVISION, SOURCE_SHA, Rejected, canonical, decode, digest, domain, domains_digest, tenant_id, validate_config, versions


FULL_SHA='c198cda74cbde4b11b9be289815bda7e9ac62042f1d023ad9bc8fbf7bff1018c'
FULL_BYTES=496305507
FULL_URL=f'https://media.githubusercontent.com/media/FinOps-Open-Cost-and-Usage-Spec/FOCUS-Sample-Data/{REVISION}/FOCUS-1.0/focus_data_table.csv.gz'
GCP_SHA='c602b6871d0a0a8bae3a80daa8b3615c7ba948f304cd1f7213706776e6f5acde'
GCP_ORDINAL_SHA='c074fe0f27e848803b20fb5eb9b78cc8f0ba615b120fa9c0ff0ddb74ec96b9fa'
PROVENANCE=['x_HB_EvidenceClass','x_HB_SourceArtifact','x_HB_SourceDigest','x_HB_SourceOrdinal']


def full_source(path):
    if not path.exists():
        path.parent.mkdir(parents=True,exist_ok=True)
        temporary=path.with_suffix('.download')
        total=0;h=hashlib.sha256()
        with urllib.request.urlopen(FULL_URL,timeout=30) as response,temporary.open('wb') as output:
            while chunk:=response.read(1024*1024):
                total+=len(chunk)
                if total>FULL_BYTES:raise Rejected('full_source_size')
                h.update(chunk);output.write(chunk)
        if total!=FULL_BYTES or h.hexdigest()!=FULL_SHA:raise Rejected('full_source_digest')
        temporary.replace(path)
    with path.open('rb') as stream:sha=hashlib.file_digest(stream,'sha256').hexdigest()
    if sha!=FULL_SHA or path.stat().st_size!=FULL_BYTES:raise Rejected('full_source_digest')


def google_rows(path):
    full_source(path)
    cache=ROOT/'.local/datasets/focus/gcp-115.csv'
    metadata=cache.with_suffix('.provenance.json')
    if cache.exists() and metadata.exists():
        p=json.loads(metadata.read_text())
        if digest(cache.read_bytes())==GCP_SHA and digest(canonical(p.get('source_ordinals')).encode())==GCP_ORDINAL_SHA:
            return list(csv.DictReader(io.StringIO(cache.read_text(),newline=''))),p['source_ordinals']
    selected=[];ordinals=[]
    with gzip.open(path,'rt',encoding='utf-8',newline='') as stream:
        for ordinal,row in enumerate(csv.DictReader(stream),1):
            if row['ProviderName']=='Google Cloud':selected.append(row);ordinals.append(ordinal)
    if len(selected)!=115 or digest(canonical(ordinals).encode())!=GCP_ORDINAL_SHA:raise Rejected('google_selection')
    return selected,ordinals


def prepare(data,gcp,ordinals,synthetic):
    if digest(data)!=SOURCE_SHA or len(gcp)!=115 or digest(canonical(ordinals).encode())!=GCP_ORDINAL_SHA:raise Rejected('sample_sources')
    official=list(csv.DictReader(io.StringIO(data.decode(),newline='')))
    synthetic_rows=list(csv.DictReader(io.StringIO(synthetic.decode(),newline='')))
    if len(synthetic_rows)!=8 or {r['ProviderName'] for r in synthetic_rows}!={'VMware','Alibaba Cloud'}:raise Rejected('synthetic_sources')
    stream=io.StringIO(newline='');writer=csv.DictWriter(stream,fieldnames=list(official[0])+PROVENANCE);writer.writeheader()
    for rows,positions,kind,artifact,sha in [(official,range(1,1001),'official-public','official1000',SOURCE_SHA),(gcp,ordinals,'official-public-subset','official-full',FULL_SHA),(synthetic_rows,range(1,9),'synthetic-contract-fixture','synthetic-vmware-alibaba',digest(synthetic))]:
        for row,ordinal in zip(rows,positions):writer.writerow({**row,**dict(zip(PROVENANCE,[kind,artifact,sha,str(ordinal)]))})
    combined=stream.getvalue().encode()
    rows=decode(combined,versions()['1.0'])
    tenants=[]
    for key,sub,company,tz in [('north-a','a','company-north','Asia/Seoul'),('north-b','b','company-north','UTC'),('foreign','foreign','company-foreign','UTC')]:
        selected=[]
        for row in rows:
            if row['provider'] in ['VMware','Alibaba Cloud']:
                include=row['subaccount']=='synthetic-'+sub
            else:include=(key=='north-b' if row['provider']=='Oracle' else key=='north-a')
            if include:selected.append(row)
        assignments=sorted({canonical(domain(r)) for r in selected})
        tenants.append({'key':key,'label':'Synthetic '+key,'company_id':company,'company_label':'Same Display Label','timezone':tz,'reporting_currency':'USD','currencies':['USD','CNY',None],'retention_days':180,'actor':'reader-'+key,'active':True,'assignments':[json.loads(x) for x in assignments]})
    config={'source':'six-provider-sample','version':'1.0','revision':1,'tenants':tenants,'snapshot':{'file_digest':digest(combined),'row_count':len(rows),'domains_digest':domains_digest(rows)},'views':[{'actor':'reader-company-north','company_id':'company-north','tenant_keys':['north-a','north-b'],'active':True}]}
    validate_config(config)
    return combined,config


def build(full_path,official_path,output):
    gcp,ordinals=google_rows(full_path)
    synthetic=(ROOT/'tests/fixtures/focus/six-provider/synthetic-vmware-alibaba.csv').read_bytes()
    data,config=prepare(official_path.read_bytes(),gcp,ordinals,synthetic)
    output.parent.mkdir(parents=True,exist_ok=True);output.write_bytes(data)
    config_path=output.with_suffix('.config.json');config_path.write_text(json.dumps(config,indent=2)+'\n')
    provenance={'declared_version':'1.0','classification':'mixed-public-and-synthetic-query-only','official1000_sha256':SOURCE_SHA,'official_full_sha256':FULL_SHA,'google_source_ordinals':ordinals,'google_rows':115,'synthetic_rows':8,'synthetic_sha256':digest(synthetic),'combined_sha256':digest(data),'combined_rows':1123,'source_revision':REVISION,'source_values_changed':False,'format_note':'Derived CSV union adds provenance columns; official lexical values retained, not original byte identity.'}
    output.with_suffix('.provenance.json').write_text(json.dumps(provenance,indent=2)+'\n')
    return {'rows':1123,'sha256':digest(data),'config':str(config_path),'data':str(output),'google_rows':115,'synthetic_rows':8}
