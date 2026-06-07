# syntax=docker/dockerfile:1.7

# ---- Build stage ----
FROM golang:1.22-alpine AS build

WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Build static binary (no CGO needed thanks to modernc.org/sqlite)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w -X main.Version=$(date +%Y%m%d)" \
    -o /out/goform .

# ---- Runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/goform /app/goform

# Persistent volume for the SQLite database
VOLUME ["/data"]
ENV GOFORM_DATA=/data
ENV GOFORM_ADDR=:3000

EXPOSE 3000

USER nonroot:nonroot

ENTRYPOINT ["/app/goform"]
