package lingo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		return nil, &ValueError{"lingo: api key is required "}
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
		return nil, &RuntimeError{fmt.Sprintf("lingo: failed to read response body: %s ", err)}
	}
	err = json.Unmarshal(byteData, &response)
	if err != nil {
		preview := truncateResponse(string(byteData))
		return nil, &RuntimeError{fmt.Sprintf("lingo: failed to parse api response as json (status %d). this may indicate a gateway or proxy error. response: %s ", resp.StatusCode, preview)}
	}
	return response, nil
}

func (c *Client) do(url string, requestData map[string]any) (any, error) {
	// Marshall data
	dataByte, err := json.Marshal(requestData)
	if err != nil {
		return nil, &RuntimeError{fmt.Sprintf("lingo: failed to marshall request data: %s ", err)}
	}

	// Create HTTP request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(dataByte))
	if err != nil {
		return nil, &RuntimeError{fmt.Sprintf("lingo: failed to create a new request: %s ", err)}
	}

	// Set headers
	authorization := fmt.Sprintf("Bearer %s", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", authorization)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &RuntimeError{fmt.Sprintf("lingo: failed to send the http request to the server: %s ", err)}
	}

	// Read Body Once
	defer resp.Body.Close()
	byteData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &RuntimeError{fmt.Sprintf("lingo: failed to read response body: %s ", err)}
	}

	// Check Status Code
	if resp.StatusCode != 200 {
		responsePreview := truncateResponse(string(byteData))

		parts := strings.SplitN(resp.Status, " ", 2)

		var reasonPhrase string

		if len(parts) >= 2 {
			reasonPhrase = parts[1]
		}

		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			return nil, &RuntimeError{fmt.Sprintf("lingo: server error %d : %s.  this may be due to temporary service issues. response: %s ", resp.StatusCode, reasonPhrase, responsePreview)}
		} else if resp.StatusCode == 400 {
			return nil, &ValueError{fmt.Sprintf("lingo: invalid request (%d): %s.  response: %s ", resp.StatusCode, reasonPhrase, responsePreview)}
		} else {
			return nil, &RuntimeError{fmt.Sprintf("lingo: request failed (%d): %s. ", resp.StatusCode, responsePreview)}
		}
	}

	// Parse JSON from same byteData

	var jsonResponse map[string]any

	err = json.Unmarshal(byteData, &jsonResponse)
	if err != nil {
		preview := truncateResponse(string(byteData))
		return nil, &RuntimeError{fmt.Sprintf("lingo: failed to parse api response as json (status %d). this may indicate a gateway or proxy error. response: %s ", resp.StatusCode, preview)}
	}

	// Check API level error
	data := jsonResponse["data"]
	apiErr := jsonResponse["error"]

	if data == nil && apiErr != nil {
		return nil, &RuntimeError{fmt.Sprintf("lingo: %s", apiErr)}
	}

	// Return data field
	return data, nil

}

func countWords(payload any) int {
	switch v := payload.(type) {
	case []any:
		total := 0
		for _, item := range v {
			total += countWords(item)
		}
		return total
	case map[string]any:
		total := 0
		for _, value := range v {
			total += countWords(value)
		}
		return total
	case string:
		return len(strings.Fields(v))
	default:
		return 0
	}
}
