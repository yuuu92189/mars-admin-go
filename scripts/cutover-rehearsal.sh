#!/bin/sh
set -eu

: "${DATABASE_URL:?DATABASE_URL is required}"
: "${REHEARSAL_DATABASE_URL:?REHEARSAL_DATABASE_URL is required}"
: "${REHEARSAL_CONFIRM:?set REHEARSAL_CONFIRM=1 after verifying the rehearsal target}"

if [ "$REHEARSAL_CONFIRM" != "1" ]; then
  echo "set REHEARSAL_CONFIRM=1 to run the destructive rehearsal target restore" >&2
  exit 1
fi
if [ "$DATABASE_URL" = "$REHEARSAL_DATABASE_URL" ]; then
  echo "source and rehearsal database URLs must differ" >&2
  exit 1
fi

backup="${1:-backup/mars-admin-rehearsal-$(date -u +%Y%m%dT%H%M%SZ).dump}"
report="${2:-backup/mars-admin-rehearsal-report.json}"
mkdir -p "$(dirname "$backup")" "$(dirname "$report")"

echo "creating source backup: $backup"
pg_dump --format=custom --no-owner --file="$backup" "$DATABASE_URL"
sha256sum "$backup" > "$backup.sha256"

echo "restoring into isolated rehearsal database"
pg_restore --clean --if-exists --no-owner --dbname="$REHEARSAL_DATABASE_URL" "$backup"

version="$(psql "$REHEARSAL_DATABASE_URL" -Atc 'SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1')"
users="$(psql "$REHEARSAL_DATABASE_URL" -Atc 'SELECT count(*) FROM identity_users')"
menus="$(psql "$REHEARSAL_DATABASE_URL" -Atc 'SELECT count(*) FROM access_menus')"
settings="$(psql "$REHEARSAL_DATABASE_URL" -Atc 'SELECT count(*) FROM system_settings')"
payments="$(psql "$REHEARSAL_DATABASE_URL" -Atc 'SELECT count(*) FROM payment_orders')"

cat > "$report" <<EOF
{
  "status": "passed",
  "backup": "$backup",
  "checksum": "$(cat "$backup.sha256")",
  "migrationVersion": $version,
  "rowCounts": {
    "identity_users": $users,
    "access_menus": $menus,
    "system_settings": $settings,
    "payment_orders": $payments
  }
}
EOF

echo "cutover rehearsal passed; report written to $report"
