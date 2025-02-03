FROM golang:1.23.5-alpine AS builder

WORKDIR /build/
COPY go.* ./
RUN go mod download

FROM builder AS build

COPY . ./
RUN go build -o api .

FROM alpine:latest

WORKDIR /app/
COPY --from=build /build/api ./api

CMD ["./api"]
