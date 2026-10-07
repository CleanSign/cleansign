# Clean Sign 로컬 개발 환경
#
#   make run                 DB + Ollama + AI 서버 + 앱을 한 번에 띄운다 (Ctrl+C로 한꺼번에 종료)
#   make run LLM=mock        Ollama 없이 키워드 분류기로 띄운다 (UI/연동 확인용, 빠름)
#   make run PORT=3100       앱 포트 변경 (BETTER_AUTH_URL도 같이 맞춘다)
#   make status              지금 무엇이 떠 있는지 본다
#   make stop                make run으로 남은 앱/AI 서버를 포트로 찾아 끈다
#   make stop-ollama         모델을 내리고 `ollama serve`를 끈다
#   make check               타입 체크 + 린트 + AI 서버 테스트
#
# 변수: PORT(앱) AI_PORT(AI 서버) LLM(ollama|mock) MODEL(Ollama 모델) MOCK_DELAY_S(mock 지연, 진행 화면 확인용)

SHELL := /bin/bash
PORT ?= 3000
AI_PORT ?= 8000
LLM ?= ollama
MODEL ?= gemma4:12b
MOCK_DELAY_S ?= 0
OLLAMA_URL ?= http://localhost:11434

.PHONY: help run setup stop stop-ollama status check

help:
	@sed -n '3,12p' Makefile | sed 's/^# \{0,1\}//'

# 의존성이 없을 때만 설치한다
setup:
	@[ -d node_modules ] || { echo "▶ npm install"; npm install; }
	@command -v uv >/dev/null || { echo "uv가 필요해요: https://docs.astral.sh/uv/"; exit 1; }
	@[ -d ai-service/.venv ] || { echo "▶ uv sync"; (cd ai-service && uv sync); }
	@[ -f .env ] || echo "⚠ .env가 없어요: cp .env.example .env 후 값을 채워 주세요"

run: setup
	@set -e; \
	port_busy() { lsof -nP -iTCP:$$1 -sTCP:LISTEN >/dev/null 2>&1; }; \
	prefix() { awk -v p="[$$1]" '{ print p " " $$0; fflush() }'; }; \
	for p in $(PORT) $(AI_PORT); do \
	  if port_busy $$p; then echo "포트 $$p 가 이미 사용 중이에요. 쓰던 서버를 끄거나(make stop) 포트를 바꿔 주세요 (예: make run PORT=3100)"; exit 1; fi; \
	done; \
	docker info >/dev/null 2>&1 || { echo "Docker가 꺼져 있어요. Docker Desktop을 켜 주세요"; exit 1; }; \
	echo "▶ DB (postgres)"; npm run -s local:up; \
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
	echo "▶ 앱 :$(PORT)"; \
	( BETTER_AUTH_URL=http://localhost:$(PORT) AI_SERVICE_URL=http://localhost:$(AI_PORT) \
	  npm run dev -- -p $(PORT) 2>&1 | prefix next ) & NEXT_PID=$$!; \
	echo; echo "  앱        http://localhost:$(PORT)"; echo "  AI 서버   http://localhost:$(AI_PORT)/healthz"; \
	echo "  종료      Ctrl+C   (로그인 OTP는 [next] 로그에 나와요)"; echo; \
	while kill -0 $$AI_PID 2>/dev/null && kill -0 $$NEXT_PID 2>/dev/null; do sleep 2; done; \
	echo "프로세스가 종료돼서 전체를 끕니다"; exit 1

status:
	@printf "DB        "; docker ps --filter name=cleansign-db --format '{{.Status}}' | grep . || echo "꺼짐"
	@printf "Ollama    "; curl -sf $(OLLAMA_URL)/api/version >/dev/null && echo "실행 중 (모델: $$(curl -s $(OLLAMA_URL)/api/ps | python3 -c 'import sys,json;print(", ".join(m["name"] for m in json.load(sys.stdin)["models"]) or "로드 안 됨")'))" || echo "꺼짐"
	@printf "AI 서버   "; curl -sf localhost:$(AI_PORT)/healthz 2>/dev/null || echo "꺼짐 (:$(AI_PORT))"
	@printf "\n앱        "; lsof -nP -iTCP:$(PORT) -sTCP:LISTEN >/dev/null 2>&1 && echo "실행 중 (:$(PORT))" || echo "꺼짐 (:$(PORT))"

# 앱/AI 서버가 쓰는 포트의 리스너만 끈다 (무엇을 끄는지 먼저 보여 준다). DB와 Ollama는 건드리지 않는다.
stop:
	@for p in $(PORT) $(AI_PORT); do \
	  for pid in $$(lsof -nP -iTCP:$$p -sTCP:LISTEN -t 2>/dev/null); do \
	    echo "포트 $$p 종료: pid $$pid ($$(ps -o command= -p $$pid | cut -c1-60))"; kill $$pid; \
	  done; \
	done; true

stop-ollama:
	@ollama stop $(MODEL) >/dev/null 2>&1 || true
	@pkill -f "ollama serve" && echo "ollama serve 종료" || echo "실행 중인 ollama serve 없음"

check:
	npx tsc --noEmit
	npx eslint src
	cd ai-service && uv run pytest -q
