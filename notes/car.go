package notes

import (
	"context"
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"time"
	"strconv"
)

type CarHandlers struct {
	db  *sql.DB
	tpls map[string]*template.Template
}

type carPage struct {
	Title string
	Car	Car
	// Error
}

type servicePage struct {
	Title string
	Services []ServiceLog
	// Error
}

type schedulePage struct {
	Title string
	Schedule []MaintenanceSchedule
	// Error
}

type Car struct {
	ID int
	Make string
	Model string
	Year int
	Trim string 
	Vin string
	LicensePlate string
	Color string
	PurchaseDate string
	PurchasePriceCents int
	PurchaseOdometer int
	OilType string
	OilCapacityL float64
	InsuranceProvider string
	InsurancePolicyNumber string
	InsuranceExp string
	RegistrationExp string
	Notes string
	UpdatedAt string
	CreatedAt string
}

type ServiceLog struct {
	ID int
	CarID int
	ServiceType string
	Date string
	Odometer int
	CostCents int
	Vendor string
	Notes string
	CreatedAt string
}

type MaintenanceSchedule struct {
	ID int
	CarID int
	Name string
	ServiceType string
	IntervalKm int
	IntervalMonths int
	Enabled bool
	Notes string
}

func NewCarHandlers(db *sql.DB, tpls map[string]*template.Template) *CarHandlers {
	return &CarHandlers{db: db, tpls: tpls}
}

func (h *CarHandlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /car", h.Index)
	mux.HandleFunc("POST /car", h.Update) // Updates the car details

	mux.HandleFunc("GET /car/service", h.ServiceIndex) // List all services from a car
	mux.HandleFunc("POST /car/service", h.CreateService) // Create/Edit a service
	mux.HandleFunc("POST /car/service/{id}/delete", h.DeleteService) // Delete a service

	mux.HandleFunc("GET /car/schedule", h.ScheduleIndex) // List all maintenance schedules from a car
	mux.HandleFunc("POST /car/schedule", h.CreateSchedule) // Create/Edit a maintenance schedule
	mux.HandleFunc("POST /car/schedule/{id}/delete", h.DeleteSchedule) // Delete a maintenance schedule
}

