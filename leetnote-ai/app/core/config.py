"""服务配置。

所有配置来自环境变量（12-Factor），本地开发时从 .env 读取。
"""

from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict

# 对话模型的默认端点与模型。
# 默认用【智谱 GLM-4.5-Flash】—— 永久免费、国内直连、编码/中文都强。
# .env 里把这两项留空时回退到这里，避免「空字符串覆盖了默认值」导致 base_url=""。
DEFAULT_CHAT_BASE_URL = "https://open.bigmodel.cn/api/paas/v4"
DEFAULT_CHAT_MODEL = "glm-4.5-flash"

# OpenCode Go 订阅的端点。它不是默认值，但代码里有几处需要按供应商特判：
#   - 两个硬性要求（自带 User-Agent + 稳定的 x-opencode-session 头）
#   - key 可以从本机 opencode 凭据库里自动发现
# 见 app/llm/__init__.py 与 app/llm/discovery.py。
OPENCODE_GO_BASE_URL = "https://opencode.ai/zen/go/v1"


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

    # ---------- Embedding ----------
    # 留空则回退到本地哈希向量（见 embedding/local.py），保证没有 API Key 也能跑通链路。
    llm_api_key: str = ""
    llm_base_url: str = "https://api.siliconflow.cn/v1"
    embedding_model: str = ""

    # 向量维度必须与 note_embeddings.embedding 的列定义一致（vector(1536)）。
    # 换用维度不同的模型时需要先做一次迁移改列类型。
    embedding_dim: int = 1536

    # ---------- 对话模型（LLM 生成讲解）----------
    # 与 embedding 分开配置：两者常常来自不同供应商。
    # 默认指向【智谱 GLM-4.5-Flash】（免费）；改成 OpenCode Go 只需换 base_url + model。
    chat_api_key: str = ""
    chat_base_url: str = "https://open.bigmodel.cn/api/paas/v4"
    chat_model: str = "glm-4.5-flash"

    # OpenCode Go 对第三方客户端有两条硬性要求
    # （见 https://opencode.ai/docs/go/#where-can-i-use-it）：
    #   1. 用【自己的】User-Agent 标识，不能是通用 SDK / HTTP 库的名字
    #   2. 每个会话带稳定的会话 ID，服务端据此优化路由与 prompt 缓存
    #      不带这个头会直接返回 503 "Request is missing x-opencode-session"
    #
    # 智谱等其它供应商不需要这两条。默认留空；
    # 若 base_url 指向 OpenCode Go，工厂会自动补上 x-opencode-session（见 llm/__init__.py）。
    chat_user_agent: str = "leetnote-ai/0.1.0"
    chat_session_header: str = ""

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
    def use_chat_model(self) -> bool:
        """是否启用了对话模型（只看 .env；实际是否可用还要看自动发现，见 llm 工厂）。"""
        return bool(self.chat_api_key and self.chat_model)

    @property
    def effective_chat_base_url(self) -> str:
        """有效端点：.env 留空则回退到默认值（智谱）。"""
        return self.chat_base_url.strip() or DEFAULT_CHAT_BASE_URL

    @property
    def effective_chat_model(self) -> str:
        """有效模型：.env 留空则回退到默认模型（GLM-4.5-Flash）。"""
        return self.chat_model.strip() or DEFAULT_CHAT_MODEL

    @property
    def uses_opencode_go(self) -> bool:
        """当前对话端点是不是 OpenCode Go。

        OpenCode Go 需要额外的会话头，且 key 可从本机 opencode 凭据自动发现；
        其它供应商（如智谱）不需要 —— 判断依据只看端点主机，避免把
        OpenCode Go 的 key 误发给别的平台。
        """
        return "opencode.ai" in self.effective_chat_base_url.lower()

    @property
    def is_production(self) -> bool:
        return self.app_env == "production"


@lru_cache
def get_settings() -> Settings:
    """带缓存的配置单例。"""
    return Settings()
