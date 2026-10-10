# Clean Sign local development environment
#
#   make run                 Start DB + Ollama + AI server + Go server + web at once (Ctrl+C stops everything)
#   make run LLM=mock        Start with the keyword classifier instead of Ollama (for checking UI/integration, fast)
#   make run PORT=3100       Change the web port
#   make status              Show what is currently running
#   make stop                Find and stop web/server/AI server left over from make run, by port
#   make stop-ollama         Unload the model and stop `ollama serve`
#   make check               Go (vet, tests, lint) + web (types, lint) + AI server tests
#   make gen                 Regenerate code: handler interfaces, DB code, web types, Service interfaces, test mocks
#   make migration NAME=x    Create a new migration file (server/internal/database/migrations)
#   make domain NAME=article Scaffold a new domain (server/internal/domain, service/article)
#   make db-reset            Drop and recreate the dev DB (cleansign) (all data is lost)
#
# Variables: PORT(web) SERVER_PORT(Go server) AI_PORT(AI server) LLM(ollama|mock) MODEL(Ollama model) MOCK_DELAY_S(mock delay, for checking the progress screen)
#
# Server settings live in server/.env (copied from server/.env.example if missing); web settings live in web/.env.
# If SMTP is not configured, the login verification code appears in the [server] log.

SHELL := /bin/bash
PORT ?= 3000
SERVER_PORT ?= 4000
AI_PORT ?= 8000
LLM ?= ollama
MODEL ?= gemma4:12b
MOCK_DELAY_S ?= 0
OLLAMA_URL ?= http://localhost:11434

.PHONY: help run setup gen migration domain db-reset stop stop-ollama status check

help:
	@sed -n '3,15p' Makefile | sed 's/^# \{0,1\}//'

# Install dependencies only when missing
setup:
	@[ -d web/node_modules ] || { echo "▶ npm install"; (cd web && npm install); }
	@command -v go >/dev/null || { echo "Go가 필요해요: https://go.dev/dl/"; exit 1; }
	@command -v uv >/dev/null || { echo "uv가 필요해요: https://docs.astral.sh/uv/"; exit 1; }
	@[ -d ai-service/.venv ] || { echo "▶ uv sync"; (cd ai-service && uv sync); }
	@[ -f server/.env ] || { cp server/.env.example server/.env; echo "▶ server/.env를 만들었어요 (필요하면 값을 고쳐 주세요)"; }
	@[ -f web/.env ] || { cp web/.env.example web/.env; echo "▶ web/.env를 만들었어요"; }

run: setup
	@set -e; \
	port_busy() { lsof -nP -iTCP:$$1 -sTCP:LISTEN >/dev/null 2>&1; }; \
	prefix() { awk -v p="[$$1]" '{ print p " " $$0; fflush() }'; }; \
	for p in $(PORT) $(SERVER_PORT) $(AI_PORT); do \
	  if port_busy $$p; then echo "포트 $$p 가 이미 사용 중이에요. 쓰던 서버를 끄거나(make stop) 포트를 바꿔 주세요 (예: make run PORT=3100)"; exit 1; fi; \
	done; \
	docker info >/dev/null 2>&1 || { echo "Docker가 꺼져 있어요. Docker Desktop을 켜 주세요"; exit 1; }; \
	echo "▶ DB (postgres)"; docker compose up -d --wait db; \
	docker compose exec -T db psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname='cleansign'" | grep -q 1 \
	  || docker compose exec -T db psql -U postgres -c "CREATE DATABASE cleansign" >/dev/null; \
	trap 'echo; echo "■ 종료 중..."; trap - INT TERM EXIT; kill 0' INT TERM EXIT; \
	if [ "$(LLM)" = "ollama" ]; then \
	  if curl -sf $(OLLAMA_URL)/api/version >/dev/null; then echo "▶ Ollama: 이미 실행 중 (그대로 사용)"; \
	  else \
	    echo "▶ Ollama 시작"; ( ollama serve 2>&1 | prefix ollama ) & \
	    for i in $$(seq 1 30); do curl -sf $(OLLAMA_URL)/api/version >/dev/null && break; sleep 1; done; \
	    curl -sf $(OLLAMA_URL)/api/version >/dev/null || { echo "Ollama가 시작되지 않았어요"; exit 1; }; \
	  fi; \
	  ollama list 2>/dev/null | awk 'NR>1{print $$1}' | grep -qx "$(MODEL)" || echo "⚠ 모델 $(MODEL) 이 없어요: ollama pull $(MODEL)"; \
	else echo "▶ LLM=$(LLM): Ollama는 띄우지 않아요"; fi; \
	echo "▶ AI 서버 :$(AI_PORT) ($(LLM))"; \
	( cd ai-service && LLM_PROVIDER=$(LLM) OLLAMA_URL=$(OLLAMA_URL) OLLAMA_MODEL=$(MODEL) MOCK_DELAY_S=$(MOCK_DELAY_S) \
	  uv run uvicorn app.main:app --port $(AI_PORT) 2>&1 | prefix ai ) & AI_PID=$$!; \
	for i in $$(seq 1 40); do curl -sf localhost:$(AI_PORT)/healthz >/dev/null && break; sleep 1; done; \
	curl -sf localhost:$(AI_PORT)/healthz >/dev/null || { echo "AI 서버가 시작되지 않았어요"; exit 1; }; \
	echo "▶ 서버 :$(SERVER_PORT)"; \
	( cd server && set -a && . ./.env && set +a && PORT=$(SERVER_PORT) AI_SERVICE_URL=http://localhost:$(AI_PORT) \
	  go run ./cmd/server 2>&1 | prefix server ) & SERVER_PID=$$!; \
	for i in $$(seq 1 60); do curl -sf localhost:$(SERVER_PORT)/healthz >/dev/null && break; sleep 1; done; \
	curl -sf localhost:$(SERVER_PORT)/healthz >/dev/null || { echo "서버가 시작되지 않았어요"; exit 1; }; \
	echo "▶ web :$(PORT)"; \
	( cd web && SERVER_URL=http://localhost:$(SERVER_PORT) npm run dev -- -p $(PORT) ) 2>&1 | prefix web & WEB_PID=$$!; \
	echo; echo "  web       http://localhost:$(PORT)"; echo "  서버      http://localhost:$(SERVER_PORT)/healthz"; echo "  AI 서버   http://localhost:$(AI_PORT)/healthz"; \
	echo "  종료      Ctrl+C   (로그인 인증코드는 [server] 로그에 나와요)"; echo; \
	while kill -0 $$AI_PID 2>/dev/null && kill -0 $$SERVER_PID 2>/dev/null && kill -0 $$WEB_PID 2>/dev/null; do sleep 2; done; \
	echo "프로세스가 종료돼서 전체를 끕니다"; exit 1

status:
	@printf "DB        "; docker ps --filter name=cleansign-db --format '{{.Status}}' | grep . || echo "꺼짐"
	@printf "Ollama    "; curl -sf $(OLLAMA_URL)/api/version >/dev/null && echo "실행 중 (모델: $$(curl -s $(OLLAMA_URL)/api/ps | python3 -c 'import sys,json;print(", ".join(m["name"] for m in json.load(sys.stdin)["models"]) or "로드 안 됨")'))" || echo "꺼짐"
	@printf "AI 서버   "; curl -sf localhost:$(AI_PORT)/healthz 2>/dev/null || echo "꺼짐 (:$(AI_PORT))"
	@printf "\n서버      "; curl -sf localhost:$(SERVER_PORT)/healthz 2>/dev/null || echo "꺼짐 (:$(SERVER_PORT))"
	@printf "\nweb       "; lsof -nP -iTCP:$(PORT) -sTCP:LISTEN >/dev/null 2>&1 && echo "실행 중 (:$(PORT))" || echo "꺼짐 (:$(PORT))"

# Regenerate code from its source of truth. Commit the generated output (CI checks that it is up to date)
#   API spec (openapi.yaml)      -> handler interface, web types
#   SQL (queries/*.sql)          -> DB query code
#   service structs              -> Service interfaces (service_interface.go)
#   ports + Service interfaces   -> test mocks (*_mock_test.go)
gen:
	@command -v sqlc >/dev/null || { echo "sqlc가 필요해요: brew install sqlc"; exit 1; }
	cd server && go generate ./... && sqlc generate
	@[ -d web/node_modules ] || (cd web && npm install)
	cd web && npm run -s gen:api

# Create a new migration file (the server applies it automatically on startup). Do not edit already-applied files; create a new one instead
migration:
	@[ -n "$(NAME)" ] || { echo "사용법: make migration NAME=add_article_table"; exit 1; }
	@command -v goose >/dev/null || { echo "goose가 필요해요: brew install goose"; exit 1; }
	goose -dir server/internal/database/migrations create "$(NAME)" sql

# Scaffold the new domain's domain/service and print what to do next
domain:
	@./scripts/new-domain.sh "$(NAME)"

# Drop and recreate the dev DB. Restarting the server reapplies migrations from scratch
db-reset:
	@docker info >/dev/null 2>&1 || { echo "Docker가 꺼져 있어요. Docker Desktop을 켜 주세요"; exit 1; }
	@docker compose up -d --wait db
	@read -p "개발 DB(cleansign)의 모든 데이터가 사라져요. 계속할까요? [y/N] " a && [ "$$a" = "y" ]
	docker compose exec -T db psql -U postgres -c "DROP DATABASE IF EXISTS cleansign WITH (FORCE)"
	docker compose exec -T db psql -U postgres -c "CREATE DATABASE cleansign"

# Stop only the listeners on the ports used by web/server/AI server (shows what it stops first). Leaves the DB and Ollama alone.
stop:
	@for p in $(PORT) $(SERVER_PORT) $(AI_PORT); do \
	  for pid in $$(lsof -nP -iTCP:$$p -sTCP:LISTEN -t 2>/dev/null); do \
	    echo "포트 $$p 종료: pid $$pid ($$(ps -o command= -p $$pid | cut -c1-60))"; kill $$pid; \
	  done; \
	done; true

stop-ollama:
	@ollama stop $(MODEL) >/dev/null 2>&1 || true
	@pkill -f "ollama serve" && echo "ollama serve 종료" || echo "실행 중인 ollama serve 없음"

check:
	cd server && go vet ./...
	cd server && go test ./...
	@if command -v golangci-lint >/dev/null; then cd server && golangci-lint run ./...; else echo "(golangci-lint가 없어 린트는 건너뜀: brew install golangci-lint)"; fi
	cd web && npx tsc --noEmit
	cd web && npx eslint src
	cd ai-service && uv run pytest -q
