# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/ayatbot

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ffmpeg ca-certificates tzdata

WORKDIR /app

RUN mkdir -p /app/backgrounds /app/fonts /app/video

COPY --from=builder /app/ayatbot /app/ayatbot
COPY quran.json /app/quran.json

RUN mkdir -p /app/fonts && \
    cp /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ 2>/dev/null || true

EXPOSE 8080

CMD ["/app/ayatbot"]
