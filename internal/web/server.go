package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/TheDevWhoSaysNi/gads-ios-file-transfer/internal/transfer"
)

//go:embed page.html
var page embed.FS

// Server is the browser upload page.
type Server struct {
	Token    string
	Provider string
	MaxBytes int64
	locks    sync.Map
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.page)
	mux.HandleFunc("GET /api/devices", s.devices)
	mux.HandleFunc("POST /api/push", s.push)
	return mux
}

func (s *Server) page(w http.ResponseWriter, r *http.Request) {
	raw, err := page.ReadFile("page.html")
	if err != nil {
		http.Error(w, "page missing", http.StatusInternalServerError)
		return
	}
	body := strings.ReplaceAll(string(raw), "__REQUIRE_TOKEN__", boolJSON(s.Token != ""))
	w.Header().Set("content-type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, body)
}

func (s *Server) devices(w http.ResponseWriter, r *http.Request) {
	if !s.allowed(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	list, err := transfer.ListDevices()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": list})
}

func (s *Server) push(w http.ResponseWriter, r *http.Request) {
	if !s.allowed(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	maxBytes := s.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 1 << 30
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+1<<20)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Choose a file under the size limit."})
		return
	}
	udid := strings.TrimSpace(r.FormValue("udid"))
	if udid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Choose a phone."})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil || header == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Choose a file."})
		return
	}
	defer file.Close()
	name, err := transfer.SafeName(header.Filename)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	tmp, err := os.CreateTemp("", "gads-upload-*")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not store the upload."})
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	written, err := io.Copy(tmp, file)
	tmp.Close()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "The upload stopped before the file finished."})
		return
	}
	if written == 0 || written > maxBytes {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "The file is empty or larger than the limit."})
		return
	}

	unlock := s.lock(udid)
	defer unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	launch := transfer.LaunchWithInstruments
	if s.Provider != "" {
		launch = transfer.ProviderLaunch(s.Provider)
	}
	outcome, err := transfer.Send(ctx, udid, tmpName, name, launch)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	if s.Provider != "" {
		transfer.ProviderHome(s.Provider, udid)
	}
	writeJSON(w, http.StatusOK, outcome)
}

func (s *Server) allowed(r *http.Request) bool {
	if s.Token == "" {
		return true
	}
	got := r.Header.Get("authorization")
	got = strings.TrimPrefix(got, "Bearer ")
	if got == "" {
		got = r.URL.Query().Get("token")
	}
	return subtleMatch(got, s.Token)
}

func (s *Server) lock(udid string) func() {
	value, _ := s.locks.LoadOrStore(udid, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func boolJSON(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func subtleMatch(got, want string) bool {
	if len(got) != len(want) {
		return false
	}
	var diff byte
	for ni := 0; ni < len(got); ni++ {
		diff |= got[ni] ^ want[ni]
	}
	return diff == 0
}

// PublicListen reports whether addr is reachable from other machines.
func PublicListen(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && !ip.IsLoopback()
}

// ListenAndServe starts the page.
func ListenAndServe(addr string, handler http.Handler) error {
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
