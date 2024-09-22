FROM golang:1.23-alpine AS builder

WORKDIR /src

ENV CGO_ENABLED=1
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o hako ./cmd/hako

FROM alpine AS runner

WORKDIR /app

ENV HAKO_HTTP_LISTEN_ADDR=":8080" \
    HAKO_DB_LOCATION=":memory:" \
    HAKO_FS_ROOT="/tmp/hako" \
    HAKO_FS_MAX_FILE_SIZE="1000000000" \
    HAKO_FS_MAX_TTL="7d" \
    GIN_MODE="release"

COPY --from=builder /src/hako .

CMD ["/app/hako"]