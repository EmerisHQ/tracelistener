package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/spf13/viper"
)

type Config struct {
	FIFOPath              string `validate:"required"`
	NodeRPC               string `validate:"required"`
	DatabaseConnectionURL string `validate:"required"`
	LogPath               string `validate:"required"`
	Type                  string `validate:"required"`
	Debug                 bool

	// Processors configs
	Gaia GaiaConfig
}

type GaiaConfig struct {
	ProcessorsEnabled []string
}

func (c Config) Validate() error {
	err := validator.New().Struct(c)
	if err == nil {
		return nil
	}
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return err
	}

	missingFields := []string{}
	for _, e := range ve {
		switch e.Tag() {
		case "required":
			missingFields = append(missingFields, e.StructField())
		}
	}

	return fmt.Errorf("missing configuration file fields: %v", strings.Join(missingFields, ", "))
}

func Read() (*Config, error) {
	viper.SetDefault("FIFOPath", "./.tracelistener.fifo")
	viper.SetDefault("LogPath", "./tracelistener.log")

	viper.SetConfigName("tracelistener")
	viper.SetConfigType("toml")
	viper.AddConfigPath("/etc/tracelistener/")
	viper.AddConfigPath("$HOME/.tracelistener")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var c Config
	if err := viper.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("config error: %s \n", err)
	}

	return &c, c.Validate()
}
