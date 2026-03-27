FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/ayatbot

FROM alpine:3.21

RUN apk add --no-cache ffmpeg ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/ayatbot /app/ayatbot
COPY --from=builder /app/fonts/* /app/fonts/
COPY --from=builder /app/backgrounds/* /app/backgrounds/
COPY --from=builder /app/config.json /app/config.json
COPY --from=builder /app/quran.json /app/quran.json

EXPOSE 8080

CMD ["/app/ayatbot"]
