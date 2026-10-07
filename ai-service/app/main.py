import asyncio
import hmac
import json
from contextlib import asynccontextmanager

from fastapi import Depends, FastAPI, File, Header, HTTPException, UploadFile
from fastapi.responses import StreamingResponse

from app.config import settings
from app.llm.factory import build_provider
from app.pipeline.extract import ExtractError, has_text, ocr_text, pdf_to_text, text_layer
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


async def read_pdf_bytes(file: UploadFile) -> bytes:
    if file.content_type not in ("application/pdf", "application/octet-stream") and not (file.filename or "").lower().endswith(".pdf"):
        raise HTTPException(status_code=415, detail="PDF 파일만 업로드할 수 있습니다.")
    data = await file.read(settings.max_upload_mb * 1024 * 1024 + 1)
    if len(data) > settings.max_upload_mb * 1024 * 1024:
        raise HTTPException(status_code=413, detail=f"파일은 {settings.max_upload_mb}MB 이하여야 합니다.")
    return data


async def read_pdf_text(file: UploadFile) -> str:
    data = await read_pdf_bytes(file)
    try:
        # 스캔본이면 OCR이 몇 초~수십 초 걸리므로 이벤트 루프를 막지 않게 스레드에서 돌린다
        return await asyncio.to_thread(pdf_to_text, data, settings.ocr_enabled, settings.ocr_max_pages)
    except ExtractError as e:
        raise HTTPException(status_code=422, detail=str(e)) from e


@app.post("/v1/analyze", response_model=AnalysisResult, response_model_by_alias=True, dependencies=[Depends(require_token)])
async def analyze_pdf(file: UploadFile = File(...)):
    text = await read_pdf_text(file)
    return await analyze_text(text, app.state.provider, settings.llm_concurrency)


@app.post("/v1/analyze/stream", dependencies=[Depends(require_token)])
async def analyze_pdf_stream(file: UploadFile = File(...)):
    """진행 상황을 줄 단위 JSON(NDJSON)으로 흘려 보낸다. 형식 오류는 스트림 시작 전에 4xx로 거절한다.
    {"type":"progress","stage":"reading|ocr|splitting|classifying|scoring","done":n,"total":m}
    {"type":"result","data":{...분석 결과...}} 또는 {"type":"error","detail":"..."} 로 끝난다."""
    data = await read_pdf_bytes(file)
    try:
        text = await asyncio.to_thread(text_layer, data)
    except ExtractError as e:
        raise HTTPException(status_code=422, detail=str(e)) from e
    needs_ocr = not has_text(text)
    if needs_ocr and not settings.ocr_enabled:
        raise HTTPException(status_code=422, detail="글자를 읽지 못했어요. 사진이나 스캔한 계약서는 지원하지 않아요. 글자가 들어 있는 PDF를 올려 주세요")
    queue: asyncio.Queue[dict | None] = asyncio.Queue()
    loop = asyncio.get_running_loop()

    async def on_progress(stage: str, done: int, total: int) -> None:
        await queue.put({"type": "progress", "stage": stage, "done": done, "total": total})

    async def work() -> None:
        nonlocal text
        try:
            if needs_ocr:  # 스캔본: 쪽마다 진행 상황을 알리며 OCR로 글자를 읽는다
                def on_page(done: int, total: int) -> None:
                    loop.call_soon_threadsafe(queue.put_nowait, {"type": "progress", "stage": "ocr", "done": done, "total": total})

                text = await asyncio.to_thread(ocr_text, data, settings.ocr_max_pages, on_page)
            result = await analyze_text(text, app.state.provider, settings.llm_concurrency, on_progress)
            await queue.put({"type": "result", "data": result.model_dump(mode="json", by_alias=True)})
        except ExtractError as e:  # 읽을 수 없는 파일: 메시지를 사용자에게 보여 줘도 된다
            await queue.put({"type": "error", "detail": str(e), "readable": True})
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
