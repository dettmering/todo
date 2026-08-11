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
	"github.com/till/todo/internal/mcp"
	"github.com/till/todo/internal/service"
	"github.com/till/todo/internal/storage/sqlite"
	"github.com/till/todo/internal/web"
	"github.com/till/todo/internal/webhook"
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
	bus.Subscribe(func(ctx context.Context, e events.Event) {
		slog.Info("event", "type", e.Type)
	})
	if url := os.Getenv("WEBHOOK_URL"); url != "" {
		bus.Subscribe(webhook.New(url).Handle)
		slog.Info("webhook configured", "url", url)
	}

	svc := service.New(store.Tasks(), store.Projects(), store.Audit(), store.Comments(), store.Absences(), bus)

	icalToken := os.Getenv("ICAL_TOKEN")
	if icalToken != "" {
		slog.Info("ical absences feed enabled", "path", "/api/absences.ics")
	}

	shareToken := os.Getenv("SHARE_TOKEN")
	shareName := os.Getenv("SHARE_NAME")
	if shareToken != "" {
		slog.Info("absence share page enabled", "path", "/api/absences.html")
	}

	webFS := web.Static()
	root := http.NewServeMux()
	root.Handle("/static/", http.StripPrefix("/static/", http.FileServerFS(webFS)))
	root.Handle("/api/", httpapi.New(svc, icalToken, shareToken, shareName))

	// MCP server over Streamable HTTP, guarded by a bearer token. Empty token
	// disables the route entirely (like ICAL_TOKEN). Behind Traefik it gets a
	// dedicated router that bypasses BasicAuth; the token is the access control.
	if mcpToken := os.Getenv("MCP_TOKEN"); mcpToken != "" {
		mcpHandler := mcp.New(svc, mcpToken)
		root.Handle("/mcp", mcpHandler)
		root.Handle("/mcp/", mcpHandler)
		slog.Info("mcp server enabled", "path", "/mcp")
	}
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
		Addr: *addr,
		// Wrapped at the root so the SPA shell, /static/, the API and the share
		// page all get the same baseline headers — see httpapi.SecurityHeaders.
		Handler:           httpapi.SecurityHeaders(root),
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