func (h *CarHandlers) Index(w http.ResponseWriter, r *http.Request) {
	car, err := h.getCar(r.Context())
	if err != nil {
		log.Printf("get car: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	data := carPage{ Title: "Car", Car: car }
	if err := h.tpls["car"].ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("car template execution failed: %v", err)
	}
}

func (h *CarHandlers) ServiceIndex(w http.ResponseWriter, r *http.Request) {
	serviceLogs, err := h.listServices(r.Context())
	if err != nil {
		log.Printf("get services: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	data := servicePage{ Title: "Services", Services: serviceLogs }
	if err := h.tpls["service"].ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("service template execution failed: %v", err)
	}
}

func (h *CarHandlers) ScheduleIndex(w http.ResponseWriter, r *http.Request) {
	maintenanceSchedule, err := h.listSchedules(r.Context())
	if err != nil {
		log.Printf("get schedules: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	data := schedulePage{ Title: "Maintenence Schedule", Schedule: maintenanceSchedule }
	if err := h.tpls["schedule"].ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("schedule template execution failed: %v", err)
	}
}

func (h *CarHandlers) listSchedules(ctx context.Context) ([]MaintenanceSchedule, error) {
	var schedule []MaintenanceSchedule
	id := 1
	query := `
		SELECT
			id, name, service_type, COALESCE(interval_km, 0),
			COALESCE(interval_months, 0), enabled, COALESCE(notes, '')
		FROM maintenance_schedule
		WHERE car_id = ?
		ORDER BY name
	`
	rows, err := h.db.QueryContext(ctx, query, id)
	if err != nil {
		log.Printf("List schedules db execution failed: %v", err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s MaintenanceSchedule
		err := rows.Scan(&s.ID, &s.Name, &s.ServiceType, &s.IntervalKm, &s.IntervalMonths, &s.Enabled,
						&s.Notes)
		if err != nil {
			log.Printf("Failed to scan maintenance schedule rows: %v", err)
			return nil, err
		}
		schedule = append(schedule, s)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Row errors: %v", err)
		return nil, err
	}
	return schedule, err
}

func (h *CarHandlers) listServices(ctx context.Context) ([]ServiceLog, error) {
	var serviceLog []ServiceLog
	id := 1	// In the future make this as a param in getCar function (only one car right now)
	query := `
		SELECT
			id, service_type, date, odometer, COALESCE(cost_cents, 0), COALESCE(vendor, ''),
			COALESCE(notes, ''), created_at
		FROM service_log
		WHERE car_id = ?
		ORDER BY date DESC, id DESC
	`
	rows, err := h.db.QueryContext(ctx, query, id)
	if err != nil {
		log.Printf("List service db execution failed: %v", err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s ServiceLog
		err := rows.Scan(&s.ID, &s.ServiceType, &s.Date, &s.Odometer, &s.CostCents, &s.Vendor,
						&s.Notes, &s.CreatedAt)
		if err != nil {
			log.Printf("Failed to scan service log rows: %v", err)
			return nil, err
		}
		serviceLog = append(serviceLog, s)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Row errors: %v", err)
		return nil, err
	}
	return serviceLog, err
}

// This is to update the car details
func (h *CarHandlers) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	id, err := formInt(r, "id")
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}
	year, err := formInt(r, "year")
	if err != nil {
		http.Error(w, "Invalid year format", http.StatusBadRequest)
		return
	}
	price, err := formInt(r, "purchase_price_cents")
	if err != nil {
		http.Error(w, "Invalid price format", http.StatusBadRequest)
		return
	}
	odometer, err := formInt(r, "purchase_odometer")
	if err != nil {
		http.Error(w, "Invalid odometer format", http.StatusBadRequest)
		return
	}
	oilCap, err := formFloat(r, "oil_capacity_l")
	if err != nil {
		http.Error(w, "Invalid oil capacity format", http.StatusBadRequest)
		return
	}
	car := Car{
		ID:                    id,
		Make:                  r.FormValue("make"),
		Model:                 r.FormValue("model"),
		Year:                  year,
		Trim:                  r.FormValue("trim"),
		Vin:                   r.FormValue("vin"),
		LicensePlate:          r.FormValue("license_plate"),
		Color:                 r.FormValue("color"),
		PurchaseDate:          r.FormValue("purchase_date"),
		PurchasePriceCents:    price,
		PurchaseOdometer:      odometer,
		OilType:               r.FormValue("oil_type"),
		OilCapacityL:          oilCap,
		InsuranceProvider:     r.FormValue("insurance_provider"),
		InsurancePolicyNumber: r.FormValue("insurance_policy_number"),
		InsuranceExp:          r.FormValue("insurance_expires"),
		RegistrationExp:       r.FormValue("registration_expires"),
		Notes:                 r.FormValue("notes"),
	}
	if err := h.updateCar(r.Context(), car); err != nil {
		http.Error(w, "Failed to update car in database", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/car", http.StatusSeeOther)
}

func (h *CarHandlers) getCar(ctx context.Context) (Car, error) {
	var car Car
	id := 1	// In the future make this as a param in getCar function (only one car right now)
	query := `
		SELECT 
			id, make, model, year, trim, vin, license_plate, color, 
			purchase_date, purchase_price_cents, purchase_odometer, 
			oil_type, oil_capacity_l, insurance_provider, 
			insurance_policy_number, insurance_expires, registration_expires, 
			notes, updated_at, created_at
		FROM car 
		WHERE id = ?
	`
	err := h.db.QueryRowContext(ctx, query, id).Scan(
		&car.ID, &car.Make, &car.Model, &car.Year, &car.Trim, &car.Vin,
		&car.LicensePlate, &car.Color,&car.PurchaseDate, &car.PurchasePriceCents,
		&car.PurchaseOdometer,&car.OilType, &car.OilCapacityL, &car.InsuranceProvider, 
		&car.InsurancePolicyNumber, &car.InsuranceExp, &car.RegistrationExp,&car.Notes,
		&car.UpdatedAt, &car.CreatedAt,
	)
	return car, err
}

// Updates the whole car, this simplifies things instead of having many insert funcs
func (h *CarHandlers) updateCar(ctx context.Context, car Car) error {
	query :=`
			UPDATE car SET 
				make = ?, model = ?, year = ?, trim = ?, vin = ?, license_plate = ?, color = ?, 
				purchase_date = ?, purchase_price_cents = ?, purchase_odometer = ?, 
				oil_type = ?, oil_capacity_l = ?, insurance_provider = ?, 
				insurance_policy_number = ?, insurance_expires = ?, registration_expires = ?, 
				notes = ?, updated_at = ?
			WHERE id = ?
			`
	updatedAt := time.Now().Format(time.RFC3339)

	_, err := h.db.ExecContext(ctx, query,
			car.Make, 
			car.Model, 
			car.Year, 
			car.Trim, 
			car.Vin, 
			car.LicensePlate, 
			car.Color,
			car.PurchaseDate, 
			car.PurchasePriceCents, 
			car.PurchaseOdometer,
			car.OilType, 
			car.OilCapacityL, 
			car.InsuranceProvider,
			car.InsurancePolicyNumber, 
			car.InsuranceExp, 
			car.RegistrationExp,
			car.Notes, 
			updatedAt,
			car.ID,    // WHERE
		)
	return err
}

func (h *CarHandlers) saveService(ctx context.Context, service ServiceLog) error {
	if service.ID == 0 { 
		query := `
			INSERT INTO service_log 
			(car_id, service_type, date, odometer, cost_cents, vendor, notes, created_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := h.db.ExecContext(ctx, query,
			service.CarID, service.ServiceType, service.Date, service.Odometer, service.CostCents, 
			service.Vendor, service.Notes, time.Now().Format(time.RFC3339),
		)
		return err
	} else {
		query := `
			UPDATE service_log 
			SET car_id = ?, service_type = ?, date = ?, odometer = ?, 
			    cost_cents = ?, vendor = ?, notes = ?
			WHERE id = ?
		`
		r, err := h.db.ExecContext(ctx, query,
			service.CarID, service.ServiceType, service.Date, service.Odometer, service.CostCents, 
			service.Vendor, service.Notes, service.ID,
		)
		if err != nil {
			return err
		}
		rowsAffected, err := r.RowsAffected()
		if err != nil {
			log.Printf("could not update service: no row found with id %d", service.ID)
			return err
		}
		if rowsAffected == 0 {
			return sql.ErrNoRows
		}
		return nil
	}
}

func (h *CarHandlers) deleteService(ctx context.Context, id int) error {
	query := `
		DELETE FROM service_log
		WHERE id = ?
	`
	_, err := h.db.ExecContext(ctx, query, id)
	return err
}

func (h *CarHandlers) CreateService(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	id, err := formInt(r, "id")
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	/* TODO Only one car right now, implement later
	carID, err := formInt(r, "car_id")
	if err != nil {
		http.Error(w, "Invalid car ID", http.StatusBadRequest)
		return
	}*/
	odometer, err := formInt(r, "odometer")
	if err != nil {
		http.Error(w, "Invalid odometer", http.StatusBadRequest)
		return
	}
	costCents, err := formInt(r, "cost_cents")
	if err != nil {
		http.Error(w, "Invalid cost cents format", http.StatusBadRequest)
		return
	}
	serviceLog := ServiceLog {
		ID: id,
		CarID: 1, // TODO: Implement way for multiple cars
		ServiceType: r.FormValue("service_type"),
		Date: r.FormValue("date"),
		Odometer: odometer,
		CostCents: costCents,
		Vendor: r.FormValue("vendor"),
		Notes: r.FormValue("notes"),
	}
	if err := h.saveService(r.Context(), serviceLog); err != nil {
		log.Printf("save service: %v", err)
		http.Error(w, "Failed to create service in database", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/car/service", http.StatusSeeOther)
}

func (h *CarHandlers) DeleteService(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid id format", http.StatusBadRequest)
		return
	}
	if err := h.deleteService(r.Context(), id); err != nil {
		log.Printf("delete service: %v", err)
		http.Error(w, "Failed to delete service in database", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/car/service", http.StatusSeeOther)
}

// Scheduled Maintenance functions
func (h *CarHandlers) saveSchedule(ctx context.Context, schedule MaintenanceSchedule) error {
	if schedule.ID == 0 { 
		query := `
			INSERT INTO maintenance_schedule 
			(car_id, name, service_type, interval_km, interval_months, enabled, notes) 
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`
		_, err := h.db.ExecContext(ctx, query,
			schedule.CarID, schedule.Name, schedule.ServiceType, schedule.IntervalKm, schedule.IntervalMonths,
			schedule.Enabled, schedule.Notes,
		)
		return err
	} else {
		query := `
			UPDATE maintenance_schedule 
			SET car_id = ?, name = ?, service_type = ?, interval_km = ?,
				interval_months = ?, enabled = ?, notes = ?
			WHERE id = ?
		`
		r, err := h.db.ExecContext(ctx, query,
			schedule.CarID, schedule.Name, schedule.ServiceType, schedule.IntervalKm, schedule.IntervalMonths,
			schedule.Enabled, schedule.Notes, schedule.ID,
		)
		if err != nil {
			return err
		}
		rowsAffected, err := r.RowsAffected()
		if err != nil {
			log.Printf("could not update schedule: no row found with id %d", schedule.ID)
			return err
		}
		if rowsAffected == 0 {
			return sql.ErrNoRows
		}
		return nil
	}
}

func (h *CarHandlers) deleteSchedule(ctx context.Context, id int) error {
	query := `
		DELETE FROM maintenance_schedule
		WHERE id = ?
	`
	_, err := h.db.ExecContext(ctx, query, id)
	return err
}

func (h *CarHandlers) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	id, err := formInt(r, "id")
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	/* TODO Only one car right now, implement later
	carID, err := formInt(r, "car_id")
	if err != nil {
		http.Error(w, "Invalid car ID", http.StatusBadRequest)
		return
	}*/
	intervalKm, err := formInt(r, "interval_km")
	if err != nil {
		http.Error(w, "Invalid inteval km", http.StatusBadRequest)
		return
	}
	intervalMonths, err := formInt(r, "interval_months")
	if err != nil {
		http.Error(w, "Invalid interval months", http.StatusBadRequest)
		return
	}
	schedule := MaintenanceSchedule {
		ID: id,
		CarID: 1, // TODO: Implement way for multiple cars
		Name: r.FormValue("name"),
		ServiceType: r.FormValue("service_type"),
		IntervalKm:	intervalKm, 
		IntervalMonths: intervalMonths,
		Enabled: r.FormValue("enabled") == "on",
		Notes: r.FormValue("notes"),
	}
	if err := h.saveSchedule(r.Context(), schedule); err != nil {
		log.Printf("save schedule: %v", err)
		http.Error(w, "Failed to create schedule in database", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/car/schedule", http.StatusSeeOther)
}

func (h *CarHandlers) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid id format", http.StatusBadRequest)
		return
	}
	if err := h.deleteSchedule(r.Context(), id); err != nil {
		log.Printf("delete schedule: %v", err)
		http.Error(w, "Failed to delete schedule in database", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/car/schedule", http.StatusSeeOther)
}

// Claculations and Helpers
func (h *CarHandlers) getOdometer(ctx context.Context, carId int) (int, error) {
	var odometer int
	query := `
			SELECT COALESCE(MAX(odometer), 0) FROM (
			  SELECT odometer FROM service_log WHERE car_id = ?
			  UNION ALL SELECT odometer FROM fuel_log WHERE car_id = ?
			  UNION ALL SELECT odometer FROM odometer_reading WHERE car_id = ?
			)
		`
	err := h.db.QueryRowContext(ctx, query, carId, carId, carId).Scan(&odometer)
	if err != nil {
		return 0, err
	}
	return odometer, err
}
// Parse integers from form data
func formInt(r *http.Request, name string) (int, error) {
	v := r.FormValue(name)
	if v == "" {
		return 0, nil
	}
	return strconv.Atoi(v)
}
// Parse floats from form data
func formFloat(r *http.Request, name string) (float64, error) {
	v := r.FormValue(name)
	if v == "" {
		return 0.0, nil
	}
	return strconv.ParseFloat(v, 64)
}
