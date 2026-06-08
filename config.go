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
	MaxConcurrency     int
}

// ConfigOption is a function that configures the SDK client.
type ConfigOption func(c *Config) error

// SetMaxConcurrency sets the maximum number of goroutines that may issue HTTP
// requests at the same time. Pass a value ≥ 1; values < 1 are ignored.
func SetMaxConcurrency(n int) ConfigOption {
	return func(c *Config) error {
		if n < 1 {
			return &ValueError{Message: "lingo: max concurrency must be >= 1"}
		}
		c.MaxConcurrency = n
		return nil
	}
}

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
		defaultMaxConcurrency     = 10
	)

	c := &Config{
		APIKey:             apiKey,
		APIURL:             defaultAPIURL,
		BatchSize:          defaultBatchSize,
		IdealBatchItemSize: defaultIdealBatchItemSize,
		MaxConcurrency:     defaultMaxConcurrency,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}
