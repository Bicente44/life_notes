
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


func initDB(ctx context.Context, db *sql.DB) error {
	// db is already opened at this point so i would just prepare the statement i assume.
	initStmt := `
	CREATE TABLE IF NOT EXISTS car (
		id INTEGER PRIMARY KEY,
		make TEXT,
		model TEXT,
		year INTEGER,
		trim TEXT,
		vin TEXT,
		license_plate TEXT,
		color TEXT,
		purchase_date TEXT,
		purchase_price_cents INTEGER,
		purchase_odometer INTEGER,
		oil_type TEXT,
		oil_capacity_l REAL,
		insurance_provider TEXT,
		insurance_policy_number TEXT,
		insurance_expires TEXT,
		registration_expires TEXT,
		notes TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);


	CREATE TABLE IF NOT EXISTS  service_log (
		id INTEGER PRIMARY KEY NOT NULL,
		car_id INTEGER NOT NULL,
		service_type TEXT NOT NULL,
		date TEXT NOT NULL,
		odometer INTEGER NOT NULL,
		cost_cents INTEGER,
		vendor TEXT,
		notes TEXT,
		created_at TEXT NOT NULL,
		FOREIGN KEY (car_id) REFERENCES car(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS  odometer_reading (
		id INTEGER PRIMARY KEY NOT NULL,
		car_id INTEGER NOT NULL,
		date TEXT NOT NULL,
		odometer INTEGER NOT NULL,
		created_at TEXT NOT NULL,
		FOREIGN KEY (car_id) REFERENCES car(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS  fuel_log (
		id INTEGER PRIMARY KEY NOT NULL,
		car_id INTEGER NOT NULL,
		date TEXT NOT NULL,
		odometer INTEGER NOT NULL,
		litres REAL NOT NULL,
		price_per_litre_cents INTEGER,
		total_cents INTEGER,
		station TEXT,
		is_full_tank INTEGER NOT NULL,
		notes TEXT,
		created_at TEXT NOT NULL,
		FOREIGN KEY (car_id) REFERENCES car(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS  part (
		id INTEGER PRIMARY KEY NOT NULL,
		car_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		description TEXT,
		cost_cents INTEGER,
		purchase_date TEXT,
		status TEXT NOT NULL,
		notes TEXT,
		created_at TEXT NOT NULL,
		FOREIGN KEY (car_id) REFERENCES car(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS  maintenance_schedule (
		id INTEGER PRIMARY KEY NOT NULL,
		car_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		service_type TEXT NOT NULL,
		interval_km INTEGER,
		interval_months INTEGER,
		enabled INTEGER NOT NULL,
		notes TEXT,
		FOREIGN KEY (car_id) REFERENCES car(id) ON DELETE CASCADE
	);
	`

	if _, err := db.ExecContext(ctx, initStmt); err != nil {
		return err
	}
	log.Println("DB initialized successfully")

	// Seed first car to ensure FK's dont break in other tables
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM car").Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		now := time.Now().Format(time.RFC3339)
		query := `
			INSERT INTO car (
				make, model, year, trim, vin, license_plate, color, 
				purchase_date, purchase_price_cents, purchase_odometer, 
				oil_type, oil_capacity_l, insurance_provider, 
				insurance_policy_number, insurance_expires, registration_expires, 
				notes, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err = db.ExecContext(ctx, query,
			"Honda",
			"Civic",
			2010,
			"Base",
			"XXXXXXXXXXX",
			"XXXXXX",
			"Black",
			"2023-06-15",
			2000,
			200000,
			"XXX",
			0.0,
			"INSURANCE",
			"POLICY",
			"XXXX-XX-XX",
			"XXXX-XX-XX",
			"Default seed, Tashonda",
			now,
			now,
		)
		if err != nil {
			return err
		}
		log.Println("Inserted a default car because the table was empty.")
	}
	return nil
}

func parseTemplates() (map[string]*template.Template, error) {
	pages := []string{ "home",									// Project pages
					"car", "service", "schedule", 				// Car pages
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
