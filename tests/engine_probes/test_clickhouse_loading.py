import io
import secrets

import clickhouse_connect
import pyarrow as pa
import pyarrow.parquet as pq
import pytest

from conftest import BUCKET
from probe_support import ch_code, first_line

COLUMN_TYPES = {"ld_null": "Nullable(Decimal(18, 6))", "ld_notnull": "Decimal(18, 6)"}
PARSE_REJECT_CODES = {6, 27, 32, 72, 117, 349}
CANDIDATE = {"input_format_csv_empty_as_default": 1, "input_format_null_as_default": 0}


def _parquet(values):
    buffer = io.BytesIO()
    pq.write_table(pa.table({"id": pa.array(range(1, len(values) + 1), pa.uint32()), "v": pa.array(values, pa.decimal128(18, 6))}), buffer)
    return buffer.getvalue()


@pytest.fixture(scope="module", autouse=True)
def loading_setup(admin, s3):
    for name, column in COLUMN_TYPES.items():
        admin.command(f"DROP TABLE IF EXISTS hb.{name}")
        admin.command(f"CREATE TABLE hb.{name} (id UInt32, v {column}) ENGINE = MergeTree ORDER BY id")
    for statement in (
        "GRANT INSERT, SELECT, TRUNCATE ON hb.* TO hb_loader",
        "GRANT S3 ON *.* TO hb_loader",
        "GRANT NAMED COLLECTION ON hb_objects TO hb_loader",
        "GRANT CREATE TEMPORARY TABLE ON *.* TO hb_loader",
    ):
        admin.command(statement)
    s3.put_object(Bucket=BUCKET, Key="probe/null-only.csv", Body=b"1,\\N\n2,5\n")
    s3.put_object(Bucket=BUCKET, Key="probe/empty-only.csv", Body=b"1,\n2,5\n")
    s3.put_object(Bucket=BUCKET, Key="probe/null-only.parquet", Body=_parquet([None, 5]))
    buffer = io.BytesIO()
    pq.write_table(pa.table({"id": pa.array([1], pa.uint32())}), buffer)
    s3.put_object(Bucket=BUCKET, Key="probe/missing.parquet", Body=buffer.getvalue())


def _load(client, table, key, fmt, settings=None):
    client.command(f"TRUNCATE TABLE hb.{table}")
    client.command(
        f"INSERT INTO hb.{table} SELECT * FROM s3(hb_objects, filename = '{key}', format = '{fmt}', structure = 'id UInt32, v {COLUMN_TYPES[table]}')",
        settings=settings,
    )
    return client.query(f"SELECT id, toString(v) FROM hb.{table} ORDER BY id").result_rows


def _classify(rows, exc):
    if exc is not None:
        if ch_code(exc) in PARSE_REJECT_CODES:
            return "rejected", f"rejected (code {ch_code(exc)}): {first_line(exc)}"
        return "other_error", f"unexpected error (code {ch_code(exc)}): {first_line(exc)}"
    value = rows[0][1]
    if value is None:
        return "null", f"row 1 value NULL; rows {rows}"
    if value in ("0", "0.000000"):
        return "zero", f"row 1 value {value}; rows {rows}"
    return "other_value", f"row 1 value {value}; rows {rows}"


CELLS = [
    (profile, table, key, fmt)
    for profile in ("default", "locked", "candidate")
    for table in COLUMN_TYPES
    for key, fmt in (("probe/null-only.csv", "CSV"), ("probe/empty-only.csv", "CSV"), ("probe/null-only.parquet", "Parquet"))
]

EXPECTED = {
    "default": "default settings: Nullable keeps NULL, non-Nullable becomes zero",
    "locked": "design-locked settings: Nullable keeps NULL, non-Nullable rejects; never zero",
    "candidate": "candidate settings: Nullable keeps NULL, non-Nullable rejects; never zero",
}


@pytest.mark.parametrize("profile,table,key,fmt", CELLS)
def test_ep04_null_and_empty_cells(admin, loader, record, profile, table, key, fmt):
    client = loader if profile == "locked" else admin
    rows, exc = None, None
    try:
        rows = _load(client, table, key, fmt, CANDIDATE if profile == "candidate" else None)
    except Exception as error:
        exc = error
    result, observed = _classify(rows, exc)
    if result == "other_error":
        outcome = "NOT_RUN"
    elif profile == "default":
        outcome = "CONFIRMS" if result == ("null" if table == "ld_null" else "zero") else "CONTRADICTS"
    else:
        outcome = "CONFIRMS" if result == ("null" if table == "ld_null" else "rejected") else "CONTRADICTS"
    record("EP-04", outcome, EXPECTED[profile], f"{profile} {table} {key}: {observed}")


@pytest.mark.parametrize("profile", ["default", "locked"])
def test_ep05_parquet_missing_column(admin, loader, record, profile):
    client = admin if profile == "default" else loader
    try:
        rows = _load(client, "ld_null", "probe/missing.parquet", "Parquet")
        result, observed = "filled", f"rows {rows}"
    except Exception as exc:
        missing = ch_code(exc) == 8 and "not found in parquet schema" in str(exc)
        result = "rejected" if missing else "other_error"
        observed = f"code {ch_code(exc)}: {first_line(exc)}"
    if result == "other_error":
        outcome = "NOT_RUN"
    elif profile == "default":
        outcome = "CONFIRMS" if result == "filled" else "CONTRADICTS"
    else:
        outcome = "CONFIRMS" if result == "rejected" else "CONTRADICTS"
    expected = "default fills a missing Parquet column silently" if profile == "default" else "locked setting rejects a missing Parquet column (code 8)"
    record("EP-05", outcome, expected, f"{profile}: {observed}")


