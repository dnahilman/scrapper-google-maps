package domain

// gosom-style payload types used inside JSONB columns and the worker→master
// task result envelope. Field names match the upstream `gmaps.Entry` schema
// so an external sync target can consume them directly.

type Image struct {
	Title string `json:"title"`
	Image string `json:"image"`
}

type LinkSource struct {
	Link   string `json:"link"`
	Source string `json:"source"`
}

// MenuItem is a single dish/product extracted from the in-page Menu tab.
type MenuItem struct {
	Name  string `json:"name"`
	Price string `json:"price,omitempty"`
}

// MenuPayload extends gosom's `LinkSource` shape with the items and photos
// pulled from the Google Maps in-place Menu tab. Pure-gosom consumers can
// still read the `link`/`source` keys; richer consumers see `items`/`photos`.
type MenuPayload struct {
	Link   string     `json:"link,omitempty"`
	Source string     `json:"source,omitempty"`
	Items  []MenuItem `json:"items,omitempty"`
	Photos []string   `json:"photos,omitempty"`
}

type Owner struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Link string `json:"link"`
}

type CompleteAddress struct {
	Borough    string `json:"borough,omitempty"`
	Street     string `json:"street,omitempty"`
	City       string `json:"city,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	State      string `json:"state,omitempty"`
	Country    string `json:"country,omitempty"`
}

type AboutOption struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type About struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Options []AboutOption `json:"options"`
}

type OwnerResponse struct {
	Text string `json:"text,omitempty"`
	Time string `json:"time,omitempty"`
}

// ReviewPayload mirrors gosom.Review (sent over the wire by worker).
type ReviewPayload struct {
	ReviewID       string         `json:"review_id,omitempty"`
	Name           string         `json:"name,omitempty"`
	ProfilePicture string         `json:"profile_picture,omitempty"`
	Rating         int            `json:"rating,omitempty"`
	Description    string         `json:"description,omitempty"`
	Images         []string       `json:"images,omitempty"`
	When           string         `json:"when,omitempty"`
	AgeDays        int            `json:"age_days,omitempty"`
	OwnerResponse  *OwnerResponse `json:"owner_response,omitempty"`
	Extended       bool           `json:"extended,omitempty"`
}

// PlacePayload is what a worker submits per scraped place.
// Schema is gosom-compatible.
type PlacePayload struct {
	InputID          string                 `json:"input_id,omitempty"`
	Link             string                 `json:"link,omitempty"`
	Cid              string                 `json:"cid,omitempty"`
	Title            string                 `json:"title"`
	Categories       []string               `json:"categories,omitempty"`
	Category         string                 `json:"category,omitempty"`
	Address          string                 `json:"address,omitempty"`
	OpenHours        map[string][]string    `json:"open_hours,omitempty"`
	PopularTimes     map[string]map[int]int `json:"popular_times,omitempty"`
	WebSite          string                 `json:"web_site,omitempty"`
	Phone            string                 `json:"phone,omitempty"`
	PlusCode         string                 `json:"plus_code,omitempty"`
	ReviewCount      int                    `json:"review_count"`
	ReviewRating     float64                `json:"review_rating"`
	ReviewsPerRating map[int]int            `json:"reviews_per_rating,omitempty"`
	Latitude         float64                `json:"latitude"`
	Longtitude       float64                `json:"longtitude"` // gosom typo preserved for wire compat
	Status           string                 `json:"status,omitempty"`
	Description      string                 `json:"description,omitempty"`
	ReviewsLink      string                 `json:"reviews_link,omitempty"`
	Thumbnail        string                 `json:"thumbnail,omitempty"`
	Timezone         string                 `json:"timezone,omitempty"`
	PriceRange       string                 `json:"price_range,omitempty"`
	DataID           string                 `json:"data_id,omitempty"`
	PlaceID          string                 `json:"place_id"`
	Images           []Image                `json:"images,omitempty"`
	Reservations     []LinkSource           `json:"reservations,omitempty"`
	OrderOnline      []LinkSource           `json:"order_online,omitempty"`
	Menu             *MenuPayload           `json:"menu,omitempty"`
	Owner            *Owner                 `json:"owner,omitempty"`
	CompleteAddress  *CompleteAddress       `json:"complete_address,omitempty"`
	About            []About                `json:"about,omitempty"`
	UserReviews      []ReviewPayload        `json:"user_reviews,omitempty"`
	Emails           []string               `json:"emails,omitempty"`
}
