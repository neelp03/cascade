FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
ENV GONOSUMDB=*
COPY go.mod ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /writer ./cmd/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /writer /writer
ENTRYPOINT ["/writer"]
