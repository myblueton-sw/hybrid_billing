import fcntl
import json
import os
from contextlib import contextmanager
from functools import wraps
from pathlib import Path
from uuid import UUID, uuid4

from psycopg import sql

from modules.focus_local.storage import object_client
from . import authority
from .model import ROOT, Rejected, canonical, digest
from .warehouse import BUCKET


DIRECTORY=ROOT/'.local/focus-lifecycle'


def durable(path,data):
    temporary=path.with_suffix('.tmp')
    with temporary.open('wb') as stream:
        os.chmod(temporary,0o600)
        stream.write(data);stream.flush();os.fsync(stream.fileno())
    temporary.replace(path)
    fd=os.open(path.parent,os.O_RDONLY)
    try:os.fsync(fd)
    finally:os.close(fd)


@contextmanager
def locked():
    DIRECTORY.parent.mkdir(parents=True,exist_ok=True)
    with (DIRECTORY.parent/'focus-coordinator.lock').open('a') as stream:
        fcntl.flock(stream,fcntl.LOCK_EX)
        try:yield
        finally:fcntl.flock(stream,fcntl.LOCK_UN)


def checkpoint(conn):
    exists=conn.execute("SELECT to_regclass('hb_focus.control_checkpoint') AS value").fetchone()['value']
    return conn.execute('SELECT * FROM hb_focus.control_checkpoint WHERE id=1').fetchone() if exists else None


class Journal:
    def __init__(self,env,conn):
        self.env=env;self.conn=conn;self.s3=object_client(env)
        self.state=None;self.head=None

    def load(self,repair=False):
        saved=checkpoint(self.conn)
        if not (DIRECTORY/'head.json').exists():
            if saved:raise Rejected('control_journal_missing')
            if DIRECTORY.exists() and any(DIRECTORY.iterdir()):raise Rejected('control_activation_incomplete')
            return None
        head=json.loads((DIRECTORY/'head.json').read_text())
        anchor_path=DIRECTORY/'authority.anchor'
        if not anchor_path.exists():raise Rejected('control_anchor_missing')
        anchor=json.loads(anchor_path.read_text())
        anchor_data=canonical(anchor).encode()
        anchor_key=f"anchors/{anchor['installation']}/{anchor['sequence']:012d}-{digest(anchor_data)}.json"
        if self.s3.get_object(Bucket=BUCKET,Key=anchor_key)['Body'].read()!=anchor_data:raise Rejected('control_anchor_mirror')
        if anchor['installation']!=head['installation'] or head['sequence']<anchor['sequence']:raise Rejected('control_rollback')
        if anchor['state']=='retirement_pending':raise Rejected('retirement_unresolved')
        if set(head)!={'installation','sequence','digest','first_sequence','previous_anchor'} or type(head['sequence']) is not int or head['sequence']<1:raise Rejected('control_head')
        UUID(head['installation'])
        files={p.name for p in DIRECTORY.glob('*.json') if p.name!='head.json'}
        if files!={f'{i:012d}.json' for i in range(head['first_sequence'],head['sequence']+1)}:raise Rejected('control_uncommitted_intent')
        previous=head['previous_anchor'];state=None
        for sequence in range(head['first_sequence'],head['sequence']+1):
            data=(DIRECTORY/f'{sequence:012d}.json').read_bytes()
            entry=json.loads(data)
            if entry['sequence']!=sequence or entry['installation']!=head['installation'] or entry['previous']!=previous:raise Rejected('control_chain')
            sha=digest(data)
            mirrored=self.s3.get_object(Bucket=BUCKET,Key=f"control/{head['installation']}/{sequence:012d}-{sha}.json")['Body'].read()
            if mirrored!=data:raise Rejected('control_mirror')
            previous=sha;state=entry['state']
        if previous!=head['digest']:raise Rejected('control_head')
        prefix=f"control/{head['installation']}/"
        keys=[];token=None
        while True:
            response=self.s3.list_objects_v2(Bucket=BUCKET,Prefix=prefix,**({'ContinuationToken':token} if token else {}))
            keys.extend(x['Key'] for x in response.get('Contents',[]))
            if not response.get('IsTruncated'):break
            token=response['NextContinuationToken']
        expected={prefix+f"{i:012d}-"+digest((DIRECTORY/f'{i:012d}.json').read_bytes())+'.json' for i in range(head['first_sequence'],head['sequence']+1)}
        if set(keys)!=expected:raise Rejected('control_mirror_head')
        if saved and (saved['sequence']>head['sequence'] or saved['sequence']==head['sequence'] and saved['digest']!=head['digest']):raise Rejected('control_rollback')
        if not repair and (not saved or str(saved['installation'])!=head['installation'] or saved['sequence']!=head['sequence'] or saved['digest']!=head['digest']):raise Rejected('control_checkpoint')
        if saved and str(saved['installation'])!=head['installation']:raise Rejected('installation_mismatch')
        self.head=head;self.state=state
        return state

    def save(self,state):
        installation=self.head['installation'] if self.head else str(uuid4())
        sequence=self.head['sequence']+1 if self.head else 1
        entry={'installation':installation,'sequence':sequence,'previous':self.head['digest'] if self.head else None,'state':state}
        data=canonical(entry).encode();sha=digest(data)
        DIRECTORY.mkdir(mode=0o700,parents=True,exist_ok=True)
        path=DIRECTORY/f'{sequence:012d}.json'
        if path.exists():raise Rejected('control_uncommitted_intent')
        durable(path,data)
        key=f'control/{installation}/{sequence:012d}-{sha}.json'
        self.s3.put_object(Bucket=BUCKET,Key=key,Body=data,IfNoneMatch='*')
        if self.s3.get_object(Bucket=BUCKET,Key=key)['Body'].read()!=data:raise Rejected('control_mirror')
        head={'installation':installation,'sequence':sequence,'digest':sha,'first_sequence':self.head['first_sequence'] if self.head else 1,'previous_anchor':self.head['previous_anchor'] if self.head else None}
        if not self.head:self.anchor('active',head)
        durable(DIRECTORY/'head.json',canonical(head).encode())
        if self.head:
            changed=self.conn.execute('UPDATE hb_focus.control_checkpoint SET sequence=%s,digest=%s WHERE id=1 AND installation=%s',(sequence,sha,UUID(installation))).rowcount
            if changed!=1:raise Rejected('control_checkpoint')
        else:self.conn.execute('INSERT INTO hb_focus.control_checkpoint VALUES (1,%s,%s,%s)',(UUID(installation),sequence,sha))
        self.head=head;self.state=state

    def anchor(self,status,head=None):
        h=head or self.head
        value={'installation':h['installation'],'sequence':h['sequence'],'digest':h['digest'],'state':status}
        data=canonical(value).encode();key=f"anchors/{h['installation']}/{h['sequence']:012d}-{digest(data)}.json"
        self.s3.put_object(Bucket=BUCKET,Key=key,Body=data,IfNoneMatch='*')
        if self.s3.get_object(Bucket=BUCKET,Key=key)['Body'].read()!=data:raise Rejected('control_anchor_mirror')
        durable(DIRECTORY/'authority.anchor',data)

    def ready(self):
        state=self.load()
        if state and state.get('retired'):raise Rejected('installation_retired')
        if state and state.get('operation'):raise Rejected('operation_unresolved')
        return state

    @contextmanager
    def effect(self,kind,target):
        if self.state.get('operation'):raise Rejected('operation_unresolved')
        self.state['operation']={'id':str(uuid4()),'kind':kind,'target':target}
        self.save(self.state)
        yield self.state
        self.state['operation']=None
        self.save(self.state)


