#!/bin/sh
set -eu

: "${DATABASE_URL:?DATABASE_URL is required}"
backup="${1:?usage: restore.sh BACKUP_FILE}"

if [ ! -f "$backup" ]; then
  echo "backup file not found: $backup" >&2
  exit 1
fi

pg_restore --clean --if-exists --no-owner --dbname="$DATABASE_URL" "$backup"
echo "database restored from $backup"
