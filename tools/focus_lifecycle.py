import argparse
import json
import sys
import time
from pathlib import Path

sys.path.insert(0,str(Path(__file__).resolve().parents[1]))

from modules.focus_import import collection, lifecycle
from modules.focus_import.intake import input_format
from modules.focus_import.model import MAX_BYTES, Rejected, decode, digest, domains_digest, versions
from modules.focus_import.temporal import window


def bounded(path):
    with path.open('rb') as stream:data=stream.read(MAX_BYTES+1)
    if len(data)>MAX_BYTES:raise Rejected('input_size')
    return data


def run_due(actor,inbox):
    if actor!='importer':raise Rejected('permission')
    with collection.access(actor,'collect') as (journal,c,conn):
        pending={key:d['descriptor']['file_digest'] for key,d in journal.state['deliveries'].items() if d['state']=='registered'}
        blocked=any(d['state']=='verified' for d in journal.state['deliveries'].values())
    results=[]
    if blocked:return {'collected':[],'registered_waiting':len(pending),'awaiting_publication':True}
    for path in sorted(p for p in inbox.iterdir() if p.is_file() and p.suffix in ['.csv','.parquet'])[:100]:
        data=bounded(path);matches=[key for key,sha in pending.items() if sha==digest(data)]
        if matches:
            key=matches[0];results.append(collection.collect(actor,key,data));pending.pop(key)
            break
    return {'collected':results,'registered_waiting':len(pending),'awaiting_publication':bool(results)}


def parser():
    p=argparse.ArgumentParser(description='Trusted local FOCUS collection, temporal query and guarded lifecycle')
    sub=p.add_subparsers(dest='command',required=True)
    sub.add_parser('hierarchy-profiles')
    c=sub.add_parser('hierarchy-plan');c.add_argument('--file',type=Path)
    c=sub.add_parser('hierarchy-apply');c.add_argument('--manifest-digest',required=True)
    c=sub.add_parser('describe');c.add_argument('--file',type=Path,required=True);c.add_argument('--version',required=True);c.add_argument('--start',required=True);c.add_argument('--end',required=True);c.add_argument('--grain',choices=['day','month'],required=True);c.add_argument('--revision',type=int,required=True);c.add_argument('--complete-through',required=True);c.add_argument('--finalized',action='store_true')
    c=sub.add_parser('activate');c.add_argument('--grain',choices=['day','month'],required=True);c.add_argument('--policies',type=Path,required=True)
    c=sub.add_parser('register');c.add_argument('--file',type=Path,required=True);c.add_argument('--descriptor',type=Path,required=True);c.add_argument('--version',required=True)
    c=sub.add_parser('collect');c.add_argument('--delivery',required=True);c.add_argument('--file',type=Path,required=True)
    c=sub.add_parser('confirm');c.add_argument('--delivery',required=True);c.add_argument('--manifest-digest',required=True)
    c=sub.add_parser('publish');c.add_argument('--delivery',required=True)
    for name in ['run-due','watch']:
        c=sub.add_parser(name);c.add_argument('--inbox',type=Path,required=True)
        if name=='watch':c.add_argument('--interval',type=int,default=60);c.add_argument('--cycles',type=int,default=0)
    c=sub.add_parser('query');c.add_argument('--start');c.add_argument('--end');c.add_argument('--grain',choices=['day','month'],default='month');c.add_argument('--limit',type=int,default=20);c.add_argument('--cursor');c.add_argument('--snapshot');c.add_argument('--provider');c.add_argument('--service');c.add_argument('--hierarchy-node');c.add_argument('--group-kind');c.add_argument('--group-axis');c.add_argument('--group-role',default='usage')
    for name in ['archive','recall']:
        c=sub.add_parser(name);c.add_argument('--slice',required=True)
    for name in ['hold','reference']:
        c=sub.add_parser(name);c.add_argument('--slice',required=True);c.add_argument('--id',required=True);c.add_argument('--reason',required=True);c.add_argument('--release',action='store_true')
    c=sub.add_parser('preview');c.add_argument('--slice',action='append')
    for name in ['approve','delete','policy-apply','retire']:
        c=sub.add_parser(name);c.add_argument('--manifest-digest',required=True)
    for name in ['status','backup','prune-backups','retire-preview']:sub.add_parser(name)
    c=sub.add_parser('restore');c.add_argument('--backup',required=True)
    c=sub.add_parser('revoke');c.add_argument('--reader',required=True)
    c=sub.add_parser('policy-preview');c.add_argument('--tenant',required=True);c.add_argument('--policy',type=Path,required=True)
    for c in sub.choices.values():
        if c.prog.split()[-1] not in ['describe','hierarchy-profiles']:c.add_argument('--actor',required=True)
    return p


