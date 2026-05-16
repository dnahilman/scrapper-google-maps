package emsifa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

type Client struct {
	baseURL string
	http    *retryablehttp.Client
}

func New(baseURL string) *Client {
	c := retryablehttp.NewClient()
	c.RetryMax = 3
	c.Logger = nil
	c.HTTPClient = &http.Client{Timeout: 20 * time.Second}
	return &Client{baseURL: baseURL, http: c}
}

type Province struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Regency struct {
	ID         string `json:"id"`
	ProvinceID string `json:"province_id"`
	Name       string `json:"name"`
}

type District struct {
	ID        string `json:"id"`
	RegencyID string `json:"regency_id"`
	Name      string `json:"name"`
}

type Village struct {
	ID         string `json:"id"`
	DistrictID string `json:"district_id"`
	Name       string `json:"name"`
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	url := c.baseURL + path
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get %s: status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) Provinces(ctx context.Context) ([]Province, error) {
	var out []Province
	err := c.get(ctx, "/provinces.json", &out)
	return out, err
}

func (c *Client) Regencies(ctx context.Context, provinceID string) ([]Regency, error) {
	var out []Regency
	err := c.get(ctx, "/regencies/"+provinceID+".json", &out)
	return out, err
}

func (c *Client) Districts(ctx context.Context, regencyID string) ([]District, error) {
	var out []District
	err := c.get(ctx, "/districts/"+regencyID+".json", &out)
	return out, err
}

func (c *Client) Villages(ctx context.Context, districtID string) ([]Village, error) {
	var out []Village
	err := c.get(ctx, "/villages/"+districtID+".json", &out)
	return out, err
}
