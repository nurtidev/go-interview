# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# Stage 1: build the frontend
# ---------------------------------------------------------------------------
FROM node:22 AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---------------------------------------------------------------------------
# Stage 2: build the Go binary (pure Go, no cgo)
# ---------------------------------------------------------------------------
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/server ./cmd/server

# ---------------------------------------------------------------------------
# Stage 3: minimal runtime image
# ---------------------------------------------------------------------------
FROM gcr.io/distroless/base-debian12 AS final
WORKDIR /app

COPY --from=build /out/server /app/server
COPY --from=web /app/web/dist /app/web/dist
COPY content/ /app/content/

ENV PORT=8080 \
    DB_PATH=/data/app.db \
    CONTENT_DIR=/app/content \
    WEB_DIST=/app/web/dist

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/app/server"]
