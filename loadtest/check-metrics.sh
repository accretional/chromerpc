#!/usr/bin/env bash
#
# Query Cloud Run monitoring metrics (MQL) for the load-test service and flag the
# health signals that matter after a burst load test:
#   - instances over time: do they scale up during pulses and back DOWN after?
#     (instances that tick up and never come down => leaked/unhealthy instances,
#      e.g. from process cleanup issues after clients disconnect)
#   - request latency p95 per minute: did it climb across pulses?
#   - container startups: instance (re)starts — a spike implies instance deaths
#   - request counts by response-code class
#
# These checks are advisory diagnostics for agents/devs — they WARN, never fail,
# and are skippable. Don't gate CI/automation on them.
#
# Usage: ./loadtest/check-metrics.sh [SERVICE] [WINDOW]
#   SERVICE default: chromerpc-bidi-pool ; WINDOW default: 30m
#   SKIP_METRICS=1 to skip entirely.
set -uo pipefail   # deliberately NOT -e: a flaky metrics query must not fail the run

if [ "${SKIP_METRICS:-0}" = "1" ]; then
  echo "## metrics check skipped (SKIP_METRICS=1)"
  exit 0
fi

SERVICE="${1:-${SERVICE:-chromerpc-bidi-pool}}"
WINDOW="${2:-30m}"
PROJECT="${PROJECT:-$(gcloud config get-value project 2>/dev/null)}"
TOKEN="$(gcloud auth print-access-token 2>/dev/null)" || { echo "WARN: no access token; skipping metrics"; exit 0; }
API="https://monitoring.googleapis.com/v3/projects/${PROJECT}/timeSeries:query"

mql() {
  local out
  out="$(curl -s --max-time 30 -X POST "$API" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
    -d "$(jq -nc --arg q "$1" '{query:$q}')" 2>/dev/null)" || { echo '{"error":{"message":"query request failed"}}'; return 0; }
  echo "$out"
}
ts() { # print "endTime value" rows sorted by time
  jq -r 'if .error then "ERROR: \(.error.message)"
         else ([.timeSeriesData[]?.pointData[]? | {t:.timeInterval.endTime, v:(.values[0].doubleValue // .values[0].int64Value // (.values[0].distributionValue.mean))}]
               | sort_by(.t)[] | "\(.t|sub("\\..*Z$";"Z")|sub(".*T";""))  \(.v)")
         end'
}

echo "### service=${SERVICE}  project=${PROJECT}  window=${WINDOW}"

echo
echo "== instances over time (sum active+idle per minute) =="
echo "   want: rises during pulses, returns toward 0 after (scale-to-zero). A floor that keeps climbing => leak."
mql "fetch cloud_run_revision
| metric 'run.googleapis.com/container/instance_count'
| filter resource.service_name == '${SERVICE}'
| align mean_aligner(1m)
| every 1m
| group_by [], [insts: sum(value.instance_count)]
| within ${WINDOW}" | ts

echo
echo "== request latency p95 per minute (ms) =="
echo "   want: comparable across the 3 pulses (no drastic climb)."
mql "fetch cloud_run_revision
| metric 'run.googleapis.com/request_latencies'
| filter resource.service_name == '${SERVICE}'
| align delta(1m)
| every 1m
| group_by [], [p95: percentile(value.request_latencies, 95)]
| within ${WINDOW}" | ts

echo
echo "== container startups per minute (instance (re)starts) =="
echo "   want: a burst at the first pulse, near-zero after; later spikes => instance deaths/restarts."
mql "fetch cloud_run_revision
| metric 'run.googleapis.com/container/startup_latencies'
| filter resource.service_name == '${SERVICE}'
| align delta(1m)
| every 1m
| group_by [], [starts: count(value.startup_latencies)]
| within ${WINDOW}" | ts

echo
echo "== request count by response-code class per minute =="
echo "   note: gRPC errors ride inside 2xx HTTP; the loadtest client is the source of truth for app-level failures."
mql "fetch cloud_run_revision
| metric 'run.googleapis.com/request_count'
| filter resource.service_name == '${SERVICE}'
| align delta(1m)
| every 1m
| group_by [metric.response_code_class], [n: sum(value.request_count)]
| within ${WINDOW}" | jq -r 'if .error then "ERROR: \(.error.message)"
   else ([.timeSeriesData[]?[]? ] | length) as $_ |
        (.timeSeriesData[]? | (.labelValues[0].stringValue // "?") as $cls |
          (.pointData[]? | "\(.timeInterval.endTime|sub("\\..*Z$";"Z")|sub(".*T";""))  class=\($cls)  \(.values[0].int64Value // .values[0].doubleValue)"))
   end' | sort | tail -20
