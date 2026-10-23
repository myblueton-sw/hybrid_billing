from uuid import UUID, uuid4

from psycopg.types.json import Jsonb

from . import database, storage
from .contract import MAX_BYTES, ROOT, Rejected, admit, canonical, controls, digest, load_registry, parse, rows_digest


class Coordinator:
    def __init__(self, actor, env=None, registry_dir=None):
        self.authority = database.Authority(actor, env)
        self.store = storage.Storage(self.authority.env)
        self.registry_dir = registry_dir

    def close(self):
        self.authority.close()

    def registry(self):
        return load_registry(self.registry_dir) if self.registry_dir else load_registry()

    def scope(self):
        return self.authority.conn.execute('SELECT * FROM hb_local.publication_scope WHERE connection_id=%s FOR UPDATE', (self.authority.scope[2],)).fetchone()

    def check_registry(self, release):
        contract = self.registry().get(release['manifest']['version'])
        if not contract or contract.sha256 != release['registry_digest']:
            raise Rejected('registry_changed')
        if digest(canonical(release['manifest']).encode()) != release['manifest_digest']:
            raise Rejected('manifest_digest')

    def ingest(self, data, version, source_revision):
        if type(source_revision) is not int or not 0 < source_revision < 2**63:
            raise Rejected('source_revision')
        if not isinstance(version, str) or len(version) > 32 or len(data) > MAX_BYTES:
            raise Rejected('input_envelope')
        with self.authority.transaction('ingest') as cap:
            admit(data, cap)
        try:
            return self.ingest_validated(data, version, source_revision)
        except Rejected as exc:
            if str(exc) not in {'unsupported_version','decimal_syntax','decimal_scale','decimal_magnitude','timestamp_format','json_value','enum_value','duplicate_header','missing_header','row_shape_or_limit','field_size','required_value','inapplicable_value','price_product','negative_unit_price','empty_input','input_format','currency'}:
                raise
            return self.quarantine(data, version, source_revision, str(exc))

    def quarantine(self, data, version, source_revision, reason):
        a = self.authority
        installation, workspace, connection = a.scope
        with a.transaction('ingest') as expected_cap:
            admit(data, expected_cap)
            previous = a.conn.execute('SELECT receipt_id,object_key FROM hb_local.quarantine WHERE connection_id=%s AND source_revision=%s AND declared_version=%s AND file_digest=%s', (connection,source_revision,version,digest(data))).fetchone()
            if previous:
                return {'state':'quarantined','reason':reason,'receipt_id':str(previous['receipt_id']),'duplicate':True}
            receipt_id, operation_id = uuid4(), uuid4()
            key = f'{installation}/{workspace}/quarantine/{receipt_id}/{digest(data)}.csv'
            a.conn.execute('INSERT INTO hb_local.quarantine VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)', (installation,workspace,receipt_id,connection,source_revision,version,digest(data),reason,key))
            a.conn.execute("INSERT INTO hb_local.operation(installation_id,workspace_id,operation_id,connection_id,quarantine_id,kind,state) VALUES (%s,%s,%s,%s,%s,'quarantine','pending')", (installation,workspace,operation_id,connection,receipt_id))
        try:
            with a.transaction('ingest',allow_unfinished=True) as current_cap:
                if current_cap != expected_cap:
                    raise Rejected('policy_changed')
                admit(data, current_cap)
                self.store.put(key,data)
                a.conn.execute("UPDATE hb_local.operation SET state='terminal' WHERE operation_id=%s", (operation_id,))
            return {'state':'quarantined','reason':reason,'receipt_id':str(receipt_id),'duplicate':False}
        except BaseException:
            a.mark_unknown(operation_id)
            raise

    def ingest_validated(self, data, version, source_revision):
        contract = self.registry().get(version)
        if not contract:
            raise Rejected('unsupported_version')
        a = self.authority
        installation, workspace, connection = a.scope
        with a.transaction('ingest') as cap:
            expected_cap = cap
            admit(data, cap)
            rows = parse(data, contract, {'account': cap['account'], 'subaccount': cap['subaccount'], 'start': cap['period_start'], 'end': cap['period_end']})
            previous = a.conn.execute('SELECT * FROM hb_local.release WHERE connection_id=%s AND source_revision=%s', (connection, source_revision)).fetchone()
            if previous:
                if previous['file_digest'] != digest(data) or previous['registry_digest'] != contract.sha256:
                    raise Rejected('revision_conflict')
                return self.result(previous, duplicate=True)
            scope = self.scope()
            if scope['release_id']:
                current = a.release(scope['release_id'])
                if source_revision <= current['source_revision']:
                    raise Rejected('stale_source_revision')
            release_id, operation_id = uuid4(), uuid4()
            original_key = f'{installation}/{workspace}/originals/{release_id}/{digest(data)}.csv'
            manifest = {'version': version, 'registry': contract.entry, 'table': contract.table,
                'registry_digest': contract.sha256, 'numeric_profile': 'synthetic-local-v1',
                'installation_id': str(installation), 'workspace_id': str(workspace), 'connection_id': str(connection),
                'source_key': 'monthly-synthetic-cost-usage', 'source_revision': source_revision,
                'file_digest': digest(data), 'file_bytes': len(data), 'original_key': original_key,
                'attempt': 1, 'row_count': len(rows), 'controls': controls(rows, contract.fields), 'rows_digest': rows_digest(rows),
                'predecessor': str(scope['release_id']) if scope['release_id'] else None,
                'scope_revision': scope['revision'], 'policy_revision': cap['policy_revision']}
            sha = digest(canonical(manifest).encode())
            a.conn.execute('INSERT INTO hb_local.release(installation_id,workspace_id,release_id,connection_id,source_revision,file_digest,registry_digest,manifest_digest,manifest,creator,state,original_key) VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)',
                (installation,workspace,release_id,connection,source_revision,digest(data),contract.sha256,sha,Jsonb(manifest),a.actor,'pending',original_key))
            a.conn.execute("INSERT INTO hb_local.operation(installation_id,workspace_id,operation_id,connection_id,release_id,kind,state) VALUES (%s,%s,%s,%s,%s,'load','pending')", (installation,workspace,operation_id,connection,release_id))
        try:
            with a.transaction('ingest', allow_unfinished=True) as cap:
                release = a.release(release_id)
                if cap != expected_cap:
                    raise Rejected('policy_changed')
                self.store.put(original_key, data)
                self.store.insert(a.scope, release_id, contract, rows)
                self.store.verify(a.scope, release)
                a.conn.execute("UPDATE hb_local.release SET state='verified' WHERE release_id=%s", (release_id,))
                a.conn.execute("UPDATE hb_local.operation SET state='terminal' WHERE operation_id=%s AND state='pending'", (operation_id,))
                return self.result(a.release(release_id))
        except BaseException:
            a.mark_unknown(operation_id)
            raise

    @staticmethod
    def result(release, duplicate=False):
        return {'release_id': str(release['release_id']), 'state': release['state'],
            'manifest_digest': release['manifest_digest'], 'version': release['manifest']['version'],
            'registry_digest': release['registry_digest'], 'row_count': release['manifest']['row_count'], 'duplicate': duplicate}

    def confirm(self, release_id, manifest_digest):
        a = self.authority
        with a.transaction('confirm') as cap:
            release = a.release(release_id)
            self.check_registry(release)
            if release['state'] != 'verified' or release['manifest_digest'] != manifest_digest or release['creator'] == a.actor:
                raise Rejected('confirmation_subject')
            if cap['policy_revision'] != release['manifest']['policy_revision']:
                raise Rejected('policy_changed')
            a.conn.execute('INSERT INTO hb_local.confirmation VALUES (%s,%s,%s,%s,%s,%s) ON CONFLICT DO NOTHING',
                (a.scope[0],a.scope[1],release['release_id'],manifest_digest,cap['policy_revision'],a.actor))
            return {'release_id': str(release['release_id']), 'confirmed': True, 'manifest_digest': manifest_digest}

    def publish(self, release_id):
        a = self.authority
        with a.transaction('publish') as cap:
            release = a.release(release_id)
            self.check_registry(release)
            previous = a.conn.execute('SELECT revision FROM hb_local.publication_decision WHERE release_id=%s', (release['release_id'],)).fetchone()
            if previous:
                return {'release_id': str(release['release_id']), 'scope_revision': previous['revision'], 'duplicate': True}
            confirmation = a.conn.execute('SELECT * FROM hb_local.confirmation WHERE release_id=%s', (release['release_id'],)).fetchone()
            if not confirmation or confirmation['manifest_digest'] != release['manifest_digest'] or confirmation['policy_revision'] != cap['policy_revision']:
                raise Rejected('confirmation_required')
            confirmer = a.conn.execute('SELECT rights FROM hb_local.principal WHERE actor=%s', (confirmation['actor'],)).fetchone()
            if not confirmer or 'confirm' not in confirmer['rights'] or confirmation['actor'] == release['creator']:
                raise Rejected('confirmation_authority')
            if cap['policy_revision'] != release['manifest']['policy_revision'] or release['state'] != 'verified':
                raise Rejected('release_ineligible')
            scope = self.scope()
            if scope['revision'] != release['manifest']['scope_revision'] or (str(scope['release_id']) if scope['release_id'] else None) != release['manifest']['predecessor']:
                raise Rejected('predecessor_conflict')
            self.store.verify(a.scope, release)
            revision = scope['revision'] + 1
            a.conn.execute('UPDATE hb_local.publication_scope SET revision=%s,release_id=%s WHERE connection_id=%s', (revision,release['release_id'],a.scope[2]))
            a.conn.execute("UPDATE hb_local.release SET state='sealed' WHERE release_id=%s", (release['release_id'],))
            a.conn.execute('INSERT INTO hb_local.publication_decision VALUES (%s,%s,%s,%s,%s,%s,%s)',
                (a.scope[0],a.scope[1],a.scope[2],revision,release['release_id'],a.actor,release['manifest_digest']))
            a.conn.execute("INSERT INTO hb_local.outbox VALUES (%s,%s,%s,'pending',NULL,NULL)", (a.scope[0],a.scope[1],release['release_id']))
            return {'release_id': str(release['release_id']), 'scope_revision': revision, 'duplicate': False}

    def query(self):
        a = self.authority
        with a.transaction('query'):
            scope = self.scope()
            if not scope['release_id']:
                return {'release_id': None, 'groups': {}, 'row_count': 0}
            release = a.release(scope['release_id'])
            if release['state'] != 'sealed':
                raise Rejected('publication_ineligible')
            groups = self.store.aggregates(a.scope, release, reader=True)
            if groups != release['manifest']['controls']:
                raise Rejected('query_controls')
            return {'release_id': str(release['release_id']), 'version': release['manifest']['version'],
                'registry_digest': release['registry_digest'], 'scope_revision': scope['revision'],
                'row_count': release['manifest']['row_count'], 'coefficient_scale': 6, 'groups': groups}

    def export(self, release_id):
        a = self.authority
        installation, workspace, connection = a.scope
        with a.transaction('export'):
            release = a.release(release_id)
            if release['state'] != 'sealed':
                raise Rejected('export_ineligible')
            effect = a.conn.execute('SELECT * FROM hb_local.outbox WHERE release_id=%s', (release['release_id'],)).fetchone()
            if effect['state'] == 'completed':
                return {'object_key': effect['object_key'], 'sha256': effect['object_digest'], 'duplicate': True}
            operation_id = uuid4()
            a.conn.execute("INSERT INTO hb_local.operation(installation_id,workspace_id,operation_id,connection_id,release_id,kind,state) VALUES (%s,%s,%s,%s,%s,'export','pending')", (installation,workspace,operation_id,connection,release['release_id']))
        try:
            with a.transaction('export', allow_unfinished=True):
                release = a.release(release_id)
                key, sha = self.store.export(a.scope, release)
                a.conn.execute("UPDATE hb_local.outbox SET state='completed',object_key=%s,object_digest=%s WHERE release_id=%s", (key,sha,release['release_id']))
                a.conn.execute("UPDATE hb_local.operation SET state='terminal' WHERE operation_id=%s", (operation_id,))
                return {'object_key': key, 'sha256': sha, 'duplicate': False}
        except BaseException:
            a.mark_unknown(operation_id)
            raise


def initialize(env=None, registry_dir=None):
    env = env or database.environment()
    registry = load_registry(registry_dir) if registry_dir else load_registry()
    database.initialize(env)
    storage.initialize(env, registry)
    return {'initialized': True, 'versions': sorted(registry), 'profile': 'synthetic-local-v1'}
