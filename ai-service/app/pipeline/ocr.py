import logging
import threading
from collections.abc import Callable

import pypdfium2 as pdfium

logger = logging.getLogger("cleansign.ocr")

_engine = None
_lock = threading.Lock()


def _get_engine():
    """RapidOCR(PP-OCRv5 한글 인식 모델, onnxruntime). 처음 쓸 때 한 번만 올리고, 모델 파일은 첫 실행 때 내려받는다."""
    global _engine
    with _lock:
        if _engine is None:
            from rapidocr import LangRec, ModelType, OCRVersion, RapidOCR

            _engine = RapidOCR(
                params={
                    "Global.log_level": "warning",
                    "Rec.lang_type": LangRec.KOREAN,
                    "Rec.ocr_version": OCRVersion.PPOCRV5,
                    "Rec.model_type": ModelType.MOBILE,
                }
            )
        return _engine


def ocr_pdf(data: bytes, max_pages: int, scale: float = 2.5, on_page: Callable[[int, int], None] | None = None) -> str:
    """텍스트 층이 없는 PDF를 쪽마다 이미지로 그려 한글 OCR로 읽는다. 앞쪽 max_pages쪽까지만 읽는다."""
    import numpy as np

    pdf = pdfium.PdfDocument(data)
    try:
        total = min(len(pdf), max_pages)
        engine = _get_engine()
        pages: list[str] = []
        for i in range(total):
            image = np.array(pdf[i].render(scale=scale).to_pil().convert("RGB"))
            result = engine(image)
            pages.append("\n".join(result.txts) if result.txts else "")
            logger.info("ocr page %d/%d", i + 1, total)
            if on_page:
                on_page(i + 1, total)
        return "\n".join(pages)
    finally:
        pdf.close()
