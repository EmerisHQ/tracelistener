package exporter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (e *Exporter) ListenAndServeHTTP(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/start", e.startHandler)
	mux.HandleFunc("/stop", e.stopHandler)
	mux.HandleFunc("/stat", e.statHandler)

	if port == "" {
		port = ":8111"
	}
	if !strings.HasPrefix(port, ":") {
		port = fmt.Sprintf(":%s", port)
	}

	if err := (&http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  100 * time.Second,
		WriteTimeout: 100 * time.Second,
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

	validParams := map[string]bool{"count": true, "size": true, "persist": true, "duration": true, "id": true}
	for p := range qp {
		if _, ok := validParams[p]; !ok {
			writeError(w, fmt.Errorf("validation error: unknown param %s", p), http.StatusBadRequest)
			return
		}
	}

	count := qp.Get("count")
	if len(count) > 0 {
		if numTraces, err = validateMustIntMustSuffix(count, "N", "count"); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
	}

	size := qp.Get("size")
	if len(size) > 0 {
		if sizeLim, err = validateMustIntMustSuffix(size, "MB", "size"); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
	}

	dur := qp.Get("duration")
	if len(dur) > 0 {
		if duration, err = validateMustDuration(dur); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
	}

	ps := qp.Get("persist")
	if len(ps) > 0 {
		if persist, err = validateMustBool(ps); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
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
		err := <-errCh
		e.muStat.Lock()
		e.Stat.Errors = append(e.Stat.Errors, err)
		e.muStat.Unlock()
	}()

	e.statHandler(w, r)
}

func (e *Exporter) stopHandler(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	var err error

	validParams := map[string]bool{"clean": true}
	for p := range qp {
		if _, ok := validParams[p]; !ok {
			writeError(w, fmt.Errorf("validation error: unknown param %s", p), http.StatusBadRequest)
			return
		}
	}

	clean := qp.Get("clean")
	if len(clean) > 0 && e.IsRunning() {
		if e.params.Clean, err = validateMustBool(clean); err != nil {
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
	writeContentType(w, []string{"application/json; charset=utf-8"})
	w.WriteHeader(code)
	errMsg := "no error"
	if err != nil {
		errMsg = err.Error()
	}
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{errMsg})
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

func validateMustIntMustSuffix(s string, suf string, pName string) (int32, error) {
	if !strings.HasSuffix(s, suf) {
		return 0, fmt.Errorf("invalid query param %s, want format 20%s got %s", pName, suf, s)
	}
	s = strings.TrimSuffix(s, suf)
	val, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid query param %s, %w", pName, err)
	}
	return int32(val), nil
}
