import datetime
import secrets

from botocore.exceptions import ClientError

from conftest import BUCKET


def _code(exc):
    return exc.response.get("Error", {}).get("Code") or str(exc.response.get("ResponseMetadata", {}).get("HTTPStatusCode"))


def _ensure_bucket(s3, bucket, **kwargs):
    try:
        s3.create_bucket(Bucket=bucket, **kwargs)
    except ClientError as exc:
        if _code(exc) not in ("BucketAlreadyOwnedByYou", "BucketAlreadyExists"):
            raise


def test_ep11_conditional_create(s3, record):
    key = f"journal/{secrets.token_hex(8)}.json"
    s3.put_object(Bucket=BUCKET, Key=key, Body=b"first", IfNoneMatch="*")
    try:
        s3.put_object(Bucket=BUCKET, Key=key, Body=b"second", IfNoneMatch="*")
        body = s3.get_object(Bucket=BUCKET, Key=key)["Body"].read()
        observed, rejected = f"second write accepted; stored body {body!r}", False
    except ClientError as exc:
        observed, rejected = f"second write rejected: {_code(exc)}", _code(exc) in ("PreconditionFailed", "412")
        if not rejected:
            record("EP-11", "NOT_RUN", "conditional create rejects a second write to the same key (412)", f"unexpected error: {_code(exc)}")
            return
    record("EP-11", "CONFIRMS" if rejected else "CONTRADICTS", "conditional create rejects a second write to the same key (412)", observed)


def test_ep11_versioning(s3, record):
    bucket = "hb-versioned"
    key = f"entry-{secrets.token_hex(4)}"
    try:
        _ensure_bucket(s3, bucket)
        s3.put_bucket_versioning(Bucket=bucket, VersioningConfiguration={"Status": "Enabled"})
        s3.put_object(Bucket=bucket, Key=key, Body=b"v1")
        s3.put_object(Bucket=bucket, Key=key, Body=b"v2")
        versions = s3.list_object_versions(Bucket=bucket, Prefix=key).get("Versions", [])
        observed, kept = f"{len(versions)} versions retained", len(versions) == 2
    except ClientError as exc:
        record("EP-11", "NOT_RUN", "overwrite with versioning keeps the prior version", f"setup error: {_code(exc)}")
        return
    record("EP-11", "CONFIRMS" if kept else "CONTRADICTS", "overwrite with versioning keeps the prior version", observed)


def test_ep11_object_lock(s3, record):
    bucket = "hb-locked"
    key = f"entry-{secrets.token_hex(4)}"
    try:
        _ensure_bucket(s3, bucket, ObjectLockEnabledForBucket=True)
        until = datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(days=1)
        put = s3.put_object(Bucket=bucket, Key=key, Body=b"locked", ObjectLockMode="COMPLIANCE", ObjectLockRetainUntilDate=until)
        version = put.get("VersionId")
        try:
            if version:
                s3.delete_object(Bucket=bucket, Key=key, VersionId=version)
            else:
                s3.delete_object(Bucket=bucket, Key=key)
            remaining = s3.list_object_versions(Bucket=bucket, Prefix=key).get("Versions", [])
            observed, protected = f"delete accepted; versions left {len(remaining)}; version id {version}", any(v.get("VersionId") == version for v in remaining)
        except ClientError as exc:
            observed, protected = f"delete rejected: {_code(exc)}", _code(exc) == "AccessDenied"
    except ClientError as exc:
        record("EP-11", "NOT_RUN", "a COMPLIANCE-locked object version cannot be deleted before its retention date", f"object lock setup failed: {_code(exc)}")
        return
    record("EP-11", "CONFIRMS" if protected else "CONTRADICTS", "a COMPLIANCE-locked object version cannot be deleted before its retention date", observed)
