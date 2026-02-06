package srv

import (
	"context"
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

type EffectiveTransaction struct {
	ID                 int64     `json:"id"`
	CardID             int64     `json:"card_id"`
	CardName           string    `json:"card_name"`
	CategoryID         *int64    `json:"category_id"`
	CategoryName       string    `json:"category_name"`
	TransactionDate    time.Time `json:"transaction_date"`
	Description        string    `json:"description"`
	Amount             int64     `json:"amount"`
	IsInstallment      int64     `json:"is_installment"`
	InstallmentMonths  *int64    `json:"installment_months"`
	InstallmentCurrent int       `json:"installment_current"`
	IsCancelled        int64     `json:"is_cancelled"`
	OriginalDate       time.Time `json:"original_date"`
	Memo               *string   `json:"memo"`
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
	
	effectiveTxns, err := s.getEffectiveTransactions(ctx, year, month)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
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
		start, end := calculateBillingPeriodForMonth(year, month, int(card.BillingStartDay), int(card.BillingEndDay))
		
		var cardTotal int64
		for _, tx := range effectiveTxns {
			if tx.CardID == card.ID {
				cardTotal += tx.Amount
			}
		}
		
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

	// Sort transactions for display (latest first)
	for i := 0; i < len(effectiveTxns); i++ {
		for j := i + 1; j < len(effectiveTxns); j++ {
			if effectiveTxns[i].OriginalDate.Before(effectiveTxns[j].OriginalDate) {
				effectiveTxns[i], effectiveTxns[j] = effectiveTxns[j], effectiveTxns[i]
			}
		}
	}

	recentTxns := effectiveTxns
	if len(recentTxns) > 20 {
		recentTxns = recentTxns[:20]
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
		// Transaction in first part of billing period (e.g., Nov 27 -> Jan)
		return time.Date(year, time.Month(month+2), 1, 0, 0, 0, 0, time.Local)
	} else if day <= endDay {
		// Transaction in second part of billing period (e.g., Dec 10 -> Jan)
		return time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.Local)
	}
	// Outside billing period - shouldn't happen
	return time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.Local)
}

// Transactions API
func (s *Server) HandleAPIGetTransactions(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	
	if yearStr == "" || monthStr == "" {
		// Fallback to all transactions if no year/month provided for backward compatibility
		q := dbgen.New(s.DB)
		txns, err := q.GetAllTransactions(r.Context())
		if err != nil {
			s.writeError(w, 500, err.Error())
			return
		}
		s.writeJSON(w, map[string]any{"transactions": txns})
		return
	}
	
	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)
	
	effectiveTxns, err := s.getEffectiveTransactions(r.Context(), year, month)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	
	// Sort transactions by date descending
	for i := 0; i < len(effectiveTxns); i++ {
		for j := i + 1; j < len(effectiveTxns); j++ {
			if effectiveTxns[i].TransactionDate.Before(effectiveTxns[j].TransactionDate) {
				effectiveTxns[i], effectiveTxns[j] = effectiveTxns[j], effectiveTxns[i]
			}
		}
	}
	
	s.writeJSON(w, map[string]any{"transactions": effectiveTxns})
}

func (s *Server) HandleAPIGetTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	row := s.DB.QueryRowContext(r.Context(), `
		SELECT t.id, t.card_id, c.name, t.category_id, COALESCE(cat.name, ''),
			   t.transaction_date, t.description, t.amount, t.is_installment,
			   t.installment_months, t.installment_current, t.original_amount, t.is_cancelled, t.memo
		FROM transactions t
		JOIN cards c ON t.card_id = c.id
		LEFT JOIN categories cat ON t.category_id = cat.id
		WHERE t.id = ?
	`, id)
	
	var tx struct {
		ID                 int64   `json:"id"`
		CardID             int64   `json:"card_id"`
		CardName           string  `json:"card_name"`
		CategoryID         *int64  `json:"category_id"`
		CategoryName       string  `json:"category_name"`
		TransactionDate    string  `json:"transaction_date"`
		Description        string  `json:"description"`
		Amount             int64   `json:"amount"`
		IsInstallment      int64   `json:"is_installment"`
		InstallmentMonths  *int64  `json:"installment_months"`
		InstallmentCurrent *int64  `json:"installment_current"`
		OriginalAmount     *int64  `json:"original_amount"`
		IsCancelled        int64   `json:"is_cancelled"`
		Memo               *string `json:"memo"`
	}
	
	err := row.Scan(&tx.ID, &tx.CardID, &tx.CardName, &tx.CategoryID, &tx.CategoryName,
		&tx.TransactionDate, &tx.Description, &tx.Amount, &tx.IsInstallment,
		&tx.InstallmentMonths, &tx.InstallmentCurrent, &tx.OriginalAmount, &tx.IsCancelled, &tx.Memo)
	if err != nil {
		s.writeError(w, 404, "Not found")
		return
	}
	s.writeJSON(w, tx)
}

