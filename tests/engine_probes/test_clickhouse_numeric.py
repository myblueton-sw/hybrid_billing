from decimal import Decimal

import pytest

from conftest import BUCKET


@pytest.fixture(scope="module", autouse=True)
def numeric_tables(admin):
    admin.command("CREATE DATABASE IF NOT EXISTS hb")
    admin.command("DROP TABLE IF EXISTS hb.dec38")
    admin.command("CREATE TABLE hb.dec38 (v Decimal(38, 0)) ENGINE = MergeTree ORDER BY tuple()")
    admin.command("INSERT INTO hb.dec38 VALUES (99999999999999999999999999999999999999), (99999999999999999999999999999999999999)")
    admin.command("DROP TABLE IF EXISTS hb.dec18")
    admin.command("CREATE TABLE hb.dec18 (v Decimal(18, 6)) ENGINE = MergeTree ORDER BY tuple()")
    admin.command("INSERT INTO hb.dec18 VALUES (900000000000.000000), (900000000000.000000)")


@pytest.mark.parametrize("check", [1, 0])
def test_ep01_decimal128_sum_overflow(admin, record, check):
    exact = Decimal(199999999999999999999999999999999999998)
    try:
        row = admin.query(
            "SELECT toString(sum(v)), toTypeName(sum(v)) FROM hb.dec38",
            settings={"decimal_check_overflow": check},
        ).result_rows[0]
        observed = f"result {row[0]} type {row[1]}; exact {exact}"
        silent_wrong = Decimal(row[0]) != exact or len(row[0].lstrip("-")) > 38
        outcome = "CONFIRMS" if silent_wrong else "CONTRADICTS"
    except Exception as exc:
        observed = f"error: {str(exc).splitlines()[0][:200]}"
        outcome = "CONTRADICTS"
    record("EP-01", outcome, "Decimal128 sum may return a wrong or out-of-precision result without error, so an independent bound is required", observed)


def test_ep02_decimal64_sum_widening(admin, record):
    row = admin.query("SELECT toString(sum(v)), toTypeName(sum(v)) FROM hb.dec18").result_rows[0]
    correct = Decimal(row[0]) == Decimal("1800000000000.000000")
    record(
        "EP-02",
        "CONFIRMS" if correct else "CONTRADICTS",
        "sum over Decimal(18,6) widens beyond 18 digits and stays exact",
        f"result {row[0]} type {row[1]}",
    )


def test_ep03_excess_scale_csv(admin, record):
    admin.command("DROP TABLE IF EXISTS hb.scale")
    admin.command("CREATE TABLE hb.scale (v Decimal(18, 6)) ENGINE = MergeTree ORDER BY tuple()")
    try:
        admin.command("INSERT INTO hb.scale FORMAT CSV\n1.1234567\n1.1234564\n")
        rows = [r[0] for r in admin.query("SELECT toString(v) FROM hb.scale ORDER BY v").result_rows]
        observed = f"accepted silently as {rows} from 1.1234567 and 1.1234564"
        outcome = "CONFIRMS"
    except Exception as exc:
        observed = f"error: {str(exc).splitlines()[0][:200]}"
        outcome = "CONTRADICTS"
    record("EP-03", outcome, "ClickHouse does not reject excess scale, so the Python manifest step must detect it", observed)


def test_ep03_excess_scale_s3(admin, s3, record):
    s3.put_object(Bucket=BUCKET, Key="probe/scale.csv", Body=b"1.1234567\n")
    try:
        rows = admin.query(
            "SELECT toString(v) FROM s3(hb_objects, filename = 'probe/scale.csv', format = 'CSV', structure = 'v Decimal(18, 6)')"
        ).result_rows
        observed = f"accepted silently as {[r[0] for r in rows]} from 1.1234567"
        outcome = "CONFIRMS"
    except Exception as exc:
        observed = f"error: {str(exc).splitlines()[0][:200]}"
        outcome = "CONTRADICTS"
    record("EP-03", outcome, "ClickHouse does not reject excess scale through s3(), so the Python manifest step must detect it", observed)
