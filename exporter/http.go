package exporter

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/emerishq/tracelistener/tracelistener/config"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (e *Exporter) ListenAndServe(cfg config.Config) {
	handler := handler{
		exporter: e,
		doOnce:   nil, // Populated when startHandler is called.
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/start", handler.startHandler)
	mux.HandleFunc("/stop", handler.stopHandler)
	mux.HandleFunc("/Stat", handler.statHandler)

	port := cfg.ExporterHTTPPort
	if port == "" {
		port = ":8111"
	}
	s := &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := s.ListenAndServe(); err != nil {
		e.logger.Errorw("server failed to start", "error", err.Error())
	}
}

type handler struct {
	exporter *Exporter
	doOnce   func(func())
}

func (h *handler) startHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := h.exporter.Init(params); err != nil {
		var vErr ValidationError
		if errors.As(err, &vErr) {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	stat, doOnce, errCh := h.exporter.StartReceiving()
	if err := <-errCh; err != nil {
		writeError(w, err, http.StatusInternalServerError)
	}
	h.doOnce = doOnce
	writeJson(w, stat, http.StatusOK)
}

func (h *handler) stopHandler(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	var err error

	c := qp.Get("clean")
	if len(c) > 0 {
		if h.exporter.params.Clean, err = validateMustBool(c); err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
	}
	h.exporter.StopReceiving(h.doOnce)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
	}
}

func (h *handler) statHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
}

func writeJson(w http.ResponseWriter, stat interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(stat)
}

func writeError(w http.ResponseWriter, err error, code int) {
	w.WriteHeader(code)
	_, _ = w.Write([]byte(err.Error()))
}

func validateMustBool(p string) (bool, error) {
	persist, err := strconv.ParseBool(p)
	if err != nil {
		return false, fmt.Errorf("invalid query param P, want either t, true, True, f, false, False, got %s", p)
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