def test_ep06_join_use_nulls(admin, loader, record):
    query = "SELECT toString(b.x) FROM (SELECT 1 AS k) AS a LEFT JOIN (SELECT 2 AS k, toDecimal64(5, 2) AS x) AS b USING k"
    default_value = admin.query(query).result_rows[0][0]
    locked_value = loader.query(query).result_rows[0][0]
    outcome = "CONFIRMS" if default_value in ("0", "0.00") and locked_value is None else "CONTRADICTS"
    record("EP-06", outcome, "unmatched join gives zero by default and NULL with join_use_nulls = 1", f"default {default_value!r}, locked {locked_value!r}")


def test_ep09_sync_lightweight_delete(admin, record):
    admin.command("DROP TABLE IF EXISTS hb.attempts")
    admin.command("CREATE TABLE hb.attempts (release_id String, attempt UInt32, v Decimal(18, 6)) ENGINE = MergeTree ORDER BY (release_id, attempt)")
    admin.command("INSERT INTO hb.attempts SELECT 'r9', 1, toDecimal64(number, 6) FROM numbers(100000)")
    admin.command("INSERT INTO hb.attempts SELECT 'r9', 2, toDecimal64(number, 6) FROM numbers(100000)")
    admin.command("DELETE FROM hb.attempts WHERE release_id = 'r9' AND attempt = 1", settings={"lightweight_deletes_sync": 2})
    remaining = admin.query("SELECT count() FROM hb.attempts WHERE release_id = 'r9' AND attempt = 1").result_rows[0][0]
    kept = admin.query("SELECT count() FROM hb.attempts WHERE release_id = 'r9' AND attempt = 2").result_rows[0][0]
    outcome = "CONFIRMS" if remaining == 0 and kept == 100000 else "CONTRADICTS"
    record("EP-09", outcome, "synchronous lightweight delete leaves zero visible rows of the stale attempt immediately", f"stale remaining {remaining}, current kept {kept}")


def test_ep10_parquet_export_types(admin, s3, record):
    admin.command("DROP TABLE IF EXISTS hb.export_src")
    admin.command("CREATE TABLE hb.export_src (cost Decimal(18, 6), amount Nullable(Decimal(38, 10))) ENGINE = MergeTree ORDER BY tuple()")
    admin.command("INSERT INTO hb.export_src VALUES (1.5, NULL), (2.25, 3.0000000001)")
    admin.command(
        "INSERT INTO FUNCTION s3(hb_objects, filename = 'probe/export.parquet', format = 'Parquet') SELECT * FROM hb.export_src",
        settings={"s3_truncate_on_insert": 1},
    )
    schema = pq.read_schema(io.BytesIO(s3.get_object(Bucket=BUCKET, Key="probe/export.parquet")["Body"].read()))
    cost, amount = schema.field("cost"), schema.field("amount")
    good = cost.type == pa.decimal128(18, 6) and not cost.nullable and amount.type == pa.decimal128(38, 10) and amount.nullable
    record(
        "EP-10",
        "CONFIRMS" if good else "CONTRADICTS",
        "Parquet export keeps DECIMAL precision, scale and nullability",
        f"cost {cost.type} nullable={cost.nullable}; amount {amount.type} nullable={amount.nullable}",
    )


def test_ep12_named_collection_without_secret_leak(admin, s3, secrets_in_use, record):
    s3.put_object(Bucket=BUCKET, Key="probe/plain.csv", Body=b"1\n2\n")
    total = admin.query("SELECT sum(x) FROM s3(hb_objects, filename = 'probe/plain.csv', format = 'CSV', structure = 'x UInt32')").result_rows[0][0]
    errors = []
    for query in (
        "SELECT * FROM s3(hb_objects, filename = 'probe/absent.csv', format = 'CSV', structure = 'x UInt32')",
        "SELECT * FROM s3(hb_objects, filename = 'probe/plain.csv', access_key_id = 'wrong-key', secret_access_key = 'wrong-secret', format = 'CSV', structure = 'x UInt32')",
    ):
        try:
            admin.query(query)
        except Exception as exc:
            errors.append(str(exc))
    try:
        clickhouse_connect.get_client(host="127.0.0.1", port=18123, username="hb_reader", password=secrets.token_hex(8)).query("SELECT 1")
    except Exception as exc:
        errors.append(str(exc))
    admin.command("SYSTEM FLUSH LOGS")
    query_log = admin.query("SELECT groupConcat(query) FROM system.query_log WHERE event_time > now() - 900").result_rows[0][0] or ""
    sources = {"query_log": query_log, "errors": "\n".join(errors)}
    try:
        sources["text_log"] = admin.query("SELECT groupConcat(message) FROM system.text_log WHERE event_time > now() - 900").result_rows[0][0] or ""
    except Exception as exc:
        record("EP-12", "NOT_RUN", "text_log available for the leak scan", f"text_log unavailable: {first_line(exc)}")
    leaks = [name for name, text in sources.items() if any(s in text for s in secrets_in_use)]
    outcome = "CONFIRMS" if total == 3 and not leaks and len(errors) == 3 else "CONTRADICTS"
    record(
        "EP-12",
        outcome,
        "named collection reads work and no credential appears in logs or errors, including after wrong-credential and failed-login errors",
        f"sum {total}; errors provoked {len(errors)}; scanned {sorted(sources)}; leaks {leaks or 'none'}",
    )