def execute(a):
    cmd=a.command
    if cmd=='hierarchy-profiles':
        from modules.focus_import.hierarchy_model import registry,registry_digest
        return {'profiles':registry(),'registry_digest':registry_digest()}
    if cmd=='hierarchy-plan':
        from modules.focus_import.hierarchy_admin import prepare
        return prepare(a.actor,json.loads(bounded(a.file)) if a.file else None)
    if cmd=='hierarchy-apply':
        from modules.focus_import.hierarchy_admin import apply
        return apply(a.actor,a.manifest_digest)
    if cmd=='describe':
        data=bounded(a.file)
        if a.version not in versions():raise Rejected('unsupported_version')
        try:rows=decode(data,versions()[a.version])
        except Rejected as e:
            if str(e)!='empty_input':raise
            rows=[]
        return {'descriptor':{'window':window(a.start,a.end,a.grain),'revision':a.revision,'file_digest':digest(data),'row_count':len(rows),'domains_digest':domains_digest(rows),'complete_through':a.complete_through,'finalized':a.finalized},'version':a.version,'input_format':input_format(data),'completeness':'operator_assertion_required'}
    if cmd=='activate':return lifecycle.activate(a.actor,a.grain,json.loads(a.policies.read_text()))
    if cmd=='register':return collection.register(a.actor,bounded(a.file),a.version,json.loads(a.descriptor.read_text()))
    if cmd=='collect':return collection.collect(a.actor,a.delivery,bounded(a.file))
    if cmd=='confirm':return collection.confirm(a.actor,a.delivery,a.manifest_digest)
    if cmd=='publish':return collection.publish(a.actor,a.delivery)
    if cmd=='run-due':return run_due(a.actor,a.inbox)
    if cmd=='watch':
        if not 1<=a.interval<=3600 or a.cycles<0:raise Rejected('watch_options')
        cycle=0
        while True:
            result=run_due(a.actor,a.inbox);cycle+=1
            print(json.dumps({'status':'ok','cycle':cycle,**result}),flush=True)
            if a.cycles and cycle>=a.cycles:return {'state':'watch_completed','cycles':cycle}
            time.sleep(a.interval)
    if cmd=='query':return collection.query(a.actor,a.start,a.end,a.grain,a.limit,json.loads(a.cursor) if a.cursor else None,a.snapshot,a.provider,a.service,a.hierarchy_node,a.group_kind,a.group_axis,a.group_role)
    if cmd=='archive':return lifecycle.archive_slice(a.actor,a.slice)
    if cmd=='recall':return lifecycle.recall(a.actor,a.slice)
    if cmd in ['hold','reference']:return lifecycle.restriction(a.actor,a.slice,a.id,a.reason,cmd,a.release)
    if cmd=='preview':return lifecycle.preview(a.actor,identities=a.slice)
    if cmd=='approve':return lifecycle.approve(a.actor,a.manifest_digest)
    if cmd=='delete':return lifecycle.delete(a.actor,a.manifest_digest)
    if cmd=='status':return lifecycle.status(a.actor)
    if cmd=='retire-preview':return lifecycle.retire_preview(a.actor)
    if cmd=='retire':return lifecycle.retire(a.actor,a.manifest_digest)
    if cmd=='backup':return lifecycle.backup(a.actor)
    if cmd=='restore':return lifecycle.restore(a.actor,a.backup)
    if cmd=='prune-backups':return lifecycle.prune_backups(a.actor)
    if cmd=='revoke':return lifecycle.revoke(a.actor,a.reader)
    if cmd=='policy-preview':return lifecycle.policy_preview(a.actor,a.tenant,json.loads(a.policy.read_text()))
    if cmd=='policy-apply':return lifecycle.policy_apply(a.actor,a.manifest_digest)
    raise Rejected('permission')


def main():
    args=parser().parse_args()
    try:print(json.dumps({'status':'ok',**execute(args)},sort_keys=True));return 0
    except Rejected as exc:print(json.dumps({'status':'rejected','reason':str(exc)}));return 2
    except Exception:print(json.dumps({'status':'unavailable','reason':'local_dependency_or_operation_failed'}));return 3


if __name__=='__main__':raise SystemExit(main())
