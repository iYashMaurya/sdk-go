package lingo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"golang.org/x/sync/errgroup"
)

// localizeRaw splits payload into chunks and localizes each chunk,
// either sequentially or concurrently based on the concurrent flag.
func (c *Client) localizeRaw(ctx context.Context, payload map[string]any, params LocalizationParams, concurrent bool) (map[string]any, error) {
	chunks := c.ExtractChunks(payload)
	if len(chunks) == 0 {
		return map[string]any{}, nil
	}

	workflowID, err := gonanoid.New()
	if err != nil {
		return nil, &RuntimeError{Message: fmt.Sprintf("lingo: failed to generate workflow id: %s", err)}
	}

	fast := false
	if params.Fast != nil {
		fast = *params.Fast
	}

	merged := make(map[string]any)

	if concurrent {
		var mu sync.Mutex
		g, gCtx := errgroup.WithContext(ctx)

		for _, chunk := range chunks {
			chunkPayload := map[string]any{"data": chunk}
			if params.Reference != nil {
				chunkPayload["reference"] = params.Reference
			}

			g.Go(func() error {
				result, err := c.localizeChunk(gCtx, params.SourceLocale, workflowID, params.TargetLocale, chunkPayload, fast)
				if err != nil {
					return err
				}

				resultMap, ok := result.(map[string]any)
				if !ok {
					return &RuntimeError{Message: "lingo: unexpected response type from server"}
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
				return nil, &RuntimeError{Message: "lingo: unexpected response type from server"}
			}

			for k, v := range resultMap {
				merged[k] = v
			}
		}
	}

	return merged, nil
}

// LocalizeText translates a single text string to the target locale specified in params.
func (c *Client) LocalizeText(ctx context.Context, text string, params LocalizationParams) (string, error) {
	if text == "" {
		return "", &ValueError{Message: "lingo: text must not be empty"}
	}

	payload := map[string]any{"text": text}

	result, err := c.localizeRaw(ctx, payload, params, false)
	if err != nil {
		return "", err
	}

	localized, ok := result["text"].(string)
	if !ok {
		return "", &RuntimeError{Message: "lingo: unexpected response type for localized text"}
	}

	return localized, nil
}

// LocalizeObject translates all string values in the given map to the target locale specified in params.
func (c *Client) LocalizeObject(ctx context.Context, obj map[string]any, params LocalizationParams, concurrent bool) (map[string]any, error) {
	return c.localizeRaw(ctx, obj, params, concurrent)
}

// LocalizeChat translates the text field of each chat message to the target locale specified in params.
func (c *Client) LocalizeChat(ctx context.Context, chat []map[string]string, params LocalizationParams) ([]map[string]string, error) {
	if len(chat) == 0 {
		return []map[string]string{}, nil
	}

	for i, msg := range chat {
		if _, ok := msg["name"]; !ok {
			return nil, &ValueError{Message: fmt.Sprintf("lingo: chat message at index %d is missing 'name' field", i)}
		}
		if _, ok := msg["text"]; !ok {
			return nil, &ValueError{Message: fmt.Sprintf("lingo: chat message at index %d is missing 'text' field", i)}
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

	result, err := c.localizeRaw(ctx, payload, params, false)
	if err != nil {
		return nil, err
	}

	rawChat, ok := result["chat"].([]any)
	if !ok {
		return nil, &RuntimeError{Message: "lingo: unexpected response type for localized chat"}
	}

	if len(rawChat) != len(chat) {
		return nil, &RuntimeError{Message: fmt.Sprintf("lingo: expected %d chat messages but got %d", len(chat), len(rawChat))}
	}

	localized := make([]map[string]string, len(rawChat))
	for i, item := range rawChat {
		msgMap, ok := item.(map[string]any)
		if !ok {
			return nil, &RuntimeError{Message: fmt.Sprintf("lingo: unexpected response type for chat message at index %d", i)}
		}
		name, ok := msgMap["name"].(string)
		if !ok {
			return nil, &RuntimeError{Message: fmt.Sprintf("lingo: unexpected response type for chat message name at index %d", i)}
		}
		text, ok := msgMap["text"].(string)
		if !ok {
			return nil, &RuntimeError{Message: fmt.Sprintf("lingo: unexpected response type for chat message text at index %d", i)}
		}
		localized[i] = map[string]string{
			"name": name,
			"text": text,
		}
	}

	return localized, nil
}

// RecognizeLocale detects the locale of the given text.
func (c *Client) RecognizeLocale(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", &ValueError{Message: "lingo: text must not be empty"}
	}

	endpoint, err := url.JoinPath(c.config.APIURL, "/recognize")
	if err != nil {
		return "", &RuntimeError{Message: fmt.Sprintf("lingo: unable to join path: %s", err)}
	}

	requestData := map[string]any{"text": text}

	result, err := c.do(ctx, endpoint, requestData)
	if err != nil {
		return "", err
	}

	locale, ok := result["locale"].(string)
	if !ok {
		return "", &RuntimeError{Message: "lingo: missing locale field in response"}
	}

	return locale, nil
}

// WhoAmI returns the authenticated user's information, or nil if not authenticated.
func (c *Client) WhoAmI(ctx context.Context) (map[string]string, error) {
	endpoint, err := url.JoinPath(c.config.APIURL, "/whoami")
	if err != nil {
		return nil, &RuntimeError{Message: fmt.Sprintf("lingo: unable to join path: %s", err)}
	}

	result, err := c.do(ctx, endpoint, map[string]any{})
	if err != nil {
		var re *RuntimeError
		if errors.As(err, &re) && re.StatusCode == http.StatusUnauthorized {
			return nil, nil
		}
		return nil, err
	}

	data := result["data"]
	if data == nil {
		return nil, nil
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return nil, &RuntimeError{Message: "lingo: unexpected response type for whoami"}
	}

	info := make(map[string]string, len(dataMap))
	for k, v := range dataMap {
		str, ok := v.(string)
		if !ok {
			continue
		}
		info[k] = str
	}

	if len(info) == 0 {
		return nil, nil
	}

	return info, nil
}

// BatchLocalizeText translates a single text string into multiple target locales concurrently.
func (c *Client) BatchLocalizeText(ctx context.Context, text string, sourceLocale *string, fast *bool, targetLocales []string) ([]string, error) {
	if text == "" {
		return nil, &ValueError{Message: "lingo: text must not be empty"}
	}
	if len(targetLocales) == 0 {
		return []string{}, nil
	}

	results := make([]string, len(targetLocales))
	g, gCtx := errgroup.WithContext(ctx)

	for i, targetLocale := range targetLocales {
		params := LocalizationParams{
			SourceLocale: sourceLocale,
			TargetLocale: targetLocale,
			Fast:         fast,
		}
		g.Go(func() error {
			localized, err := c.LocalizeText(gCtx, text, params)
			if err != nil {
				return err
			}
			results[i] = localized
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// BatchLocalizeObjects translates multiple objects concurrently using the same localization params.
func (c *Client) BatchLocalizeObjects(ctx context.Context, objects []map[string]any, params LocalizationParams) ([]map[string]any, error) {
	if len(objects) == 0 {
		return []map[string]any{}, nil
	}

	results := make([]map[string]any, len(objects))
	g, gCtx := errgroup.WithContext(ctx)

	for i, obj := range objects {
		g.Go(func() error {
			localized, err := c.LocalizeObject(gCtx, obj, params, false)
			if err != nil {
				return err
			}
			results[i] = localized
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}
