FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o deepseek-monitor .

FROM alpine:3.18
RUN apk add --no-cache ca-certificates tzdata sqlite
RUN mkdir -p /app/data
WORKDIR /app
COPY --from=builder /app/deepseek-monitor .
COPY config.yaml .

EXPOSE 8080
CMD ["./deepseek-monitor", "config.yaml"]
