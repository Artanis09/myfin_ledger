# Agent Instructions

이 문서는 항상 한글로 작성하여 업데이트 합니다.
모든 문서는 항상 한글로 작성하여 업데이트 합니다.
작업이 완료될때마다 DB의 변경사항들을 모두 분석하여, 스키마, 관계테이블등을 DB_DESIGN.md에 상세히 업데이트 합니다
This is a Go web application template for exe.dev.

See README.md for details on the structure and components.

## Recent Updates (2026-02-03)

### 1. PWA & Docker Environment
- **PWA Implementation**: Integrated `vite-plugin-pwa` in `frontend/vite.config.ts`.
  - Service Worker (SW) registered in `frontend/src/main.tsx`.
  - Manifest and icons configured for standalone mobile experience.
  - Offline caching strategy: `NetworkFirst` for `/api/*` and `StaleWhileRevalidate` for assets.
- **Dockerization**:
  - `Dockerfile`: Multi-stage build (Bun for frontend, Go for backend).
  - `docker-compose.yml`: Persists DB in `./data/db.sqlite3`.

### 2. Core Logic: Billing Month & Effective Transactions
- **Billing Month Concept**: The app now operates on "Billing Month" (결제예정월) rather than calendar months.
  - Transactions are grouped based on card billing cycles (e.g., 23rd to 22nd).
  - Default view for Dashboard, Transactions, and Statistics is `current_month + 1` (Next Billing Month).
- **Virtual Installments (1/N)**: 
  - Centralized logic in `srv/api_handlers.go`: `getEffectiveTransactions`.
  - Splits `amount` by `installment_months`.
  - Generates virtual transaction entries for each billing month until the installment ends.
- **Cancellation Logic**: Transactions with `is_cancelled = 1` are subtracted from totals and shown as negative/strikethrough entries.

### 3. Frontend Pages
- **Statistics**:
  - Displays total spending for the billing month with a comparison to the previous billing month.
  - Shows increase/decrease percentage badges.
  - Includes Category and Card breakdown charts (Recharts).
- **Transactions**:
  - Lists virtual transactions (including monthly installment slices).
  - Supports month-by-month navigation.
- **Dashboard**:
  - Pastel Purple theme with gradient background.
  - Horizontal scrolling card billing summaries at the top.

### 4. Technical Stack
- **Backend**: Go with `net/http`, `sqlite3`.
- **Frontend**: React, TypeScript, Vite, CSS Modules, Lucide-React, Recharts, date-fns.
- **State Management**: React `useState`/`useEffect` with centralized API calls in `frontend/src/api/index.ts`.
