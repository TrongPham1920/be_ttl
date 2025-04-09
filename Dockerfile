FROM golang:1.23-alpine3.20
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod tidy
COPY . .
RUN go build -o main .
EXPOSE 8083
CMD ["./main"]