FROM golang:1.23-alpine AS builder
# git is required by go mod for VCS-based module resolution
RUN apk add --no-cache git
WORKDIR /app
ENV GONOSUMDB=*
COPY go.mod ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ingest ./cmd/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /ingest /ingest
EXPOSE 8080
ENTRYPOINT ["/ingest"]
