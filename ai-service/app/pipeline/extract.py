import io
from collections.abc import Callable

from pypdf import PdfReader
from pypdf.errors import PdfReadError

MIN_TEXT_CHARS = 20


class ExtractError(Exception):
    """사용자에게 그대로 보여 줘도 되는 메시지를 담는다."""


def text_layer(data: bytes) -> str:
    """PDF에 들어 있는 글자를 읽는다. 스캔본이면 거의 빈 문자열이 나온다."""
    try:
        reader = PdfReader(io.BytesIO(data))
        return "\n".join((page.extract_text() or "") for page in reader.pages)
    except (PdfReadError, ValueError, OSError) as e:
        raise ExtractError("PDF를 열 수 없어요. 파일이 손상됐는지 확인해 주세요") from e


def has_text(text: str) -> bool:
    return len(text.strip()) >= MIN_TEXT_CHARS


def ocr_text(data: bytes, max_pages: int, on_page: Callable[[int, int], None] | None = None) -> str:
    """스캔본을 OCR로 읽는다. 오래 걸리니 스레드에서 부른다."""
    from app.pipeline.ocr import ocr_pdf

    try:
        text = ocr_pdf(data, max_pages, on_page=on_page)
    except ExtractError:
        raise
    except Exception as e:
        raise ExtractError("스캔한 계약서를 읽는 중 문제가 생겼어요. 글자가 들어 있는 PDF로 다시 올려 주세요") from e
    if not has_text(text):
        raise ExtractError("글자를 읽지 못했어요. 사진이 흐리거나 계약서가 아닌 파일일 수 있어요")
    return text


def pdf_to_text(data: bytes, ocr_enabled: bool = False, max_pages: int = 15) -> str:
    """텍스트 PDF는 바로, 스캔본은 OCR이 켜져 있을 때만 OCR로 읽는다."""
    text = text_layer(data)
    if has_text(text):
        return text
    if not ocr_enabled:
        raise ExtractError("글자를 읽지 못했어요. 사진이나 스캔한 계약서는 지원하지 않아요. 글자가 들어 있는 PDF를 올려 주세요")
    return ocr_text(data, max_pages)
