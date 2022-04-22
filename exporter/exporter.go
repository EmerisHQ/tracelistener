package exporter

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

type (
	// Stat used to report progress to the caller.
	Stat struct {
		StartTime     time.Time
		RunningTime   time.Duration
		TotalSize     int32
		TraceCount    int32
		LocalFile     *os.File
		RunningStatus string
		Errors        []error
	}

	// Params represents the acceptable http request params.
	Params struct {
		NumTraces int32
		SizeLim   int32
		Duration  time.Duration
		Upload    bool
		Clean     bool
		FileId    string
	}

	Exporter struct {
		running   bool
		muRunning sync.Mutex
		muStat    sync.Mutex
		params    *Params
		Stat      *Stat
		logger    *zap.SugaredLogger

		traceChan chan []byte
		doneChan  chan struct{}
		doOnce    func(func())
	}

	Option func(*Exporter) error
)

var (
	// ErrExporterRunning is used when we try to run the exporter, but another exporting is underway.
	ErrExporterRunning = errors.New("exporter: already running")
	// ErrExporterNotRunning is used when we try to stop an exporter, but there is no running exporting task.
	ErrExporterNotRunning = errors.New("exporter: not running")
	// ErrNotAcceptingData is used when exporter is running but, not receiving data anymore. Usually used
	// when we close the done channel, but the export process is still running. (Maybe uploading file for ex)
	ErrNotAcceptingData = errors.New("exporter: not accepting data")
	// ErrShouldNotAcceptData is used when exporter should not accept data, but accepting anyway.
	// Indicates an inconsistent state.
	ErrShouldNotAcceptData = errors.New("exporter: should not accepting data")
)

const (
	Million       = 1_000_000
	MaxSizeLim    = 1024
	MaxTraceCount = Million
	MaxDuration   = 24 * time.Hour
)

func WithLogger(l *zap.SugaredLogger) Option {
	return func(e *Exporter) error {
		if l == nil {
			return fmt.Errorf("logger can not be nil")
		}
		e.logger = l
		return nil
	}
}

func New(opts ...Option) (*Exporter, error) {
	e := &Exporter{
		muRunning: sync.Mutex{},
		muStat:    sync.Mutex{},
		logger:    zap.NewNop().Sugar(),
		running:   false, // Being explicit!
	}
	for _, o := range opts {
		if err := o(e); err != nil {
			return nil, err
		}
	}
	return e, nil
}

