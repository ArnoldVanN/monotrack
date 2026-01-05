FROM golang:1.25-bullseye as builder

WORKDIR /app

COPY . .

RUN mkdir -p ./.out && \
    go build -o ./monotrack ./main.go

FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y jq && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/monotrack /usr/local/bin/monotrack

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
