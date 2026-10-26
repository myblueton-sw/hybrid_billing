import copy
import re
from fractions import Fraction

import boto3
import psycopg
from botocore import UNSIGNED
from botocore.config import Config
from botocore.exceptions import ClientError

from fixture_data import ACTORS, COMPANY, KEYS, encoded
from runtime import BUCKET
from modules.focus_import import authority, collection, warehouse
from modules.focus_import.model import Rejected, digest, tenant_id, validate_config


def denied(action):
    try:
        action()
    except Exception as error:
        return type(error).__name__
    raise AssertionError('forbidden action succeeded')


def engine_denied(action):
    try:
        action()
    except Exception as error:
        match = re.search(r'\((READONLY|SETTING_CONSTRAINT_VIOLATION|ACCESS_DENIED)\)', str(error))
        assert match is not None, 'engine failure is not an authoritative denial: ' + type(error).__name__
        return match[1]
    raise AssertionError('forbidden engine action succeeded')


def rejected(action, reason):
    try:
        action()
    except Rejected as error:
        assert str(error) == reason, (reason, str(error))
        return reason
    raise AssertionError('forbidden product action succeeded')


def postgres_scope(suite):
    own, foreign = tenant_id(KEYS[0]), tenant_id(KEYS[2])
    role = authority.reader_role(own)
    with authority.connect(suite.env, role) as connection:
        rows = connection.execute('SELECT tenant_id FROM hb_focus.tenant').fetchall()
        assert {r['tenant_id'] for r in rows} == {own}
        connection.execute("SELECT set_config('app.tenant_id',%s,false)", (str(foreign),))
        foreign_rows = connection.execute('SELECT * FROM hb_focus.slice WHERE tenant_id=%s', (foreign,)).fetchall()
        assert not foreign_rows
        again = connection.execute('SELECT tenant_id FROM hb_focus.tenant').fetchall()
        assert {r['tenant_id'] for r in again} == {own}
        return {'visible_tenants': len(rows), 'foreign_rows_after_context_spoof': len(foreign_rows), 'connection_reuse_scoped': True}


def postgres_privileges(suite):
    role = authority.reader_role(tenant_id(KEYS[0]))
    attempted = []
    with authority.connect(suite.env, role) as connection:
        for query in ['SELECT * FROM hb_focus.configuration', 'SELECT * FROM hb_focus.batch',
            "UPDATE hb_focus.tenant SET active=false", 'SET ROLE hb_focus_ingest']:
            kind = denied(lambda query=query: connection.execute(query))
            assert kind == 'InsufficientPrivilege', kind
            attempted.append(kind)
        connection.execute('SET row_security=off')
        kind = denied(lambda: connection.execute('SELECT * FROM hb_focus.tenant').fetchall())
        assert kind == 'InsufficientPrivilege', kind
    return {'denied_actions': len(attempted) + 1, 'row_security_off_bypass': False}


def clickhouse_scope(suite):
    role = authority.reader_role(tenant_id(KEYS[0]))
    client = warehouse.client(suite.env, role)
    try:
        count, total = client.query('SELECT count(),sum(billed_cost) FROM hb_focus.fact').result_rows[0]
        assert count == 3 and Fraction(total) == 25
        target = str(tenant_id(KEYS[2]))
        foreign = client.query('SELECT count() FROM hb_focus.fact WHERE tenant_id={t:UUID}', parameters={'t': target}).result_rows[0][0]
        assert foreign == 0
        union = client.query('SELECT tenant_id FROM hb_focus.fact UNION ALL SELECT tenant_id FROM hb_focus.fact WHERE tenant_id={t:UUID}', parameters={'t': target}).result_rows
        assert len(union) == 3 and all(str(row[0]) == str(tenant_id(KEYS[0])) for row in union)
        return {'unfiltered_rows': count, 'unfiltered_billed_cost': str(total), 'foreign_predicate_rows': foreign, 'union_rows': len(union)}
    finally:
        client.close()


def clickhouse_settings(suite):
    role = authority.reader_role(tenant_id(KEYS[0]))
    client = warehouse.client(suite.env, role)
    try:
        errors = [engine_denied(lambda query=query: client.command(query)) for query in [
            'SET join_use_nulls=0', 'SET input_format_null_as_default=1', 'SET serialize_query_plan=1',
            'INSERT INTO hb_focus.fact SELECT * FROM hb_focus.fact WHERE 0']]
        mode, foreign = client.query("SELECT getSetting('readonly'),(SELECT count() FROM hb_focus.fact WHERE tenant_id={t:UUID})",
            parameters={'t': str(tenant_id(KEYS[2]))}, settings={'readonly': 0}).result_rows[0]
        assert int(mode) == 0 and foreign == 0
        return {'denied_locked_settings_and_write': len(errors), 'error_types': errors,
            'global_readonly_not_claimed': True, 'per_query_readonly': int(mode), 'foreign_rows_with_per_query_setting': foreign}
    finally:
        client.close()


