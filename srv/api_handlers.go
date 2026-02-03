package srv

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"srv.exe.dev/db/dbgen"
)

// WeeklyStats represents weekly spending statistics
type WeeklyStats struct {
	WeekStart   string           `json:"week_start"`
	WeekEnd     string           `json:"week_end"`
	WeekLabel   string           `json:"week_label"`
	Total       int64            `json:"total"`
	ByCard      []CardWeekTotal  `json:"by_card"`
}

type CardWeekTotal struct {
	CardID   int64  `json:"card_id"`
	CardName string `json:"card_name"`
	Total    int64  `json:"total"`
}

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
// GET /api/dashboard?year=2026&month=3
func (s *Server) HandleAPIDashboard(w http.ResponseWriter, r *http.Request) {
	q := dbgen.New(s.DB)
	ctx := r.Context()
	
	// Parse year/month from query params (결제예정월)
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	
	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)
	if year == 0 {
		year = time.Now().Year()
	}
	if month == 0 {
		month = int(time.Now().Month())
	}
	
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
		// 결제예정월 기준으로 이용기간 계산
		start, end := calculateBillingPeriodForMonth(year, month, int(card.BillingStartDay), int(card.BillingEndDay))
		startStr := start.Format("2006-01-02")
		endStr := end.Format("2006-01-02")
		
		// Calculate card total including:
		// 1. Regular transactions (non-installment)
		// 2. Installment transactions - monthly payment amount
		// 3. Cancelled transactions - subtract from total
		var cardTotal int64
		
		// Query for non-cancelled, non-installment transactions in this period
		row := s.DB.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(amount), 0) FROM transactions 
			WHERE card_id = ? 
			AND substr(transaction_date, 1, 10) >= ? 
			AND substr(transaction_date, 1, 10) <= ?
			AND is_cancelled = 0
			AND is_installment = 0
		`, card.ID, startStr, endStr)
		row.Scan(&cardTotal)
		
		// Add installment transactions for this billing month
		// An installment tx should be charged if:
		// - Current installment_current <= this billing month's offset from original tx month
		// - AND it hasn't been fully paid (installment_current < installment_months)
		rows, err := s.DB.QueryContext(ctx, `
			SELECT amount, installment_months, installment_current, transaction_date 
			FROM transactions 
			WHERE card_id = ? 
			AND is_cancelled = 0 
			AND is_installment = 1
		`, card.ID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var amount, months, current int64
				var txDateStr string
				if err := rows.Scan(&amount, &months, &current, &txDateStr); err != nil {
					continue
				}
				
				// Parse original transaction date
				txDate, _ := time.Parse("2006-01-02T15:04:05Z", txDateStr)
				if txDate.IsZero() {
					txDate, _ = time.Parse("2006-01-02 15:04:05", txDateStr)
				}
				
				// Calculate which installment month this billing month corresponds to
				// The first installment is charged in the billing month following the tx
				txBillingMonth := getNextBillingMonth(txDate, int(card.BillingStartDay), int(card.BillingEndDay))
				targetBillingMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
				
				// Calculate months difference
				monthsDiff := (targetBillingMonth.Year()-txBillingMonth.Year())*12 + int(targetBillingMonth.Month()-txBillingMonth.Month())
				installmentNum := monthsDiff + 1 // 1-indexed
				
				// If this billing month falls within the installment period
				if installmentNum >= 1 && installmentNum <= int(months) {
					cardTotal += amount // amount is already monthly payment
				}
			}
		}
		
		// Subtract cancelled transactions
		var cancelledTotal int64
		row = s.DB.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(amount), 0) FROM transactions 
			WHERE card_id = ? 
			AND substr(transaction_date, 1, 10) >= ? 
			AND substr(transaction_date, 1, 10) <= ?
			AND is_cancelled = 1
		`, card.ID, startStr, endStr)
		row.Scan(&cancelledTotal)
		cardTotal -= cancelledTotal
		
		if cardTotal < 0 {
			cardTotal = 0
		}
		
		billingPeriods = append(billingPeriods, BillingPeriodJSON{
			CardName:  card.Name,
			CardID:    card.ID,
			StartDay:  card.BillingStartDay,
			EndDay:    card.BillingEndDay,
			StartDate: start.Format("01/02"),
			EndDate:   end.Format("01/02"),
			Total:     cardTotal,
		})
		totalThisMonth += cardTotal
	}
	
	// Filter to only show cards with non-zero totals
	var activeBillingPeriods []BillingPeriodJSON
	for _, bp := range billingPeriods {
		if bp.Total > 0 {
			activeBillingPeriods = append(activeBillingPeriods, bp)
		}
	}
	
	s.writeJSON(w, map[string]any{
		"total_this_month":     totalThisMonth,
		"current_month":        fmt.Sprintf("%d년 %02d월", year, month),
		"billing_periods":      activeBillingPeriods,
		"recent_transactions":  recentTxns,
	})
}

