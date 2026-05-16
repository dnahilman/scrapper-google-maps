package workeragent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/dnahilman/scrapper-go/internal/domain"
	"github.com/dnahilman/scrapper-go/internal/queue"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// HTTPError carries the HTTP status code from a non-2xx response.
type HTTPError struct {
	Code int
	Body string
}

func (e *HTTPError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.Code, e.Body) }

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return &HTTPError{Code: resp.StatusCode, Body: string(b)}
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

type RegisterRequest struct {
	Name           string         `json:"name"`
	Hostname       string         `json:"hostname,omitempty"`
	MaxConcurrency int            `json:"max_concurrency"`
	Capabilities   map[string]any `json:"capabilities,omitempty"`
}

func (c *Client) Register(ctx context.Context, req *RegisterRequest) (*domain.Worker, error) {
	var w domain.Worker
	if err := c.do(ctx, http.MethodPost, "/api/v1/internal/workers/register", req, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

func (c *Client) Heartbeat(ctx context.Context, workerID uuid.UUID) error {
	return c.do(ctx, http.MethodPost, "/api/v1/internal/workers/"+workerID.String()+"/heartbeat", nil, nil)
}

// Claim returns nil, nil when the queue is empty.
func (c *Client) Claim(ctx context.Context, workerID uuid.UUID) (*queue.ClaimedTask, error) {
	var out queue.ClaimedTask
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/internal/tasks/claim",
		bytes.NewReader(mustJSON(map[string]any{"worker_id": workerID})))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil, nil
	case http.StatusOK:
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}
		return &out, nil
	default:
		b, _ := io.ReadAll(resp.Body)
		return nil, errors.New("claim failed: " + string(b))
	}
}

func (c *Client) TaskHeartbeat(ctx context.Context, taskID uuid.UUID) error {
	return c.do(ctx, http.MethodPost, "/api/v1/internal/tasks/"+taskID.String()+"/heartbeat", nil, nil)
}

func (c *Client) Ack(ctx context.Context, taskID uuid.UUID, placesCount int, resultPath string) error {
	return c.do(ctx, http.MethodPost, "/api/v1/internal/tasks/"+taskID.String()+"/ack",
		map[string]any{"places_count": placesCount, "result_path": resultPath}, nil)
}

func (c *Client) Nack(ctx context.Context, taskID uuid.UUID, errMsg string) error {
	return c.do(ctx, http.MethodPost, "/api/v1/internal/tasks/"+taskID.String()+"/nack",
		map[string]any{"error": errMsg}, nil)
}

type SubmitPlacesRequest struct {
	TaskID      uuid.UUID              `json:"task_id"`
	KelurahanID uuid.UUID              `json:"kelurahan_id"`
	Keyword     string                 `json:"keyword"`
	Places      []domain.PlacePayload  `json:"places"`
}

func (c *Client) SubmitPlaces(ctx context.Context, req *SubmitPlacesRequest) error {
	return c.do(ctx, http.MethodPost, "/api/v1/internal/places", req, nil)
}

type PlaceScrapedEventPayload struct {
	TaskID        string `json:"task_id"`
	PlaceID       string `json:"place_id"`
	Title         string `json:"title"`
	KelurahanName string `json:"kelurahan_name"`
	KecamatanName string `json:"kecamatan_name"`
	CityName      string `json:"city_name"`
	Index         int    `json:"index"`
	Total         int    `json:"total"`
}

func (c *Client) PublishPlaceScraped(ctx context.Context, p *PlaceScrapedEventPayload) error {
	return c.do(ctx, http.MethodPost, "/api/v1/internal/events/place-scraped", p, nil)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
