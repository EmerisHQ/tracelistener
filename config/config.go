package config

import (
	"fmt"

	"github.com/allinbits/tracelistener/utils/validation"

	"github.com/allinbits/tracelistener/utils/configuration"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	FIFOPath              string `validate:"required"`
	ChainName             string `validate:"required"`
	DatabaseConnectionURL string `validate:"required"`
	LogPath               string
	Version               string `validate:"required"`
	ServiceProvider       string
	RunBlockWatcher       bool
	Debug                 bool

	// Processors configs
	ProcessorConfig ProcessorConfig
}

type ProcessorConfig struct {
	ProcessorsEnabled []string
}

func (c Config) Validate() error {
	err := validator.New().Struct(c)
	if err == nil {
		return nil
	}

	return validation.MissingFieldsErr(err, false)
}

func (c Config) ServiceProviderAddress() string {
	if c.ServiceProvider != "" {
		return c.ServiceProvider
	}

	return fmt.Sprintf("sdk-service-%s:9090", c.Version)
}

func Read() (*Config, error) {
	var c Config

	return &c, configuration.ReadConfig(&c, "tracelistener", map[string]string{
		"FIFOPath":        "./.tracelistener.fifo",
		"RunBlockWatcher": "true",
	})
}
