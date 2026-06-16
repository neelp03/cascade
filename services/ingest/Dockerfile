FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ingest ./cmd/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /ingest /ingest
EXPOSE 8080
ENTRYPOINT ["/ingest"]
