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
    """Allow only app server -> AI server calls. If INTERNAL_TOKEN is empty, no check is done (local development only)."""
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
        # OCR on a scan takes seconds to tens of seconds, so run it in a thread to avoid blocking the event loop
        return await asyncio.to_thread(pdf_to_text, data, settings.ocr_enabled, settings.ocr_max_pages)
    except ExtractError as e:
        raise HTTPException(status_code=422, detail=str(e)) from e


@app.post("/v1/analyze", response_model=AnalysisResult, response_model_by_alias=True, dependencies=[Depends(require_token)])
async def analyze_pdf(file: UploadFile = File(...)):
    text = await read_pdf_text(file)
    return await analyze_text(text, app.state.provider, settings.llm_concurrency)


@app.post("/v1/analyze/stream", dependencies=[Depends(require_token)])
async def analyze_pdf_stream(file: UploadFile = File(...)):
    """Stream progress as line-delimited JSON (NDJSON). Format errors are rejected with 4xx before the stream starts.
    {"type":"progress","stage":"reading|ocr|splitting|classifying|scoring","done":n,"total":m}
    {"type":"result","data":{...analysis result...}} or {"type":"error","detail":"..."} 로 끝난다."""
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
            if needs_ocr:  # Scan: read text via OCR, reporting progress page by page
                def on_page(done: int, total: int) -> None:
                    loop.call_soon_threadsafe(queue.put_nowait, {"type": "progress", "stage": "ocr", "done": done, "total": total})

                text = await asyncio.to_thread(ocr_text, data, settings.ocr_max_pages, on_page)
            result = await analyze_text(text, app.state.provider, settings.llm_concurrency, on_progress)
            await queue.put({"type": "result", "data": result.model_dump(mode="json", by_alias=True)})
        except ExtractError as e:  # Unreadable file: the message is safe to show to the user
            await queue.put({"type": "error", "detail": str(e), "readable": True})
        except Exception as e:  # Failures mid-stream are reported as an error event
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
            task.cancel()  # If the client disconnects, stop the analysis too

    return StreamingResponse(events(), media_type="application/x-ndjson")


@app.post("/v1/analyze/text", response_model=AnalysisResult, response_model_by_alias=True, dependencies=[Depends(require_token)])
async def analyze_plain_text(body: TextRequest):
    """Analyze text directly without a PDF. For prompt experiments and integration checks during app development."""
    return await analyze_text(body.text, app.state.provider, settings.llm_concurrency)
