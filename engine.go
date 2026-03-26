package lingo

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"golang.org/x/sync/errgroup"
)

func (c *Client) localizeRaw(payload map[string]any, params LocalizationParams, concurrent bool) (map[string]any, error) {
	chunks := c.extractChunks(payload)
	if len(chunks) == 0 {
		return map[string]any{}, nil
	}

	workflowID, err := gonanoid.New()
	if err != nil {
		return nil, &RuntimeError{fmt.Sprintf("lingo: failed to generate workflow id: %s", err)}
	}

	fast := false
	if params.Fast != nil {
		fast = *params.Fast
	}

	merged := make(map[string]any)

	if concurrent {
		var mu sync.Mutex
		g, ctx := errgroup.WithContext(context.Background())

		for _, chunk := range chunks {
			chunkPayload := map[string]any{"data": chunk}
			if params.Reference != nil {
				chunkPayload["reference"] = params.Reference
			}

			g.Go(func() error {
				result, err := c.localizeChunk(ctx, params.SourceLocale, workflowID, params.TargetLocale, chunkPayload, fast)
				if err != nil {
					return err
				}

				resultMap, ok := result.(map[string]any)
				if !ok {
					return &RuntimeError{"lingo: unexpected response type from server"}
				}

				mu.Lock()
				for k, v := range resultMap {
					merged[k] = v
				}
				mu.Unlock()

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, err
		}
	} else {
		ctx := context.Background()
		for _, chunk := range chunks {
			chunkPayload := map[string]any{"data": chunk}
			if params.Reference != nil {
				chunkPayload["reference"] = params.Reference
			}

			result, err := c.localizeChunk(ctx, params.SourceLocale, workflowID, params.TargetLocale, chunkPayload, fast)
			if err != nil {
				return nil, err
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				return nil, &RuntimeError{"lingo: unexpected response type from server"}
			}

			for k, v := range resultMap {
				merged[k] = v
			}
		}
	}

	return merged, nil
}

// LocalizeText translates a single text string to the target locale specified in params.
func (c *Client) LocalizeText(text string, params LocalizationParams) (string, error) {
	if text == "" {
		return "", &ValueError{"lingo: text must not be empty"}
	}

	payload := map[string]any{"text": text}

	result, err := c.localizeRaw(payload, params, false)
	if err != nil {
		return "", err
	}

	localized, ok := result["text"].(string)
	if !ok {
		return "", &RuntimeError{"lingo: unexpected response type for localized text"}
	}

	return localized, nil
}

// LocalizeObject translates all string values in the given map to the target locale specified in params.
func (c *Client) LocalizeObject(obj map[string]any, params LocalizationParams, concurrent bool) (map[string]any, error) {
	return c.localizeRaw(obj, params, concurrent)
}

// LocalizeChat translates the text field of each chat message to the target locale specified in params.
func (c *Client) LocalizeChat(chat []map[string]string, params LocalizationParams) ([]map[string]string, error) {
	if len(chat) == 0 {
		return []map[string]string{}, nil
	}

	for i, msg := range chat {
		if _, ok := msg["name"]; !ok {
			return nil, &ValueError{fmt.Sprintf("lingo: chat message at index %d is missing 'name' field", i)}
		}
		if _, ok := msg["text"]; !ok {
			return nil, &ValueError{fmt.Sprintf("lingo: chat message at index %d is missing 'text' field", i)}
		}
	}

	chatPayload := make([]any, len(chat))
	for i, msg := range chat {
		chatPayload[i] = map[string]any{
			"name": msg["name"],
			"text": msg["text"],
		}
	}
	payload := map[string]any{"chat": chatPayload}

	result, err := c.localizeRaw(payload, params, false)
	if err != nil {
		return nil, err
	}

	rawChat, ok := result["chat"].([]any)
	if !ok {
		return nil, &RuntimeError{"lingo: unexpected response type for localized chat"}
	}

	if len(rawChat) != len(chat) {
		return nil, &RuntimeError{fmt.Sprintf("lingo: expected %d chat messages but got %d", len(chat), len(rawChat))}
	}

	localized := make([]map[string]string, len(rawChat))
	for i, item := range rawChat {
		msgMap, ok := item.(map[string]any)
		if !ok {
			return nil, &RuntimeError{fmt.Sprintf("lingo: unexpected response type for chat message at index %d", i)}
		}
		name, ok := msgMap["name"].(string)
		if !ok {
			return nil, &RuntimeError{fmt.Sprintf("lingo: unexpected response type for chat message name at index %d", i)}
		}
		text, ok := msgMap["text"].(string)
		if !ok {
			return nil, &RuntimeError{fmt.Sprintf("lingo: unexpected response type for chat message text at index %d", i)}
		}
		localized[i] = map[string]string{
			"name": name,
			"text": text,
		}
	}

	return localized, nil
}

// RecognizeLocale detects the locale of the given text.
func (c *Client) RecognizeLocale(text string) (string, error) {
	if text == "" {
		return "", &ValueError{"lingo: text must not be empty"}
	}

	endpoint, err := url.JoinPath(c.config.APIURL, "/recognize")
	if err != nil {
		return "", &RuntimeError{fmt.Sprintf("lingo: unable to join path: %s", err)}
	}

	requestData := map[string]any{"text": text}

	data, err := c.do(context.Background(), endpoint, requestData)
	if err != nil {
		return "", err
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return "", &RuntimeError{"lingo: unexpected response type for recognized locale"}
	}

	locale, ok := dataMap["locale"].(string)
	if !ok {
		return "", &RuntimeError{"lingo: missing locale field in response"}
	}

	return locale, nil
}

// WhoAmI returns the authenticated user's information, or nil if not authenticated.
func (c *Client) WhoAmI() (map[string]string, error) {
	endpoint, err := url.JoinPath(c.config.APIURL, "/whoami")
	if err != nil {
		return nil, &RuntimeError{fmt.Sprintf("lingo: unable to join path: %s", err)}
	}

	data, err := c.do(context.Background(), endpoint, map[string]any{})
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, nil
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return nil, &RuntimeError{"lingo: unexpected response type for whoami"}
	}

	result := make(map[string]string, len(dataMap))
	for k, v := range dataMap {
		str, ok := v.(string)
		if !ok {
			continue
		}
		result[k] = str
	}

	return result, nil
}
