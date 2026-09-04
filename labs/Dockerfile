FROM golang:1.22-alpine3.21 AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -v -o app .

FROM scratch

WORKDIR /app

COPY --from=builder /app/app /app/app

CMD ["./app"]