// getNextBillingMonth returns the billing month for a transaction
func getNextBillingMonth(txDate time.Time, startDay, endDay int) time.Time {
	year := txDate.Year()
	month := int(txDate.Month())
	day := txDate.Day()
	
	if startDay <= endDay {
		// Same month billing (e.g., 1~31)
		if day <= endDay {
			return time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.Local)
		}
		return time.Date(year, time.Month(month+2), 1, 0, 0, 0, 0, time.Local)
	}
	
	// Cross-month billing (e.g., 23~22)
	if day >= startDay {
		// Transaction in first part of billing period
		return time.Date(year, time.Month(month+2), 1, 0, 0, 0, 0, time.Local)
	} else if day <= endDay {
		// Transaction in second part of billing period
		return time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.Local)
	}
	// Outside billing period - shouldn't happen
	return time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.Local)
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
		IsCancelled       int64  `json:"is_cancelled"`
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
		IsCancelled:       req.IsCancelled,
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
		IsCancelled       int64  `json:"is_cancelled"`
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
		IsCancelled:       req.IsCancelled,
	})
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) HandleAPIDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, 400, "invalid id")
		return
	}
	
	// Clear foreign key reference in sms_logs first
	_, err = s.DB.ExecContext(r.Context(), "UPDATE sms_logs SET transaction_id = NULL WHERE transaction_id = ?", id)
	if err != nil {
		slog.Error("failed to clear sms_logs reference", "id", id, "error", err)
	}
	
	// Delete transaction
	_, err = s.DB.ExecContext(r.Context(), "DELETE FROM transactions WHERE id = ?", id)
	if err != nil {
		slog.Error("failed to delete transaction", "id", id, "error", err)
		s.writeError(w, 500, err.Error())
		return
	}
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

