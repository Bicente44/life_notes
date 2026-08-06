
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
)

//go:embed static
var staticFS embed.FS

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

	// TODO: Create tables here, possibly call a function outside of run like init db or something

	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("static subtree: %w", err)
	}

	// Multiplexer (request router/lookup table) & serve files
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))
	

	// The URL endpoints live in each respective section
	// Add each section when they are implemented
	/*sections := []section.Section{
		car.New(db, tpl),
		//finance.New(db, tpl),
	}

	home := section.NewHome(sections, tpl)
	mux.HandleFunc("GET /{$}", home.Index)

	for _, s := range sections {
		s.Register(mux)
	}*/

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
