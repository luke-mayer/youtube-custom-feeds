FROM golang:1.23.5-alpine AS builder

WORKDIR /build
COPY . .

RUN go mod download
RUN go build -o ./api

FROM debian:stable-slim

# RUN apt-get update && apt-get install -y ca-certificates

# COPY youtube-custom-feeds /bin/youtube-custom-feeds

WORKDIR /app
COPY --from=builder /build/api ./api

CMD ["/app/api"]
