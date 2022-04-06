package exporter_test

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/emerishq/tracelistener/exporter"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
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
		_, err = ex.Stop(false, doOnce)
		require.NoError(t, err)
		require.False(t, ex.IsRunning())
		require.False(t, ex.AcceptingData())
	}

	t.Cleanup(func() {
		require.NoError(t, os.Remove(ex.LocalFile.Name()))
	})
}

func TestStart_AcceptXXXRecords(t *testing.T) {
	XXX := int32(10)
	p, err := setUpParams(t, XXX, 100, "XXXRecords", 100*time.Minute, false)
	require.NoError(t, err)

	ex, err := exporter.New(&p)
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

	f, err := os.Open(ex.LocalFile.Name())
	require.NoError(t, err)

	asByte, err := ioutil.ReadFile(f.Name())
	require.NoError(t, err)
	parts := bytes.Fields(asByte)

	t.Log("Parts", parts)
	// Parts come from the file, records is the original raw data. Must match.
	for i, r := range records[:XXX] {
		require.True(t, bytes.Equal(parts[i], r))
	}

	t.Cleanup(func() {
		_ = f.Close() // Not needed, OCD kick.
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

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = source.Close()
	}()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = destination.Close()
	}()

	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func linesFromFile(f *os.File) ([][]byte, error) {
	scanner := bufio.NewScanner(f)
	var ret [][]byte
	for scanner.Scan() {
		ret = append(ret, scanner.Bytes())
		fmt.Println("Last ret", ret[len(ret)-1])
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}
