# Build stage
FROM golang:1.26.7-alpine AS builder

WORKDIR /build

COPY . .

RUN go mod download && \
    CGO_ENABLED=1 GOOS=linux go build -o /build/dimd ./cmd/dimd

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/dimd /usr/local/bin/dimd

ENTRYPOINT ["dimd"]
