# Foodwise

An evidence-backed restaurant operations investigator, built with **Go/Gin, Python, LangGraph, LangChain, PostgreSQL/pgvector, Kafka, and Next.js**.

This repository implements a complete local MVP: **short delivery → investigation → manager approval → replacement stock request → actual receipt → verified recovery**.

## Start the platform

Requirements: Docker with Compose, approximately 8 GB of available memory, and internet access for the initial image/package/model downloads.

```bash
cp .env.example .env
docker compose up --build -d
docker compose ps
```

| Service | Address | Default local login |
|---|---|---|
| Manager dashboard | http://localhost:3000 | `manager` / `restaurantops` |
| Go API | http://localhost:8080 | Bearer `MANAGER_API_TOKEN` or `AGENT_API_TOKEN` |
| Agent health | http://localhost:8000/readyz | Health endpoint is public |
| Grafana | http://localhost:3001 | `admin` / `restaurantops` |

The dashboard uses signed, expiring HttpOnly manager-session cookies and a server-side API proxy. API tokens never need to be sent to browser JavaScript. All published service ports bind to localhost. The MVP has one seeded restaurant branch and separate manager/agent service roles.

### Walk through the demo

1. Sign in to the dashboard and click **Run incident demo**.
2. The scenario records a 40 kg purchase order, a 25 kg receipt, synthetic ingredient consumption, and a delayed kitchen measurement.
3. Go preserves the canonical purchase order as **partial**, even though its external supplier flag says **completed**.
4. The detector creates a delivery-shortfall incident. LangGraph gathers receiving, inventory, kitchen, forecast, and knowledge evidence, evaluates four hypotheses, and proposes 15 kg replacement stock.
5. Click **Approve action**. The graph resumes its persisted approval checkpoint and Go creates an internal stock request.
6. Click **Simulate receipt & recovery**. This explicitly synthetic demo records the 15 kg receipt and three healthy kitchen measurements over approximately 10 seconds.
7. Verification closes the incident only after receipt, positive stock, sufficient fresh samples, a minimum observation span, healthy preparation times, and healthy cancellation rates are confirmed.

The demo button is idempotent: clicking it again reuses the same scenario. Create additional purchase orders and partial receipts from **Inventory & supply** to investigate new incidents. The API smoke test also creates independent scenarios.

## Architecture

```text
Next.js dashboard ── server-side proxy ── Go/Gin API
                                           │
                               Business transactions + outbox
                                           │
                          PostgreSQL ◄──────┴──────► Kafka
                              ▲                       │
                              │                       ▼
                         pgvector ◄──── Python agent service
                              │            │
                         SOPs + cases       ├── numerical forecasting
                                           ├── delivery discrepancy detection
                                           └── LangGraph persisted workflow
                                                 │
                                      LangChain tools / optional LLM
                                                 │
                                         Go action APIs

Go/Python traces → OpenTelemetry Collector → Tempo → Grafana
Go/Python metrics → Prometheus → Grafana
```

### Repository layout

```text
backend/cmd/server/       Minimal executable entry point
backend/internal/        Application, configuration, domain policies, controllers, persistence, events
intelligence/app/         Composition root and HTTP routes
intelligence/app/services/ Forecasting, diagnosis, bounded LLM reasoning, recovery verification
intelligence/app/workflows/ Persistent investigation state machine
intelligence/app/workers/ Kafka consumption and checkpoint reconciliation
intelligence/app/infrastructure/ Go API client and vector knowledge repository
intelligence/tests/       Forecast, diagnosis, and recovery verification tests
frontend/app/             Page composition and server-side API proxy
frontend/components/      Layout, investigations, inventory, forecasts, shared UI
frontend/hooks/           Dashboard loading, polling, selection, mutation feedback
frontend/lib/             OperationsClient, shared domain types, presentation helpers
db/migrations/           Database schema and constraints
db/seeds/                Demo restaurant, recipes, synthetic history, SOPs
infra/                   OTel, Prometheus, Tempo, Grafana provisioning
scripts/smoke_test.py     End-to-end API / event / graph validation
docs/API.md              API request contracts
docs/ARCHITECTURE.md      Reliability, state transitions, and implementation boundaries
```

## AI and retrieval modes

**No LLM key is required.** LangGraph still executes the real persisted workflow; a deterministic evidence engine evaluates the hypotheses and proposes a quantity based on receiving records. The dashboard identifies this mode.

To enable model-assisted diagnosis, set `OPENAI_API_KEY` and optionally `OPENAI_MODEL` in `.env`, then recreate the agent:

```bash
docker compose up -d intelligence
```

The model uses bounded, read-only LangChain tools. Its narrative supplements the deterministic quantity calculation. It cannot approve actions or edit inventory directly.

## Code organization and formatting

