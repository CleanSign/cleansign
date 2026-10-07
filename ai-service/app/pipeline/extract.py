import io

from pypdf import PdfReader
from pypdf.errors import PdfReadError


class ExtractError(Exception):
    pass


def pdf_to_text(data: bytes) -> str:
    """텍스트 PDF만 지원한다. 스캔/촬영본(텍스트 층 없음)은 OCR 단계(Phase 5)가 생기기 전까지 거절한다."""
    try:
        reader = PdfReader(io.BytesIO(data))
        text = "\n".join((page.extract_text() or "") for page in reader.pages)
    except (PdfReadError, ValueError, OSError) as e:
        raise ExtractError("PDF를 열 수 없어요. 파일이 손상됐는지 확인해 주세요") from e
    if len(text.strip()) < 20:
        raise ExtractError("글자를 읽지 못했어요. 사진이나 스캔한 계약서는 아직 지원하지 않아요. 글자가 들어 있는 PDF를 올려 주세요")
    return text
