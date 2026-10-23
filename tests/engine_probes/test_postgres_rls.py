import secrets

import psycopg
import pytest


@pytest.fixture(scope="module")
def app_conn(pg_admin):
    password = secrets.token_hex(16)
    with pg_admin.cursor() as cur:
        cur.execute("DROP TABLE IF EXISTS rls_probe")
        cur.execute("DROP ROLE IF EXISTS hb_app")
        cur.execute(f"CREATE ROLE hb_app LOGIN PASSWORD '{password}' NOSUPERUSER NOBYPASSRLS")
        cur.execute("CREATE TABLE rls_probe (tenant text NOT NULL, v int NOT NULL)")
        cur.execute("INSERT INTO rls_probe VALUES ('A', 1), ('A', 2), ('B', 3)")
        cur.execute("ALTER TABLE rls_probe ENABLE ROW LEVEL SECURITY")
        cur.execute("ALTER TABLE rls_probe FORCE ROW LEVEL SECURITY")
        cur.execute("CREATE POLICY scope_only ON rls_probe USING (tenant = current_setting('hb.tenant', true))")
        cur.execute("GRANT SELECT ON rls_probe TO hb_app")
    conn = psycopg.connect(host="127.0.0.1", port=15432, dbname="hb", user="hb_app", password=password)
    yield conn
    conn.close()


def test_ep13_role_attributes(app_conn, record):
    with app_conn.cursor() as cur:
        cur.execute("SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user")
        superuser, bypass = cur.fetchone()
        cur.execute("SELECT tableowner = current_user FROM pg_tables WHERE tablename = 'rls_probe'")
        owner = cur.fetchone()[0]
    app_conn.rollback()
    outcome = "CONFIRMS" if not (superuser or bypass or owner) else "CONTRADICTS"
    record("EP-13", outcome, "application role is neither superuser, BYPASSRLS nor table owner", f"superuser={superuser} bypassrls={bypass} owner={owner}")


def test_ep13_scope_lifecycle(app_conn, record):
    observations = {}
    with app_conn.cursor() as cur:
        cur.execute("SELECT count(*) FROM rls_probe")
        observations["no_scope"] = cur.fetchone()[0]
    app_conn.commit()
    with app_conn.cursor() as cur:
        cur.execute("SET LOCAL hb.tenant = 'A'")
        cur.execute("SELECT count(*) FROM rls_probe")
        observations["scope_a"] = cur.fetchone()[0]
    app_conn.commit()
    with app_conn.cursor() as cur:
        cur.execute("SELECT count(*), current_setting('hb.tenant', true) FROM rls_probe")
        count, setting = cur.fetchone()
        observations["after_commit"] = count
        observations["setting_after_commit"] = setting
    app_conn.commit()
    ok = observations["no_scope"] == 0 and observations["scope_a"] == 2 and observations["after_commit"] == 0 and observations["setting_after_commit"] in ("", None)
    record("EP-13", "CONFIRMS" if ok else "CONTRADICTS", "no scope gives zero rows; SET LOCAL scope gives only that tenant; scope does not survive COMMIT", f"{observations}")