- **Go:** `Application` owns lifecycle; domain-specific controllers compose a shared transactional controller; storage and Kafka publishing use interfaces. Domain policies have no HTTP dependencies.
- **Python:** explicitly injected service classes, an `InvestigationWorkflow` whose nodes are methods, repository/client adapters, and separate Kafka/reconciliation workers. Process state lives in application objects rather than a global runtime dictionary.
- **React:** focused functional components, shared TypeScript interfaces, an `OperationsClient` class, and a dedicated dashboard hook.

Formatting conventions: Go `gofmt`, Python Ruff (100-column target), TypeScript/TSX/CSS Prettier (2-space indentation). Formatter configuration is included.

**Vector retrieval:** FastEmbed runs `sentence-transformers/all-MiniLM-L6-v2` locally, producing 384-dimensional embeddings stored in pgvector with a cosine HNSW index. Retrieved SOPs and verified incident summaries carry source references. Searches are scoped to the configured demo branch.

The embedding model downloads on first use and is cached in a named volume. If download or initialization fails, retrieval explicitly falls back to PostgreSQL lexical ranking, and `/status` reports the mode and error. Set `RETRIEVAL_MODE=lexical` for a deliberately offline run.

## Forecasting

The baseline estimates lunch and dinner usage from the same weekday over up to 12 weeks. A rolling-origin 14-day holdout compares its mean absolute error against a seven-day average. Forecasts are derived from **seeded synthetic demand history**; live order consumption is recorded in the ledger but automatic historical aggregation/retraining is a future extension.

The upper planning estimate is a heuristic buffer, not a calibrated prediction interval. Holiday effects, promotions, weather, and annual seasonality are not modeled in this MVP.

## Verification

```bash
# Numerical engine tests
docker compose exec intelligence python -m unittest discover -s tests -v

# Real API + PostgreSQL + Kafka + LangGraph workflow
python3 scripts/smoke_test.py --restart-agent

# Build checks: Go tests and compilation, Python tests, Next.js typechecked production build
docker compose build
```

The smoke test verifies authorization, concurrent idempotent receipts, rejection of over-receipts, recipe deductions, competing hypotheses, persisted approval resume, single execution, actual receipt, prevention of early closure, and final recovery. It adds traceable test records; it does not erase existing data. The test expects the Compose demo observation window of 10 seconds.

### Observation windows

Go defaults to **600 seconds** outside demo mode. Compose explicitly sets `DEMO_MODE=true` and a **10-second** window for an interactive demonstration. The agent and Go must use the same configured window. Three post-receipt measurements totaling at least 30 orders are required. The most recent three must meet the recovery thresholds, and the newest must be less than 15 minutes old.

## Operations

For frontend development outside Docker:

```bash
docker compose up -d backend intelligence
cd frontend
cp .env.example .env.local
npm ci
npm run dev
```

Keep the API tokens in `frontend/.env.local` consistent with the root `.env` if you customize them.

```bash
docker compose logs -f backend intelligence
docker compose restart intelligence
docker compose down
```

Database, Kafka, and embedding-cache data persist in named volumes. Initial SQL runs only when the database volume is first created. Future schema changes should be added as numbered migrations and applied to existing databases explicitly.

Grafana provisions **Foodwise / Operations** with API and agent metrics. Use its Tempo datasource to explore service traces. Business audit history is stored in PostgreSQL and shown in the incident workspace.

### Organization-managed TLS certificates

If your network uses an organization-managed certificate authority, provide its trusted PEM bundle through the optional Docker build secret. Certificate verification remains enabled:

```bash
# Keep the bundle within this workspace, e.g. the ignored .tools/ directory.
EXTRA_CA_CERT=.tools/company-ca.pem docker compose \
  -f docker-compose.yml -f docker-compose.certs.yml build
docker compose up -d
```

The build adds the supplied CA to Go/Python trust stores and uses it for Node dependency installation. CA bundles and secrets should not be committed.

## MVP boundaries

- One branch, one configured manager identity, and service-token authorization.
- Single-ingredient purchase orders; recipes can contain multiple ingredients.
- Automated incident detection currently targets completed-but-short deliveries.
- Stock recovery execution is an **internal request adapter**, not a live supplier order, payment, email, or inter-branch logistics integration.
- Replacement receipts currently require the full requested amount. They do not falsely mark the original supplier purchase order fulfilled.
- Rejected actions and failed two-hour recovery windows are escalated. The latter reopens evidence review before escalation; automatic second purchasing plans are not generated.
- Kitchen evidence is branch-level, so the diagnosis distinguishes correlated kitchen impact from proven dish-level causation.
- The worker runs as one coordinated process. Distributed graph scheduling, multi-tenant identity, live POS/supplier adapters, and additional anomaly families are extension points.
