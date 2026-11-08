#!/bin/sh
set -eu

: "${DATABASE_URL:?DATABASE_URL is required}"
output="${1:-backup/mars-admin-$(date -u +%Y%m%dT%H%M%SZ).dump}"
mkdir -p "$(dirname "$output")"

pg_dump --format=custom --no-owner --file="$output" "$DATABASE_URL"
sha256sum "$output" > "$output.sha256"
echo "database backup written to $output"
echo "sha256 checksum written to $output.sha256"
