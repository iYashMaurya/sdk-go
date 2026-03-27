package lingo

import (
	"strings"
)

// Config holds the SDK configuration options.
type Config struct {
	APIKey             string
	APIURL             string
	BatchSize          int
	IdealBatchItemSize int
}

// ConfigOption is a function that configures the SDK client.
type ConfigOption func(c *Config) error

// SetURL configures the API endpoint URL.
// The URL must start with http:// or https://.
func SetURL(url string) ConfigOption {
	return func(c *Config) error {
		if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
			return &ValueError{Message: "lingo: api url must be a valid http/https url"}
		}
		c.APIURL = url
		return nil
	}
}

// SetBatchSize configures the maximum number of items per chunk (1-250).
func SetBatchSize(batch int) ConfigOption {
	return func(c *Config) error {
		if batch < 1 || batch > 250 {
			return &ValueError{Message: "lingo: batch size should be between 1-250"}
		}
		c.BatchSize = batch
		return nil
	}
}

// SetIdealBatchItemSize configures the target word count per chunk (1-2500).
func SetIdealBatchItemSize(size int) ConfigOption {
	return func(c *Config) error {
		if size < 1 || size > 2500 {
			return &ValueError{Message: "lingo: ideal batch item size should be between 1-2500"}
		}
		c.IdealBatchItemSize = size
		return nil
	}
}

func newEngineConfig(apiKey string, opts ...ConfigOption) (*Config, error) {
	const (
		defaultAPIURL             = "https://engine.lingo.dev"
		defaultBatchSize          = 25
		defaultIdealBatchItemSize = 250
	)

	c := &Config{
		APIKey:             apiKey,
		APIURL:             defaultAPIURL,
		BatchSize:          defaultBatchSize,
		IdealBatchItemSize: defaultIdealBatchItemSize,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}
