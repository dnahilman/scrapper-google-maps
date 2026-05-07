import json
from datetime import datetime, timezone

from fastapi import APIRouter, HTTPException
from fastapi.responses import FileResponse

from web.models import FileInfo
from web.services.job_manager import DATA_DIR

router = APIRouter()


def _safe_keyword_dir(keyword: str):
    """Validate + return path data/<keyword>/. Reject path traversal."""
    if not keyword or "/" in keyword or "\\" in keyword or keyword.startswith("."):
        raise HTTPException(status_code=400, detail="invalid keyword")
    d = DATA_DIR / keyword
    if not d.is_dir():
        raise HTTPException(status_code=404, detail="keyword not found")
    return d


def _safe_filename(filename: str) -> str:
    if (
        not filename
        or "/" in filename
        or "\\" in filename
        or filename.startswith(".")
        or not filename.endswith(".json")
    ):
        raise HTTPException(status_code=400, detail="invalid filename")
    return filename


@router.get("/keywords")
async def list_keywords() -> list[str]:
    if not DATA_DIR.exists():
        return []
    return sorted(
        p.name for p in DATA_DIR.iterdir() if p.is_dir() and not p.name.startswith(".")
    )


@router.get("/keywords/{keyword}/files", response_model=list[FileInfo])
async def list_files(keyword: str) -> list[FileInfo]:
    d = _safe_keyword_dir(keyword)
    out: list[FileInfo] = []
    for p in d.glob("*.json"):
        try:
            stat = p.stat()
            shop_count: int | None = None
            # cheap shop_count: hanya kalau file < 5MB (hindari cost untuk file besar)
            if stat.st_size < 5 * 1024 * 1024:
                try:
                    data = json.loads(p.read_text(encoding="utf-8"))
                    if isinstance(data, list):
                        shop_count = len(data)
                except (OSError, json.JSONDecodeError):
                    pass
            out.append(
                FileInfo(
                    name=p.name,
                    size_bytes=stat.st_size,
                    modified=datetime.fromtimestamp(stat.st_mtime, tz=timezone.utc),
                    shop_count=shop_count,
                )
            )
        except OSError:
            continue
    out.sort(key=lambda f: f.modified, reverse=True)
    return out


@router.get("/keywords/{keyword}/files/{filename}")
async def read_file(keyword: str, filename: str) -> dict | list:
    d = _safe_keyword_dir(keyword)
    fname = _safe_filename(filename)
    path = d / fname
    if not path.is_file():
        raise HTTPException(status_code=404, detail="file not found")
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as e:
        raise HTTPException(status_code=500, detail=f"json parse error: {e}")


@router.get("/keywords/{keyword}/files/{filename}/download")
async def download_file(keyword: str, filename: str):
    d = _safe_keyword_dir(keyword)
    fname = _safe_filename(filename)
    path = d / fname
    if not path.is_file():
        raise HTTPException(status_code=404, detail="file not found")
    return FileResponse(
        path,
        media_type="application/json",
        filename=f"{keyword}-{fname}",
    )
