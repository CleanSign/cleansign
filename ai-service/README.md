# Clean Sign AI 서버

계약서(PDF 또는 텍스트)를 받아 위험 조항, 조항별 예상 손실액, 계약 위험 점수를 JSON으로 돌려주는 Python(FastAPI) 서버입니다. 설계 배경은 [`docs/06-ai-analysis-plan.md`](../docs/06-ai-analysis-plan.md).

## 실행

```bash
cd ai-service
cp .env.example .env
uv sync
uv run uvicorn app.main:app --port 8000      # Ollama가 켜져 있어야 함 (기본 모델 gemma4:12b)
LLM_PROVIDER=mock uv run uvicorn app.main:app # 모델 없이 키워드 분류로 구동
LLM_PROVIDER=mock MOCK_DELAY_S=2 uv run uvicorn app.main:app # 조항당 2초 지연: 진행 화면 시험용
uv run pytest -q
```

## API

| 메서드 | 경로 | 설명 |
|---|---|---|
| GET | `/healthz` | 상태, 현재 모델 계층 |
| POST | `/v1/analyze` | `multipart/form-data` `file`(PDF, 텍스트 층 필요). 분석 결과 반환 |
| POST | `/v1/analyze/stream` | `/v1/analyze`와 같지만 진행 단계(`reading → splitting → classifying N/M → scoring`)를 줄 단위 JSON(NDJSON)으로 흘리고 `result` 이벤트로 끝난다. 앱은 이 경로를 쓴다 |
| POST | `/v1/analyze/text` | `{"text": "..."}` 로 같은 분석 (개발·실험용) |

`INTERNAL_TOKEN`을 설정하면 `X-Internal-Token` 헤더가 필요합니다. 앱 서버에서만 호출하고 브라우저에 노출하지 마세요.
응답 모양은 `app/schemas.py`(camelCase)가 기준입니다. 앱의 `contract` / `contract_danger_clause` 스키마와 대응합니다.

## 구조

```
app/main.py            FastAPI, 토큰 검사, 업로드 검증
app/llm/               모델 계층: base(인터페이스) · ollama · mock · factory   ← 모델 교체는 여기만
app/pipeline/          extract(PDF) → mask(개인정보) → split(조항) → facts(보증금/월세) → quantify → run
app/policy.py          정량화 정책 상수와 유형별 고정 문구 (v0-placeholder)
```

모델을 바꾸려면 `LLMProvider`(`classify(clause)`)를 구현해 `factory.py`에 등록하고 `LLM_PROVIDER`를 바꿉니다. 외부 API나 별도 GPU 서버로 옮겨도 파이프라인은 그대로입니다.

## 지금 되는 것 / 안 되는 것

되는 것: 텍스트 PDF 추출, 개인정보 마스킹, 조항 분리(규칙), 숫자 표기 보증금·월세 추출, LLM 유형 분류(조항별 1회 호출), 손실 산출, 점수, 부분 실패 시 `warnings`로 보고.

아직 없는 것 (응답 필드는 있고 값이 비어 있음)
- **OCR**: 텍스트 층이 없는 스캔 PDF는 RapidOCR(PP-OCRv5 한글 인식 모델, onnxruntime, CPU)로 읽습니다. 쪽당 약 2초(M2 Pro), 첫 실행 때 모델 약 13MB를 내려받습니다(오프라인 배포는 `rapidocr/models/`를 이미지에 포함). 스트림에는 `stage: "ocr"` 진행 이벤트가 쪽마다 나옵니다. `OCR_ENABLED=false`면 스캔본을 422로 거절하고, `OCR_MAX_PAGES`(기본 15)쪽까지만 읽습니다. 이미지 파일·사진(JPG/PNG) 업로드는 아직 PDF만 받습니다. 띄어쓰기가 일부 빠지고 드물게 글자가 틀립니다.
- **RAG / 법령 MCP**: `precedents`는 항상 `[]`. `lawReference`, `reason`, `safeText`는 유형별 **고정 문구**이고 조항 내용을 읽고 쓴 것이 아닙니다. 조문 번호는 검증 전입니다.
- **위험 흐름도**: `flow`는 `null`.
- **한글 수사 금액**(예: 일억오천만원): 읽지 못하고 `deposit`이 `null`이 됩니다.

## 알려진 문제

- **점수가 낮게 나옵니다.** `riskScore`는 `신뢰도 × (L_max/보증금) × 인정비율`을 noisy-OR로 합치는데, 이 값들이 작아서 위험 조항이 2개여도 한 자릿수가 나옵니다(실측: 원상복구 + 갱신청구권 포기 계약서 → 3점, low). 정책 상수가 PLACEHOLDER인 데다 점수 정의가 금액 비율에만 의존하는 문제입니다. 사용자에게 보여 주기 전에 `policy.py`와 `quantify.contract_score`를 다시 정해야 합니다.
- **속도**: gemma4:12b 기준 조항당 약 7초(4조항 계약서 약 29초). 로컬 Ollama는 개발·평가용입니다.
- **정확도 미검증**: 분류 프롬프트는 라벨 조항 16개(임시 정상 4개 포함)로만 시험했습니다.
