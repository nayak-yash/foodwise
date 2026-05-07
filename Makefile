.PHONY: up down logs test smoke format
up:
	docker compose up --build -d
down:
	docker compose down
logs:
	docker compose logs -f backend intelligence
test:
	docker compose build backend intelligence frontend
smoke:
	python3 scripts/smoke_test.py --restart-agent
format:
	cd backend && gofmt -w cmd internal
	cd intelligence && ruff check --fix app tests && ruff format app tests
	cd frontend && npm run format
