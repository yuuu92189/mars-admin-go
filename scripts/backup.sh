#!/bin/sh
set -eu

: "${DATABASE_URL:?DATABASE_URL is required}"
output="${1:-backup/mars-admin-$(date -u +%Y%m%dT%H%M%SZ).dump}"
mkdir -p "$(dirname "$output")"

pg_dump --format=custom --no-owner --file="$output" "$DATABASE_URL"
echo "database backup written to $output"
