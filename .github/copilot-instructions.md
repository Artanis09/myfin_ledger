# MyLedger - AI Coding Instructions

## 🏗 Big Picture & Architecture
- **Billing-Centric Model**: Everything revolves around the "Billing Month" (결제예정월). Data is filtered by card billing cycles (e.g., 23rd to 22nd) rather than calendar months. 
- **Default Timeframe**: Display defaults to the *next month* (`current_month + 1`) to reflect upcoming credit card payments.
- **Effective Transactions**: Do NOT query the `transactions` table directly for UI display. Always use `getEffectiveTransactions` in `srv/api_handlers.go`.
  - **1/N Installments**: Large purchases are virtually split over several months.
  - **Cancellations**: Cancelled transactions are subtracted from totals and shown as negative values.
- **Stack**: Go backend (`srv/`), React/Vite frontend (`frontend/`), SQLite (`db/`).

## 🛠 Critical Workflows
- **Frontend Build**: Use `bun run build` in `/frontend`. Outputs to `../cmd/srv/dist/`.
- **Backend Build**: `go build -o myledger ./cmd/srv`.
- **Docker**: Deploy using `docker compose up --build`. Persistent DB at `./data/db.sqlite3`.
- **PWA**: Service Worker and Manifest are managed via `vite-plugin-pwa`. Update `frontend/vite.config.ts` for caching rules.

## 📏 Conventions & Patterns
- **API Response**: Backend sends `total_last_month` for statistics comparison. Frontend expects `total`, `by_category`, and `by_card` fields in `/api/statistics`.
- **Date Handling**: Use `date-fns` on frontend. Backend expects `year` and `month` query parameters for transactional data.
- **Styling**: Pastel Purple theme (`#8b5cf6`). Use CSS Modules in `/frontend/src/pages/*.module.css`. Dashboard uses horizontal scrolling for billing summaries.

## 🧩 Integration Points
- **SMS Parsing**: `srv/handlers_sms.go` handles bank/card SMS parsing via description-based regex.
- **SQLC**: Database queries are generated. DO NOT edit `db/dbgen/*.go` manually. Update `db/queries/*.sql` and run `sqlc generate`.

## ⚠️ Important Note
Always ensure `AGENTS.md` is updated after major logic changes to keep cross-session agents informed.
