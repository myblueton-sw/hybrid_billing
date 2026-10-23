import clickhouse_connect
import pytest

from conftest import POLICY_PASSWORD
from probe_support import ch_code, first_line

UNMATCHED_POLICY_TEXT = "has row policies, but none of them"


def _prepare(client):
    client.command("CREATE DATABASE IF NOT EXISTS hb")
    for table in ("facts", "facts_copy"):
        client.command(f"DROP TABLE IF EXISTS hb.{table}")
        client.command(f"CREATE TABLE hb.{table} (tenant String, release_id String, attempt UInt32, cost Decimal(18, 6)) ENGINE = MergeTree PARTITION BY tenant ORDER BY (tenant, release_id, attempt)")
    client.command("INSERT INTO hb.facts VALUES ('A', 'r1', 1, 10), ('A', 'r1', 1, 20), ('B', 'r2', 1, 99)")
    for user in ("hb_nopolicy", "hb_tenant_a"):
        client.command(f"DROP USER IF EXISTS {user}")
        client.command(f"CREATE USER {user} IDENTIFIED WITH sha256_password BY '{POLICY_PASSWORD}'")
        client.command(f"GRANT SELECT ON hb.facts TO {user}")
    client.command("DROP ROW POLICY IF EXISTS tenant_a ON hb.facts")
    client.command("CREATE ROW POLICY tenant_a ON hb.facts AS PERMISSIVE FOR SELECT USING tenant = 'A' TO hb_tenant_a")


def _connect(port, user):
    return clickhouse_connect.get_client(host="127.0.0.1", port=port, username=user, password=POLICY_PASSWORD, autogenerate_session_id=False)


@pytest.fixture(scope="module", autouse=True)
def access_setup(admin, defaults_admin):
    _prepare(admin)
    _prepare(defaults_admin)
    admin.command("GRANT SELECT ON hb.facts TO hb_reader")
    admin.command("DROP ROW POLICY IF EXISTS reader_a ON hb.facts")
    admin.command("CREATE ROW POLICY reader_a ON hb.facts AS PERMISSIVE FOR SELECT USING tenant = 'A' TO hb_reader")


def test_ep07_effective_server_settings(admin, record):
    rows = admin.query("SELECT name, value FROM system.server_settings WHERE name ILIKE '%row_polic%'").result_rows
    observed = {r[0]: r[1] for r in rows}
    throw = observed.get("access_control_improvements.throw_on_unmatched_row_policies")
    can_read = observed.get("access_control_improvements.users_without_row_policies_can_read_rows")
    if not observed:
        outcome = "NOT_RUN"
    else:
        outcome = "CONFIRMS" if throw in ("1", "true") and can_read in ("0", "false") else "CONTRADICTS"
    record("EP-07", outcome, "configured access_control_improvements values are in effect", f"{observed or 'no matching server settings exposed'}")


@pytest.mark.parametrize("server,port", [("configured", 18123), ("defaults", 18124)])
def test_ep07_user_without_policy(record, server, port):
    try:
        rows = _connect(port, "hb_nopolicy").query("SELECT tenant, count() FROM hb.facts GROUP BY tenant ORDER BY tenant").result_rows
        result, observed = ("all" if {r[0] for r in rows} == {"A", "B"} else "partial"), f"rows {rows}"
    except Exception as exc:
        result = "unmatched_error" if ch_code(exc) == 497 and UNMATCHED_POLICY_TEXT in str(exc) else "other_error"
        observed = f"code {ch_code(exc)}: {first_line(exc)}"
    if result == "other_error":
        outcome = "NOT_RUN"
    elif server == "configured":
        outcome = "CONFIRMS" if result == "unmatched_error" else "CONTRADICTS"
    else:
        outcome = "CONFIRMS" if result == "all" else "CONTRADICTS"
    expected = (
        "configured server raises the unmatched-row-policy error (throw_on_unmatched_row_policies) for a user without a matching policy"
        if server == "configured"
        else "default server lets a user without a matching policy read every row"
    )
    record("EP-07", outcome, expected, f"{server}: {observed}")


@pytest.mark.parametrize("server,port", [("configured", 18123), ("defaults", 18124)])
def test_ep07_cross_tenant_isolation(record, server, port):
    client = _connect(port, "hb_tenant_a")
    rows = client.query("SELECT tenant, toString(sum(cost)) FROM hb.facts GROUP BY tenant ORDER BY tenant").result_rows
    b_rows = client.query("SELECT count() FROM hb.facts WHERE tenant = 'B'").result_rows[0][0]
    outcome = "CONFIRMS" if b_rows == 0 and [r[0] for r in rows] == ["A"] else "CONTRADICTS"
    record("EP-07", outcome, "a tenant-A user sees zero tenant-B rows", f"{server}: rows {rows}, tenant-B count {b_rows}")


BYPASS_ATTEMPTS = {
    "s3": ("SELECT count() FROM s3(hb_objects, filename = 'probe/plain.csv', format = 'CSV', structure = 'x UInt32')", 497),
    "url": ("SELECT * FROM url('http://seaweedfs:8333/', 'RawBLOB', 'x String') LIMIT 1", 497),
    "remote": ("SELECT * FROM remote('127.0.0.1', system.one)", 497),
    "file": ("SELECT * FROM file('probe.csv', 'CSV', 'x String')", 497),
    "insert_into_function": ("INSERT INTO FUNCTION s3(hb_objects, filename = 'probe/leak.csv', format = 'CSV') SELECT * FROM hb.facts", 497),
    "alter_delete": ("ALTER TABLE hb.facts DELETE WHERE 1", 497),
    "attach_partition": ("ALTER TABLE hb.facts_copy ATTACH PARTITION 'B' FROM hb.facts", 497),
    "move_partition": ("ALTER TABLE hb.facts MOVE PARTITION 'B' TO TABLE hb.facts_copy", 497),
    "named_collections": ("SELECT name FROM system.named_collections", 497),
    "override_join_use_nulls": ("SELECT 1 SETTINGS join_use_nulls = 0", 452),
    "override_null_as_default": ("SELECT 1 SETTINGS input_format_null_as_default = 1", 452),
    "override_serialize_query_plan": ("SELECT 1 SETTINGS serialize_query_plan = 1", 452),
}


@pytest.mark.parametrize("name", sorted(BYPASS_ATTEMPTS))
def test_ep08_reader_bypass_denied(reader, record, name):
    query, expected_code = BYPASS_ATTEMPTS[name]
    try:
        result = reader.query(query).result_rows
        outcome = "CONFIRMS" if name == "named_collections" and result == [] else "CONTRADICTS"
        observed = f"allowed: {result[:3]}"
    except Exception as exc:
        code = ch_code(exc)
        outcome = "CONFIRMS" if code == expected_code else "NOT_RUN"
        observed = f"code {code} (expected {expected_code}): {first_line(exc)}"
    record("EP-08", outcome, "reader cannot bypass scope, read external sources, write, alter or change locked settings", f"{name}: {observed}")
