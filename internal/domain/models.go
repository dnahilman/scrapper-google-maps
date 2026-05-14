package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ---------- JSON helpers for GORM ----------

// JSONB is a generic JSONB column. Use typed wrappers below when shape is known.
type JSONB json.RawMessage

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

func (j *JSONB) Scan(src any) error {
	if src == nil {
		*j = nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		*j = append((*j)[:0], v...)
	case string:
		*j = []byte(v)
	default:
		return errors.New("JSONB: unsupported scan source")
	}
	return nil
}

func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

func (j *JSONB) UnmarshalJSON(data []byte) error {
	*j = append((*j)[:0], data...)
	return nil
}

// ---------- Models ----------

type City struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	EmsifaRegencyID   string    `gorm:"column:emsifa_regency_id;uniqueIndex"          json:"emsifa_regency_id"`
	EmsifaProvinceID  string    `gorm:"column:emsifa_province_id"                     json:"emsifa_province_id"`
	Name              string    `gorm:"column:name"                                   json:"name"`
	Slug              string    `gorm:"column:slug;uniqueIndex"                       json:"slug"`
	ProvinceName      string    `gorm:"column:province_name"                          json:"province_name"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (City) TableName() string { return "cities" }

type Kelurahan struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CityID            uuid.UUID `gorm:"type:uuid;index"                                json:"city_id"`
	EmsifaVillageID   string    `gorm:"column:emsifa_village_id;uniqueIndex"           json:"emsifa_village_id"`
	EmsifaDistrictID  string    `gorm:"column:emsifa_district_id;index"                json:"emsifa_district_id"`
	Name              string    `gorm:"column:name"                                    json:"name"`
	KecamatanName     string    `gorm:"column:kecamatan_name"                          json:"kecamatan_name"`
	Code              string    `gorm:"column:code"                                    json:"code,omitempty"`
	CreatedAt         time.Time `json:"created_at"`

	City *City `gorm:"foreignKey:CityID" json:"city,omitempty"`
}

func (Kelurahan) TableName() string { return "kelurahan" }

type Worker struct {
	ID             uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name           string       `gorm:"column:name;uniqueIndex"                        json:"name"`
	Hostname       string       `gorm:"column:hostname"                                json:"hostname,omitempty"`
	IPAddr         string       `gorm:"column:ip_addr;type:inet"                       json:"ip_addr,omitempty"`
	MaxConcurrency int          `gorm:"column:max_concurrency;default:2"               json:"max_concurrency"`
	Capabilities   JSONB        `gorm:"column:capabilities;type:jsonb;default:'{}'"    json:"capabilities,omitempty"`
	Status         WorkerStatus `gorm:"column:status;default:'offline'"                json:"status"`
	LastHeartbeat  *time.Time   `gorm:"column:last_heartbeat"                          json:"last_heartbeat,omitempty"`
	RegisteredAt   time.Time    `gorm:"column:registered_at"                           json:"registered_at"`
	Metadata       JSONB        `gorm:"column:metadata;type:jsonb;default:'{}'"        json:"metadata,omitempty"`
}

func (Worker) TableName() string { return "workers" }

type Job struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CityID       uuid.UUID `gorm:"type:uuid;index"                                json:"city_id"`
	Keyword      string    `gorm:"column:keyword;index"                           json:"keyword"`
	Status       JobStatus `gorm:"column:status;default:'pending'"                json:"status"`
	Options      JSONB     `gorm:"column:options;type:jsonb;default:'{}'"         json:"options,omitempty"`
	TotalTasks   int       `gorm:"column:total_tasks"                             json:"total_tasks"`
	DoneCount    int       `gorm:"column:done_count"                              json:"done_count"`
	FailedCount  int       `gorm:"column:failed_count"                            json:"failed_count"`
	CreatedAt    time.Time `json:"created_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedBy    string    `gorm:"column:created_by"                              json:"created_by,omitempty"`

	City  *City  `gorm:"foreignKey:CityID" json:"city,omitempty"`
	Tasks []Task `gorm:"foreignKey:JobID"  json:"-"`
}

func (Job) TableName() string { return "jobs" }

type Task struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	JobID          uuid.UUID  `gorm:"type:uuid;index"                                json:"job_id"`
	KelurahanID    uuid.UUID  `gorm:"type:uuid"                                      json:"kelurahan_id"`
	Priority       int        `gorm:"column:priority;default:0"                      json:"priority"`
	Status         TaskStatus `gorm:"column:status;default:'queued'"                 json:"status"`
	WorkerID       *uuid.UUID `gorm:"type:uuid;column:worker_id"                     json:"worker_id,omitempty"`
	Attempt        int        `gorm:"column:attempt;default:0"                       json:"attempt"`
	MaxAttempts    int        `gorm:"column:max_attempts;default:3"                  json:"max_attempts"`
	VisibleAfter   time.Time  `gorm:"column:visible_after"                           json:"visible_after"`
	LastHeartbeat  *time.Time `gorm:"column:last_heartbeat"                          json:"last_heartbeat,omitempty"`
	LastError      string     `gorm:"column:last_error"                              json:"last_error,omitempty"`
	ResultPath     string     `gorm:"column:result_path"                             json:"result_path,omitempty"`
	PlacesCount    *int       `gorm:"column:places_count"                            json:"places_count,omitempty"`
	EnqueuedAt     time.Time  `gorm:"column:enqueued_at"                             json:"enqueued_at"`
	StartedAt      *time.Time `gorm:"column:started_at"                              json:"started_at,omitempty"`
	CompletedAt    *time.Time `gorm:"column:completed_at"                            json:"completed_at,omitempty"`

	Kelurahan *Kelurahan `gorm:"foreignKey:KelurahanID" json:"kelurahan,omitempty"`
	Worker    *Worker    `gorm:"foreignKey:WorkerID"    json:"worker,omitempty"`
}

