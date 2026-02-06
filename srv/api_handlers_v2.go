package srv

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"srv.exe.dev/db/dbgen"
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
	IsRecurring        int64   `json:"is_recurring"`
	IsAutoRepeat       int64   `json:"is_auto_repeat"`
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
		       t.installment_months, t.is_cancelled, t.memo, t.recurring_schedule_id, COALESCE(t.is_recurring, 0)
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
			&tx.InstallmentMonths, &tx.IsCancelled, &tx.Memo, &tx.RecurringID, &tx.IsRecurring)
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
		IsAutoRepeat      int64   `json:"is_auto_repeat"`
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
			is_cancelled, memo, is_recurring, is_auto_repeat
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.TxType, req.AssetTypeID, cardID, req.CategoryID, req.IncomeCategoryID,
		req.TransactionDate, req.Description, req.Amount, req.IsInstallment,
		req.InstallmentMonths, req.IsCancelled, req.Memo, req.IsRecurring, req.IsAutoRepeat)
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
		       t.installment_months, t.is_cancelled, t.memo, COALESCE(t.is_recurring, 0), COALESCE(t.is_auto_repeat, 0)
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
		IsAutoRepeat       int64   `json:"is_auto_repeat"`
	}

	err := row.Scan(&tx.ID, &tx.TxType, &tx.AssetTypeID, &tx.AssetTypeName, &tx.CardID,
		&tx.CategoryID, &tx.CategoryName, &tx.IncomeCategoryID, &tx.IncomeCategoryName,
		&tx.TransactionDate, &tx.Description, &tx.Amount, &tx.IsInstallment,
		&tx.InstallmentMonths, &tx.IsCancelled, &tx.Memo, &tx.IsRecurring, &tx.IsAutoRepeat)
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
		IsAutoRepeat      int64   `json:"is_auto_repeat"`
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
			installment_months = ?, is_cancelled = ?, memo = ?, is_recurring = ?, is_auto_repeat = ?
		WHERE id = ?`,
		req.TxType, req.AssetTypeID, cardID, req.CategoryID, req.IncomeCategoryID,
		req.TransactionDate, req.Description, req.Amount, req.IsInstallment,
		req.InstallmentMonths, req.IsCancelled, req.Memo, req.IsRecurring, req.IsAutoRepeat, id)
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
	currentYearMonth := now.Format("2006-01")

	// 1. recurring_schedules 테이블 기반 반복 (기존 로직)
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

	// 2. transactions 테이블의 is_auto_repeat = 1인 거래 반복 (새로운 로직)
	autoRows, err := s.DB.Query(`
		SELECT id, tx_type, asset_type_id, card_id, category_id, income_category_id,
		       transaction_date, description, amount, is_installment, installment_months,
		       memo, is_recurring
		FROM transactions
		WHERE is_auto_repeat = 1 
		  AND is_cancelled = 0
		  AND parent_transaction_id IS NULL
		  AND (last_repeat_date IS NULL OR substr(last_repeat_date, 1, 7) < ?)
	`, currentYearMonth)
	if err != nil {
		return err
	}
	defer autoRows.Close()

	for autoRows.Next() {
		var tx struct {
			ID               int64
			TxType           string
			AssetTypeID      sql.NullInt64
			CardID           sql.NullInt64
			CategoryID       sql.NullInt64
			IncomeCategoryID sql.NullInt64
			TransactionDate  string
			Description      string
			Amount           int64
			IsInstallment    int64
			InstallmentMonths sql.NullInt64
			Memo             sql.NullString
			IsRecurring      int64
		}
		err := autoRows.Scan(&tx.ID, &tx.TxType, &tx.AssetTypeID, &tx.CardID,
			&tx.CategoryID, &tx.IncomeCategoryID, &tx.TransactionDate,
			&tx.Description, &tx.Amount, &tx.IsInstallment, &tx.InstallmentMonths,
			&tx.Memo, &tx.IsRecurring)
		if err != nil {
			continue
		}

		// 원본 거래 날짜에서 일자 추출
		var origDay int
		origDate, err := time.Parse("2006-01-02T15:04:05Z07:00", tx.TransactionDate)
		if err != nil {
			origDate, err = time.Parse("2006-01-02", tx.TransactionDate[:10])
			if err != nil {
				continue
			}
		}
		origDay = origDate.Day()

		// 이번 달의 같은 날짜로 새 거래 생성
		monthEnd := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.Local)
		day := origDay
		if day > monthEnd.Day() {
			day = monthEnd.Day()
		}
		newDate := time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, time.Local)

		// 새 거래 생성 (자동 반복 해제, parent_transaction_id 설정)
		var assetTypeID, cardID, categoryID, incomeCategoryID *int64
		var installmentMonths *int64
		var memo *string

		if tx.AssetTypeID.Valid {
			v := tx.AssetTypeID.Int64
			assetTypeID = &v
		}
		if tx.CardID.Valid {
			v := tx.CardID.Int64
			cardID = &v
		}
		if tx.CategoryID.Valid {
			v := tx.CategoryID.Int64
			categoryID = &v
		}
		if tx.IncomeCategoryID.Valid {
			v := tx.IncomeCategoryID.Int64
			incomeCategoryID = &v
		}
		if tx.InstallmentMonths.Valid {
			v := tx.InstallmentMonths.Int64
			installmentMonths = &v
		}
		if tx.Memo.Valid {
			memo = &tx.Memo.String
		}

		_, err = s.DB.Exec(`
			INSERT INTO transactions (
				tx_type, asset_type_id, card_id, category_id, income_category_id,
				transaction_date, description, amount, is_installment, installment_months,
				is_cancelled, memo, is_recurring, is_auto_repeat, parent_transaction_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, 0, ?)`,
			tx.TxType, assetTypeID, cardID, categoryID, incomeCategoryID,
			newDate.Format("2006-01-02"), tx.Description, tx.Amount, tx.IsInstallment,
			installmentMonths, memo, tx.IsRecurring, tx.ID)
		if err != nil {
			continue
		}

		// 원본 거래의 last_repeat_date 업데이트
		s.DB.Exec(`UPDATE transactions SET last_repeat_date = ? WHERE id = ?`, currentYearMonth, tx.ID)
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
		       t.installment_months, t.is_cancelled, t.memo, COALESCE(t.is_recurring, 0)
		FROM transactions t
		LEFT JOIN asset_types at ON t.asset_type_id = at.id
		LEFT JOIN categories cat ON t.category_id = cat.id
		LEFT JOIN income_categories ic ON t.income_category_id = ic.id
		WHERE substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) <= ?
		ORDER BY t.transaction_date DESC
		LIMIT 100`, startDate, endDate)
	defer rows.Close()

	var transactions []TransactionV2
	for rows.Next() {
		var tx TransactionV2
		rows.Scan(&tx.ID, &tx.TxType, &tx.AssetTypeID, &tx.AssetTypeName, &tx.CardID,
			&tx.CategoryID, &tx.CategoryName, &tx.IncomeCategoryID, &tx.IncomeCategoryName,
			&tx.TransactionDate, &tx.Description, &tx.Amount, &tx.IsInstallment,
			&tx.InstallmentMonths, &tx.IsCancelled, &tx.Memo, &tx.IsRecurring)
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

// ========== Category Mapping API ==========

// normalizeDescription 정규화된 description 생성 (공백 제거, 소문자, 특수문자 간소화)
func normalizeDescription(desc string) string {
	// 공백 제거
	desc = strings.ReplaceAll(desc, " ", "")
	// 괄호 안 내용 제거 (지점명 등)
	re := regexp.MustCompile(`\([^)]*\)`)
	desc = re.ReplaceAllString(desc, "")
	// 소문자 변환
	desc = strings.ToLower(desc)
	return desc
}

// HandleAPIGetCategoryMapping 특정 description에 대한 카테고리 매핑 조회
func (s *Server) HandleAPIGetCategoryMapping(w http.ResponseWriter, r *http.Request) {
	desc := r.URL.Query().Get("description")
	if desc == "" {
		s.writeJSON(w, map[string]any{"category_id": nil, "category_name": ""})
		return
	}
	
	normalized := normalizeDescription(desc)
	
	var categoryID int64
	var categoryName string
	err := s.DB.QueryRowContext(r.Context(), `
		SELECT cm.category_id, c.name 
		FROM category_mappings cm
		JOIN categories c ON cm.category_id = c.id
		WHERE cm.normalized_description = ?`, normalized).Scan(&categoryID, &categoryName)
	
	if err != nil {
		s.writeJSON(w, map[string]any{"category_id": nil, "category_name": ""})
		return
	}
	
	s.writeJSON(w, map[string]any{
		"category_id":   categoryID,
		"category_name": categoryName,
	})
}

// HandleAPISaveCategoryMapping 카테고리 매핑 저장/업데이트
func (s *Server) HandleAPISaveCategoryMapping(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Description string `json:"description"`
		CategoryID  int64  `json:"category_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	
	if req.Description == "" || req.CategoryID == 0 {
		http.Error(w, "description and category_id required", http.StatusBadRequest)
		return
	}
	
	normalized := normalizeDescription(req.Description)
	
	// UPSERT: 있으면 업데이트, 없으면 삽입
	_, err := s.DB.ExecContext(r.Context(), `
		INSERT INTO category_mappings (description, normalized_description, category_id, usage_count, updated_at)
		VALUES (?, ?, ?, 1, CURRENT_TIMESTAMP)
		ON CONFLICT(normalized_description) DO UPDATE SET
			category_id = excluded.category_id,
			usage_count = usage_count + 1,
			updated_at = CURRENT_TIMESTAMP`,
		req.Description, normalized, req.CategoryID)
	
	if err != nil {
		slog.Error("failed to save category mapping", "error", err)
		http.Error(w, "failed to save mapping", http.StatusInternalServerError)
		return
	}
	
	// 카테고리 이름 조회
	var categoryName string
	s.DB.QueryRowContext(r.Context(), "SELECT name FROM categories WHERE id = ?", req.CategoryID).Scan(&categoryName)
	
	s.writeJSON(w, map[string]any{
		"success":       true,
		"category_id":   req.CategoryID,
		"category_name": categoryName,
	})
}

// HandleAPIGetAllCategoryMappings 모든 카테고리 매핑 목록 조회
func (s *Server) HandleAPIGetAllCategoryMappings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT cm.id, cm.description, cm.category_id, c.name, cm.usage_count, cm.updated_at
		FROM category_mappings cm
		JOIN categories c ON cm.category_id = c.id
		ORDER BY cm.usage_count DESC, cm.updated_at DESC`)
	if err != nil {
		http.Error(w, "failed to query mappings", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var mappings []map[string]any
	for rows.Next() {
		var id, categoryID, usageCount int64
		var desc, categoryName, updatedAt string
		rows.Scan(&id, &desc, &categoryID, &categoryName, &usageCount, &updatedAt)
		mappings = append(mappings, map[string]any{
			"id":            id,
			"description":   desc,
			"category_id":   categoryID,
			"category_name": categoryName,
			"usage_count":   usageCount,
			"updated_at":    updatedAt,
		})
	}
	
	if mappings == nil {
		mappings = []map[string]any{}
	}
	
	s.writeJSON(w, map[string]any{"mappings": mappings})
}

// HandleAPIDeleteCategoryMapping 카테고리 매핑 삭제
func (s *Server) HandleAPIDeleteCategoryMapping(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	
	_, err := s.DB.ExecContext(r.Context(), "DELETE FROM category_mappings WHERE id = ?", id)
	if err != nil {
		http.Error(w, "failed to delete mapping", http.StatusInternalServerError)
		return
	}
	
	s.writeJSON(w, map[string]any{"success": true})
}

// SMS Mapping Rules API
func (s *Server) HandleAPIGetSMSMappingRules(w http.ResponseWriter, r *http.Request) {
	queries := dbgen.New(s.DB)
	rules, err := queries.GetSMSMappingRules(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s.writeJSON(w, map[string]any{"rules": rules})
}

func (s *Server) HandleAPICreateSMSMappingRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RuleType    string  `json:"rule_type"`
		Keyword     string  `json:"keyword"`
		AssetTypeID *int64  `json:"asset_type_id"`
		TxType      *string `json:"tx_type"`
		Description *string `json:"description"`
		Priority    int64   `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	queries := dbgen.New(s.DB)
	rule, err := queries.CreateSMSMappingRule(r.Context(), dbgen.CreateSMSMappingRuleParams{
		RuleType:    req.RuleType,
		Keyword:     req.Keyword,
		AssetTypeID: req.AssetTypeID,
		TxType:      req.TxType,
		Description: req.Description,
		Priority:    req.Priority,
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s.writeJSON(w, rule)
}

func (s *Server) HandleAPIDeleteSMSMappingRule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	queries := dbgen.New(s.DB)
	if err := queries.DeleteSMSMappingRule(r.Context(), id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s.writeJSON(w, map[string]string{"status": "deleted"})
}
