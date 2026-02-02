package srv

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"srv.exe.dev/db/dbgen"
)

type DashboardData struct {
	Cards           []dbgen.Card
	Categories      []dbgen.Category
	RecentTxns      []dbgen.GetAllTransactionsRow
	CardSummaries   []CardSummary
	TotalThisMonth  int64
	CurrentMonth    string
	BillingPeriods  []BillingPeriod
}

type CardSummary struct {
	CardName    string
	CardID      int64
	Total       int64
	Count       int64
	StartDate   string
	EndDate     string
}

type BillingPeriod struct {
	CardName  string
	CardID    int64
	StartDay  int64
	EndDay    int64
	StartDate string
	EndDate   string
	Total     int64
}

func (s *Server) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	q := dbgen.New(s.DB)
	ctx := r.Context()
	
	cards, _ := q.GetAllCards(ctx)
	categories, _ := q.GetAllCategories(ctx)
	txns, _ := q.GetAllTransactions(ctx)
	
	// Limit recent transactions
	recentTxns := txns
	if len(recentTxns) > 10 {
		recentTxns = recentTxns[:10]
	}
	
	// Calculate billing periods for each card
	now := time.Now()
	var billingPeriods []BillingPeriod
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
			billingPeriods = append(billingPeriods, BillingPeriod{
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
	
	data := DashboardData{
		Cards:          cards,
		Categories:     categories,
		RecentTxns:     recentTxns,
		TotalThisMonth: totalThisMonth,
		CurrentMonth:   now.Format("2006년 01월"),
		BillingPeriods: billingPeriods,
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderTemplate(w, "dashboard.html", data); err != nil {
		slog.Warn("render template", "error", err)
		http.Error(w, err.Error(), 500)
	}
}

func calculateBillingPeriod(now time.Time, startDay, endDay int) (time.Time, time.Time) {
	year, month, day := now.Date()
	
	var start, end time.Time
	
	if startDay <= endDay {
		// Same month billing (e.g., 1st to 31st)
		start = time.Date(year, month, startDay, 0, 0, 0, 0, now.Location())
		end = time.Date(year, month, endDay, 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	} else {
		// Cross-month billing (e.g., 22nd to 21st of next month)
		if day >= startDay {
			// We're in the current billing period
			start = time.Date(year, month, startDay, 0, 0, 0, 0, now.Location())
			end = time.Date(year, month+1, endDay, 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
		} else {
			// We're in the previous billing period
			start = time.Date(year, month-1, startDay, 0, 0, 0, 0, now.Location())
			end = time.Date(year, month, endDay, 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
		}
	}
	
	return start, end
}

func (s *Server) HandleTransactions(w http.ResponseWriter, r *http.Request) {
	q := dbgen.New(s.DB)
	ctx := r.Context()
	
	txns, _ := q.GetAllTransactions(ctx)
	cards, _ := q.GetAllCards(ctx)
	
	data := struct {
		Transactions []dbgen.GetAllTransactionsRow
		Cards        []dbgen.Card
	}{txns, cards}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderTemplate(w, "transactions.html", data); err != nil {
		slog.Warn("render template", "error", err)
		http.Error(w, err.Error(), 500)
	}
}

func (s *Server) HandleTransactionForm(w http.ResponseWriter, r *http.Request) {
	q := dbgen.New(s.DB)
	ctx := r.Context()
	
	cards, _ := q.GetAllCards(ctx)
	categories, _ := q.GetAllCategories(ctx)
	
	data := struct {
		Cards      []dbgen.Card
		Categories []dbgen.Category
	}{cards, categories}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderTemplate(w, "transaction_form.html", data); err != nil {
		slog.Warn("render template", "error", err)
		http.Error(w, err.Error(), 500)
	}
}

func (s *Server) HandleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	q := dbgen.New(s.DB)
	ctx := r.Context()
	
	cardID, _ := strconv.ParseInt(r.FormValue("card_id"), 10, 64)
	amountStr := strings.ReplaceAll(r.FormValue("amount"), ",", "")
	amount, _ := strconv.ParseInt(amountStr, 10, 64)
	dateStr := r.FormValue("transaction_date")
	txDate, _ := time.Parse("2006-01-02", dateStr)
	description := r.FormValue("description")
	isInstallment := r.FormValue("is_installment") == "1"
	installmentMonths, _ := strconv.ParseInt(r.FormValue("installment_months"), 10, 64)
	installmentCurrent, _ := strconv.ParseInt(r.FormValue("installment_current"), 10, 64)
	originalAmountStr := strings.ReplaceAll(r.FormValue("original_amount"), ",", "")
	originalAmount, _ := strconv.ParseInt(originalAmountStr, 10, 64)
	
	// Auto-categorize
	categoryID := s.autoCategorize(ctx, q, description)
	
	params := dbgen.CreateTransactionParams{
		CardID:          cardID,
		CategoryID:      categoryID,
		TransactionDate: txDate,
		Description:     description,
		Amount:          amount,
		IsInstallment:   0,
	}
	
	if isInstallment {
		params.IsInstallment = 1
		params.InstallmentMonths = &installmentMonths
		params.InstallmentCurrent = &installmentCurrent
		params.OriginalAmount = &originalAmount
	}
	
	_, err := q.CreateTransaction(ctx, params)
	if err != nil {
		slog.Warn("create transaction", "error", err)
		http.Error(w, err.Error(), 500)
		return
	}
	
	http.Redirect(w, r, "/transactions", http.StatusSeeOther)
}

func (s *Server) autoCategorize(ctx context.Context, q *dbgen.Queries, description string) *int64 {
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
	
	// Return '기타' category if exists
	for _, cat := range categories {
		if cat.Name == "기타" {
			return &cat.ID
		}
	}
	
	return nil
}

func (s *Server) HandleEditTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	q := dbgen.New(s.DB)
	ctx := r.Context()
	
	tx, err := q.GetTransaction(ctx, id)
	if err != nil {
		http.Error(w, "Transaction not found", 404)
		return
	}
	
	cards, _ := q.GetAllCards(ctx)
	categories, _ := q.GetAllCategories(ctx)
	
	data := struct {
		Transaction dbgen.Transaction
		Cards       []dbgen.Card
		Categories  []dbgen.Category
	}{tx, cards, categories}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderTemplate(w, "transaction_edit.html", data); err != nil {
		slog.Warn("render template", "error", err)
		http.Error(w, err.Error(), 500)
	}
}

func (s *Server) HandleUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	q := dbgen.New(s.DB)
	ctx := r.Context()
	
	cardID, _ := strconv.ParseInt(r.FormValue("card_id"), 10, 64)
	categoryIDStr := r.FormValue("category_id")
	var categoryID *int64
	if categoryIDStr != "" {
		cid, _ := strconv.ParseInt(categoryIDStr, 10, 64)
		categoryID = &cid
	}
	amountStr := strings.ReplaceAll(r.FormValue("amount"), ",", "")
	amount, _ := strconv.ParseInt(amountStr, 10, 64)
	dateStr := r.FormValue("transaction_date")
	txDate, _ := time.Parse("2006-01-02", dateStr)
	description := r.FormValue("description")
	isInstallment := r.FormValue("is_installment") == "1"
	
	params := dbgen.UpdateTransactionParams{
		ID:              id,
		CardID:          cardID,
		CategoryID:      categoryID,
		TransactionDate: txDate,
		Description:     description,
		Amount:          amount,
		IsInstallment:   0,
	}
	
	if isInstallment {
		params.IsInstallment = 1
		installmentMonths, _ := strconv.ParseInt(r.FormValue("installment_months"), 10, 64)
		installmentCurrent, _ := strconv.ParseInt(r.FormValue("installment_current"), 10, 64)
		originalAmountStr := strings.ReplaceAll(r.FormValue("original_amount"), ",", "")
		originalAmount, _ := strconv.ParseInt(originalAmountStr, 10, 64)
		params.InstallmentMonths = &installmentMonths
		params.InstallmentCurrent = &installmentCurrent
		params.OriginalAmount = &originalAmount
	}
	
	err := q.UpdateTransaction(ctx, params)
	if err != nil {
		slog.Warn("update transaction", "error", err)
		http.Error(w, err.Error(), 500)
		return
	}
	
	http.Redirect(w, r, "/transactions", http.StatusSeeOther)
}

func (s *Server) HandleDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	q := dbgen.New(s.DB)
	q.DeleteTransaction(r.Context(), id)
	
	http.Redirect(w, r, "/transactions", http.StatusSeeOther)
}
