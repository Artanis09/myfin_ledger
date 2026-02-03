# Build stage for Frontend
FROM oven/bun:1 AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/bun.lock* ./
RUN bun install --frozen-lockfile
COPY frontend/ ./
RUN bun run build

# Build stage for Backend
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/cmd/srv/dist ./cmd/srv/dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/srv

# Final stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata sqlite
WORKDIR /app

COPY --from=backend-builder /app/server .
COPY --from=backend-builder /app/db/migrations ./db/migrations

ENV TZ=Asia/Seoul
ENV PORT=8000

EXPOSE 8000

VOLUME ["/app/data"]

CMD ["./server"]
