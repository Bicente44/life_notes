package notes

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
)

type CarHandlers struct {
	db  *sql.DB
	tpl *template.Template
}

type carPage struct {
	Title string
}

func NewCarHandlers(db *sql.DB, tpl *template.Template) *CarHandlers {
	return &CarHandlers{db: db, tpl: tpl}
}

func (h *CarHandlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /car", h.Index)
}

func (h *CarHandlers) Index(w http.ResponseWriter, r *http.Request) {
	data := carPage{
		Title: "Car",
	}

	err := h.tpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Printf("car template execution failed: %v", err)
		return
	}
}
