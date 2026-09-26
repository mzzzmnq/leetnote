"""开发/生产统一入口。

【为什么需要这个文件，而不是直接 `uvicorn app.main:app`】

Windows 上 asyncio 默认使用 ProactorEventLoop，而 psycopg3 的异步模式依赖
add_reader/add_writer —— Proactor 没有这两个 API。结果是连接池**一个连接
都建不起来，启动时静默卡死**，只在日志里反复刷：

    Psycopg cannot use the 'ProactorEventLoop' to run in async mode.

试过但【不管用】的办法：
  - 在 app/main.py 里 set_event_loop_policy：
    uvicorn 的顺序是「建事件循环 → 导入 app」，导入太晚。
  - 在 run.py 顶部 set_event_loop_policy：
    uvicorn 0.53 已经不用事件循环策略了，改用 loop factory，
    而它内部是这么写的（见 uvicorn/loops/asyncio.py）：

        def asyncio_loop_factory(use_subprocess=False):
            if sys.platform == "win32" and not use_subprocess:
                return asyncio.ProactorEventLoop   # 硬编码 Proactor
            return asyncio.SelectorEventLoop

    也就是说策略被完全绕过了。

【本文件的做法】
不调用 `uvicorn.run()` / `Server.run()`（它们会套用上面的 factory），
而是自己在 asyncio.run() 里跑 `Server.serve()` —— 事件循环由
asyncio.run() 按当前策略创建，于是 SelectorEventLoop 生效。

代价：`--reload` 用不了（那需要 uvicorn 的 supervisor 子进程）。
改代码后手动重启即可，这个服务的改动频率很低。
"""

from __future__ import annotations

import argparse
import asyncio
import sys

# 必须在 asyncio.run() 之前设置，所以放在模块最顶部
if sys.platform == "win32":
    asyncio.set_event_loop_policy(asyncio.WindowsSelectorEventLoopPolicy())

import uvicorn  # noqa: E402


def main() -> None:
    parser = argparse.ArgumentParser(description="启动 leetnote-ai")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=8000)
    parser.add_argument("--log-level", default="info")
    args = parser.parse_args()

    config = uvicorn.Config(
        "app.main:app",
        host=args.host,
        port=args.port,
        log_level=args.log_level,
    )
    server = uvicorn.Server(config)

    print(f"启动 leetnote-ai: http://{args.host}:{args.port}")
    print(f"事件循环策略: {type(asyncio.new_event_loop()).__name__}")

    # 关键：不调 server.run()，自己在当前策略创建的事件循环上跑 serve()
    asyncio.run(server.serve())


if __name__ == "__main__":
    main()
