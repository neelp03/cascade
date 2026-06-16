FROM golang:1.23-alpine AS builder
WORKDIR /app
# Copy everything first so go mod tidy can scan imports
COPY go.mod ./
COPY . .
RUN go mod tidy && go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ingest ./cmd/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /ingest /ingest
EXPOSE 8080
ENTRYPOINT ["/ingest"]
