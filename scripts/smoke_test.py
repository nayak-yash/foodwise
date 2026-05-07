"""Exercise the real Go/Postgres/Kafka/LangGraph stack; leaves traceable test data.

Run: python3 scripts/smoke_test.py [--restart-agent]
Requires a running Docker Compose stack. No third-party Python packages required.
"""
import argparse
import json
import os
import subprocess
import time
import urllib.error
import urllib.request
import uuid
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
if (ROOT / ".env").exists():
    for line in (ROOT / ".env").read_text().splitlines():
        if line.strip() and not line.startswith("#") and "=" in line:
            key, value = line.split("=", 1)
            os.environ.setdefault(key, value.strip().strip("\"'"))

BASE = os.getenv("API_URL", "http://localhost:8080") + "/api/v1"
MANAGER = os.getenv("MANAGER_API_TOKEN", "local-manager-change-me")
AGENT = os.getenv("AGENT_API_TOKEN", "local-agent-change-me")
CHICKEN = "22222222-2222-2222-2222-222222222222"
SUPPLIER = "33333333-3333-3333-3333-333333333333"
MENU = "44444444-4444-4444-4444-444444444444"
PREFIX = "smoke-" + uuid.uuid4().hex[:12]


def request(path, body=None, key=None, token=MANAGER, expected=200):
    data = None if body is None else json.dumps(body).encode()
    headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}
    if data is not None:
        headers["Idempotency-Key"] = key or PREFIX + "-" + uuid.uuid4().hex
    req = urllib.request.Request(BASE + path, data=data, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            status, result = response.status, json.load(response)
    except urllib.error.HTTPError as error:
        status, result = error.code, json.load(error)
    assert status == expected, f"{path}: expected {expected}, got {status}: {result}"
    return result


def until(description, check, timeout=120):
    deadline = time.monotonic() + timeout
    last = None
    while time.monotonic() < deadline:
        last = check()
        if last:
            print(f"PASS {description}", flush=True)
            return last
        time.sleep(2)
    raise AssertionError(f"Timed out: {description}; last value: {last}")


def stock(ingredient=CHICKEN):
    return next(i["balance"] for i in request("/inventory") if i["id"] == ingredient)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--restart-agent", action="store_true", help="Restart the agent while waiting for manager approval")
    args = parser.parse_args()
    request("/inventory", token="invalid", expected=401)
    request("/purchase-orders", {"supplier_id": SUPPLIER, "ingredient_id": CHICKEN, "quantity": 40}, token=AGENT, expected=403)
    print("PASS authentication and manager/agent separation", flush=True)

    opening = stock()
    po = request("/purchase-orders", {"supplier_id": SUPPLIER, "ingredient_id": CHICKEN, "quantity": 40})["id"]
    receipt = {"quantity": 25, "source_reference": PREFIX + "-signed-receipt", "external_status": "completed"}
    receipt_key = PREFIX + "-receipt"
    with ThreadPoolExecutor(max_workers=2) as pool:
        results = list(pool.map(lambda _: request(f"/purchase-orders/{po}/receipts", receipt, receipt_key), range(2)))
    assert results[0] == results[1]
    assert abs(stock() - opening - 25) < 0.001
    assert results[0]["status"] == "partial" and results[0]["outstanding"] == 15
    request(f"/purchase-orders/{po}/receipts", {**receipt, "quantity": 24}, receipt_key, expected=409)
    request(f"/purchase-orders/{po}/receipts", {**receipt, "quantity": 16}, expected=409)
    print("PASS concurrent receipt retries, shortfall invariant, over-receipt protection", flush=True)

    # Ingredient deductions must be recipe-driven and idempotent.
    before_order = stock()
    order_body = {"menu_item_id": MENU, "quantity": 2}
    request("/orders", order_body, PREFIX + "-order")
    request("/orders", order_body, PREFIX + "-order")
    assert abs(stock() - before_order + 0.5) < 0.001
    print("PASS recipe consumption occurs once", flush=True)
    request("/kitchen/metrics", {"prep_minutes": 32, "cancellation_rate": 0.18, "order_count": 65})

    def find_incident():
        return next((i for i in request("/incidents") if i["purchase_order_id"] == po and i["status"] == "awaiting_approval"), None)

    incident = until("Kafka-triggered investigation reaches approval", find_incident)
    incident_id = incident["id"]
    detail = request(f"/incidents/{incident_id}")
    assert detail["diagnosis"]["missing_kg"] == 15
    assert len(detail["diagnosis"]["hypotheses"]) == 4
    action = detail["actions"][0]
    request(f"/actions/{action['id']}/execute", {}, expected=409)
    request(f"/actions/{action['id']}/approve", {}, token=AGENT, expected=403)
    request(f"/incidents/{incident_id}/report", {"status": "resolved"}, token=AGENT, expected=409)
    print("PASS evidence-grounded hypotheses and approval enforcement", flush=True)

    if args.restart_agent:
        subprocess.run(["docker", "compose", "restart", "intelligence"], cwd=ROOT, check=True)
        print("PASS agent restarted while approval checkpoint is persisted", flush=True)

    before_execution = stock()
    request(f"/actions/{action['id']}/approve", {}, PREFIX + "-approve")
    stock_request = until("approved checkpoint resumes and creates a stock request", lambda: next((r for r in request("/stock-requests") if r["incident_id"] == incident_id), None))
    request(f"/actions/{action['id']}/execute", {}, PREFIX + "-execute-retry")
    assert abs(stock() - before_execution) < 0.001
    assert len([r for r in request("/stock-requests") if r["incident_id"] == incident_id]) == 1
    request(f"/incidents/{incident_id}/report", {"status": "resolved"}, token=AGENT, expected=409)
    print("PASS execution creates one request, adds no fictional stock, cannot resolve early", flush=True)

    actual_receipt = {"quantity": 15, "source_reference": PREFIX + "-replacement-receipt"}
    request(f"/stock-requests/{stock_request['id']}/receive", actual_receipt, PREFIX + "-replacement")
    request(f"/stock-requests/{stock_request['id']}/receive", actual_receipt, PREFIX + "-replacement")
    assert abs(stock() - before_execution - 15) < 0.001
    assert request(f"/incidents/{incident_id}")["status"] != "resolved"
    for index in range(3):
        if index:
            time.sleep(5.2)
        request("/kitchen/metrics", {"prep_minutes": 14, "cancellation_rate": 0.02, "order_count": 12})
    resolved = until("receipt plus healthy post-action windows produce verified recovery", lambda: (lambda i: i if i["status"] == "resolved" else None)(request(f"/incidents/{incident_id}")))
    assert all(resolved["recovery"]["checks"].values())
    po_after = next(p for p in request("/purchase-orders") if p["id"] == po)
    assert po_after["outstanding"] == 15, "Replacement stock must not rewrite supplier fulfillment history"
    print(f"\nEND-TO-END PASS — incident {incident_id}", flush=True)


if __name__ == "__main__":
    main()
