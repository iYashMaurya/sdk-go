package lingo

import (
	"context"
	"fmt"
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
		for _, chunk := range chunks {
			chunkPayload := map[string]any{"data": chunk}
			if params.Reference != nil {
				chunkPayload["reference"] = params.Reference
			}

			result, err := c.localizeChunk(context.Background(), params.SourceLocale, workflowID, params.TargetLocale, chunkPayload, fast)
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
