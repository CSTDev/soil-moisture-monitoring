FROM golang:1.23.7-alpine3.21 AS builder
WORKDIR /app
RUN apk add build-base
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . ./
RUN go mod tidy
RUN env GOOS=linux GOARCH=arm GOARM=7 go build -o ./consumer ./cmd/consumer.go

FROM arm32v7/alpine:3
WORKDIR /app
COPY --from=builder /app/consumer ./
ENTRYPOINT ["./consumer"]