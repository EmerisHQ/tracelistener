package exporter

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestValidate_Id(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		requireErr bool
		err        error
	}{
		{
			"Len out of limit",
			"12345678901",
			true,
			NewValidationError(fmt.Errorf("accepted max id len 10 received %d", len("12345678901"))),
		},
		{
			"Id contains non alpha numeric value - 1",
			"af-af",
			true,
			NewValidationError(fmt.Errorf("accepted characters a-z, A-Z and 0-9, received %s", "af-af")),
		},
		{
			"Id contains non alpha numeric value - 2",
			"afaf®",
			true,
			NewValidationError(fmt.Errorf("accepted characters a-z, A-Z and 0-9, received %s", "afaf®")),
		},
		{
			"Valid Id - 1",
			"1234567890",
			false,
			nil,
		},
		{
			"Valid Id - 2",
			"a1B2cc33XX",
			false,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := setUpParams(t, 0, 0, tt.id, 0)
			err := validateFileId(&p)
			if tt.requireErr {
				require.Equal(t, tt.err, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidate_ParamCombination(t *testing.T) {
	tests := []struct {
		name      string
		recordLim int32
		sizeLim   int32
		duration  time.Duration
		err       error
	}{
		{
			"ok - all three valid params present",
			int32(10),
			int32(20),
			time.Second,
			nil,
		},
		{
			"ok - two valid params present",
			int32(0),
			int32(20),
			time.Second,
			nil,
		},
		{
			"ok - one valid params present",
			int32(0),
			int32(0),
			5 * time.Second,
			nil,
		},
		{
			"no valid params present",
			int32(0),
			int32(0),
			0,
			NewValidationError(fmt.Errorf("invalid param combination")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := setUpParams(t, tt.recordLim, tt.sizeLim, "", tt.duration)
			err := ValidateParamCombination(&p)
			require.Equal(t, tt.err, err)
		})
	}
}

func setUpParams(t *testing.T, n, s int32, id string, d time.Duration) Params {
	t.Helper()
	return Params{
		NumTraces: n,
		SizeLim:   s,
		Duration:  d,
		Upload:    false,
		FileId:    id,
	}
}
