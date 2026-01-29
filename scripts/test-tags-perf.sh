#!/bin/bash
# Quick performance test for GET /api/tags
# Usage: ./scripts/test-tags-perf.sh [base_url]
# Example: ./scripts/test-tags-perf.sh http://127.0.0.1:8888

set -e
BASE="${1:-http://127.0.0.1:8888}"
N="${2:-10}"

echo "Testing GET $BASE/api/tags (${N} requests)"
echo ""

# Single request with timing
echo "=== Single request (page=1, limit=9) ==="
curl -s -o /dev/null -w "  time_total: %{time_total}s\n  http_code: %{http_code}\n" "$BASE/api/tags?page=1&limit=9"
echo ""

# Multiple requests
echo "=== $N sequential requests ==="
total=0
for i in $(seq 1 "$N"); do
  t=$(curl -s -o /dev/null -w "%{time_total}" "$BASE/api/tags?page=1&limit=9")
  total=$(echo "$total + $t" | bc)
  printf "  #%02d: %ss\n" "$i" "$t"
done
avg=$(echo "scale=4; $total / $N" | bc)
echo "  ---"
echo "  avg: ${avg}s"
echo ""

# Pagination pages
echo "=== Different pages (1,2,3 limit=9) ==="
for p in 1 2 3; do
  t=$(curl -s -o /dev/null -w "%{time_total}" "$BASE/api/tags?page=$p&limit=9")
  echo "  page=$p: ${t}s"
done
echo ""
echo "Done."
