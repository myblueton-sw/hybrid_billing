from datetime import datetime, timezone
from uuid import UUID

from .contract import MAX_BYTES, MAX_ROWS, Rejected, load_registry
from .database import Authority


class Connections:
    def __init__(self, actor, env=None):
        self.authority = Authority(actor, env)

    def close(self):
        self.authority.close()

    def detail(self, connection_id=None):
        a = self.authority
        with a.metadata_inspection() as cap:
            if connection_id is not None:
                try:
                    if UUID(connection_id) != a.scope[2]:
                        raise Rejected('connection_unavailable')
                except (ValueError, TypeError):
                    raise Rejected('connection_unavailable') from None
            latest = a.conn.execute("SELECT release_id,source_revision,state,created_at,manifest_digest,manifest->>'version' AS version FROM hb_local.release WHERE connection_id=%s ORDER BY created_at DESC,release_id DESC LIMIT 1", (a.scope[2],)).fetchone()
            holds = a.conn.execute("SELECT EXISTS(SELECT 1 FROM hb_local.quarantine WHERE connection_id=%s) AS quarantine,EXISTS(SELECT 1 FROM hb_local.operation WHERE connection_id=%s AND state<>'terminal') AS unresolved", (a.scope[2], a.scope[2])).fetchone()
            contracts = [{'version': version, 'revision': str(contract.entry['revision']), 'digest': contract.sha256} for version, contract in sorted(load_registry().items())]
            return {'classification': 'local-connection-metadata', 'connection': str(a.scope[2]),
                'workspace': str(a.scope[1]), 'account': cap['account'], 'subaccount': cap['subaccount'],
                'period_start': cap['period_start'].isoformat(), 'period_end': cap['period_end'].isoformat(),
                'policy_revision': str(cap['policy_revision']), 'observed_at': datetime.now(timezone.utc).isoformat(),
                'contracts': contracts, 'max_bytes': MAX_BYTES, 'max_rows': MAX_ROWS,
                'completeness': 'unknown', 'provider_authentication': 'unconnected',
                'provider_reads': 'unconnected', 'invoices': 'unconnected',
                'holds': holds, 'actions': ['imports'] if a.actor.startswith('loader-') else ['source-data', 'reconciliation'],
                'latest_candidate': {'release_id': str(latest['release_id']), 'source_revision': str(latest['source_revision']),
                    'state': latest['state'], 'created_at': latest['created_at'].isoformat(),
                    'manifest_digest': latest['manifest_digest'], 'version': latest['version']} if latest else None}
