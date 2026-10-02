#!/usr/bin/env python3
"""Python example: point an HTTP client at fake-jev and check the answer.

Run it against a started server (see examples/README.md):

    ./fake-jev run --config examples/fake-jev.yaml -- python3 examples/python/systemone.py

Standard library only: no pip install, no requirements.txt, no virtualenv,
no provider SDK, no credential, no model, and no network beyond the loopback
server.
"""

import json
import os
import sys
import urllib.error
import urllib.request

BASE_URL = os.environ.get("FAKE_JEV_URL", "http://127.0.0.1:8787")

FAILURES: list[str] = []


def check(condition: bool, message: str) -> None:
    if not condition:
        FAILURES.append(message)


def show(label: str, value: object) -> None:
    print(f"{label}: {json.dumps(value, sort_keys=True)}")


def request_json(method: str, path: str, body: dict | None = None) -> object:
    data = None if body is None else json.dumps(body).encode("utf-8")
    headers = {
        "Content-Type": "application/json",
        # §39.5: fake-jev accepts any bearer value and never logs it. A real
        # key is neither needed nor read.
        "Authorization": "Bearer not-a-real-key",
    }
    req = urllib.request.Request(BASE_URL + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=10) as response:  # noqa: S310 - loopback only
            if response.status != 200:
                raise RuntimeError(f"{method} {path} returned {response.status}")
            return json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as error:
        raise RuntimeError(f"{method} {path} returned {error.code}") from error


def main() -> int:
    print(f"fake-jev base URL: {BASE_URL}")

    # 1. Model listing. §39.1 returns the configured (here: default) model list.
    models = request_json("GET", "/v1/models")
    show("GET /v1/models", models)
    entries = models["models"]
    check(len(entries) == 1, "expected exactly one listed model")
    check(entries[0]["name"] == "jev-latest", "expected the model jev-latest")

    # 2. One System One call with a choice and a noul question.
    request = {
        "state": {"task": "triage"},
        "model": "jev-latest",
        "questions": {
            "route": {
                "type": "choice",
                "criteria": {
                    "frontend": "the view layer",
                    "backend": "the api layer",
                    "infra": "the deployment",
                },
            },
            "urgent": {"type": "noul", "criteria": {"deadline": "today"}},
        },
    }
    answers = request_json("POST", "/v1/systemone", request)
    show("POST /v1/systemone", answers)

    route = answers["answers"]["route"]
    urgent = answers["answers"]["urgent"]
    check(answers["model"] == "jev-latest", "response model should echo the request model")
    check(route["type"] == "choice", "route answer should be a choice")
    check(route["choice"] == "backend", "route choice should be backend")
    check(route["confidence"] == 0.91, "route confidence should be 0.91")
    check(route["probabilities"]["backend"] == 0.91, "backend probability should be 0.91")
    check(urgent["type"] == "noul", "urgent answer should be a noul")
    check(urgent["noul"] == 0.94, "urgent noul should be 0.94")
    check(
        answers["usage"] == {"input_tokens": 0, "output_tokens": 0},
        "usage should be the zero default",
    )

    # 3. Control-plane verification: the server decides whether the exchanges
    # above matched the fixture. §41.9.
    verification = request_json("GET", "/__fake/v1/verify")
    show("GET /__fake/v1/verify", verification)
    check(verification["passed"] is True, "server verification should pass")

    # The same check the CLI performs:
    #   ./fake-jev verify --url <base URL>
    if FAILURES:
        for failure in FAILURES:
            print(f"FAIL: {failure}", file=sys.stderr)
        return 1
    print("OK: the fixture answers matched the expected values")
    return 0


if __name__ == "__main__":
    sys.exit(main())
