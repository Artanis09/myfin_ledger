package srv

import (
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"srv.exe.dev/db"
)

type Server struct {
	DB           *sql.DB
	Hostname     string
	TemplatesDir string
	StaticDir    string
}

func New(dbPath, hostname string) (*Server, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(thisFile)
	srv := &Server{
		Hostname:     hostname,
		TemplatesDir: filepath.Join(baseDir, "templates"),
		StaticDir:    filepath.Join(baseDir, "static"),
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

func (s *Server) Serve(addr string) error {
	mux := http.NewServeMux()
	
	// Pages
	mux.HandleFunc("GET /{$}", s.HandleDashboard)
	mux.HandleFunc("GET /transactions", s.HandleTransactions)
	mux.HandleFunc("GET /transactions/new", s.HandleTransactionForm)
	mux.HandleFunc("POST /transactions", s.HandleCreateTransaction)
	mux.HandleFunc("GET /transactions/{id}/edit", s.HandleEditTransaction)
	mux.HandleFunc("POST /transactions/{id}", s.HandleUpdateTransaction)
	mux.HandleFunc("POST /transactions/{id}/delete", s.HandleDeleteTransaction)
	
	mux.HandleFunc("GET /cards", s.HandleCards)
	mux.HandleFunc("POST /cards", s.HandleCreateCard)
	mux.HandleFunc("POST /cards/{id}", s.HandleUpdateCard)
	mux.HandleFunc("POST /cards/{id}/delete", s.HandleDeleteCard)
	
	mux.HandleFunc("GET /categories", s.HandleCategories)
	mux.HandleFunc("POST /categories", s.HandleCreateCategory)
	mux.HandleFunc("POST /categories/{id}", s.HandleUpdateCategory)
	mux.HandleFunc("POST /categories/{id}/delete", s.HandleDeleteCategory)
	
	mux.HandleFunc("GET /statistics", s.HandleStatistics)
	
	// API
	mux.HandleFunc("POST /api/parse-sms", s.HandleParseSMS)
	
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(s.StaticDir))))
	
	slog.Info("starting server", "addr", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) renderTemplate(w http.ResponseWriter, name string, data any) error {
	path := filepath.Join(s.TemplatesDir, name)
	layoutPath := filepath.Join(s.TemplatesDir, "layout.html")
	
	funcMap := template.FuncMap{
		"formatMoney": func(amount interface{}) string {
			var val int64
			switch v := amount.(type) {
			case int64:
				val = v
			case *int64:
				if v != nil {
					val = *v
				}
			default:
				return "0원"
			}
			str := fmt.Sprintf("%d", val)
			var result []string
			for i := len(str); i > 0; i -= 3 {
				start := i - 3
				if start < 0 {
					start = 0
				}
				result = append([]string{str[start:i]}, result...)
			}
			return strings.Join(result, ",") + "원"
		},
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04")
		},
		"deref": func(p *int64) int64 {
			if p != nil {
				return *p
			}
			return 0
		},
	}
	
	tmpl, err := template.New("layout.html").Funcs(funcMap).ParseFiles(layoutPath, path)
	if err != nil {
		return fmt.Errorf("parse template %q: %w", name, err)
	}
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("execute template %q: %w", name, err)
	}
	return nil
}
