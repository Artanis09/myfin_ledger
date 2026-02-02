package srv

import (
	"log/slog"
	"net/http"
	"time"
)

type StatisticsData struct {
	ByCategory    []CategoryStat
	ByCard        []CardStat
	Total         int64
	StartDate     string
	EndDate       string
	CurrentMonth  string
}

type CategoryStat struct {
	CategoryName string
	TotalAmount  int64
	Count        int64
}

type CardStat struct {
	CardName    string
	CardID      int64
	TotalAmount int64
	Count       int64
}

func (s *Server) HandleStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	now := time.Now()
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	
	var startDate, endDate string
	if startStr != "" && endStr != "" {
		startDate = startStr
		endDate = endStr
	} else {
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0)
		startDate = start.Format("2006-01-02")
		endDate = end.Format("2006-01-02")
	}
	
	// Query by category using raw SQL
	var byCategory []CategoryStat
	rowsCat, err := s.DB.QueryContext(ctx, `
		SELECT 
			COALESCE(cat.name, '미분류') as category_name,
			SUM(t.amount) as total_amount,
			COUNT(*) as count
		FROM transactions t
		LEFT JOIN categories cat ON t.category_id = cat.id
		WHERE substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) < ?
		GROUP BY t.category_id
		ORDER BY total_amount DESC
	`, startDate, endDate)
	if err == nil {
		defer rowsCat.Close()
		for rowsCat.Next() {
			var cs CategoryStat
			if err := rowsCat.Scan(&cs.CategoryName, &cs.TotalAmount, &cs.Count); err == nil {
				byCategory = append(byCategory, cs)
			}
		}
	} else {
		slog.Warn("query by category", "error", err)
	}
	
	// Query by card using raw SQL
	var byCard []CardStat
	rowsCard, err := s.DB.QueryContext(ctx, `
		SELECT 
			c.name as card_name,
			c.id as card_id,
			SUM(t.amount) as total_amount,
			COUNT(*) as count
		FROM transactions t
		JOIN cards c ON t.card_id = c.id
		WHERE substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) < ?
		GROUP BY t.card_id
		ORDER BY total_amount DESC
	`, startDate, endDate)
	if err == nil {
		defer rowsCard.Close()
		for rowsCard.Next() {
			var cs CardStat
			if err := rowsCard.Scan(&cs.CardName, &cs.CardID, &cs.TotalAmount, &cs.Count); err == nil {
				byCard = append(byCard, cs)
			}
		}
	} else {
		slog.Warn("query by card", "error", err)
	}
	
	var total int64
	for _, c := range byCard {
		total += c.TotalAmount
	}
	
	data := StatisticsData{
		ByCategory:   byCategory,
		ByCard:       byCard,
		Total:        total,
		StartDate:    startDate,
		EndDate:      endDate,
		CurrentMonth: now.Format("2006년 01월"),
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderTemplate(w, "statistics.html", data); err != nil {
		slog.Warn("render template", "error", err)
		http.Error(w, err.Error(), 500)
	}
}
