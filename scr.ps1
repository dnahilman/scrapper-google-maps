# PowerShell helper untuk scrapper container — dual-mode (LOCAL default, -Vps untuk remote).
#
# LOCAL MODE (default):
#   Pre-req: docker compose -f docker-compose.dev.yml up -d --build  (sekali)
#   .\scr.ps1 scrape cafe --kelurahan "Cihapit"
#
# VPS MODE (-Vps switch):
#   Pre-req: $env:VPS_HOST = "user@host"  (taruh di $PROFILE supaya persistent)
#   .\scr.ps1 -Vps scrape cafe
#
# Catatan: di bash equivalent (bin/scr) flag-nya --vps. Di PowerShell native: -Vps.

param(
    [switch]$Vps,

    [Parameter(Position = 0)]
    [string]$Command = "help",

    [Parameter(Position = 1)]
    [string]$Keyword,

    [Parameter(Position = 2, ValueFromRemainingArguments = $true)]
    [string[]]$ExtraArgs
)

$DevService = "scraper"
$Script:DevComposeBin = $null
$Script:DevComposeSub = $null

# Wrapper supaya tiap call docker-compose dev cuma satu identifier (pakai @args splatting).
function Invoke-DevCompose {
    if ($Script:DevComposeSub) {
        & $Script:DevComposeBin $Script:DevComposeSub -f docker-compose.dev.yml @args
    } else {
        & $Script:DevComposeBin -f docker-compose.dev.yml @args
    }
}

# Auto-detect compose tool. Prefer 'docker compose', fallback ke 'podman-compose'.
# Override paksa: $env:COMPOSE_TOOL = 'docker' atau 'podman'.
function Detect-DevCompose {
    $override = $env:COMPOSE_TOOL
    if ($override -eq 'docker') {
        $Script:DevComposeBin = 'docker'
        $Script:DevComposeSub = 'compose'
        return
    }
    if ($override -eq 'podman') {
        $Script:DevComposeBin = 'podman-compose'
        $Script:DevComposeSub = $null
        return
    }
    if ($override) {
        Write-Error "COMPOSE_TOOL='$override' tidak valid. Pakai 'docker' atau 'podman'."
        exit 1
    }

    try {
        & docker compose version 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            $Script:DevComposeBin = 'docker'
            $Script:DevComposeSub = 'compose'
            return
        }
    } catch { }

    try {
        & podman-compose --version 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            $Script:DevComposeBin = 'podman-compose'
            $Script:DevComposeSub = $null
            return
        }
    } catch { }
}

if ($Vps) {
    if (-not $env:VPS_HOST) {
        Write-Error 'Set $env:VPS_HOST = "user@host" dulu (mis. dios@182.23.12.142)'
        exit 1
    }
    $Mode = "vps"
    $VpsHost = $env:VPS_HOST
    $ProjDir = if ($env:VPS_PROJ_DIR) { $env:VPS_PROJ_DIR } else { "/opt/scrapper-google-maps" }
    $ProdContainer = "scrapper-prod"
    $ProdCompose = "docker compose -f $ProjDir/docker-compose.prod.yml"
} else {
    $Mode = "local"
    Detect-DevCompose
    if (-not $Script:DevComposeBin) {
        Write-Error "'docker compose' atau 'podman-compose' tidak ditemukan di PATH. Install Docker Desktop atau 'pip install podman-compose'."
        exit 1
    }
}

$ExtraArgsArr = if ($ExtraArgs) { $ExtraArgs } else { @() }
$ArgStr = if ($ExtraArgs) { $ExtraArgs -join " " } else { "" }
$ModePrefix = if ($Mode -eq "vps") { "-Vps " } else { "" }

function Assert-Keyword {
    if (-not $Keyword) {
        Write-Error "action '$Command' butuh <keyword> sebagai argumen pertama. Contoh: .\scr.ps1 ${ModePrefix}$Command cafe"
        exit 1
    }
}

