// Command server runs the todo HTTP server.
package main

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/till/todo/internal/events"
	"github.com/till/todo/internal/httpapi"
	"github.com/till/todo/internal/service"
	"github.com/till/todo/internal/storage/sqlite"
	"github.com/till/todo/internal/web"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	dbPath := flag.String("db", "todo.db", "path to SQLite database file")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	store, err := sqlite.Open(*dbPath)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	bus := events.NewBus()
	// Future: bus.Subscribe(webhookDispatcher.Handle) once Lastenheft 5.11 is implemented.
	bus.Subscribe(func(ctx context.Context, e events.Event) {
		slog.Info("event", "type", e.Type)
	})

	svc := service.New(store.Tasks(), store.Projects(), bus)

	webFS := web.Static()
	root := http.NewServeMux()
	root.Handle("/static/", http.StripPrefix("/static/", http.FileServerFS(webFS)))
	root.Handle("/api/", httpapi.New(svc))
	root.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// SPA catch-all: every path without a file extension serves index.html so
		// the client-side router can handle /today, /projects/<slug>, /tasks/<id>,
		// etc. Paths with an extension (favicon.ico, missing.png) fall through to
		// 404 so the browser doesn't try to render HTML as an image.
		if path.Ext(r.URL.Path) != "" {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(webFS, "index.html")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", *addr, "db", *dbPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
