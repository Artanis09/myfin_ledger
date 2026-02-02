package srv

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"srv.exe.dev/db/dbgen"
)

// HandleShortcutMessage handles incoming SMS messages from iPhone Shortcuts
// POST /api/shortcut/message
// Headers: X-API-Key: <api_key>
// Body: {"text": "SMS message content"}
func (s *Server) HandleShortcutMessage(w http.ResponseWriter, r *http.Request) {
	// 1. Validate API key
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		http.Error(w, `{"error": "missing X-API-Key header"}`, http.StatusUnauthorized)
		return
	}

	queries := dbgen.New(s.DB)
	keyHash := hashAPIKey(apiKey)

	apiKeyRecord, err := queries.GetAPIKeyByHash(r.Context(), keyHash)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "invalid API key"}`, http.StatusUnauthorized)
		} else {
			slog.Error("failed to validate API key", "error", err)
			http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	// Update last used timestamp
	go queries.UpdateAPIKeyLastUsed(context.Background(), apiKeyRecord.ID)

	// 2. Parse request body
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		http.Error(w, `{"error": "text field is required"}`, http.StatusBadRequest)
		return
	}

	slog.Info("received SMS from shortcut", "text_length", len(req.Text))

	// 3. Parse SMS
	parsed := parseSMS(req.Text)

	// 4. Create SMS log entry
	var parsedCardName, parsedDescription, parsedDate *string
	var parsedAmount *int64

	if parsed.CardName != "" {
		parsedCardName = &parsed.CardName
	}
	if parsed.Amount > 0 {
		parsedAmount = &parsed.Amount
	}
	if parsed.Description != "" {
		parsedDescription = &parsed.Description
	}
	if parsed.Date != "" {
		parsedDate = &parsed.Date
	}

	smsLog, err := queries.CreateSMSLog(r.Context(), dbgen.CreateSMSLogParams{
		RawText:           req.Text,
		ParsedCardName:    parsedCardName,
		ParsedAmount:      parsedAmount,
		ParsedDescription: parsedDescription,
		ParsedDate:        parsedDate,
		Status:            "received",
	})
	if err != nil {
		slog.Error("failed to create SMS log", "error", err)
		http.Error(w, `{"error": "failed to save SMS log"}`, http.StatusInternalServerError)
		return
	}

	// 5. Try to create transaction if parsing succeeded
	var transactionID *int64
	var status = "parsed"
	var errorMsg *string

	if parsed.CardName != "" && parsed.Amount > 0 {
		// Find card by name
		card, err := queries.GetCardByName(r.Context(), parsed.CardName)
		if err != nil {
			if err == sql.ErrNoRows {
				errStr := "card not found: " + parsed.CardName
				errorMsg = &errStr
				status = "error"
			} else {
				slog.Error("failed to find card", "error", err)
			}
		} else {
			// Parse transaction date
			transDate := time.Now()
			if parsed.Date != "" {
				if t, err := time.Parse("2006-01-02", parsed.Date); err == nil {
					transDate = t
					if parsed.Time != "" {
						parts := strings.Split(parsed.Time, ":")
						if len(parts) == 2 {
							var hour, min int
							hour = parseInt(parts[0])
							min = parseInt(parts[1])
							transDate = time.Date(t.Year(), t.Month(), t.Day(), hour, min, 0, 0, time.Local)
						}
					}
				}
			}

			// Find matching category
			var categoryID *int64
			categories, _ := queries.GetAllCategories(r.Context())
			for _, cat := range categories {
				keywords := strings.Split(cat.Keywords, ",")
				for _, kw := range keywords {
					kw = strings.TrimSpace(kw)
					if kw != "" && strings.Contains(strings.ToLower(parsed.Description), strings.ToLower(kw)) {
						categoryID = &cat.ID
						break
					}
				}
				if categoryID != nil {
					break
				}
			}

			// Create transaction
			var installmentMonths, installmentCurrent, originalAmount *int64
			var isInstallment int64 = 0
			if parsed.IsInstallment {
				isInstallment = 1
				installmentMonths = &parsed.InstallmentMonths
				var current int64 = 1
				installmentCurrent = &current
				original := parsed.Amount * parsed.InstallmentMonths
				originalAmount = &original
			}

			trans, err := queries.CreateTransaction(r.Context(), dbgen.CreateTransactionParams{
				CardID:             card.ID,
				CategoryID:         categoryID,
				TransactionDate:    transDate,
				Description:        parsed.Description,
				Amount:             parsed.Amount,
				IsInstallment:      isInstallment,
				InstallmentMonths:  installmentMonths,
				InstallmentCurrent: installmentCurrent,
				OriginalAmount:     originalAmount,
			})
			if err != nil {
				slog.Error("failed to create transaction", "error", err)
				errStr := "failed to create transaction: " + err.Error()
				errorMsg = &errStr
				status = "error"
			} else {
				transactionID = &trans.ID
				status = "saved"
				slog.Info("created transaction from SMS",
					"transaction_id", trans.ID,
					"card", parsed.CardName,
					"amount", parsed.Amount,
					"description", parsed.Description)
			}
		}
	} else {
		errStr := "could not parse SMS: missing card name or amount"
		errorMsg = &errStr
		status = "error"
	}

	// Update SMS log status
	queries.UpdateSMSLogStatus(r.Context(), dbgen.UpdateSMSLogStatusParams{
		ID:            smsLog.ID,
		Status:        status,
		TransactionID: transactionID,
		ErrorMessage:  errorMsg,
	})

	// 6. Return response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  status,
		"sms_log_id": smsLog.ID,
		"parsed": map[string]interface{}{
			"card_name":   parsed.CardName,
			"amount":      parsed.Amount,
			"description": parsed.Description,
			"date":        parsed.Date,
			"time":        parsed.Time,
		},
	}
	if transactionID != nil {
		response["transaction_id"] = *transactionID
	}
	if errorMsg != nil {
		response["error"] = *errorMsg
	}
	json.NewEncoder(w).Encode(response)
}

// HandleGenerateAPIKey creates a new API key
// POST /api/shortcut/keys
// Body: {"name": "key name"}
func (s *Server) HandleGenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, `{"error": "name is required"}`, http.StatusBadRequest)
		return
	}

	// Generate random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		http.Error(w, `{"error": "failed to generate key"}`, http.StatusInternalServerError)
		return
	}
	apiKey := hex.EncodeToString(keyBytes)
	keyHash := hashAPIKey(apiKey)

	queries := dbgen.New(s.DB)
	_, err := queries.CreateAPIKey(r.Context(), dbgen.CreateAPIKeyParams{
		KeyHash: keyHash,
		Name:    req.Name,
	})
	if err != nil {
		slog.Error("failed to create API key", "error", err)
		http.Error(w, `{"error": "failed to create API key"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"api_key": apiKey,
		"name":    req.Name,
		"message": "API 키가 생성되었습니다. 이 키는 다시 표시되지 않으므로 안전하게 보관하세요.",
	})
}

