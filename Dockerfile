FROM golang:1.25 as builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./monotrack ./main.go

FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y jq && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/monotrack /usr/local/bin/monotrack

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
