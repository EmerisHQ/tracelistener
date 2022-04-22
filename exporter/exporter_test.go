package exporter_test

import (
	"bytes"
	"errors"
	"github.com/emerishq/emeris-utils/logging"
	"github.com/emerishq/tracelistener/exporter"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"sync"
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

	_ = ex.StartReceiving()
	require.True(t, ex.IsRunning())
	require.True(t, ex.IsAcceptingData())

	// Only one running process allowed
	require.ErrorIs(t, <-ex.StartReceiving(), exporter.ErrExporterRunning)

	// StopReceiving must be idempotent
	for i := 0; i < 10; i++ {
		require.NoError(t, ex.StopReceiving())
	}

	require.Eventually(t, func() bool {
		return !ex.IsRunning()
	}, time.Second*4, time.Millisecond*100)
}

func TestStart_AcceptXXXRecords(t *testing.T) {
	XXX := int32(5)
	params, err := setUpParams(t, XXX, 100, "XXXRecords", 100*time.Minute, false)
	require.NoError(t, err)

	ex, err := exporter.New(exporter.WithLogger(logging.New(logging.LoggingConfig{Debug: true})))
	require.NoError(t, err)
	// Build the exporter.
	err = ex.Init(&params)
	require.NoError(t, err)

	errCh := ex.StartReceiving()

	records := [][]byte{[]byte("go is"), []byte("short but"), []byte("handle the error"), []byte("java is"), []byte("dark and"), []byte("full of terror")}

	// After XXX records, no more processed.
	for i, record := range records {
		err = ex.UnblockedReceive(record)
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
	parts := bytes.Split(asByte, []byte("\n"))

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
	// Init the exporter.
	require.NoError(t, ex.Init(&params))

	errCh := ex.StartReceiving()

	dataInserterDone := make(chan struct{})
	dataInsertInterval := 300 * time.Millisecond
	var wg sync.WaitGroup
	wg.Add(1)
	// Simulate: traces capture until stopped.
	go func(t *testing.T, selfDone chan struct{}, interval time.Duration, wg *sync.WaitGroup) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer wg.Done()
		for {
			select {
			case <-ticker.C:
				err := ex.UnblockedReceive([]byte{66, 66, 66, 66}) // Simulate trace capture
				require.NoError(t, err)
			case <-selfDone:
				return
			}
		}
	}(t, dataInserterDone, dataInsertInterval, &wg)

	// Ensure at least some data get accepted.
	time.Sleep(3 * dataInsertInterval)

	// 1. Simulate: no more data. (Being explicit for reader's ease).
	close(dataInserterDone)
	wg.Wait()
	// 2. User called stop.
	require.NoError(t, ex.StopReceiving())
	//_, err = ex.Stop(false, doOnce, false)
	//require.NoError(t, err)
	require.NoError(t, <-errCh)

	// Check exporter.finish() was called.
	require.Eventually(t, func() bool {
		return !ex.IsRunning()
	}, time.Second*4, dataInsertInterval)

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

	errCh := ex.StartReceiving()

	dataInserterDone := make(chan struct{})
	dataInsertInterval := 200 * time.Millisecond
	// Simulate: traces capture until stopped.
	go func(t *testing.T, selfDone chan struct{}, interval time.Duration) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := ex.UnblockedReceive([]byte{33, 44, 55, 66})
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
	}(t, dataInserterDone, dataInsertInterval)

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
		Upload:    p,
		FileId:    id,
	}, nil
}