// HandleGetAPIKeys returns list of API keys (without the actual key)
// GET /api/shortcut/keys
func (s *Server) HandleGetAPIKeys(w http.ResponseWriter, r *http.Request) {
	queries := dbgen.New(s.DB)
	keys, err := queries.GetAllAPIKeys(r.Context())
	if err != nil {
		http.Error(w, `{"error": "failed to get API keys"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

// HandleDeactivateAPIKey deactivates an API key
// DELETE /api/shortcut/keys/{id}
func (s *Server) HandleDeactivateAPIKey(w http.ResponseWriter, r *http.Request) {
	id := parseInt64(r.PathValue("id"))
	if id == 0 {
		http.Error(w, `{"error": "invalid id"}`, http.StatusBadRequest)
		return
	}

	queries := dbgen.New(s.DB)
	if err := queries.DeactivateAPIKey(r.Context(), id); err != nil {
		http.Error(w, `{"error": "failed to deactivate API key"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deactivated"})
}

// HandleGetSMSLogs returns recent SMS logs
// GET /api/shortcut/logs
func (s *Server) HandleGetSMSLogs(w http.ResponseWriter, r *http.Request) {
	limit := parseInt64(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	queries := dbgen.New(s.DB)
	logs, err := queries.GetSMSLogs(r.Context(), limit)
	if err != nil {
		http.Error(w, `{"error": "failed to get SMS logs"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// HandleDeleteSMSLog deletes a single SMS log
// DELETE /api/shortcut/logs/{id}
func (s *Server) HandleDeleteSMSLog(w http.ResponseWriter, r *http.Request) {
	id := parseInt64(r.PathValue("id"))
	if id == 0 {
		http.Error(w, `{"error": "invalid id"}`, http.StatusBadRequest)
		return
	}

	queries := dbgen.New(s.DB)
	if err := queries.DeleteSMSLog(r.Context(), id); err != nil {
		http.Error(w, `{"error": "failed to delete SMS log"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// HandleDeleteAllSMSLogs deletes all SMS logs
// DELETE /api/shortcut/logs
func (s *Server) HandleDeleteAllSMSLogs(w http.ResponseWriter, r *http.Request) {
	queries := dbgen.New(s.DB)
	if err := queries.DeleteAllSMSLogs(r.Context()); err != nil {
		http.Error(w, `{"error": "failed to delete SMS logs"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "all deleted"})
}

func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func parseInt64(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}
