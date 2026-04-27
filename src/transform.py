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


def to_sync_item(shop: dict) -> dict:
    """Konversi 1 shop dict → SyncItem dict. String fields di-clean dari icon chars."""
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
