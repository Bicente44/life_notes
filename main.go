
package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "modernc.org/sqlite"
	"html/template"
	"github.com/Bicente44/life_notes/notes"
)

//go:embed static
var staticFS embed.FS

//go:embed templates
var templatesFS embed.FS

const (
	addr = ":42069"
	dbFile = "file:life_notes.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
    }
    defer db.Close()
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	if err := initDB(ctx, db); err != nil {
		return fmt.Errorf("init db: %w", err)
	}

	templates, err := parseTemplates()
	if err != nil {
		return fmt.Errorf("parse templates: %w", err)
	}

	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("static subtree: %w", err)
	}

	// Multiplexer (request router/lookup table) & serve files
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))
	
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		data := struct{ Title string }{Title: "Home"}
		if err := templates["home"].ExecuteTemplate(w, "base", data); err != nil {
			log.Printf("home template failed: %v", err)
		}
	})

	// Setup all the handlers for each note
	carHandlers := notes.NewCarHandlers(db, templates)
	carHandlers.Register(mux)

	// Configure HTTP server
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("serving on http://localhost%s", addr)
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErr <- err
	}()
	
	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Println("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func parseTemplates() (map[string]*template.Template, error) {
	pages := []string{ "home",													// Project pages
					"car", "service", "schedule", "parts", "fuel", 				// Car pages
				}
	templates := make(map[string]*template.Template)

	for _, name := range pages {
		ts, err := template.ParseFS(templatesFS, "templates/base.gohtml", "templates/"+name+".gohtml")
		if err != nil {
			return nil, err
		}
		templates[name] = ts
	}
	return templates, nil
}
