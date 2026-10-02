#!/usr/bin/env bash
# Raw HTTP example: no language, no client library — curl only.
#
# Run it against a started server (see examples/README.md):
#
#   ./fake-jev run --config examples/fake-jev.yaml -- \
#     bash examples/raw-http/systemone.sh
#
# Requirements: curl (the HTTP client) and jq (to read the JSON reply). Both
# are environment tools, not language packages: nothing is installed, no
# provider SDK is used, no credential is needed, and no network beyond the
# loopback server is touched.
set -euo pipefail

for tool in curl jq; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "missing required tool: $tool" >&2
    exit 1
  fi
done

base_url="${FAKE_JEV_URL:-http://127.0.0.1:8787}"
echo "fake-jev base URL: ${base_url}"

failures=0

check() {
  local condition="$1" message="$2"
  if [ "$condition" != "true" ]; then
    echo "FAIL: ${message}" >&2
    failures=$((failures + 1))
  fi
}

# 1. Model listing. §39.1 returns the configured (here: default) model list.
models="$(curl --silent --show-error --fail-with-body "${base_url}/v1/models")"
echo "GET /v1/models: ${models}"
check "$(jq -r '.models | length == 1' <<<"$models")" "expected exactly one listed model"
check "$(jq -r '.models[0].name == "jev-latest"' <<<"$models")" \
  "expected the model jev-latest"

# 2. One System One call with a choice and a noul question. §39.5: fake-jev
# accepts any bearer value and never logs it, so a real key is not needed;
# the header is sent only to show that it is ignored.
request='{
  "state": {"task": "triage"},
  "model": "jev-latest",
  "questions": {
    "route": {
      "type": "choice",
      "criteria": {
        "frontend": "the view layer",
        "backend": "the api layer",
        "infra": "the deployment"
      }
    },
    "urgent": {"type": "noul", "criteria": {"deadline": "today"}}
  }
}'

answers="$(curl --silent --show-error --fail-with-body \
  --header 'Content-Type: application/json' \
  --header 'Authorization: Bearer not-a-real-key' \
  --data "$request" \
  "${base_url}/v1/systemone")"
echo "POST /v1/systemone: ${answers}"

check "$(jq -r '.model == "jev-latest"' <<<"$answers")" \
  "response model should echo the request model"
check "$(jq -r '.answers.route.type == "choice"' <<<"$answers")" \
  "route answer should be a choice"
check "$(jq -r '.answers.route.choice == "backend"' <<<"$answers")" \
  "route choice should be backend"
check "$(jq -r '.answers.route.confidence == 0.91' <<<"$answers")" \
  "route confidence should be 0.91"
check "$(jq -r '.answers.route.probabilities.backend == 0.91' <<<"$answers")" \
  "backend probability should be 0.91"
check "$(jq -r '.answers.urgent.type == "noul"' <<<"$answers")" \
  "urgent answer should be a noul"
check "$(jq -r '.answers.urgent.noul == 0.94' <<<"$answers")" \
  "urgent noul should be 0.94"
check "$(jq -r '.usage == {"input_tokens": 0, "output_tokens": 0}' <<<"$answers")" \
  "usage should be the zero default"

# 3. Control-plane verification: the server decides whether the exchanges
# above matched the fixture. §41.9. The CLI wraps this same call:
#   ./fake-jev verify --url <base URL>
verification="$(curl --silent --show-error --fail-with-body \
  "${base_url}/__fake/v1/verify")"
echo "GET /__fake/v1/verify: ${verification}"
check "$(jq -r '.passed == true' <<<"$verification")" \
  "server verification should pass"

if [ "$failures" -ne 0 ]; then
  exit 1
fi
echo "OK: the fixture answers matched the expected values"
