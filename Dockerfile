FROM golang:1.23.2-alpine3.20 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .

RUN go build -o myapp .

FROM alpine:latest

COPY .env /

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/myapp /usr/local/bin/myapp

EXPOSE 8083

CMD ["myapp"]