// Init takes a Params as input, then
// 1. Validate the Params. Returns ValidationError if failed.
// 2. Creates the local file to hold the traces
// 3. Updates the Exporter in place
func (e *Exporter) Init(params *Params) error {
	if e.IsRunning() {
		return ErrExporterRunning
	}
	if err := runParamValidators(
		params,
		validateSizeLim,          // 0 <= size <= MaxSizeLim.
		validateNumTrace,         // 0 <= trace count <= MaxTraceCount.
		validateDuration,         // 0 <= duration <= 24 hours.
		validateFileId,           // len(id) <= 10; only alphanumeric.
		ValidateParamCombination, // At least one valid param present.
	); err != nil {
		return err
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
		return err
	}
	localFile, err := os.OpenFile(f.Name(), os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	e.params = params
	e.Stat = &Stat{
		StartTime:     startTime,
		RunningTime:   0,
		TotalSize:     0,
		TraceCount:    0,
		LocalFile:     localFile,
		RunningStatus: "Init",
		Errors:        make([]error, 0),
	}

	if e.params.NumTraces == 0 {
		e.params.NumTraces = MaxTraceCount
	}
	e.params.SizeLim *= Million // Convert to byte
	if e.params.SizeLim == 0 {
		e.params.SizeLim = MaxSizeLim * Million // 1024 * 1_000_000; 1 GB converted to bytes
	}
	if e.params.Duration == 0 {
		e.params.Duration = MaxDuration
	}

	e.doOnce = (&sync.Once{}).Do
	e.doneChan = nil // Indicates that exporter is initialised, but not yet ready to receive traces.
	e.traceChan = nil

	return nil
}

func (e *Exporter) StartReceiving() chan error {
	errChan := make(chan error, 1)

	if e.IsRunning() {
		errChan <- ErrExporterRunning
		return errChan
	}

	if err := e.setRunning(true); err != nil {
		errChan <- err
		return errChan
	}

	e.muStat.Lock()
	e.Stat.RunningStatus = "Receiving traces"
	e.muStat.Unlock()

	e.traceChan = make(chan []byte, e.params.NumTraces)
	e.doneChan = make(chan struct{})

	go func(errCh chan error) {
		errCh <- e.Orchestrate()
	}(errChan)

	return errChan
}

// StopReceiving is idempotent, it can be called multiple times. It's used in
// 1. UnblockedReceive: we've reached limit fot NumTraces or SizeLim.
// 2. When user calls stop from rest endpoint.
func (e *Exporter) StopReceiving() error {
	if e.doOnce == nil {
		return ErrExporterNotRunning
	}
	e.doOnce(func() {
		close(e.doneChan)
		e.muStat.Lock()
		e.Stat.RunningStatus = "Stopped receiving traces, processing remaining traces"
		e.muStat.Unlock()
	})
	return nil
}

func (e *Exporter) Orchestrate() error {
	if err := e.HandleTrace(); err != nil {
		return err
	}
	// e.HandleTrace returned with no error. That means e.StopReceiving was called.
	// Now we finish the exporter i.e. upload file to cloud, cleanup etc.
	if err := e.finish(); err != nil {
		return err
	}

	return nil
}

// finish the trace exporting process. This method must be called once.
// multiple calls to this method indicates logical error in code.
// 1. Uploads file to cloud if necessary. <- Not yet; proposed.
// 2. Closes the local file descriptor.
// 3. Removed the local file if necessary.
// 4. sets state for running to false.
func (e *Exporter) finish() error {
	if !e.IsRunning() {
		return ErrExporterNotRunning
	}

	if e.IsAcceptingData() {
		return ErrShouldNotAcceptData
	}

	if e.params.Upload {
		// TODO
		// 1. Upload to google
		// 2. Generate slack msg
		// 3. Delete local file
		e.logger.Debugw("Upload", e.params.Upload)
	}

	if err := e.Stat.LocalFile.Close(); err != nil {
		return err
	}

	if e.params.Clean {
		if _, err := os.Stat(e.Stat.LocalFile.Name()); !errors.Is(err, fs.ErrNotExist) {
			if err2 := os.Remove(e.Stat.LocalFile.Name()); err2 != nil {
				e.logger.Errorw("finish: could not remove",
					"file", e.Stat.LocalFile.Name(),
					"error", err2.Error())
			}
			e.Stat.LocalFile = nil
		}
	}

	if err := e.setRunning(false); err != nil {
		return err
	}
	e.muStat.Lock()
	e.Stat.RunningStatus = "Finished"
	e.Stat.RunningTime = time.Since(e.Stat.StartTime).Round(0)
	e.muStat.Unlock()
	return nil
}

func (e *Exporter) UnblockedReceive(trace []byte) error {
	if !e.IsAcceptingData() {
		return ErrNotAcceptingData
	}
	select {
	case e.traceChan <- trace:
		e.Stat.TraceCount++
		e.Stat.TotalSize += int32(len(trace))
		// Stop export process if condition reached.
		if e.reachedTraceCount() || e.reachedSizeLimit() ||
			time.Since(e.Stat.StartTime).Round(0) >= e.params.Duration {
			if err := e.StopReceiving(); err != nil {
				return err
			}
		}
	default:
		e.logger.Errorw("UnblockedReceive:", "blocked on chan; trace", trace)
		return fmt.Errorf("blocked on sending data to trace channel")
	}
	return nil
}

func (e *Exporter) HandleTrace() error {
	for {
		select {
		case <-e.doneChan:
			// Drain the traceChan.
			for {
				select {
				case r := <-e.traceChan:
					if n, err := e.Stat.LocalFile.Write(append(r, []byte("\n")...)); err != nil {
						e.logger.Errorw("HandleTrace: failed to write", "size", n, "bytes", r)
						return fmt.Errorf("handleTrace: could not write to file, error: %w", err)
					}
				default:
					return nil
				}
			}
		case r := <-e.traceChan:
			if n, err := e.Stat.LocalFile.Write(append(r, []byte("\n")...)); err != nil {
				e.logger.Errorw("HandleTrace: failed to write", "size", n, "bytes", r)
				return fmt.Errorf("handleTrace: could not write to file, error: %w", err)
			}
		}
	}
}

// IsAcceptingData checks if exporter is running i.e. traceChan
// is not nil AND done channel is not closed.
func (e *Exporter) IsAcceptingData() bool {
	if e.traceChan == nil || e.doneChan == nil {
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

func (e *Exporter) setRunning(newStatus bool) error {
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

func (e *Exporter) GetStat() (Stat, error) {
	e.muStat.Lock()
	defer e.muStat.Unlock()
	if e.Stat == nil {
		return Stat{}, ErrExporterNotRunning
	}
	if e.IsRunning() {
		e.Stat.RunningTime = time.Since(e.Stat.StartTime).Round(0)
	}
	return *e.Stat, nil
}

func (e *Exporter) reachedTraceCount() bool {
	if e.params.NumTraces <= 0 {
		return false
	}
	return e.Stat.TraceCount >= e.params.NumTraces
}

func (e *Exporter) reachedSizeLimit() bool {
	if e.params.SizeLim <= 0 {
		return false
	}
	return e.Stat.TotalSize >= e.params.SizeLim
}

func (s Stat) Public() any {
	errMsg := make([]string, 0, len(s.Errors))
	for _, e := range s.Errors {
		if e == nil {
			errMsg = append(errMsg, "no error")
			continue
		}
		errMsg = append(errMsg, e.Error())
	}
	fileName := "-- Removed --"
	if s.LocalFile != nil {
		fileName = s.LocalFile.Name()
	}
	return struct {
		StartTime  time.Time `json:"start_time"`
		Runtime    string    `json:"runtime"`
		TotalSize  string    `json:"total_size"`
		TraceCount int32     `json:"trace_count"`
		LocalFile  string    `json:"file_name"`
		Status     string    `json:"status"`
		Errors     []string  `json:"errors,omitempty"`
	}{
		s.StartTime,
		s.RunningTime.String(),
		fmt.Sprintf("%0.8f MB", float32(s.TotalSize)/float32(Million)),
		s.TraceCount,
		fileName,
		s.RunningStatus,
		errMsg,
	}
}

func createFile(t time.Time, n, s int32, d time.Duration, id string) (*os.File, error) {
	fileNameParts := []string{t.Format("2006-01-02-15:04:05")}
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
