package lingo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	maxLength = 200
)

type Client struct {
	config     Config
	httpClient *http.Client
}

func NewClient(apiKey string, opts ...ConfigOption) (*Client, error) {
	if apiKey == "" {
		return nil, &ValueError{"lingo: api key is required"}
	}
	config, err := newEngineConfig(apiKey, opts...)
	if err != nil {
		return nil, err
	}
	c := &Client{
		config: *config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
	return c, nil
}

func truncateResponse(text string) string {
	if len(text) > maxLength {
		return text[:maxLength] + "..."
	}
	return text
}

func safeParseJSON(resp *http.Response) (map[string]any, error) {
	var response map[string]any
	defer resp.Body.Close()
	byteData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &RuntimeError{fmt.Sprintf("lingo: failed to read response body: %s", err)}
	}
	err = json.Unmarshal(byteData, &response)
	if err != nil {
		preview := truncateResponse(string(byteData))
		return nil, &RuntimeError{fmt.Sprintf("lingo: failed to parse api response as json (status %d). this may indicate a gateway or proxy error. response: %s", resp.StatusCode, preview)}
	}
	return response, nil
}
