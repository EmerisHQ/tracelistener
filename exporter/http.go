package exporter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emerishq/tracelistener/tracelistener/config"
)

func (e *Exporter) ListenAndServeHTTP(cfg *config.Config) {
	mux := http.NewServeMux()
	mux.HandleFunc("/start", e.startHandler)
	mux.HandleFunc("/stop", e.stopHandler)
	mux.HandleFunc("/stat", e.statHandler)

	port := cfg.ExporterHTTPPort
	if port == "" {
		port = ":8111"
	}
	if err := (&http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}).ListenAndServe(); err != nil {
		e.logger.Errorw("server failed to start", "error", err.Error())
	}
}

// startHandler listens on /start. Initializes the exporter with params from url
// and starts exporter.StartReceiving. If another exporter is already running,
// exporter.Init will return error.
func (e *Exporter) startHandler(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	var numTraces int32
	var sizeLim int32
	var duration time.Duration
	var persist bool
	var err error

	n := qp.Get("N")
	if len(n) > 0 {
		if numTraces, err = validateMustIntMustSuffix(n, "N", "N"); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
	}

	m := qp.Get("M")
	if len(m) > 0 {
		if sizeLim, err = validateMustIntMustSuffix(m, "MB", "M"); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
	}

	d := qp.Get("D")
	if len(d) > 0 {
		if duration, err = validateMustDuration(d); err != nil {
			writeError(w, err, http.StatusBadRequest)
		}
	}

	p := qp.Get("P")
	if len(p) > 0 {
		if persist, err = validateMustBool(p); err != nil {
			writeError(w, err, http.StatusBadRequest)
		}
	}

	params := &Params{
		NumTraces: numTraces,
		SizeLim:   sizeLim,
		Duration:  duration,
		Upload:    persist,
		FileId:    qp.Get("id"), // Validation for file id is
		// sufficiently handled when we call e.Init
	}

	if err := e.Init(params); err != nil {
		var vErr ValidationError
		if errors.As(err, &vErr) {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	errCh := e.StartReceiving()
	go func() {
		e.Stat.Errors = append(e.Stat.Errors, <-errCh)
	}()

	e.statHandler(w, r)
}

func (e *Exporter) stopHandler(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	var err error

	c := qp.Get("clean")
	if len(c) > 0 {
		if e.params.Clean, err = validateMustBool(c); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
	}
	if err := e.StopReceiving(); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	e.statHandler(w, r)
}

func (e *Exporter) statHandler(w http.ResponseWriter, _ *http.Request) {
	stat, err := e.GetStat()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	if err := writeStat(w, stat, http.StatusOK); err != nil {
		e.logger.Errorw("StatHandler", "write json", stat, "error", err)
		writeError(w, err, http.StatusInternalServerError)
	}
}

func writeStat(w http.ResponseWriter, stat Stat, code int) error {
	writeContentType(w, []string{"application/json; charset=utf-8"})
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(stat.Public()); err != nil {
		return err
	}
	return nil
}

func writeContentType(w http.ResponseWriter, value []string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = value
	}
}

func writeError(w http.ResponseWriter, err error, code int) {
	w.WriteHeader(code)
	msg := "nil"
	if err != nil {
		msg = "Error: " + err.Error()
	}
	_, _ = w.Write([]byte(msg))
}

func validateMustBool(p string) (bool, error) {
	persist, err := strconv.ParseBool(p)
	if err != nil {
		return false, fmt.Errorf("invalid query param, want either t, true, True, f, false, False, got %s", p)
	}
	return persist, nil
}

func validateMustDuration(d string) (time.Duration, error) {
	duration, err := time.ParseDuration(d)
	if err != nil {
		return 0, fmt.Errorf("invalid query param D, %w", err)
	}
	return duration, nil
}

func validateMustIntMustSuffix(n string, suf string, pName string) (int32, error) {
	if !strings.HasSuffix(n, suf) {
		return 0, fmt.Errorf("invalid query param %s, want format 20%s got %s", pName, suf, n)
	}
	n = strings.TrimSuffix(n, suf)
	val, err := strconv.ParseInt(n, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid query param %s, %w", pName, err)
	}
	return int32(val), nil
}
