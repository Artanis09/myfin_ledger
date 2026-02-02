package srv

import (
	"log/slog"
	"net/http"
	"strconv"

	"srv.exe.dev/db/dbgen"
)

func (s *Server) HandleCards(w http.ResponseWriter, r *http.Request) {
	q := dbgen.New(s.DB)
	cards, _ := q.GetAllCards(r.Context())
	
	data := struct {
		Cards []dbgen.Card
	}{cards}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderTemplate(w, "cards.html", data); err != nil {
		slog.Warn("render template", "error", err)
		http.Error(w, err.Error(), 500)
	}
}

func (s *Server) HandleCreateCard(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	q := dbgen.New(s.DB)
	
	name := r.FormValue("name")
	startDay, _ := strconv.ParseInt(r.FormValue("billing_start_day"), 10, 64)
	endDay, _ := strconv.ParseInt(r.FormValue("billing_end_day"), 10, 64)
	
	if startDay == 0 {
		startDay = 1
	}
	if endDay == 0 {
		endDay = 31
	}
	
	_, err := q.CreateCard(r.Context(), dbgen.CreateCardParams{
		Name:            name,
		BillingStartDay: startDay,
		BillingEndDay:   endDay,
	})
	if err != nil {
		slog.Warn("create card", "error", err)
	}
	
	http.Redirect(w, r, "/cards", http.StatusSeeOther)
}

func (s *Server) HandleUpdateCard(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	q := dbgen.New(s.DB)
	
	name := r.FormValue("name")
	startDay, _ := strconv.ParseInt(r.FormValue("billing_start_day"), 10, 64)
	endDay, _ := strconv.ParseInt(r.FormValue("billing_end_day"), 10, 64)
	
	err := q.UpdateCard(r.Context(), dbgen.UpdateCardParams{
		ID:              id,
		Name:            name,
		BillingStartDay: startDay,
		BillingEndDay:   endDay,
	})
	if err != nil {
		slog.Warn("update card", "error", err)
	}
	
	http.Redirect(w, r, "/cards", http.StatusSeeOther)
}

func (s *Server) HandleDeleteCard(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	q := dbgen.New(s.DB)
	q.DeleteCard(r.Context(), id)
	
	http.Redirect(w, r, "/cards", http.StatusSeeOther)
}
