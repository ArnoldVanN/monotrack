FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
COPY cmd/ . internal/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /usr/local/bin/monotrack ./main.go

FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y jq && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/local/bin/monotrack /usr/local/bin/monotrack

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
