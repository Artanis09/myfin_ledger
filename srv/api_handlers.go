package srv

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"srv.exe.dev/db/dbgen"
)

func (s *Server) writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// Dashboard API
func (s *Server) HandleAPIDashboard(w http.ResponseWriter, r *http.Request) {
	q := dbgen.New(s.DB)
	ctx := r.Context()
	now := time.Now()
	
	cards, _ := q.GetAllCards(ctx)
	txns, _ := q.GetAllTransactions(ctx)
	
	recentTxns := txns
	if len(recentTxns) > 10 {
		recentTxns = recentTxns[:10]
	}
	
	type BillingPeriodJSON struct {
		CardName  string `json:"card_name"`
		CardID    int64  `json:"card_id"`
		StartDay  int64  `json:"start_day"`
		EndDay    int64  `json:"end_day"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		Total     int64  `json:"total"`
	}
	
	var billingPeriods []BillingPeriodJSON
	var totalThisMonth int64
	
	for _, card := range cards {
		start, end := calculateBillingPeriod(now, int(card.BillingStartDay), int(card.BillingEndDay))
		startStr := start.Format("2006-01-02")
		endStr := end.Format("2006-01-02")
		
		var cardTotal int64
		row := s.DB.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(amount), 0) FROM transactions 
			WHERE card_id = ? AND substr(transaction_date, 1, 10) >= ? AND substr(transaction_date, 1, 10) < ?
		`, card.ID, startStr, endStr)
		row.Scan(&cardTotal)
		
		if cardTotal > 0 {
			billingPeriods = append(billingPeriods, BillingPeriodJSON{
				CardName:  card.Name,
				CardID:    card.ID,
				StartDay:  card.BillingStartDay,
				EndDay:    card.BillingEndDay,
				StartDate: start.Format("01/02"),
				EndDate:   end.Add(-24*time.Hour).Format("01/02"),
				Total:     cardTotal,
			})
			totalThisMonth += cardTotal
		}
	}
	
	s.writeJSON(w, map[string]any{
		"total_this_month":     totalThisMonth,
		"current_month":        now.Format("2006년 01월"),
		"billing_periods":      billingPeriods,
		"recent_transactions":  recentTxns,
	})
}

// Transactions API
func (s *Server) HandleAPIGetTransactions(w http.ResponseWriter, r *http.Request) {
	q := dbgen.New(s.DB)
	txns, err := q.GetAllTransactions(r.Context())
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]any{"transactions": txns})
}

func (s *Server) HandleAPIGetTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	q := dbgen.New(s.DB)
	tx, err := q.GetTransaction(r.Context(), id)
	if err != nil {
		s.writeError(w, 404, "Not found")
		return
	}
	s.writeJSON(w, tx)
}

func (s *Server) HandleAPICreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CardID            int64  `json:"card_id"`
		TransactionDate   string `json:"transaction_date"`
		Amount            int64  `json:"amount"`
		Description       string `json:"description"`
		CategoryID        *int64 `json:"category_id"`
		IsInstallment     int64  `json:"is_installment"`
		InstallmentMonths *int64 `json:"installment_months"`
		InstallmentCurrent *int64 `json:"installment_current"`
		OriginalAmount    *int64 `json:"original_amount"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}
	
	q := dbgen.New(s.DB)
	ctx := r.Context()
	
	txDate, _ := time.Parse("2006-01-02", req.TransactionDate)
	
	categoryID := req.CategoryID
	if categoryID == nil {
		categoryID = s.autoCategorize(ctx, q, req.Description)
	}
	
	tx, err := q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		CardID:            req.CardID,
		CategoryID:        categoryID,
		TransactionDate:   txDate,
		Description:       req.Description,
		Amount:            req.Amount,
		IsInstallment:     req.IsInstallment,
		InstallmentMonths: req.InstallmentMonths,
		InstallmentCurrent: req.InstallmentCurrent,
		OriginalAmount:    req.OriginalAmount,
	})
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, tx)
}

func (s *Server) HandleAPIUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	var req struct {
		CardID            int64  `json:"card_id"`
		TransactionDate   string `json:"transaction_date"`
		Amount            int64  `json:"amount"`
		Description       string `json:"description"`
		CategoryID        *int64 `json:"category_id"`
		IsInstallment     int64  `json:"is_installment"`
		InstallmentMonths *int64 `json:"installment_months"`
		InstallmentCurrent *int64 `json:"installment_current"`
		OriginalAmount    *int64 `json:"original_amount"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}
	
	q := dbgen.New(s.DB)
	txDate, _ := time.Parse("2006-01-02", req.TransactionDate)
	
	err := q.UpdateTransaction(r.Context(), dbgen.UpdateTransactionParams{
		ID:                id,
		CardID:            req.CardID,
		CategoryID:        req.CategoryID,
		TransactionDate:   txDate,
		Description:       req.Description,
		Amount:            req.Amount,
		IsInstallment:     req.IsInstallment,
		InstallmentMonths: req.InstallmentMonths,
		InstallmentCurrent: req.InstallmentCurrent,
		OriginalAmount:    req.OriginalAmount,
	})
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) HandleAPIDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	q := dbgen.New(s.DB)
	q.DeleteTransaction(r.Context(), id)
	s.writeJSON(w, map[string]string{"status": "ok"})
}

// Cards API
func (s *Server) HandleAPIGetCards(w http.ResponseWriter, r *http.Request) {
	q := dbgen.New(s.DB)
	cards, err := q.GetAllCards(r.Context())
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]any{"cards": cards})
}

