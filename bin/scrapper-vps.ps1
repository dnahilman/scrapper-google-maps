# PowerShell helper untuk scrapper container di VPS via SSH.
#
# Setup (sekali per session):
#   $env:VPS_HOST = "dios@182.23.12.142"
#
# Persistent (taruh di $PROFILE):
#   notepad $PROFILE
#   Tambah: $env:VPS_HOST = "dios@182.23.12.142"
#
# Actions:
#   status       - cek progress + sync summary
#   scrape       - START scrape di background (detached, SSH close OK)
#   stop         - kill background scrape process
#   is-running   - cek apakah scrape sedang berjalan
#   logs         - tail latest python log file (tail -f, Ctrl+C exit)
#   sync         - manual sync (foreground, butuh wait)
#   shell        - bash interactive di container
#   ps           - container status
#   pull         - download data/output dari VPS ke laptop
#   redeploy     - force pull image baru + restart

param(
    [Parameter(Position = 0)]
    [string]$Command = "help",

    [Parameter(Position = 1, ValueFromRemainingArguments = $true)]
    [string[]]$ExtraArgs
)

if (-not $env:VPS_HOST) {
    Write-Error 'Set $env:VPS_HOST = "user@host" dulu (mis. dios@182.23.12.142)'
    exit 1
}

$VpsHost = $env:VPS_HOST
$ProjDir = if ($env:VPS_PROJ_DIR) { $env:VPS_PROJ_DIR } else { "/opt/scrapper-google-maps" }
$Container = "scrapper-prod"
$Compose = "docker compose -f $ProjDir/docker-compose.prod.yml"
$ArgStr = if ($ExtraArgs) { $ExtraArgs -join " " } else { "" }

switch ($Command) {

    "status" {
        $pyCmd = "from src.storage import progress_summary, sync_summary; print('scrape:', progress_summary()); print('sync:', sync_summary())"
        ssh $VpsHost "docker exec -i $Container python -c `"$pyCmd`""
    }

    "scrape" {
        # Detached: process tetap jalan walau SSH close / PowerShell exit
        ssh $VpsHost "docker exec -d $Container python scripts/scraper.py --resume --auto-sync $ArgStr"
        Write-Host ""
        Write-Host "[OK] Scraper started in BACKGROUND (detached)" -ForegroundColor Green
        Write-Host "Anda boleh close PowerShell / shutdown laptop. Scrape jalan di VPS." -ForegroundColor Cyan
        Write-Host ""
        Write-Host "Monitor:" -ForegroundColor Yellow
        Write-Host "  .\bin\scrapper-vps.ps1 logs        # tail log realtime"
        Write-Host "  .\bin\scrapper-vps.ps1 status      # progress count"
        Write-Host "  .\bin\scrapper-vps.ps1 is-running  # cek alive"
        Write-Host "  .\bin\scrapper-vps.ps1 stop        # kill graceful"
    }

    "stop" {
        ssh $VpsHost "docker exec $Container pkill -f 'scripts/scraper.py' 2>/dev/null && echo '[OK] Scraper stopped' || echo '[--] Tidak ada scraper berjalan'"
    }

    "is-running" {
        $running = ssh $VpsHost "docker exec $Container pgrep -af 'scripts/scraper.py' 2>/dev/null"
        if ($running) {
            Write-Host "[RUNNING] $running" -ForegroundColor Green
        } else {
            Write-Host "[STOPPED] No scrape process running" -ForegroundColor Yellow
        }
    }

    "logs" {
        # Tail file log Python terbaru di container (lebih informatif dari docker logs)
        $inner = 'F=$(ls -t /app/logs/scraper-*.log 2>/dev/null | head -1); [ -n "$F" ] && tail -f -n 100 "$F" || echo "No log files yet — jalankan scrape dulu"'
        ssh -t $VpsHost "docker exec -t $Container bash -c '$inner'"
    }

    "sync" {
        # Foreground (sync biasanya cepat, ok block)
        ssh -t $VpsHost "docker exec -it $Container python scripts/sync.py $ArgStr"
    }

    "shell" {
        ssh -t $VpsHost "docker exec -it $Container bash"
    }

    "pull" {
        New-Item -ItemType Directory -Force -Path "./data/output-from-vps" | Out-Null
        scp -r "${VpsHost}:${ProjDir}/data/output/*" "./data/output-from-vps/"
    }

    "ps" {
        ssh $VpsHost "$Compose ps"
    }

    "redeploy" {
        ssh $VpsHost "$Compose pull; $Compose up -d; $Compose ps"
    }

    Default {
        Write-Host "Usage: .\bin\scrapper-vps.ps1 {action} [args...]" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "Actions:"
        Write-Host "  scrape       Start scrape di BACKGROUND (default, detached)"
        Write-Host "  stop         Kill background scrape"
        Write-Host "  is-running   Cek apakah scrape jalan"
        Write-Host "  logs         Tail log file realtime (Ctrl+C exit)"
        Write-Host "  status       Progress + sync summary"
        Write-Host "  sync         Manual sync (foreground)"
        Write-Host "  shell        Bash di container"
        Write-Host "  ps           Container status"
        Write-Host "  pull         Download data ke laptop"
        Write-Host "  redeploy     Force pull image baru"
        Write-Host ""
        Write-Host "Examples:"
        Write-Host "  .\bin\scrapper-vps.ps1 scrape"
        Write-Host "  .\bin\scrapper-vps.ps1 scrape --kelurahan `"Cihapit`""
        Write-Host "  .\bin\scrapper-vps.ps1 sync --all --force"
        if ($Command -ne "help") { exit 1 }
    }
}
