package notes

import (
	"database/sql"
	"html/template"
	"net/http"
)

type Handlers struct {
	db  *sql.DB
	tpl *template.Template
}
func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /car", h.Index)
}
