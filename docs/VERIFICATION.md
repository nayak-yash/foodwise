# Local verification

## Passed

- Go unit tests and compilation, both natively and inside the backend Docker build.
- Python numerical tests: weekday/weekend forecasts, source-based delivery diagnosis, rejection of premature/stale/backdated recovery evidence.
- Ruff lint/import organization and formatting for Python.
- Prettier formatting and the Next.js production build with TypeScript checking.
- Real Go + PostgreSQL + Kafka + LangGraph end-to-end smoke test, including an agent restart at its approval checkpoint.
- Browser sign-in, incident simulation, manager approval, stock-receipt simulation, and recovery verification.
- Browser console: no errors during the checked workflow.
- FastEmbed model initialization and pgvector indexing of SOPs and a verified resolved case; agent status reported `retrieval_mode: vector` with no embedding error.

## Environment notes

The Go and Python services and supporting infrastructure run in Docker. The checked dashboard runs from its local Next.js standalone production build at port 3000.

The frontend Docker build encountered a registry HTTP `403` for an npm dependency. Native npm installation and the production build succeeded. The standard frontend Dockerfile and lockfile are included; its full container build remains unverified on this network.

No OpenAI API key was configured, so model-assisted tool calling has not been exercised against a live LLM. The verified workflow uses the explicitly labeled deterministic evidence mode with real LangGraph checkpointing and vector retrieval.

Run `python3 scripts/smoke_test.py --restart-agent` against the provided demo configuration to reproduce the integration check. It creates additional traceable records without deleting prior data.
