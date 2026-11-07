#!/bin/sh
set -eu

base_url="${1:-http://localhost:8080}"

curl -fsS "$base_url/health/ready" >/dev/null
curl -fsS "$base_url/openapi.yaml" >/dev/null
curl -fsS "$base_url/metrics" | grep -q "mars_admin_http_requests_total"

echo "Mars Admin smoke test passed: $base_url"
