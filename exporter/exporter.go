package exporter

import (
	"errors"
	"fmt"
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
		NumRecords  int32
		Err         error
	}

	// Params represents the acceptable http request params.
	Params struct {
		RecordLim int32
		SizeLim   int32
		Duration  time.Duration
		Persis    bool
		FileId    string
	}

	Exporter struct {
		running   bool
		muRunning sync.Mutex
		// Full name in pod.
		localFile *os.File
		params    Params
		stat      Stat

		recordChan chan []byte
		doneChan   chan struct{}
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
	MaxSizeLim   = 1024
	MaxRecordLim = 1000000
	MaxDuration  = 24 * time.Hour
)

// New takes a params as input, then
// 1. Validate the params. Returns ValidationError if failed.
// 2. Creates the local file to hold the records
// 3. Builds and returns a pointer to the Exporter
func New(params *Params) (*Exporter, error) {
	err := runParamValidators(
		params,
		validateSizeLim,          // 0 <= size <= MaxSizeLim.
		validateRecordCount,      // 0 <= record count <= MaxRecordLim.
		validateDuration,         // 0 <= duration <= 24 hours.
		validateFileId,           // len(id) <= 10; only alphanumeric.
		ValidateParamCombination) // At least one valid param present.
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	recordChCap := int32(MaxRecordLim)
	if params.RecordLim > 0 {
		recordChCap = params.RecordLim
	}

	// fileName = 01-02-2006 15:04:05-100MB-1h20m0s-fileId-03118414
	// `fileId` comes from user. Useful if the user wants to insert (max len 10)
	// an identifier in the file name.
	// 03118414 is a random id inserted by the library.
	f, err := createFile(startTime, params.RecordLim, params.SizeLim, params.Duration, params.FileId)
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
		localFile: localFile,
		params:    *params,
		stat: Stat{
			StartTime:   startTime,
			RunningTime: 0,
			TotalSize:   0,
			NumRecords:  0,
			Err:         nil,
		},
		recordChan: make(chan []byte, recordChCap),
		doneChan:   make(chan struct{}),
	}
	return e, nil
}

func (e *Exporter) AcceptingData() bool {
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

func (e *Exporter) GetRecordChan() (chan []byte, error) {
	if !e.IsRunning() {
		return nil, ErrExporterNotRunning
	}
	return e.recordChan, nil
}

func (e *Exporter) GetDoneChan() (chan struct{}, error) {
	if !e.IsRunning() {
		return nil, ErrExporterNotRunning
	}
	return e.doneChan, nil
}

func (e *Exporter) UnblockedReceive(record []byte) error {
	if !e.AcceptingData() {
		return ErrNotAcceptingData
	}
	select {
	case e.recordChan <- record:
	default:
		return fmt.Errorf("blocked on sending data to record channel")
	}
	return nil
}

func (e *Exporter) Start() (interface{}, func(func()), chan error) {
	errChan := make(chan error, 1)
	defer close(errChan)

	if e.IsRunning() {
		errChan <- ErrExporterRunning
		return nil, nil, errChan
	}

	err := e.SetRunning(true)
	if err != nil {
		errChan <- err
		return nil, nil, errChan
	}

	doOnce := (&sync.Once{}).Do

	if e.params.Duration > 0 {
		time.AfterFunc(e.params.Duration, func() {
			report, err := e.Stop(e.params.Persis, doOnce)
			if err != nil {
				// TODO: Handle error
				fmt.Println(err)
			}
			// TODO: e.Stat(report)
			fmt.Println(report)
		})
	}
	// go e.WatchStorage(e.doneChan, 1000) // TODO: implement later

	go func(errCh chan error) {
		err := e.HandleRecord(doOnce)
		if err != nil {
			return
		}
	}(errChan)

	return nil, doOnce, errChan
}

// Stop stops the record exporting process. This is an idempotent method. Calling
// it multiple times should not have any adversary effect.
func (e *Exporter) Stop(persist bool, doOnce func(func())) (interface{}, error) {
	if !e.IsRunning() {
		return nil, ErrExporterNotRunning
	}

	if e.params.Persis || persist {
		// TODO
		// 1. Upload to google
		// 2. Generate slack msg
		// 3. Delete local file
		fmt.Println(persist)
	}
	// TODO: Process user report

	doOnce(func() {
		close(e.recordChan)
		close(e.doneChan)
	})

	// Drain the recordChan.
	for r := range e.recordChan {
		_, err := e.localFile.Write(r)
		if err != nil {
			return nil,
				fmt.Errorf("stop: could not write to file: %s, error: %w", e.localFile.Name(), err)
		}
	}

	err := e.SetRunning(false)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (e *Exporter) HandleRecord(doOnce func(func())) error {
	for {
		select {
		case <-e.doneChan:
			return fmt.Errorf("handleRecord: stop called")
		case r, ok := <-e.recordChan:
			if !ok {
				return fmt.Errorf("handleRecord: record chan drained")
			}
			n, err := e.localFile.Write(r)
			if err != nil {
				return fmt.Errorf("handleRecord: could not write to file: %s, error: %w", e.localFile.Name(), err)
			}
			e.stat.TotalSize += int32(n)
			e.stat.NumRecords += 1

			// Stop export process if condition reached.
			if e.stat.TotalSize >= e.params.SizeLim || e.stat.NumRecords >= e.params.RecordLim {
				report, err := e.Stop(e.params.Persis, doOnce)
				if err != nil {
					return fmt.Errorf("handleRecord: error while calling stop %w", err)
				}
				// TODO: handle stat
				fmt.Println(report)
			}
		}
	}
}

func createFile(t time.Time, n, s int32, d time.Duration, id string) (*os.File, error) {
	fileNameParts := []string{t.Format("01-02-2006 15:04:05")}
	if n > 0 { // Number of records.
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
	fileName := strings.Join(append(fileNameParts, "-"), "-")
	return ioutil.TempFile("", fileName)
}
