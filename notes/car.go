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
	tpl *template.Template
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

type carPage struct {
	Title string
	Car	Car
}

func NewCarHandlers(db *sql.DB, tpl *template.Template) *CarHandlers {
	return &CarHandlers{db: db, tpl: tpl}
}

func (h *CarHandlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /car", h.Index)
	mux.HandleFunc("POST /car", h.Update) // Updates the car details
}

func (h *CarHandlers) Index(w http.ResponseWriter, r *http.Request) {
	car, err := h.getCar(r.Context())
	if err != nil {
		log.Printf("get car: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	data := carPage{ Title: "Car", Car: car }
	if err := h.tpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("car template execution failed: %v", err)
	}
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

// Helpers
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
