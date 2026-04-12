package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

const (
	baseURL    = "https://apis.data.go.kr"
	maxRetries = 3
)

// Client is a rate-limited, retrying HTTP client for the data.go.kr open API.
type Client struct {
	apiKey     string
	httpClient *http.Client
	limiter    *rate.Limiter // 5 requests/second
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		limiter:    rate.NewLimiter(rate.Every(200*time.Millisecond), 5),
	}
}

// Get calls endpoint (relative to baseURL) with the given query params plus
// the service key. It retries up to maxRetries times with exponential backoff.
func (c *Client) Get(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	params.Set("serviceKey", c.apiKey)
	params.Set("_type", "json")

	fullURL := baseURL + endpoint + "?" + params.Encode()

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(1<<attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}

		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Get(fullURL)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, endpoint)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}
		return body, nil
	}
	return nil, fmt.Errorf("all retries exhausted for %s: %w", endpoint, lastErr)
}

// --- Common response envelope ---

// APIResponse is the standard data.go.kr JSON envelope.
type APIResponse struct {
	Response struct {
		Header struct {
			ResultCode string `json:"resultCode"`
			ResultMsg  string `json:"resultMsg"`
		} `json:"header"`
		Body struct {
			Items      json.RawMessage `json:"items"`
			NumOfRows  int             `json:"numOfRows"`
			PageNo     int             `json:"pageNo"`
			TotalCount int             `json:"totalCount"`
		} `json:"body"`
	} `json:"response"`
}

// Parse decodes a raw API response body into APIResponse.
func Parse(body []byte) (*APIResponse, error) {
	var r APIResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	if r.Response.Header.ResultCode != "00" {
		return nil, fmt.Errorf("API error %s: %s",
			r.Response.Header.ResultCode,
			r.Response.Header.ResultMsg)
	}
	return &r, nil
}