func (s *Server) HandleAPICreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CardID            int64   `json:"card_id"`
		TransactionDate   string  `json:"transaction_date"`
		Amount            int64   `json:"amount"`
		Description       string  `json:"description"`
		CategoryID        *int64  `json:"category_id"`
		IsInstallment     int64   `json:"is_installment"`
		InstallmentMonths *int64  `json:"installment_months"`
		InstallmentCurrent *int64 `json:"installment_current"`
		OriginalAmount    *int64  `json:"original_amount"`
		IsCancelled       int64   `json:"is_cancelled"`
		Memo              *string `json:"memo"`
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
	
	// Use raw SQL to support memo field
	result, err := s.DB.ExecContext(ctx, `
		INSERT INTO transactions (card_id, category_id, transaction_date, description, amount, is_installment, installment_months, installment_current, original_amount, is_cancelled, memo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, req.CardID, categoryID, txDate, req.Description, req.Amount, req.IsInstallment, req.InstallmentMonths, req.InstallmentCurrent, req.OriginalAmount, req.IsCancelled, req.Memo)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	txID, _ := result.LastInsertId()
	tx := map[string]interface{}{"id": txID}
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
		CardID            int64   `json:"card_id"`
		TransactionDate   string  `json:"transaction_date"`
		Amount            int64   `json:"amount"`
		Description       string  `json:"description"`
		CategoryID        *int64  `json:"category_id"`
		IsInstallment     int64   `json:"is_installment"`
		InstallmentMonths *int64  `json:"installment_months"`
		InstallmentCurrent *int64 `json:"installment_current"`
		OriginalAmount    *int64  `json:"original_amount"`
		IsCancelled       int64   `json:"is_cancelled"`
		Memo              *string `json:"memo"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}
	
	txDate, _ := time.Parse("2006-01-02", req.TransactionDate)
	
	_, err := s.DB.ExecContext(r.Context(), `
		UPDATE transactions SET 
			card_id = ?, category_id = ?, transaction_date = ?, description = ?, 
			amount = ?, is_installment = ?, installment_months = ?, 
			installment_current = ?, original_amount = ?, is_cancelled = ?, memo = ?
		WHERE id = ?
	`, req.CardID, req.CategoryID, txDate, req.Description, req.Amount, req.IsInstallment, 
		req.InstallmentMonths, req.InstallmentCurrent, req.OriginalAmount, req.IsCancelled, req.Memo, id)
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
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	
	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)
	if year == 0 {
		year = time.Now().Year()
	}
	if month == 0 {
		now := time.Now()
		// Default to current month if it's not the 1st? 
		// Actually user said "시작페이지 기준도 마찬가지로 매달 1일을 기준으로 다음달 결제 예정금액을 기본페이지로 한다"
		// This means if it's Feb 1st, show March. Today is Feb 3rd, so show March.
		next := now.AddDate(0, 1, 0)
		year = next.Year()
		month = int(next.Month())
	}
	
	ctx := r.Context()
	
	// Get stats for current period
	txns, err := s.getEffectiveTransactions(ctx, year, month)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	
	// Get stats for last period (for comparison)
	lastYear := year
	lastMonth := month - 1
	if lastMonth == 0 {
		lastMonth = 12
		lastYear--
	}
	lastTxns, _ := s.getEffectiveTransactions(ctx, lastYear, lastMonth)
	
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
	
	// Aggregation for current month
	byCategoryMap := make(map[string]*CategoryStatJSON)
	byCardMap := make(map[int64]*CardStatJSON)
	var total int64
	
	for _, tx := range txns {
		total += tx.Amount
		
		if _, ok := byCategoryMap[tx.CategoryName]; !ok {
			byCategoryMap[tx.CategoryName] = &CategoryStatJSON{CategoryName: tx.CategoryName}
		}
		byCategoryMap[tx.CategoryName].TotalAmount += tx.Amount
		byCategoryMap[tx.CategoryName].Count++
		
		if _, ok := byCardMap[tx.CardID]; !ok {
			byCardMap[tx.CardID] = &CardStatJSON{CardName: tx.CardName, CardID: tx.CardID}
		}
		byCardMap[tx.CardID].TotalAmount += tx.Amount
		byCardMap[tx.CardID].Count++
	}
	
	var totalLastMonth int64
	for _, tx := range lastTxns {
		totalLastMonth += tx.Amount
	}
	
	// Convert maps to sorted slices
	var byCategory []CategoryStatJSON
	for _, v := range byCategoryMap {
		byCategory = append(byCategory, *v)
	}
	// Sort by amount descending
	for i := 0; i < len(byCategory); i++ {
		for j := i + 1; j < len(byCategory); j++ {
			if byCategory[i].TotalAmount < byCategory[j].TotalAmount {
				byCategory[i], byCategory[j] = byCategory[j], byCategory[i]
			}
		}
	}
	
	var byCard []CardStatJSON
	for _, v := range byCardMap {
		byCard = append(byCard, *v)
	}
	for i := 0; i < len(byCard); i++ {
		for j := i + 1; j < len(byCard); j++ {
			if byCard[i].TotalAmount < byCard[j].TotalAmount {
				byCard[i], byCard[j] = byCard[j], byCard[i]
			}
		}
	}
	
	s.writeJSON(w, map[string]any{
		"year":             year,
		"month":            month,
		"by_category":      byCategory,
		"by_card":          byCard,
		"total":            total,
		"total_last_month": totalLastMonth,
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
	
	result := s.parseSMSWithDB(r.Context(), req.Text)
	s.writeJSON(w, map[string]interface{}{
		"success":     true,
		"transaction": result,
	})
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
		next := time.Now().AddDate(0, 1, 0)
		year = next.Year()
		month = int(next.Month())
	}
	
	q := dbgen.New(s.DB)
	cards, err := q.GetAllCards(r.Context())
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	
	// Get all effective transactions for this month
	txns, err := s.getEffectiveTransactions(r.Context(), year, month)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}

	// Calculate overall period across all cards for week generation
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
	
	// Generate weeks from period start to period end
	var weeks []WeeklyStats
	weekStart := periodStart
	// Adjust to previous Monday (or Friday as before?) 
	// The user mention "Friday to Thursday" in comments, but let's stick to what's natural or what was there.
	// Actually the user didn't specify the week start day, but let's use Monday as standard.
	// Wait, the previous code used Friday. Let's keep it if that's what was intended.
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	
	for weekStart.Before(periodEnd) {
		weekEnd := weekStart.AddDate(0, 0, 6) // Sunday
		
		week := WeeklyStats{
			WeekStart: weekStart.Format("2006-01-02"),
			WeekEnd:   weekEnd.Format("2006-01-02"),
			WeekLabel: weekStart.Format("01/02") + "~" + weekEnd.Format("01/02"),
			ByCard:    []CardWeekTotal{},
		}
		
		cardTotals := make(map[int64]*CardWeekTotal)
		for _, tx := range txns {
			// Use TransactionDate which we set to the start of billing period for installments
			// But for weekly stats, installments should maybe be spread out?
			// Actually, for installments, we don't have a specific "week" within the billing month.
			// Let's just put all installments in the first week of the period.
			
			if !tx.TransactionDate.Before(weekStart) && !tx.TransactionDate.After(weekEnd.Add(23*time.Hour + 59*time.Minute)) {
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
	// Example: Jan billing covers Nov 23 ~ Dec 22
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

func parseTime(s string) time.Time {
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05 +0900 KST",
		"2006-01-02 15:04:05 +0000 UTC",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			return t
		}
	}
	// Fallback: try to parse just the date and time part
	if len(s) >= 19 {
		t, err := time.Parse("2006-01-02 15:04:05", s[:19])
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func (s *Server) getEffectiveTransactions(ctx context.Context, year, month int) ([]EffectiveTransaction, error) {
	q := dbgen.New(s.DB)
	cards, err := q.GetAllCards(ctx)
	if err != nil {
		return nil, err
	}

	var results []EffectiveTransaction

	for _, card := range cards {
		start, end := calculateBillingPeriodForMonth(year, month, int(card.BillingStartDay), int(card.BillingEndDay))
		startStr := start.Format("2006-01-02")
		endStr := end.Format("2006-01-02")

		// 1. Regular transactions (is_installment = 0, is_cancelled = 0)
		rows, err := s.DB.QueryContext(ctx, `
SELECT t.id, t.card_id, c.name, t.category_id, COALESCE(cat.name, '미분류'), 
       t.transaction_date, t.description, t.amount, t.is_installment, 
       t.installment_months, t.is_cancelled, t.memo
FROM transactions t
JOIN cards c ON t.card_id = c.id
LEFT JOIN categories cat ON t.category_id = cat.id
WHERE t.card_id = ? 
AND substr(t.transaction_date, 1, 10) >= ? 
AND substr(t.transaction_date, 1, 10) <= ?
AND t.is_cancelled = 0
AND t.is_installment = 0
`, card.ID, startStr, endStr)
		if err == nil {
			for rows.Next() {
				var et EffectiveTransaction
				var txDateStr string
				if err := rows.Scan(&et.ID, &et.CardID, &et.CardName, &et.CategoryID, &et.CategoryName,
					&txDateStr, &et.Description, &et.Amount, &et.IsInstallment,
					&et.InstallmentMonths, &et.IsCancelled, &et.Memo); err == nil {
					et.TransactionDate = parseTime(txDateStr)
					et.OriginalDate = et.TransactionDate
					results = append(results, et)
				}
			}
			rows.Close()
		}

		// 2. Installments (is_installment = 1, is_cancelled = 0)
		rowsInst, err := s.DB.QueryContext(ctx, `
SELECT t.id, t.card_id, c.name, t.category_id, COALESCE(cat.name, '미분류'), 
       t.transaction_date, t.description, t.amount, t.is_installment, 
       t.installment_months, t.is_cancelled, t.memo
FROM transactions t
JOIN cards c ON t.card_id = c.id
LEFT JOIN categories cat ON t.category_id = cat.id
WHERE t.card_id = ? 
AND t.is_cancelled = 0
AND t.is_installment = 1
`, card.ID)
		if err == nil {
			for rowsInst.Next() {
				var et EffectiveTransaction
				var txDateStr string
				var months int64
				if err := rowsInst.Scan(&et.ID, &et.CardID, &et.CardName, &et.CategoryID, &et.CategoryName,
					&txDateStr, &et.Description, &et.Amount, &et.IsInstallment,
					&months, &et.IsCancelled, &et.Memo); err == nil {

					et.InstallmentMonths = &months
					et.OriginalDate = parseTime(txDateStr)

					txBillingMonth := getNextBillingMonth(et.OriginalDate, int(card.BillingStartDay), int(card.BillingEndDay))
					targetBillingMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)

					monthsDiff := (targetBillingMonth.Year()-txBillingMonth.Year())*12 + int(targetBillingMonth.Month()-txBillingMonth.Month())
					installmentNum := monthsDiff + 1

					if installmentNum >= 1 && installmentNum <= int(months) {
						et.InstallmentCurrent = installmentNum
						// Split amount
						et.Amount = et.Amount / months
						
						// Keep original date so it can be grouped with other transactions on the same day.
						// The filtering is already handled by installmentNum logic.
						et.TransactionDate = et.OriginalDate
						
						results = append(results, et)
					}
				}
			}
			rowsInst.Close()
		}

		// 3. Subtract cancelled transactions (is_cancelled = 1)
		rowsCan, err := s.DB.QueryContext(ctx, `
SELECT t.id, t.card_id, c.name, t.category_id, COALESCE(cat.name, '미분류'), 
       t.transaction_date, t.description, t.amount, t.is_installment, 
       t.installment_months, t.is_cancelled, t.memo
FROM transactions t
JOIN cards c ON t.card_id = c.id
LEFT JOIN categories cat ON t.category_id = cat.id
WHERE t.card_id = ? 
AND substr(t.transaction_date, 1, 10) >= ? 
AND substr(t.transaction_date, 1, 10) <= ?
AND t.is_cancelled = 1
`, card.ID, startStr, endStr)
		if err == nil {
			for rowsCan.Next() {
				var et EffectiveTransaction
				var txDateStr string
				if err := rowsCan.Scan(&et.ID, &et.CardID, &et.CardName, &et.CategoryID, &et.CategoryName,
					&txDateStr, &et.Description, &et.Amount, &et.IsInstallment,
					&et.InstallmentMonths, &et.IsCancelled, &et.Memo); err == nil {
					et.TransactionDate = parseTime(txDateStr)
					et.OriginalDate = et.TransactionDate
					et.Amount = -et.Amount // Negative for cancellation
					results = append(results, et)
				}
			}
			rowsCan.Close()
		}
	}

	return results, nil
}
