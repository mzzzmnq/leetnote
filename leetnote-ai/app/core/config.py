"""服务配置。

所有配置来自环境变量（12-Factor），本地开发时从 .env 读取。
"""

from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    app_env: str = "development"
    http_port: int = 8000

    # 注意用 libpq 格式（postgresql://），不是 SQLAlchemy 的 postgresql+psycopg://
    database_url: str = "postgresql://leetnote:leetnote@localhost:5432/leetnote"

    # ---------- Embedding / LLM ----------
    # 留空则回退到本地哈希向量（见 embedding/local.py），保证没有 API Key 也能跑通链路。
    llm_api_key: str = ""
    llm_base_url: str = "https://api.siliconflow.cn/v1"
    embedding_model: str = ""

    # 向量维度必须与 note_embeddings.embedding 的列定义一致（vector(1536)）。
    # 换用维度不同的模型时需要先做一次迁移改列类型。
    embedding_dim: int = 1536

    # 检索返回的相似题数量上限
    similar_limit_default: int = 5
    similar_limit_max: int = 20

    # ---------- 服务间鉴权 ----------
    # 本服务【只允许内部调用】（由 Go 服务转发，鉴权在 Go 侧完成）。
    # 用共享密钥拦住直接访问，避免有人绕过 Go 直接打这个端口。
    # 生产环境务必改掉，并且不要把这个端口暴露到公网。
    internal_api_token: str = "dev-internal-token-change-me"

    @property
    def use_remote_embedding(self) -> bool:
        """是否启用远程 embedding 模型。"""
        return bool(self.llm_api_key and self.embedding_model)

    @property
    def is_production(self) -> bool:
        return self.app_env == "production"


@lru_cache
def get_settings() -> Settings:
    """带缓存的配置单例。"""
    return Settings()
