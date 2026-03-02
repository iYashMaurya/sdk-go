package lingo

import (
	"fmt"
	"strings"
)

type Config struct {
	APIKey             string
	APIUrl             string
	BatchSize          int
	IdealBatchItemSize int
}


type ConfigOption func (c *Config) error


func SetUrl (url string) ConfigOption {
	return func (c *Config) error {
		if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
			return fmt.Errorf("API URL must be a valid HTTP/HTTPS URL")
		}
		c.APIUrl = url
		return nil
	}
}

func SetBatchSize (batch int) ConfigOption {
	return func (c *Config) error {
		if (batch < 1 || batch > 250) {
			return fmt.Errorf("Batch Size should be between 1-250")
		}
		c.BatchSize = batch
		return nil 
	}
}

func SetIdealBatchItemSize (idelsize int) ConfigOption {
	return func(c *Config) error {
		if (idelsize < 1 || idelsize > 2500) {
			return fmt.Errorf("Batch Size should be between 1-2500")
		} 
		c.IdealBatchItemSize = idelsize
		return nil
	}
}



func NewEngineConfig (apiKey string, opts ...ConfigOption) (*Config, error) {
	const (
		defaultAPIUrl = "https://engine.lingo.dev"
		defaultBatchSize = 25
		defaultIdealBatchItemSize = 250
	)
	
	if apiKey == "" {
		return nil, fmt.Errorf("lingo: api key is required")
	}
	
	c := &Config{
		APIKey: apiKey,
		APIUrl:  defaultAPIUrl,
		BatchSize: defaultBatchSize,
		IdealBatchItemSize: defaultIdealBatchItemSize,
	}
	
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	
	return c, nil
}