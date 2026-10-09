import logging
import threading
from collections.abc import Callable

import pypdfium2 as pdfium

logger = logging.getLogger("cleansign.ocr")

_engine = None
_lock = threading.Lock()


def _get_engine():
    """RapidOCR (PP-OCRv5 Korean recognition model, onnxruntime). Loaded once on first use; the model files are downloaded on first run."""
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
    """Render each page of a PDF with no text layer as an image and read it with Korean OCR. Reads only up to the first max_pages pages."""
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
