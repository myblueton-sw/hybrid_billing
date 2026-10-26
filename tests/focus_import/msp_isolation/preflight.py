import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))

import clickhouse_connect
from botocore.exceptions import ClientError

from modules.focus_import import authority, model, warehouse
from modules.focus_local.database import environment


DATABASE = 'hb32_msp_isolation_20261027'
BUCKET = 'hb32-msp-isolation-20261027'


def inspect():
    env = environment()
    with authority.connect(env) as connection:
        pg_version = connection.execute('SHOW server_version').fetchone()['server_version']
        database_exists = connection.execute('SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=%s) AS v', (DATABASE,)).fetchone()['v']
        can_create = connection.execute('SELECT rolcreatedb OR rolsuper AS v FROM pg_roles WHERE rolname=current_user').fetchone()['v']
        roles = connection.execute("SELECT rolname,rolsuper,rolbypassrls,rolcreaterole,rolcreatedb,rolreplication FROM pg_roles WHERE rolname IN ('hb_focus_ingest','hb_focus_approve') ORDER BY rolname").fetchall()
        memberships = connection.execute("SELECT roleid,member,grantor,admin_option FROM pg_auth_members WHERE roleid IN (SELECT oid FROM pg_roles WHERE rolname IN ('hb_focus_ingest','hb_focus_approve')) OR member IN (SELECT oid FROM pg_roles WHERE rolname IN ('hb_focus_ingest','hb_focus_approve')) ORDER BY roleid,member,grantor").fetchall()
    ch = clickhouse_connect.get_client(host='127.0.0.1', port=18124, username='probe_admin', password=env['CH_ADMIN_PASSWORD'],
        autogenerate_session_id=False, connect_timeout=5, send_receive_timeout=10)
    try:
        ch_version = ch.query('SELECT version()').result_rows[0][0]
        ch_database = ch.query("SELECT count() FROM system.databases WHERE name='hb_focus'").result_rows[0][0]
        ch_users = ch.query("SELECT count() FROM system.users WHERE startsWith(name,'hb_focus_')").result_rows[0][0]
        ch_profiles = ch.query("SELECT count() FROM system.settings_profiles WHERE name='hb_focus_locked'").result_rows[0][0]
        ch_policies = ch.query("SELECT count() FROM system.row_policies WHERE database='hb_focus'").result_rows[0][0]
    finally:
        ch.close()
    s3 = warehouse.object_client(env)
    bucket_exists = True
    try:
        s3.head_bucket(Bucket=BUCKET)
    except ClientError as error:
        if str(error.response['Error']['Code']) not in {'404', 'NoSuchBucket', 'NotFound'}:
            raise RuntimeError('bucket ownership unavailable') from None
        bucket_exists = False
    result = {'postgres_version': pg_version, 'clickhouse_version': ch_version, 'database': DATABASE, 'bucket': BUCKET,
        'pg_database_exists': database_exists, 'pg_can_create_database': can_create,
        'pg_shared_roles': [dict(row) for row in roles], 'pg_shared_memberships': [dict(row) for row in memberships],
        'ch_database_exists': bool(ch_database), 'ch_user_count': ch_users, 'ch_profile_count': ch_profiles,
        'ch_policy_count': ch_policies, 's3_bucket_exists': bucket_exists,
        'default_bindings_sha256': model.digest(model.BINDINGS.read_bytes())}
    valid_roles = {row['rolname'] for row in roles} == {'hb_focus_ingest', 'hb_focus_approve'} and not memberships
    valid_roles = valid_roles and all(not row[name] for row in roles for name in ['rolsuper', 'rolbypassrls', 'rolcreaterole', 'rolcreatedb', 'rolreplication'])
    result['shared_roles_valid'] = valid_roles
    allowed = not database_exists and can_create and valid_roles and not any([ch_database, ch_users, ch_profiles, ch_policies, bucket_exists])
    result['status'] = 'PASS' if allowed else 'BLOCKED_NAMESPACE_OWNERSHIP'
    return result


if __name__ == '__main__':
    try:
        result = inspect()
        target = ROOT / '.local/runs/HB-32-preflight.json'
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(json.dumps(result, sort_keys=True, indent=2) + '\n')
        print(json.dumps(result, sort_keys=True))
        raise SystemExit(0 if result['status'] == 'PASS' else 2)
    except Exception as error:
        print(json.dumps({'status': 'UNAVAILABLE', 'error': type(error).__name__}))
        raise SystemExit(3)
