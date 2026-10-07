import asyncio
import hmac
import json
from contextlib import asynccontextmanager

from fastapi import Depends, FastAPI, File, Header, HTTPException, UploadFile
from fastapi.responses import StreamingResponse

from app.config import settings
from app.llm.factory import build_provider
from app.pipeline.extract import ExtractError, pdf_to_text
from app.pipeline.run import analyze_text
from app.schemas import AnalysisResult, TextRequest


@asynccontextmanager
async def lifespan(app: FastAPI):
    app.state.provider = build_provider(settings)
    yield
    await app.state.provider.aclose()


app = FastAPI(title="Clean Sign AI", version="0.1.0", lifespan=lifespan)


def require_token(x_internal_token: str | None = Header(default=None)) -> None:
    """앱 서버 → AI 서버 호출만 허용한다. INTERNAL_TOKEN이 비어 있으면 검사하지 않는다(로컬 개발 전용)."""
    if not settings.internal_token:
        return
    if not x_internal_token or not hmac.compare_digest(x_internal_token, settings.internal_token):
        raise HTTPException(status_code=401, detail="unauthorized")


@app.get("/healthz")
async def healthz():
    return {"status": "ok", "provider": app.state.provider.name, "model": getattr(app.state.provider, "model", None)}


async def read_pdf_text(file: UploadFile) -> str:
    if file.content_type not in ("application/pdf", "application/octet-stream") and not (file.filename or "").lower().endswith(".pdf"):
        raise HTTPException(status_code=415, detail="PDF 파일만 업로드할 수 있습니다.")
    data = await file.read(settings.max_upload_mb * 1024 * 1024 + 1)
    if len(data) > settings.max_upload_mb * 1024 * 1024:
        raise HTTPException(status_code=413, detail=f"파일은 {settings.max_upload_mb}MB 이하여야 합니다.")
    try:
        return pdf_to_text(data)
    except ExtractError as e:
        raise HTTPException(status_code=422, detail=str(e)) from e


@app.post("/v1/analyze", response_model=AnalysisResult, response_model_by_alias=True, dependencies=[Depends(require_token)])
async def analyze_pdf(file: UploadFile = File(...)):
    text = await read_pdf_text(file)
    return await analyze_text(text, app.state.provider, settings.llm_concurrency)


@app.post("/v1/analyze/stream", dependencies=[Depends(require_token)])
async def analyze_pdf_stream(file: UploadFile = File(...)):
    """진행 상황을 줄 단위 JSON(NDJSON)으로 흘려 보낸다. 형식 오류는 스트림 시작 전에 4xx로 거절한다.
    {"type":"progress","stage":"reading|splitting|classifying|scoring","done":n,"total":m}
    {"type":"result","data":{...분석 결과...}} 또는 {"type":"error","detail":"..."} 로 끝난다."""
    text = await read_pdf_text(file)
    queue: asyncio.Queue[dict | None] = asyncio.Queue()

    async def on_progress(stage: str, done: int, total: int) -> None:
        await queue.put({"type": "progress", "stage": stage, "done": done, "total": total})

    async def work() -> None:
        try:
            result = await analyze_text(text, app.state.provider, settings.llm_concurrency, on_progress)
            await queue.put({"type": "result", "data": result.model_dump(mode="json", by_alias=True)})
        except Exception as e:  # 스트림 도중의 실패는 error 이벤트로 알린다
            await queue.put({"type": "error", "detail": f"{type(e).__name__}: {e}"})
        finally:
            await queue.put(None)

    async def events():
        task = asyncio.create_task(work())
        try:
            yield json.dumps({"type": "progress", "stage": "reading", "done": 0, "total": 0}) + "\n"
            while (item := await queue.get()) is not None:
                yield json.dumps(item, ensure_ascii=False) + "\n"
        finally:
            task.cancel()  # 클라이언트가 끊으면 분석도 멈춘다

    return StreamingResponse(events(), media_type="application/x-ndjson")


@app.post("/v1/analyze/text", response_model=AnalysisResult, response_model_by_alias=True, dependencies=[Depends(require_token)])
async def analyze_plain_text(body: TextRequest):
    """PDF 없이 텍스트로 바로 분석한다. 프롬프트 실험과 앱 개발 중 연동 확인용."""
    return await analyze_text(body.text, app.state.provider, settings.llm_concurrency)
