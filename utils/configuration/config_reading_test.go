package configuration_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/allinbits/demeris-backend/utils/configuration"
)

const progName = "test"

type testConfig struct {
	String string
	Int    int
	Bool   bool
}

func (t testConfig) Validate() error {
	return nil
}

func TestReadConfig(t *testing.T) {
	tests := []struct {
		name          string
		expected      testConfig
		defaultValues map[string]string
		env           map[string]string
		wantErr       bool
	}{
		{
			"config reads successfully",
			testConfig{
				String: "string",
				Int:    42,
				Bool:   true,
			},
			map[string]string{},
			map[string]string{
				"String": "string",
				"Int":    "42",
				"Bool":   "true",
			},
			false,
		},
		{
			"config reads successfully from default value",
			testConfig{
				String: "string",
				Int:    42,
				Bool:   true,
			},
			map[string]string{
				"String": "string",
				"Int":    "42",
				"Bool":   "true",
			},
			map[string]string{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				require.NoError(t, os.Setenv(
					strings.ToUpper(fmt.Sprintf("%s_%s", progName, k)),
					v,
				),
				)
			}

			defer func() {
				os.Clearenv()
			}()

			read := testConfig{}
			err := configuration.ReadConfig(&read, progName, tt.defaultValues)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tt.expected, read)
		})
	}
}
