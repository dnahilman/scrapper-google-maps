import asyncio

from fastapi import APIRouter, HTTPException
from fastapi.responses import StreamingResponse

from server.services.job_manager import LOG_DIR
from server.services.log_tailer import sse_format, tail_file

router = APIRouter()


@router.get("/logs/files")
async def list_log_files() -> list[dict]:
    """List logfiles di /app/logs (daily scraper logs + per-job logs)."""
    if not LOG_DIR.exists():
        return []
    out = []
    for p in sorted(LOG_DIR.glob("*.log"), key=lambda f: f.stat().st_mtime, reverse=True):
        try:
            stat = p.stat()
            out.append(
                {
                    "name": p.name,
                    "size_bytes": stat.st_size,
                    "modified": stat.st_mtime,
                    "kind": "job" if p.name.startswith("job-") else "scraper",
                }
            )
        except OSError:
            continue
    return out


@router.get("/logs/{filename}/stream")
async def stream_log_file(filename: str, seed: int = 200):
    if "/" in filename or "\\" in filename or filename.startswith("."):
        raise HTTPException(status_code=400, detail="invalid filename")
    path = LOG_DIR / filename
    if not path.exists() or not path.is_file():
        raise HTTPException(status_code=404, detail="log file not found")

    async def event_gen():
        try:
            async for line in tail_file(path, seed_lines=seed):
                yield sse_format(line)
        except asyncio.CancelledError:
            return

    return StreamingResponse(
        event_gen(),
        media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )
