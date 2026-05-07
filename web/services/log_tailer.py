"""Async file tailer untuk SSE log streaming.

Pattern: open file, seek ke end (atau N baris dari end untuk seed), lalu polling
size berkala. Kalau bertambah, baca chunk baru, yield per-line ke client.
"""
from __future__ import annotations

import asyncio
import os
from collections import deque
from pathlib import Path
from typing import AsyncIterator


def _read_last_n_lines(path: Path, n: int) -> list[str]:
    """Baca N baris terakhir tanpa load seluruh file (efisien untuk log besar)."""
    if not path.exists() or n <= 0:
        return []
    try:
        with path.open("rb") as f:
            f.seek(0, os.SEEK_END)
            size = f.tell()
            block = 4096
            data = b""
            pos = size
            while pos > 0 and data.count(b"\n") <= n:
                step = min(block, pos)
                pos -= step
                f.seek(pos)
                data = f.read(step) + data
            lines = data.decode("utf-8", errors="replace").splitlines()
            return lines[-n:]
    except OSError:
        return []


async def tail_file(
    path: Path,
    seed_lines: int = 100,
    poll_interval: float = 0.5,
) -> AsyncIterator[str]:
    """Yield line-by-line dari file, follow new appends. Untuk SSE event stream."""
    seeded = _read_last_n_lines(path, seed_lines)
    for line in seeded:
        yield line

    last_size = path.stat().st_size if path.exists() else 0
    buffer = ""

    while True:
        try:
            await asyncio.sleep(poll_interval)
            if not path.exists():
                # File belum ada (job baru spawn), tetap tunggu
                continue
            current_size = path.stat().st_size
            if current_size < last_size:
                # File di-truncate / rotate — reset
                last_size = 0
                buffer = ""
                continue
            if current_size > last_size:
                with path.open("rb") as f:
                    f.seek(last_size)
                    chunk = f.read(current_size - last_size)
                last_size = current_size
                buffer += chunk.decode("utf-8", errors="replace")
                while "\n" in buffer:
                    line, buffer = buffer.split("\n", 1)
                    yield line
        except asyncio.CancelledError:
            break
        except OSError:
            await asyncio.sleep(poll_interval)


def sse_format(line: str) -> str:
    """Format 1 baris jadi SSE event. Escape newline (SSE pakai \\n sebagai delimiter)."""
    safe = line.replace("\r", "")
    return f"data: {safe}\n\n"
