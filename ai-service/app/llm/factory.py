from app.config import Settings
from app.llm.base import LLMProvider
from app.llm.mock import MockProvider
from app.llm.ollama import OllamaProvider


def build_provider(s: Settings) -> LLMProvider:
    if s.llm_provider == "mock":
        return MockProvider(s.mock_delay_s)
    if s.llm_provider == "ollama":
        return OllamaProvider(s.ollama_url, s.ollama_model, s.llm_timeout_s)
    raise ValueError(f"LLM_PROVIDER={s.llm_provider!r} 는 지원하지 않습니다 (ollama | mock)")
