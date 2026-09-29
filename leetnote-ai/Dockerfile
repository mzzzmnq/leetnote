# =============================================================
# leetnote-ai（Python AI 服务）
# =============================================================
#
# 说明几点刻意的选择：
#
# 1. 用 python:3.11-slim 而不是 alpine
#    numpy / psycopg 在 alpine（musl libc）上要么没有预编译 wheel、
#    要么行为有差异。slim（glibc）能直接用官方 wheel，装得快也稳。
#
# 2. 先拷 requirements.txt 再拷代码
#    理由同 Go 那边：依赖没变就不重装，构建从几分钟降到几秒。
#
# 3. 不用 uvicorn 直接启动，而是走 run.py
#    这不是风格问题 —— uvicorn 把 ProactorEventLoop 硬编码进了 loop factory，
#    与 psycopg3 的异步不兼容。run.py 里做了事件循环的适配，必须走它。
#    （详见 leetnote-ai/run.py 顶部的注释）
# =============================================================

FROM python:3.11-slim

# - PYTHONDONTWRITEBYTECODE：容器里不需要 .pyc，少一堆文件
# - PYTHONUNBUFFERED：让日志实时输出，否则 docker logs 会攒着不出来
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /app

# 依赖单独一层：代码改动不会触发重新安装
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# 非 root 运行
RUN useradd --create-home --uid 10001 appuser \
    && chown -R appuser:appuser /app
USER appuser

EXPOSE 8000

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD python -c "import urllib.request,sys; sys.exit(0 if urllib.request.urlopen('http://127.0.0.1:8000/health', timeout=2).status==200 else 1)" || exit 1

# 必须用 run.py（见上面第 3 点）
CMD ["python", "run.py"]
