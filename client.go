package lingo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	MaxResponseLength = 200
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

// TruncateResponse truncates text to MaxResponseLength, appending "..." if truncated.
func TruncateResponse(text string) string {
	if len(text) > MaxResponseLength {
		return text[:MaxResponseLength] + "..."
	}
	return text
}

func (c *Client) do(ctx context.Context, endpoint string, requestData any) (any, error) {
	// Marshall data
	dataByte, err := json.Marshal(requestData)
	if err != nil {
		return nil, &RuntimeError{Message: fmt.Sprintf("lingo: failed to marshall request data: %s", err)}
	}

	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
		}

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(dataByte))
		if err != nil {
			return nil, &RuntimeError{Message: fmt.Sprintf("lingo: failed to create a new request: %s", err)}
		}

		// Set headers
		authorization := fmt.Sprintf("Bearer %s", c.config.APIKey)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Authorization", authorization)

		// Execute request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = &RuntimeError{Message: fmt.Sprintf("lingo: failed to send the http request to the server: %s", err)}
			continue
		}

		// Read Body Once
		byteData, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, &RuntimeError{Message: fmt.Sprintf("lingo: failed to read response body: %s", err)}
		}

		// Check Status Code
		if resp.StatusCode != http.StatusOK {
			responsePreview := TruncateResponse(string(byteData))

			parts := strings.SplitN(resp.Status, " ", 2)

			var reasonPhrase string

			if len(parts) >= 2 {
				reasonPhrase = parts[1]
			}

			if resp.StatusCode >= http.StatusInternalServerError && resp.StatusCode < 600 {
				lastErr = &RuntimeError{Message: fmt.Sprintf("lingo: server error %d : %s. this may be due to temporary service issues. response: %s", resp.StatusCode, reasonPhrase, responsePreview), StatusCode: resp.StatusCode}
				continue
			} else if resp.StatusCode == http.StatusBadRequest {
				return nil, &ValueError{fmt.Sprintf("lingo: invalid request (%d): %s. response: %s", resp.StatusCode, reasonPhrase, responsePreview)}
			} else {
				return nil, &RuntimeError{Message: fmt.Sprintf("lingo: request failed (%d): %s.", resp.StatusCode, responsePreview), StatusCode: resp.StatusCode}
			}
		}

		// Parse JSON from same byteData

		var jsonResponse map[string]any

		err = json.Unmarshal(byteData, &jsonResponse)
		if err != nil {
			preview := TruncateResponse(string(byteData))
			return nil, &RuntimeError{Message: fmt.Sprintf("lingo: failed to parse api response as json (status %d). this may indicate a gateway or proxy error. response: %s", resp.StatusCode, preview), StatusCode: resp.StatusCode}
		}

		// Check API level error
		data := jsonResponse["data"]
		apiErr := jsonResponse["error"]

		if data == nil && apiErr != nil {
			return nil, &RuntimeError{Message: fmt.Sprintf("lingo: %s", apiErr)}
		}

		// Return data field
		return data, nil
	}

	return nil, lastErr
}

// CountWords counts the total number of words in the given payload recursively.
func CountWords(payload any) int {
	switch v := payload.(type) {
	case []any:
		total := 0
		for _, item := range v {
			total += CountWords(item)
		}
		return total
	case map[string]any:
		total := 0
		for _, value := range v {
			total += CountWords(value)
		}
		return total
	case string:
		return len(strings.Fields(v))
	default:
		return 0
	}
}

// ExtractChunks splits a payload map into smaller chunks based on configured batch size and ideal item size.
func (c *Client) ExtractChunks(payload map[string]any) []map[string]any {
	total := len(payload)
	processed := 0
	var result []map[string]any
	currentChunk := make(map[string]any)
	var currentItemCount int

	for key, value := range payload {
		currentChunk[key] = value
		currentItemCount++
		currentChunkSize := CountWords(currentChunk)
		processed++

		if currentChunkSize > c.config.IdealBatchItemSize || currentItemCount >= c.config.BatchSize || processed == total {
			result = append(result, currentChunk)
			currentChunk = make(map[string]any)
			currentItemCount = 0
		}
	}

	return result
}

func (c *Client) localizeChunk(ctx context.Context, sourceLocale *string, workflowID, targetLocale string, payload map[string]any, fast bool) (any, error) {
	endpoint, err := url.JoinPath(c.config.APIURL, "/i18n")
	if err != nil {
		return nil, &RuntimeError{Message: fmt.Sprintf("lingo: unable to join path: %s", err)}
	}

	requestData := &RequestData{
		Param: parameter{
			WorkflowID: workflowID,
			Fast:       fast,
		},
		Locale: locale{
			Source: sourceLocale,
			Target: targetLocale,
		},
		Data: payload["data"],
	}

	if raw, ok := payload["reference"]; ok {
		ref, ok := raw.(map[string]map[string]any)
		if !ok {
			return nil, &ValueError{"lingo: reference has invalid type"}
		}
		requestData.Reference = ref
	}

	data, err := c.do(ctx, endpoint, requestData)

	if err != nil {
		return nil, err
	}

	return data, nil
}
