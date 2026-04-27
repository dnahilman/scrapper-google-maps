import os
from pathlib import Path
from dotenv import load_dotenv

# ============================================================================
# Paths
# ============================================================================
ROOT = Path(__file__).parent

# Load env dari .env.local (single source of truth)
load_dotenv(ROOT / ".env.local")
DATA_DIR = ROOT / "data"
OUTPUT_DIR = DATA_DIR / "output"
LOG_DIR = ROOT / "logs"
KELURAHAN_FILE = DATA_DIR / "kelurahan_bandung.json"
PROGRESS_DB = ROOT / "progress.db"

OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
LOG_DIR.mkdir(parents=True, exist_ok=True)

# ============================================================================
# Browser & delay
# ============================================================================
HEADLESS = os.getenv("HEADLESS", "true").lower() == "true"
MIN_DELAY_SEC = float(os.getenv("MIN_DELAY_SEC", "5"))
MAX_DELAY_SEC = float(os.getenv("MAX_DELAY_SEC", "12"))

# ============================================================================
# Reviews
# ============================================================================
MAX_REVIEWS_PER_SHOP = int(os.getenv("MAX_REVIEWS_PER_SHOP", "200"))
MAX_REVIEW_AGE_DAYS = int(os.getenv("MAX_REVIEW_AGE_DAYS", "730"))
SKIP_EMPTY_REVIEWS = os.getenv("SKIP_EMPTY_REVIEWS", "true").lower() == "true"
SORT_REVIEWS_BY_NEWEST = os.getenv("SORT_REVIEWS_BY_NEWEST", "true").lower() == "true"

# ============================================================================
# Safety
# ============================================================================
MAX_CAPTCHA_RETRY = int(os.getenv("MAX_CAPTCHA_RETRY", "2"))
MAX_NETWORK_ERRORS = 5
LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO")

# ============================================================================
# Sync API (POST hasil scraping ke remote backend)
# ============================================================================
APP_URL = os.getenv("APP_URL", "https://api.hilman.imola.ai")
GOOGLE_MAPS_SYNC_API_KEY = os.getenv("GOOGLE_MAPS_SYNC_API_KEY", "")
SYNC_ENDPOINT = "/v1/web/sync-google-maps"

# ============================================================================
# Search query template & target
# ============================================================================
SEARCH_QUERY_TEMPLATE = "barbershop di {kelurahan}, {kecamatan}, Bandung"
GMAPS_BASE_URL = "https://www.google.com/maps"

# ============================================================================
# Browser fingerprint randomization
# ============================================================================
VIEWPORTS = [
    {"width": 1366, "height": 768},
    {"width": 1440, "height": 900},
    {"width": 1536, "height": 864},
    {"width": 1920, "height": 1080},
]

LOCALES = ["id-ID", "en-US"]
TIMEZONE = "Asia/Jakarta"
