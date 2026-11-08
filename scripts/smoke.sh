#!/bin/sh
set -eu

base_url="${1:-http://localhost:8080}"

curl -fsS "$base_url/health/ready" >/dev/null
curl -fsS "$base_url/openapi.yaml" >/dev/null
if [ -n "${AUTH_TOKEN:-}" ]; then
  curl -fsS -H "Authorization: Bearer $AUTH_TOKEN" "$base_url/metrics" \
    | grep -q "mars_admin_http_requests_total"
else
  echo "metrics check skipped: set AUTH_TOKEN with audit:read to verify /metrics" >&2
fi

echo "Mars Admin smoke test passed: $base_url"
