package configuration

import (
	"fmt"
	"strings"

	"github.com/iamolegga/enviper"

	"github.com/spf13/viper"
)

// Validator is an object that implements a validation method, which accepts no argument and returns an error.
type Validator interface {
	Validate() error
}

// ReadConfig reads the TOML configuration file in predefined standard paths into v, returns an error if v.Validate()
// returns error, or some configuration file reading error happens.
// v is the destination struct, configName is the name used for the configuration file.
// ReadConfig will not return an error for missing configuration file, since the fields contained in v can be also
// read from environment variables.
func ReadConfig(v Validator, configName string, defaultValues map[string]string) error {
	vip := enviper.New(viper.New())

	for k, v := range defaultValues {
		vip.SetDefault(k, v)
	}

	vip.SetConfigName(configName)
	vip.AddConfigPath(fmt.Sprintf("/etc/%s", configName))
	vip.AddConfigPath(fmt.Sprintf("$HOME/.%s", configName))
	vip.AddConfigPath(".")
	vip.SetEnvPrefix(strings.ToLower(configName))
	vip.AutomaticEnv()

	if err := vip.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	if err := vip.Unmarshal(v); err != nil {
		return fmt.Errorf("config error: %s", err)
	}

	return v.Validate()
}
