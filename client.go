package lingo

import (
	"net/http"
	"time"
)

type Client struct {
	config Config
	httpClient *http.Client
}

func NewClient(apiKey string, opts ...ConfigOption) (*Client, error) {
	if apiKey == "" {
		return nil, &valueError{"lingo: api key is required"}
	}
	config, err  := newEngineConfig(apiKey, opts...)
	if err != nil {
		return nil, err
	}
	c := &Client{
		config: *config,
		httpClient: &http.Client{
			Timeout: 60*time.Second,
		},
	}
	return c, nil
}
