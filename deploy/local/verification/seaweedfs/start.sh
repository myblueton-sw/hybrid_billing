#!/bin/sh
set -eu
umask 077
cat > /tmp/s3.json <<EOF
{
  "identities": [
    {
      "name": "probe",
      "credentials": [{"accessKey": "${S3_ACCESS_KEY}", "secretKey": "${S3_SECRET_KEY}"}],
      "actions": ["Admin", "Read", "Write", "List", "Tagging"]
    }
  ]
}
EOF
exec weed server -dir=/data -volume.max=100 -s3 -s3.port=8333 -s3.config=/tmp/s3.json -ip.bind=0.0.0.0
