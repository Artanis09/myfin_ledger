package srv

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// ========== Settings API ==========

func (s *Server) HandleAPIGetSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), "SELECT key, value FROM app_settings")
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		settings[k] = v
	}
	s.writeJSON(w, settings)
}

func (s *Server) HandleAPIUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}

	for k, v := range req {
		_, err := s.DB.ExecContext(r.Context(),
			`INSERT INTO app_settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`, k, v)
		if err != nil {
			s.writeError(w, 500, err.Error())
			return
		}
	}
	s.writeJSON(w, map[string]string{"status": "ok"})
}

// ========== Asset Types API ==========

type AssetType struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	IsCard       int64   `json:"is_card"`
	CardID       *int64  `json:"card_id"`
	IsRecurring  int64   `json:"is_recurring"`
	DisplayOrder int64   `json:"display_order"`
	IsSystem     int64   `json:"is_system"`
}

func (s *Server) HandleAPIGetAssetTypes(w http.ResponseWriter, r *http.Request) {
	typeFilter := r.URL.Query().Get("type")
	
	var rows *sql.Rows
	var err error
	if typeFilter != "" {
		rows, err = s.DB.QueryContext(r.Context(),
			"SELECT id, name, type, is_card, card_id, is_recurring, display_order, is_system FROM asset_types WHERE type = ? ORDER BY display_order, name", typeFilter)
	} else {
		rows, err = s.DB.QueryContext(r.Context(),
			"SELECT id, name, type, is_card, card_id, is_recurring, display_order, is_system FROM asset_types ORDER BY type, display_order, name")
	}
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	var assets []AssetType
	for rows.Next() {
		var a AssetType
		rows.Scan(&a.ID, &a.Name, &a.Type, &a.IsCard, &a.CardID, &a.IsRecurring, &a.DisplayOrder, &a.IsSystem)
		assets = append(assets, a)
	}
	s.writeJSON(w, map[string]any{"asset_types": assets})
}

func (s *Server) HandleAPICreateAssetType(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Type         string `json:"type"`
		IsRecurring  int64  `json:"is_recurring"`
		DisplayOrder int64  `json:"display_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}

	result, err := s.DB.ExecContext(r.Context(),
		`INSERT INTO asset_types (name, type, is_card, is_recurring, display_order, is_system) VALUES (?, ?, 0, ?, ?, 0)`,
		req.Name, req.Type, req.IsRecurring, req.DisplayOrder)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	id, _ := result.LastInsertId()
	s.writeJSON(w, map[string]int64{"id": id})
}

func (s *Server) HandleAPIDeleteAssetType(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	
	// 시스템 자산은 삭제 불가
	_, err := s.DB.ExecContext(r.Context(), "DELETE FROM asset_types WHERE id = ? AND is_system = 0", id)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]string{"status": "ok"})
}

// ========== Income Categories API ==========

type IncomeCategory struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	DisplayOrder int64  `json:"display_order"`
}

func (s *Server) HandleAPIGetIncomeCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(),
		"SELECT id, name, display_order FROM income_categories ORDER BY display_order, name")
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	var cats []IncomeCategory
	for rows.Next() {
		var c IncomeCategory
		rows.Scan(&c.ID, &c.Name, &c.DisplayOrder)
		cats = append(cats, c)
	}
	s.writeJSON(w, map[string]any{"income_categories": cats})
}

func (s *Server) HandleAPICreateIncomeCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		DisplayOrder int64  `json:"display_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}

	result, err := s.DB.ExecContext(r.Context(),
		"INSERT INTO income_categories (name, display_order) VALUES (?, ?)", req.Name, req.DisplayOrder)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	id, _ := result.LastInsertId()
	s.writeJSON(w, map[string]int64{"id": id})
}

func (s *Server) HandleAPIUpdateIncomeCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	var req struct {
		Name         string `json:"name"`
		DisplayOrder int64  `json:"display_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}

	_, err := s.DB.ExecContext(r.Context(),
		"UPDATE income_categories SET name = ?, display_order = ? WHERE id = ?", req.Name, req.DisplayOrder, id)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) HandleAPIDeleteIncomeCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	_, err := s.DB.ExecContext(r.Context(), "DELETE FROM income_categories WHERE id = ?", id)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]string{"status": "ok"})
}

// ========== Transactions V2 API (수입/지출 통합) ==========

type TransactionV2 struct {
	ID                 int64   `json:"id"`
	TxType             string  `json:"tx_type"` // 'expense' or 'income'
	AssetTypeID        *int64  `json:"asset_type_id"`
	AssetTypeName      string  `json:"asset_type_name"`
	CardID             *int64  `json:"card_id"`
	CategoryID         *int64  `json:"category_id"`
	CategoryName       string  `json:"category_name"`
	IncomeCategoryID   *int64  `json:"income_category_id"`
	IncomeCategoryName string  `json:"income_category_name"`
	TransactionDate    string  `json:"transaction_date"`
	Description        string  `json:"description"`
	Amount             int64   `json:"amount"`
	IsInstallment      int64   `json:"is_installment"`
	InstallmentMonths  *int64  `json:"installment_months"`
	IsCancelled        int64   `json:"is_cancelled"`
	Memo               *string `json:"memo"`
	RecurringID        *int64  `json:"recurring_schedule_id"`
}

func (s *Server) HandleAPIGetTransactionsV2(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	txType := r.URL.Query().Get("type") // 'expense', 'income', or '' for all

	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)
	if year == 0 {
		year = time.Now().Year()
	}
	if month == 0 {
		month = int(time.Now().Month())
	}

	// 가계부 관리 기간 설정 가져오기
	startDay, endDay := s.getLedgerPeriod(r.Context())
	startDate, endDate := calculateLedgerPeriod(year, month, startDay, endDay)

	query := `
		SELECT t.id, t.tx_type, t.asset_type_id, COALESCE(at.name, ''), t.card_id,
		       t.category_id, COALESCE(cat.name, ''), t.income_category_id, COALESCE(ic.name, ''),
		       t.transaction_date, t.description, t.amount, t.is_installment,
		       t.installment_months, t.is_cancelled, t.memo, t.recurring_schedule_id
		FROM transactions t
		LEFT JOIN asset_types at ON t.asset_type_id = at.id
		LEFT JOIN categories cat ON t.category_id = cat.id
		LEFT JOIN income_categories ic ON t.income_category_id = ic.id
		WHERE substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) <= ?`
	
	args := []any{startDate, endDate}
	if txType != "" {
		query += " AND t.tx_type = ?"
		args = append(args, txType)
	}
	query += " ORDER BY t.transaction_date DESC"

	rows, err := s.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	var txns []TransactionV2
	for rows.Next() {
		var tx TransactionV2
		rows.Scan(&tx.ID, &tx.TxType, &tx.AssetTypeID, &tx.AssetTypeName, &tx.CardID,
			&tx.CategoryID, &tx.CategoryName, &tx.IncomeCategoryID, &tx.IncomeCategoryName,
			&tx.TransactionDate, &tx.Description, &tx.Amount, &tx.IsInstallment,
			&tx.InstallmentMonths, &tx.IsCancelled, &tx.Memo, &tx.RecurringID)
		txns = append(txns, tx)
	}
	s.writeJSON(w, map[string]any{"transactions": txns})
}

func (s *Server) HandleAPICreateTransactionV2(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TxType            string  `json:"tx_type"`
		AssetTypeID       *int64  `json:"asset_type_id"`
		TransactionDate   string  `json:"transaction_date"`
		Amount            int64   `json:"amount"`
		Description       string  `json:"description"`
		CategoryID        *int64  `json:"category_id"`
		IncomeCategoryID  *int64  `json:"income_category_id"`
		IsInstallment     int64   `json:"is_installment"`
		InstallmentMonths *int64  `json:"installment_months"`
		IsCancelled       int64   `json:"is_cancelled"`
		Memo              *string `json:"memo"`
		IsRecurring       int64   `json:"is_recurring"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}

	if req.TxType == "" {
		req.TxType = "expense"
	}

	// asset_type에서 card_id 가져오기
	var cardID *int64
	if req.AssetTypeID != nil {
		row := s.DB.QueryRowContext(r.Context(), "SELECT card_id FROM asset_types WHERE id = ?", *req.AssetTypeID)
		var cid sql.NullInt64
		if err := row.Scan(&cid); err == nil && cid.Valid {
			cardID = &cid.Int64
		}
	}

	result, err := s.DB.ExecContext(r.Context(), `
		INSERT INTO transactions (
			tx_type, asset_type_id, card_id, category_id, income_category_id,
			transaction_date, description, amount, is_installment, installment_months,
			is_cancelled, memo, is_recurring
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.TxType, req.AssetTypeID, cardID, req.CategoryID, req.IncomeCategoryID,
		req.TransactionDate, req.Description, req.Amount, req.IsInstallment,
		req.InstallmentMonths, req.IsCancelled, req.Memo, req.IsRecurring)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	id, _ := result.LastInsertId()

	s.writeJSON(w, map[string]int64{"id": id})
}

// HandleAPIGetTransactionV2 - 단일 거래 조회
func (s *Server) HandleAPIGetTransactionV2(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	row := s.DB.QueryRowContext(r.Context(), `
		SELECT t.id, t.tx_type, t.asset_type_id, COALESCE(at.name, ''), t.card_id,
		       t.category_id, COALESCE(cat.name, ''), t.income_category_id, COALESCE(ic.name, ''),
		       t.transaction_date, t.description, t.amount, t.is_installment,
		       t.installment_months, t.is_cancelled, t.memo, COALESCE(t.is_recurring, 0)
		FROM transactions t
		LEFT JOIN asset_types at ON t.asset_type_id = at.id
		LEFT JOIN categories cat ON t.category_id = cat.id
		LEFT JOIN income_categories ic ON t.income_category_id = ic.id
		WHERE t.id = ?`, id)

	var tx struct {
		ID                 int64   `json:"id"`
		TxType             string  `json:"tx_type"`
		AssetTypeID        *int64  `json:"asset_type_id"`
		AssetTypeName      string  `json:"asset_type_name"`
		CardID             *int64  `json:"card_id"`
		CategoryID         *int64  `json:"category_id"`
		CategoryName       string  `json:"category_name"`
		IncomeCategoryID   *int64  `json:"income_category_id"`
		IncomeCategoryName string  `json:"income_category_name"`
		TransactionDate    string  `json:"transaction_date"`
		Description        string  `json:"description"`
		Amount             int64   `json:"amount"`
		IsInstallment      int64   `json:"is_installment"`
		InstallmentMonths  *int64  `json:"installment_months"`
		IsCancelled        int64   `json:"is_cancelled"`
		Memo               *string `json:"memo"`
		IsRecurring        int64   `json:"is_recurring"`
	}

	err := row.Scan(&tx.ID, &tx.TxType, &tx.AssetTypeID, &tx.AssetTypeName, &tx.CardID,
		&tx.CategoryID, &tx.CategoryName, &tx.IncomeCategoryID, &tx.IncomeCategoryName,
		&tx.TransactionDate, &tx.Description, &tx.Amount, &tx.IsInstallment,
		&tx.InstallmentMonths, &tx.IsCancelled, &tx.Memo, &tx.IsRecurring)
	if err != nil {
		s.writeError(w, 404, "Not found")
		return
	}
	s.writeJSON(w, tx)
}

// HandleAPIUpdateTransactionV2 - 거래 수정
func (s *Server) HandleAPIUpdateTransactionV2(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	var req struct {
		TxType            string  `json:"tx_type"`
		AssetTypeID       *int64  `json:"asset_type_id"`
		TransactionDate   string  `json:"transaction_date"`
		Amount            int64   `json:"amount"`
		Description       string  `json:"description"`
		CategoryID        *int64  `json:"category_id"`
		IncomeCategoryID  *int64  `json:"income_category_id"`
		IsInstallment     int64   `json:"is_installment"`
		InstallmentMonths *int64  `json:"installment_months"`
		IsCancelled       int64   `json:"is_cancelled"`
		Memo              *string `json:"memo"`
		IsRecurring       int64   `json:"is_recurring"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 400, err.Error())
		return
	}

	// asset_type에서 card_id 가져오기
	var cardID *int64
	if req.AssetTypeID != nil {
		row := s.DB.QueryRowContext(r.Context(), "SELECT card_id FROM asset_types WHERE id = ?", *req.AssetTypeID)
		var cid sql.NullInt64
		if err := row.Scan(&cid); err == nil && cid.Valid {
			cardID = &cid.Int64
		}
	}

	_, err := s.DB.ExecContext(r.Context(), `
		UPDATE transactions SET
			tx_type = ?, asset_type_id = ?, card_id = ?, category_id = ?, income_category_id = ?,
			transaction_date = ?, description = ?, amount = ?, is_installment = ?,
			installment_months = ?, is_cancelled = ?, memo = ?, is_recurring = ?
		WHERE id = ?`,
		req.TxType, req.AssetTypeID, cardID, req.CategoryID, req.IncomeCategoryID,
		req.TransactionDate, req.Description, req.Amount, req.IsInstallment,
		req.InstallmentMonths, req.IsCancelled, req.Memo, req.IsRecurring, id)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]string{"status": "ok"})
}

