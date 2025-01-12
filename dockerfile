FROM golang:1.23 AS builder

WORKDIR /app


COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
-ldflags="-w -s" -o tic_tac_toe ./cmd

FROM alpine:latest


WORKDIR /app

COPY --from=builder /app/tic_tac_toe .

COPY conf /app/conf

ENV CONFIG_PATH=/app/conf/local.env

CMD ["./tic_tac_toe"]