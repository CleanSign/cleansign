import json

import httpx

from app.llm.base import CLAUSE_TYPES, Classification, LLMError
from app.llm.prompts import SYSTEM_CLASSIFY

SCHEMA = {
    "type": "object",
    "properties": {
        "clause_type": {"type": "string", "enum": CLAUSE_TYPES},
        "confidence": {"type": "number"},
    },
    "required": ["clause_type", "confidence"],
}


class OllamaProvider:
    name = "ollama"

    def __init__(self, url: str, model: str, timeout_s: float):
        self.model = model
        self._client = httpx.AsyncClient(base_url=url, timeout=timeout_s)

    async def classify(self, clause: str) -> Classification:
        body = {
            "model": self.model,
            "stream": False,
            "think": False,
            "format": SCHEMA,
            "options": {"temperature": 0, "num_ctx": 4096},
            "messages": [
                {"role": "system", "content": SYSTEM_CLASSIFY},
                {"role": "user", "content": f"조항: {clause}"},
            ],
        }
        try:
            res = await self._client.post("/api/chat", json=body)
            res.raise_for_status()
            out = json.loads(res.json()["message"]["content"])
            ctype = out["clause_type"]
            if ctype not in CLAUSE_TYPES:
                raise ValueError(f"unknown clause_type {ctype!r}")
            return Classification(ctype, min(1.0, max(0.0, float(out["confidence"]))))
        except (httpx.HTTPError, KeyError, ValueError, json.JSONDecodeError) as e:
            raise LLMError(f"{type(e).__name__}: {e}") from e

    async def aclose(self) -> None:
        await self._client.aclose()
