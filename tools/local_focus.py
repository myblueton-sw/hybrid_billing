import argparse
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from modules.focus_local.contract import MAX_BYTES, Rejected, load_registry
from modules.focus_local.pipeline import Coordinator, initialize


def main():
    parser = argparse.ArgumentParser(description='HB-27 synthetic local FOCUS coordinator')
    sub = parser.add_subparsers(dest='command', required=True)
    sub.add_parser('init')
    sub.add_parser('versions')
    demo = sub.add_parser('demo')
    demo.add_argument('--file', type=Path, required=True)
    demo.add_argument('--version', required=True)
    demo.add_argument('--revision', type=int, default=1)
    for name in ('ingest', 'confirm', 'publish', 'query', 'export'):
        command = sub.add_parser(name)
        command.add_argument('--actor', required=True)
        if name == 'ingest':
            command.add_argument('--file', type=Path, required=True)
            command.add_argument('--version', required=True)
            command.add_argument('--revision', type=int, required=True)
        if name in ('confirm', 'publish', 'export'):
            command.add_argument('--release', required=True)
        if name == 'confirm':
            command.add_argument('--manifest-digest', required=True)
    args = parser.parse_args()
    coordinator = None
    try:
        if args.command == 'init':
            result = initialize()
        elif args.command == 'versions':
            result = {'versions': sorted(load_registry())}
        elif args.command == 'demo':
            result = run_demo(args.file,args.version,args.revision)
        else:
            coordinator = Coordinator(args.actor)
            if args.command == 'ingest':
                with args.file.open('rb') as stream:
                    data = stream.read(MAX_BYTES + 1)
                result = coordinator.ingest(data, args.version, args.revision)
            elif args.command == 'confirm':
                result = coordinator.confirm(args.release, args.manifest_digest)
            elif args.command == 'publish':
                result = coordinator.publish(args.release)
            elif args.command == 'export':
                result = coordinator.export(args.release)
            else:
                result = coordinator.query()
        quarantined = result.get('state') == 'quarantined'
        print(json.dumps({'status': 'quarantined' if quarantined else 'ok', **result}, sort_keys=True))
        return 2 if quarantined else 0
    except Rejected as exc:
        print(json.dumps({'status': 'rejected', 'reason': str(exc)}))
        return 2
    except Exception:
        print(json.dumps({'status': 'unavailable', 'reason': 'local_dependency_or_operation_failed'}))
        return 3
    finally:
        if coordinator:
            coordinator.close()


def run_demo(path,version,revision):
    initialize()
    loader=Coordinator('loader-a')
    approver=Coordinator('approver-a')
    reader=Coordinator('reader-a')
    try:
        with path.open('rb') as stream:data=stream.read(MAX_BYTES+1)
        item=loader.ingest(data,version,revision)
        if item['state']=='quarantined':return item
        if item['state']!='sealed':approver.confirm(item['release_id'],item['manifest_digest'])
        approver.publish(item['release_id'])
        result=reader.query()
        if result['release_id']!=item['release_id']:raise Rejected('demo_not_current')
        exported=approver.export(item['release_id'])
        return {'synthetic_demo':True,'state':'sealed',**result,'export_sha256':exported['sha256']}
    finally:
        loader.close();approver.close();reader.close()


if __name__ == '__main__':
    raise SystemExit(main())
