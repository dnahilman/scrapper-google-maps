from datetime import datetime
from typing import Literal, Optional
from pydantic import BaseModel, Field


class JobCreateRequest(BaseModel):
    keyword: str = Field(..., min_length=1, description="Target keyword (cafe, barbershop, ...)")
    shard: Optional[str] = Field(None, pattern=r"^\d+/\d+$", description="K/N format, e.g. 1/2")
    kelurahan: Optional[str] = Field(None, description="Substring filter (case-insensitive)")
    limit: Optional[int] = Field(None, ge=1, description="Max places per kelurahan")
    resume: bool = Field(True, description="Skip kelurahan yang sudah status=done")
    dry_run: bool = Field(False, description="Hanya list kelurahan, tidak scrape")


class JobInfo(BaseModel):
    job_id: str
    pid: int
    keyword: str
    shard: Optional[str]
    kelurahan: Optional[str]
    limit: Optional[int]
    resume: bool
    dry_run: bool
    status: Literal["running", "exited", "unknown"]
    started_at: datetime
    cmd: list[str]
    log_file: str


class HealthResponse(BaseModel):
    ok: bool
    version: str
    uptime_sec: float
    active_jobs: int
    data_dir_size_mb: float
    keywords: list[str]


class KelurahanProgress(BaseModel):
    kelurahan: str
    kecamatan: Optional[str]
    status: Literal["in_progress", "done", "failed"]
    shop_count: int = 0
    error: Optional[str] = None
    started_at: Optional[str] = None
    finished_at: Optional[str] = None


class ProgressSummary(BaseModel):
    keyword: str
    counts: dict[str, int]
    total: int
    items: list[KelurahanProgress]


class FileInfo(BaseModel):
    name: str
    size_bytes: int
    modified: datetime
    shop_count: Optional[int] = None
