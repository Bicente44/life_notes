package main

import (
	"context"
	"database/sql"
	"log"
	"time"
)

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
			"DFRF 955",
			"Black",
			"2025-08-30",
			200000,
			200000,
			"XXX",
			3.7,
			"belairdirect",
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
