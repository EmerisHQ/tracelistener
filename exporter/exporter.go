package exporter

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

type (
	// Stat used to report progress to the caller.
	Stat struct {
		StartTime   time.Time
		RunningTime time.Duration
		TotalSize   int32
		TraceCount  int32
		Err         error
	}

	// Params represents the acceptable http request params.
	Params struct {
		NumTraces int32
		SizeLim   int32
		Duration  time.Duration
		Persis    bool
		FileId    string
	}

	Exporter struct {
		running   bool
		muRunning sync.Mutex
		// Full name in pod.
		LocalFile *os.File
		params    Params
		stat      Stat

		traceChan chan []byte
		doneChan  chan struct{}

		logger *zap.SugaredLogger
	}
)

var (
	// ErrExporterRunning is used when we try to run the exporter, but another exporting is underway.
	ErrExporterRunning = errors.New("exporter: running")
	// ErrExporterNotRunning is used when we try to stop an exporter, but there is no running exporting task.
	ErrExporterNotRunning = errors.New("exporter: not running")
	// ErrNotAcceptingData is used when exporter is running but, not receiving data anymore. Usually used
	// when we close the done channel, but the export process is still running. (Maybe uploading file for ex)
	ErrNotAcceptingData = errors.New("exporter: not accepting data")
)

const (
	MaxSizeLim    = 1024
	MaxTraceCount = 1000000
	MaxDuration   = 24 * time.Hour
)

