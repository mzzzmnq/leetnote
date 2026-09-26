"""日志配置。

用标准库 logging，而不是引入 loguru/structlog：
这个服务的日志量很小，标准库完全够用，少一个依赖少一份维护成本。
"""

import logging
import sys


def setup_logging(app_env: str = "development") -> None:
    level = logging.INFO if app_env == "production" else logging.DEBUG

    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(
        logging.Formatter(
            fmt="%(asctime)s %(levelname)-5s %(name)s: %(message)s",
            datefmt="%Y-%m-%d %H:%M:%S",
        )
    )

    root = logging.getLogger()
    root.handlers.clear()
    root.addHandler(handler)
    root.setLevel(level)

    # uvicorn 的访问日志单独控制，避免 DEBUG 时刷屏
    logging.getLogger("uvicorn.access").setLevel(logging.INFO)