func (s *Server) HandleAPICreateCard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name            string `json:"name"`
		BillingStartDay int64  `json:"billing_start_day"`
		BillingEndDay   int64  `json:"billing_end_day"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}
	
	q := dbgen.New(s.DB)
	card, err := q.CreateCard(r.Context(), dbgen.CreateCardParams{
		Name:            req.Name,
		BillingStartDay: req.BillingStartDay,
		BillingEndDay:   req.BillingEndDay,
	})
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, card)
}

func (s *Server) HandleAPIUpdateCard(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	var req struct {
		Name            string `json:"name"`
		BillingStartDay int64  `json:"billing_start_day"`
		BillingEndDay   int64  `json:"billing_end_day"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}
	
	q := dbgen.New(s.DB)
	err := q.UpdateCard(r.Context(), dbgen.UpdateCardParams{
		ID:              id,
		Name:            req.Name,
		BillingStartDay: req.BillingStartDay,
		BillingEndDay:   req.BillingEndDay,
	})
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) HandleAPIDeleteCard(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	q := dbgen.New(s.DB)
	q.DeleteCard(r.Context(), id)
	s.writeJSON(w, map[string]string{"status": "ok"})
}

// Categories API
func (s *Server) HandleAPIGetCategories(w http.ResponseWriter, r *http.Request) {
	q := dbgen.New(s.DB)
	categories, err := q.GetAllCategories(r.Context())
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]any{"categories": categories})
}

func (s *Server) HandleAPICreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Keywords string `json:"keywords"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}
	
	q := dbgen.New(s.DB)
	cat, err := q.CreateCategory(r.Context(), dbgen.CreateCategoryParams{
		Name:     req.Name,
		Keywords: req.Keywords,
	})
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, cat)
}

func (s *Server) HandleAPIUpdateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	var req struct {
		Name     string `json:"name"`
		Keywords string `json:"keywords"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}
	
	q := dbgen.New(s.DB)
	err := q.UpdateCategory(r.Context(), dbgen.UpdateCategoryParams{
		ID:       id,
		Name:     req.Name,
		Keywords: req.Keywords,
	})
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) HandleAPIDeleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	q := dbgen.New(s.DB)
	q.DeleteCategory(r.Context(), id)
	s.writeJSON(w, map[string]string{"status": "ok"})
}

// Statistics API
func (s *Server) HandleAPIStatistics(w http.ResponseWriter, r *http.Request) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	
	if startStr == "" || endStr == "" {
		now := time.Now()
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0)
		startStr = start.Format("2006-01-02")
		endStr = end.Format("2006-01-02")
	}
	
	ctx := r.Context()
	
	type CategoryStatJSON struct {
		CategoryName string `json:"category_name"`
		TotalAmount  int64  `json:"total_amount"`
		Count        int64  `json:"count"`
	}
	
	type CardStatJSON struct {
		CardName    string `json:"card_name"`
		CardID      int64  `json:"card_id"`
		TotalAmount int64  `json:"total_amount"`
		Count       int64  `json:"count"`
	}
	
	var byCategory []CategoryStatJSON
	rowsCat, err := s.DB.QueryContext(ctx, `
		SELECT 
			COALESCE(cat.name, '미분류') as category_name,
			SUM(t.amount) as total_amount,
			COUNT(*) as count
		FROM transactions t
		LEFT JOIN categories cat ON t.category_id = cat.id
		WHERE substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) <= ?
		GROUP BY t.category_id
		ORDER BY total_amount DESC
	`, startStr, endStr)
	if err == nil {
		defer rowsCat.Close()
		for rowsCat.Next() {
			var cs CategoryStatJSON
			if err := rowsCat.Scan(&cs.CategoryName, &cs.TotalAmount, &cs.Count); err == nil {
				byCategory = append(byCategory, cs)
			}
		}
	} else {
		slog.Warn("query by category", "error", err)
	}
	
	var byCard []CardStatJSON
	rowsCard, err := s.DB.QueryContext(ctx, `
		SELECT 
			c.name as card_name,
			c.id as card_id,
			SUM(t.amount) as total_amount,
			COUNT(*) as count
		FROM transactions t
		JOIN cards c ON t.card_id = c.id
		WHERE substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) <= ?
		GROUP BY t.card_id
		ORDER BY total_amount DESC
	`, startStr, endStr)
	if err == nil {
		defer rowsCard.Close()
		for rowsCard.Next() {
			var cs CardStatJSON
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
	
	s.writeJSON(w, map[string]any{
		"by_category": byCategory,
		"by_card":     byCard,
		"total":       total,
	})
}

// SMS Parsing API
func (s *Server) HandleAPIParseSMS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}
	
	result := parseSMS(req.Text)
	s.writeJSON(w, result)
}

func (s *Server) autoCategorize(ctx interface{ Done() <-chan struct{}; Deadline() (time.Time, bool); Err() error; Value(any) any }, q *dbgen.Queries, description string) *int64 {
	categories, err := q.GetAllCategories(ctx)
	if err != nil {
		return nil
	}
	
	descLower := strings.ToLower(description)
	
	for _, cat := range categories {
		if cat.Keywords == "" {
			continue
		}
		keywords := strings.Split(cat.Keywords, ",")
		for _, kw := range keywords {
			kw = strings.TrimSpace(strings.ToLower(kw))
			if kw != "" && strings.Contains(descLower, kw) {
				return &cat.ID
			}
		}
	}
	
	for _, cat := range categories {
		if cat.Name == "기타" {
			return &cat.ID
		}
	}
	
	return nil
}
