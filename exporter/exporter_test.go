package exporter_test

import (
	"github.com/emerishq/tracelistener/exporter"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	p, err := setUpParams(t, 10, 100, "xxxx", 10*time.Minute, false)
	require.NoError(t, err)

	// Build the exporter.
	ex, err := exporter.New(&p)
	require.NoError(t, err)

	// Exporter should not be running, and not yet accepting records.
	require.False(t, ex.IsRunning())
	require.False(t, ex.AcceptingData())

	_, doOnce, errCh := ex.Start()
	require.True(t, ex.IsRunning())
	require.True(t, ex.AcceptingData())

	// Only one running process allowed
	_, _, errCh = ex.Start()
	require.ErrorIs(t, <-errCh, exporter.ErrExporterRunning)

	t.Log("local file name:", ex.LocalFile.Name())

	// Stop should be idempotent.
	for i := 0; i < 10; i++ {
		_, err = ex.Stop(false, doOnce, errCh)
		require.NoError(t, err)
		require.False(t, ex.IsRunning())
		require.False(t, ex.AcceptingData())
	}

	t.Cleanup(func() {
		require.NoError(t, os.Remove(ex.LocalFile.Name()))
	})
}

func setUpParams(t *testing.T, n, s int32, id string, d time.Duration, p bool) (exporter.Params, error) {
	t.Helper()
	return exporter.Params{
		RecordLim: n,
		SizeLim:   s,
		Duration:  d,
		Persis:    p,
		FileId:    id,
	}, nil
}
