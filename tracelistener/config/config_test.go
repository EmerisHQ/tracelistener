package config_test

import (
	"os"
	"testing"

	"github.com/allinbits/demeris-backend/tracelistener/config"

	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{
			"configuration which doesn't passes validation",
			config.Config{},
			true,
		},
		{
			"configuration which passes validation",
			config.Config{
				FIFOPath:              "fifo",
				DatabaseConnectionURL: "db",
				ChainName:             "cn",
				Type:                  "type",
				Debug:                 false,
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err, "config content: %+v", tt.cfg)
		})
	}
}

func TestRead(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{
			"invalid configuration yields error",
			nil,
			true,
		},
		{
			"minimal valid configuration yields no error",
			map[string]string{
				"TRACELISTENER_DATABASECONNECTIONURL": "postgres://root:admin@?host=%2Fvar%2Ffolders%2F5l%2Frsbdhptd0tsgd07nqx0f4r7w0000gn%2FT%2Fdemo091325796&port=26257",
				"TRACELISTENER_CHAINNAME":             "gaia",
				"TRACELISTENER_TYPE":                  "gaia",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				require.NoError(t, os.Setenv(k, v))
			}

			defer os.Clearenv()

			cfg, err := config.Read()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)
		})
	}
}
