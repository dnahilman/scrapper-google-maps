import asyncio

from fastapi import APIRouter, HTTPException
from fastapi.responses import StreamingResponse

from web.models import JobCreateRequest, JobInfo
from web.services import job_manager
from web.services.job_manager import LOG_DIR
from web.services.log_tailer import sse_format, tail_file

router = APIRouter()


@router.post("/jobs", response_model=JobInfo, status_code=201)
async def create_job(req: JobCreateRequest) -> JobInfo:
    try:
        data = job_manager.start_job(
            keyword=req.keyword,
            shard=req.shard,
            kelurahan=req.kelurahan,
            limit=req.limit,
            resume=req.resume,
            dry_run=req.dry_run,
        )
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    data["status"] = "running"
    return JobInfo(**data)


@router.get("/jobs", response_model=list[JobInfo])
async def list_jobs(keyword: str | None = None) -> list[JobInfo]:
    jobs = job_manager.list_jobs(keyword)
    return [JobInfo(**j) for j in jobs]


@router.get("/jobs/{job_id}", response_model=JobInfo)
async def get_job(job_id: str) -> JobInfo:
    job = job_manager.get_job(job_id)
    if not job:
        raise HTTPException(status_code=404, detail="job not found")
    return JobInfo(**job)


@router.delete("/jobs/{job_id}")
async def stop_job(job_id: str) -> dict:
    job = job_manager.get_job(job_id)
    if not job:
        raise HTTPException(status_code=404, detail="job not found")
    ok = job_manager.stop_job(job_id)
    return {"ok": ok, "job_id": job_id}


@router.get("/jobs/{job_id}/logs/stream")
async def stream_job_logs(job_id: str, seed: int = 100):
    job = job_manager.get_job(job_id)
    if not job:
        raise HTTPException(status_code=404, detail="job not found")
    log_path = LOG_DIR / job["log_file"]

    async def event_gen():
        try:
            async for line in tail_file(log_path, seed_lines=seed):
                yield sse_format(line)
        except asyncio.CancelledError:
            return

    return StreamingResponse(
        event_gen(),
        media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )
