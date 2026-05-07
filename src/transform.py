"""Transform shop dict (hasil scraping mentah) → SyncItem schema (sesuai API target).

SyncItem schema:
{
  name, address, phone,
  location: [lng, lat] | null,
  openingHours: { day: {open, close} | null } | null,
  description, features: string[],
  coverImage,
  rating: string,        // "4.9" sebagai string
  reviewCount: number,
  website, urlGoogleMaps, googlePlaceId,
  status, claimed,
  reviews: Review[]
}

Review schema:
{
  rating: 1|2|3|4|5|null,
  comment: string|null,
  photoUrl: string|null,
  guest: {name, image: null} | null
}
"""
import re


def _clean(s):
    """Strip whitespace + Material Symbols icon chars (Private Use Area Unicode).
    Sama logic dengan gmaps._clean. Diduplikasi disini untuk menghindari circular import."""
    if not s or not isinstance(s, str):
        return s
    no_icons = "".join(c for c in s if not ("" <= c <= ""))
    cleaned = " ".join(no_icons.split())
    return cleaned or None


def _parse_hours_value(time_str: str | None) -> dict | None:
    """Parse '10.00–21.00' → {open: '10.00', close: '21.00'}. 'Tutup' → None."""
    if not time_str:
        return None
    s = time_str.strip().lower()
    if s in ("", "tutup", "closed") or "tutup" in s or "closed" in s:
        return None
    if "24 jam" in s or "24 hours" in s or "open 24" in s:
        return {"open": "00.00", "close": "23.59"}
    # Match "HH.MM–HH.MM" / "HH:MM-HH:MM" — terima berbagai dash unicode
    m = re.match(r"(\d{1,2}[.:]\d{2})\s*[\-–—−]\s*(\d{1,2}[.:]\d{2})", s)
    if m:
        return {"open": m.group(1), "close": m.group(2)}
    return None


def _transform_hours(hours: dict | None) -> dict | None:
    if not hours:
        return None
    out: dict = {}
    for day, time_str in hours.items():
        out[day] = _parse_hours_value(time_str)
    return out or None


def _flatten_features(about: dict | None) -> list[str]:
    """Flatten semua items dari about sections jadi list features unik."""
    if not about:
        return []
    out: list[str] = []
    for items in about.values():
        for it in items:
            if it and it not in out:
                out.append(it)
    return out


def _to_review(r: dict) -> dict:
    rating = r.get("rating")
    if rating is not None:
        try:
            rating_int = int(round(float(rating)))
            if rating_int < 1 or rating_int > 5:
                rating_int = None
        except (TypeError, ValueError):
            rating_int = None
    else:
        rating_int = None

    photos = r.get("review_photos") or []
    photo_url = photos[0] if photos else None

    author = _clean(r.get("author"))
    guest = {"name": author, "image": None} if author else None

    return {
        "rating": rating_int,
        "comment": _clean(r.get("text")),
        "photoUrl": photo_url,
        "guest": guest,
    }


def to_sync_item(shop: dict, kecamatan: str | None = None) -> dict:
    """Konversi 1 shop dict → SyncItem dict (default = barbershop schema, backward compat).

    `kecamatan` kwarg di-accept untuk uniform dispatcher signature, tapi tidak dipakai
    di base schema. Cafe transformer (`to_sync_item_cafe`) yang pakai.
    """
    photos = shop.get("photos") or []
    cover_image = photos[0] if photos else None

    lat = shop.get("lat")
    lng = shop.get("lng")
    location = [lng, lat] if (lng is not None and lat is not None) else None

    rating = shop.get("rating")
    rating_str = str(rating) if rating is not None else None

    return {
        "name": _clean(shop.get("name")),
        "address": _clean(shop.get("address")),
        "phone": _clean(shop.get("phone")),
        "location": location,
        "openingHours": _transform_hours(shop.get("hours")),
        "description": None,
        "features": [_clean(f) for f in _flatten_features(shop.get("about")) if _clean(f)],
        "coverImage": cover_image,
        "rating": rating_str,
        "reviewCount": shop.get("review_count") or 0,
        "website": shop.get("website"),
        "urlGoogleMaps": shop.get("url"),
        "googlePlaceId": shop.get("place_id"),
        "status": shop.get("status") or "active",
        "claimed": False,
        "reviews": [_to_review(r) for r in (shop.get("reviews") or [])],
    }


def is_already_sync_schema(payload) -> bool:
    """Detect apakah file sudah dalam SyncItem schema (array) atau masih raw (object)."""
    return isinstance(payload, list) and (
        len(payload) == 0
        or (isinstance(payload[0], dict) and "googlePlaceId" in payload[0])
    )


def transform_payload(payload) -> list[dict]:
    """Transform payload dari schema lama ({kelurahan, barbershops}) ke list[SyncItem].
    Kalau sudah dalam schema baru, return as-is."""
    if is_already_sync_schema(payload):
        return payload
    if isinstance(payload, dict) and "barbershops" in payload:
        return [to_sync_item(s) for s in (payload["barbershops"] or [])]
    raise ValueError(f"Unknown payload shape: {type(payload).__name__}")


# ============================================================================
# Cafe / restaurant extended schema
# ============================================================================
# Backward-compatible: barbershop run pakai to_sync_item (default), cafe/resto run
# pakai to_sync_item_cafe yang nambah field menu/payment/wifi/parking/etc.
#
# Dispatcher get_transformer(keyword) dipakai di src/storage.py — auto-pilih transformer
# berdasarkan keyword aktif.

_WIFI_KEYWORDS = ("wi-fi", "wifi", "wi fi")
_PARKING_KEYWORDS = ("parkir", "parking")

_PAYMENT_RULES = {
    # output_key: tuple of substring matchers (case-insensitive)
    "cash":       ("tunai", "cash"),
    "debitCard":  ("kartu debit", "debit card"),
    "creditCard": ("kartu kredit", "credit card"),
    "qris":       ("qris",),
    "nfc":        ("nfc",),
    "ewallet":    ("gopay", "ovo", "dana", "shopeepay", "linkaja", "e-wallet", "dompet digital"),
}


def _scan_about(about: dict | None, keywords: tuple[str, ...]) -> bool:
    """Return True kalau ada item di about (semua section) yang mengandung salah satu keyword."""
    if not about:
        return False
    for items in about.values():
        for it in items:
            if not it:
                continue
            low = it.lower()
            if any(kw in low for kw in keywords):
                return True
    return False


def _has_wifi(about: dict | None) -> bool:
    return _scan_about(about, _WIFI_KEYWORDS)


def _has_parking(about: dict | None) -> bool:
    return _scan_about(about, _PARKING_KEYWORDS)


def _extract_payment(about: dict | None) -> dict:
    """Extract section 'Pembayaran'/'Payments' → structured boolean dict."""
    flags = {k: False for k in _PAYMENT_RULES}
    if not about:
        return flags
    payment_items: list[str] = []
    for sec_name, items in about.items():
        sec_lower = (sec_name or "").lower()
        if "pembayaran" in sec_lower or "payment" in sec_lower:
            payment_items.extend(items)
    if not payment_items:
        return flags
    blob = " ".join(payment_items).lower()
    for key, patterns in _PAYMENT_RULES.items():
        if any(p in blob for p in patterns):
            flags[key] = True
    return flags


def _to_menu(menu: dict | None) -> dict:
    """Normalize hasil scrape menu → {items: [{name, price}], photos: [url]}."""
    if not menu:
        return {"items": [], "photos": []}
    raw_items = menu.get("items") or []
    raw_photos = menu.get("photos") or []
    items: list[dict] = []
    seen: set[str] = set()
    for it in raw_items:
        name = _clean(it.get("name") if isinstance(it, dict) else None)
        price = _clean(it.get("price") if isinstance(it, dict) else None)
        if not name:
            continue
        key = f"{name}|{price or ''}"
        if key in seen:
            continue
        seen.add(key)
        items.append({"name": name, "price": price})
    return {"items": items, "photos": list(raw_photos)}


_EMPTY_DISTRIBUTION = {
    "oneStar": 0, "twoStar": 0, "threeStar": 0, "fourStar": 0, "fiveStar": 0,
}


def to_sync_item_cafe(shop: dict, kecamatan: str | None = None) -> dict:
    """Cafe/resto schema — superset dari barbershop dengan field tambahan.

    Field cafe-only:
    - category (mis. "Coffee shop", "Restoran")
    - rating: float (override base yang string)
    - totalReviews: int (rename dari reviewCount, lebih descriptive)
    - reviewsDistribution: {oneStar, twoStar, threeStar, fourStar, fiveStar} dari histogram Google Maps
    - wifiAvailable, hasParking: bool dari scan about
    - payment: structured {cash, debitCard, creditCard, qris, nfc, ewallet}
    - pricing: harga string (mis. "Rp 25.000-50.000") dari price_level
    - gallery: full photos array
    - menu: {items, photos} dari tab Menu
    - city: "Bandung" (hardcoded sesuai search scope)
    - district: kecamatan dari seed
    - reviews: pindah ke akhir (heavy array)

    Frontend yang konsumsi base field (name/address/coverImage/etc) tetap kompatibel
    karena field-field itu masih ada — yang berbeda hanya rating type (string→float)
    dan reviewCount→totalReviews. Sesuaikan frontend.
    """
    base = to_sync_item(shop)
    reviews_transformed = base["reviews"]
    about = shop.get("about") or {}

    return {
        "name": base["name"],
        "address": base["address"],
        "phone": base["phone"],
        "category": _clean(shop.get("category")),
        "location": base["location"],
        "openingHours": base["openingHours"],
        "description": base["description"],
        "features": base["features"],
        "coverImage": base["coverImage"],
        "rating": shop.get("rating"),  # float, bukan string
        "totalReviews": shop.get("review_count") or 0,
        "reviewsDistribution": shop.get("review_distribution") or dict(_EMPTY_DISTRIBUTION),
        "website": base["website"],
        "urlGoogleMaps": base["urlGoogleMaps"],
        "googlePlaceId": base["googlePlaceId"],
        "status": base["status"],
        "claimed": base["claimed"],
        "wifiAvailable": _has_wifi(about),
        "hasParking": _has_parking(about),
        "payment": _extract_payment(about),
        "pricing": _clean(shop.get("price_level")),
        "gallery": list(shop.get("photos") or []),
        "menu": _to_menu(shop.get("menu")),
        "city": "Bandung",
        "district": _clean(kecamatan) if kecamatan else None,
        "reviews": reviews_transformed,
    }


# Registry: keyword → transformer function. Default fallback = barbershop schema.
_TRANSFORMERS = {
    "cafe":       to_sync_item_cafe,
    "kafe":       to_sync_item_cafe,
    "resto":      to_sync_item_cafe,
    "restaurant": to_sync_item_cafe,
    "kuliner":    to_sync_item_cafe,
}


def get_transformer(keyword: str | None):
    """Pilih transformer berdasarkan keyword. Default = to_sync_item (barbershop)."""
    return _TRANSFORMERS.get((keyword or "").lower(), to_sync_item)
