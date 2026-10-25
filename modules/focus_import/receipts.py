import base64
import re
from datetime import timedelta

from . import archive
from .model import Rejected, canonical, digest, tenant_id
from .temporal import eligible, utc


def bare_descriptor(descriptor):
    return {k:v for k,v in descriptor.items() if k not in ['replacement_plan','reopen_plan']}


def safe_reason(reason):
    return reason if isinstance(reason, str) and re.fullmatch('[a-z][a-z0-9_]{0,79}', reason) else 'source_validation_failed'


def record(j, data, version, descriptor, reason, config, connection=None, admitted=False, plans=()):
    from .collection import now
    receipts = j.state.setdefault('receipts', {})
    if len(receipts) >= 10000:
        raise Rejected('receipt_bound')
    body = {'file_digest': digest(data), 'version': version if isinstance(version, str) and len(version) <= 16 else 'unknown',
        'connection_id': connection['id'] if connection else None, 'reason': safe_reason(reason), 'admitted': admitted,
        'descriptor_digest':digest(canonical(bare_descriptor(descriptor)).encode()) if admitted else None}
    key = digest(canonical(body).encode())
    if key in receipts:
        result={'receipt_id':key,'state':receipts[key]['state'],'duplicate':True,'reason':body['reason']}
        for plan in plans:consume(j,plan,result)
        if plans:j.save(j.state)
        return result
    entry = {**body, 'state': 'quarantined' if admitted else 'rejected', 'received_at': now(),
        'descriptor': descriptor if admitted else None, 'key': None, 'holds': {}, 'attempts': 0, 'replaced_by': None}
    entry['lineage_status']='open'
    entry['root_receipt']=next((j.state['receipts'][p['body']['receipt_id']].get('root_receipt',p['body']['receipt_id']) for p in plans if p['kind']=='replacement'),key)
    if entry['root_receipt']!=key:
        entry['origin']=receipts[entry['root_receipt']]['origin']
    if admitted:
        from .intake import input_format
        from .connections import owners
        domains = owners(config)
        tenants = {domains[canonical(d)]['key'] for d in connection['domains']}
        entry['tenant_keys'] = sorted(tenants)
        entry.setdefault('origin',entry['received_at'])
        entry['key'] = 'quarantine/' + j.head['installation'] + '/' + key + '/' + digest(data) + '.' + input_format(data)
        receipts[key] = entry
        for plan in plans:consume(j,plan,{'receipt_id':key})
        with j.effect('quarantine', key):
            archive.put(j.s3, entry['key'], data)
    else:
        receipts[key] = entry
        j.save(j.state)
    return {'receipt_id': key, 'state': entry['state'], 'duplicate': False, 'reason': body['reason']}


def prepare(actor, kind, body):
    from .collection import access, now
    if actor != 'operator' or kind not in ['reopen', 'replacement']:
        raise Rejected('permission')
    with access(actor, 'register') as (j, c, conn):
        if set(body) != {'connection_id', 'descriptor', 'reason', 'expires_at', 'receipt_id'}:
            raise Rejected('intake_plan')
        from .connections import get, identity, token
        from .temporal import checked_window,DELIVERY_FIELDS
        descriptor=body['descriptor']
        if not isinstance(descriptor,dict) or set(descriptor)-{'connection_id','zero_domains'}!=DELIVERY_FIELDS or type(descriptor['revision']) is not int or descriptor['revision']<1 or not isinstance(descriptor['file_digest'],str) or not re.fullmatch('[a-f0-9]{64}',descriptor['file_digest']):raise Rejected('intake_plan_descriptor')
        checked_window(descriptor['window'])
        connection = get(j.state, body['descriptor'])
        if not connection or connection['id'] != body['connection_id'] or not token(body['reason']):
            raise Rejected('intake_plan')
        current = utc(now())
        expiry = utc(body['expires_at'])
        if not current < expiry <= current + timedelta(days=7):
            raise Rejected('intake_plan_expiry')
        key = identity(connection, body['descriptor']['window'])
        target = j.state['windows'].get(key)
        predecessor = target['current'] if target else None
        if kind == 'replacement':
            receipt = j.state.get('receipts', {}).get(body['receipt_id'])
            if not receipt or receipt['state'] not in ['quarantined', 'replacement-requested'] or receipt['connection_id'] != connection['id']:
                raise Rejected('replacement_receipt')
            root=j.state['receipts'][receipt.get('root_receipt',body['receipt_id'])]
            previous_plan=j.state['plans'].get(receipt.get('replacement_plan'))
            if previous_plan and previous_plan['state']=='prepared' and utc(now())<utc(previous_plan['body']['expires_at']):raise Rejected('replacement_pending')
            if previous_plan and previous_plan['state']=='prepared':previous_plan['state']='expired'
            if root['state']=='escalated' or root['attempts'] >= connection['max_replacements']:
                receipt['state'] = 'escalated'; j.save(j.state)
                return {'state': 'escalated', 'receipt_id': body['receipt_id']}
            if receipt['descriptor']['window'] != body['descriptor']['window'] or receipt['version'] != connection['version'] or receipt['file_digest'] == body['descriptor']['file_digest'] and bare_descriptor(receipt['descriptor'])==bare_descriptor(body['descriptor']):
                raise Rejected('replacement_scope')
        elif body['receipt_id'] is not None or predecessor is None:
            raise Rejected('reopen_predecessor')
        payload = {**body, 'predecessor': predecessor, 'connection_digest': digest(canonical(connection).encode()),
            'predecessor_digest': j.state['receipts'][body['receipt_id']]['file_digest'] if kind == 'replacement' else None,
            'predecessor_descriptor_digest':digest(canonical(bare_descriptor(j.state['receipts'][body['receipt_id']]['descriptor'])).encode()) if kind=='replacement' else None}
        sha = digest(canonical(payload).encode())
        j.state['plans'][sha] = {'kind': kind, 'body': payload, 'requester': actor, 'approver': None, 'state': 'prepared'}
        if kind == 'replacement':
            j.state['receipts'][body['receipt_id']]['state'] = 'replacement-requested'
            j.state['receipts'][body['receipt_id']]['replacement_plan']=sha
        j.save(j.state)
        return {'state': 'prepared', 'manifest_digest': sha}


def authorize(j, sha, kind, connection, descriptor, predecessor):
    from .collection import now
    plan = j.state['plans'].get(sha)
    if not plan or plan['kind'] != kind or plan['state'] != 'prepared' or plan['approver'] != 'lifecycle-approver' or digest(canonical(plan['body']).encode()) != sha:
        raise Rejected('intake_approval_required')
    b = plan['body']
    bare = {k: v for k, v in descriptor.items() if k not in ['replacement_plan', 'reopen_plan']}
    if b['descriptor'] != bare or b['predecessor'] != predecessor or utc(now()) >= utc(b['expires_at']) or b['connection_digest'] != digest(canonical(connection).encode()):
        raise Rejected('intake_approval_changed')
    if kind == 'replacement':
        receipt = j.state['receipts'].get(b['receipt_id'])
        if not receipt or receipt['state'] != 'replacement-requested' or receipt['file_digest'] != b['predecessor_digest'] or receipt.get('replacement_plan')!=sha or digest(canonical(bare_descriptor(receipt['descriptor'])).encode())!=b['predecessor_descriptor_digest']:
            raise Rejected('replacement_consumed')
        root=j.state['receipts'][receipt.get('root_receipt',b['receipt_id'])]
        if root['state']=='escalated' or root['attempts']>=connection['max_replacements']:
            raise Rejected('replacement_consumed')
    return plan


def consume(j, plan, result):
    plan['state'] = 'executed'
    if plan['kind'] == 'replacement':
        predecessor=plan['body']['receipt_id']
        q=j.state['receipts'][predecessor]
        root_id=q.get('root_receipt',predecessor)
        root=j.state['receipts'][root_id]
        root['attempts']+=1
        q['replaced_by']=result.get('receipt_id',result.get('delivery_id'))
        terminal='resolved' if result.get('delivery_id') else 'escalated' if root['attempts']>=j.state['connections'][q['connection_id']]['max_replacements'] else None
        if terminal:
            root['lineage_status']=terminal
            for key,item in j.state['receipts'].items():
                if item.get('root_receipt',key)==root_id and item['state']!='deleted':item['state']=terminal
        else:q['state']='quarantined'


def managed(j):
    return [q for q in j.state.get('receipts', {}).values() if q['key'] and q['state'] != 'deleted']


def dates_for(j,q,at):
    return [eligible(q['origin'] if j.state['policies'][key]['basis']=='received_at' else q['descriptor']['window']['end'],j.state['policies'][key],at) for key in q['tenant_keys']]