// Init takes a params as input, then
// 1. Validate the params. Returns ValidationError if failed.
// 2. Creates the local file to hold the traces
// 3. Builds and returns a pointer to the Exporter
func Init(params *Params, l *zap.SugaredLogger) (*Exporter, error) {
	err := runParamValidators(
		params,
		validateSizeLim,          // 0 <= size <= MaxSizeLim.
		validateNumTrace,         // 0 <= trace count <= MaxTraceCount.
		validateDuration,         // 0 <= duration <= 24 hours.
		validateFileId,           // len(id) <= 10; only alphanumeric.
		ValidateParamCombination, // At least one valid param present.
	)
	if err != nil {
		return nil, err
	}

	startTime := time.Now().Round(0)

	// fileName = 01-02-2006-15:04:05-1000N-100MB-1h20m0s-fileId-03118414
	// `fileId` comes from user. Useful if the user wants to insert (max len 10)
	// an identifier in the file name. Ex: JunoProd, irisDev, atomStag
	// 03118414 is a random id inserted by the library.
	//
	// 1000N-100MB-1h20m0s-fileId these are optional fields. Not included in the
	// file name if empty.
	f, err := createFile(startTime, params.NumTraces, params.SizeLim, params.Duration, params.FileId)
	if err != nil {
		return nil, err
	}
	localFile, err := os.OpenFile(f.Name(), os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	e := &Exporter{
		running:   false,
		muRunning: sync.Mutex{},
		LocalFile: localFile,
		params:    *params,
		stat: Stat{
			StartTime: startTime,
			Err:       nil,
		},
		traceChan: nil, // Initialised when start is called.
		doneChan:  nil, // ditto
		logger:    l,
	}
	return e, nil
}

func (e *Exporter) Start() (interface{}, func(func()), chan error) {
	errChan := make(chan error, 1)

	if e.IsRunning() {
		errChan <- ErrExporterRunning
		return nil, nil, errChan
	}

	err := e.SetRunning(true)
	if err != nil {
		errChan <- err
		return nil, nil, errChan
	}

	traceChCap := int32(MaxTraceCount)
	if e.params.NumTraces > 0 {
		traceChCap = e.params.NumTraces
	}
	e.traceChan = make(chan []byte, traceChCap)
	e.doneChan = make(chan struct{})

	doOnce := (&sync.Once{}).Do

	go func(errCh chan error) {
		errCh <- e.Orchestrate(doOnce)
	}(errChan)

	if e.params.Duration > 0 {
		time.AfterFunc(e.params.Duration, func() {
			e.StopReceiving(doOnce)
		})
	}
	// go e.WatchStorage(e.doneChan, 1000) // TODO: implement later

	return nil, doOnce, errChan
}

// Stop stops the trace exporting process. This method must be called once.
func (e *Exporter) Stop(persistOverride bool, doOnce func(func()), forceClean bool) (interface{}, error) {
	if !e.IsRunning() {
		return nil, ErrExporterNotRunning
	}

	if e.IsAcceptingData() {
		e.StopReceiving(doOnce)
	}

	err := e.SetRunning(false)
	if err != nil {
		return nil, err
	}

	if (e.params.Persis || persistOverride) && e.IsRunning() {
		// TODO
		// 1. Upload to google
		// 2. Generate slack msg
		// 3. Delete local file
		e.logger.Debugw("Persist", persistOverride)
	}
	// TODO: Process user report

	e.stat.RunningTime = time.Since(e.stat.StartTime).Round(0)
	if forceClean {
		if _, err := os.Stat(e.LocalFile.Name()); !os.IsNotExist(err) {
			err2 := os.Remove(e.LocalFile.Name())
			if err2 != nil {
				return nil, err2
			}
		}
	}

	return nil, nil
}

func (e *Exporter) Orchestrate(doOnce func(func())) error {
	err := e.HandleTrace()
	if err != nil {
		return err
	}

	// Exporter is still running. i.e. not forced stopped by user.
	// So we stop it as e.HandleTrace has stored all the traces to file.
	if e.IsRunning() {
		_, err = e.Stop(e.params.Persis, doOnce, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Exporter) UnblockedReceive(trace []byte, doOnce func(func())) error {
	if !e.IsAcceptingData() {
		return ErrNotAcceptingData
	}
	select {
	case e.traceChan <- trace:
		e.logger.Debugw("UnblockedReceive:", "e.traceChan <- trace:", trace)
		e.stat.TraceCount++
		e.stat.TotalSize += int32(len(trace))
		// Stop export process if condition reached.
		if e.reachedTraceCount() || e.reachedSizeLimit() {
			e.logger.Debugw("UnblockedReceive:", "e.reachedTraceCount calling StopReceiving")
			e.StopReceiving(doOnce)
		}
	default:
		e.logger.Debugw("UnblockedReceive:", "blocked on chan; trace", trace)
		return fmt.Errorf("blocked on sending data to trace channel")
	}
	return nil
}

func (e *Exporter) HandleTrace() error {
	for {
		select {
		case <-e.doneChan:
			// Drain the traceChan.
			for r := range e.traceChan {
				e.logger.Debugw("HandleTrace:", "draining: r", r)
				n, err := e.LocalFile.Write(append(r, []byte("\n")...))
				if err != nil {
					return fmt.Errorf("handleTrace: could not write to file: %s, error: %w", e.LocalFile.Name(), err)
				}
				e.logger.Debugw("HandleTrace:", "Write size", n, "bytes", r)
			}
			return nil
		case r, ok := <-e.traceChan:
			e.logger.Debugw("HandleTrace:", "from <-e.traceChan", r, "ok", ok)
			if !ok {
				return nil
			}
			n, err := e.LocalFile.Write(append(r, []byte("\n")...))
			if err != nil {
				return fmt.Errorf("handleTrace: could not write to file: %s, error: %w", e.LocalFile.Name(), err)
			}
			e.logger.Debugw("HandleTrace:", "Wrote size", n, "bytes", r)
		}
	}
}

func (e *Exporter) StopReceiving(doOnce func(func())) {
	doOnce(func() {
		close(e.traceChan)
		close(e.doneChan)
	})
}

// IsAcceptingData checks if exporter is running i.e. traceChan
// is not nil AND done channel is not closed.
func (e *Exporter) IsAcceptingData() bool {
	if e.traceChan == nil {
		return false
	}
	select {
	default:
		return true
	case <-e.doneChan:
	}
	return false
}

func (e *Exporter) IsRunning() bool {
	e.muRunning.Lock()
	r := e.running
	e.muRunning.Unlock()
	return r
}

func (e *Exporter) SetRunning(newStatus bool) error {
	curStatus := e.IsRunning()
	if curStatus == newStatus {
		if curStatus {
			return ErrExporterRunning
		}
		return ErrExporterNotRunning
	}
	e.muRunning.Lock()
	e.running = newStatus
	e.muRunning.Unlock()
	return nil
}

func (e *Exporter) GetTraceChan() (chan []byte, error) {
	if !e.IsRunning() {
		return nil, ErrExporterNotRunning
	}
	return e.traceChan, nil
}

func (e *Exporter) GetDoneChan() (chan struct{}, error) {
	if !e.IsRunning() {
		return nil, ErrExporterNotRunning
	}
	return e.doneChan, nil
}

func (e *Exporter) reachedTraceCount() bool {
	if e.params.NumTraces <= 0 {
		return false
	}
	return e.stat.TraceCount >= e.params.NumTraces
}

func (e *Exporter) reachedSizeLimit() bool {
	if e.params.SizeLim <= 0 {
		return false
	}
	return e.stat.TotalSize >= e.params.SizeLim
}

func createFile(t time.Time, n, s int32, d time.Duration, id string) (*os.File, error) {
	fileNameParts := []string{t.Format("01-02-2006-15:04:05")}
	if n > 0 { // Number of traces.
		fileNameParts = append(fileNameParts, fmt.Sprintf("%dN", n))
	}
	if s > 0 { // Size of the file.
		fileNameParts = append(fileNameParts, fmt.Sprintf("%dMB", s))
	}
	if d > 0 {
		fileNameParts = append(fileNameParts, d.String())
	}
	if id != "" {
		fileNameParts = append(fileNameParts, id)
	}
	fileName := strings.Join(append(fileNameParts, "-*.txt"), "-")
	return ioutil.TempFile("", fileName)
}