func (Task) TableName() string { return "tasks" }

type Place struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TaskID           *uuid.UUID `gorm:"type:uuid;column:task_id"                       json:"task_id,omitempty"`
	KelurahanID      *uuid.UUID `gorm:"type:uuid;column:kelurahan_id"                  json:"kelurahan_id,omitempty"`
	Keyword          string     `gorm:"column:keyword;index"                           json:"keyword"`

	PlaceID  string `gorm:"column:place_id;uniqueIndex" json:"place_id"`
	DataID   string `gorm:"column:data_id"              json:"data_id,omitempty"`
	Cid      string `gorm:"column:cid"                  json:"cid,omitempty"`

	Title           string  `gorm:"column:title"            json:"title"`
	Categories      StringArray `gorm:"column:categories;type:text[]" json:"categories,omitempty"`
	Category        string  `gorm:"column:category"         json:"category,omitempty"`
	Address         string  `gorm:"column:address"          json:"address,omitempty"`
	CompleteAddress JSONB   `gorm:"column:complete_address;type:jsonb" json:"complete_address,omitempty"`

	OpenHours    JSONB `gorm:"column:open_hours;type:jsonb"    json:"open_hours,omitempty"`
	PopularTimes JSONB `gorm:"column:popular_times;type:jsonb" json:"popular_times,omitempty"`

	Website  string `gorm:"column:website"   json:"website,omitempty"`
	Phone    string `gorm:"column:phone"     json:"phone,omitempty"`
	PlusCode string `gorm:"column:plus_code" json:"plus_code,omitempty"`

	ReviewCount      int     `gorm:"column:review_count"   json:"review_count"`
	ReviewRating     float64 `gorm:"column:review_rating"  json:"review_rating"`
	ReviewsPerRating JSONB   `gorm:"column:reviews_per_rating;type:jsonb" json:"reviews_per_rating,omitempty"`

	Latitude  float64 `gorm:"column:latitude"  json:"latitude"`
	Longitude float64 `gorm:"column:longitude" json:"longitude"`

	Status      string `gorm:"column:status;default:'active'" json:"status"`
	Description string `gorm:"column:description"             json:"description,omitempty"`
	ReviewsLink string `gorm:"column:reviews_link"            json:"reviews_link,omitempty"`
	Thumbnail   string `gorm:"column:thumbnail"               json:"thumbnail,omitempty"`
	Timezone    string `gorm:"column:timezone"                json:"timezone,omitempty"`
	Price       string `gorm:"column:price"                   json:"price,omitempty"`

	Images       JSONB       `gorm:"column:images;type:jsonb;default:'[]'"       json:"images,omitempty"`
	Reservations JSONB       `gorm:"column:reservations;type:jsonb;default:'[]'" json:"reservations,omitempty"`
	OrderOnline  JSONB       `gorm:"column:order_online;type:jsonb;default:'[]'" json:"order_online,omitempty"`
	Menu         JSONB       `gorm:"column:menu;type:jsonb"                      json:"menu,omitempty"`
	Owner        JSONB       `gorm:"column:owner;type:jsonb"                     json:"owner,omitempty"`
	About        JSONB       `gorm:"column:about;type:jsonb;default:'[]'"        json:"about,omitempty"`
	Emails       StringArray `gorm:"column:emails;type:text[]"                   json:"emails,omitempty"`

	ScrapedAt time.Time `gorm:"column:scraped_at" json:"scraped_at"`
}

func (Place) TableName() string { return "places" }

type Review struct {
	ID             uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PlaceID        string      `gorm:"column:place_id;index"                          json:"place_id"`
	ReviewID       string      `gorm:"column:review_id"                               json:"review_id,omitempty"`
	Name           string      `gorm:"column:name"                                    json:"name,omitempty"`
	ProfilePicture string      `gorm:"column:profile_picture"                         json:"profile_picture,omitempty"`
	Rating         *int        `gorm:"column:rating"                                  json:"rating,omitempty"`
	Description    string      `gorm:"column:description"                             json:"description,omitempty"`
	Images         StringArray `gorm:"column:images;type:text[]"                      json:"images,omitempty"`
	WhenText       string      `gorm:"column:when_text"                               json:"when,omitempty"`
	AgeDays        *int        `gorm:"column:age_days"                                json:"age_days,omitempty"`
	OwnerResponse  JSONB       `gorm:"column:owner_response;type:jsonb"               json:"owner_response,omitempty"`
	Extended       bool        `gorm:"column:extended;default:false"                  json:"extended"`
	CreatedAt      time.Time   `json:"created_at"`
}

func (Review) TableName() string { return "reviews" }

type SyncRecord struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TaskID     uuid.UUID  `gorm:"type:uuid;uniqueIndex"                          json:"task_id"`
	Status     SyncStatus `gorm:"column:status;default:'pending'"                json:"status"`
	Response   JSONB      `gorm:"column:response;type:jsonb"                     json:"response,omitempty"`
	Attempts   int        `gorm:"column:attempts;default:0"                      json:"attempts"`
	SyncedAt   *time.Time `gorm:"column:synced_at"                               json:"synced_at,omitempty"`
	LastError  string     `gorm:"column:last_error"                              json:"last_error,omitempty"`
}

func (SyncRecord) TableName() string { return "sync_records" }
