#!/bin/sh
set -eu

: "${DATABASE_URL:?DATABASE_URL is required}"
backup="${1:?usage: restore.sh BACKUP_FILE}"

if [ "${ALLOW_DESTRUCTIVE_RESTORE:-0}" != "1" ]; then
  echo "set ALLOW_DESTRUCTIVE_RESTORE=1 only after verifying the target database" >&2
  exit 1
fi
if [ ! -f "$backup" ]; then
  echo "backup file not found: $backup" >&2
  exit 1
fi
if [ -f "$backup.sha256" ]; then
  sha256sum -c "$backup.sha256"
fi

pg_restore --clean --if-exists --no-owner --dbname="$DATABASE_URL" "$backup"
echo "database restored from $backup"
