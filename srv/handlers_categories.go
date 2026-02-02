package srv

import (
	"log/slog"
	"net/http"
	"strconv"

	"srv.exe.dev/db/dbgen"
)

func (s *Server) HandleCategories(w http.ResponseWriter, r *http.Request) {
	q := dbgen.New(s.DB)
	categories, _ := q.GetAllCategories(r.Context())
	
	data := struct {
		Categories []dbgen.Category
	}{categories}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderTemplate(w, "categories.html", data); err != nil {
		slog.Warn("render template", "error", err)
		http.Error(w, err.Error(), 500)
	}
}

func (s *Server) HandleCreateCategory(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	q := dbgen.New(s.DB)
	
	name := r.FormValue("name")
	keywords := r.FormValue("keywords")
	
	_, err := q.CreateCategory(r.Context(), dbgen.CreateCategoryParams{
		Name:     name,
		Keywords: keywords,
	})
	if err != nil {
		slog.Warn("create category", "error", err)
	}
	
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (s *Server) HandleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	q := dbgen.New(s.DB)
	
	name := r.FormValue("name")
	keywords := r.FormValue("keywords")
	
	err := q.UpdateCategory(r.Context(), dbgen.UpdateCategoryParams{
		ID:       id,
		Name:     name,
		Keywords: keywords,
	})
	if err != nil {
		slog.Warn("update category", "error", err)
	}
	
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (s *Server) HandleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	q := dbgen.New(s.DB)
	q.DeleteCategory(r.Context(), id)
	
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}
