package lingo

import (
	"strings"
)

type Config struct {
	APIKey             string
	APIURL             string
	BatchSize          int
	IdealBatchItemSize int
}

type ConfigOption func(c *Config) error

func SetURL(url string) ConfigOption {
	return func(c *Config) error {
		if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
			return &valueError{"lingo: api url must be a valid http/https url"}
		}
		c.APIURL = url
		return nil
	}
}

func SetBatchSize(batch int) ConfigOption {
	return func(c *Config) error {
		if batch < 1 || batch > 250 {
			return &valueError{"lingo: batch size should be between 1-250"}
		}
		c.BatchSize = batch
		return nil
	}
}

func SetIdealBatchItemSize(size int) ConfigOption {
	return func(c *Config) error {
		if size < 1 || size > 2500 {
			return &valueError{"lingo: ideal batch item size should be between 1-2500"}
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
