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
    ocr_enabled: bool = True  # Read scanned files with no text layer via OCR (if off, scans are rejected)
    ocr_max_pages: int = 15
    mock_delay_s: float = 0  # Adds a per-clause delay to the mock model, used to develop and test the progress screen


settings = Settings()
