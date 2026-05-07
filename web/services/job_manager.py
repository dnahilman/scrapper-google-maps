"""Subprocess-based job manager untuk scrape jobs.

State source of truth: pidfile JSON di `data/<keyword>/.jobs/<job_id>.json`.
Pakai psutil (cross-platform) untuk PID liveness check + signal.

Integrasi dengan scraper existing: spawn `python scripts/scraper.py ...` sebagai
child process dari uvicorn worker. Stdout/stderr di-redirect ke `logs/job-<id>.log`.
Scraper tetap menulis daily log `logs/scraper-YYYYMMDD.log` lewat logger internalnya.
"""
from __future__ import annotations

import json
import os
import signal
import subprocess
import sys
import time
import uuid
from datetime import datetime, timezone, timedelta
from pathlib import Path
from typing import Optional

import psutil

WIB = timezone(timedelta(hours=7))
ROOT = Path(__file__).resolve().parents[2]
DATA_DIR = ROOT / "data"
LOG_DIR = ROOT / "logs"
SCRAPER_SCRIPT = ROOT / "scripts" / "scraper.py"


def _jobs_dir(keyword: str) -> Path:
    d = DATA_DIR / keyword / ".jobs"
    d.mkdir(parents=True, exist_ok=True)
    return d


def _all_pidfiles() -> list[Path]:
    if not DATA_DIR.exists():
        return []
    out: list[Path] = []
    for kw_dir in DATA_DIR.iterdir():
        jobs = kw_dir / ".jobs"
        if jobs.is_dir():
            out.extend(jobs.glob("*.json"))
    return out


def _pid_alive(pid: int, expected_substring: str = "scraper.py") -> bool:
    """Check pid alive AND cmdline berisi scraper script (anti PID reuse)."""
    if not psutil.pid_exists(pid):
        return False
    try:
        proc = psutil.Process(pid)
        cmdline = " ".join(proc.cmdline())
        return expected_substring in cmdline
    except (psutil.NoSuchProcess, psutil.AccessDenied):
        return False


def start_job(
    keyword: str,
    shard: Optional[str] = None,
    kelurahan: Optional[str] = None,
    limit: Optional[int] = None,
    resume: bool = True,
    dry_run: bool = False,
) -> dict:
    """Spawn scrape subprocess. Return job metadata dict (sama dengan pidfile content)."""
    keyword = keyword.strip().lower()
    if not keyword:
        raise ValueError("keyword tidak boleh kosong")

    job_id = uuid.uuid4().hex[:8]
    LOG_DIR.mkdir(parents=True, exist_ok=True)
    log_file = LOG_DIR / f"job-{job_id}.log"

    cmd: list[str] = [sys.executable, str(SCRAPER_SCRIPT), "--keyword", keyword]
    if shard:
        cmd += ["--shard", shard]
    if kelurahan:
        cmd += ["--kelurahan", kelurahan]
    if limit:
        cmd += ["--limit", str(limit)]
    if resume:
        cmd += ["--resume"]
    if dry_run:
        cmd += ["--dry-run"]

    started_at = datetime.now(WIB)
    log_fh = open(log_file, "w", encoding="utf-8", buffering=1)
    log_fh.write(f"# job {job_id} started at {started_at.isoformat()}\n")
    log_fh.write(f"# cmd: {' '.join(cmd)}\n\n")
    log_fh.flush()

    popen_kwargs = dict(
        stdout=log_fh,
        stderr=subprocess.STDOUT,
        cwd=str(ROOT),
        env={**os.environ, "PYTHONUNBUFFERED": "1"},
    )
    # POSIX only — start_new_session jadikan subprocess pemimpin process group
    # supaya killpg bisa terminate seluruh tree (Playwright spawn Chromium child)
    if os.name == "posix":
        popen_kwargs["start_new_session"] = True

    proc = subprocess.Popen(cmd, **popen_kwargs)

    pidfile_data = {
        "job_id": job_id,
        "pid": proc.pid,
        "keyword": keyword,
        "shard": shard,
        "kelurahan": kelurahan,
        "limit": limit,
        "resume": resume,
        "dry_run": dry_run,
        "started_at": started_at.isoformat(),
        "cmd": cmd,
        "log_file": log_file.name,
    }

    pidfile = _jobs_dir(keyword) / f"{job_id}.json"
    pidfile.write_text(json.dumps(pidfile_data, indent=2), encoding="utf-8")

    return pidfile_data


def list_jobs(keyword: Optional[str] = None) -> list[dict]:
    """Return semua job (running + recently exited). Cleanup pidfile yang stale > 1 jam."""
    pidfiles = (
        list(_jobs_dir(keyword).glob("*.json")) if keyword else _all_pidfiles()
    )
    now = datetime.now(WIB)
    jobs: list[dict] = []
    for pf in pidfiles:
        try:
            data = json.loads(pf.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            continue
        alive = _pid_alive(data.get("pid", -1))
        data["status"] = "running" if alive else "exited"
        # cleanup pidfile yang sudah exit > 1 jam
        if not alive:
            try:
                started = datetime.fromisoformat(data["started_at"])
                if (now - started).total_seconds() > 3600:
                    pf.unlink(missing_ok=True)
                    continue
            except (KeyError, ValueError):
                pass
        jobs.append(data)
    jobs.sort(key=lambda j: j.get("started_at", ""), reverse=True)
    return jobs


def get_job(job_id: str) -> Optional[dict]:
    for pf in _all_pidfiles():
        if pf.stem == job_id:
            try:
                data = json.loads(pf.read_text(encoding="utf-8"))
                data["status"] = "running" if _pid_alive(data.get("pid", -1)) else "exited"
                return data
            except (OSError, json.JSONDecodeError):
                return None
    return None


def stop_job(job_id: str, timeout_sec: float = 5.0) -> bool:
    """SIGTERM seluruh process group, wait timeout, lalu SIGKILL kalau masih hidup.
    Return True kalau process berhasil dihentikan (atau memang sudah mati).
    """
    job = get_job(job_id)
    if not job:
        return False
    pid = job.get("pid", -1)
    pidfile = _jobs_dir(job["keyword"]) / f"{job_id}.json"

    if not _pid_alive(pid):
        pidfile.unlink(missing_ok=True)
        return True

    try:
        proc = psutil.Process(pid)
        children = proc.children(recursive=True)

        # POSIX: signal seluruh process group sekaligus
        if os.name == "posix":
            try:
                os.killpg(os.getpgid(pid), signal.SIGTERM)
            except (ProcessLookupError, PermissionError):
                proc.terminate()
        else:
            for c in children:
                try:
                    c.terminate()
                except psutil.NoSuchProcess:
                    pass
            proc.terminate()

        gone, alive = psutil.wait_procs([proc, *children], timeout=timeout_sec)
        for p in alive:
            try:
                p.kill()
            except psutil.NoSuchProcess:
                pass
        psutil.wait_procs(alive, timeout=2.0)
    except psutil.NoSuchProcess:
        pass
    finally:
        # Tunggu sebentar supaya pid benar-benar gone sebelum hapus pidfile
        time.sleep(0.2)
        pidfile.unlink(missing_ok=True)

    return True


def count_active_jobs(keyword: Optional[str] = None) -> int:
    return sum(1 for j in list_jobs(keyword) if j["status"] == "running")