def backup_parts(j, at):
    parts = {}; deadlines = {}
    for q in managed(j):
        horizon=max(d['backup_until'] for d in dates_for(j,q,at))
        if utc(at)>=utc(horizon):continue
        deadlines[q['key']]=horizon
        data = j.s3.get_object(Bucket=archive.BUCKET, Key=q['key'])['Body'].read()
        if digest(data) != q['file_digest']:
            raise Rejected('quarantine_digest')
        parts[q['key']] = base64.b64encode(data).decode()
    return parts, deadlines


def restore_parts(j, body):
    for q in managed(j):
        try:
            data = j.s3.get_object(Bucket=archive.BUCKET, Key=q['key'])['Body'].read()
        except Exception as exc:
            if getattr(exc, 'response', {}).get('Error', {}).get('Code') not in ['404', 'NoSuchKey', 'NotFound']:
                raise
            if q['key'] not in body.get('quarantine', {}):
                raise Rejected('backup_missing_quarantine')
            data = base64.b64decode(body['quarantine'][q['key']], validate=True)
            if digest(data) != q['file_digest']:
                raise Rejected('quarantine_digest')
            archive.put(j.s3, q['key'], data)
        if digest(data) != q['file_digest']:
            raise Rejected('quarantine_digest')


def retention(actor, receipt_id, at=None, hold=None, release=False):
    from .collection import access, now
    with access(actor, 'register') as (j, c, conn):
        q = j.state.get('receipts', {}).get(receipt_id)
        if not q or not q['key'] or q['state'] == 'deleted':
            raise Rejected('receipt_unavailable')
        if hold is not None:
            from .connections import token
            if not token(hold):
                raise Rejected('hold_identity')
            if release: q['holds'].pop(hold, None)
            else: q['holds'][hold] = now()
            j.save(j.state)
            return {'state': 'held' if q['holds'] else 'unheld'}
        current = at or now()
        dates = dates_for(j,q,current)
        return {'receipt_id': receipt_id, 'eligible': not q['holds'] and all(current >= d['delete_after'] for d in dates),
            'holds': sorted(q['holds']), 'state': q['state'], 'physical_erasure': False,
            'retained_managed_backups': [k for k, b in j.state['backups'].items() if b['state'] == 'retained']}


def deletion_plan(actor, receipt_id, at=None):
    from .collection import access, now
    with access(actor, 'register') as (j, c, conn):
        q = j.state.get('receipts', {}).get(receipt_id)
        if not q or not q['key'] or q['state'] == 'deleted' or q['holds']:
            raise Rejected('quarantine_deletion_denied')
        current = utc(at or now()).isoformat()
        if any(current < d['delete_after'] for d in dates_for(j, q, current)):
            raise Rejected('quarantine_retention')
        body = {'receipt_id': receipt_id, 'receipt_digest': digest(canonical(q).encode()),
            'policy_digest': digest(canonical(j.state['policies']).encode()), 'at': current}
        sha = digest(canonical(body).encode())
        j.state['plans'][sha] = {'kind': 'receipt-delete', 'body': body, 'requester': actor, 'approver': None, 'state': 'prepared'}
        j.save(j.state)
        return {'state': 'prepared', 'manifest_digest': sha, 'managed_backup_residual': True}


def delete(actor, sha, at=None):
    from .collection import access, now
    from .lifecycle import delete_key
    with access(actor, 'register') as (j, c, conn):
        plan = j.state['plans'].get(sha)
        if not plan or plan['kind'] != 'receipt-delete' or plan['state'] != 'prepared' or plan['approver'] != 'lifecycle-approver' or digest(canonical(plan['body']).encode()) != sha:
            raise Rejected('approval_required')
        b = plan['body']; q = j.state['receipts'][b['receipt_id']]
        current = utc(at or now()).isoformat()
        if current < b['at'] or q['holds'] or digest(canonical(q).encode()) != b['receipt_digest'] or digest(canonical(j.state['policies']).encode()) != b['policy_digest']:
            raise Rejected('quarantine_deletion_changed')
        with j.effect('receipt-delete', b['receipt_id']):
            q['state'] = 'deleted'; q['intake_closed']=True; plan['state'] = 'executed'
            delete_key(j.s3, q['key'])
        return {'state': 'deleted', 'managed_backups_may_retain_copy': True, 'physical_erasure': False}
