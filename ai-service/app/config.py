from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    llm_provider: str = "ollama"  # ollama | mock
    ollama_url: str = "http://localhost:11434"
    ollama_model: str = "gemma4:12b"
    llm_timeout_s: float = 60
    llm_concurrency: int = 1
    internal_token: str = ""
    max_upload_mb: int = 10
    mock_delay_s: float = 0  # mock 모델에 조항당 지연을 줘서 진행 화면을 개발·시험할 때 쓴다


settings = Settings()
