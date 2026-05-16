package scraper

// Centralized CSS / XPath / locator strings for Google Maps DOM.
// If Google changes their layout, fix it here in one place.
//
// Mirrored from server/src/gmaps.py so behaviour stays comparable.
const (
	// Search results feed + place cards.
	SelFeedOrCard = `div[role="feed"], a[href*="/maps/place/"]`
	SelFeed       = `div[role="feed"]`
	SelPlaceCard  = `a[href*="/maps/place/"]`

	// Place detail (h1 = title).
	SelTitle = `h1`

	// Rating block — the .F7nice container holds both rating and review count.
	SelRatingText        = `div.F7nice span[aria-hidden="true"]`
	SelReviewCountIDAria = `div.F7nice span[aria-label*="ulasan" i]`
	SelReviewCountEnAria = `div.F7nice span[aria-label*="review" i]`
	SelReviewCountAria   = `div.F7nice span[aria-label]`

	// Action buttons on the right-rail panel.
	SelAddressBtn  = `button[data-item-id="address"]`
	SelPhoneBtn    = `button[data-item-id^="phone"]`
	SelWebsiteLink = `a[data-item-id="authority"]`
	SelPlusCodeBtn = `button[data-item-id="oloc"]`
	SelCategoryBtn = `button[jsaction*="category"]`

	// Status / closed-state text is parsed from the main panel via evaluate().
	SelMainPanel = `div[role="main"]`

	// Reviews — list root + individual cards.
	SelReviewsTab     = `button[aria-label*="ulasan" i], button[aria-label*="review" i]`
	SelReviewsList    = `div[role="main"] div[data-review-id]`
	SelReviewSortBtn  = `button[aria-label*="urutkan" i], button[aria-label*="sort" i]`
	SelReviewSortMenu = `div[role="menuitem"]`
)
