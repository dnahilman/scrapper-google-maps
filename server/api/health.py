import time
from pathlib import Path

from fastapi import APIRouter

from server import __version__
from server.models import HealthResponse
from server.services.job_manager import DATA_DIR, count_active_jobs

router = APIRouter()

_STARTUP_TIME = time.time()


def _data_dir_size_mb() -> float:
    if not DATA_DIR.exists():
        return 0.0
    total = 0
    for p in DATA_DIR.rglob("*"):
        if p.is_file():
            try:
                total += p.stat().st_size
            except OSError:
                continue
    return round(total / (1024 * 1024), 2)


def _list_keywords() -> list[str]:
    if not DATA_DIR.exists():
        return []
    out: list[str] = []
    for p in DATA_DIR.iterdir():
        if p.is_dir() and not p.name.startswith("."):
            out.append(p.name)
    return sorted(out)


@router.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    return HealthResponse(
        ok=True,
        version=__version__,
        uptime_sec=round(time.time() - _STARTUP_TIME, 1),
        active_jobs=count_active_jobs(),
        data_dir_size_mb=_data_dir_size_mb(),
        keywords=_list_keywords(),
    )
