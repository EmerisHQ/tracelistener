package config

import (
	"github.com/go-playground/validator/v10"

	"github.com/emerishq/tracelistener/configuration"
	"github.com/emerishq/tracelistener/validation"
)

type Config struct {
	FIFOPath              string `validate:"required"`
	ChainName             string `validate:"required"`
	DatabaseConnectionURL string `validate:"required"`
	LogPath               string
	Debug                 bool
	JSONLogs              bool
	EnableCpuProfiling    bool

	// Processors configs
	Processor ProcessorConfig

	// Exporter http port
	ExporterHTTPPort string
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

func Read() (*Config, error) {
	var c Config

	return &c, configuration.ReadConfig(&c, "tracelistener", map[string]string{
		"FIFOPath": "./.tracelistener.fifo",
	})
}