// HandleAPIWeeklyStats returns weekly spending statistics
// GET /api/weekly-stats?year=2026&month=3
func (s *Server) HandleAPIWeeklyStats(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	
	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)
	if year == 0 {
		year = time.Now().Year()
	}
	if month == 0 {
		month = int(time.Now().Month())
	}
	
	// Get all cards for billing period calculation
	q := dbgen.New(s.DB)
	cards, err := q.GetAllCards(r.Context())
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	
	// Calculate billing period for this month (결제예정월)
	// Find the earliest start and latest end across all cards
	var periodStart, periodEnd time.Time
	for i, card := range cards {
		start, end := calculateBillingPeriodForMonth(year, month, int(card.BillingStartDay), int(card.BillingEndDay))
		if i == 0 || start.Before(periodStart) {
			periodStart = start
		}
		if i == 0 || end.After(periodEnd) {
			periodEnd = end
		}
	}
	
	// Get all transactions in this period
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT t.id, t.card_id, t.amount, t.transaction_date, c.name as card_name,
		       c.billing_start_day, c.billing_end_day
		FROM transactions t
		JOIN cards c ON t.card_id = c.id
		WHERE substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) <= ?
		ORDER BY t.transaction_date
	`, periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"))
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()
	
	// Group by week (Friday to Thursday, ending Friday)
	type txData struct {
		CardID    int64
		CardName  string
		Amount    int64
		Date      time.Time
		StartDay  int
		EndDay    int
	}
	var transactions []txData
	for rows.Next() {
		var tx txData
		var dateStr string
		var txID int64
		if err := rows.Scan(&txID, &tx.CardID, &tx.Amount, &dateStr, &tx.CardName, &tx.StartDay, &tx.EndDay); err != nil {
			continue
		}
		tx.Date, _ = time.Parse("2006-01-02T15:04:05Z", dateStr)
		if tx.Date.IsZero() {
			tx.Date, _ = time.Parse("2006-01-02 15:04:05", dateStr)
		}
		if tx.Date.IsZero() {
			tx.Date, _ = time.Parse("2006-01-02 15:04:05 -0700 MST", dateStr)
		}
		if tx.Date.IsZero() {
			tx.Date, _ = time.Parse("2006-01-02 15:04:05 +0000 UTC", dateStr)
		}
		
		// Check if this transaction belongs to this billing month for its card
		cardStart, cardEnd := calculateBillingPeriodForMonth(year, month, tx.StartDay, tx.EndDay)
		if tx.Date.Before(cardStart) || tx.Date.After(cardEnd) {
			continue
		}
		transactions = append(transactions, tx)
	}
	
	// Generate weeks from period start to period end
	var weeks []WeeklyStats
	weekStart := periodStart
	// Adjust to previous Friday if not Friday
	for weekStart.Weekday() != time.Friday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	
	for weekStart.Before(periodEnd) {
		weekEnd := weekStart.AddDate(0, 0, 6) // Thursday
		if weekEnd.After(periodEnd) {
			weekEnd = periodEnd
		}
		
		week := WeeklyStats{
			WeekStart: weekStart.Format("2006-01-02"),
			WeekEnd:   weekEnd.Format("2006-01-02"),
			WeekLabel: weekStart.Format("01/02") + "~" + weekEnd.Format("01/02"),
			ByCard:    []CardWeekTotal{},
		}
		
		// Sum transactions for this week
		cardTotals := make(map[int64]*CardWeekTotal)
		for _, tx := range transactions {
			if !tx.Date.Before(weekStart) && !tx.Date.After(weekEnd.Add(24*time.Hour)) {
				week.Total += tx.Amount
				if _, ok := cardTotals[tx.CardID]; !ok {
					cardTotals[tx.CardID] = &CardWeekTotal{
						CardID:   tx.CardID,
						CardName: tx.CardName,
					}
				}
				cardTotals[tx.CardID].Total += tx.Amount
			}
		}
		
		for _, ct := range cardTotals {
			week.ByCard = append(week.ByCard, *ct)
		}
		
		weeks = append(weeks, week)
		weekStart = weekStart.AddDate(0, 0, 7)
	}
	
	s.writeJSON(w, map[string]interface{}{
		"year":         year,
		"month":        month,
		"period_start": periodStart.Format("2006-01-02"),
		"period_end":   periodEnd.Format("2006-01-02"),
		"weeks":        weeks,
	})
}

// calculateBillingPeriodForMonth calculates the billing period for a given month
func calculateBillingPeriodForMonth(year, month, startDay, endDay int) (time.Time, time.Time) {
	if startDay <= endDay {
		// Same month billing (e.g., 1~31)
		prevMonth := month - 1
		prevYear := year
		if prevMonth == 0 {
			prevMonth = 12
			prevYear--
		}
		start := time.Date(prevYear, time.Month(prevMonth), startDay, 0, 0, 0, 0, time.Local)
		end := time.Date(prevYear, time.Month(prevMonth), endDay, 23, 59, 59, 0, time.Local)
		return start, end
	}
	
	// Cross-month billing (e.g., 23~22)
	startMonth := month - 2
	startYear := year
	if startMonth <= 0 {
		startMonth += 12
		startYear--
	}
	endMonth := month - 1
	endYear := year
	if endMonth <= 0 {
		endMonth += 12
		endYear--
	}
	
	start := time.Date(startYear, time.Month(startMonth), startDay, 0, 0, 0, 0, time.Local)
	end := time.Date(endYear, time.Month(endMonth), endDay, 23, 59, 59, 0, time.Local)
	return start, end
}

// HandleAPIGetGoal returns monthly spending goal
// GET /api/goals?year=2026&month=3
func (s *Server) HandleAPIGetGoal(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	
	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)
	if year == 0 {
		year = time.Now().Year()
	}
	if month == 0 {
		month = int(time.Now().Month())
	}
	
	q := dbgen.New(s.DB)
	goal, err := q.GetMonthlyGoal(r.Context(), dbgen.GetMonthlyGoalParams{
		Year:  int64(year),
		Month: int64(month),
	})
	if err != nil {
		// No goal set
		s.writeJSON(w, map[string]interface{}{
			"year":          year,
			"month":         month,
			"target_amount": nil,
		})
		return
	}
	
	s.writeJSON(w, map[string]interface{}{
		"year":          goal.Year,
		"month":         goal.Month,
		"target_amount": goal.TargetAmount,
	})
}

// HandleAPISetGoal sets monthly spending goal
// POST /api/goals
func (s *Server) HandleAPISetGoal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Year         int   `json:"year"`
		Month        int   `json:"month"`
		TargetAmount int64 `json:"target_amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, "invalid request")
		return
	}
	
	if req.Year == 0 {
		req.Year = time.Now().Year()
	}
	if req.Month == 0 {
		req.Month = int(time.Now().Month())
	}
	
	q := dbgen.New(s.DB)
	goal, err := q.SetMonthlyGoal(r.Context(), dbgen.SetMonthlyGoalParams{
		Year:         int64(req.Year),
		Month:        int64(req.Month),
		TargetAmount: req.TargetAmount,
	})
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	
	s.writeJSON(w, goal)
}
