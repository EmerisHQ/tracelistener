package exporter_test

import (
	"bytes"
	"errors"
	"github.com/emerishq/emeris-utils/logging"
	"github.com/emerishq/tracelistener/exporter"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	params, err := setUpParams(t, 10, 100, "xxxx", 10*time.Minute, false)
	require.NoError(t, err)

	ex, err := exporter.New(exporter.WithLogger(logging.New(logging.LoggingConfig{Debug: true})))
	require.NoError(t, err)
	// Build the exporter.
	err = ex.Init(&params)
	require.NoError(t, err)

	// Exporter should not be running, and not yet accepting records.
	require.False(t, ex.IsRunning())
	require.False(t, ex.IsAcceptingData())

	_, doOnce, errCh := ex.Start()
	require.True(t, ex.IsRunning())
	require.True(t, ex.IsAcceptingData())

	// Only one running process allowed
	_, _, errCh = ex.Start()
	require.ErrorIs(t, <-errCh, exporter.ErrExporterRunning)

	// StopReceiving must be idempotent
	for i := 0; i < 10; i++ {
		ex.StopReceiving(doOnce)
	}

	require.Eventually(t, func() bool {
		return !ex.IsRunning()
	}, time.Second*4, time.Millisecond*100)
}

func TestStart_AcceptXXXRecords(t *testing.T) {
	XXX := int32(10)
	params, err := setUpParams(t, XXX, 100, "XXXRecords", 100*time.Minute, false)
	require.NoError(t, err)

	ex, err := exporter.New(exporter.WithLogger(logging.New(logging.LoggingConfig{Debug: true})))
	require.NoError(t, err)
	// Build the exporter.
	err = ex.Init(&params)
	require.NoError(t, err)

	_, doOnce, errCh := ex.Start()

	records := [][]byte{{14, 14}, {24, 24}, {34, 34}, {44, 44}, {54, 54}, {64, 64}, {74, 74}, {84, 84}, {94, 94}, {104, 104}, {114, 114}, {124, 124}}

	// After XXX records, no more processed.
	for i, record := range records {
		err = ex.UnblockedReceive(record, doOnce)
		if i < int(XXX) {
			require.NoError(t, err)
			continue
		}
		require.ErrorIs(t, err, exporter.ErrNotAcceptingData)
	}

	require.NoError(t, <-errCh)

	f, err := os.Open(ex.Stat.LocalFile.Name())
	require.NoError(t, err)

	asByte, err := ioutil.ReadFile(f.Name())
	require.NoError(t, err)
	parts := bytes.Fields(asByte)

	// Parts come from the file, records is the original raw data. Must match.
	for i, r := range records[:XXX] {
		require.True(t, bytes.Equal(parts[i], r))
	}

	t.Cleanup(func() {
		_ = f.Close() // Not needed, OCD kick.
		require.NoError(t, os.Remove(ex.Stat.LocalFile.Name()))
	})
}

func TestExporter_User_Called_Stop(t *testing.T) {
	params, err := setUpParams(t, 0, 0, "1hr", 1*time.Hour, false)
	require.NoError(t, err)

	ex, err := exporter.New(exporter.WithLogger(logging.New(logging.LoggingConfig{Debug: true})))
	require.NoError(t, err)
	// Build the exporter.
	err = ex.Init(&params)
	require.NoError(t, err)

	_, doOnce, errCh := ex.Start()

	dataInserterDone := make(chan struct{})
	dataInsertInterval := 300 * time.Millisecond

	go func(t *testing.T, doOnce func(func()), selfDone chan struct{}, interval time.Duration) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := ex.UnblockedReceive([]byte{33, 44, 55, 66}, doOnce)
				require.NoError(t, err)
			case <-selfDone:
				return
			}
		}
	}(t, doOnce, dataInserterDone, dataInsertInterval)

	// Ensure at least some data get accepted.
	time.Sleep(3 * dataInsertInterval)

	// 1. Simulate: no more data.
	close(dataInserterDone)
	// 2. User called stop.
	ex.StopReceiving(doOnce)
	//_, err = ex.Stop(false, doOnce, false)
	//require.NoError(t, err)
	require.NoError(t, <-errCh)

	// Check exporter.finish() was called.
	require.Eventually(t, func() bool {
		return !ex.IsRunning()
	}, time.Second*4, time.Millisecond*100)

	// Ensure something is written to the file.
	f, err := os.Open(ex.Stat.LocalFile.Name())
	require.NoError(t, err)
	asByte, err := ioutil.ReadFile(f.Name())
	require.NoError(t, err)
	parts := bytes.Fields(asByte)
	require.NotEmpty(t, parts)

	t.Cleanup(func() {
		_ = f.Close() // Not needed, OCD kick.
		require.NoError(t, os.Remove(ex.Stat.LocalFile.Name()))
	})
}

func TestExporter_DurationExpired(t *testing.T) {
	params, err := setUpParams(t, 0, 0, "2Second", 2*time.Second, false)
	require.NoError(t, err)

	ex, err := exporter.New(exporter.WithLogger(logging.New(logging.LoggingConfig{Debug: true})))
	require.NoError(t, err)
	// Build the exporter.
	err = ex.Init(&params)
	require.NoError(t, err)

	_, doOnce, errCh := ex.Start()

	dataInserterDone := make(chan struct{})
	dataInsertInterval := 200 * time.Millisecond

	go func(t *testing.T, doOnce func(func()), selfDone chan struct{}, interval time.Duration) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := ex.UnblockedReceive([]byte{33, 44, 55, 66}, doOnce)
				require.Condition(t, func() bool {
					if err == nil {
						return true
					}
					return errors.Is(err, exporter.ErrNotAcceptingData)
				})
			case <-selfDone:
				return
			}
		}
	}(t, doOnce, dataInserterDone, dataInsertInterval)

	// Ensure deadline expired.
	require.Eventually(t, func() bool {
		select {
		case err := <-errCh:
			require.NoError(t, err)
			require.False(t, ex.IsRunning())
			require.False(t, ex.IsAcceptingData())
			return true
		default:
			return false
		}
	}, 4*time.Second, dataInsertInterval)

	// Ensure something is written to the file.
	f, err := os.Open(ex.Stat.LocalFile.Name())
	require.NoError(t, err)
	asByte, err := ioutil.ReadFile(f.Name())
	require.NoError(t, err)
	parts := bytes.Fields(asByte)
	require.NotEmpty(t, parts)

	t.Cleanup(func() {
		_ = f.Close() // Not needed, OCD kick.
		require.NoError(t, os.Remove(ex.Stat.LocalFile.Name()))
	})
}

func setUpParams(t *testing.T, n, s int32, id string, d time.Duration, p bool) (exporter.Params, error) {
	t.Helper()
	return exporter.Params{
		NumTraces: n,
		SizeLim:   s,
		Duration:  d,
		Persis:    p,
		FileId:    id,
	}, nil
}
