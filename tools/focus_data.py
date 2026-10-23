import argparse
import json
import sys
import urllib.request
from pathlib import Path

sys.path.insert(0,str(Path(__file__).resolve().parents[1]))

from modules.focus_import.model import MAX_BYTES, ROOT, SOURCE_SHA, SOURCE_URL, Rejected, demo_config, digest, versions
from modules.focus_import.service import Coordinator, initialize


def fetch(path):
    request=urllib.request.Request(SOURCE_URL,headers={'User-Agent':'Hybrid-Billing-HB27-demo'})
    with urllib.request.urlopen(request,timeout=30) as response:
        data=response.read(MAX_BYTES+1)
    if len(data)>MAX_BYTES or digest(data)!=SOURCE_SHA:raise Rejected('official_fixture_digest')
    path.parent.mkdir(parents=True,exist_ok=True)
    temporary=path.with_suffix('.download')
    temporary.write_bytes(data);temporary.replace(path)
    return {'downloaded_bytes':len(data),'sha256':digest(data),'declared_version':'1.0'}


def bounded(path):
    with path.open('rb') as stream:return stream.read(MAX_BYTES+1)


def main():
    p=argparse.ArgumentParser(description='Local tenant-aware FOCUS evidence ingestion and query')
    sub=p.add_subparsers(dest='command',required=True)
    sub.add_parser('versions')
    f=sub.add_parser('fetch-demo');f.add_argument('--output',type=Path,default=ROOT/'.local/datasets/focus/official-1000.csv')
    i=sub.add_parser('init-demo');i.add_argument('--file',type=Path,required=True)
    i=sub.add_parser('init');i.add_argument('--config',type=Path,required=True)
    i=sub.add_parser('demo');i.add_argument('--file',type=Path,required=True)
    i=sub.add_parser('build-samples');i.add_argument('--full-source',type=Path,default=ROOT/'.local/datasets/focus/full.csv.gz');i.add_argument('--official',type=Path,default=ROOT/'.local/datasets/focus/official-1000.csv');i.add_argument('--output',type=Path,default=ROOT/'.local/datasets/focus/six-provider.csv')
    for name in ['ingest','confirm','publish','query']:
        c=sub.add_parser(name);c.add_argument('--actor',required=True)
        if name=='ingest':
            c.add_argument('--file',type=Path,required=True);c.add_argument('--version',required=True);c.add_argument('--revision',type=int,required=True)
        if name in ['confirm','publish']:c.add_argument('--batch',required=True)
        if name=='confirm':c.add_argument('--manifest-digest',required=True)
        if name=='query':
            c.add_argument('--limit',type=int,default=20);c.add_argument('--after',type=int,default=0);c.add_argument('--provider');c.add_argument('--service');c.add_argument('--expected-batch');c.add_argument('--expected-hierarchy')
    args=p.parse_args();coordinator=None
    try:
        if args.command=='versions':result={'versions':list(versions()),'input_formats':['csv','parquet'],'classification':'exploratory-query-only'}
        elif args.command=='fetch-demo':result=fetch(args.output)
        elif args.command=='build-samples':
            from modules.focus_import.samples import build
            result=build(args.full_source,args.official,args.output)
        elif args.command=='init-demo':result=initialize(demo_config(bounded(args.file)))
        elif args.command=='init':result=initialize(json.loads(args.config.read_text()))
        elif args.command=='demo':
            data=bounded(args.file);initialize(demo_config(data))
            loader=Coordinator('importer');approver=Coordinator('approver')
            try:
                result=loader.ingest(data,'1.0',1)
                approver.confirm(result['batch_id'],result['manifest_digest']);result=approver.publish(result['batch_id'])
            finally:loader.close();approver.close()
            result={**result,'synthetic_tenant_assignment':True,'synthetic_approval_scenario':True}
        else:
            coordinator=Coordinator(args.actor)
            if args.command=='ingest':result=coordinator.ingest(bounded(args.file),args.version,args.revision)
            elif args.command=='confirm':result=coordinator.confirm(args.batch,args.manifest_digest)
            elif args.command=='publish':result=coordinator.publish(args.batch)
            else:result=coordinator.query(args.limit,args.after,args.provider,args.service,args.expected_batch,args.expected_hierarchy)
        print(json.dumps({'status':'ok',**result},sort_keys=True));return 0
    except Rejected as exc:print(json.dumps({'status':'rejected','reason':str(exc)}));return 2
    except Exception:print(json.dumps({'status':'unavailable','reason':'local_dependency_or_operation_failed'}));return 3
    finally:
        if coordinator:coordinator.close()


if __name__=='__main__':raise SystemExit(main())