switch ($Command) {

    "status" {
        Assert-Keyword
        $pyCmd = "import config; config.set_keyword('$Keyword'); from src.storage import progress_summary, sync_summary; print('keyword:', config.get_keyword()); print('scrape:', progress_summary()); print('sync:', sync_summary())"
        if ($Mode -eq "vps") {
            ssh $VpsHost "docker exec -i $ProdContainer python -c `"$pyCmd`""
        } else {
            Invoke-DevCompose exec -T $DevService python -c $pyCmd
        }
    }

    "scrape" {
        Assert-Keyword
        # Auto-detect: kalau user pass --kelurahan X (re-scrape spesifik), skip --resume.
        # Full run (tanpa --kelurahan) → --resume tetap on supaya skip kelurahan yang done.
        $HasKelurahan = $ExtraArgsArr -contains '--kelurahan'
        $ResumeArgs = if ($HasKelurahan) { @() } else { @('--resume') }
        $ResumeNote = if ($HasKelurahan) {
            "re-scrape mode (--kelurahan filter, no --resume)"
        } else {
            "resume mode (skip kelurahan yang sudah done)"
        }

        if ($Mode -eq "vps") {
            $extraStr = ($ResumeArgs + $ExtraArgsArr) -join " "
            ssh $VpsHost "docker exec -d $ProdContainer python scripts/scraper.py --keyword $Keyword $extraStr"
        } else {
            Invoke-DevCompose exec -d $DevService python scripts/scraper.py --keyword $Keyword @ResumeArgs @ExtraArgsArr
        }
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Scraper gagal start (exit $LASTEXITCODE). Container mungkin belum running. Cek: .\scr.ps1 ps"
            exit $LASTEXITCODE
        }
        Write-Host ""
        if ($Mode -eq "vps") {
            Write-Host "[OK] Scraper started in BACKGROUND (detached) -- keyword=$Keyword (VPS)" -ForegroundColor Green
            Write-Host "    $ResumeNote" -ForegroundColor DarkGray
            Write-Host "Anda boleh close PowerShell / shutdown laptop. Scrape jalan di VPS." -ForegroundColor Cyan
        } else {
            Write-Host "[OK] Scraper started in BACKGROUND (detached) -- keyword=$Keyword (LOCAL)" -ForegroundColor Green
            Write-Host "    $ResumeNote" -ForegroundColor DarkGray
            Write-Host "Process jalan di container local. Container restart akan kill process." -ForegroundColor Cyan
        }
        Write-Host "    Sync: opt-in via --auto-sync (butuh GOOGLE_MAPS_SYNC_API_KEY di .env.local)" -ForegroundColor DarkGray
        Write-Host ""
        Write-Host "Monitor:" -ForegroundColor Yellow
        Write-Host "  .\scr.ps1 ${ModePrefix}logs                # tail log realtime"
        Write-Host "  .\scr.ps1 ${ModePrefix}status $Keyword     # progress count"
        Write-Host "  .\scr.ps1 ${ModePrefix}is-running          # cek alive"
        Write-Host "  .\scr.ps1 ${ModePrefix}stop                # kill graceful"
    }

    "stop" {
        $stopCmd = "pkill -f 'scripts/scraper.py' 2>/dev/null && echo '[OK] Scraper stopped' || echo '[--] Tidak ada scraper berjalan'"
        if ($Mode -eq "vps") {
            ssh $VpsHost "docker exec $ProdContainer $stopCmd"
        } else {
            Invoke-DevCompose exec -T $DevService bash -c $stopCmd
        }
    }

    "is-running" {
        if ($Mode -eq "vps") {
            $running = ssh $VpsHost "docker exec $ProdContainer pgrep -af 'scripts/scraper.py' 2>/dev/null"
        } else {
            $running = Invoke-DevCompose exec -T $DevService pgrep -af 'scripts/scraper.py' 2>$null
        }
        if ($running) {
            Write-Host "[RUNNING] $running" -ForegroundColor Green
        } else {
            Write-Host "[STOPPED] No scrape process running" -ForegroundColor Yellow
        }
    }

    "logs" {
        $inner = 'F=$(ls -t /app/logs/scraper-*.log 2>/dev/null | head -1); [ -n "$F" ] && tail -f -n 100 "$F" || echo "No log files yet -- jalankan scrape dulu"'
        if ($Mode -eq "vps") {
            ssh -t $VpsHost "docker exec -t $ProdContainer bash -c '$inner'"
        } else {
            Invoke-DevCompose exec $DevService bash -c $inner
        }
    }

    "sync" {
        Assert-Keyword
        if ($Mode -eq "vps") {
            ssh -t $VpsHost "docker exec -it $ProdContainer python scripts/sync.py --keyword $Keyword $ArgStr"
        } else {
            Invoke-DevCompose exec $DevService python scripts/sync.py --keyword $Keyword @ExtraArgsArr
        }
    }

    "shell" {
        if ($Mode -eq "vps") {
            ssh -t $VpsHost "docker exec -it $ProdContainer bash"
        } else {
            Invoke-DevCompose exec $DevService bash
        }
    }

    "pull" {
        Assert-Keyword
        if ($Mode -eq "vps") {
            $LocalDir = "./data/$Keyword-from-vps"
            New-Item -ItemType Directory -Force -Path $LocalDir | Out-Null
            scp -r "${VpsHost}:${ProjDir}/data/$Keyword/*" $LocalDir
        } else {
            Write-Host "[--] Local mode: data sudah di ./data/$Keyword/ (volume-mounted dari container)." -ForegroundColor Yellow
            Write-Host "    Tidak perlu pull."
        }
    }

    "ps" {
        if ($Mode -eq "vps") {
            ssh $VpsHost "$ProdCompose ps"
        } else {
            Invoke-DevCompose ps
        }
    }

    "redeploy" {
        if ($Mode -eq "vps") {
            ssh $VpsHost "$ProdCompose pull; $ProdCompose up -d; $ProdCompose ps"
        } else {
            Invoke-DevCompose build && Invoke-DevCompose up -d && Invoke-DevCompose ps
        }
    }

    Default {
        Write-Host "Usage: .\scr.ps1 [-Vps] {action} [keyword] [args...]" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "Mode:"
        Write-Host "  (default)    LOCAL  - auto-detect 'docker compose' / 'podman-compose' (override: `$env:COMPOSE_TOOL='docker'|'podman')"
        Write-Host "  -Vps         VPS    - ssh `$env:VPS_HOST + docker exec scrapper-prod"
        Write-Host ""
        Write-Host "Actions yang butuh <keyword>:"
        Write-Host "  scrape <keyword> [args]   Start scrape di BACKGROUND (detached)"
        Write-Host "  sync   <keyword> [args]   Manual sync (foreground)"
        Write-Host "  status <keyword>          Progress + sync summary"
        Write-Host "  pull   <keyword>          Download data dari VPS (vps mode only)"
        Write-Host ""
        Write-Host "Actions tanpa keyword:"
        Write-Host "  stop         Kill background scrape"
        Write-Host "  is-running   Cek apakah scrape jalan"
        Write-Host "  logs         Tail log file realtime (Ctrl+C exit)"
        Write-Host "  shell        Bash di container"
        Write-Host "  ps           Container status"
        Write-Host "  redeploy     LOCAL: rebuild image. VPS: pull image baru + restart"
        Write-Host ""
        Write-Host "Local pre-req (sekali):"
        Write-Host "  docker compose -f docker-compose.dev.yml up -d --build"
        Write-Host ""
        Write-Host "VPS pre-req:"
        Write-Host "  `$env:VPS_HOST = `"user@host`"   # mis. dios@182.23.12.142"
        Write-Host ""
        Write-Host "Examples:"
        Write-Host "  .\scr.ps1 scrape cafe                                  # scrape only, no sync"
        Write-Host "  .\scr.ps1 scrape cafe --auto-sync                      # + auto-sync (butuh API key)"
        Write-Host "  .\scr.ps1 scrape cafe --kelurahan `"Cihapit`"            # re-scrape spesifik"
        Write-Host "  .\scr.ps1 sync cafe --all --force                      # manual sync semua file"
        Write-Host "  .\scr.ps1 status cafe                                  # progress + sync summary"
        Write-Host "  .\scr.ps1 -Vps scrape cafe --auto-sync                 # remote full run"
        Write-Host "  .\scr.ps1 -Vps status cafe                             # remote status"
        if ($Command -ne "help") { exit 1 }
    }
}
