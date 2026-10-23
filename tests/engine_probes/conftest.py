import datetime
import json
import os
import pathlib
import secrets

import boto3
import clickhouse_connect
import psycopg
import pytest
from botocore.config import Config

ROOT = pathlib.Path(__file__).resolve().parents[2]
ENV_FILE = ROOT / "deploy" / "local" / "verification" / ".env"
BUCKET = "hb-probe"


def _load_env():
    values = {}
    for line in ENV_FILE.read_text().splitlines():
        if line and not line.startswith("#"):
            key, _, value = line.partition("=")
            values[key] = value
    return values


ENV = _load_env()
RUN_ID = os.environ.get("HB_RUN_ID") or (
    datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + secrets.token_hex(3)
)
RUN_DIR = ROOT / ".local" / "runs" / RUN_ID
POLICY_PASSWORD = secrets.token_hex(16)
STARTED = datetime.datetime.now(datetime.timezone.utc).isoformat()
OUTCOMES = []


def ch(user="probe_admin", password_key="CH_ADMIN_PASSWORD", port=18123, **settings):
    return clickhouse_connect.get_client(
        host="127.0.0.1",
        port=port,
        username=user,
        password=ENV[password_key],
        settings=settings or None,
        autogenerate_session_id=False,
    )


@pytest.fixture(scope="session")
def admin():
    client = ch()
    client.command("CREATE DATABASE IF NOT EXISTS hb")
    for user, key in (("hb_loader", "CH_LOADER_PASSWORD"), ("hb_reader", "CH_READER_PASSWORD")):
        client.command(f"DROP USER IF EXISTS {user}")
        client.command(f"CREATE USER {user} IDENTIFIED WITH sha256_password BY '{ENV[key]}' SETTINGS PROFILE 'hb_locked'")
    return client


@pytest.fixture(scope="session")
def loader(admin):
    return ch("hb_loader", "CH_LOADER_PASSWORD")


@pytest.fixture(scope="session")
def reader(admin):
    return ch("hb_reader", "CH_READER_PASSWORD")


@pytest.fixture(scope="session")
def defaults_admin():
    return ch(port=18124)


@pytest.fixture(scope="session")
def s3():
    client = boto3.client(
        "s3",
        endpoint_url="http://127.0.0.1:18333",
        aws_access_key_id=ENV["S3_ACCESS_KEY"],
        aws_secret_access_key=ENV["S3_SECRET_KEY"],
        region_name="us-east-1",
        config=Config(s3={"addressing_style": "path"}, retries={"max_attempts": 1}),
    )
    try:
        client.create_bucket(Bucket=BUCKET)
    except client.exceptions.BucketAlreadyOwnedByYou:
        pass
    return client


@pytest.fixture(scope="session")
def pg_admin():
    conn = psycopg.connect(
        host="127.0.0.1",
        port=15432,
        dbname="hb",
        user=ENV["PG_ADMIN_USER"],
        password=ENV["PG_ADMIN_PASSWORD"],
        autocommit=True,
    )
    yield conn
    conn.close()


@pytest.fixture(scope="session")
def secrets_in_use():
    return [ENV[k] for k in ("S3_ACCESS_KEY", "S3_SECRET_KEY", "CH_ADMIN_PASSWORD", "CH_LOADER_PASSWORD", "CH_READER_PASSWORD", "PG_ADMIN_PASSWORD")] + [POLICY_PASSWORD]


@pytest.fixture
def record(request):
    def _record(probe_id, outcome, expected, observed):
        assert outcome in ("CONFIRMS", "CONTRADICTS", "NOT_RUN")
        OUTCOMES.append(
            {
                "probe": probe_id,
                "case": request.node.name,
                "outcome": outcome,
                "expected": expected,
                "observed": observed,
            }
        )

    return _record


def pytest_sessionfinish(session, exitstatus):
    RUN_DIR.mkdir(parents=True, exist_ok=True)
    (RUN_DIR / "outcomes.json").write_text(
        json.dumps(
            {
                "run_id": RUN_ID,
                "started_utc": STARTED,
                "finished_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                "pytest_exit": int(exitstatus),
                "outcomes": OUTCOMES,
            },
            indent=2,
        )
        + "\n"
    )