// HandleAPIDeleteTransactionV2 - 거래 삭제
func (s *Server) HandleAPIDeleteTransactionV2(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	// sms_logs 참조 해제
	s.DB.ExecContext(r.Context(), "UPDATE sms_logs SET transaction_id = NULL WHERE transaction_id = ?", id)

	_, err := s.DB.ExecContext(r.Context(), "DELETE FROM transactions WHERE id = ?", id)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]string{"status": "ok"})
}

// 가계부 관리 기간 설정 가져오기
func (s *Server) getLedgerPeriod(ctx context.Context) (int, int) {
	var startDay, endDay int = 1, 31
	s.DB.QueryRowContext(ctx, "SELECT value FROM app_settings WHERE key = 'ledger_period_start_day'").Scan(&startDay)
	s.DB.QueryRowContext(ctx, "SELECT value FROM app_settings WHERE key = 'ledger_period_end_day'").Scan(&endDay)
	return startDay, endDay
}

// 가계부 관리 기간 계산
func calculateLedgerPeriod(year, month, startDay, endDay int) (string, string) {
	if startDay <= endDay {
		// 같은 달 내 (1일~31일)
		start := time.Date(year, time.Month(month), startDay, 0, 0, 0, 0, time.Local)
		end := time.Date(year, time.Month(month), endDay, 0, 0, 0, 0, time.Local)
		return start.Format("2006-01-02"), end.Format("2006-01-02")
	}
	// 월 경계 (5일~익월 4일)
	prevMonth := month - 1
	prevYear := year
	if prevMonth == 0 {
		prevMonth = 12
		prevYear--
	}
	start := time.Date(prevYear, time.Month(prevMonth), startDay, 0, 0, 0, 0, time.Local)
	end := time.Date(year, time.Month(month), endDay, 0, 0, 0, 0, time.Local)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

// ========== Statistics V2 API (가계부 기준 / 카드 기준 분리) ==========

// 가계부 통계 (가계부 관리 기간 기준)
func (s *Server) HandleAPIStatisticsLedger(w http.ResponseWriter, r *http.Request) {
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

	startDay, endDay := s.getLedgerPeriod(r.Context())
	startDate, endDate := calculateLedgerPeriod(year, month, startDay, endDay)

	// 총 수입
	var totalIncome int64
	s.DB.QueryRowContext(r.Context(), `
		SELECT COALESCE(SUM(amount), 0) FROM transactions 
		WHERE tx_type = 'income' AND is_cancelled = 0
		AND substr(transaction_date, 1, 10) >= ? AND substr(transaction_date, 1, 10) <= ?`,
		startDate, endDate).Scan(&totalIncome)

	// 총 지출
	var totalExpense int64
	s.DB.QueryRowContext(r.Context(), `
		SELECT COALESCE(SUM(amount), 0) FROM transactions 
		WHERE tx_type = 'expense' AND is_cancelled = 0
		AND substr(transaction_date, 1, 10) >= ? AND substr(transaction_date, 1, 10) <= ?`,
		startDate, endDate).Scan(&totalExpense)

	// 카테고리별 지출
	rows, _ := s.DB.QueryContext(r.Context(), `
		SELECT COALESCE(cat.name, '미분류'), SUM(t.amount), COUNT(*)
		FROM transactions t
		LEFT JOIN categories cat ON t.category_id = cat.id
		WHERE t.tx_type = 'expense' AND t.is_cancelled = 0
		AND substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) <= ?
		GROUP BY t.category_id ORDER BY SUM(t.amount) DESC`, startDate, endDate)
	defer rows.Close()

	type CategoryStat struct {
		Name   string `json:"category_name"`
		Amount int64  `json:"total_amount"`
		Count  int64  `json:"count"`
	}
	var byCategory []CategoryStat
	for rows.Next() {
		var c CategoryStat
		rows.Scan(&c.Name, &c.Amount, &c.Count)
		byCategory = append(byCategory, c)
	}

	// 자산 유형별 지출
	rows2, _ := s.DB.QueryContext(r.Context(), `
		SELECT COALESCE(at.name, '미분류'), SUM(t.amount), COUNT(*)
		FROM transactions t
		LEFT JOIN asset_types at ON t.asset_type_id = at.id
		WHERE t.tx_type = 'expense' AND t.is_cancelled = 0
		AND substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) <= ?
		GROUP BY t.asset_type_id ORDER BY SUM(t.amount) DESC`, startDate, endDate)
	defer rows2.Close()

	type AssetStat struct {
		Name   string `json:"asset_name"`
		Amount int64  `json:"total_amount"`
		Count  int64  `json:"count"`
	}
	var byAsset []AssetStat
	for rows2.Next() {
		var a AssetStat
		rows2.Scan(&a.Name, &a.Amount, &a.Count)
		byAsset = append(byAsset, a)
	}

	// 수입 카테고리별
	rows3, _ := s.DB.QueryContext(r.Context(), `
		SELECT COALESCE(ic.name, '미분류'), SUM(t.amount), COUNT(*)
		FROM transactions t
		LEFT JOIN income_categories ic ON t.income_category_id = ic.id
		WHERE t.tx_type = 'income' AND t.is_cancelled = 0
		AND substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) <= ?
		GROUP BY t.income_category_id ORDER BY SUM(t.amount) DESC`, startDate, endDate)
	defer rows3.Close()

	var byIncomeCategory []CategoryStat
	for rows3.Next() {
		var c CategoryStat
		rows3.Scan(&c.Name, &c.Amount, &c.Count)
		byIncomeCategory = append(byIncomeCategory, c)
	}

	// 고정지출 내역 (is_recurring = 1인 지출)
	rows4, _ := s.DB.QueryContext(r.Context(), `
		SELECT COALESCE(cat.name, '미분류'), t.description, t.amount, substr(t.transaction_date, 1, 10)
		FROM transactions t
		LEFT JOIN categories cat ON t.category_id = cat.id
		WHERE t.tx_type = 'expense' AND t.is_cancelled = 0 AND COALESCE(t.is_recurring, 0) = 1
		AND substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) <= ?
		ORDER BY t.amount DESC`, startDate, endDate)
	defer rows4.Close()

	type RecurringItem struct {
		CategoryName string `json:"category_name"`
		Description  string `json:"description"`
		Amount       int64  `json:"amount"`
		Date         string `json:"date"`
	}
	var recurringExpenses []RecurringItem
	var totalRecurring int64
	for rows4.Next() {
		var r RecurringItem
		rows4.Scan(&r.CategoryName, &r.Description, &r.Amount, &r.Date)
		recurringExpenses = append(recurringExpenses, r)
		totalRecurring += r.Amount
	}

	// 고정수입 내역
	rows5, _ := s.DB.QueryContext(r.Context(), `
		SELECT COALESCE(ic.name, '미분류'), t.description, t.amount, substr(t.transaction_date, 1, 10)
		FROM transactions t
		LEFT JOIN income_categories ic ON t.income_category_id = ic.id
		WHERE t.tx_type = 'income' AND t.is_cancelled = 0 AND COALESCE(t.is_recurring, 0) = 1
		AND substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) <= ?
		ORDER BY t.amount DESC`, startDate, endDate)
	defer rows5.Close()

	var recurringIncome []RecurringItem
	var totalRecurringIncome int64
	for rows5.Next() {
		var r RecurringItem
		rows5.Scan(&r.CategoryName, &r.Description, &r.Amount, &r.Date)
		recurringIncome = append(recurringIncome, r)
		totalRecurringIncome += r.Amount
	}

	s.writeJSON(w, map[string]any{
		"year":                   year,
		"month":                  month,
		"period_start":           startDate,
		"period_end":             endDate,
		"total_income":           totalIncome,
		"total_expense":          totalExpense,
		"balance":                totalIncome - totalExpense,
		"by_category":            byCategory,
		"by_asset":               byAsset,
		"by_income_category":     byIncomeCategory,
		"recurring_expenses":     recurringExpenses,
		"total_recurring":        totalRecurring,
		"recurring_income":       recurringIncome,
		"total_recurring_income": totalRecurringIncome,
	})
}

// 카드 통계 (카드별 이용 기간 기준 - 기존 로직 유지)
func (s *Server) HandleAPIStatisticsCard(w http.ResponseWriter, r *http.Request) {
	// 기존 HandleAPIStatistics와 동일한 로직 사용
	s.HandleAPIStatistics(w, r)
}

// ========== Recurring Schedules API ==========

type RecurringSchedule struct {
	ID                int64   `json:"id"`
	TxType            string  `json:"tx_type"`
	AssetTypeID       int64   `json:"asset_type_id"`
	AssetTypeName     string  `json:"asset_type_name"`
	CategoryID        *int64  `json:"category_id"`
	CategoryName      string  `json:"category_name"`
	IncomeCategoryID  *int64  `json:"income_category_id"`
	IncomeCatName     string  `json:"income_category_name"`
	Description       string  `json:"description"`
	Amount            int64   `json:"amount"`
	DayOfMonth        int64   `json:"day_of_month"`
	Memo              *string `json:"memo"`
	IsActive          int64   `json:"is_active"`
	LastGeneratedDate *string `json:"last_generated_date"`
}

func (s *Server) HandleAPIGetRecurringSchedules(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT rs.id, rs.tx_type, rs.asset_type_id, COALESCE(at.name, ''),
		       rs.category_id, COALESCE(cat.name, ''), rs.income_category_id, COALESCE(ic.name, ''),
		       rs.description, rs.amount, rs.day_of_month, rs.memo, rs.is_active, rs.last_generated_date
		FROM recurring_schedules rs
		LEFT JOIN asset_types at ON rs.asset_type_id = at.id
		LEFT JOIN categories cat ON rs.category_id = cat.id
		LEFT JOIN income_categories ic ON rs.income_category_id = ic.id
		WHERE rs.is_active = 1
		ORDER BY rs.day_of_month`)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	var schedules []RecurringSchedule
	for rows.Next() {
		var s RecurringSchedule
		rows.Scan(&s.ID, &s.TxType, &s.AssetTypeID, &s.AssetTypeName,
			&s.CategoryID, &s.CategoryName, &s.IncomeCategoryID, &s.IncomeCatName,
			&s.Description, &s.Amount, &s.DayOfMonth, &s.Memo, &s.IsActive, &s.LastGeneratedDate)
		schedules = append(schedules, s)
	}
	s.writeJSON(w, map[string]any{"schedules": schedules})
}

func (s *Server) HandleAPIDeleteRecurringSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	// 비활성화 (완전 삭제 대신)
	_, err := s.DB.ExecContext(r.Context(), "UPDATE recurring_schedules SET is_active = 0 WHERE id = ?", id)
	if err != nil {
		s.writeError(w, 500, err.Error())
		return
	}
	s.writeJSON(w, map[string]string{"status": "ok"})
}

// 반복 거래 자동 생성 (서버 시작 시 또는 주기적으로 호출)
func (s *Server) ProcessRecurringTransactions() error {
	now := time.Now()
	currentDate := now.Format("2006-01-02")

	rows, err := s.DB.Query(`
		SELECT id, tx_type, asset_type_id, category_id, income_category_id,
		       description, amount, day_of_month, memo, last_generated_date
		FROM recurring_schedules
		WHERE is_active = 1`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rs RecurringSchedule
		var lastGen sql.NullString
		err := rows.Scan(&rs.ID, &rs.TxType, &rs.AssetTypeID, &rs.CategoryID, &rs.IncomeCategoryID,
			&rs.Description, &rs.Amount, &rs.DayOfMonth, &rs.Memo, &lastGen)
		if err != nil {
			continue
		}

		// 이번 달 생성 여부 확인
		thisMonthTarget := time.Date(now.Year(), now.Month(), int(rs.DayOfMonth), 0, 0, 0, 0, time.Local)
		if thisMonthTarget.After(now) {
			continue // 아직 날짜가 안 됨
		}

		targetDate := thisMonthTarget.Format("2006-01-02")
		if lastGen.Valid && lastGen.String >= targetDate {
			continue // 이미 생성됨
		}

		// card_id 가져오기
		var cardID *int64
		row := s.DB.QueryRow("SELECT card_id FROM asset_types WHERE id = ?", rs.AssetTypeID)
		var cid sql.NullInt64
		if err := row.Scan(&cid); err == nil && cid.Valid {
			cardID = &cid.Int64
		}

		// 거래 생성
		result, err := s.DB.Exec(`
			INSERT INTO transactions (
				tx_type, asset_type_id, card_id, category_id, income_category_id,
				transaction_date, description, amount, is_installment, is_cancelled, memo, recurring_schedule_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?)`,
			rs.TxType, rs.AssetTypeID, cardID, rs.CategoryID, rs.IncomeCategoryID,
			targetDate, rs.Description, rs.Amount, rs.Memo, rs.ID)
		if err != nil {
			continue
		}

		if _, err := result.LastInsertId(); err == nil {
			s.DB.Exec("UPDATE recurring_schedules SET last_generated_date = ? WHERE id = ?", currentDate, rs.ID)
		}
	}
	return nil
}

// ========== Dashboard V2 API ==========

func (s *Server) HandleAPIDashboardV2(w http.ResponseWriter, r *http.Request) {
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

	startDay, endDay := s.getLedgerPeriod(r.Context())
	startDate, endDate := calculateLedgerPeriod(year, month, startDay, endDay)

	// 총 수입
	var totalIncome int64
	s.DB.QueryRowContext(r.Context(), `
		SELECT COALESCE(SUM(amount), 0) FROM transactions 
		WHERE tx_type = 'income' AND is_cancelled = 0
		AND substr(transaction_date, 1, 10) >= ? AND substr(transaction_date, 1, 10) <= ?`,
		startDate, endDate).Scan(&totalIncome)

	// 총 지출
	var totalExpense int64
	s.DB.QueryRowContext(r.Context(), `
		SELECT COALESCE(SUM(amount), 0) FROM transactions 
		WHERE tx_type = 'expense' AND is_cancelled = 0
		AND substr(transaction_date, 1, 10) >= ? AND substr(transaction_date, 1, 10) <= ?`,
		startDate, endDate).Scan(&totalExpense)

	// 최근 거래 (수입+지출 통합)
	rows, _ := s.DB.QueryContext(r.Context(), `
		SELECT t.id, t.tx_type, t.asset_type_id, COALESCE(at.name, ''), t.card_id,
		       t.category_id, COALESCE(cat.name, ''), t.income_category_id, COALESCE(ic.name, ''),
		       t.transaction_date, t.description, t.amount, t.is_installment,
		       t.installment_months, t.is_cancelled, t.memo
		FROM transactions t
		LEFT JOIN asset_types at ON t.asset_type_id = at.id
		LEFT JOIN categories cat ON t.category_id = cat.id
		LEFT JOIN income_categories ic ON t.income_category_id = ic.id
		WHERE substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) <= ?
		ORDER BY t.transaction_date DESC
		LIMIT 30`, startDate, endDate)
	defer rows.Close()

	var transactions []TransactionV2
	for rows.Next() {
		var tx TransactionV2
		rows.Scan(&tx.ID, &tx.TxType, &tx.AssetTypeID, &tx.AssetTypeName, &tx.CardID,
			&tx.CategoryID, &tx.CategoryName, &tx.IncomeCategoryID, &tx.IncomeCategoryName,
			&tx.TransactionDate, &tx.Description, &tx.Amount, &tx.IsInstallment,
			&tx.InstallmentMonths, &tx.IsCancelled, &tx.Memo)
		transactions = append(transactions, tx)
	}

	// 일별 합계 (달력 뷰용)
	rows2, _ := s.DB.QueryContext(r.Context(), `
		SELECT substr(transaction_date, 1, 10) as dt, tx_type, SUM(amount)
		FROM transactions
		WHERE is_cancelled = 0
		AND substr(transaction_date, 1, 10) >= ? AND substr(transaction_date, 1, 10) <= ?
		GROUP BY substr(transaction_date, 1, 10), tx_type
		ORDER BY dt`, startDate, endDate)
	defer rows2.Close()

	dailySummary := make(map[string]map[string]int64)
	for rows2.Next() {
		var dt, txType string
		var amount int64
		rows2.Scan(&dt, &txType, &amount)
		if dailySummary[dt] == nil {
			dailySummary[dt] = make(map[string]int64)
		}
		dailySummary[dt][txType] = amount
	}

	s.writeJSON(w, map[string]any{
		"year":              year,
		"month":             month,
		"period_start":      startDate,
		"period_end":        endDate,
		"total_income":      totalIncome,
		"total_expense":     totalExpense,
		"balance":           totalIncome - totalExpense,
		"transactions":      transactions,
		"daily_summary":     dailySummary,
	})
}
