FROM golang:1.26.1 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /bin/server /app/server
COPY --from=builder /src/internal/web/static /app/static

EXPOSE 8080
CMD ["/app/server"]