def guarded(function):
    @wraps(function)
    def call(self,*args,**kwargs):
        if getattr(self,'_control_nested',False):return function(self,*args,**kwargs)
        with locked():
            journal=Journal(self.a.env,self.a.conn)
            journal.ready()
            if journal.state and self.a.actor in journal.state['revoked_actors']:raise Rejected('reader_revoked')
            if journal.state and function.__name__ in ['ingest','confirm','publish']:raise Rejected('lifecycle_registered_delivery_required')
            self._journal=journal;self._control_nested=True
            try:return function(self,*args,**kwargs)
            finally:self._control_nested=False;self._journal=None
    return call


def provision(conn,roles):
    for role in roles:
        ident=sql.Identifier(role)
        conn.execute(sql.SQL('GRANT SELECT ON hb_focus.control_checkpoint TO {}').format(ident))
        conn.execute(sql.SQL('DROP POLICY IF EXISTS {} ON hb_focus.control_checkpoint').format(sql.Identifier('p_'+role)))
        conn.execute(sql.SQL('CREATE POLICY {} ON hb_focus.control_checkpoint TO {} USING(true) WITH CHECK(true)').format(sql.Identifier('p_'+role),ident))
        if role in [authority.IMPORTER,authority.APPROVER]:
            conn.execute(sql.SQL('GRANT UPDATE(sequence,digest) ON hb_focus.control_checkpoint TO {}').format(ident))


def local_denial():
    path=DIRECTORY/'authority.anchor'
    if path.exists():
        status=json.loads(path.read_text())['state']
        if status=='retirement_pending':raise Rejected('retirement_unresolved')
        if status=='retired':raise Rejected('installation_retired')
