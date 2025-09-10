package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	reqsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "Total HTTP requests"},
		[]string{"route", "method", "code"},
	)
	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"route", "method"},
	)
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.NewConsoleWriter())

	addr := getenv("HTTP_ADDR", ":8080")

	prometheus.MustRegister(reqsTotal, reqDuration)

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, middleware.Timeout(30*time.Second))
	r.Use(logMiddleware)

	// health
	r.Get("/healthz", instrument("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// metrics
	r.Handle("/metrics", promhttp.Handler())

	store := &Store{}

	// create events
	r.Post("/events", instrument("/events", func(w http.ResponseWriter, r *http.Request) {
		var in Event
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Type == "" {
			http.Error(w, "invalid json (need type, payload)", http.StatusBadRequest)
			return
		}
		created := store.Add(in)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created)
	}))

	// list events
	r.Get("/events", instrument("/events", func(w http.ResponseWriter, r *http.Request) {
		list := store.List(50)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	}))

	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func instrument(route string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: 200}
		h(sw, r)
		reqsTotal.WithLabelValues(route, r.Method, http.StatusText(sw.code)).Inc()
		reqDuration.WithLabelValues(route, r.Method).Observe(time.Since(start).Seconds())
	}
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Dur("duration", time.Since(start)).
			Msg("request")
	})
}

type Event struct {
	ID         int64     `json:"id"`
	Type       string    `json:"type"`
	Payload    string    `json:"payload"`
	ReceivedAt time.Time `json:"received_at"`
}

type Store struct {
	seq    int64
	events []Event
	mu     sync.Mutex
}

func (s *Store) Add(e Event) Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	e.ID = s.seq
	e.ReceivedAt = time.Now().UTC()
	s.events = append(s.events, e)
	return e
}

func (s *Store) List(limit int) []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}
	out := make([]Event, 0, limit)
	for i := len(s.events) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, s.events[i])
	}
	return out
}
