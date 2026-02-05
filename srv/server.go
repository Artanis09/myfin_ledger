package srv

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"srv.exe.dev/db"
)

type Server struct {
	DB           *sql.DB
	Hostname     string
	StaticDir    string
}

func New(dbPath, hostname string) (*Server, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(thisFile)
	srv := &Server{
		Hostname:  hostname,
		StaticDir: filepath.Join(baseDir, "static"),
	}
	if err := srv.setUpDatabase(dbPath); err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *Server) setUpDatabase(dbPath string) error {
	wdb, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}
	s.DB = wdb
	if err := db.RunMigrations(wdb); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func (s *Server) Serve(addr string, frontendFS fs.FS) error {
	mux := http.NewServeMux()
	
	// API routes
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/dashboard", s.HandleAPIDashboard)
	
	mux.HandleFunc("GET /api/transactions", s.HandleAPIGetTransactions)
	mux.HandleFunc("GET /api/transactions/{id}", s.HandleAPIGetTransaction)
	mux.HandleFunc("POST /api/transactions", s.HandleAPICreateTransaction)
	mux.HandleFunc("PUT /api/transactions/{id}", s.HandleAPIUpdateTransaction)
	mux.HandleFunc("DELETE /api/transactions/{id}", s.HandleAPIDeleteTransaction)
	
	mux.HandleFunc("GET /api/cards", s.HandleAPIGetCards)
	mux.HandleFunc("POST /api/cards", s.HandleAPICreateCard)
	mux.HandleFunc("PUT /api/cards/{id}", s.HandleAPIUpdateCard)
	mux.HandleFunc("DELETE /api/cards/{id}", s.HandleAPIDeleteCard)
	
	mux.HandleFunc("GET /api/categories", s.HandleAPIGetCategories)
	mux.HandleFunc("POST /api/categories", s.HandleAPICreateCategory)
	mux.HandleFunc("PUT /api/categories/{id}", s.HandleAPIUpdateCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", s.HandleAPIDeleteCategory)
	
	mux.HandleFunc("GET /api/statistics", s.HandleAPIStatistics)
	mux.HandleFunc("POST /api/parse-sms", s.HandleAPIParseSMS)
	mux.HandleFunc("POST /api/parse-image", s.HandleAPIParseImage)
	mux.HandleFunc("POST /api/transactions/bulk", s.HandleAPICreateMultipleTransactions)
	
	// Weekly stats and goals
	mux.HandleFunc("GET /api/weekly-stats", s.HandleAPIWeeklyStats)
	mux.HandleFunc("GET /api/goals", s.HandleAPIGetGoal)
	mux.HandleFunc("POST /api/goals", s.HandleAPISetGoal)
	
	// iPhone Shortcut API routes
	mux.HandleFunc("POST /api/shortcut/message", s.HandleShortcutMessage)
	mux.HandleFunc("POST /api/shortcut/keys", s.HandleGenerateAPIKey)
	mux.HandleFunc("GET /api/shortcut/keys", s.HandleGetAPIKeys)
	mux.HandleFunc("DELETE /api/shortcut/keys/{id}", s.HandleDeactivateAPIKey)
	mux.HandleFunc("GET /api/shortcut/logs", s.HandleGetSMSLogs)
	mux.HandleFunc("DELETE /api/shortcut/logs/{id}", s.HandleDeleteSMSLog)
	mux.HandleFunc("DELETE /api/shortcut/logs", s.HandleDeleteAllSMSLogs)

	// V2 API routes (통합 관리 시스템)
	mux.HandleFunc("GET /api/v2/settings", s.HandleAPIGetSettings)
	mux.HandleFunc("PUT /api/v2/settings", s.HandleAPIUpdateSettings)
	
	mux.HandleFunc("GET /api/v2/asset-types", s.HandleAPIGetAssetTypes)
	mux.HandleFunc("POST /api/v2/asset-types", s.HandleAPICreateAssetType)
	mux.HandleFunc("DELETE /api/v2/asset-types/{id}", s.HandleAPIDeleteAssetType)
	
	mux.HandleFunc("GET /api/v2/income-categories", s.HandleAPIGetIncomeCategories)
	mux.HandleFunc("POST /api/v2/income-categories", s.HandleAPICreateIncomeCategory)
	mux.HandleFunc("PUT /api/v2/income-categories/{id}", s.HandleAPIUpdateIncomeCategory)
	mux.HandleFunc("DELETE /api/v2/income-categories/{id}", s.HandleAPIDeleteIncomeCategory)
	
	mux.HandleFunc("GET /api/v2/transactions", s.HandleAPIGetTransactionsV2)
	mux.HandleFunc("POST /api/v2/transactions", s.HandleAPICreateTransactionV2)
	
	mux.HandleFunc("GET /api/v2/transactions/{id}", s.HandleAPIGetTransactionV2)
	mux.HandleFunc("PUT /api/v2/transactions/{id}", s.HandleAPIUpdateTransactionV2)
	mux.HandleFunc("DELETE /api/v2/transactions/{id}", s.HandleAPIDeleteTransactionV2)
	mux.HandleFunc("GET /api/v2/dashboard", s.HandleAPIDashboardV2)
	mux.HandleFunc("GET /api/v2/statistics/ledger", s.HandleAPIStatisticsLedger)
	mux.HandleFunc("GET /api/v2/statistics/card", s.HandleAPIStatisticsCard)
	
	mux.HandleFunc("GET /api/v2/recurring", s.HandleAPIGetRecurringSchedules)
	mux.HandleFunc("DELETE /api/v2/recurring/{id}", s.HandleAPIDeleteRecurringSchedule)

	// Handle trailing slash redirects for API routes
	mux.HandleFunc("/api/shortcut/keys/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/shortcut/keys", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/api/shortcut/logs/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/shortcut/logs", http.StatusMovedPermanently)
	})
	
	// Serve frontend
	if frontendFS != nil {
		fileServer := http.FileServer(http.FS(frontendFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Try to serve file, if not found serve index.html for SPA
			path := r.URL.Path
			if path == "/" {
				path = "/index.html"
			}
			
			// Check if file exists
			if _, err := fs.Stat(frontendFS, path[1:]); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
			
			// Serve index.html for SPA routing
			r.URL.Path = "/index.html"
			fileServer.ServeHTTP(w, r)
		})
	}
	
	slog.Info("starting server", "addr", addr)
	return http.ListenAndServe(addr, mux)
}

func calculateBillingPeriod(now time.Time, startDay, endDay int) (time.Time, time.Time) {
	year, month, day := now.Date()
	
	var start, end time.Time
	
	if startDay <= endDay {
		start = time.Date(year, month, startDay, 0, 0, 0, 0, now.Location())
		end = time.Date(year, month, endDay, 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	} else {
		if day >= startDay {
			start = time.Date(year, month, startDay, 0, 0, 0, 0, now.Location())
			end = time.Date(year, month+1, endDay, 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
		} else {
			start = time.Date(year, month-1, startDay, 0, 0, 0, 0, now.Location())
			end = time.Date(year, month, endDay, 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
		}
	}
	
	return start, end
}
