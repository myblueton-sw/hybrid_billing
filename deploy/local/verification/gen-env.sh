#!/bin/sh
set -eu
cd "$(dirname "$0")"
[ -e .env ] && { echo ".env exists; remove it to regenerate" >&2; exit 1; }
umask 077
r() { LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c "$1"; }
{
  echo "PG_ADMIN_USER=hb_admin"
  echo "PG_ADMIN_PASSWORD=$(r 32)"
  echo "CH_ADMIN_PASSWORD=$(r 32)"
  echo "CH_LOADER_PASSWORD=$(r 32)"
  echo "CH_READER_PASSWORD=$(r 32)"
  echo "S3_ACCESS_KEY=$(r 20)"
  echo "S3_SECRET_KEY=$(r 40)"
} > .env
echo "wrote .env"