def audience_tamper(suite):
    invalid = copy.deepcopy(suite.config)
    invalid['views'][0]['tenant_keys'].append(KEYS[2])
    try:
        validate_config(invalid)
    except Rejected as error:
        assert str(error) == 'view_company'
    else:
        raise AssertionError('cross-company view accepted')
    errors = []
    foreign = tenant_id(KEYS[2])
    missing = tenant_id(KEYS[1])
    with authority.connect(suite.env) as connection:
        connection.execute('INSERT INTO hb_focus.audience_member VALUES (%s,%s)', (COMPANY, foreign))
        try:
            error = rejected(lambda: suite.query(COMPANY), 'view_members_changed')
            errors.append(error)
        finally:
            connection.execute('DELETE FROM hb_focus.audience_member WHERE actor=%s AND tenant_id=%s', (COMPANY, foreign))
        connection.execute('DELETE FROM hb_focus.audience_member WHERE actor=%s AND tenant_id=%s', (COMPANY, missing))
        try:
            error = rejected(lambda: suite.query(COMPANY), 'view_members_changed')
            errors.append(error)
        finally:
            connection.execute('INSERT INTO hb_focus.audience_member VALUES (%s,%s)', (COMPANY, missing))
        connection.execute('UPDATE hb_focus.audience SET active=false WHERE actor=%s', (COMPANY,))
        try:
            error = rejected(lambda: suite.query(COMPANY), 'view_unavailable')
            errors.append(error)
        finally:
            connection.execute('UPDATE hb_focus.audience SET active=true WHERE actor=%s', (COMPANY,))
    suite.check(COMPANY, suite.query(COMPANY))
    return {'invalid_definition_denied': True, 'live_tamper_denials': len(errors), 'valid_view_restored': True}


def unsigned_storage(suite):
    seed_batch = suite.seed['batches'][KEYS[2]]['batch_id']
    key = 'shared-originals/' + seed_batch + '/' + digest(encoded(suite.rows[KEYS[2]])) + '.csv'
    privileged = warehouse.object_client(suite.env)
    assert privileged.head_object(Bucket=BUCKET, Key=key)['ContentLength'] > 0
    anonymous = boto3.client('s3', endpoint_url='http://127.0.0.1:18333', region_name='us-east-1',
        config=Config(signature_version=UNSIGNED, connect_timeout=3, read_timeout=5, s3={'addressing_style': 'path'}))
    results = []
    for operation in [lambda: anonymous.get_object(Bucket=BUCKET, Key=key),
        lambda: anonymous.list_objects_v2(Bucket=BUCKET, Prefix='shared-originals/' + seed_batch + '/', MaxKeys=1)]:
        try:
            response = operation()
            if 'Body' in response:
                response['Body'].close()
            results.append({'denied': False})
        except ClientError as error:
            code = str(error.response['Error']['Code'])
            assert code in {'AccessDenied', '403', '401', 'InvalidAccessKeyId', 'SignatureDoesNotMatch'}, code
            results.append({'denied': True, 'code': code})
    assert all(r['denied'] for r in results), 'unsigned access to isolated bucket succeeded'
    return {'unsigned_operations': results, 'customer_download_endpoint': 'NOT_IMPLEMENTED'}


def engine_metadata(suite):
    role = authority.reader_role(tenant_id(KEYS[0]))
    client = warehouse.client(suite.env, role)
    try:
        try:
            physical = client.query("SELECT sum(rows) FROM system.parts WHERE active AND database='hb_focus' AND table='fact'").result_rows[0][0]
        except Exception as error:
            assert '(ACCESS_DENIED)' in str(error), 'metadata query did not return an authoritative access denial'
            return {'physical_metadata_denied': True, 'error_type': type(error).__name__, 'profile': 'conditional-untrusted-SQL'}
        assert physical == 3, 'untrusted SQL can observe global physical row counts: ' + str(physical)
        return {'physical_rows': physical, 'profile': 'conditional-untrusted-SQL'}
    finally:
        client.close()


def revoked_engine(suite):
    pg_rows = suite.preopened_pg.execute('SELECT coalesce(sum(row_count),0) AS n FROM hb_focus.slice').fetchone()['n']
    ch_rows = suite.preopened_ch.query('SELECT count() FROM hb_focus.fact').result_rows[0][0]
    assert pg_rows == 0 and ch_rows == 0, 'internal database access remains usable after product revocation: PG=' + str(pg_rows) + ', CH=' + str(ch_rows)
    return {'pg_slice_rows_after_revocation': pg_rows, 'ch_fact_rows_after_revocation': ch_rows,
        'pg_connection_opened_before_revocation': True, 'ch_stateless_credentials_rechecked': True, 'profile': 'conditional-untrusted-SQL'}
