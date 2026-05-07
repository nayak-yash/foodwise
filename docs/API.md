# API contracts

Base URL: `http://localhost:8080/api/v1`

Every API request requires `Authorization: Bearer <token>`. Every mutation also requires `Idempotency-Key: <unique-key>` (maximum 160 characters). Repeating the same key with the same method, path, role, and exact JSON body returns the committed result. Reusing the key for a different request returns `409`.

Amounts use ingredient base units, with at most three decimal places. All seeded ingredients are measured in kg. All timestamps and seeded service history are UTC.

## Read endpoints — manager or agent

| GET | Response |
|---|---|
| `/overview` | Branch health, incident counts, latest kitchen measurement, unpublished event count |
| `/inventory` | Ingredient IDs, balances, units, reorder points |
| `/catalog` | Suppliers and menu items |
| `/purchase-orders` | Last 100 orders, receipts-to-date, outstanding quantity, canonical and external status |
| `/incidents` | Last 100 incidents |
| `/incidents/:id` | Diagnosis, action records, recovery checks, audit timeline |
| `/incidents/:id/evidence` | Consistent snapshot of PO, receipts, stock ledger, kitchen measurements, actions |
| `/demand-history` | Ingredient/service/day consumption history |
| `/stock-requests` | Replacement requests and receipt status |

## Manager mutations

### Create a purchase order

`POST /purchase-orders`

```json
{"supplier_id":"33333333-3333-3333-3333-333333333333","ingredient_id":"22222222-2222-2222-2222-222222222222","quantity":40}
```

### Record an actual receiving quantity

`POST /purchase-orders/:id/receipts`

```json
{"quantity":25,"source_reference":"SIGNED-RECEIVING-1042","external_status":"completed"}
```

`external_status` is `partial` or `completed`. Canonical fulfillment is calculated from received quantity; receipts cannot exceed outstanding quantity. A completed external flag with a short receipt triggers the detector.

### Start order preparation

`POST /orders`

```json
{"menu_item_id":"44444444-4444-4444-4444-444444444444","quantity":2}
```

Recipe consumption is deducted atomically at preparation start, after locking ingredients in stable order and checking available stock. Retries with the original idempotency key do not consume stock twice.

### Record waste

`POST /waste`

```json
{"ingredient_id":"22222222-2222-2222-2222-222222222222","quantity":0.5,"reason":"Spoiled during preparation"}
```

### Record a kitchen measurement window

`POST /kitchen/metrics`

```json
{"prep_minutes":14,"cancellation_rate":0.02,"order_count":12}
```

The server timestamps receipt of the measurement. Integrators must send distinct non-overlapping operational windows; the local simulator does this explicitly with synthetic values.

### Approval and receipt

| POST | Body | Result |
|---|---|---|
| `/actions/:id/approve` | `{}` | Approves immutable proposed parameters; emits resume event |
| `/actions/:id/reject` | `{}` | Rejects action and escalates incident |
| `/stock-requests/:id/receive` | `{"quantity":15,"source_reference":"REPLACEMENT-RECEIPT-1"}` | Records actual replacement stock once |
| `/demo/short-delivery` | `{}` | Creates synthetic disruption; available only with `DEMO_MODE=true` |

## Execution — manager or agent

`POST /actions/:id/execute`, body `{}`

Requires a manager-approved action and revalidates the outstanding shortfall. Produces one internal stock request. An executed action can be retried without creating a second request or increasing inventory. Changed stock conditions return `409` for manager review.

## Agent-only workflow mutations

| POST | Body |
|---|---|
| `/incidents` | `{"purchase_order_id":"...","fingerprint":"delivery-shortfall:...","title":"Chicken delivery shortfall","severity":"high"}` |
| `/incidents/:id/report` | `{"status":"investigating","diagnosis":{...},"recovery":{...}}` |
| `/incidents/:id/plan` | `{"quantity":15,"reason":"Receiving records support this shortfall"}` |

The report endpoint enforces incident state transitions. Resolution additionally checks live stock and kitchen evidence in Go; an agent assertion alone is insufficient. Report calls cannot grant approval.

## Agent service

Base URL: `http://localhost:8000`. These endpoints require the **agent** bearer token:

- `GET /status`: readiness, actual retrieval mode, reasoning configuration, last processing error, observation window.
- `GET /forecasts?target=2026-11-14`: weekday/service forecasts, recommended replenishment, holdout errors. Target is optional.

Both services expose `/healthz`, `/readyz`, and `/metrics`. Go readiness checks database availability; agent readiness indicates Kafka consumer startup. Dependencies and workflow outcomes are visible separately through logs and metrics.
